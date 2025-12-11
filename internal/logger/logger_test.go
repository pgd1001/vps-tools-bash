package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &LoggerConfig{
			Level:  InfoLevel,
			Format: TextFormat,
			Output: "stdout",
		}

		logger, err := NewLogger(config)
		require.NoError(t, err)
		assert.NotNil(t, logger)
		assert.Equal(t, InfoLevel, logger.GetLevel())
		assert.Equal(t, TextFormat, logger.GetFormat())
	})

	t.Run("invalid level", func(t *testing.T) {
		config := &LoggerConfig{
			Level:  "invalid",
			Format: TextFormat,
			Output: "stdout",
		}

		logger, err := NewLogger(config)
		assert.Error(t, err)
		assert.Nil(t, logger)
	})

	t.Run("invalid format", func(t *testing.T) {
		config := &LoggerConfig{
			Level:  InfoLevel,
			Format: "invalid",
			Output: "stdout",
		}

		logger, err := NewLogger(config)
		assert.Error(t, err)
		assert.Nil(t, logger)
	})

	t.Run("file output", func(t *testing.T) {
		logFile := t.TempDir() + "/test.log"
		config := &LoggerConfig{
			Level:  InfoLevel,
			Format: TextFormat,
			Output: logFile,
		}

		logger, err := NewLogger(config)
		require.NoError(t, err)
		assert.NotNil(t, logger)

		// Test writing to file
		logger.Info("test message")

		// Verify file was created and contains message
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test message")
	})
}

func TestNewDefaultLogger(t *testing.T) {
	logger, err := NewDefaultLogger()
	require.NoError(t, err)
	assert.NotNil(t, logger)
	assert.Equal(t, InfoLevel, logger.GetLevel())
	assert.Equal(t, TextFormat, logger.GetFormat())
}

func TestLogger_WithMethods(t *testing.T) {
	config := &LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: "stdout",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Test WithField
	entry := logger.WithField("key", "value")
	assert.NotNil(t, entry)

	// Test WithFields
	entry = logger.WithFields(map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	})
	assert.NotNil(t, entry)

	// Test WithError
	entry = logger.WithError(assert.AnError)
	assert.NotNil(t, entry)

	// Test WithServerID
	entry = logger.WithServerID("server-1")
	assert.NotNil(t, entry)

	// Test WithJobID
	entry = logger.WithJobID("job-1")
	assert.NotNil(t, entry)

	// Test WithComponent
	entry = logger.WithComponent("test-component")
	assert.NotNil(t, entry)

	// Test WithDuration
	entry = logger.WithDuration(time.Second * 5)
	assert.NotNil(t, entry)

	// Test WithUser
	entry = logger.WithUser("testuser")
	assert.NotNil(t, entry)

	// Test WithCommand
	entry = logger.WithCommand("uptime")
	assert.NotNil(t, entry)

	// Test WithExitCode
	entry = logger.WithExitCode(0)
	assert.NotNil(t, entry)
}

func TestLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: "stdout",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Redirect output to buffer for testing
	logger.SetOutput(&buf)

	// Test different log levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Parse JSON output
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var entry map[string]interface{}
		err := json.Unmarshal(line, &entry)
		require.NoError(t, err)

		level, ok := entry["level"].(string)
		require.True(t, ok)
		assert.Contains(t, []string{"debug", "info", "warning", "error"}, level)
	}
}

func TestLogger_SetLevel(t *testing.T) {
	logger, err := NewDefaultLogger()
	require.NoError(t, err)

	// Test setting valid levels
	err = logger.SetLevel(DebugLevel)
	require.NoError(t, err)
	assert.Equal(t, DebugLevel, logger.GetLevel())

	err = logger.SetLevel(ErrorLevel)
	require.NoError(t, err)
	assert.Equal(t, ErrorLevel, logger.GetLevel())

	// Test setting invalid level
	err = logger.SetLevel("invalid")
	assert.Error(t, err)
	assert.Equal(t, ErrorLevel, logger.GetLevel()) // Should remain unchanged
}

func TestLogger_SetFormat(t *testing.T) {
	logger, err := NewDefaultLogger()
	require.NoError(t, err)

	// Test setting valid formats
	err = logger.SetFormat(JSONFormat)
	require.NoError(t, err)
	assert.Equal(t, JSONFormat, logger.GetFormat())

	err = logger.SetFormat(TextFormat)
	require.NoError(t, err)
	assert.Equal(t, TextFormat, logger.GetFormat())

	// Test setting invalid format
	err = logger.SetFormat("invalid")
	assert.Error(t, err)
	assert.Equal(t, TextFormat, logger.GetFormat()) // Should remain unchanged
}

