package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-bolt/bolt"
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/server"
)

// Store represents the data store interface
type Store interface {
	// Server operations
	GetServer(id string) (*server.Server, error)
	ListServers(filter *server.ServerFilter) ([]*server.Server, error)
	CreateServer(srv *server.Server) error
	UpdateServer(srv *server.Server) error
	DeleteServer(id string) error
	GetServersByTag(tag string) ([]*server.Server, error)

	// Job operations
	GetJob(id string) (*server.Job, error)
	ListJobs(filter *server.JobFilter) ([]*server.Job, error)
	CreateJob(job *server.Job) error
	UpdateJob(job *server.Job) error
	DeleteJob(id string) error
	GetJobsByServer(serverID string) ([]*server.Job, error)

	// Health check operations
	GetHealthCheck(id string) (*server.HealthCheck, error)
	ListHealthChecks(serverID string, limit int) ([]*server.HealthCheck, error)
	CreateHealthCheck(check *server.HealthCheck) error
	DeleteHealthChecks(olderThan time.Time) error

	// Configuration operations
	GetConfig() (*config.Config, error)
	SaveConfig(cfg *config.Config) error

	// Utility operations
	Close() error
	Backup(path string) error
	Restore(path string) error
	GetStats() (*StoreStats, error)
}

// StoreStats represents database statistics
type StoreStats struct {
	ServerCount     int64 `json:"server_count"`
	JobCount        int64 `json:"job_count"`
	HealthCheckCount int64 `json:"health_check_count"`
	DatabaseSize    int64 `json:"database_size"`
	LastBackup      time.Time `json:"last_backup"`
}

// BoltStore implements Store interface using BoltDB
type BoltStore struct {
	db     *bolt.DB
	config *config.Config
}

// Bucket names
const (
	ServersBucket     = "servers"
	JobsBucket       = "jobs"
	HealthBucket     = "health_checks"
	ConfigBucket     = "config"
	MetadataBucket   = "metadata"
)

// NewBoltStore creates a new BoltDB store
func NewBoltStore(cfg *config.Config) (*BoltStore, error) {
	// Ensure database directory exists
	dbPath := cfg.Storage.BoltDBPath
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout:      30 * time.Second,
		NoGrowSync:   false,
		FreelistType: bolt.FreelistArrayType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := []string{ServersBucket, JobsBucket, HealthBucket, ConfigBucket, MetadataBucket}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	return &BoltStore{
		db:     db,
		config: cfg,
	}, nil
}

// Close closes the database connection
func (bs *BoltStore) Close() error {
	if bs.db != nil {
		return bs.db.Close()
	}
	return nil
}

// Server operations

// GetServer retrieves a server by ID
func (bs *BoltStore) GetServer(id string) (*server.Server, error) {
	var srv *server.Server
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServersBucket))
		data := b.Get([]byte(id))
		if data == nil {
			return server.ErrServerNotFound
		}
		return json.Unmarshal(data, &srv)
	})
	if err != nil {
		return nil, err
	}
	return srv, nil
}

// ListServers retrieves all servers with optional filtering
func (bs *BoltStore) ListServers(filter *server.ServerFilter) ([]*server.Server, error) {
	var servers []*server.Server
	
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServersBucket))
		return b.ForEach(func(k, v []byte) error {
			var srv server.Server
			if err := json.Unmarshal(v, &srv); err != nil {
				return err
			}

			// Apply filters
			if filter != nil {
				// Status filter
				if filter.Status != "" && string(srv.Status) != filter.Status {
					return nil
				}

				// Tag filter
				if len(filter.Tags) > 0 {
					hasTag := false
					for _, tag := range filter.Tags {
						if srv.HasTag(tag) {
							hasTag = true
							break
						}
					}
					if !hasTag {
						return nil
					}
				}

				// Search filter
				if filter.Search != "" {
					searchLower := filter.Search
					nameMatch := containsIgnoreCase(srv.Name, searchLower)
					hostMatch := containsIgnoreCase(srv.Host, searchLower)
					if !nameMatch && !hostMatch {
						return nil
					}
				}
			}

			servers = append(servers, &srv)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	// Apply pagination
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		if start >= len(servers) {
			return []*server.Server{}, nil
		}
		end := start + filter.Limit
		if end > len(servers) {
			end = len(servers)
		}
		servers = servers[start:end]
	}

	return servers, nil
}

// CreateServer creates a new server
func (bs *BoltStore) CreateServer(srv *server.Server) error {
	// Validate server
	if err := srv.Validate(); err != nil {
		return err
	}

	// Set timestamps
	now := time.Now()
	srv.CreatedAt = now
	srv.UpdatedAt = now

	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServersBucket))
		
		// Check if server already exists
		if data := b.Get([]byte(srv.ID)); data != nil {
			return server.ErrServerAlreadyExists
		}

		// Serialize and store
		data, err := json.Marshal(srv)
		if err != nil {
			return fmt.Errorf("failed to marshal server: %w", err)
		}

		return b.Put([]byte(srv.ID), data)
	})
}

