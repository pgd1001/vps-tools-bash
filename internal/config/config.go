package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	App        AppConfig        `yaml:"app"`
	Servers    []ServerConfig   `yaml:"servers"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Security   SecurityConfig   `yaml:"security"`
	Docker     DockerConfig     `yaml:"docker"`
	Alerts     AlertsConfig     `yaml:"alerts"`
	Storage    StorageConfig    `yaml:"storage"`
	SSH        SSHConfig        `yaml:"ssh"`
}

// AppConfig represents application-wide configuration
type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	LogLevel    string `yaml:"log_level"`
	LogFormat   string `yaml:"log_format"` // json, text
	ConfigDir   string `yaml:"config_dir"`
	DataDir     string `yaml:"data_dir"`
	MaxWorkers  int    `yaml:"max_workers"`
	Timeout     int    `yaml:"timeout"` // default timeout in seconds
}

// ServerConfig represents server configuration from YAML file
type ServerConfig struct {
	ID         string                 `yaml:"id"`
	Name       string                 `yaml:"name"`
	Host       string                 `yaml:"host"`
	Port       int                    `yaml:"port"`
	User       string                 `yaml:"user"`
	AuthMethod map[string]interface{} `yaml:"auth_method"`
	Tags       []string               `yaml:"tags"`
	Meta       map[string]string      `yaml:"meta"`
	Disabled   bool                   `yaml:"disabled"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	Enabled    bool              `yaml:"enabled"`
	Interval   string            `yaml:"interval"`   // check interval (e.g., "5m", "30s")
	Thresholds ThresholdsConfig  `yaml:"thresholds"`
	Checks     []string          `yaml:"checks"`     // which checks to run
	Retention  RetentionConfig   `yaml:"retention"`
}

// ThresholdsConfig represents monitoring thresholds
type ThresholdsConfig struct {
	DiskWarning    int `yaml:"disk_warning"`    // percentage
	DiskCritical   int `yaml:"disk_critical"`   // percentage
	MemoryWarning  int `yaml:"memory_warning"`  // percentage
	MemoryCritical int `yaml:"memory_critical"` // percentage
	CPUWarning     int `yaml:"cpu_warning"`     // percentage
	CPUCritical    int `yaml:"cpu_critical"`    // percentage
	LoadWarning    int `yaml:"load_warning"`    // load average
	LoadCritical   int `yaml:"load_critical"`   // load average
}

// RetentionConfig represents data retention settings
type RetentionConfig struct {
	HealthChecks string `yaml:"health_checks"` // e.g., "7d", "30d"
	Jobs        string `yaml:"jobs"`         // e.g., "30d", "90d"
	AuditLogs   string `yaml:"audit_logs"`   // e.g., "90d", "1y"
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	SSHAuditEnabled         bool `yaml:"ssh_audit_enabled"`
	FailedLoginThreshold    int  `yaml:"failed_login_threshold"`
	SSLWarningDays         int  `yaml:"ssl_warning_days"`
	PortScanEnabled        bool `yaml:"port_scan_enabled"`
	AutoBlockIPs           bool `yaml:"auto_block_ips"`
	KeyRotationDays        int  `yaml:"key_rotation_days"`
	RequireKnownHosts      bool `yaml:"require_known_hosts"`
}

// DockerConfig represents Docker-specific configuration
type DockerConfig struct {
	Enabled       bool                `yaml:"enabled"`
	Socket        string              `yaml:"socket"`        // Docker socket path
	APIVersion    string              `yaml:"api_version"`   // Docker API version
	Timeout       int                 `yaml:"timeout"`       // timeout in seconds
	LogRotation   LogRotationConfig    `yaml:"log_rotation"`
	Cleanup       CleanupConfig        `yaml:"cleanup"`
	Backup        BackupConfig         `yaml:"backup"`
}

// LogRotationConfig represents Docker log rotation settings
type LogRotationConfig struct {
	MaxSize  string `yaml:"max_size"`  // e.g., "100m", "1g"
	MaxFiles int    `yaml:"max_files"`
	Driver   string `yaml:"driver"`    // json-file, local, etc.
}

// CleanupConfig represents Docker cleanup settings
type CleanupConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Schedule    string `yaml:"schedule"`    // cron expression
	Aggressive  bool   `yaml:"aggressive"`  // remove stopped containers
	PruneImages bool   `yaml:"prune_images"`
	MaxAge      string `yaml:"max_age"`     // e.g., "7d", "30d"
}

// BackupConfig represents backup configuration
type BackupConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Schedule   string `yaml:"schedule"`   // cron expression
	BackupDir  string `yaml:"backup_dir"`
	Compression bool  `yaml:"compression"`
	Retention  string `yaml:"retention"`  // e.g., "30d", "90d"
}

// AlertsConfig represents alerting configuration
type AlertsConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Email       string   `yaml:"email"`
	WebhookURL  string   `yaml:"webhook_url"`
	Slack       SlackConfig `yaml:"slack"`
	Webhook     WebhookConfig `yaml:"webhook"`
	Thresholds  AlertThresholds `yaml:"thresholds"`
}

