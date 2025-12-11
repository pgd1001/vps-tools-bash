package store

import (
	"os"
	"testing"
	"time"

	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoltStore_ServerOperations(t *testing.T) {
	// Create temporary database
	dbPath := t.TempDir() + "/test.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := NewBoltStore(cfg)
	require.NoError(t, err)
	defer store.Close()

	// Test server creation
	testServer := &server.Server{
		ID:   "test-server",
		Name: "Test Server",
		Host: "192.168.1.10",
		Port: 22,
		User: "ubuntu",
		AuthMethod: server.AuthConfig{
			Type: "ssh_agent",
		},
		Tags: []string{"web", "test"},
	}

	err = store.CreateServer(testServer)
	require.NoError(t, err)

	// Test server retrieval
	retrieved, err := store.GetServer("test-server")
	require.NoError(t, err)
	assert.Equal(t, testServer.ID, retrieved.ID)
	assert.Equal(t, testServer.Name, retrieved.Name)
	assert.Equal(t, testServer.Host, retrieved.Host)
	assert.Equal(t, testServer.Port, retrieved.Port)
	assert.Equal(t, testServer.User, retrieved.User)
	assert.Equal(t, testServer.Tags, retrieved.Tags)
	assert.False(t, retrieved.CreatedAt.IsZero())
	assert.False(t, retrieved.UpdatedAt.IsZero())

	// Test duplicate server creation
	err = store.CreateServer(testServer)
	assert.Error(t, err)
	assert.Equal(t, server.ErrServerAlreadyExists, err)

	// Test server update
	testServer.Name = "Updated Server"
	err = store.UpdateServer(testServer)
	require.NoError(t, err)

	retrieved, err = store.GetServer("test-server")
	require.NoError(t, err)
	assert.Equal(t, "Updated Server", retrieved.Name)

	// Test server listing
	servers, err := store.ListServers(nil)
	require.NoError(t, err)
	assert.Len(t, servers, 1)

	// Test server listing with filters
	filter := &server.ServerFilter{
		Tags: []string{"web"},
	}
	servers, err = store.ListServers(filter)
	require.NoError(t, err)
	assert.Len(t, servers, 1)

	filter = &server.ServerFilter{
		Tags: []string{"database"},
	}
	servers, err = store.ListServers(filter)
	require.NoError(t, err)
	assert.Len(t, servers, 0)

	// Test server deletion
	err = store.DeleteServer("test-server")
	require.NoError(t, err)

	_, err = store.GetServer("test-server")
	assert.Error(t, err)
	assert.Equal(t, server.ErrServerNotFound, err)
}

func TestBoltStore_JobOperations(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := NewBoltStore(cfg)
	require.NoError(t, err)
	defer store.Close()

	// Create a test server first
	testServer := &server.Server{
		ID:   "test-server",
		Name: "Test Server",
		Host: "192.168.1.10",
		Port: 22,
		User: "ubuntu",
		AuthMethod: server.AuthConfig{
			Type: "ssh_agent",
		},
	}
	err = store.CreateServer(testServer)
	require.NoError(t, err)

	// Test job creation
	testJob := &server.Job{
		ID:       "test-job",
		ServerID: "test-server",
		Command:  "uptime",
		Status:   server.JobStatusPending,
	}

	err = store.CreateJob(testJob)
	require.NoError(t, err)

	// Test job retrieval
	retrieved, err := store.GetJob("test-job")
	require.NoError(t, err)
	assert.Equal(t, testJob.ID, retrieved.ID)
	assert.Equal(t, testJob.ServerID, retrieved.ServerID)
	assert.Equal(t, testJob.Command, retrieved.Command)
	assert.Equal(t, testJob.Status, retrieved.Status)
	assert.False(t, retrieved.CreatedAt.IsZero())

	// Test job listing
	jobs, err := store.ListJobs(nil)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)

	// Test job listing with filters
	filter := &server.JobFilter{
		ServerID: "test-server",
	}
	jobs, err = store.ListJobs(filter)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)

	filter = &server.JobFilter{
		ServerID: "other-server",
	}
	jobs, err = store.ListJobs(filter)
	require.NoError(t, err)
	assert.Len(t, jobs, 0)

	// Test job update
	testJob.Status = server.JobStatusCompleted
	testJob.ExitCode = 0
	finishedAt := time.Now()
	testJob.FinishedAt = &finishedAt

	err = store.UpdateJob(testJob)
	require.NoError(t, err)

	retrieved, err = store.GetJob("test-job")
	require.NoError(t, err)
	assert.Equal(t, server.JobStatusCompleted, retrieved.Status)
	assert.Equal(t, 0, retrieved.ExitCode)
	assert.NotNil(t, retrieved.FinishedAt)

	// Test job deletion
	err = store.DeleteJob("test-job")
	require.NoError(t, err)

	_, err = store.GetJob("test-job")
	assert.Error(t, err)
	assert.Equal(t, server.ErrJobNotFound, err)
}

