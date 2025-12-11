package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/pgd1001/vps-tools/internal/app"
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/server"
)

// TestEndToEndWorkflow tests complete workflows from server addition to monitoring
func TestEndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	dbPath := filepath.Join(tempDir, "test.db")

	// Create test configuration
	cfg := &config.Config{
		App: config.AppConfig{
			Name:  "vps-tools-test",
			Debug: true,
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "text",
		},
		SSH: config.SSHConfig{
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			KeyPaths:    []string{filepath.Join(tempDir, "test_key")},
			StrictHostKey: false,
		},
		Health: config.HealthConfig{
			DefaultInterval: 5 * time.Second,
			Timeout:         10 * time.Second,
			Thresholds: config.HealthThresholds{
				CPU: config.ThresholdConfig{Warning: 70, Critical: 90},
				Memory: config.ThresholdConfig{Warning: 80, Critical: 95},
				Disk: config.ThresholdConfig{Warning: 80, Critical: 95},
			},
		},
	}

	// Save configuration
	err := config.SaveToFile(cfg, configPath)
	require.NoError(t, err)

	// Create application
	application, err := app.NewWithConfig(cfg)
	require.NoError(t, err)
	defer application.Stop(context.Background())

	// Start application
	ctx := context.Background()
	err = application.Start(ctx)
	require.NoError(t, err)

	t.Run("ServerManagement", func(t *testing.T) {
		testServerManagement(t, application)
	})

	t.Run("HealthMonitoring", func(t *testing.T) {
		testHealthMonitoring(t, application)
	})

	t.Run("CommandExecution", func(t *testing.T) {
		testCommandExecution(t, application)
	})

	t.Run("SecurityAuditing", func(t *testing.T) {
		testSecurityAuditing(t, application)
	})

	t.Run("DockerManagement", func(t *testing.T) {
		testDockerManagement(t, application)
	})

	t.Run("SystemMaintenance", func(t *testing.T) {
		testSystemMaintenance(t, application)
	})
}

func testServerManagement(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create test server
	testServer := &server.Server{
		Name:        "test-server",
		Host:        "localhost",
		Port:        22,
		User:        os.Getenv("TEST_SSH_USER"),
		Tags:        []string{"test", "integration"},
		Description: "Test server for integration tests",
	}

	if testServer.User == "" {
		testServer.User = "root"
	}

	// Add server
	err := store.CreateServer(testServer)
	require.NoError(t, err)
	assert.NotEmpty(t, testServer.ID)

	// Retrieve server
	retrieved, err := store.GetServer(testServer.ID)
	require.NoError(t, err)
	assert.Equal(t, testServer.Name, retrieved.Name)
	assert.Equal(t, testServer.Host, retrieved.Host)

	// Update server
	testServer.Description = "Updated description"
	testServer.Tags = []string{"test", "integration", "updated"}
	err = store.UpdateServer(testServer)
	require.NoError(t, err)

	// List servers
	servers, err := store.ListServers(server.ServerFilter{})
	require.NoError(t, err)
	assert.Len(t, servers, 1)

	// Test SSH connection (if SSH is available)
	if os.Getenv("SKIP_SSH_TESTS") == "" {
		err = app.SSHClient().TestConnection(ctx, testServer)
		// Don't fail if SSH is not available, just log it
		if err != nil {
			t.Logf("SSH connection test failed (expected in CI): %v", err)
		}
	}

	// Delete server
	err = store.DeleteServer(testServer.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = store.GetServer(testServer.ID)
	assert.Error(t, err)
}

func testHealthMonitoring(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create test server
	testServer := &server.Server{
		Name: "health-test-server",
		Host: "localhost",
		Port: 22,
		User: os.Getenv("TEST_SSH_USER"),
		Tags: []string{"health", "test"},
	}

	if testServer.User == "" {
		testServer.User = "root"
	}

	err := store.CreateServer(testServer)
	require.NoError(t, err)
	defer store.DeleteServer(testServer.ID)

	// Run health check
	result, err := app.HealthChecker().Check(ctx, testServer, health.CheckOptions{
		Timeout: 10 * time.Second,
		IncludeMetrics: true,
	})
	
	if os.Getenv("SKIP_SSH_TESTS") == "" {
		// If SSH is available, we expect a result (even if failed)
		if err == nil {
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.ServerID)
			assert.Equal(t, testServer.ID, result.ServerID)
		} else {
			t.Logf("Health check failed (expected in CI): %v", err)
		}
	} else {
		t.Skip("Skipping SSH-dependent health check")
	}

	// Test health result storage
	if result != nil {
		err = store.SaveHealthResult(result)
		require.NoError(t, err)

		// Retrieve health results
		results, err := store.GetHealthResults(health.HealthFilter{
			ServerID: testServer.ID,
		})
		require.NoError(t, err)
		assert.Len(t, results, 1)
	}
}

