package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/server"
	"github.com/pgd1001/vps-tools/internal/store"
)

// Migrator handles migration from bash scripts configuration to vps-tools format
type Migrator struct {
	configManager *config.ConfigManager
	store        store.Store
	logger       Logger
}

// Logger interface for migration logging
type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

// NewMigrator creates a new migrator
func NewMigrator(configManager *config.ConfigManager, store store.Store, logger Logger) *Migrator {
	return &Migrator{
		configManager: configManager,
		store:        store,
		logger:       logger,
	}
}

// BashScriptConfig represents configuration from bash scripts
type BashScriptConfig struct {
	Servers []BashServer `json:"servers"`
	Cron    BashCron     `json:"cron"`
	Env     BashEnv      `json:"env"`
}

// BashServer represents server configuration from bash scripts
type BashServer struct {
	ID       string            `json:"id"`
	Hostname string            `json:"hostname"`
	IP       string            `json:"ip"`
	Port     int               `json:"port"`
	User     string            `json:"user"`
	SSHKey   string            `json:"ssh_key,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// BashCron represents cron configuration from bash scripts
type BashCron struct {
	MailTo     string            `json:"mail_to"`
	Schedules  map[string]string `json:"schedules"`
	AlertEmail string            `json:"alert_email"`
}

// BashEnv represents environment variables from bash scripts
type BashEnv struct {
	LogDir      string `json:"log_dir"`
	BackupDir   string `json:"backup_dir"`
	ConfigDir   string `json:"config_dir"`
	DataDir     string `json:"data_dir"`
}

// MigrationResult represents the result of a migration operation
type MigrationResult struct {
	Success        bool     `json:"success"`
	ServersMigrated int      `json:"servers_migrated"`
	Errors         []string `json:"errors,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	Duration       string   `json:"duration"`
	Timestamp      string   `json:"timestamp"`
}

// MigrateFromBashScripts migrates configuration from bash scripts
func (m *Migrator) MigrateFromBashScripts(bashScriptsDir string) (*MigrationResult, error) {
	startTime := time.Now()
	result := &MigrationResult{
		Timestamp: startTime.Format("2006-01-02T15:04:05.000Z07:00"),
		Success:   true,
	}

	m.logger.Info("Starting migration from bash scripts...")
	m.logger.Infof("Source directory: %s", bashScriptsDir)

	// Step 1: Parse existing bash script configurations
	bashConfig, err := m.parseBashScripts(bashScriptsDir)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse bash scripts: %v", err))
		return result, err
	}

	// Step 2: Convert to vps-tools configuration
	vpsConfig, err := m.convertToVPSConfig(bashConfig)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to convert configuration: %v", err))
		return result, err
	}

	// Step 3: Save new configuration
	if err := m.configManager.Save(); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to save configuration: %v", err))
		return result, err
	}

	// Step 4: Migrate servers to database
	for _, bashServer := range bashConfig.Servers {
		if err := m.migrateServer(bashServer); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to migrate server %s: %v", bashServer.ID, err))
			result.Success = false
		} else {
			result.ServersMigrated++
		}
	}

	// Step 5: Create backup of old configuration
	if err := m.createBackup(bashScriptsDir); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to create backup: %v", err))
	}

	result.Duration = time.Since(startTime).String()

	if result.Success {
		m.logger.Infof("Migration completed successfully in %s", result.Duration)
		m.logger.Infof("Migrated %d servers", result.ServersMigrated)
	} else {
		m.logger.Errorf("Migration failed after %s", result.Duration)
		for _, err := range result.Errors {
			m.logger.Errorf("Error: %s", err)
		}
	}

	for _, warning := range result.Warnings {
		m.logger.Warnf("Warning: %s", warning)
	}

	return result, nil
}

// parseBashScripts parses configuration from bash scripts directory
func (m *Migrator) parseBashScripts(bashScriptsDir string) (*BashScriptConfig, error) {
	config := &BashScriptConfig{
		Servers: []BashServer{},
		Cron: BashCron{
			Schedules: make(map[string]string),
		},
		Env: BashEnv{},
	}

	// Parse vps-build.sh for server information
	if err := m.parseVPSBuild(bashScriptsDir, config); err != nil {
		return nil, fmt.Errorf("failed to parse vps-build.sh: %w", err)
	}

	// Parse cron configuration
	if err := m.parseCronConfig(bashScriptsDir, config); err != nil {
		return nil, fmt.Errorf("failed to parse cron config: %w", err)
	}

	// Parse environment variables
	if err := m.parseEnvironment(bashScriptsDir, config); err != nil {
		return nil, fmt.Errorf("failed to parse environment: %w", err)
	}

	return config, nil
}