func TestBoltStore_HealthCheckOperations(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := NewBoltStore(cfg)
	require.NoError(t, err)
	defer store.Close()

	// Test health check creation
	testCheck := &server.HealthCheck{
		ID:        "health-1",
		ServerID:  "test-server",
		Timestamp: time.Now(),
		Status:    server.StatusOnline,
		Metrics: map[string]interface{}{
			"cpu_usage":    45.5,
			"memory_usage": 67.2,
			"disk_usage":   23.8,
		},
		Checks: map[string]server.CheckResult{
			"disk": {
				Status:  "ok",
				Message: "Disk usage is normal",
				Value:   23.8,
				Unit:    "%",
			},
			"memory": {
				Status:  "warning",
				Message: "Memory usage is high",
				Value:   67.2,
				Unit:    "%",
			},
		},
		Duration: time.Second * 5,
	}

	err = store.CreateHealthCheck(testCheck)
	require.NoError(t, err)

	// Test health check retrieval
	retrieved, err := store.GetHealthCheck("health-1")
	require.NoError(t, err)
	assert.Equal(t, testCheck.ID, retrieved.ID)
	assert.Equal(t, testCheck.ServerID, retrieved.ServerID)
	assert.Equal(t, testCheck.Status, retrieved.Status)
	assert.Equal(t, testCheck.Duration, retrieved.Duration)
	assert.Equal(t, testCheck.Metrics["cpu_usage"], retrieved.Metrics["cpu_usage"])
	assert.Equal(t, testCheck.Checks["disk"].Status, retrieved.Checks["disk"].Status)

	// Test health check listing
	checks, err := store.ListHealthChecks("test-server", 10)
	require.NoError(t, err)
	assert.Len(t, checks, 1)

	// Test health check cleanup
	oldTime := time.Now().Add(-24 * time.Hour)
	err = store.DeleteHealthChecks(oldTime)
	require.NoError(t, err)

	// Create another health check to test cleanup
	newCheck := &server.HealthCheck{
		ID:        "health-2",
		ServerID:  "test-server",
		Timestamp: time.Now(),
		Status:    server.StatusOnline,
	}
	err = store.CreateHealthCheck(newCheck)
	require.NoError(t, err)

	// Delete old health checks (should delete health-1 but not health-2)
	err = store.DeleteHealthChecks(time.Now().Add(-1 * time.Hour))
	require.NoError(t, err)

	checks, err = store.ListHealthChecks("test-server", 10)
	require.NoError(t, err)
	assert.Len(t, checks, 1)
	assert.Equal(t, "health-2", checks[0].ID)
}

