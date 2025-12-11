package server

import (
	"time"
)

// Server represents a managed server in the vps-tools inventory
type Server struct {
	ID           string            `json:"id" bolt:"id"`
	Name         string            `json:"name" bolt:"name"`
	Host         string            `json:"host" bolt:"host"`
	Port         int               `json:"port" bolt:"port"`
	User         string            `json:"user" bolt:"user"`
	AuthMethod   AuthConfig        `json:"auth_method" bolt:"auth_method"`
	Tags         []string          `json:"tags" bolt:"tags"`
	Status       ServerStatus      `json:"status" bolt:"status"`
	LastSeen     time.Time         `json:"last_seen" bolt:"last_seen"`
	Meta         map[string]string `json:"meta" bolt:"meta"`
	CreatedAt    time.Time         `json:"created_at" bolt:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" bolt:"updated_at"`
}

// AuthConfig represents authentication configuration for a server
type AuthConfig struct {
	Type         string `json:"type" yaml:"type"`                   // ssh_agent, private_key, password
	PrivateKey   string `json:"private_key,omitempty" yaml:"private_key,omitempty"`
	KeyPath      string `json:"key_path,omitempty" yaml:"key_path,omitempty"`
	Password     string `json:"password,omitempty" yaml:"password,omitempty"`
	UseAgent     bool   `json:"use_agent,omitempty" yaml:"use_agent,omitempty"`
	KnownHosts   string `json:"known_hosts,omitempty" yaml:"known_hosts,omitempty"`
	BastionHost  string `json:"bastion_host,omitempty" yaml:"bastion_host,omitempty"`
}

// ServerStatus represents the current status of a server
type ServerStatus string

const (
	StatusUnknown     ServerStatus = "unknown"
	StatusOnline      ServerStatus = "online"
	StatusOffline     ServerStatus = "offline"
	StatusMaintenance ServerStatus = "maintenance"
	StatusError      ServerStatus = "error"
)

// Job represents a task executed on a server
type Job struct {
	ID         string     `json:"id" bolt:"id"`
	ServerID   string     `json:"server_id" bolt:"server_id"`
	Command    string     `json:"command" bolt:"command"`
	StartedAt  time.Time  `json:"started_at" bolt:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty" bolt:"finished_at"`
	ExitCode   int        `json:"exit_code" bolt:"exit_code"`
	Stdout     string     `json:"stdout" bolt:"stdout"`
	Stderr     string     `json:"stderr" bolt:"stderr"`
	Status     JobStatus  `json:"status" bolt:"status"`
	User       string     `json:"user" bolt:"user"`
	WorkingDir string     `json:"working_dir,omitempty" bolt:"working_dir"`
	Timeout    int        `json:"timeout,omitempty" bolt:"timeout"` // timeout in seconds
	CreatedAt  time.Time  `json:"created_at" bolt:"created_at"`
}

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusTimeout   JobStatus = "timeout"
	JobStatusCancelled JobStatus = "cancelled"
)

// HealthCheck represents a health check result for a server
type HealthCheck struct {
	ID          string                 `json:"id" bolt:"id"`
	ServerID    string                 `json:"server_id" bolt:"server_id"`
	Timestamp   time.Time              `json:"timestamp" bolt:"timestamp"`
	Status      ServerStatus           `json:"status" bolt:"status"`
	Metrics     map[string]interface{} `json:"metrics" bolt:"metrics"`
	Checks      map[string]CheckResult `json:"checks" bolt:"checks"`
	Duration    time.Duration          `json:"duration" bolt:"duration"`
	Error       string                 `json:"error,omitempty" bolt:"error"`
}

// CheckResult represents the result of a specific health check
type CheckResult struct {
	Status  string      `json:"status"`  // ok, warning, critical, unknown
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
	Unit    string      `json:"unit,omitempty"`
}

// ServerFilter represents filters for querying servers
type ServerFilter struct {
	Tags    []string `json:"tags,omitempty"`
	Status  string   `json:"status,omitempty"`
	Search  string   `json:"search,omitempty"` // search in name, host
	Limit   int      `json:"limit,omitempty"`
	Offset  int      `json:"offset,omitempty"`
}

// JobFilter represents filters for querying jobs
type JobFilter struct {
	ServerID   string     `json:"server_id,omitempty"`
	Status     JobStatus  `json:"status,omitempty"`
	StartedAfter  *time.Time `json:"started_after,omitempty"`
	StartedBefore *time.Time `json:"started_before,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// Validation methods

// Validate validates the server configuration
func (s *Server) Validate() error {
	if s.ID == "" {
		return ErrServerIDRequired
	}
	if s.Name == "" {
		return ErrServerNameRequired
	}
	if s.Host == "" {
		return ErrServerHostRequired
	}
	if s.Port <= 0 || s.Port > 65535 {
		return ErrInvalidPort
	}
	if s.User == "" {
		return ErrServerUserRequired
	}
	
	return s.AuthMethod.Validate()
}

// Validate validates the authentication configuration
func (a *AuthConfig) Validate() error {
	switch a.Type {
	case "ssh_agent":
		// SSH agent doesn't require additional validation
	case "private_key":
		if a.KeyPath == "" && a.PrivateKey == "" {
			return ErrPrivateKeyRequired
		}
	case "password":
		if a.Password == "" {
			return ErrPasswordRequired
		}
	default:
		return ErrInvalidAuthType
	}
	return nil
}

// Validate validates the job configuration
func (j *Job) Validate() error {
	if j.ID == "" {
		return ErrJobIDRequired
	}
	if j.ServerID == "" {
		return ErrJobServerIDRequired
	}
	if j.Command == "" {
		return ErrJobCommandRequired
	}
	if j.Timeout <= 0 {
		j.Timeout = 300 // default 5 minutes
	}
	return nil
}

// HasTag checks if server has a specific tag
func (s *Server) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the server if it doesn't already exist
func (s *Server) AddTag(tag string) {
	if !s.HasTag(tag) {
		s.Tags = append(s.Tags, tag)
	}
}

// RemoveTag removes a tag from the server
func (s *Server) RemoveTag(tag string) {
	for i, t := range s.Tags {
		if t == tag {
			s.Tags = append(s.Tags[:i], s.Tags[i+1:]...)
			break
		}
	}
}

// IsRunning checks if the job is currently running
func (j *Job) IsRunning() bool {
	return j.Status == JobStatusRunning
}

// IsCompleted checks if the job has completed (successfully or failed)
func (j *Job) IsCompleted() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed || j.Status == JobStatusTimeout
}

// Duration returns the job execution duration
func (j *Job) Duration() time.Duration {
	if j.StartedAt.IsZero() {
		return 0
	}
	if j.FinishedAt != nil {
		return j.FinishedAt.Sub(j.StartedAt)
	}
	return time.Since(j.StartedAt)
}