// parseVPSBuild parses vps-build.sh for server configuration
func (m *Migrator) parseVPSBuild(bashScriptsDir string, config *BashScriptConfig) error {
	vpsBuildPath := filepath.Join(bashScriptsDir, "vps-build.sh")
	if _, err := os.Stat(vpsBuildPath); os.IsNotExist(err) {
		// vps-build.sh not found, create a default server
		config.Servers = append(config.Servers, BashServer{
			ID:       "localhost",
			Hostname: "localhost",
			IP:       "127.0.0.1",
			Port:     22,
			User:     "root",
			Tags:     []string{"local", "migrated"},
		})
		return nil
	}

	content, err := os.ReadFile(vpsBuildPath)
	if err != nil {
		return err
	}

	// Parse for default values
	contentStr := string(content)
	
	// Extract default user
	defaultUser := "root"
	if strings.Contains(contentStr, "ubuntu") {
		defaultUser = "ubuntu"
	}

	// Extract default port
	defaultPort := 22
	if strings.Contains(contentStr, "Port:") {
		// This is a simplified parser - in production, you'd want more robust parsing
		if strings.Contains(contentStr, "2222") {
			defaultPort = 2222
		}
	}

	// Create a default server based on vps-build.sh
	config.Servers = append(config.Servers, BashServer{
		ID:       "default-server",
		Hostname: "Default Server",
		IP:       "127.0.0.1",
		Port:     defaultPort,
		User:     defaultUser,
		Tags:     []string{"default", "migrated"},
		Meta: map[string]string{
			"source": "vps-build.sh",
			"migrated_at": time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
		},
	})

	return nil
}

// parseCronConfig parses vps-tools-cron.conf
func (m *Migrator) parseCronConfig(bashScriptsDir string, config *BashScriptConfig) error {
	cronPath := filepath.Join(bashScriptsDir, "vps-tools-cron.conf")
	if _, err := os.Stat(cronPath); os.IsNotExist(err) {
		return nil // Not found, skip
	}

	content, err := os.ReadFile(cronPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "MAILTO=") {
			config.Cron.MailTo = strings.TrimPrefix(line, "MAILTO=")
		} else if strings.HasPrefix(line, "#") {
			// Comment line, might contain schedule info
			if strings.Contains(line, "Every 5 minutes") {
				config.Cron.Schedules["health_monitor"] = "*/5 * * * *"
			} else if strings.Contains(line, "Daily at 2 AM") {
				config.Cron.Schedules["log_analysis"] = "0 2 * * *"
			}
		}
	}

	return nil
}

// parseEnvironment parses environment variables from bash scripts
func (m *Migrator) parseEnvironment(bashScriptsDir string, config *BashScriptConfig) error {
	// Set default environment paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/home/user" // fallback
	}

	config.Env.LogDir = filepath.Join(homeDir, ".local", "share", "vps-tools", "logs")
	config.Env.BackupDir = filepath.Join(homeDir, ".local", "share", "vps-tools", "backups")
	config.Env.ConfigDir = filepath.Join(homeDir, ".config", "vps-tools")
	config.Env.DataDir = filepath.Join(homeDir, ".local", "share", "vps-tools", "data")

	return nil
}

// convertToVPSConfig converts bash configuration to vps-tools format
func (m *Migrator) convertToVPSConfig(bashConfig *BashScriptConfig) (*config.Config, error) {
	vpsConfig := &config.Config{
		App: config.AppConfig{
			Name:    "vps-tools",
			Version: "1.0.0",
			LogLevel: "info",
		},
		Servers: []config.ServerConfig{},
		Monitoring: config.MonitoringConfig{
			Enabled:  true,
			Interval: "5m",
			Thresholds: config.ThresholdsConfig{
				DiskWarning:    80,
				DiskCritical:   90,
				MemoryWarning:  75,
				MemoryCritical: 85,
				CPUWarning:     80,
				CPUCritical:    95,
			},
			Checks: []string{"disk", "memory", "cpu", "services"},
		},
		Security: config.SecurityConfig{
			SSHAuditEnabled:      true,
			FailedLoginThreshold: 10,
			SSLWarningDays:      30,
			PortScanEnabled:     true,
			AutoBlockIPs:        false,
			KeyRotationDays:     90,
		},
		Docker: config.DockerConfig{
			Enabled: true,
			Socket:  "/var/run/docker.sock",
			LogRotation: config.LogRotationConfig{
				MaxSize:  "100m",
				MaxFiles: 5,
				Driver:   "json-file",
			},
		},
		Storage: config.StorageConfig{
			Type:       "bolt",
			BoltDBPath: filepath.Join(bashConfig.Env.DataDir, "vps-tools.db"),
		},
		SSH: config.SSHConfig{
			DefaultUser:     "root",
			DefaultPort:    22,
			Timeout:        30,
			MaxRetries:     3,
			RetryDelay:     5,
			KeepAlive:      30,
			StrictHostKey:  true,
		},
	}

	// Set alert email if found
	if bashConfig.Cron.MailTo != "" && bashConfig.Cron.MailTo != "admin@example.com" {
		vpsConfig.Alerts = config.AlertsConfig{
			Enabled: true,
			Email:   bashConfig.Cron.MailTo,
		}
	}

	// Convert servers
	for _, bashServer := range bashConfig.Servers {
		vpsServer := config.ServerConfig{
			ID:   bashServer.ID,
			Name: bashServer.Hostname,
			Host: bashServer.IP,
			Port: bashServer.Port,
			User: bashServer.User,
			Tags: bashServer.Tags,
			Meta: bashServer.Meta,
		}

		// Convert SSH configuration
		if bashServer.SSHKey != "" {
			vpsServer.AuthMethod = map[string]interface{}{
				"type":       "private_key",
				"key_path":   bashServer.SSHKey,
				"use_agent":  false,
			}
		} else {
			vpsServer.AuthMethod = map[string]interface{}{
				"type":      "ssh_agent",
				"use_agent": true,
			}
		}

		vpsConfig.Servers = append(vpsConfig.Servers, vpsServer)
	}

	return vpsConfig, nil
}

