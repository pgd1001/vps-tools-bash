package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Validator interface for configuration validation
type Validator interface {
	Validate() error
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	var errors ValidationErrors

	// Validate app configuration
	if err := c.App.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "app",
				Message: err.Error(),
			})
		}
	}

	// Validate database configuration
	if err := c.Database.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "database",
				Message: err.Error(),
			})
		}
	}

	// Validate logging configuration
	if err := c.Logging.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "logging",
				Message: err.Error(),
			})
		}
	}

	// Validate SSH configuration
	if err := c.SSH.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "ssh",
				Message: err.Error(),
			})
		}
	}

	// Validate health configuration
	if err := c.Health.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "health",
				Message: err.Error(),
			})
		}
	}

	// Validate security configuration
	if err := c.Security.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "security",
				Message: err.Error(),
			})
		}
	}

	// Validate Docker configuration
	if err := c.Docker.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "docker",
				Message: err.Error(),
			})
		}
	}

	// Validate maintenance configuration
	if err := c.Maintenance.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "maintenance",
				Message: err.Error(),
			})
		}
	}

	// Validate TUI configuration
	if err := c.TUI.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "tui",
				Message: err.Error(),
			})
		}
	}

	// Validate notifications configuration
	if err := c.Notifications.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "notifications",
				Message: err.Error(),
			})
		}
	}

	// Validate plugins configuration
	if err := c.Plugins.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "plugins",
				Message: err.Error(),
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates AppConfig
func (c AppConfig) Validate() error {
	var errors ValidationErrors

	if strings.TrimSpace(c.Name) == "" {
		errors = append(errors, ValidationError{
			Field:   "app.name",
			Value:   c.Name,
			Message: "application name cannot be empty",
		})
	}

	if c.Version != "" && !isValidVersion(c.Version) {
		errors = append(errors, ValidationError{
			Field:   "app.version",
			Value:   c.Version,
			Message: "invalid version format (expected semantic version, e.g., 1.0.0)",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates DatabaseConfig
func (c DatabaseConfig) Validate() error {
	var errors ValidationErrors

	if strings.TrimSpace(c.Path) == "" {
		errors = append(errors, ValidationError{
			Field:   "database.path",
			Value:   c.Path,
			Message: "database path cannot be empty",
		})
	} else {
		// Expand path and check if directory exists
		expandedPath := expandPath(c.Path)
		dir := filepath.Dir(expandedPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   "database.path",
				Value:   c.Path,
				Message: fmt.Sprintf("database directory does not exist: %s", dir),
			})
		}
	}

	if c.BackupEnabled && c.BackupInterval <= 0 {
		errors = append(errors, ValidationError{
			Field:   "database.backup_interval",
			Value:   c.BackupInterval,
			Message: "backup interval must be positive when backup is enabled",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates LoggingConfig
func (c LoggingConfig) Validate() error {
	var errors ValidationErrors

	validLevels := []string{"debug", "info", "warn", "error"}
	if !isValidString(c.Level, validLevels) {
		errors = append(errors, ValidationError{
			Field:   "logging.level",
			Value:   c.Level,
			Message: fmt.Sprintf("invalid log level, must be one of: %s", strings.Join(validLevels, ", ")),
		})
	}

	validFormats := []string{"json", "text"}
	if !isValidString(c.Format, validFormats) {
		errors = append(errors, ValidationError{
			Field:   "logging.format",
			Value:   c.Format,
			Message: fmt.Sprintf("invalid log format, must be one of: %s", strings.Join(validFormats, ", ")),
		})
	}

	if c.File != "" {
		expandedPath := expandPath(c.File)
		dir := filepath.Dir(expandedPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   "logging.file",
				Value:   c.File,
				Message: fmt.Sprintf("log file directory does not exist: %s", dir),
			})
		}
	}

	if c.MaxSize <= 0 {
		errors = append(errors, ValidationError{
			Field:   "logging.max_size",
			Value:   c.MaxSize,
			Message: "max size must be positive",
		})
	}

	if c.MaxBackups < 0 {
		errors = append(errors, ValidationError{
			Field:   "logging.max_backups",
			Value:   c.MaxBackups,
			Message: "max backups cannot be negative",
		})
	}

	if c.MaxAge <= 0 {
		errors = append(errors, ValidationError{
			Field:   "logging.max_age",
			Value:   c.MaxAge,
			Message: "max age must be positive",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates SSHConfig
func (c SSHConfig) Validate() error {
	var errors ValidationErrors

	if c.Timeout <= 0 {
		errors = append(errors, ValidationError{
			Field:   "ssh.timeout",
			Value:   c.Timeout,
			Message: "SSH timeout must be positive",
		})
	}

	if c.MaxRetries < 0 {
		errors = append(errors, ValidationError{
			Field:   "ssh.max_retries",
			Value:   c.MaxRetries,
			Message: "max retries cannot be negative",
		})
	}

	if c.KeepAlive <= 0 {
		errors = append(errors, ValidationError{
			Field:   "ssh.keep_alive",
			Value:   c.KeepAlive,
			Message: "keep alive interval must be positive",
		})
	}

	// Validate SSH key paths
	for i, keyPath := range c.KeyPaths {
		if strings.TrimSpace(keyPath) == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("ssh.key_paths[%d]", i),
				Value:   keyPath,
				Message: "SSH key path cannot be empty",
			})
		} else {
			expandedPath := expandPath(keyPath)
			if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
				// Only warn about missing key files, don't error
				// as they might be created later
			}
		}
	}

	if c.KnownHosts != "" {
		expandedPath := expandPath(c.KnownHosts)
		dir := filepath.Dir(expandedPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   "ssh.known_hosts",
				Value:   c.KnownHosts,
				Message: fmt.Sprintf("known_hosts directory does not exist: %s", dir),
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates HealthConfig
func (c HealthConfig) Validate() error {
	var errors ValidationErrors

	if c.DefaultInterval <= 0 {
		errors = append(errors, ValidationError{
			Field:   "health.default_interval",
			Value:   c.DefaultInterval,
			Message: "default interval must be positive",
		})
	}

	if c.Timeout <= 0 {
		errors = append(errors, ValidationError{
			Field:   "health.timeout",
			Value:   c.Timeout,
			Message: "health check timeout must be positive",
		})
	}

	// Validate thresholds
	if err := c.Thresholds.Validate(); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "health.thresholds",
				Message: err.Error(),
			})
		}
	}

	if c.CustomChecks != "" {
		expandedPath := expandPath(c.CustomChecks)
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   "health.custom_checks",
				Value:   c.CustomChecks,
				Message: "custom checks file does not exist",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates HealthThresholds
func (c HealthThresholds) Validate() error {
	var errors ValidationErrors

	// Validate CPU thresholds
	if err := validateThreshold("health.thresholds.cpu", c.CPU.Warning, c.CPU.Critical); err != nil {
		errors = append(errors, err)
	}

	// Validate memory thresholds
	if err := validateThreshold("health.thresholds.memory", c.Memory.Warning, c.Memory.Critical); err != nil {
		errors = append(errors, err)
	}

	// Validate disk thresholds
	if err := validateThreshold("health.thresholds.disk", c.Disk.Warning, c.Disk.Critical); err != nil {
		errors = append(errors, err)
	}

	// Validate load thresholds
	if err := validateThreshold("health.thresholds.load", c.Load.Warning, c.Load.Critical); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates SecurityConfig
func (c SecurityConfig) Validate() error {
	var errors ValidationErrors

	if c.ScanTimeout <= 0 {
		errors = append(errors, ValidationError{
			Field:   "security.scan_timeout",
			Value:   c.ScanTimeout,
			Message: "scan timeout must be positive",
		})
	}

	if c.MaxConcurrentScans <= 0 {
		errors = append(errors, ValidationError{
			Field:   "security.max_concurrent_scans",
			Value:   c.MaxConcurrentScans,
			Message: "max concurrent scans must be positive",
		})
	}

	if !isValidPortRange(c.DefaultPortRange) {
		errors = append(errors, ValidationError{
			Field:   "security.default_port_range",
			Value:   c.DefaultPortRange,
			Message: "invalid port range format (expected: 'start-end' or single port)",
		})
	}

	if c.HardeningRules != "" {
		expandedPath := expandPath(c.HardeningRules)
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   "security.hardening_rules",
				Value:   c.HardeningRules,
				Message: "hardening rules file does not exist",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates DockerConfig
func (c DockerConfig) Validate() error {
	var errors ValidationErrors

	if c.Timeout <= 0 {
		errors = append(errors, ValidationError{
			Field:   "docker.timeout",
			Value:   c.Timeout,
			Message: "Docker timeout must be positive",
		})
	}

	if strings.TrimSpace(c.SocketPath) == "" {
		errors = append(errors, ValidationError{
			Field:   "docker.socket_path",
			Value:   c.SocketPath,
			Message: "Docker socket path cannot be empty",
		})
	}

	if strings.TrimSpace(c.DefaultRegistry) == "" {
		errors = append(errors, ValidationError{
			Field:   "docker.default_registry",
			Value:   c.DefaultRegistry,
			Message: "default registry cannot be empty",
		})
	}

	if c.CleanupInterval <= 0 {
		errors = append(errors, ValidationError{
			Field:   "docker.cleanup_interval",
			Value:   c.CleanupInterval,
			Message: "cleanup interval must be positive",
		})
	}

	if c.HealthCheckInterval <= 0 {
		errors = append(errors, ValidationError{
			Field:   "docker.health_check_interval",
			Value:   c.HealthCheckInterval,
			Message: "health check interval must be positive",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates MaintenanceConfig
func (c MaintenanceConfig) Validate() error {
	var errors ValidationErrors

	if strings.TrimSpace(c.BackupPath) == "" {
		errors = append(errors, ValidationError{
			Field:   "maintenance.backup_path",
			Value:   c.BackupPath,
			Message: "backup path cannot be empty",
		})
	} else {
		expandedPath := expandPath(c.BackupPath)
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(expandedPath, 0755); err != nil {
				errors = append(errors, ValidationError{
					Field:   "maintenance.backup_path",
					Value:   c.BackupPath,
					Message: fmt.Sprintf("cannot create backup directory: %v", err),
				})
			}
		}
	}

	if c.LogRetention <= 0 {
		errors = append(errors, ValidationError{
			Field:   "maintenance.log_retention",
			Value:   c.LogRetention,
			Message: "log retention must be positive",
		})
	}

	if c.OptimizationRules != "" {
		expandedPath := expandPath(c.OptimizationRules)
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   "maintenance.optimization_rules",
				Value:   c.OptimizationRules,
				Message: "optimization rules file does not exist",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates TUIConfig
func (c TUIConfig) Validate() error {
	var errors ValidationErrors

	validThemes := []string{"default", "dark", "light"}
	if !isValidString(c.Theme, validThemes) {
		errors = append(errors, ValidationError{
			Field:   "tui.theme",
			Value:   c.Theme,
			Message: fmt.Sprintf("invalid theme, must be one of: %s", strings.Join(validThemes, ", ")),
		})
	}

	if c.RefreshInterval <= 0 {
		errors = append(errors, ValidationError{
			Field:   "tui.refresh_interval",
			Value:   c.RefreshInterval,
			Message: "refresh interval must be positive",
		})
	}

	if c.MaxLogLines <= 0 {
		errors = append(errors, ValidationError{
			Field:   "tui.max_log_lines",
			Value:   c.MaxLogLines,
			Message: "max log lines must be positive",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates NotificationsConfig
func (c NotificationsConfig) Validate() error {
	var errors ValidationErrors

	if len(c.Methods) == 0 {
		errors = append(errors, ValidationError{
			Field:   "notifications.methods",
			Value:   c.Methods,
			Message: "at least one notification method must be specified",
		})
	} else {
		validMethods := []string{"tui", "log", "email", "webhook"}
		for _, method := range c.Methods {
			if !isValidString(method, validMethods) {
				errors = append(errors, ValidationError{
					Field:   "notifications.methods",
					Value:   method,
					Message: fmt.Sprintf("invalid notification method, must be one of: %s", strings.Join(validMethods, ", ")),
				})
			}
		}
	}

	// Validate email configuration if enabled
	if contains(c.Methods, "email") && c.Email.Enabled {
		if strings.TrimSpace(c.Email.SMTPServer) == "" {
			errors = append(errors, ValidationError{
				Field:   "notifications.email.smtp_server",
				Value:   c.Email.SMTPServer,
				Message: "SMTP server cannot be empty when email notifications are enabled",
			})
		}

		if c.Email.SMTPPort <= 0 || c.Email.SMTPPort > 65535 {
			errors = append(errors, ValidationError{
				Field:   "notifications.email.smtp_port",
				Value:   c.Email.SMTPPort,
				Message: "SMTP port must be between 1 and 65535",
			})
		}

		if strings.TrimSpace(c.Email.From) == "" {
			errors = append(errors, ValidationError{
				Field:   "notifications.email.from",
				Value:   c.Email.From,
				Message: "from address cannot be empty when email notifications are enabled",
			})
		}

		if len(c.Email.To) == 0 {
			errors = append(errors, ValidationError{
				Field:   "notifications.email.to",
				Value:   c.Email.To,
				Message: "at least one recipient address must be specified when email notifications are enabled",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Validate validates PluginsConfig
func (c PluginsConfig) Validate() error {
	var errors ValidationErrors

	if c.Enabled && c.Directory == "" {
		errors = append(errors, ValidationError{
			Field:   "plugins.directory",
			Value:   c.Directory,
			Message: "plugins directory must be specified when plugins are enabled",
		})
	}

	if c.Directory != "" {
		expandedPath := expandPath(c.Directory)
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   "plugins.directory",
				Value:   c.Directory,
				Message: "plugins directory does not exist",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// Helper functions

func isValidVersion(version string) bool {
	// Simple semantic version validation
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		// Allow numeric versions with optional pre-release
		for i, char := range part {
			if i == 0 && char == '-' {
				continue
			}
			if !((char >= '0' && char <= '9') || char == '-' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')) {
				return false
			}
		}
	}
	
	return true
}

func isValidString(value string, validValues []string) bool {
	for _, valid := range validValues {
		if value == valid {
			return true
		}
	}
	return false
}

func isValidPortRange(portRange string) bool {
	if strings.Contains(portRange, "-") {
		parts := strings.Split(portRange, "-")
		if len(parts) != 2 {
			return false
		}
		// Validate both parts are valid ports
		for _, part := range parts {
			if !isValidPort(part) {
				return false
			}
		}
		// Validate range
		start := parseInt(parts[0])
		end := parseInt(parts[1])
		return start > 0 && end <= 65535 && start <= end
	}
	return isValidPort(portRange)
}

func isValidPort(port string) bool {
	p := parseInt(port)
	return p > 0 && p <= 65535
}

func parseInt(s string) int {
	var result int
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			return -1
		}
	}
	return result
}

func validateThreshold(field string, warning, critical float64) error {
	var errors ValidationErrors

	if warning < 0 || warning > 100 {
		errors = append(errors, ValidationError{
			Field:   field + ".warning",
			Value:   warning,
			Message: "warning threshold must be between 0 and 100",
		})
	}

	if critical < 0 || critical > 100 {
		errors = append(errors, ValidationError{
			Field:   field + ".critical",
			Value:   critical,
			Message: "critical threshold must be between 0 and 100",
		})
	}

	if warning >= critical {
		errors = append(errors, ValidationError{
			Field:   field,
			Value:   fmt.Sprintf("warning: %.1f, critical: %.1f", warning, critical),
			Message: "warning threshold must be less than critical threshold",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}