func TestNewAuditLogger(t *testing.T) {
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: TextFormat, // Should be overridden to JSON
		Output: "stdout",
	}

	auditLogger, err := NewAuditLogger(config)
	require.NoError(t, err)
	assert.NotNil(t, auditLogger)
	assert.Equal(t, JSONFormat, auditLogger.GetFormat()) // Should always be JSON
}

func TestAuditLogger_LogSSHConnection(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: "stdout",
	}

	auditLogger, err := NewAuditLogger(config)
	require.NoError(t, err)
	auditLogger.SetOutput(&buf)

	// Test successful connection
	auditLogger.LogSSHConnection("server-1", "ubuntu", "192.168.1.100", true, nil)

	// Test failed connection
	auditLogger.LogSSHConnection("server-2", "root", "192.168.1.101", false, assert.AnError)

	// Parse JSON output
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	assert.GreaterOrEqual(t, len(lines), 2)

	// Check first log entry (successful connection)
	var entry map[string]interface{}
	err = json.Unmarshal(lines[0], &entry)
	require.NoError(t, err)
	assert.Equal(t, "ssh_connection", entry["event_type"])
	assert.Equal(t, "server-1", entry["server_id"])
	assert.Equal(t, "ubuntu", entry["user"])
	assert.Equal(t, "192.168.1.100", entry["source_ip"])
	assert.Equal(t, true, entry["success"])
}

func TestAuditLogger_LogCommandExecution(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: "stdout",
	}

	auditLogger, err := NewAuditLogger(config)
	require.NoError(t, err)
	auditLogger.SetOutput(&buf)

	// Test successful command
	auditLogger.LogCommandExecution("server-1", "ubuntu", "uptime", 0, time.Second*2, "10:30:15 up 5 days", "")

	// Parse JSON output
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	assert.Greater(t, len(lines), 0)

	var entry map[string]interface{}
	err = json.Unmarshal(lines[0], &entry)
	require.NoError(t, err)
	assert.Equal(t, "command_execution", entry["event_type"])
	assert.Equal(t, "server-1", entry["server_id"])
	assert.Equal(t, "ubuntu", entry["user"])
	assert.Equal(t, "uptime", entry["command"])
	assert.Equal(t, float64(0), entry["exit_code"])
	assert.Equal(t, "2s", entry["duration"])
	assert.Equal(t, "10:30:15 up 5 days", entry["stdout"])
}

func TestAuditLogger_LogConfigurationChange(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: "stdout",
	}

	auditLogger, err := NewAuditLogger(config)
	require.NoError(t, err)
	auditLogger.SetOutput(&buf)

	oldConfig := map[string]interface{}{"key": "old_value"}
	newConfig := map[string]interface{}{"key": "new_value"}

	auditLogger.LogConfigurationChange("admin", "ssh", "update", oldConfig, newConfig)

	// Parse JSON output
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	assert.Greater(t, len(lines), 0)

	var entry map[string]interface{}
	err = json.Unmarshal(lines[0], &entry)
	require.NoError(t, err)
	assert.Equal(t, "configuration_change", entry["event_type"])
	assert.Equal(t, "admin", entry["user"])
	assert.Equal(t, "ssh", entry["component"])
	assert.Equal(t, "update", entry["action"])
	assert.Equal(t, oldConfig, entry["old_config"])
	assert.Equal(t, newConfig, entry["new_config"])
}

func TestAuditLogger_LogSecurityEvent(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: "stdout",
	}

	auditLogger, err := NewAuditLogger(config)
	require.NoError(t, err)
	auditLogger.SetOutput(&buf)

	details := map[string]interface{}{
		"source_ip": "192.168.1.100",
		"attempts":  5,
	}

	auditLogger.LogSecurityEvent("brute_force", "server-1", "Multiple failed login attempts", details)

	// Parse JSON output
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	assert.Greater(t, len(lines), 0)

	var entry map[string]interface{}
	err = json.Unmarshal(lines[0], &entry)
	require.NoError(t, err)
	assert.Equal(t, "security_event", entry["event_type"])
	assert.Equal(t, "brute_force", entry["type"])
	assert.Equal(t, "server-1", entry["server_id"])
	assert.Equal(t, "Multiple failed login attempts", entry["description"])
	assert.Equal(t, details, entry["details"])
}

func TestGetCaller(t *testing.T) {
	funcName, fileName, line := GetCaller(1)
	assert.NotEmpty(t, funcName)
	assert.NotEmpty(t, fileName)
	assert.Greater(t, line, 0)
	assert.Contains(t, fileName, "logger_test.go")
	assert.Contains(t, funcName, "TestGetCaller")
}