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

// Migration represents a configuration migration
type Migration struct {
	Version     string
	Description string
	Migrate     func(*Config) error
}

// Migrator handles configuration migrations
type Migrator struct {
	migrations []Migration
	currentVersion string
}

// NewMigrator creates a new configuration migrator
func NewMigrator() *Migrator {
	return &Migrator{
		migrations: []Migration{
			{
				Version:     "1.0.0",
				Description: "Initial configuration format",
				Migrate:     migrateFromV1_0_0,
			},
			{
				Version:     "1.1.0",
				Description: "Add TUI configuration section",
				Migrate:     migrateFromV1_1_0,
			},
			{
				Version:     "1.2.0",
				Description: "Add notifications and plugins configuration",
				Migrate:     migrateFromV1_2_0,
			},
			{
				Version:     "1.3.0",
				Description: "Add Docker and maintenance configuration",
				Migrate:     migrateFromV1_3_0,
			},
		},
	}
}

// Migrate performs configuration migration
func (m *Migrator) Migrate(config *Config, targetVersion string) error {
	// Detect current version
	currentVersion := m.detectVersion(config)
	
	if currentVersion == targetVersion {
		return nil // No migration needed
	}

	// Find migration path
	migrationPath := m.findMigrationPath(currentVersion, targetVersion)
	if migrationPath == nil {
		return fmt.Errorf("no migration path from %s to %s", currentVersion, targetVersion)
	}

	// Apply migrations
	for _, migration := range migrationPath {
		if err := migration.Migrate(config); err != nil {
			return fmt.Errorf("migration to %s failed: %w", migration.Version, err)
		}
	}

	// Set target version
	if config.App.Version == "" {
		config.App.Version = targetVersion
	}

	return nil
}

// detectVersion attempts to detect the configuration version
func (m *Migrator) detectVersion(config *Config) string {
	// Check if version is explicitly set
	if config.App.Version != "" {
		return config.App.Version
	}

	// Try to detect version based on available fields
	if hasTUIFields(config) {
		if hasNotificationFields(config) {
			if hasDockerFields(config) {
				return "1.3.0"
			}
			return "1.2.0"
		}
		return "1.1.0"
	}

	return "1.0.0"
}

// findMigrationPath finds the migration path from current to target version
func (m *Migrator) findMigrationPath(currentVersion, targetVersion string) []Migration {
	var path []Migration
	
	// Build version index
	versionIndex := make(map[string]int)
	for i, migration := range m.migrations {
		versionIndex[migration.Version] = i
	}

	currentIndex := versionIndex[currentVersion]
	targetIndex := versionIndex[targetVersion]

	// If target version is older, return empty path (no downgrade)
	if currentIndex >= targetIndex {
		return nil
	}

	// Build migration path
	for i := currentIndex + 1; i <= targetIndex; i++ {
		path = append(path, m.migrations[i])
	}

	return path
}

// hasTUIFields checks if configuration has TUI fields
func hasTUIFields(config *Config) bool {
	return !reflect.DeepEqual(config.TUI, TUIConfig{})
}

// hasNotificationFields checks if configuration has notification fields
func hasNotificationFields(config *Config) bool {
	return !reflect.DeepEqual(config.Notifications, NotificationsConfig{})
}

// hasDockerFields checks if configuration has Docker fields
func hasDockerFields(config *Config) bool {
	return !reflect.DeepEqual(config.Docker, DockerConfig{})
}

// Migration functions

func migrateFromV1_0_0(config *Config) error {
	// Add default TUI configuration
	config.TUI = TUIConfig{
		Theme:           "default",
		RefreshInterval: 5 * time.Second,
		MaxLogLines:     1000,
		ConfirmDestructive: true,
		ShowNotifications: true,
	}

	return nil
}

func migrateFromV1_1_0(config *Config) error {
	// Add notifications configuration
	config.Notifications = NotificationsConfig{
		Enabled: true,
		Methods: []string{"tui", "log"},
		Email: EmailConfig{
			Enabled:   false,
			SMTPPort: 587,
		},
	}

	// Add plugins configuration
	config.Plugins = PluginsConfig{
		Enabled: false,
		Load:    []string{},
	}

	return nil
}

