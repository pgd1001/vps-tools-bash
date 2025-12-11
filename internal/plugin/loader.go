package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
)

// Loader handles loading of external plugins
type Loader struct {
	loadedPlugins map[string]*plugin.Plugin
	mu           sync.RWMutex
}

// NewLoader creates a new plugin loader
func NewLoader() *Loader {
	return &Loader{
		loadedPlugins: make(map[string]*plugin.Plugin),
	}
}

// LoadFromDirectory loads plugins from a directory
func (l *Loader) LoadFromDirectory(directory string) ([]Plugin, error) {
	var plugins []Plugin

	// Scan directory for .so files
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, ".so") {
			continue
		}

		pluginPath := filepath.Join(directory, filename)
		plugin, err := l.LoadPlugin(pluginPath)
		if err != nil {
			// Log error but continue loading other plugins
			fmt.Printf("Failed to load plugin %s: %v\n", filename, err)
			continue
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// LoadPlugin loads a single plugin from a .so file
func (l *Loader) LoadPlugin(path string) (Plugin, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already loaded
	if _, exists := l.loadedPlugins[path]; exists {
		return nil, fmt.Errorf("plugin already loaded: %s", path)
	}

	// Load the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for the NewPlugin symbol
	newPluginSymbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export NewPlugin symbol: %w", err)
	}

	// Type assert the symbol
	newPlugin, ok := newPluginSymbol.(func() Plugin)
	if !ok {
		return nil, fmt.Errorf("plugin NewPlugin symbol has wrong signature")
	}

	// Create plugin instance
	pluginInstance := newPlugin()

	// Store loaded plugin
	l.loadedPlugins[path] = p

	return pluginInstance, nil
}

// UnloadPlugin unloads a plugin
func (l *Loader) UnloadPlugin(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.loadedPlugins[path]; !exists {
		return fmt.Errorf("plugin not loaded: %s", path)
	}

	delete(l.loadedPlugins, path)
	return nil
}

// Registry manages plugin registration and discovery
type Registry struct {
	plugins map[string]PluginInfo
	mu      sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]PluginInfo),
	}
}

// Register registers a plugin
func (r *Registry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin already registered: %s", name)
	}

	r.plugins[name] = PluginInfo{
		Plugin:     plugin,
		Name:       name,
		Version:    plugin.Version(),
		Author:     plugin.Author(),
		Description: plugin.Description(),
	}

	return nil
}

// Unregister unregisters a plugin
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("plugin not registered: %s", name)
	}

	delete(r.plugins, name)
	return nil
}

// Get gets a plugin by name
func (r *Registry) Get(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	return info.Plugin, nil
}

// List lists all registered plugins
func (r *Registry) List() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var plugins []PluginInfo
	for _, info := range r.plugins {
		plugins = append(plugins, info)
	}

	return plugins
}

// FindByTag finds plugins by tag (if supported)
func (r *Registry) FindByTag(tag string) []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var plugins []PluginInfo
	for _, info := range r.plugins {
		// Check if plugin implements TaggedPlugin interface
		if taggedPlugin, ok := info.Plugin.(interface{ Tags() []string }); ok {
			tags := taggedPlugin.Tags()
			for _, t := range tags {
				if t == tag {
					plugins = append(plugins, info)
					break
				}
			}
		}
	}

	return plugins
}

// Validator validates plugins
type Validator struct {
	allowedPlugins []string
	bannedPlugins  []string
}

// NewValidator creates a new plugin validator
func NewValidator(allowed, banned []string) *Validator {
	return &Validator{
		allowedPlugins: allowed,
		bannedPlugins:  banned,
	}
}