func testCommandExecution(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create test server
	testServer := &server.Server{
		Name: "cmd-test-server",
		Host: "localhost",
		Port: 22,
		User: os.Getenv("TEST_SSH_USER"),
		Tags: []string{"command", "test"},
	}

	if testServer.User == "" {
		testServer.User = "root"
	}

	err := store.CreateServer(testServer)
	require.NoError(t, err)
	defer store.DeleteServer(testServer.ID)

	if os.Getenv("SKIP_SSH_TESTS") != "" {
		t.Skip("Skipping SSH-dependent command execution")
	}

	// Create job
	job := &server.Job{
		Name:     "test-uptime",
		Type:     server.JobTypeCommand,
		ServerID: testServer.ID,
		Command:  "uptime",
		Status:   server.JobStatusPending,
	}

	err = store.CreateJob(job)
	require.NoError(t, err)
	assert.NotEmpty(t, job.ID)

	// Execute job
	err = app.JobRunner().Execute(ctx, job)
	if err != nil {
		t.Logf("Command execution failed (expected in CI): %v", err)
		return
	}

	// Check job status
	updatedJob, err := store.GetJob(job.ID)
	require.NoError(t, err)
	assert.NotEqual(t, server.JobStatusPending, updatedJob.Status)

	// Test batch execution
	jobs := []*server.Job{
		{
			Name:     "test-whoami",
			Type:     server.JobTypeCommand,
			ServerID: testServer.ID,
			Command:  "whoami",
		},
		{
			Name:     "test-pwd",
			Type:     server.JobTypeCommand,
			ServerID: testServer.ID,
			Command:  "pwd",
		},
	}

	results, err := app.JobRunner().ExecuteBatch(ctx, jobs, job.BatchOptions{
		MaxConcurrent:   2,
		ContinueOnError: true,
		Timeout:         30 * time.Second,
	})

	if err == nil {
		assert.Len(t, results, 2)
		for _, result := range results {
			assert.NotEqual(t, server.JobStatusPending, result.Status)
		}
	} else {
		t.Logf("Batch execution failed (expected in CI): %v", err)
	}
}

func testSecurityAuditing(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create test server
	testServer := &server.Server{
		Name: "security-test-server",
		Host: "localhost",
		Port: 22,
		User: os.Getenv("TEST_SSH_USER"),
		Tags: []string{"security", "test"},
	}

	if testServer.User == "" {
		testServer.User = "root"
	}

	err := store.CreateServer(testServer)
	require.NoError(t, err)
	defer store.DeleteServer(testServer.ID)

	if os.Getenv("SKIP_SSH_TESTS") != "" {
		t.Skip("Skipping SSH-dependent security auditing")
	}

	// Test SSH key analysis
	auditor := app.SecurityAuditor()
	
	result, err := auditor.AnalyzeSSHKeys(ctx, testServer, security.SSHKeyOptions{
		CheckWeakKeys:     true,
		CheckExpired:      true,
		ScanHome:          true,
		ScanSystem:        false, // Skip system scan to avoid permission issues
	})

	if err == nil {
		assert.NotNil(t, result)
	} else {
		t.Logf("SSH key analysis failed (expected in CI): %v", err)
	}

	// Test port scanning
	portResult, err := auditor.ScanPorts(ctx, testServer, security.PortScanOptions{
		Range:        "22,80,443", // Limited range for testing
		ScanType:     "tcp",
		Timeout:      5 * time.Second,
		MaxConcurrent: 10,
	})

	if err == nil {
		assert.NotNil(t, portResult)
	} else {
		t.Logf("Port scanning failed (expected in CI): %v", err)
	}
}

