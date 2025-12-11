package plugin

import (
	"context"
	"fmt"
	"plugin"
	"reflect"
	"sync"
	"time"

	"github.com/pgd1001/vps-tools/internal/config"
)

// Plugin interface defines the contract for all plugins
type Plugin interface {
	// Plugin metadata
	Name() string
	Version() string
	Description() string
	Author() string

	// Plugin lifecycle
	Initialize(ctx context.Context, cfg *config.Config) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Cleanup() error

	// Plugin functionality
	Execute(ctx context.Context, input *PluginInput) (*PluginOutput, error)
	Validate(input *PluginInput) error
}

// PluginInput represents input to a plugin
type PluginInput struct {
	Action   string                 `json:"action"`
	Server   *Server                `json:"server,omitempty"`
	Data     map[string]interface{} `json:"data"`
	Options  map[string]interface{} `json:"options"`
	Metadata map[string]string      `json:"metadata"`
}

// PluginOutput represents output from a plugin
type PluginOutput struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error,omitempty"`
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// Server represents a server for plugin operations
type Server struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	User        string            `json:"user"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
}

// PluginInfo contains information about a loaded plugin
type PluginInfo struct {
	Plugin     Plugin `json:"-"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Author     string `json:"author"`
	Description string `json:"description"`
	LoadedAt   time.Time `json:"loaded_at"`
	Status     string `json:"status"`
}

// Manager manages plugin lifecycle and execution
type Manager struct {
	plugins map[string]*PluginInfo
	mu      sync.RWMutex
	config  *config.Config
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		plugins: make(map[string]*PluginInfo),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// LoadPlugins loads plugins from a directory
func (m *Manager) LoadPlugins(directory string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// For now, we'll implement a simple plugin loading mechanism
	// In a real implementation, this would scan the directory for .so files
	// and load them using the Go plugin package

	// Load built-in plugins
	builtinPlugins := []Plugin{
		&MonitoringPlugin{},
		&BackupPlugin{},
		&SecurityPlugin{},
		&NotificationPlugin{},
	}

	for _, p := range builtinPlugins {
		if err := m.loadPlugin(p); err != nil {
			return fmt.Errorf("failed to load plugin %s: %w", p.Name(), err)
		}
	}

	return nil
}

// RegisterPlugin registers a plugin
func (m *Manager) RegisterPlugin(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.loadPlugin(plugin)
}

// loadPlugin loads and initializes a plugin
func (m *Manager) loadPlugin(p Plugin) error {
	name := p.Name()
	
	// Check if plugin already exists
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// Initialize plugin
	if err := p.Initialize(m.ctx, m.config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Store plugin info
	m.plugins[name] = &PluginInfo{
		Plugin:     p,
		Name:       name,
		Version:    p.Version(),
		Author:     p.Author(),
		Description: p.Description(),
		LoadedAt:   time.Now(),
		Status:     "loaded",
	}

	return nil
}

// Start starts all loaded plugins
func (m *Manager) Start() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, info := range m.plugins {
		if err := info.Plugin.Start(m.ctx); err != nil {
			return fmt.Errorf("failed to start plugin %s: %w", name, err)
		}
		info.Status = "running"
	}

	return nil
}

// Stop stops all running plugins
func (m *Manager) Stop() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, info := range m.plugins {
		if info.Status == "running" {
			if err := info.Plugin.Stop(m.ctx); err != nil {
				return fmt.Errorf("failed to stop plugin %s: %w", name, err)
			}
			info.Status = "stopped"
		}
	}

	return nil
}

// GetPlugin returns a plugin by name
func (m *Manager) GetPlugin(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return info.Plugin, nil
}

// ListPlugins returns all loaded plugins
func (m *Manager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var plugins []PluginInfo
	for _, info := range m.plugins {
		plugins = append(plugins, *info)
	}

	return plugins
}