func TestBoltStore_ConfigOperations(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := NewBoltStore(cfg)
	require.NoError(t, err)
	defer store.Close()

	// Test config save
	testConfig := &config.Config{
		App: config.AppConfig{
			Name:     "test-app",
			LogLevel: "debug",
		},
		Storage: config.StorageConfig{
			Type: "bolt",
		},
	}

	err = store.SaveConfig(testConfig)
	require.NoError(t, err)

	// Test config retrieval
	retrieved, err := store.GetConfig()
	require.NoError(t, err)
	assert.Equal(t, testConfig.App.Name, retrieved.App.Name)
	assert.Equal(t, testConfig.App.LogLevel, retrieved.App.LogLevel)
	assert.Equal(t, testConfig.Storage.Type, retrieved.Storage.Type)

	// Test config not found
	store2, err := NewBoltStore(&config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: t.TempDir() + "/empty.db",
		},
	})
	require.NoError(t, err)
	defer store2.Close()

	_, err = store2.GetConfig()
	assert.Error(t, err)
	assert.Equal(t, config.ErrConfigNotFound, err)
}

func TestBoltStore_BackupAndRestore(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	backupPath := t.TempDir() + "/backup.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := NewBoltStore(cfg)
	require.NoError(t, err)

	// Create test data
	testServer := &server.Server{
		ID:   "test-server",
		Name: "Test Server",
		Host: "192.168.1.10",
		Port: 22,
		User: "ubuntu",
		AuthMethod: server.AuthConfig{
			Type: "ssh_agent",
		},
	}
	err = store.CreateServer(testServer)
	require.NoError(t, err)

	// Test backup
	err = store.Backup(backupPath)
	require.NoError(t, err)

	// Verify backup file exists
	_, err = os.Stat(backupPath)
	require.NoError(t, err)

	// Close store to allow restore
	err = store.Close()
	require.NoError(t, err)

	// Remove original database
	err = os.Remove(dbPath)
	require.NoError(t, err)

	// Create new store instance
	store2, err := NewBoltStore(cfg)
	require.NoError(t, err)
	defer store2.Close()

	// Test restore
	err = store2.Restore(backupPath)
	require.NoError(t, err)

	// Verify data was restored
	retrieved, err := store2.GetServer("test-server")
	require.NoError(t, err)
	assert.Equal(t, testServer.ID, retrieved.ID)
	assert.Equal(t, testServer.Name, retrieved.Name)
}

func TestBoltStore_GetStats(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := NewBoltStore(cfg)
	require.NoError(t, err)
	defer store.Close()

	// Create test data
	testServer := &server.Server{
		ID:   "test-server",
		Name: "Test Server",
		Host: "192.168.1.10",
		Port: 22,
		User: "ubuntu",
		AuthMethod: server.AuthConfig{
			Type: "ssh_agent",
		},
	}
	err = store.CreateServer(testServer)
	require.NoError(t, err)

	testJob := &server.Job{
		ID:       "test-job",
		ServerID: "test-server",
		Command:  "uptime",
		Status:   server.JobStatusPending,
	}
	err = store.CreateJob(testJob)
	require.NoError(t, err)

	testCheck := &server.HealthCheck{
		ID:        "health-1",
		ServerID:  "test-server",
		Timestamp: time.Now(),
		Status:    server.StatusOnline,
	}
	err = store.CreateHealthCheck(testCheck)
	require.NoError(t, err)

	// Test stats
	stats, err := store.GetStats()
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.ServerCount)
	assert.Equal(t, int64(1), stats.JobCount)
	assert.Equal(t, int64(1), stats.HealthCheckCount)
	assert.Greater(t, stats.DatabaseSize, int64(0))
}

func TestBoltStore_Validation(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := NewBoltStore(cfg)
	require.NoError(t, err)
	defer store.Close()

	// Test invalid server creation
	invalidServer := &server.Server{
		ID: "", // Missing ID
	}
	err = store.CreateServer(invalidServer)
	assert.Error(t, err)

	// Test invalid job creation
	invalidJob := &server.Job{
		ID: "", // Missing ID
	}
	err = store.CreateJob(invalidJob)
	assert.Error(t, err)
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "test", false},
		{"", "test", false},
		{"Hello", "", true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.want, containsIgnoreCase(tt.s, tt.substr))
		})
	}
}