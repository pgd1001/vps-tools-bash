package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigManager_Load(t *testing.T) {
	t.Run("load valid config", func(t *testing.T) {
		// Create a temporary config file
		configData := `
app:
  name: "vps-tools"
  log_level: "info"
  log_format: "json"

servers:
  - id: "web-01"
    name: "Web Server 1"
    host: "192.168.1.10"
    port: 22
    user: "ubuntu"
    auth_method:
      type: "ssh_agent"
    tags: ["web", "production"]

monitoring:
  enabled: true
  interval: "5m"
  thresholds:
    disk_warning: 80
    disk_critical: 90
    memory_warning: 75
    memory_critical: 85

security:
  ssh_audit_enabled: true
  failed_login_threshold: 10
  ssl_warning_days: 30

storage:
  type: "bolt"
  bolt_db_path: "/tmp/test.db"
`
		configPath := t.TempDir() + "/config.yaml"
		err := writeFile(configPath, configData)
		require.NoError(t, err)

		cm := NewConfigManager(configPath)
		err = cm.Load()
		require.NoError(t, err)

		config := cm.Get()
		assert.Equal(t, "vps-tools", config.App.Name)
		assert.Equal(t, "info", config.App.LogLevel)
		assert.Equal(t, "json", config.App.LogFormat)
		assert.Len(t, config.Servers, 1)
		assert.Equal(t, "web-01", config.Servers[0].ID)
		assert.Equal(t, "Web Server 1", config.Servers[0].Name)
		assert.Equal(t, "192.168.1.10", config.Servers[0].Host)
		assert.Equal(t, 22, config.Servers[0].Port)
		assert.Equal(t, "ubuntu", config.Servers[0].User)
		assert.Equal(t, []string{"web", "production"}, config.Servers[0].Tags)
		assert.True(t, config.Monitoring.Enabled)
		assert.Equal(t, "5m", config.Monitoring.Interval)
		assert.Equal(t, 80, config.Monitoring.Thresholds.DiskWarning)
		assert.Equal(t, 90, config.Monitoring.Thresholds.DiskCritical)
		assert.True(t, config.Security.SSHAuditEnabled)
		assert.Equal(t, 10, config.Security.FailedLoginThreshold)
		assert.Equal(t, 30, config.Security.SSLWarningDays)
		assert.Equal(t, "bolt", config.Storage.Type)
	})

	t.Run("create default config if not exists", func(t *testing.T) {
		configPath := t.TempDir() + "/nonexistent.yaml"
		cm := NewConfigManager(configPath)
		err := cm.Load()
		require.NoError(t, err)

		config := cm.Get()
		assert.Equal(t, "vps-tools", config.App.Name)
		assert.Equal(t, "info", config.App.LogLevel)
		assert.Equal(t, "text", config.App.LogFormat)
		assert.Equal(t, "bolt", config.Storage.Type)
		assert.Equal(t, 10, config.App.MaxWorkers)
		assert.Equal(t, 300, config.App.Timeout)
	})

	t.Run("invalid config", func(t *testing.T) {
		configData := `
app:
  name: ""
`
		configPath := t.TempDir() + "/invalid.yaml"
		err := writeFile(configPath, configData)
		require.NoError(t, err)

		cm := NewConfigManager(configPath)
		err = cm.Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid configuration")
	})
}

func TestConfigManager_Save(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	cm := NewConfigManager(configPath)
	
	// Load first to create default
	err := cm.Load()
	require.NoError(t, err)

	// Modify config
	config := cm.Get()
	config.App.Name = "test-app"
	config.App.LogLevel = "debug"

	// Save
	err = cm.Save()
	require.NoError(t, err)

	// Load again to verify
	cm2 := NewConfigManager(configPath)
	err = cm2.Load()
	require.NoError(t, err)

	savedConfig := cm2.Get()
	assert.Equal(t, "test-app", savedConfig.App.Name)
	assert.Equal(t, "debug", savedConfig.App.LogLevel)
}