// ExecutePlugin executes a plugin
func (m *Manager) ExecutePlugin(ctx context.Context, name string, input *PluginInput) (*PluginOutput, error) {
	plugin, err := m.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	// Validate input
	if err := plugin.Validate(input); err != nil {
		return &PluginOutput{
			Success: false,
			Error:   fmt.Sprintf("input validation failed: %v", err),
		}, nil
	}

	// Execute plugin
	output, err := plugin.Execute(ctx, input)
	if err != nil {
		return &PluginOutput{
			Success: false,
			Error:   fmt.Sprintf("plugin execution failed: %v", err),
		}, nil
	}

	return output, nil
}

// UnloadPlugin unloads a plugin
func (m *Manager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Stop plugin if running
	if info.Status == "running" {
		if err := info.Plugin.Stop(m.ctx); err != nil {
			return fmt.Errorf("failed to stop plugin %s: %w", name, err)
		}
	}

	// Cleanup plugin
	if err := info.Plugin.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup plugin %s: %w", name, err)
	}

	// Remove from registry
	delete(m.plugins, name)

	return nil
}

// Shutdown shuts down the plugin manager
func (m *Manager) Shutdown() error {
	m.cancel()
	return m.Stop()
}

// SetConfig sets the configuration for the plugin manager
func (m *Manager) SetConfig(cfg *config.Config) {
	m.config = cfg
}

// Built-in plugin implementations

// MonitoringPlugin provides monitoring capabilities
type MonitoringPlugin struct {
	config *config.Config
}

func (p *MonitoringPlugin) Name() string { return "monitoring" }
func (p *MonitoringPlugin) Version() string { return "1.0.0" }
func (p *MonitoringPlugin) Author() string { return "VPS Tools Team" }
func (p *MonitoringPlugin) Description() string { return "Built-in monitoring plugin" }

func (p *MonitoringPlugin) Initialize(ctx context.Context, cfg *config.Config) error {
	p.config = cfg
	return nil
}

func (p *MonitoringPlugin) Start(ctx context.Context) error {
	return nil
}

func (p *MonitoringPlugin) Stop(ctx context.Context) error {
	return nil
}

func (p *MonitoringPlugin) Cleanup() error {
	return nil
}

func (p *MonitoringPlugin) Execute(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	switch input.Action {
	case "collect_metrics":
		return p.collectMetrics(ctx, input)
	case "analyze_trends":
		return p.analyzeTrends(ctx, input)
	default:
		return nil, fmt.Errorf("unknown action: %s", input.Action)
	}
}

func (p *MonitoringPlugin) Validate(input *PluginInput) error {
	if input.Action == "" {
		return fmt.Errorf("action is required")
	}
	return nil
}

func (p *MonitoringPlugin) collectMetrics(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	// Simulate metric collection
	metrics := map[string]interface{}{
		"cpu_usage":    45.2,
		"memory_usage": 67.8,
		"disk_usage":   23.4,
		"network_in":   1024.5,
		"network_out":  2048.7,
		"timestamp":    time.Now().Unix(),
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"metrics": metrics,
		},
		Metrics: map[string]interface{}{
			"collection_time": time.Since(time.Now()).String(),
		},
	}, nil
}

func (p *MonitoringPlugin) analyzeTrends(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	// Simulate trend analysis
	trends := map[string]interface{}{
		"cpu_trend":      "stable",
		"memory_trend":   "increasing",
		"disk_trend":     "stable",
		"recommendations": []string{"Consider memory optimization"},
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"trends": trends,
		},
	}, nil
}

// BackupPlugin provides backup capabilities
type BackupPlugin struct {
	config *config.Config
}

func (p *BackupPlugin) Name() string { return "backup" }
func (p *BackupPlugin) Version() string { return "1.0.0" }
func (p *BackupPlugin) Author() string { return "VPS Tools Team" }
func (p *BackupPlugin) Description() string { return "Built-in backup plugin" }

func (p *BackupPlugin) Initialize(ctx context.Context, cfg *config.Config) error {
	p.config = cfg
	return nil
}