// Validate validates a plugin
func (v *Validator) Validate(plugin Plugin) error {
	name := plugin.Name()

	// Check banned list
	for _, banned := range v.bannedPlugins {
		if name == banned {
			return fmt.Errorf("plugin %s is banned", name)
		}
	}

	// Check allowed list (if specified)
	if len(v.allowedPlugins) > 0 {
		allowed := false
		for _, allowedName := range v.allowedPlugins {
			if name == allowedName {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("plugin %s is not in allowed list", name)
		}
	}

	// Validate plugin interface
	if err := v.validateInterface(plugin); err != nil {
		return fmt.Errorf("plugin interface validation failed: %w", err)
	}

	return nil
}

// validateInterface validates that a plugin implements the required interface
func (v *Validator) validateInterface(plugin Plugin) error {
	// Check required methods
	if plugin.Name() == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if plugin.Version() == "" {
		return fmt.Errorf("plugin version cannot be empty")
	}

	if plugin.Description() == "" {
		return fmt.Errorf("plugin description cannot be empty")
	}

	if plugin.Author() == "" {
		return fmt.Errorf("plugin author cannot be empty")
	}

	return nil
}

// Sandbox provides a sandboxed environment for plugin execution
type Sandbox struct {
	allowedActions map[string]bool
	resourceLimits ResourceLimits
}

// ResourceLimits defines resource limits for plugin execution
type ResourceLimits struct {
	MaxMemoryMB int64
	MaxCPUTime  int64 // in seconds
	MaxFileSize int64 // in bytes
}

// NewSandbox creates a new plugin sandbox
func NewSandbox(allowedActions []string, limits ResourceLimits) *Sandbox {
	actions := make(map[string]bool)
	for _, action := range allowedActions {
		actions[action] = true
	}

	return &Sandbox{
		allowedActions: actions,
		resourceLimits: limits,
	}
}

// Execute executes a plugin in the sandbox
func (s *Sandbox) Execute(ctx context.Context, plugin Plugin, input *PluginInput) (*PluginOutput, error) {
	// Validate action
	if !s.allowedActions[input.Action] {
		return &PluginOutput{
			Success: false,
			Error:   fmt.Sprintf("action %s is not allowed", input.Action),
		}, nil
	}

	// Create sandboxed context with timeout
	sandboxCtx := ctx
	if s.resourceLimits.MaxCPUTime > 0 {
		var cancel context.CancelFunc
		sandboxCtx, cancel = context.WithTimeout(ctx, 
			time.Duration(s.resourceLimits.MaxCPUTime)*time.Second)
		defer cancel()
	}

	// Execute plugin
	output, err := plugin.Execute(sandboxCtx, input)
	if err != nil {
		return &PluginOutput{
			Success: false,
			Error:   fmt.Sprintf("plugin execution failed: %v", err),
		}, nil
	}

	// Validate output
	if err := s.validateOutput(output); err != nil {
		return &PluginOutput{
			Success: false,
			Error:   fmt.Sprintf("plugin output validation failed: %v", err),
		}, nil
	}

	return output, nil
}

// validateOutput validates plugin output
func (s *Sandbox) validateOutput(output *PluginOutput) error {
	// Check output size
	if s.resourceLimits.MaxFileSize > 0 {
		// This is a simplified check - in practice you'd serialize and check size
		if len(output.Error) > int(s.resourceLimits.MaxFileSize) {
			return fmt.Errorf("output too large")
		}
	}

	return nil
}

// PluginBuilder helps build plugins
type PluginBuilder struct {
	name        string
	version     string
	author      string
	description string
	actions     map[string]func(context.Context, *PluginInput) (*PluginOutput, error)
	validator   func(*PluginInput) error
}

// NewPluginBuilder creates a new plugin builder
func NewPluginBuilder(name, version, author, description string) *PluginBuilder {
	return &PluginBuilder{
		name:        name,
		version:     version,
		author:      author,
		description: description,
		actions:     make(map[string]func(context.Context, *PluginInput) (*PluginOutput, error)),
	}
}

// AddAction adds an action to the plugin
func (pb *PluginBuilder) AddAction(name string, handler func(context.Context, *PluginInput) (*PluginOutput, error)) *PluginBuilder {
	pb.actions[name] = handler
	return pb
}

// SetValidator sets the input validator
func (pb *PluginBuilder) SetValidator(validator func(*PluginInput) error) *PluginBuilder {
	pb.validator = validator
	return pb
}

// Build builds the plugin
func (pb *PluginBuilder) Build() Plugin {
	return &builtInPlugin{
		name:        pb.name,
		version:     pb.version,
		author:      pb.author,
		description: pb.description,
		actions:     pb.actions,
		validator:   pb.validator,
	}
}

// builtInPlugin is a plugin built using the builder
type builtInPlugin struct {
	name        string
	version     string
	author      string
	description string
	actions     map[string]func(context.Context, *PluginInput) (*PluginOutput, error)
	validator   func(*PluginInput) error
}

func (p *builtInPlugin) Name() string        { return p.name }
func (p *builtInPlugin) Version() string     { return p.version }
func (p *builtInPlugin) Author() string      { return p.author }
func (p *builtInPlugin) Description() string { return p.description }

func (p *builtInPlugin) Initialize(ctx context.Context, cfg *config.Config) error {
	return nil
}

func (p *builtInPlugin) Start(ctx context.Context) error {
	return nil
}

func (p *builtInPlugin) Stop(ctx context.Context) error {
	return nil
}

func (p *builtInPlugin) Cleanup() error {
	return nil
}

func (p *builtInPlugin) Execute(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
	handler, exists := p.actions[input.Action]
	if !exists {
		return nil, fmt.Errorf("unknown action: %s", input.Action)
	}

	return handler(ctx, input)
}

func (p *builtInPlugin) Validate(input *PluginInput) error {
	if p.validator != nil {
		return p.validator(input)
	}

	if input.Action == "" {
		return fmt.Errorf("action is required")
	}

	return nil
}

// Example plugin creation using the builder
func CreateExamplePlugin() Plugin {
	return NewPluginBuilder(
		"example",
		"1.0.0",
		"Example Author",
		"An example plugin built with the builder",
	).
		AddAction("hello", func(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
			name, _ := input.Data["name"].(string)
			if name == "" {
				name = "World"
			}

			return &PluginOutput{
				Success: true,
				Data: map[string]interface{}{
					"message": fmt.Sprintf("Hello, %s!", name),
				},
			}, nil
		}).
		AddAction("echo", func(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
			message, _ := input.Data["message"].(string)

			return &PluginOutput{
				Success: true,
				Data: map[string]interface{}{
					"echo": message,
				},
			}, nil
		}).
		Build()
}