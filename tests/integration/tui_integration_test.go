package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/pgd1001/vps-tools/internal/app"
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/server"
)

// TestTUIIntegration tests the TUI interface integration
func TestTUIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TUI integration test in short mode")
	}

	tempDir := t.TempDir()
	configPath := createTestConfig(t, tempDir)

	// Create application
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	application, err := app.NewWithConfig(cfg)
	require.NoError(t, err)
	defer application.Stop(context.Background())

	err = application.Start(context.Background())
	require.NoError(t, err)

	t.Run("TUIModelInitialization", func(t *testing.T) {
		testTUIModelInitialization(t, application)
	})

	t.Run("TUIModelUpdates", func(t *testing.T) {
		testTUIModelUpdates(t, application)
	})

	t.Run("TUIComponentInteraction", func(t *testing.T) {
		testTUIComponentInteraction(t, application)
	})
}

func testTUIModelInitialization(t *testing.T, app app.App) {
	// Create TUI model
	model := tui.NewModel(app)

	// Initialize model
	cmd := model.Init()
	assert.NotNil(t, cmd)

	// Check initial state
	assert.Equal(t, tui.ServersView, model.CurrentView())
	assert.Empty(t, model.Error())
	assert.NotNil(t, model.Servers())
	assert.NotNil(t, model.Health())
	assert.NotNil(t, model.Jobs())
	assert.NotNil(t, model.Notifications())
}

func testTUIModelUpdates(t *testing.T, app app.App) {
	model := tui.NewModel(app)
	model.Init()

	// Test server list update
	cmd := model.Update(tui.ServerListUpdateMsg{})
	assert.NotNil(t, cmd)

	// Test health check update
	healthResult := &server.HealthResult{
		ServerID:  "test-server",
		Timestamp: time.Now(),
		Status:    server.HealthStatusHealthy,
		Metrics: map[string]interface{}{
			"cpu":    50.0,
			"memory": 60.0,
			"disk":   70.0,
		},
	}

	cmd = model.Update(tui.HealthCheckResultMsg{Result: healthResult})
	assert.NotNil(t, cmd)

	// Test job status update
	job := &server.Job{
		ID:     "test-job",
		Name:   "test-job",
		Status: server.JobStatusCompleted,
	}

	cmd = model.Update(tui.JobStatusUpdateMsg{Job: job})
	assert.NotNil(t, cmd)

	// Test notification
	notification := &tui.Notification{
		ID:      "test-notification",
		Title:   "Test Notification",
		Message: "This is a test notification",
		Type:    tui.NotificationTypeInfo,
		Time:    time.Now(),
	}

	cmd = model.Update(tui.NotificationMsg{Notification: notification})
	assert.NotNil(t, cmd)
}

func testTUIComponentInteraction(t *testing.T, app app.App) {
	model := tui.NewModel(app)
	model.Init()

	// Test view switching
	cmd := model.Update(tui.ViewSwitchMsg{View: tui.HealthView})
	assert.NotNil(t, cmd)
	assert.Equal(t, tui.HealthView, model.CurrentView())

	cmd = model.Update(tui.ViewSwitchMsg{View: tui.JobsView})
	assert.NotNil(t, cmd)
	assert.Equal(t, tui.JobsView, model.CurrentView())

	// Test key messages
	cmd = model.Update(tui.KeyMsg{Type: tui.KeyTypeEnter})
	assert.NotNil(t, cmd)

	cmd = model.Update(tui.KeyMsg{Type: tui.KeyTypeEsc})
	assert.NotNil(t, cmd)

	cmd = model.Update(tui.KeyMsg{Type: tui.KeyTypeTab})
	assert.NotNil(t, cmd)

	// Test server creation
	createServerMsg := tui.CreateServerMsg{
		Server: &server.Server{
			Name: "interactive-test-server",
			Host: "localhost",
			Port: 22,
			User: "root",
			Tags: []string{"interactive", "test"},
		},
	}

	cmd = model.Update(createServerMsg)
	assert.NotNil(t, cmd)

	// Test job creation
	createJobMsg := tui.CreateJobMsg{
		Job: &server.Job{
			Name:     "interactive-test-job",
			Type:     server.JobTypeCommand,
			ServerID: "interactive-test-server",
			Command:  "echo 'Hello from TUI'",
		},
	}

	cmd = model.Update(createJobMsg)
	assert.NotNil(t, cmd)
}