func migrateFromV1_2_0(config *Config) error {
	// Add Docker configuration
	config.Docker = DockerConfig{
		SocketPath:         "/var/run/docker.sock",
		Timeout:            30 * time.Second,
		DefaultRegistry:    "docker.io",
		CleanupInterval:     24 * time.Hour,
		HealthCheckInterval: 30 * time.Second,
	}

	// Add maintenance configuration
	config.Maintenance = MaintenanceConfig{
		BackupPath:    "~/.local/share/vps-tools/backups",
		LogRetention:  "30d",
		AutoCleanup:   true,
		CleanupSchedule: "0 2 * * 0", // Weekly
	}

	return nil
}

func migrateFromV1_3_0(config *Config) error {
	// This is the latest version, no migration needed
	return nil
}

// BackupConfiguration creates a backup of the current configuration
func BackupConfiguration(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", configPath)
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.backup.%s", configPath, timestamp)

	// Copy file
	source, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	if err := os.WriteFile(backupPath, source, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return nil
}

// RestoreConfiguration restores configuration from backup
func RestoreConfiguration(backupPath, configPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Create backup of current config before restore
	if _, err := os.Stat(configPath); err == nil {
		if err := BackupConfiguration(configPath); err != nil {
			return fmt.Errorf("failed to backup current configuration: %w", err)
		}
	}

	// Copy backup to config location
	source, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(configPath, source, 0644); err != nil {
		return fmt.Errorf("failed to restore configuration: %w", err)
	}

	return nil
}

// ListBackups lists available configuration backups
func ListBackups(configPath string) ([]string, error) {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration directory: %w", err)
	}

	var backups []string
	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, base+".backup.") {
			backups = append(backups, filepath.Join(dir, name))
		}
	}

	return backups, nil
}

// CleanupOldBackups removes old configuration backups
func CleanupOldBackups(configPath string, keepCount int) error {
	backups, err := ListBackups(configPath)
	if err != nil {
		return err
	}

	if len(backups) <= keepCount {
		return nil // No cleanup needed
	}

	// Sort backups by modification time (newest first)
	type backupInfo struct {
		path    string
		modTime time.Time
	}

	var backupInfos []backupInfo
	for _, backup := range backups {
		info, err := os.Stat(backup)
		if err != nil {
			continue // Skip files that can't be stat'd
		}
		backupInfos = append(backupInfos, backupInfo{
			path:    backup,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(backupInfos)-1; i++ {
		for j := i + 1; j < len(backupInfos); j++ {
			if backupInfos[i].modTime.Before(backupInfos[j].modTime) {
				backupInfos[i], backupInfos[j] = backupInfos[j], backupInfos[i]
			}
		}
	}

	// Remove oldest backups beyond keepCount
	for i := keepCount; i < len(backupInfos); i++ {
		if err := os.Remove(backupInfos[i].path); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", backupInfos[i].path, err)
		}
	}

	return nil
}

// ValidateAndMigrate validates and migrates configuration if needed
func ValidateAndMigrate(configPath string) (*Config, error) {
	// Load configuration
	config, err := LoadFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create migrator
	migrator := NewMigrator()

	// Detect current version
	currentVersion := migrator.detectVersion(config)
	targetVersion := "1.3.0" // Latest version

	// Check if migration is needed
	if currentVersion != targetVersion {
		// Create backup before migration
		if err := BackupConfiguration(configPath); err != nil {
			return nil, fmt.Errorf("failed to create backup before migration: %w", err)
		}

		// Perform migration
		if err := migrator.Migrate(config, targetVersion); err != nil {
			return nil, fmt.Errorf("configuration migration failed: %w", err)
		}

		// Save migrated configuration
		if err := SaveToFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to save migrated configuration: %w", err)
		}
	}

	// Validate migrated configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// CheckForUpdates checks if there are available configuration updates
func CheckForUpdates(config *Config) (bool, string, error) {
	// This is a placeholder for future update checking functionality
	// In a real implementation, this might check for new configuration
	// schemas or default values from a remote source
	
	currentVersion := config.App.Version
	if currentVersion == "" {
		currentVersion = "1.0.0"
	}

	latestVersion := "1.3.0" // This would be determined dynamically
	
	if currentVersion != latestVersion {
		return true, latestVersion, nil
	}

	return false, "", nil
}

// GetMigrationSummary returns a summary of available migrations
func GetMigrationSummary() string {
	migrator := NewMigrator()
	
	var summary strings.Builder
	summary.WriteString("Available Configuration Migrations:\n\n")
	
	for _, migration := range migrator.migrations {
		summary.WriteString(fmt.Sprintf("Version %s: %s\n", migration.Version, migration.Description))
	}
	
	summary.WriteString("\nUse 'vps-tools config migrate' to apply migrations.\n")
	
	return summary.String()
}