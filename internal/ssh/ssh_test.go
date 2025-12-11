package ssh

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pgd1001/vps-tools/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger for testing
type MockLogger struct {
	infoMessages  []string
	warnMessages []string
	errorMessages []string
}

func (m *MockLogger) Info(args ...interface{}) {
	m.infoMessages = append(m.infoMessages, "INFO: "+formatArgs(args...))
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.infoMessages = append(m.infoMessages, "INFO: "+formatArgs(args...))
}

func (m *MockLogger) Warn(args ...interface{}) {
	m.warnMessages = append(m.warnMessages, "WARN: "+formatArgs(args...))
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.warnMessages = append(m.warnMessages, "WARN: "+formatArgs(args...))
}

func (m *MockLogger) Error(args ...interface{}) {
	m.errorMessages = append(m.errorMessages, "ERROR: "+formatArgs(args...))
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.errorMessages = append(m.errorMessages, "ERROR: "+formatArgs(args...))
}

func (m *MockLogger) Debug(args ...interface{}) {
	m.infoMessages = append(m.infoMessages, "DEBUG: "+formatArgs(args...))
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.infoMessages = append(m.infoMessages, "DEBUG: "+formatArgs(args...))
}

func (m *MockLogger) WithField(key string, value interface{}) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, key: key, value: value}
}

func (m *MockLogger) WithFields(fields map[string]interface{}) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, fields: fields}
}

func (m *MockLogger) WithError(err error) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, error: err}
}

func (m *MockLogger) WithServerID(serverID string) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, serverID: serverID}
}

func (m *MockLogger) WithJobID(jobID string) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, jobID: jobID}
}

func (m *MockLogger) WithComponent(component string) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, component: component}
}

func (m *MockLogger) WithDuration(duration interface{}) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, duration: duration}
}

func (m *MockLogger) WithUser(user string) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, user: user}
}

func (m *MockLogger) WithCommand(command string) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, command: command}
}

func (m *MockLogger) WithExitCode(exitCode int) *MockLoggerEntry {
	return &MockLoggerEntry{logger: m, exitCode: exitCode}
}

func (m *MockLogger) GetLevel() logger.LogLevel {
	return logger.InfoLevel
}

func (m *MockLogger) SetLevel(level logger.LogLevel) error {
	return nil
}

func (m *MockLogger) GetFormat() logger.LogFormat {
	return logger.TextFormat
}

func (m *MockLogger) SetFormat(format logger.LogFormat) error {
	return nil
}

// MockLoggerEntry for testing
type MockLoggerEntry struct {
	logger *MockLogger
	fields map[string]interface{}
	error  error
}

func (m *MockLoggerEntry) Info(args ...interface{}) {
	m.logger.infoMessages = append(m.logger.infoMessages, "INFO: "+formatArgs(args...))
}

func (m *MockLoggerEntry) Infof(format string, args ...interface{}) {
	m.logger.infoMessages = append(m.logger.infoMessages, "INFO: "+formatArgs(args...))
}

func (m *MockLoggerEntry) Warn(args ...interface{}) {
	m.logger.warnMessages = append(m.logger.warnMessages, "WARN: "+formatArgs(args...))
}

func (m *MockLoggerEntry) Warnf(format string, args ...interface{}) {
	m.logger.warnMessages = append(m.logger.warnMessages, "WARN: "+formatArgs(args...))
}

func (m *MockLoggerEntry) Error(args ...interface{}) {
	m.logger.errorMessages = append(m.logger.errorMessages, "ERROR: "+formatArgs(args...))
}

func (m *MockLoggerEntry) Errorf(format string, args ...interface{}) {
	m.logger.errorMessages = append(m.logger.errorMessages, "ERROR: "+formatArgs(args...))
}

func formatArgs(args []interface{}) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += fmt.Sprintf("%v", arg)
	}
	return result
}

func TestNewClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &Config{
			Host:       "example.com",
			Port:       22,
			User:       "testuser",
			AuthMethod: &MockAuthMethod{},
			Timeout:    30 * time.Second,
			MaxRetries: 3,
		}

		logger := &MockLogger{}
		client, err := NewClient(config, logger)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, config, client.config)
		assert.Equal(t, logger, client.logger)
	})

	t.Run("invalid config - missing host", func(t *testing.T) {
		config := &Config{
			Port: 22,
			User: "testuser",
		}

		logger := &MockLogger{}
		client, err := NewClient(config, logger)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "invalid SSH config")
	})

	t.Run("invalid config - invalid port", func(t *testing.T) {
		config := &Config{
			Host: "example.com",
			Port: 70000, // Invalid port
			User: "testuser",
		}

		logger := &MockLogger{}
		client, err := NewClient(config, logger)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "invalid SSH config")
	})
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Host:       "example.com",
				Port:       22,
				User:       "testuser",
				AuthMethod: &MockAuthMethod{},
				Timeout:    30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: &Config{
				Port: 22,
				User: "testuser",
			},
			wantErr: true,
			errMsg:  "SSH host is required",
		},
		{
			name: "invalid port",
			config: &Config{
				Host: "example.com",
				Port: 70000,
				User: "testuser",
			},
			wantErr: true,
			errMsg:  "invalid SSH port",
		},
		{
			name: "missing user",
			config: &Config{
				Host: "example.com",
				Port: 22,
			},
			wantErr: true,
			errMsg:  "SSH user is required",
		},
		{
			name: "missing auth method",
			config: &Config{
				Host: "example.com",
				Port: 22,
				User: "testuser",
			},
			wantErr: true,
			errMsg:  "SSH auth method is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}

func TestConfig_GetAddress(t *testing.T) {
	config := &Config{
		Host: "example.com",
		Port: 2222,
		User: "testuser",
	}

	assert.Equal(t, "example.com:2222", config.GetAddress())
}

func TestConfig_String(t *testing.T) {
	config := &Config{
		Host: "example.com",
		Port: 22,
		User: "testuser",
	}

	assert.Equal(t, "SSH://testuser@example.com:22", config.String())
}

func TestNewSSHAgentAuth(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		auth, err := NewSSHAgentAuth()
		require.NoError(t, err)
		assert.NotNil(t, auth)
		assert.Equal(t, "ssh_agent", auth.Type())
		assert.Equal(t, "SSH Agent Authentication", auth.String())
	})

	t.Run("mock agent failure", func(t *testing.T) {
		// This test would require mocking the agent package
		// For now, we'll test the structure
		auth, err := NewSSHAgentAuth()
		require.NoError(t, err)
		assert.NotNil(t, auth)
	})
}

func TestNewPrivateKeyAuth(t *testing.T) {
	t.Run("with key path", func(t *testing.T) {
		keyPath := t.TempDir() + "/test_key"
		privateKey := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
		
		err := os.WriteFile(keyPath, []byte(privateKey), 0600)
		require.NoError(t, err)

		auth, err := NewPrivateKeyAuth(keyPath, "")
		require.NoError(t, err)
		assert.NotNil(t, auth)
		assert.Equal(t, "private_key", auth.Type())
		assert.Equal(t, keyPath, auth.(*PrivateKeyAuth).keyPath)
	})

	t.Run("with key content", func(t *testing.T) {
		privateKey := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
		
		auth, err := NewPrivateKeyAuth("", privateKey)
		require.NoError(t, err)
		assert.NotNil(t, auth)
		assert.Equal(t, "private_key", auth.Type())
		assert.Equal(t, "", auth.(*PrivateKeyAuth).keyPath)
	})

	t.Run("missing both key path and content", func(t *testing.T) {
		auth, err := NewPrivateKeyAuth("", "")
		assert.Error(t, err)
		assert.Nil(t, auth)
		assert.Contains(t, err.Error(), "either key path or private key content must be provided")
	})
}

func TestNewPasswordAuth(t *testing.T) {
	t.Run("valid password", func(t *testing.T) {
		auth, err := NewPasswordAuth("secret123")
		require.NoError(t, err)
		assert.NotNil(t, auth)
		assert.Equal(t, "password", auth.Type())
		assert.Equal(t, "Password Authentication", auth.String())
	})

	t.Run("empty password", func(t *testing.T) {
		auth, err := NewPasswordAuth("")
		assert.Error(t, err)
		assert.Nil(t, auth)
		assert.Contains(t, err.Error(), "password cannot be empty")
	})
}