// SlackConfig represents Slack integration
type SlackConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Webhook  string `yaml:"webhook"`
	Channel  string `yaml:"channel"`
	Username string `yaml:"username"`
}

// WebhookConfig represents generic webhook configuration
type WebhookConfig struct {
	Enabled  bool              `yaml:"enabled"`
	URL      string            `yaml:"url"`
	Method   string            `yaml:"method"`    // POST, PUT
	Headers  map[string]string `yaml:"headers"`
	Timeout  int               `yaml:"timeout"`   // timeout in seconds
	Retries  int               `yaml:"retries"`
}

// AlertThresholds represents alert thresholds
type AlertThresholds struct {
	FailureCount    int `yaml:"failure_count"`    // consecutive failures before alert
	DiskUsage       int `yaml:"disk_usage"`       // percentage
	MemoryUsage     int `yaml:"memory_usage"`     // percentage
	CPUUsage        int `yaml:"cpu_usage"`        // percentage
	ResponseTime    int  `yaml:"response_time"`    // milliseconds
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type         string `yaml:"type"`          // bolt, sqlite, postgres
	BoltDBPath   string `yaml:"bolt_db_path"`
	SQLitePath   string `yaml:"sqlite_path"`
	Postgres     PostgresConfig `yaml:"postgres"`
	BackupEnabled bool   `yaml:"backup_enabled"`
	BackupPath   string `yaml:"backup_path"`
	BackupSchedule string `yaml:"backup_schedule"`
}

// PostgresConfig represents PostgreSQL configuration
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
}

// SSHConfig represents SSH client configuration
type SSHConfig struct {
	DefaultUser     string `yaml:"default_user"`
	DefaultPort    int    `yaml:"default_port"`
	Timeout        int    `yaml:"timeout"`         // connection timeout in seconds
	MaxRetries     int    `yaml:"max_retries"`
	RetryDelay     int    `yaml:"retry_delay"`     // delay between retries in seconds
	KeepAlive      int    `yaml:"keep_alive"`      // keep alive interval in seconds
	StrictHostKey  bool   `yaml:"strict_host_key"`
	KnownHostsFile string `yaml:"known_hosts_file"`
	PrivateKeyPath string `yaml:"private_key_path"`
}