func (p *BackupPlugin) Start(ctx context.Context) error {
	return nil
}

func (p *BackupPlugin) Stop(ctx context.Context) error {
	return nil
}

func (p *BackupPlugin) Cleanup() error {
	return nil
}

func (p *BackupPlugin) Execute(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	switch input.Action {
	case "create_backup":
		return p.createBackup(ctx, input)
	case "restore_backup":
		return p.restoreBackup(ctx, input)
	case "list_backups":
		return p.listBackups(ctx, input)
	default:
		return nil, fmt.Errorf("unknown action: %s", input.Action)
	}
}

func (p *BackupPlugin) Validate(input *PluginInput) error {
	if input.Action == "" {
		return fmt.Errorf("action is required")
	}
	return nil
}

func (p *BackupPlugin) createBackup(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	// Simulate backup creation
	backupID := fmt.Sprintf("backup_%d", time.Now().Unix())
	
	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"backup_id": backupID,
			"status":    "created",
			"size":      "1.2GB",
			"location":  "/var/backups/" + backupID + ".tar.gz",
		},
		Metrics: map[string]interface{}{
			"backup_time": "2m 15s",
		},
	}, nil
}

func (p *BackupPlugin) restoreBackup(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	backupID, ok := input.Data["backup_id"].(string)
	if !ok {
		return &PluginOutput{
			Success: false,
			Error:   "backup_id is required",
		}, nil
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"backup_id": backupID,
			"status":    "restored",
			"restored_at": time.Now().Format(time.RFC3339),
		},
	}, nil
}

func (p *BackupPlugin) listBackups(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	// Simulate backup listing
	backups := []map[string]interface{}{
		{
			"id":       "backup_1640995200",
			"size":     "1.2GB",
			"created":  "2021-12-31T23:00:00Z",
			"status":   "completed",
		},
		{
			"id":       "backup_1641081600",
			"size":     "1.3GB",
			"created":  "2022-01-01T23:00:00Z",
			"status":   "completed",
		},
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"backups": backups,
			"count":   len(backups),
		},
	}, nil
}

// SecurityPlugin provides security capabilities
type SecurityPlugin struct {
	config *config.Config
}

func (p *SecurityPlugin) Name() string { return "security" }
func (p *SecurityPlugin) Version() string { return "1.0.0" }
func (p *SecurityPlugin) Author() string { return "VPS Tools Team" }
func (p *SecurityPlugin) Description() string { return "Built-in security plugin" }

func (p *SecurityPlugin) Initialize(ctx context.Context, cfg *config.Config) error {
	p.config = cfg
	return nil
}

func (p *SecurityPlugin) Start(ctx context.Context) error {
	return nil
}

func (p *SecurityPlugin) Stop(ctx context.Context) error {
	return nil
}

func (p *SecurityPlugin) Cleanup() error {
	return nil
}

func (p *SecurityPlugin) Execute(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	switch input.Action {
	case "scan_vulnerabilities":
		return p.scanVulnerabilities(ctx, input)
	case "check_permissions":
		return p.checkPermissions(ctx, input)
	case "audit_logs":
		return p.auditLogs(ctx, input)
	default:
		return nil, fmt.Errorf("unknown action: %s", input.Action)
	}
}

func (p *SecurityPlugin) Validate(input *PluginInput) error {
	if input.Action == "" {
		return fmt.Errorf("action is required")
	}
	return nil
}

func (p *SecurityPlugin) scanVulnerabilities(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	// Simulate vulnerability scan
	vulnerabilities := []map[string]interface{}{
		{
			"id":          "CVE-2021-44228",
			"severity":    "critical",
			"package":     "log4j",
			"version":     "2.14.1",
			"description": "Remote code execution vulnerability",
		},
		{
			"id":          "CVE-2021-45046",
			"severity":    "high",
			"package":     "log4j",
			"version":     "2.14.1",
			"description": "Denial of service vulnerability",
		},
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"vulnerabilities": vulnerabilities,
			"count":           len(vulnerabilities),
			"scan_time":       time.Now().Format(time.RFC3339),
		},
		Metrics: map[string]interface{}{
			"scan_duration": "45s",
		},
	}, nil
}

