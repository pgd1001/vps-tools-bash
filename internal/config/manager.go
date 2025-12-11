package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigWatcher watches for configuration file changes
type ConfigWatcher struct {
	configPath    string
	lastModTime   time.Time
	callback      func(*Config) error
	stopChan      chan bool
}

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(configPath string, callback func(*Config) error) *ConfigWatcher {
	return &ConfigWatcher{
		configPath: configPath,
		callback:   callback,
		stopChan:   make(chan bool),
	}
}

// Start starts watching for configuration changes
func (w *ConfigWatcher) Start() error {
	// Get initial modification time
	info, err := os.Stat(w.configPath)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}
	w.lastModTime = info.ModTime()

	// Start watching goroutine
	go w.watch()

	return nil
}

// Stop stops the configuration watcher
func (w *ConfigWatcher) Stop() {
	close(w.stopChan)
}

// watch watches for file changes
func (w *ConfigWatcher) watch() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			info, err := os.Stat(w.configPath)
			if err != nil {
				continue
			}

			if info.ModTime().After(w.lastModTime) {
				w.lastModTime = info.ModTime()
				
				// Reload configuration
				config, err := LoadFromFile(w.configPath)
				if err != nil {
					continue
				}

				// Call callback
				if w.callback != nil {
					w.callback(config)
				}
			}
		}
	}
}

// ConfigManager manages configuration with validation, migration, and watching
type ConfigManager struct {
	configPath string
	config     *Config
	watcher    *ConfigWatcher
	migrator   *Migrator
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
		migrator:  NewMigrator(),
	}
}

// Load loads and validates configuration
func (cm *ConfigManager) Load() error {
	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Create default configuration
		cm.config = DefaultConfig()
		if err := cm.Save(); err != nil {
			return fmt.Errorf("failed to create default configuration: %w", err)
		}
		return nil
	}

	// Load and migrate configuration
	config, err := ValidateAndMigrate(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to load and migrate configuration: %w", err)
	}

	cm.config = config
	return nil
}

// Save saves the current configuration
func (cm *ConfigManager) Save() error {
	if cm.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	// Validate before saving
	if err := cm.config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Save to file
	return SaveToFile(cm.config, cm.configPath)
}

// Get returns the current configuration
func (cm *ConfigManager) Get() *Config {
	return cm.config
}

// Set updates the configuration
func (cm *ConfigManager) Set(config *Config) error {
	cm.config = config
	return cm.Save()
}

// Update updates specific configuration values
func (cm *ConfigManager) Update(updates map[string]interface{}) error {
	if cm.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Apply updates
	for key, value := range updates {
		if err := cm.setNestedValue(key, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	// Save updated configuration
	return cm.Save()
}

// setNestedValue sets a nested configuration value using dot notation
func (cm *ConfigManager) setNestedValue(key string, value interface{}) error {
	parts := strings.Split(key, ".")
	
	// Use reflection to set nested value
	configValue := reflect.ValueOf(cm.config).Elem()
	
	for i, part := range parts[:len(parts)-1] {
		field := configValue.FieldByName(strings.Title(part))
		if !field.IsValid() {
			return fmt.Errorf("invalid configuration path: %s", key)
		}
		
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				// Initialize pointer
				field.Set(reflect.New(field.Type().Elem()))
			}
			configValue = field.Elem()
		} else {
			configValue = field
		}
		
		if i == len(parts)-2 {
			// This is the parent of the final field
			finalField := configValue.FieldByName(strings.Title(parts[len(parts)-1]))
			if !finalField.IsValid() {
				return fmt.Errorf("invalid configuration path: %s", key)
			}
			
			// Convert value to appropriate type
			convertedValue := reflect.ValueOf(value)
			if !convertedValue.Type().ConvertibleTo(finalField.Type()) {
				return fmt.Errorf("type mismatch for %s: expected %s, got %T", key, finalField.Type(), value)
			}
			
			finalField.Set(convertedValue.Convert(finalField.Type()))
			return nil
		}
	}
	
	return fmt.Errorf("invalid configuration path: %s", key)
}

// GetSection returns a specific configuration section
func (cm *ConfigManager) GetSection(section string) (interface{}, error) {
	if cm.config == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	configValue := reflect.ValueOf(cm.config).Elem()
	field := configValue.FieldByName(strings.Title(section))
	
	if !field.IsValid() {
		return nil, fmt.Errorf("invalid configuration section: %s", section)
	}
	
	return field.Interface(), nil
}

