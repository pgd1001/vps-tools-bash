package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		server  *Server
		wantErr bool
	}{
		{
			name:    "valid server",
			server: &Server{
				ID:     "test-server",
				Name:   "Test Server",
				Host:   "192.168.1.10",
				Port:   22,
				User:   "ubuntu",
				AuthMethod: AuthConfig{
					Type: "ssh_agent",
				},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			server: &Server{
				Name:   "Test Server",
				Host:   "192.168.1.10",
				Port:   22,
				User:   "ubuntu",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			server: &Server{
				ID:   "test-server",
				Host: "192.168.1.10",
				Port: 22,
				User: "ubuntu",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			server: &Server{
				ID:   "test-server",
				Name: "Test Server",
				Host: "192.168.1.10",
				Port: 70000,
				User: "ubuntu",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.server.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name       string
		authConfig AuthConfig
		wantErr    bool
	}{
		{
			name: "valid SSH agent",
			authConfig: AuthConfig{
				Type: "ssh_agent",
			},
			wantErr: false,
		},
		{
			name: "valid private key with key path",
			authConfig: AuthConfig{
				Type:    "private_key",
				KeyPath: "/home/user/.ssh/id_rsa",
			},
			wantErr: false,
		},
		{
			name: "valid private key with key content",
			authConfig: AuthConfig{
				Type:       "private_key",
				PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\n...",
			},
			wantErr: false,
		},
		{
			name: "private key missing both path and content",
			authConfig: AuthConfig{
				Type: "private_key",
			},
			wantErr: true,
		},
		{
			name: "valid password",
			authConfig: AuthConfig{
				Type:     "password",
				Password: "secret123",
			},
			wantErr: false,
		},
		{
			name: "password missing password",
			authConfig: AuthConfig{
				Type: "password",
			},
			wantErr: true,
		},
		{
			name: "invalid auth type",
			authConfig: AuthConfig{
				Type: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.authConfig.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJob_Validate(t *testing.T) {
	tests := []struct {
		name    string
		job     *Job
		wantErr bool
	}{
		{
			name: "valid job",
			job: &Job{
				ID:       "job-1",
				ServerID: "server-1",
				Command:  "uptime",
				Timeout:  300,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			job: &Job{
				ServerID: "server-1",
				Command:  "uptime",
			},
			wantErr: true,
		},
		{
			name: "missing server ID",
			job: &Job{
				ID:      "job-1",
				Command: "uptime",
			},
			wantErr: true,
		},
		{
			name: "missing command",
			job: &Job{
				ID:       "job-1",
				ServerID: "server-1",
			},
			wantErr: true,
		},
		{
			name: "zero timeout should get default",
			job: &Job{
				ID:       "job-1",
				ServerID: "server-1",
				Command:  "uptime",
				Timeout:  0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			// Check that default timeout is set
			if !tt.wantErr && tt.job.Timeout == 0 {
				assert.Equal(t, 300, tt.job.Timeout)
			}
		})
	}
}

func TestServer_HasTag(t *testing.T) {
	server := &Server{
		Tags: []string{"web", "production", "ubuntu"},
	}

	assert.True(t, server.HasTag("web"))
	assert.True(t, server.HasTag("production"))
	assert.False(t, server.HasTag("database"))
	assert.False(t, server.HasTag("WEB")) // case sensitive
}

func TestServer_AddTag(t *testing.T) {
	server := &Server{
		Tags: []string{"web"},
	}

	server.AddTag("production")
	assert.Contains(t, server.Tags, "production")
	assert.Len(t, server.Tags, 2)

	// Adding existing tag should not duplicate
	server.AddTag("web")
	assert.Len(t, server.Tags, 2)
}

func TestServer_RemoveTag(t *testing.T) {
	server := &Server{
		Tags: []string{"web", "production", "database"},
	}

	server.RemoveTag("production")
	assert.NotContains(t, server.Tags, "production")
	assert.Contains(t, server.Tags, "web")
	assert.Contains(t, server.Tags, "database")
	assert.Len(t, server.Tags, 2)

	// Removing non-existent tag should not error
	server.RemoveTag("nonexistent")
	assert.Len(t, server.Tags, 2)
}

func TestJob_IsRunning(t *testing.T) {
	tests := []struct {
		name  string
		status JobStatus
		want  bool
	}{
		{"running", JobStatusRunning, true},
		{"pending", JobStatusPending, false},
		{"completed", JobStatusCompleted, false},
		{"failed", JobStatusFailed, false},
		{"timeout", JobStatusTimeout, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{Status: tt.status}
			assert.Equal(t, tt.want, job.IsRunning())
		})
	}
}

func TestJob_IsCompleted(t *testing.T) {
	tests := []struct {
		name  string
		status JobStatus
		want  bool
	}{
		{"completed", JobStatusCompleted, true},
		{"failed", JobStatusFailed, true},
		{"timeout", JobStatusTimeout, true},
		{"running", JobStatusRunning, false},
		{"pending", JobStatusPending, false},
		{"cancelled", JobStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{Status: tt.status}
			assert.Equal(t, tt.want, job.IsCompleted())
		})
	}
}

func TestJob_Duration(t *testing.T) {
	now := time.Now()
	
	t.Run("job not started", func(t *testing.T) {
		job := &Job{}
		assert.Equal(t, time.Duration(0), job.Duration())
	})

	t.Run("job running", func(t *testing.T) {
		startTime := now.Add(-5 * time.Minute)
		job := &Job{StartedAt: startTime}
		duration := job.Duration()
		assert.True(t, duration > 4*time.Minute && duration < 6*time.Minute)
	})

	t.Run("job completed", func(t *testing.T) {
		startTime := now.Add(-10 * time.Minute)
		endTime := now.Add(-5 * time.Minute)
		job := &Job{
			StartedAt:  startTime,
			FinishedAt: &endTime,
		}
		assert.Equal(t, 5*time.Minute, job.Duration())
	})
}

func TestServerStatus_String(t *testing.T) {
	tests := []struct {
		status ServerStatus
		want   string
	}{
		{StatusUnknown, "unknown"},
		{StatusOnline, "online"},
		{StatusOffline, "offline"},
		{StatusMaintenance, "maintenance"},
		{StatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.status))
		})
	}
}

func TestJobStatus_String(t *testing.T) {
	tests := []struct {
		status JobStatus
		want   string
	}{
		{JobStatusPending, "pending"},
		{JobStatusRunning, "running"},
		{JobStatusCompleted, "completed"},
		{JobStatusFailed, "failed"},
		{JobStatusTimeout, "timeout"},
		{JobStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.status))
		})
	}
}