func TestConfigManager_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				App: AppConfig{
					Name:      "test",
					LogLevel:  "info",
					LogFormat: "text",
				},
				Storage: StorageConfig{
					Type: "bolt",
				},
				Monitoring: MonitoringConfig{
					Thresholds: ThresholdsConfig{
						DiskWarning:  80,
						DiskCritical: 90,
					},
				},
				Servers: []ServerConfig{
					{
						ID:   "test-server",
						Name: "Test Server",
						Host: "192.168.1.10",
						Port: 22,
						User: "ubuntu",
						AuthMethod: map[string]interface{}{
							"type": "ssh_agent",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing app name",
			config: &Config{
				App: AppConfig{
					Name: "",
				},
				Storage: StorageConfig{
					Type: "bolt",
				},
			},
			expectError: true,
			errorMsg:   "app name is required",
		},
		{
			name: "invalid log level",
			config: &Config{
				App: AppConfig{
					Name:     "test",
					LogLevel: "invalid",
				},
				Storage: StorageConfig{
					Type: "bolt",
				},
			},
			expectError: true,
			errorMsg:   "invalid log level",
		},
		{
			name: "invalid storage type",
			config: &Config{
				App: AppConfig{
					Name: "test",
				},
				Storage: StorageConfig{
					Type: "invalid",
				},
			},
			expectError: true,
			errorMsg:   "invalid storage type",
		},
		{
			name: "disk warning >= critical",
			config: &Config{
				App: AppConfig{
					Name: "test",
				},
				Storage: StorageConfig{
					Type: "bolt",
				},
				Monitoring: MonitoringConfig{
					Thresholds: ThresholdsConfig{
						DiskWarning:  90,
						DiskCritical: 80,
					},
				},
			},
			expectError: true,
			errorMsg:   "disk warning threshold must be less than critical threshold",
		},
		{
			name: "invalid server port",
			config: &Config{
				App: AppConfig{
					Name: "test",
				},
				Storage: StorageConfig{
					Type: "bolt",
				},
				Servers: []ServerConfig{
					{
						ID:   "test-server",
						Name: "Test Server",
						Host: "192.168.1.10",
						Port: 70000,
						User: "ubuntu",
						AuthMethod: map[string]interface{}{
							"type": "ssh_agent",
						},
					},
				},
			},
			expectError: true,
			errorMsg:   "invalid port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &ConfigManager{}
			err := cm.validate(tt.config)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	config := &Config{}
	cm := &ConfigManager{}
	cm.applyDefaults(config)

	assert.Equal(t, "vps-tools", config.App.Name)
	assert.Equal(t, "1.0.0", config.App.Version)
	assert.Equal(t, "info", config.App.LogLevel)
	assert.Equal(t, "text", config.App.LogFormat)
	assert.Equal(t, 10, config.App.MaxWorkers)
	assert.Equal(t, 300, config.App.Timeout)
	assert.Equal(t, "bolt", config.Storage.Type)
	assert.Equal(t, "5m", config.Monitoring.Interval)
	assert.Equal(t, 80, config.Monitoring.Thresholds.DiskWarning)
	assert.Equal(t, 90, config.Monitoring.Thresholds.DiskCritical)
	assert.Equal(t, 75, config.Monitoring.Thresholds.MemoryWarning)
	assert.Equal(t, 85, config.Monitoring.Thresholds.MemoryCritical)
	assert.Equal(t, 10, config.Security.FailedLoginThreshold)
	assert.Equal(t, 30, config.Security.SSLWarningDays)
	assert.Equal(t, 90, config.Security.KeyRotationDays)
	assert.Equal(t, "root", config.SSH.DefaultUser)
	assert.Equal(t, 22, config.SSH.DefaultPort)
	assert.Equal(t, 30, config.SSH.Timeout)
	assert.Equal(t, 3, config.SSH.MaxRetries)
	assert.Equal(t, 5, config.SSH.RetryDelay)
	assert.Equal(t, 30, config.SSH.KeepAlive)
}

func TestGetDefaultConfigPath(t *testing.T) {
	// Test with environment variable
	t.Setenv("VPS_TOOLS_CONFIG", "/custom/config.yaml")
	path := getDefaultConfigPath()
	assert.Equal(t, "/custom/config.yaml", path)

	// Test without environment variable (will use home directory or fallback)
	t.Setenv("VPS_TOOLS_CONFIG", "")
	path = getDefaultConfigPath()
	// Should contain either user home directory or fallback to current directory
	assert.True(t, len(path) > 0)
}

func TestContains(t *testing.T) {
	tests := []struct {
		slice []string
		item  string
		want  bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{[]string{"a"}, "a", true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.want, contains(tt.slice, tt.item))
		})
	}
}

// Helper function to write test config file
func writeFile(path, content string) error {
	return writeFileContent(path, []byte(content), 0644)
}

// Helper function to write file content
func writeFileContent(path string, content []byte, perm os.FileMode) error {
	return os.WriteFile(path, content, perm)
}