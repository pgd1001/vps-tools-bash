package server

import "errors"

// Server validation errors
var (
	ErrServerIDRequired     = errors.New("server ID is required")
	ErrServerNameRequired   = errors.New("server name is required")
	ErrServerHostRequired   = errors.New("server host is required")
	ErrServerUserRequired   = errors.New("server user is required")
	ErrInvalidPort          = errors.New("invalid port number (must be 1-65535)")
	ErrInvalidAuthType      = errors.New("invalid authentication type")
	ErrPrivateKeyRequired    = errors.New("private key or key path is required for private key authentication")
	ErrPasswordRequired      = errors.New("password is required for password authentication")
	ErrServerNotFound       = errors.New("server not found")
	ErrServerAlreadyExists  = errors.New("server already exists")
)

// Job validation errors
var (
	ErrJobIDRequired       = errors.New("job ID is required")
	ErrJobServerIDRequired = errors.New("job server ID is required")
	ErrJobCommandRequired   = errors.New("job command is required")
	ErrJobNotFound         = errors.New("job not found")
	ErrJobAlreadyRunning    = errors.New("job is already running")
	ErrJobNotRunning       = errors.New("job is not running")
	ErrJobTimeout          = errors.New("job timed out")
)

// Health check errors
var (
	ErrHealthCheckNotFound = errors.New("health check not found")
	ErrInvalidMetric      = errors.New("invalid metric value")
)

// SSH operation errors
var (
	ErrSSHConnectionFailed = errors.New("SSH connection failed")
	ErrSSHAuthFailed      = errors.New("SSH authentication failed")
	ErrSSHCommandFailed   = errors.New("SSH command execution failed")
	ErrSSHTimeout         = errors.New("SSH operation timed out")
)

// Storage errors
var (
	ErrDatabaseNotFound    = errors.New("database not found")
	ErrDatabaseCorrupted  = errors.New("database is corrupted")
	ErrMigrationFailed    = errors.New("database migration failed")
	ErrTransactionFailed  = errors.New("database transaction failed")
)

// Configuration errors
var (
	ErrConfigNotFound     = errors.New("configuration file not found")
	ErrConfigInvalid     = errors.New("configuration is invalid")
	ErrConfigPermission  = errors.New("permission denied accessing configuration")
)