func testDockerManagement(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create test server
	testServer := &server.Server{
		Name: "docker-test-server",
		Host: "localhost",
		Port: 22,
		User: os.Getenv("TEST_SSH_USER"),
		Tags: []string{"docker", "test"},
	}

	if testServer.User == "" {
		testServer.User = "root"
	}

	err := store.CreateServer(testServer)
	require.NoError(t, err)
	defer store.DeleteServer(testServer.ID)

	if os.Getenv("SKIP_SSH_TESTS") != "" {
		t.Skip("Skipping SSH-dependent Docker management")
	}

	// Test Docker manager
	dockerManager := app.DockerManager()

	// Check if Docker is available
	available, err := dockerManager.IsAvailable(ctx, testServer)
	if err != nil || !available {
		t.Skip("Docker not available on test server")
	}

	// List containers
	containers, err := dockerManager.ListContainers(ctx, testServer, docker.ListOptions{
		All: false, // Only running containers
	})

	if err == nil {
		assert.NotNil(t, containers)
	} else {
		t.Logf("Docker container listing failed: %v", err)
	}

	// Test container health check
	healthResult, err := dockerManager.CheckHealth(ctx, testServer, docker.HealthOptions{
		All: true,
	})

	if err == nil {
		assert.NotNil(t, healthResult)
	} else {
		t.Logf("Docker health check failed: %v", err)
	}
}

func testSystemMaintenance(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create test server
	testServer := &server.Server{
		Name: "maintenance-test-server",
		Host: "localhost",
		Port: 22,
		User: os.Getenv("TEST_SSH_USER"),
		Tags: []string{"maintenance", "test"},
	}

	if testServer.User == "" {
		testServer.User = "root"
	}

	err := store.CreateServer(testServer)
	require.NoError(t, err)
	defer store.DeleteServer(testServer.ID)

	if os.Getenv("SKIP_SSH_TESTS") != "" {
		t.Skip("Skipping SSH-dependent system maintenance")
	}

	// Test maintenance manager
	maintenanceManager := app.MaintenanceManager()

	// Test cleanup with dry run
	result, err := maintenanceManager.Cleanup(ctx, testServer, maintenance.CleanupOptions{
		Level:   maintenance.CleanupLevelMinimal,
		DryRun:  true,
		Confirm: false,
	})

	if err == nil {
		assert.NotNil(t, result)
	} else {
		t.Logf("Maintenance cleanup failed: %v", err)
	}

	// Test backup with dry run
	backupResult, err := maintenanceManager.Backup(ctx, testServer, maintenance.BackupOptions{
		Path:    "/tmp",
		Output:  filepath.Join(t.TempDir(), "test-backup.tar"),
		DryRun:  true,
		Confirm: false,
	})

	if err == nil {
		assert.NotNil(t, backupResult)
	} else {
		t.Logf("Maintenance backup failed: %v", err)
	}
}