func (p *SecurityPlugin) checkPermissions(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	// Simulate permission check
	issues := []map[string]interface{}{
		{
			"path":         "/etc/passwd",
			"issue":        "world-readable",
			"severity":     "medium",
			"recommendation": "chmod 640 /etc/passwd",
		},
		{
			"path":         "/var/log/auth.log",
			"issue":        "world-writable",
			"severity":     "high",
			"recommendation": "chmod 640 /var/log/auth.log",
		},
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"issues": issues,
			"count":  len(issues),
		},
	}, nil
}

func (p *SecurityPlugin) auditLogs(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	// Simulate log audit
	audit := map[string]interface{}{
		"total_entries":     15420,
		"suspicious_ips":    []string{"192.168.1.100", "10.0.0.50"},
		"failed_logins":     23,
		"privileged_access": 156,
		"audit_period":      "24h",
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"audit": audit,
		},
	}, nil
}

// NotificationPlugin provides notification capabilities
type NotificationPlugin struct {
	config *config.Config
}

func (p *NotificationPlugin) Name() string { return "notification" }
func (p *NotificationPlugin) Version() string { return "1.0.0" }
func (p *NotificationPlugin) Author() string { return "VPS Tools Team" }
func (p *NotificationPlugin) Description() string { return "Built-in notification plugin" }

func (p *NotificationPlugin) Initialize(ctx context.Context, cfg *config.Config) error {
	p.config = cfg
	return nil
}

func (p *NotificationPlugin) Start(ctx context.Context) error {
	return nil
}

func (p *NotificationPlugin) Stop(ctx context.Context) error {
	return nil
}

func (p *NotificationPlugin) Cleanup() error {
	return nil
}

func (p *NotificationPlugin) Execute(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	switch input.Action {
	case "send_notification":
		return p.sendNotification(ctx, input)
	case "send_email":
		return p.sendEmail(ctx, input)
	case "send_webhook":
		return p.sendWebhook(ctx, input)
	default:
		return nil, fmt.Errorf("unknown action: %s", input.Action)
	}
}

func (p *NotificationPlugin) Validate(input *PluginInput) error {
	if input.Action == "" {
		return fmt.Errorf("action is required")
	}
	return nil
}

func (p *NotificationPlugin) sendNotification(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	title, _ := input.Data["title"].(string)
	message, _ := input.Data["message"].(string)
	
	if title == "" || message == "" {
		return &PluginOutput{
			Success: false,
			Error:   "title and message are required",
		}, nil
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"notification_id": fmt.Sprintf("notif_%d", time.Now().Unix()),
			"title":          title,
			"message":        message,
			"sent_at":        time.Now().Format(time.RFC3339),
		},
	}, nil
}

func (p *NotificationPlugin) sendEmail(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	to, _ := input.Data["to"].(string)
	subject, _ := input.Data["subject"].(string)
	body, _ := input.Data["body"].(string)
	
	if to == "" || subject == "" || body == "" {
		return &PluginOutput{
			Success: false,
			Error:   "to, subject, and body are required",
		}, nil
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"email_id": fmt.Sprintf("email_%d", time.Now().Unix()),
			"to":       to,
			"subject":  subject,
			"sent_at":  time.Now().Format(time.RFC3339),
		},
	}, nil
}

func (p *NotificationPlugin) sendWebhook(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	url, _ := input.Data["url"].(string)
	payload, _ := input.Data["payload"].(map[string]interface{})
	
	if url == "" {
		return &PluginOutput{
			Success: false,
			Error:   "url is required",
		}, nil
	}

	return &PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"webhook_id": fmt.Sprintf("webhook_%d", time.Now().Unix()),
			"url":        url,
			"payload":    payload,
			"sent_at":    time.Now().Format(time.RFC3339),
		},
	}, nil
}