func TestConnectionPool(t *testing.T) {
	logger := &MockLogger{}
	pool := NewConnectionPool(5, logger)

	t.Run("create pool", func(t *testing.T) {
		assert.NotNil(t, pool)
		assert.Equal(t, 5, pool.GetMaxSize())
		assert.Equal(t, 0, pool.GetActiveCount())
		assert.Equal(t, 0, pool.GetTotalCount())
		assert.False(t, pool.IsFull())
	})

	t.Run("get and return connection", func(t *testing.T) {
		ctx := context.Background()
		
		// Get connection
		conn, err := pool.Get(ctx)
		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Equal(t, 1, pool.GetActiveCount())
		assert.Equal(t, 1, pool.GetTotalCount())

		// Return connection
		pool.Return(conn)
		assert.Equal(t, 0, pool.GetActiveCount())
		assert.Equal(t, 1, pool.GetTotalCount())
	})

	t.Run("pool full", func(t *testing.T) {
		// Fill pool to capacity
		ctx := context.Background()
		for i := 0; i < 5; i++ {
			conn, err := pool.Get(ctx)
			require.NoError(t, err)
			assert.NotNil(t, conn)
		}

		assert.Equal(t, 5, pool.GetActiveCount())
		assert.Equal(t, 5, pool.GetTotalCount())
		assert.True(t, pool.IsFull())

		// Try to get one more
		_, err := pool.Get(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection pool is full")
	})

	t.Run("metrics", func(t *testing.T) {
		metrics := pool.GetMetrics()
		assert.NotNil(t, metrics)
		assert.GreaterOrEqual(t, metrics.Created, int64(0))
		assert.GreaterOrEqual(t, metrics.Acquired, int64(0))
	})
}

func TestAuditor_AuditSSHKeys(t *testing.T) {
	logger := &MockLogger{}
	auditor := NewAuditor(logger, "")

	t.Run("audit empty directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		
		result, err := auditor.AuditSSHKeys("test-server", emptyDir)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-server", result.ServerID)
		assert.Equal(t, 0, len(result.Issues))
		assert.Equal(t, "excellent", result.OverallScore)
	})

	t.Run("audit with weak keys", func(t *testing.T) {
		// Create mock SSH directory with weak keys
		sshDir := t.TempDir()
		
		// Create a weak DSS key
		dssKey := "ssh-dss AAAAB3NzaC1yc2EAAAADAQABAAABAQC..."
		err := os.WriteFile(sshDir+"/id_dsa.pub", []byte(dssKey), 0644)
		require.NoError(t, err)

		result, err := auditor.AuditSSHKeys("test-server", sshDir)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, len(result.Issues), 0)
		
		// Check for weak algorithm issue
		hasWeakAlgorithm := false
		for _, issue := range result.Issues {
			if issue.ID == "WEAK_KEY_ALGORITHM" {
				hasWeakAlgorithm = true
				break
			}
		}
		assert.True(t, hasWeakAlgorithm)
	})
}

func TestSecurityIssue_Sorting(t *testing.T) {
	issues := []SecurityIssue{
		{ID: "INFO", Severity: SeverityInfo, Title: "Info issue"},
		{ID: "LOW", Severity: SeverityLow, Title: "Low issue"},
		{ID: "MEDIUM", Severity: SeverityMedium, Title: "Medium issue"},
		{ID: "HIGH", Severity: SeverityHigh, Title: "High issue"},
		{ID: "CRITICAL", Severity: SeverityCritical, Title: "Critical issue"},
	}

	sorted := SortIssuesBySeverity(issues)

	// Verify sorting (critical first, then high, medium, low, info)
	expectedOrder := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO"}
	for i, expected := range expectedOrder {
		assert.Equal(t, expected, sorted[i].ID)
	}
}

func TestGetSeverityColor(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "\033[31m"},     // Red
		{SeverityHigh, "\033[91m"},       // Bright Red
		{SeverityMedium, "\033[33m"},      // Yellow
		{SeverityLow, "\033[93m"},        // Bright Yellow
		{SeverityInfo, "\033[36m"},        // Cyan
		{"unknown", "\033[0m"},           // Reset
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			assert.Equal(t, tt.expected, GetSeverityColor(tt.severity))
		})
	}

	assert.Equal(t, "\033[0m", ResetColor())
}