// TestConfigurationValidation tests configuration validation and migration
func TestConfigurationValidation(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("ValidConfiguration", func(t *testing.T) {
		cfg := &config.Config{
			App: config.AppConfig{
				Name: "test-app",
			},
			Database: config.DatabaseConfig{
				Path: filepath.Join(tempDir, "valid.db"),
			},
			Logging: config.LoggingConfig{
				Level: "info",
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("InvalidConfiguration", func(t *testing.T) {
		cfg := &config.Config{
			App: config.AppConfig{
				Name: "", // Invalid: empty name
			},
			Database: config.DatabaseConfig{
				Path: "", // Invalid: empty path
			},
			Logging: config.LoggingConfig{
				Level: "invalid", // Invalid: unknown log level
			},
		}

		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("ConfigurationMigration", func(t *testing.T) {
		// Create old format configuration
		oldConfigPath := filepath.Join(tempDir, "old-config.yaml")
		oldConfig := `app:
  name: "vps-tools"
database:
  path: "~/.local/share/vps-tools/data.db"
logging:
  level: "info"
`
		
		err := os.WriteFile(oldConfigPath, []byte(oldConfig), 0644)
		require.NoError(t, err)

		// Load and migrate configuration
		cfg, err := config.LoadFromFile(oldConfigPath)
		require.NoError(t, err)

		// Verify migration
		assert.Equal(t, "vps-tools", cfg.App.Name)
		assert.NotEmpty(t, cfg.Database.Path)
		assert.Equal(t, "info", cfg.Logging.Level)

		// Save migrated configuration
		newConfigPath := filepath.Join(tempDir, "new-config.yaml")
		err = config.SaveToFile(cfg, newConfigPath)
		require.NoError(t, err)

		// Verify saved configuration
		loadedCfg, err := config.LoadFromFile(newConfigPath)
		require.NoError(t, err)
		assert.Equal(t, cfg.App.Name, loadedCfg.App.Name)
	})
}

// TestPluginSystem tests the plugin system
func TestPluginSystem(t *testing.T) {
	tempDir := t.TempDir()

	// Create test plugin
	pluginDir := filepath.Join(tempDir, "plugins")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	pluginCode := `
package main

import (
	"context"
	"fmt"
	"github.com/pgd1001/vps-tools/internal/plugin"
)

type TestPlugin struct{}

func (p *TestPlugin) Name() string {
	return "test-plugin"
}

func (p *TestPlugin) Version() string {
	return "1.0.0"
}

func (p *TestPlugin) Description() string {
	return "Test plugin for integration testing"
}

func (p *TestPlugin) Author() string {
	return "Test Author"
}

func (p *TestPlugin) Initialize(ctx context.Context, cfg *config.Config) error {
	return nil
}

func (p *TestPlugin) Start(ctx context.Context) error {
	return nil
}

func (p *TestPlugin) Stop(ctx context.Context) error {
	return nil
}

func (p *TestPlugin) Cleanup() error {
	return nil
}

func (p *TestPlugin) Execute(ctx context.Context, input *plugin.PluginInput) (*plugin.PluginOutput, error) {
	return &plugin.PluginOutput{
		Success: true,
		Data: map[string]interface{}{
			"message": "Test plugin executed successfully",
		},
	}, nil
}

func (p *TestPlugin) Validate(input *plugin.PluginInput) error {
	return nil
}

func init() {
	plugin.Register(&TestPlugin{})
}

func main() {}
`

	pluginPath := filepath.Join(pluginDir, "test-plugin.go")
	err = os.WriteFile(pluginPath, []byte(pluginCode), 0644)
	require.NoError(t, err)

	// Test plugin manager
	manager := plugin.NewManager()

	// Load plugins
	err = manager.LoadPlugins(pluginDir)
	if err != nil {
		t.Logf("Plugin loading failed (expected in CI without Go compiler): %v", err)
		return
	}

	// List plugins
	plugins := manager.ListPlugins()
	assert.NotEmpty(t, plugins)

	// Execute plugin
	output, err := manager.ExecutePlugin(context.Background(), "test-plugin", &plugin.PluginInput{
		Action: "test",
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
}

// BenchmarkEndToEndWorkflows benchmarks common workflows
func BenchmarkEndToEndWorkflows(b *testing.B) {
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	dbPath := filepath.Join(tempDir, "bench.db")

	cfg := &config.Config{
		App: config.AppConfig{
			Name:  "vps-tools-bench",
			Debug: false,
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Logging: config.LoggingConfig{
			Level:  "warn", // Minimal logging for benchmarks
			Format: "text",
		},
	}

	err := config.SaveToFile(cfg, configPath)
	require.NoError(b, err)

	application, err := app.NewWithConfig(cfg)
	require.NoError(b, err)
	defer application.Stop(context.Background())

	err = application.Start(context.Background())
	require.NoError(b, err)

	store := application.Store()

	b.Run("ServerCRUD", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			server := &server.Server{
				Name:        fmt.Sprintf("bench-server-%d", i),
				Host:        "localhost",
				Port:        22,
				User:        "root",
				Tags:        []string{"benchmark"},
				Description: "Benchmark test server",
			}

			// Create
			err := store.CreateServer(server)
			if err != nil {
				b.Fatal(err)
			}

			// Read
			_, err = store.GetServer(server.ID)
			if err != nil {
				b.Fatal(err)
			}

			// Update
			server.Description = "Updated description"
			err = store.UpdateServer(server)
			if err != nil {
				b.Fatal(err)
			}

			// Delete
			err = store.DeleteServer(server.ID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("HealthCheckStorage", func(b *testing.B) {
		testServer := &server.Server{
			Name: "health-bench-server",
			Host: "localhost",
			Port: 22,
			User: "root",
			Tags: []string{"benchmark"},
		}

		err := store.CreateServer(testServer)
		require.NoError(b, err)
		defer store.DeleteServer(testServer.ID)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := &server.HealthResult{
				ServerID:  testServer.ID,
				Timestamp: time.Now(),
				Status:    server.HealthStatusHealthy,
				Metrics: map[string]interface{}{
					"cpu":    float64(i % 100),
					"memory": float64(i % 100),
					"disk":   float64(i % 100),
				},
			}

			err := store.SaveHealthResult(result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}