// SetSection updates a specific configuration section
func (cm *ConfigManager) SetSection(section string, value interface{}) error {
	if cm.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	configValue := reflect.ValueOf(cm.config).Elem()
	field := configValue.FieldByName(strings.Title(section))
	
	if !field.IsValid() {
		return fmt.Errorf("invalid configuration section: %s", section)
	}
	
	// Convert value to appropriate type
	convertedValue := reflect.ValueOf(value)
	if !convertedValue.Type().ConvertibleTo(field.Type()) {
		return fmt.Errorf("type mismatch for section %s: expected %s, got %T", section, field.Type(), value)
	}
	
	field.Set(convertedValue.Convert(field.Type()))
	return cm.Save()
}

// StartWatching starts watching for configuration changes
func (cm *ConfigManager) StartWatching(callback func(*Config) error) error {
	if cm.watcher != nil {
		return fmt.Errorf("configuration watcher already started")
	}

	cm.watcher = NewConfigWatcher(cm.configPath, callback)
	return cm.watcher.Start()
}

// StopWatching stops watching for configuration changes
func (cm *ConfigManager) StopWatching() {
	if cm.watcher != nil {
		cm.watcher.Stop()
		cm.watcher = nil
	}
}

// Reload reloads the configuration from file
func (cm *ConfigManager) Reload() error {
	return cm.Load()
}

// Backup creates a backup of the current configuration
func (cm *ConfigManager) Backup() error {
	return BackupConfiguration(cm.configPath)
}

// Restore restores configuration from a backup
func (cm *ConfigManager) Restore(backupPath string) error {
	if err := RestoreConfiguration(backupPath, cm.configPath); err != nil {
		return err
	}
	return cm.Reload()
}

// ListBackups returns a list of available backups
func (cm *ConfigManager) ListBackups() ([]string, error) {
	return ListBackups(cm.configPath)
}

// CleanupBackups removes old backups
func (cm *ConfigManager) CleanupBackups(keepCount int) error {
	return CleanupOldBackups(cm.configPath, keepCount)
}

// Validate validates the current configuration
func (cm *ConfigManager) Validate() error {
	if cm.config == nil {
		return fmt.Errorf("configuration not loaded")
	}
	return cm.config.Validate()
}

// GetVersion returns the current configuration version
func (cm *ConfigManager) GetVersion() string {
	if cm.config == nil {
		return ""
	}
	return cm.config.App.Version
}

// Migrate migrates the configuration to the latest version
func (cm *ConfigManager) Migrate() error {
	if cm.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Create backup before migration
	if err := cm.Backup(); err != nil {
		return fmt.Errorf("failed to create backup before migration: %w", err)
	}

	// Perform migration
	targetVersion := "1.3.0" // Latest version
	if err := cm.migrator.Migrate(cm.config, targetVersion); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Save migrated configuration
	return cm.Save()
}

// CheckForUpdates checks if there are available configuration updates
func (cm *ConfigManager) CheckForUpdates() (bool, string, error) {
	if cm.config == nil {
		return false, "", fmt.Errorf("configuration not loaded")
	}
	return CheckForUpdates(cm.config)
}

// GetMigrationSummary returns a summary of available migrations
func (cm *ConfigManager) GetMigrationSummary() string {
	return GetMigrationSummary()
}

// Reset resets configuration to defaults
func (cm *ConfigManager) Reset() error {
	// Create backup before reset
	if err := cm.Backup(); err != nil {
		return fmt.Errorf("failed to create backup before reset: %w", err)
	}

	// Reset to defaults
	cm.config = DefaultConfig()
	return cm.Save()
}

// Export exports configuration to a different format
func (cm *ConfigManager) Export(outputPath string, format string) error {
	if cm.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "yaml", "yml":
		data, err = yaml.Marshal(cm.config)
	case "json":
		data, err = json.Marshal(cm.config)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// Import imports configuration from a different format
func (cm *ConfigManager) Import(inputPath string, format string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var config Config

	switch strings.ToLower(format) {
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &config)
	case "json":
		err = json.Unmarshal(data, &config)
	default:
		return fmt.Errorf("unsupported import format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate imported configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("imported configuration validation failed: %w", err)
	}

	// Create backup before import
	if err := cm.Backup(); err != nil {
		return fmt.Errorf("failed to create backup before import: %w", err)
	}

	// Set imported configuration
	cm.config = &config
	return cm.Save()
}

// Close cleans up resources
func (cm *ConfigManager) Close() {
	cm.StopWatching()
}