package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/server"
	"github.com/pgd1001/vps-tools/internal/store"
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
	m.infoMessages = append(m.infoMessages, fmt.Sprint(args...))
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.infoMessages = append(m.infoMessages, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Warn(args ...interface{}) {
	m.warnMessages = append(m.warnMessages, fmt.Sprint(args...))
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.warnMessages = append(m.warnMessages, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Error(args ...interface{}) {
	m.errorMessages = append(m.errorMessages, fmt.Sprint(args...))
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.errorMessages = append(m.errorMessages, fmt.Sprintf(format, args...))
}

func TestMigrator_ParseBashScripts(t *testing.T) {
	// Create temporary bash scripts directory
	bashDir := t.TempDir()
	
	// Create a mock vps-build.sh
	vpsBuildContent := `#!/bin/bash
# Ubuntu 24.04 VPS Provisioning Script
DEFAULT_USER="ubuntu"
DEFAULT_PORT="2222"
echo "Setting up server with user: $DEFAULT_USER"
`
	err := os.WriteFile(filepath.Join(bashDir, "vps-build.sh"), []byte(vpsBuildContent), 0755)
	require.NoError(t, err)

	// Create a mock cron config
	cronContent := `MAILTO=admin@example.com
# Every 5 minutes
*/5 * * * * root /opt/vps-tools/monitoring/vps-health-monitor.sh
# Daily at 2 AM
0 2 * * * root /opt/vps-tools/monitoring/vps-log-analyzer.sh
`
	err = os.WriteFile(filepath.Join(bashDir, "vps-tools-cron.conf"), []byte(cronContent), 0644)
	require.NoError(t, err)

	// Test parsing
	migrator := &Migrator{}
	bashConfig, err := migrator.parseBashScripts(bashDir)
	require.NoError(t, err)

	assert.Len(t, bashConfig.Servers, 1)
	assert.Equal(t, "default-server", bashConfig.Servers[0].ID)
	assert.Equal(t, "Default Server", bashConfig.Servers[0].Hostname)
	assert.Equal(t, "127.0.0.1", bashConfig.Servers[0].IP)
	assert.Equal(t, 2222, bashConfig.Servers[0].Port)
	assert.Equal(t, "ubuntu", bashConfig.Servers[0].User)
	assert.Equal(t, "admin@example.com", bashConfig.Cron.MailTo)
	assert.Equal(t, "*/5 * * * *", bashConfig.Cron.Schedules["health_monitor"])
	assert.Equal(t, "0 2 * * *", bashConfig.Cron.Schedules["log_analysis"])
}

func TestMigrator_ParseBashScripts_NoFiles(t *testing.T) {
	// Test with empty directory
	bashDir := t.TempDir()

	migrator := &Migrator{}
	bashConfig, err := migrator.parseBashScripts(bashDir)
	require.NoError(t, err)

	assert.Len(t, bashConfig.Servers, 1) // Should create default server
	assert.Equal(t, "localhost", bashConfig.Servers[0].ID)
	assert.Equal(t, []string{"local", "migrated"}, bashConfig.Servers[0].Tags)
}

func TestMigrator_ConvertToVPSConfig(t *testing.T) {
	bashConfig := &BashScriptConfig{
		Servers: []BashServer{
			{
				ID:       "web-01",
				Hostname: "Web Server 1",
				IP:       "192.168.1.10",
				Port:     22,
				User:     "ubuntu",
				SSHKey:   "/home/user/.ssh/id_rsa",
				Tags:     []string{"web", "production"},
				Meta: map[string]string{
					"location": "us-east-1",
					"role":     "web",
				},
			},
		},
		Cron: BashCron{
			MailTo: "admin@example.com",
		},
		Env: BashEnv{
			DataDir: "/opt/vps-tools/data",
		},
	}

	migrator := &Migrator{}
	vpsConfig, err := migrator.convertToVPSConfig(bashConfig)
	require.NoError(t, err)

	assert.Equal(t, "vps-tools", vpsConfig.App.Name)
	assert.Equal(t, "1.0.0", vpsConfig.App.Version)
	assert.Equal(t, "info", vpsConfig.App.LogLevel)
	assert.True(t, vpsConfig.Monitoring.Enabled)
	assert.Equal(t, "5m", vpsConfig.Monitoring.Interval)
	assert.Equal(t, 80, vpsConfig.Monitoring.Thresholds.DiskWarning)
	assert.Equal(t, 90, vpsConfig.Monitoring.Thresholds.DiskCritical)
	assert.True(t, vpsConfig.Security.SSHAuditEnabled)
	assert.Equal(t, 10, vpsConfig.Security.FailedLoginThreshold)
	assert.Equal(t, "bolt", vpsConfig.Storage.Type)
	assert.Equal(t, "/opt/vps-tools/data/vps-tools.db", vpsConfig.Storage.BoltDBPath)
	assert.Equal(t, "admin@example.com", vpsConfig.Alerts.Email)

	assert.Len(t, vpsConfig.Servers, 1)
	server := vpsConfig.Servers[0]
	assert.Equal(t, "web-01", server.ID)
	assert.Equal(t, "Web Server 1", server.Name)
	assert.Equal(t, "192.168.1.10", server.Host)
	assert.Equal(t, 22, server.Port)
	assert.Equal(t, "ubuntu", server.User)
	assert.Equal(t, []string{"web", "production"}, server.Tags)
	assert.Equal(t, "us-east-1", server.Meta["location"])
	assert.Equal(t, "web", server.Meta["role"])

	// Check SSH auth method
	assert.Equal(t, "private_key", server.AuthMethod["type"])
	assert.Equal(t, "/home/user/.ssh/id_rsa", server.AuthMethod["key_path"])
	assert.Equal(t, false, server.AuthMethod["use_agent"])
}

func TestMigrator_ConvertToVPSConfig_SSHAgent(t *testing.T) {
	bashConfig := &BashScriptConfig{
		Servers: []BashServer{
			{
				ID:       "agent-server",
				Hostname: "Agent Server",
				IP:       "192.168.1.20",
				Port:     22,
				User:     "ubuntu",
				// No SSHKey - should use SSH agent
				Tags: []string{"agent"},
			},
		},
	}

	migrator := &Migrator{}
	vpsConfig, err := migrator.convertToVPSConfig(bashConfig)
	require.NoError(t, err)

	server := vpsConfig.Servers[0]
	assert.Equal(t, "ssh_agent", server.AuthMethod["type"])
	assert.Equal(t, true, server.AuthMethod["use_agent"])
}

func TestMigrator_MigrateServer(t *testing.T) {
	// Create temporary database
	dbPath := t.TempDir() + "/test.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := store.NewBoltStore(cfg)
	require.NoError(t, err)
	defer store.Close()

	logger := &MockLogger{}
	migrator := NewMigrator(nil, store, logger)

	bashServer := BashServer{
		ID:       "test-server",
		Hostname: "Test Server",
		IP:       "192.168.1.30",
		Port:     22,
		User:     "ubuntu",
		SSHKey:   "/home/user/.ssh/id_rsa",
		Tags:     []string{"test"},
		Meta: map[string]string{
			"migrated": "true",
		},
	}

	err = migrator.migrateServer(bashServer)
	require.NoError(t, err)

	// Verify server was created
	srv, err := store.GetServer("test-server")
	require.NoError(t, err)
	assert.Equal(t, "test-server", srv.ID)
	assert.Equal(t, "Test Server", srv.Name)
	assert.Equal(t, "192.168.1.30", srv.Host)
	assert.Equal(t, 22, srv.Port)
	assert.Equal(t, "ubuntu", srv.User)
	assert.Equal(t, []string{"test"}, srv.Tags)
	assert.Equal(t, "true", srv.Meta["migrated"])
	assert.Equal(t, "private_key", srv.AuthMethod.Type)
	assert.Equal(t, "/home/user/.ssh/id_rsa", srv.AuthMethod.KeyPath)
}

func TestMigrator_MigrateServer_Invalid(t *testing.T) {
	// Create temporary database
	dbPath := t.TempDir() + "/test.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := store.NewBoltStore(cfg)
	require.NoError(t, err)
	defer store.Close()

	logger := &MockLogger{}
	migrator := NewMigrator(nil, store, logger)

	bashServer := BashServer{
		ID:       "", // Invalid - missing ID
		Hostname: "Invalid Server",
		IP:       "192.168.1.40",
		Port:     22,
		User:     "ubuntu",
	}

	err = migrator.migrateServer(bashServer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server configuration")
}

func TestMigrator_ExportConfiguration(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:    "test-app",
			Version: "1.0.0",
		},
		Storage: config.StorageConfig{
			Type: "bolt",
		},
	}

	configManager := &config.ConfigManager{}
	// Use reflection to set the private config field
	// In real code, you'd have a proper setter
	configValue := reflect.ValueOf(configManager).Elem()
	configField := configValue.FieldByName("config")
	configField.Set(reflect.ValueOf(cfg).Elem())

	logger := &MockLogger{}
	migrator := NewMigrator(configManager, nil, logger)

	// Test YAML export
	yamlPath := t.TempDir() + "/config.yaml"
	err := migrator.ExportConfiguration("yaml", yamlPath)
	require.NoError(t, err)

	content, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test-app")
	assert.Contains(t, string(content), "bolt")

	// Test JSON export
	jsonPath := t.TempDir() + "/config.json"
	err = migrator.ExportConfiguration("json", jsonPath)
	require.NoError(t, err)

	content, err = os.ReadFile(jsonPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test-app")
	assert.Contains(t, string(content), "bolt")

	// Test ENV export
	envPath := t.TempDir() + "/config.env"
	err = migrator.ExportConfiguration("env", envPath)
	require.NoError(t, err)

	content, err = os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "VPS_TOOLS_NAME=test-app")
	assert.Contains(t, string(content), "VPS_TOOLS_DB_TYPE=bolt")

	// Test invalid format
	invalidPath := t.TempDir() + "/config.invalid"
	err = migrator.ExportConfiguration("invalid", invalidPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestMigrator_ValidateMigration(t *testing.T) {
	// Create temporary database
	dbPath := t.TempDir() + "/test.db"
	cfg := &config.Config{
		Storage: config.StorageConfig{
			BoltDBPath: dbPath,
		},
	}

	store, err := store.NewBoltStore(cfg)
	require.NoError(t, err)
	defer store.Close()

	// Add a test server
	testServer := &server.Server{
		ID:   "test-server",
		Name: "Test Server",
		Host: "192.168.1.50",
		Port: 22,
		User: "ubuntu",
		AuthMethod: server.AuthConfig{
			Type: "ssh_agent",
		},
	}
	err = store.CreateServer(testServer)
	require.NoError(t, err)

	configManager := &config.ConfigManager{}
	// Set config using reflection
	configValue := reflect.ValueOf(configManager).Elem()
	configField := configValue.FieldByName("config")
	configField.Set(reflect.ValueOf(cfg).Elem())

	logger := &MockLogger{}
	migrator := NewMigrator(configManager, store, logger)

	// Test successful validation
	err = migrator.ValidateMigration()
	require.NoError(t, err)

	// Test validation with no servers
	err = store.DeleteServer("test-server")
	require.NoError(t, err)

	err = migrator.ValidateMigration()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no servers migrated")
}

func TestCopyDir(t *testing.T) {
	// Create source directory with files
	srcDir := t.TempDir() + "/src"
	err := os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)

	subDir := filepath.Join(srcDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "file3.txt"), []byte("content3"), 0644)
	require.NoError(t, err)

	// Copy directory
	dstDir := t.TempDir() + "/dst"
	err = copyDir(srcDir, dstDir)
	require.NoError(t, err)

	// Verify copied files
	content, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content))

	content, err = os.ReadFile(filepath.Join(dstDir, "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content2", string(content))

	content, err = os.ReadFile(filepath.Join(dstDir, "subdir", "file3.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content3", string(content))
}

func TestCopyFile(t *testing.T) {
	srcFile := t.TempDir() + "/src.txt"
	dstFile := t.TempDir() + "/dst.txt"

	content := []byte("test content")
	err := os.WriteFile(srcFile, content, 0644)
	require.NoError(t, err)

	err = copyFile(srcFile, dstFile)
	require.NoError(t, err)

	copiedContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, content, copiedContent)
}