// ConfigManager manages configuration loading and validation
type ConfigManager struct {
	configPath string
	config     *Config
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// Load loads configuration from file
func (cm *ConfigManager) Load() error {
	// Set default config path if not provided
	if cm.configPath == "" {
		cm.configPath = getDefaultConfigPath()
	}

	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Create default config if it doesn't exist
		if err := cm.createDefaultConfig(); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
	}

	// Read config file
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	cm.applyDefaults(config)

	// Validate configuration
	if err := cm.validate(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cm.config = config
	return nil
}

// Get returns the loaded configuration
func (cm *ConfigManager) Get() *Config {
	return cm.config
}

// Save saves the current configuration to file
func (cm *ConfigManager) Save() error {
	if cm.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	// Ensure directory exists
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getDefaultConfigPath returns the default configuration file path
func getDefaultConfigPath() string {
	// Check environment variable first
	if configPath := os.Getenv("VPS_TOOLS_CONFIG"); configPath != "" {
		return configPath
	}

	// Check user config directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".config", "vps-tools", "config.yaml")
	}

	// Fallback to current directory
	return "config.yaml"
}

// createDefaultConfig creates a default configuration file
func (cm *ConfigManager) createDefaultConfig() error {
	config := &Config{}
	cm.applyDefaults(config)

	// Ensure directory exists
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	return nil
}

// applyDefaults applies default values to configuration
func (cm *ConfigManager) applyDefaults(config *Config) {
	// App defaults
	if config.App.Name == "" {
		config.App.Name = "vps-tools"
	}
	if config.App.Version == "" {
		config.App.Version = "1.0.0"
	}
	if config.App.LogLevel == "" {
		config.App.LogLevel = "info"
	}
	if config.App.LogFormat == "" {
		config.App.LogFormat = "text"
	}
	if config.App.MaxWorkers == 0 {
		config.App.MaxWorkers = 10
	}
	if config.App.Timeout == 0 {
		config.App.Timeout = 300
	}

	// Monitoring defaults
	if config.Monitoring.Interval == "" {
		config.Monitoring.Interval = "5m"
	}
	if config.Monitoring.Checks == nil {
		config.Monitoring.Checks = []string{"disk", "memory", "cpu", "services"}
	}

	// Thresholds defaults
	if config.Monitoring.Thresholds.DiskWarning == 0 {
		config.Monitoring.Thresholds.DiskWarning = 80
	}
	if config.Monitoring.Thresholds.DiskCritical == 0 {
		config.Monitoring.Thresholds.DiskCritical = 90
	}
	if config.Monitoring.Thresholds.MemoryWarning == 0 {
		config.Monitoring.Thresholds.MemoryWarning = 75
	}
	if config.Monitoring.Thresholds.MemoryCritical == 0 {
		config.Monitoring.Thresholds.MemoryCritical = 85
	}
	if config.Monitoring.Thresholds.CPUWarning == 0 {
		config.Monitoring.Thresholds.CPUWarning = 80
	}
	if config.Monitoring.Thresholds.CPUCritical == 0 {
		config.Monitoring.Thresholds.CPUCritical = 95
	}

	// Security defaults
	if config.Security.FailedLoginThreshold == 0 {
		config.Security.FailedLoginThreshold = 10
	}
	if config.Security.SSLWarningDays == 0 {
		config.Security.SSLWarningDays = 30
	}
	if config.Security.KeyRotationDays == 0 {
		config.Security.KeyRotationDays = 90
	}

	// Docker defaults
	if config.Docker.Socket == "" {
		config.Docker.Socket = "/var/run/docker.sock"
	}
	if config.Docker.APIVersion == "" {
		config.Docker.APIVersion = "1.41"
	}
	if config.Docker.Timeout == 0 {
		config.Docker.Timeout = 30
	}
	if config.Docker.LogRotation.MaxSize == "" {
		config.Docker.LogRotation.MaxSize = "100m"
	}
	if config.Docker.LogRotation.MaxFiles == 0 {
		config.Docker.LogRotation.MaxFiles = 5
	}
	if config.Docker.LogRotation.Driver == "" {
		config.Docker.LogRotation.Driver = "json-file"
	}

	// Storage defaults
	if config.Storage.Type == "" {
		config.Storage.Type = "bolt"
	}
	if config.Storage.BoltDBPath == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			config.Storage.BoltDBPath = filepath.Join(homeDir, ".config", "vps-tools", "vps-tools.db")
		} else {
			config.Storage.BoltDBPath = "vps-tools.db"
		}
	}

	// SSH defaults
	if config.SSH.DefaultUser == "" {
		config.SSH.DefaultUser = "root"
	}
	if config.SSH.DefaultPort == 0 {
		config.SSH.DefaultPort = 22
	}
	if config.SSH.Timeout == 0 {
		config.SSH.Timeout = 30
	}
	if config.SSH.MaxRetries == 0 {
		config.SSH.MaxRetries = 3
	}
	if config.SSH.RetryDelay == 0 {
		config.SSH.RetryDelay = 5
	}
	if config.SSH.KeepAlive == 0 {
		config.SSH.KeepAlive = 30
	}
	if config.SSH.KnownHostsFile == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			config.SSH.KnownHostsFile = filepath.Join(homeDir, ".ssh", "known_hosts")
		}
	}
}

// validate validates the configuration
func (cm *ConfigManager) validate(config *Config) error {
	// Validate app configuration
	if config.App.Name == "" {
		return fmt.Errorf("app name is required")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, config.App.LogLevel) {
		return fmt.Errorf("invalid log level: %s (valid: %s)", config.App.LogLevel, strings.Join(validLogLevels, ", "))
	}

	// Validate log format
	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, config.App.LogFormat) {
		return fmt.Errorf("invalid log format: %s (valid: %s)", config.App.LogFormat, strings.Join(validLogFormats, ", "))
	}

	// Validate storage type
	validStorageTypes := []string{"bolt", "sqlite", "postgres"}
	if !contains(validStorageTypes, config.Storage.Type) {
		return fmt.Errorf("invalid storage type: %s (valid: %s)", config.Storage.Type, strings.Join(validStorageTypes, ", "))
	}

	// Validate thresholds
	if config.Monitoring.Thresholds.DiskWarning <= 0 || config.Monitoring.Thresholds.DiskWarning > 100 {
		return fmt.Errorf("disk warning threshold must be between 1 and 100")
	}
	if config.Monitoring.Thresholds.DiskCritical <= 0 || config.Monitoring.Thresholds.DiskCritical > 100 {
		return fmt.Errorf("disk critical threshold must be between 1 and 100")
	}
	if config.Monitoring.Thresholds.DiskWarning >= config.Monitoring.Thresholds.DiskCritical {
		return fmt.Errorf("disk warning threshold must be less than critical threshold")
	}

	// Validate server configurations
	for i, server := range config.Servers {
		if server.ID == "" {
			return fmt.Errorf("server %d: ID is required", i)
		}
		if server.Name == "" {
			return fmt.Errorf("server %s: name is required", server.ID)
		}
		if server.Host == "" {
			return fmt.Errorf("server %s: host is required", server.ID)
		}
		if server.Port <= 0 || server.Port > 65535 {
			return fmt.Errorf("server %s: invalid port: %d", server.ID, server.Port)
		}
		if server.User == "" {
			return fmt.Errorf("server %s: user is required", server.ID)
		}
		if server.AuthMethod == nil {
			return fmt.Errorf("server %s: auth_method is required", server.ID)
		}
	}

	return nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}