// migrateServer migrates a single server to the database
func (m *Migrator) migrateServer(bashServer BashServer) error {
	// Convert to server.Server
	srv := &server.Server{
		ID:   bashServer.ID,
		Name: bashServer.Hostname,
		Host: bashServer.IP,
		Port: bashServer.Port,
		User: bashServer.User,
		Tags: bashServer.Tags,
		Meta: bashServer.Meta,
		Status: server.StatusUnknown,
	}

	// Convert auth method
	if bashServer.SSHKey != "" {
		srv.AuthMethod = server.AuthConfig{
			Type:    "private_key",
			KeyPath: bashServer.SSHKey,
		}
	} else {
		srv.AuthMethod = server.AuthConfig{
			Type:    "ssh_agent",
			UseAgent: true,
		}
	}

	// Validate server
	if err := srv.Validate(); err != nil {
		return fmt.Errorf("invalid server configuration: %w", err)
	}

	// Save to database
	return m.store.CreateServer(srv)
}

// createBackup creates a backup of the bash scripts directory
func (m *Migrator) createBackup(bashScriptsDir string) error {
	backupDir := bashScriptsDir + "_backup_" + time.Now().Format("20060102_150405")
	
	// Copy directory
	return copyDir(bashScriptsDir, backupDir)
}

// copyDir copies a directory recursively
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		return copyFile(path, dstPath)
	})
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = destination.ReadFrom(source)
	return err
}

// ExportConfiguration exports current configuration to various formats
func (m *Migrator) ExportConfiguration(format string, outputPath string) error {
	cfg := m.configManager.Get()

	switch strings.ToLower(format) {
	case "yaml":
		return m.exportYAML(cfg, outputPath)
	case "json":
		return m.exportJSON(cfg, outputPath)
	case "env":
		return m.exportEnv(cfg, outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportYAML exports configuration to YAML format
func (m *Migrator) exportYAML(cfg *config.Config, outputPath string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// exportJSON exports configuration to JSON format
func (m *Migrator) exportJSON(cfg *config.Config, outputPath string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// exportEnv exports configuration to environment variables format
func (m *Migrator) exportEnv(cfg *config.Config, outputPath string) error {
	var envVars []string

	// App configuration
	envVars = append(envVars, fmt.Sprintf("VPS_TOOLS_NAME=%s", cfg.App.Name))
	envVars = append(envVars, fmt.Sprintf("VPS_TOOLS_LOG_LEVEL=%s", cfg.App.LogLevel))
	envVars = append(envVars, fmt.Sprintf("VPS_TOOLS_CONFIG_DIR=%s", cfg.App.ConfigDir))

	// Storage configuration
	envVars = append(envVars, fmt.Sprintf("VPS_TOOLS_DB_TYPE=%s", cfg.Storage.Type))
	envVars = append(envVars, fmt.Sprintf("VPS_TOOLS_DB_PATH=%s", cfg.Storage.BoltDBPath))

	// Alerts configuration
	if cfg.Alerts.Email != "" {
		envVars = append(envVars, fmt.Sprintf("VPS_TOOLS_ALERT_EMAIL=%s", cfg.Alerts.Email))
	}

	// Write to file
	content := strings.Join(envVars, "\n")
	return os.WriteFile(outputPath, []byte(content), 0644)
}

// ValidateMigration checks if migration was successful
func (m *Migrator) ValidateMigration() error {
	// Check if configuration exists
	cfg := m.configManager.Get()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Check if servers were migrated
	servers, err := m.store.ListServers(nil)
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no servers migrated")
	}

	// Validate each server
	for _, srv := range servers {
		if err := srv.Validate(); err != nil {
			return fmt.Errorf("migrated server %s is invalid: %w", srv.ID, err)
		}
	}

	m.logger.Infof("Migration validation successful: %d servers migrated", len(servers))
	return nil
}