package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// LogLevel represents the logging level
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
)

// LogFormat represents the log output format
type LogFormat string

const (
	TextFormat LogFormat = "text"
	JSONFormat LogFormat = "json"
)

// Logger represents the application logger
type Logger struct {
	*logrus.Logger
	config *LoggerConfig
}

// LoggerConfig represents logger configuration
type LoggerConfig struct {
	Level      LogLevel `yaml:"level"`
	Format     LogFormat `yaml:"format"`
	Output     string   `yaml:"output"`     // stdout, stderr, file path
	MaxSize    int      `yaml:"max_size"`   // max file size in MB
	MaxBackups int      `yaml:"max_backups"` // max number of backup files
	MaxAge     int      `yaml:"max_age"`    // max age in days
	Compress   bool     `yaml:"compress"`    // compress old log files
}

// NewLogger creates a new logger with the given configuration
func NewLogger(config *LoggerConfig) (*Logger, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(string(config.Level))
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logger.SetLevel(level)

	// Set log format
	switch config.Format {
	case JSONFormat:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
				logrus.FieldKeyFile:  "file",
			},
		})
	case TextFormat:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	default:
		return nil, fmt.Errorf("invalid log format: %s", config.Format)
	}

	// Set output
	output, err := getOutput(config)
	if err != nil {
		return nil, fmt.Errorf("failed to set log output: %w", err)
	}
	logger.SetOutput(output)

	// Add caller information
	logger.SetReportCaller(true)

	return &Logger{
		Logger: logger,
		config: config,
	}, nil
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() (*Logger, error) {
	config := &LoggerConfig{
		Level:      InfoLevel,
		Format:     TextFormat,
		Output:     "stdout",
		MaxSize:    100, // 100MB
		MaxBackups: 3,
		MaxAge:     28, // 28 days
		Compress:   true,
	}
	return NewLogger(config)
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithError adds an error field to the logger
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// Debug logs a debug message
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
}

// Debugf logs a debug message with formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
}

// Infof logs an info message with formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(args ...interface{}) {
	l.Logger.Warn(args...)
}

// Warnf logs a warning message with formatting
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warnf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
}

// Errorf logs an error message with formatting
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
}

// Fatalf logs a fatal message with formatting and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatalf(format, args...)
}

// Panic logs a panic message and panics
func (l *Logger) Panic(args ...interface{}) {
	l.Logger.Panic(args...)
}

// Panicf logs a panic message with formatting and panics
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.Logger.Panicf(format, args...)
}

// WithServerID adds server ID field to the logger
func (l *Logger) WithServerID(serverID string) *logrus.Entry {
	return l.WithField("server_id", serverID)
}

// WithJobID adds job ID field to the logger
func (l *Logger) WithJobID(jobID string) *logrus.Entry {
	return l.WithField("job_id", jobID)
}

// WithComponent adds component field to the logger
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.WithField("component", component)
}

// WithDuration adds duration field to the logger
func (l *Logger) WithDuration(duration interface{}) *logrus.Entry {
	return l.WithField("duration", duration)
}

// WithUser adds user field to the logger
func (l *Logger) WithUser(user string) *logrus.Entry {
	return l.WithField("user", user)
}

// WithCommand adds command field to the logger
func (l *Logger) WithCommand(command string) *logrus.Entry {
	return l.WithField("command", command)
}

// WithExitCode adds exit code field to the logger
func (l *Logger) WithExitCode(exitCode int) *logrus.Entry {
	return l.WithField("exit_code", exitCode)
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	return LogLevel(l.Logger.GetLevel().String())
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) error {
	logLevel, err := logrus.ParseLevel(string(level))
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	l.Logger.SetLevel(logLevel)
	l.config.Level = level
	return nil
}

// GetFormat returns the current log format
func (l *Logger) GetFormat() LogFormat {
	return l.config.Format
}