// TestRealTimeUpdates tests real-time update functionality
func TestRealTimeUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-time update test in short mode")
	}

	tempDir := t.TempDir()
	configPath := createTestConfig(t, tempDir)

	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	// Enable real-time updates
	cfg.TUI.RefreshInterval = 100 * time.Millisecond

	application, err := app.NewWithConfig(cfg)
	require.NoError(t, err)
	defer application.Stop(context.Background())

	err = application.Start(context.Background())
	require.NoError(t, err)

	model := tui.NewModel(application)
	model.Init()

	// Start real-time updates
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simulate real-time updates
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Update health metrics
				healthResult := &server.HealthResult{
					ServerID:  "realtime-server",
					Timestamp: time.Now(),
					Status:    server.HealthStatusHealthy,
					Metrics: map[string]interface{}{
						"cpu":    float64(time.Now().Unix() % 100),
						"memory": float64(time.Now().Unix() % 100),
						"disk":   float64(time.Now().Unix() % 100),
					},
				}

				model.Update(tui.HealthCheckResultMsg{Result: healthResult})
			}
		}
	}()

	// Monitor updates for a few seconds
	updateCount := 0
	startTime := time.Now()

	for time.Since(startTime) < 2*time.Second {
		cmd := model.Update(tui.TickMsg{Time: time.Now()})
		if cmd != nil {
			updateCount++
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Verify we received updates
	assert.Greater(t, updateCount, 0, "Should receive real-time updates")
}

// TestTUIErrorHandling tests TUI error handling
func TestTUIErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	configPath := createTestConfig(t, tempDir)

	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	application, err := app.NewWithConfig(cfg)
	require.NoError(t, err)
	defer application.Stop(context.Background())

	err = application.Start(context.Background())
	require.NoError(t, err)

	model := tui.NewModel(application)
	model.Init()

	t.Run("SSHConnectionError", func(t *testing.T) {
		// Create server with invalid host
		invalidServer := &server.Server{
			Name: "invalid-server",
			Host: "invalid-host-that-does-not-exist.com",
			Port: 22,
			User: "root",
		}

		// Try to test connection
		err := application.SSHClient().TestConnection(context.Background(), invalidServer)
		assert.Error(t, err)

		// Update TUI with error
		cmd := model.Update(tui.ErrorMessage{Error: err})
		assert.NotNil(t, cmd)
		assert.NotEmpty(t, model.Error())
	})

	t.Run("HealthCheckError", func(t *testing.T) {
		// Create server for health check
		testServer := &server.Server{
			Name: "health-error-server",
			Host: "localhost",
			Port: 22,
			User: "root",
		}

		// Run health check with short timeout to force error
		_, err := application.HealthChecker().Check(context.Background(), testServer, health.CheckOptions{
			Timeout: 1 * time.Millisecond, // Very short timeout
		})

		// Error is expected, update TUI
		if err != nil {
			cmd := model.Update(tui.ErrorMessage{Error: err})
			assert.NotNil(t, cmd)
		}
	})

	t.Run("JobExecutionError", func(t *testing.T) {
		// Create job with invalid command
		job := &server.Job{
			Name:     "error-job",
			Type:     server.JobTypeCommand,
			ServerID: "invalid-server",
			Command:  "invalid-command-that-does-not-exist",
		}

		// Try to execute job
		err := application.JobRunner().Execute(context.Background(), job)
		assert.Error(t, err)

		// Update TUI with job error
		cmd := model.Update(tui.JobErrorMsg{JobID: job.ID, Error: err})
		assert.NotNil(t, cmd)
	})
}

// TestTUIPerformance tests TUI performance
func TestTUIPerformance(t *testing.T) {
	tempDir := t.TempDir()
	configPath := createTestConfig(t, tempDir)

	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	application, err := app.NewWithConfig(cfg)
	require.NoError(t, err)
	defer application.Stop(context.Background())

	err = application.Start(context.Background())
	require.NoError(t, err)

	model := tui.NewModel(application)
	model.Init()

	// Create test data
	servers := make([]*server.Server, 100)
	for i := 0; i < 100; i++ {
		servers[i] = &server.Server{
			Name:        fmt.Sprintf("perf-server-%d", i),
			Host:        "localhost",
			Port:        22,
			User:        "root",
			Tags:        []string{"performance", "test"},
			Description: fmt.Sprintf("Performance test server %d", i),
		}
	}

	// Benchmark server list updates
	b := testing.B{}
	b.ResetTimer()
	startTime := time.Now()

	for i := 0; i < 1000; i++ {
		server := servers[i%len(servers)]
		cmd := model.Update(tui.ServerListUpdateMsg{Servers: []*server.Server{server}})
		if cmd != nil {
			// Process command
		}
	}

	duration := time.Since(startTime)
	assert.Less(t, duration, 1*time.Second, "TUI should handle 1000 updates in under 1 second")

	// Benchmark health check updates
	healthResults := make([]*server.HealthResult, 100)
	for i := 0; i < 100; i++ {
		healthResults[i] = &server.HealthResult{
			ServerID:  fmt.Sprintf("perf-server-%d", i),
			Timestamp: time.Now(),
			Status:    server.HealthStatusHealthy,
			Metrics: map[string]interface{}{
				"cpu":    float64(i % 100),
				"memory": float64(i % 100),
				"disk":   float64(i % 100),
			},
		}
	}

	startTime = time.Now()
	for i := 0; i < 1000; i++ {
		result := healthResults[i%len(healthResults)]
		cmd := model.Update(tui.HealthCheckResultMsg{Result: result})
		if cmd != nil {
			// Process command
		}
	}

	duration = time.Since(startTime)
	assert.Less(t, duration, 1*time.Second, "TUI should handle 1000 health updates in under 1 second")
}

// Helper function to create test configuration
func createTestConfig(t *testing.T, tempDir string) string {
	configPath := filepath.Join(tempDir, "config.yaml")
	
	cfg := &config.Config{
		App: config.AppConfig{
			Name:  "vps-tools-test",
			Debug: true,
		},
		Database: config.DatabaseConfig{
			Path: filepath.Join(tempDir, "test.db"),
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "text",
		},
		SSH: config.SSHConfig{
			Timeout:        30 * time.Second,
			MaxRetries:     3,
			StrictHostKey:  false,
		},
		Health: config.HealthConfig{
			DefaultInterval: 5 * time.Second,
			Timeout:         10 * time.Second,
		},
		TUI: config.TUIConfig{
			RefreshInterval: 1 * time.Second,
			Theme:           "default",
		},
	}

	err := config.SaveToFile(cfg, configPath)
	require.NoError(t, err)

	return configPath
}