// UpdateServer updates an existing server
func (bs *BoltStore) UpdateServer(srv *server.Server) error {
	// Validate server
	if err := srv.Validate(); err != nil {
		return err
	}

	// Set updated timestamp
	srv.UpdatedAt = time.Now()

	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServersBucket))
		
		// Check if server exists
		if data := b.Get([]byte(srv.ID)); data == nil {
			return server.ErrServerNotFound
		}

		// Serialize and update
		data, err := json.Marshal(srv)
		if err != nil {
			return fmt.Errorf("failed to marshal server: %w", err)
		}

		return b.Put([]byte(srv.ID), data)
	})
}

// DeleteServer deletes a server by ID
func (bs *BoltStore) DeleteServer(id string) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServersBucket))
		
		// Check if server exists
		if data := b.Get([]byte(id)); data == nil {
			return server.ErrServerNotFound
		}

		// Delete server
		if err := b.Delete([]byte(id)); err != nil {
			return fmt.Errorf("failed to delete server: %w", err)
		}

		// Optionally delete associated jobs and health checks
		// This is a design decision - for now we keep them for audit purposes
		return nil
	})
}

// GetServersByTag retrieves servers with a specific tag
func (bs *BoltStore) GetServersByTag(tag string) ([]*server.Server, error) {
	filter := &server.ServerFilter{
		Tags: []string{tag},
	}
	return bs.ListServers(filter)
}

// Job operations

// GetJob retrieves a job by ID
func (bs *BoltStore) GetJob(id string) (*server.Job, error) {
	var job *server.Job
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(JobsBucket))
		data := b.Get([]byte(id))
		if data == nil {
			return server.ErrJobNotFound
		}
		return json.Unmarshal(data, &job)
	})
	if err != nil {
		return nil, err
	}
	return job, nil
}

// ListJobs retrieves jobs with optional filtering
func (bs *BoltStore) ListJobs(filter *server.JobFilter) ([]*server.Job, error) {
	var jobs []*server.Job
	
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(JobsBucket))
		return b.ForEach(func(k, v []byte) error {
			var job server.Job
			if err := json.Unmarshal(v, &job); err != nil {
				return err
			}

			// Apply filters
			if filter != nil {
				// Server ID filter
				if filter.ServerID != "" && job.ServerID != filter.ServerID {
					return nil
				}

				// Status filter
				if filter.Status != "" && job.Status != filter.Status {
					return nil
				}

				// Time filters
				if filter.StartedAfter != nil && job.StartedAt.Before(*filter.StartedAfter) {
					return nil
				}
				if filter.StartedBefore != nil && job.StartedAt.After(*filter.StartedBefore) {
					return nil
				}
			}

			jobs = append(jobs, &job)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	// Apply pagination
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		if start >= len(jobs) {
			return []*server.Job{}, nil
		}
		end := start + filter.Limit
		if end > len(jobs) {
			end = len(jobs)
		}
		jobs = jobs[start:end]
	}

	return jobs, nil
}

// CreateJob creates a new job
func (bs *BoltStore) CreateJob(job *server.Job) error {
	// Validate job
	if err := job.Validate(); err != nil {
		return err
	}

	// Set created timestamp
	job.CreatedAt = time.Now()

	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(JobsBucket))
		
		// Serialize and store
		data, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("failed to marshal job: %w", err)
		}

		return b.Put([]byte(job.ID), data)
	})
}

// UpdateJob updates an existing job
func (bs *BoltStore) UpdateJob(job *server.Job) error {
	// Validate job
	if err := job.Validate(); err != nil {
		return err
	}

	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(JobsBucket))
		
		// Check if job exists
		if data := b.Get([]byte(job.ID)); data == nil {
			return server.ErrJobNotFound
		}

		// Serialize and update
		data, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("failed to marshal job: %w", err)
		}

		return b.Put([]byte(job.ID), data)
	})
}

// DeleteJob deletes a job by ID
func (bs *BoltStore) DeleteJob(id string) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(JobsBucket))
		
		// Check if job exists
		if data := b.Get([]byte(id)); data == nil {
			return server.ErrJobNotFound
		}

		return b.Delete([]byte(id))
	})
}

// GetJobsByServer retrieves jobs for a specific server
func (bs *BoltStore) GetJobsByServer(serverID string) ([]*server.Job, error) {
	filter := &server.JobFilter{
		ServerID: serverID,
	}
	return bs.ListJobs(filter)
}

// Health check operations

// GetHealthCheck retrieves a health check by ID
func (bs *BoltStore) GetHealthCheck(id string) (*server.HealthCheck, error) {
	var check *server.HealthCheck
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(HealthBucket))
		data := b.Get([]byte(id))
		if data == nil {
			return server.ErrHealthCheckNotFound
		}
		return json.Unmarshal(data, &check)
	})
	if err != nil {
		return nil, err
	}
	return check, nil
}