// SetFormat sets the log format
func (l *Logger) SetFormat(format LogFormat) error {
	switch format {
	case JSONFormat:
		l.Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case TextFormat:
		l.Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	default:
		return fmt.Errorf("invalid log format: %s", format)
	}
	l.config.Format = format
	return nil
}

// getOutput returns the appropriate output writer based on configuration
func getOutput(config *LoggerConfig) (io.Writer, error) {
	switch strings.ToLower(config.Output) {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		// Assume it's a file path
		return createFileWriter(config)
	}
}

// createFileWriter creates a file writer with rotation support
func createFileWriter(config *LoggerConfig) (io.Writer, error) {
	// Ensure directory exists
	dir := filepath.Dir(config.Output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// For now, just return a simple file writer
	// In a production environment, you might want to use lumberjack for log rotation
	return os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// GetCaller returns the calling function information
func GetCaller(skip int) (string, string, int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", "", 0
	}

	funcName := runtime.FuncForPC(pc).Name()
	fileName := filepath.Base(file)

	return funcName, fileName, line
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	ServerID  string                 `json:"server_id,omitempty"`
	JobID     string                 `json:"job_id,omitempty"`
	User      string                 `json:"user,omitempty"`
	Command   string                 `json:"command,omitempty"`
	ExitCode  int                    `json:"exit_code,omitempty"`
	Duration  string                 `json:"duration,omitempty"`
	Function  string                 `json:"function,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// AuditLogger represents a logger for audit events
type AuditLogger struct {
	*Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *LoggerConfig) (*AuditLogger, error) {
	// Audit logs should always be in JSON format for structured processing
	auditConfig := *config
	auditConfig.Format = JSONFormat
	
	logger, err := NewLogger(&auditConfig)
	if err != nil {
		return nil, err
	}

	return &AuditLogger{
		Logger: logger,
	}, nil
}

// LogSSHConnection logs an SSH connection event
func (a *AuditLogger) LogSSHConnection(serverID, user, sourceIP string, success bool, err error) {
	fields := logrus.Fields{
		"event_type": "ssh_connection",
		"server_id":  serverID,
		"user":       user,
		"source_ip":  sourceIP,
		"success":    success,
		"timestamp":  time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
	}

	if err != nil {
		fields["error"] = err.Error()
	}

	if success {
		a.WithFields(fields).Info("SSH connection established")
	} else {
		a.WithFields(fields).Error("SSH connection failed")
	}
}

// LogCommandExecution logs a command execution event
func (a *AuditLogger) LogCommandExecution(serverID, user, command string, exitCode int, duration time.Duration, stdout, stderr string) {
	fields := logrus.Fields{
		"event_type": "command_execution",
		"server_id":  serverID,
		"user":       user,
		"command":    command,
		"exit_code":  exitCode,
		"duration":   duration.String(),
		"stdout":     stdout,
		"stderr":     stderr,
		"timestamp":  time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
	}

	if exitCode == 0 {
		a.WithFields(fields).Info("Command executed successfully")
	} else {
		a.WithFields(fields).Error("Command execution failed")
	}
}

// LogConfigurationChange logs a configuration change event
func (a *AuditLogger) LogConfigurationChange(user, component, action string, oldConfig, newConfig interface{}) {
	fields := logrus.Fields{
		"event_type": "configuration_change",
		"user":       user,
		"component":  component,
		"action":     action,
		"old_config": oldConfig,
		"new_config": newConfig,
		"timestamp":  time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
	}

	a.WithFields(fields).Info("Configuration changed")
}

// LogSecurityEvent logs a security-related event
func (a *AuditLogger) LogSecurityEvent(eventType, serverID, description string, details map[string]interface{}) {
	fields := logrus.Fields{
		"event_type":  "security_event",
		"type":        eventType,
		"server_id":   serverID,
		"description": description,
		"details":     details,
		"timestamp":   time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
	}

	a.WithFields(fields).Warn("Security event detected")
}