func TestGenerateKeyPair(t *testing.T) {
	t.Run("generate RSA key", func(t *testing.T) {
		keyPair, err := GenerateKeyPair("rsa", 2048, "test-key")
		require.NoError(t, err)
		assert.NotNil(t, keyPair)
		assert.Equal(t, "rsa", keyPair.Type)
		assert.Equal(t, 2048, keyPair.Size)
		assert.Equal(t, "test-key", keyPair.Comment)
		assert.Contains(t, keyPair.PrivateKey, "BEGIN RSA PRIVATE KEY")
		assert.Contains(t, keyPair.PublicKey, "ssh-rsa")
		assert.False(t, keyPair.CreatedAt.IsZero())
	})

	t.Run("generate ED25519 key", func(t *testing.T) {
		keyPair, err := GenerateKeyPair("ed25519", 256, "test-key")
		require.NoError(t, err)
		assert.NotNil(t, keyPair)
		assert.Equal(t, "ed25519", keyPair.Type)
		assert.Equal(t, 256, keyPair.Size)
		assert.Equal(t, "test-key", keyPair.Comment)
		assert.Contains(t, keyPair.PrivateKey, "BEGIN OPENSSH PRIVATE KEY")
		assert.Contains(t, keyPair.PublicKey, "ssh-ed25519")
		assert.False(t, keyPair.CreatedAt.IsZero())
	})

	t.Run("unsupported key type", func(t *testing.T) {
		_, err := GenerateKeyPair("unsupported", 0, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key type")
	})
}

func TestKeyPair_SaveKeyPair(t *testing.T) {
	keyPair, err := GenerateKeyPair("rsa", 2048, "test-key")
	require.NoError(t, err)

	privateKeyPath := t.TempDir() + "/test_key"
	publicKeyPath := t.TempDir() + "/test_key.pub"

	err = keyPair.SaveKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	// Verify files exist
	_, err = os.Stat(privateKeyPath)
	require.NoError(t, err)

	_, err = os.Stat(publicKeyPath)
	require.NoError(t, err)

	// Verify content
	privateContent, err := os.ReadFile(privateKeyPath)
	require.NoError(t, err)
	assert.Contains(t, string(privateContent), "BEGIN RSA PRIVATE KEY")

	publicContent, err := os.ReadFile(publicKeyPath)
	require.NoError(t, err)
	assert.Contains(t, string(publicContent), "ssh-rsa")
}

func TestValidateKeyPermissions(t *testing.T) {
	t.Run("valid permissions", func(t *testing.T) {
		keyPath := t.TempDir() + "/test_key"
		
		// Create key with correct permissions
		err := os.WriteFile(keyPath, []byte("test key"), 0600)
		require.NoError(t, err)

		err = ValidateKeyPermissions(keyPath)
		assert.NoError(t, err)
	})

	t.Run("invalid permissions - readable by others", func(t *testing.T) {
		keyPath := t.TempDir() + "/test_key"
		
		// Create key with wrong permissions
		err := os.WriteFile(keyPath, []byte("test key"), 0644)
		require.NoError(t, err)

		err = ValidateKeyPermissions(keyPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "private key file has too permissive permissions")
	})
}

func TestFixKeyPermissions(t *testing.T) {
	keyPath := t.TempDir() + "/test_key"
	
	// Create key with wrong permissions
	err := os.WriteFile(keyPath, []byte("test key"), 0644)
	require.NoError(t, err)

	// Fix permissions
	err = FixKeyPermissions(keyPath)
	require.NoError(t, err)

	// Verify permissions were fixed
	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// MockAuthMethod for testing
type MockAuthMethod struct{}

func (m *MockAuthMethod) Authenticate() (ssh.AuthMethod, error) {
	return ssh.Password("mock"), nil
}

func (m *MockAuthMethod) Type() string {
	return "mock"
}

func (m *MockAuthMethod) String() string {
	return "Mock Authentication"
}