// ListHealthChecks retrieves health checks for a server
func (bs *BoltStore) ListHealthChecks(serverID string, limit int) ([]*server.HealthCheck, error) {
	var checks []*server.HealthCheck
	
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(HealthBucket))
		return b.ForEach(func(k, v []byte) error {
			var check server.HealthCheck
			if err := json.Unmarshal(v, &check); err != nil {
				return err
			}

			// Filter by server ID
			if serverID != "" && check.ServerID != serverID {
				return nil
			}

			checks = append(checks, &check)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	// Sort by timestamp (newest first) and apply limit
	if len(checks) > 1 {
		// Simple sort - in production, you might want to use a more efficient method
		for i := 0; i < len(checks)-1; i++ {
			for j := i + 1; j < len(checks); j++ {
				if checks[i].Timestamp.Before(checks[j].Timestamp) {
					checks[i], checks[j] = checks[j], checks[i]
				}
			}
		}
	}

	if limit > 0 && len(checks) > limit {
		checks = checks[:limit]
	}

	return checks, nil
}

// CreateHealthCheck creates a new health check record
func (bs *BoltStore) CreateHealthCheck(check *server.HealthCheck) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(HealthBucket))
		
		// Serialize and store
		data, err := json.Marshal(check)
		if err != nil {
			return fmt.Errorf("failed to marshal health check: %w", err)
		}

		return b.Put([]byte(check.ID), data)
	})
}

// DeleteHealthChecks deletes health checks older than specified time
func (bs *BoltStore) DeleteHealthChecks(olderThan time.Time) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(HealthBucket))
		var toDelete [][]byte
		
		// Find health checks to delete
		err := b.ForEach(func(k, v []byte) error {
			var check server.HealthCheck
			if err := json.Unmarshal(v, &check); err != nil {
				return err
			}
			
			if check.Timestamp.Before(olderThan) {
				toDelete = append(toDelete, k)
			}
			return nil
		})
		
		if err != nil {
			return err
		}

		// Delete old health checks
		for _, key := range toDelete {
			if err := b.Delete(key); err != nil {
				return fmt.Errorf("failed to delete health check: %w", err)
			}
		}

		return nil
	})
}

// Configuration operations

// GetConfig retrieves the stored configuration
func (bs *BoltStore) GetConfig() (*config.Config, error) {
	var cfg *config.Config
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ConfigBucket))
		data := b.Get([]byte("global"))
		if data == nil {
			return config.ErrConfigNotFound
		}
		return json.Unmarshal(data, &cfg)
	})
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// SaveConfig saves the configuration
func (bs *BoltStore) SaveConfig(cfg *config.Config) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ConfigBucket))
		
		// Serialize and store
		data, err := json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		return b.Put([]byte("global"), data)
	})
}

// Utility operations

// Backup creates a backup of the database
func (bs *BoltStore) Backup(path string) error {
	// Ensure backup directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy database file
	return bs.db.View(func(tx *bolt.Tx) error {
		return tx.CopyFile(path, 0600)
	})
}

// Restore restores a database from backup
func (bs *BoltStore) Restore(path string) error {
	// Close current database
	if err := bs.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	// Copy backup file to database location
	err := copyFile(path, bs.config.Storage.BoltDBPath)
	if err != nil {
		// Try to reopen database even if restore failed
		bs.reopen()
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Reopen database
	return bs.reopen()
}

// GetStats returns database statistics
func (bs *BoltStore) GetStats() (*StoreStats, error) {
	stats := &StoreStats{}
	
	err := bs.db.View(func(tx *bolt.Tx) error {
		// Count servers
		b := tx.Bucket([]byte(ServersBucket))
		stats.ServerCount = int64(b.Stats().KeyN)

		// Count jobs
		b = tx.Bucket([]byte(JobsBucket))
		stats.JobCount = int64(b.Stats().KeyN)

		// Count health checks
		b = tx.Bucket([]byte(HealthBucket))
		stats.HealthCheckCount = int64(b.Stats().KeyN)

		// Get database size
		dbInfo := bs.db.Info()
		stats.DatabaseSize = dbInfo.PageSize * int64(dbInfo.NumPage)

		// Get last backup time from metadata
		b = tx.Bucket([]byte(MetadataBucket))
		data := b.Get([]byte("last_backup"))
		if data != nil {
			var lastBackup time.Time
			if err := json.Unmarshal(data, &lastBackup); err == nil {
				stats.LastBackup = lastBackup
			}
		}

		return nil
	})

	return stats, err
}

// Helper functions

// reopen reopens the database connection
func (bs *BoltStore) reopen() error {
	db, err := bolt.Open(bs.config.Storage.BoltDBPath, 0600, &bolt.Options{
		Timeout:      30 * time.Second,
		NoGrowSync:   false,
		FreelistType: bolt.FreelistArrayType,
	})
	if err != nil {
		return fmt.Errorf("failed to reopen database: %w", err)
	}
	
	bs.db = db
	return nil
}

// containsIgnoreCase checks if a string contains another string (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = destination.ReadFrom(source)
	return err
}