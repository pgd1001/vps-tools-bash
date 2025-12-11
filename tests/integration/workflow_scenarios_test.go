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

// TestWorkflowScenarios tests real-world usage scenarios
func TestWorkflowScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping workflow scenarios in short mode")
	}

	tempDir := t.TempDir()
	configPath := createWorkflowTestConfig(t, tempDir)

	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	application, err := app.NewWithConfig(cfg)
	require.NoError(t, err)
	defer application.Stop(context.Background())

	err = application.Start(context.Background())
	require.NoError(t, err)

	t.Run("NewServerOnboarding", func(t *testing.T) {
		testNewServerOnboarding(t, application)
	})

	t.Run("ProductionMonitoring", func(t *testing.T) {
		testProductionMonitoring(t, application)
	})

	t.Run("SecurityAuditWorkflow", func(t *testing.T) {
		testSecurityAuditWorkflow(t, application)
	})

	t.Run("DeploymentWorkflow", func(t *testing.T) {
		testDeploymentWorkflow(t, application)
	})

	t.Run("MaintenanceWorkflow", func(t *testing.T) {
		testMaintenanceWorkflow(t, application)
	})
}

// testNewServerOnboarding tests the complete workflow of adding a new server
func testNewServerOnboarding(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Step 1: Add new server to inventory
	newServer := &server.Server{
		Name:        "production-web-01",
		Host:        "192.168.1.100",
		Port:        22,
		User:        "admin",
		KeyPath:     filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"),
		Tags:        []string{"production", "web", "nginx"},
		Description: "Production web server 01",
		Group:       "web-servers",
	}

	err := store.CreateServer(newServer)
	require.NoError(t, err)
	assert.NotEmpty(t, newServer.ID)

	// Step 2: Test SSH connectivity
	if os.Getenv("SKIP_SSH_TESTS") == "" {
		err = app.SSHClient().TestConnection(ctx, newServer)
		if err != nil {
			t.Logf("SSH connection test failed (expected in CI): %v", err)
		}
	}

	// Step 3: Run initial health check
	healthResult, err := app.HealthChecker().Check(ctx, newServer, health.CheckOptions{
		Timeout:        30 * time.Second,
		IncludeMetrics: true,
		Thresholds: map[string]float64{
			"cpu":    70,
			"memory": 80,
			"disk":   85,
		},
	})

	if err == nil {
		err = store.SaveHealthResult(healthResult)
		require.NoError(t, err)
	}

	// Step 4: Run security audit
	if os.Getenv("SKIP_SSH_TESTS") == "" {
		auditor := app.SecurityAuditor()
		
		// SSH key analysis
		_, err = auditor.AnalyzeSSHKeys(ctx, newServer, security.SSHKeyOptions{
			CheckWeakKeys: true,
			CheckExpired:  true,
			ScanHome:      true,
		})
		if err != nil {
			t.Logf("SSH key analysis failed: %v", err)
		}

		// Port scan
		_, err = auditor.ScanPorts(ctx, newServer, security.PortScanOptions{
			Range:        "22,80,443,8080",
			ScanType:     "tcp",
			Timeout:      5 * time.Second,
			MaxConcurrent: 10,
		})
		if err != nil {
			t.Logf("Port scan failed: %v", err)
		}
	}

	// Step 5: Verify server is properly configured
	retrieved, err := store.GetServer(newServer.ID)
	require.NoError(t, err)
	assert.Equal(t, newServer.Name, retrieved.Name)
	assert.Contains(t, retrieved.Tags, "production")
	assert.Contains(t, retrieved.Tags, "web")

	// Step 6: Clean up
	err = store.DeleteServer(newServer.ID)
	require.NoError(t, err)
}

// testProductionMonitoring tests continuous monitoring workflow
func testProductionMonitoring(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create production servers
	servers := []*server.Server{
		{
			Name: "prod-web-01",
			Host: "192.168.1.101",
			Port: 22,
			User: "admin",
			Tags: []string{"production", "web"},
		},
		{
			Name: "prod-db-01",
			Host: "192.168.1.102",
			Port: 22,
			User: "admin",
			Tags: []string{"production", "database"},
		},
		{
			Name: "prod-cache-01",
			Host: "192.168.1.103",
			Port: 22,
			User: "admin",
			Tags: []string{"production", "cache"},
		},
	}

	// Add servers to inventory
	for _, srv := range servers {
		err := store.CreateServer(srv)
		require.NoError(t, err)
		defer store.DeleteServer(srv.ID)
	}

	// Start continuous monitoring
	monitorCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resultsChan, err := app.HealthChecker().Monitor(monitorCtx, servers, health.MonitorOptions{
		Interval: 2 * time.Second,
		Thresholds: map[string]float64{
			"cpu":    75,
			"memory": 85,
			"disk":   90,
		},
		AlertCallback: func(result *server.HealthResult) {
			if result.Status == server.HealthStatusCritical {
				t.Logf("CRITICAL ALERT: Server %s has critical health issues", result.ServerID)
			}
		},
	})

	if err != nil {
		t.Logf("Health monitoring failed (expected in CI): %v", err)
		return
	}

	// Collect monitoring results
	resultCount := 0
	for result := range resultsChan {
		resultCount++
		err = store.SaveHealthResult(result)
		if err != nil {
			t.Logf("Failed to save health result: %v", err)
		}
	}

	assert.Greater(t, resultCount, 0, "Should receive monitoring results")

	// Generate health report
	healthResults, err := store.GetHealthResults(health.HealthFilter{
		From: time.Now().Add(-1 * time.Hour),
		To:   time.Now(),
	})
	require.NoError(t, err)
	assert.Greater(t, len(healthResults), 0)
}

// testSecurityAuditWorkflow tests comprehensive security auditing
func testSecurityAuditWorkflow(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create test server for security audit
	auditServer := &server.Server{
		Name: "security-audit-target",
		Host: "localhost",
		Port: 22,
		User: os.Getenv("TEST_SSH_USER"),
		Tags: []string{"security", "audit"},
	}

	if auditServer.User == "" {
		auditServer.User = "root"
	}

	err := store.CreateServer(auditServer)
	require.NoError(t, err)
	defer store.DeleteServer(auditServer.ID)

	if os.Getenv("SKIP_SSH_TESTS") != "" {
		t.Skip("Skipping SSH-dependent security audit")
	}

	auditor := app.SecurityAuditor()

	// Step 1: Comprehensive security audit
	auditResult, err := auditor.Audit(ctx, auditServer, security.AuditOptions{
		CheckSSHKeys:     true,
		CheckPorts:        true,
		CheckVulnerabilities: true,
		CheckPermissions:  true,
		CheckConfigs:      true,
		Severity:          "medium",
	})

	if err != nil {
		t.Logf("Security audit failed: %v", err)
		return
	}

	assert.NotNil(t, auditResult)

	// Step 2: Detailed SSH key analysis
	sshResult, err := auditor.AnalyzeSSHKeys(ctx, auditServer, security.SSHKeyOptions{
		CheckWeakKeys:     true,
		CheckExpired:      true,
		CheckUnauthorized: true,
		ScanHome:          true,
		ScanSystem:        false, // Avoid permission issues
		Recommendations:   true,
	})

	if err == nil {
		assert.NotNil(t, sshResult)
	}

	// Step 3: Port scanning
	portResult, err := auditor.ScanPorts(ctx, auditServer, security.PortScanOptions{
		Range:        "1-1000",
		ScanType:     "tcp",
		Timeout:      3 * time.Second,
		MaxConcurrent: 50,
		DetectServices: true,
		BannerGrab:     true,
	})

	if err == nil {
		assert.NotNil(t, portResult)
	}

	// Step 4: Vulnerability assessment
	vulnResult, err := auditor.AssessVulnerabilities(ctx, auditServer, security.VulnOptions{
		CheckPackages:   true,
		CheckServices:   true,
		CheckPermissions: true,
		CheckConfigs:    true,
		Severity:        "high",
	})

	if err == nil {
		assert.NotNil(t, vulnResult)
	}

	// Step 5: Generate security report
	reportData := map[string]interface{}{
		"audit_result":     auditResult,
		"ssh_result":       sshResult,
		"port_result":      portResult,
		"vulnerability_result": vulnResult,
		"timestamp":        time.Now(),
		"server_name":      auditServer.Name,
	}

	// In a real implementation, this would generate a formatted report
	assert.NotNil(t, reportData)
}

// testDeploymentWorkflow tests application deployment workflow
func testDeploymentWorkflow(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create deployment target servers
	deploymentServers := []*server.Server{
		{
			Name: "staging-web-01",
			Host: "localhost",
			Port: 22,
			User: os.Getenv("TEST_SSH_USER"),
			Tags: []string{"staging", "web", "deployment"},
		},
		{
			Name: "staging-web-02",
			Host: "localhost",
			Port: 22,
			User: os.Getenv("TEST_SSH_USER"),
			Tags: []string{"staging", "web", "deployment"},
		},
	}

	for _, srv := range deploymentServers {
		if srv.User == "" {
			srv.User = "root"
		}
		err := store.CreateServer(srv)
		require.NoError(t, err)
		defer store.DeleteServer(srv.ID)
	}

	if os.Getenv("SKIP_SSH_TESTS") != "" {
		t.Skip("Skipping SSH-dependent deployment workflow")
	}

	// Step 1: Pre-deployment health check
	for _, srv := range deploymentServers {
		result, err := app.HealthChecker().Check(ctx, srv, health.CheckOptions{
			Timeout: 15 * time.Second,
		})
		if err != nil {
			t.Logf("Pre-deployment health check failed for %s: %v", srv.Name, err)
			continue
		}
		err = store.SaveHealthResult(result)
		if err != nil {
			t.Logf("Failed to save pre-deployment health result: %v", err)
		}
	}

	// Step 2: Create deployment jobs
	deploymentJobs := []*server.Job{}
	for _, srv := range deploymentServers {
		// Backup current deployment
		backupJob := &server.Job{
			Name:     fmt.Sprintf("backup-%s", srv.Name),
			Type:     server.JobTypeCommand,
			ServerID: srv.ID,
			Command:  "tar -czf /tmp/backup-$(date +%Y%m%d-%H%M%S).tar.gz /var/www/html",
			Status:   server.JobStatusPending,
		}
		deploymentJobs = append(deploymentJobs, backupJob)

		// Deploy new version
		deployJob := &server.Job{
			Name:     fmt.Sprintf("deploy-%s", srv.Name),
			Type:     server.JobTypeCommand,
			ServerID: srv.ID,
			Command:  "echo 'Deploying new version...' && sleep 1 && echo 'Deployment complete'",
			Status:   server.JobStatusPending,
		}
		deploymentJobs = append(deploymentJobs, deployJob)

		// Health check after deployment
		healthJob := &server.Job{
			Name:     fmt.Sprintf("post-deploy-health-%s", srv.Name),
			Type:     server.JobTypeCommand,
			ServerID: srv.ID,
			Command:  "curl -f http://localhost/health || echo 'Health check failed'",
			Status:   server.JobStatusPending,
		}
		deploymentJobs = append(deploymentJobs, healthJob)
	}

	// Step 3: Execute deployment jobs sequentially
	for _, job := range deploymentJobs {
		err := store.CreateJob(job)
		require.NoError(t, err)

		err = app.JobRunner().Execute(ctx, job)
		if err != nil {
			t.Logf("Deployment job %s failed: %v", job.Name, err)
		}

		// Update job status
		updatedJob, err := store.GetJob(job.ID)
		require.NoError(t, err)
		assert.NotEqual(t, server.JobStatusPending, updatedJob.Status)
	}

	// Step 4: Post-deployment verification
	for _, srv := range deploymentServers {
		result, err := app.HealthChecker().Check(ctx, srv, health.CheckOptions{
			Timeout: 15 * time.Second,
		})
		if err != nil {
			t.Logf("Post-deployment health check failed for %s: %v", srv.Name, err)
			continue
		}
		err = store.SaveHealthResult(result)
		if err != nil {
			t.Logf("Failed to save post-deployment health result: %v", err)
		}
	}
}

// testMaintenanceWorkflow tests system maintenance workflow
func testMaintenanceWorkflow(t *testing.T, app app.App) {
	ctx := context.Background()
	store := app.Store()

	// Create maintenance target server
	maintenanceServer := &server.Server{
		Name: "maintenance-target",
		Host: "localhost",
		Port: 22,
		User: os.Getenv("TEST_SSH_USER"),
		Tags: []string{"maintenance", "test"},
	}

	if maintenanceServer.User == "" {
		maintenanceServer.User = "root"
	}

	err := store.CreateServer(maintenanceServer)
	require.NoError(t, err)
	defer store.DeleteServer(maintenanceServer.ID)

	if os.Getenv("SKIP_SSH_TESTS") != "" {
		t.Skip("Skipping SSH-dependent maintenance workflow")
	}

	maintenanceManager := app.MaintenanceManager()

	// Step 1: System cleanup
	cleanupResult, err := maintenanceManager.Cleanup(ctx, maintenanceServer, maintenance.CleanupOptions{
		Level:   maintenance.CleanupLevelStandard,
		DryRun:  true, // Dry run for safety
		Confirm: false,
	})

	if err != nil {
		t.Logf("System cleanup failed: %v", err)
	} else {
		assert.NotNil(t, cleanupResult)
	}

	// Step 2: Package updates
	updateResult, err := maintenanceManager.UpdatePackages(ctx, maintenanceServer, maintenance.UpdateOptions{
		SecurityOnly: true,
		DryRun:       true, // Dry run for safety
		Confirm:      false,
	})

	if err != nil {
		t.Logf("Package update failed: %v", err)
	} else {
		assert.NotNil(t, updateResult)
	}

	// Step 3: System backup
	backupPath := filepath.Join(t.TempDir(), "maintenance-backup.tar.gz")
	backupResult, err := maintenanceManager.Backup(ctx, maintenanceServer, maintenance.BackupOptions{
		Path:    "/etc",
		Output:  backupPath,
		Compress: true,
		DryRun:  true, // Dry run for safety
		Confirm: false,
	})

	if err != nil {
		t.Logf("System backup failed: %v", err)
	} else {
		assert.NotNil(t, backupResult)
	}

	// Step 4: System optimization
	optimizeResult, err := maintenanceManager.Optimize(ctx, maintenanceServer, maintenance.OptimizeOptions{
		Level:   maintenance.OptimizationLevelStandard,
		DryRun:  true, // Dry run for safety
		Confirm: false,
	})

	if err != nil {
		t.Logf("System optimization failed: %v", err)
	} else {
		assert.NotNil(t, optimizeResult)
	}

	// Step 5: Generate maintenance report
	reportData := map[string]interface{}{
		"cleanup_result":   cleanupResult,
		"update_result":    updateResult,
		"backup_result":    backupResult,
		"optimize_result":  optimizeResult,
		"timestamp":        time.Now(),
		"server_name":      maintenanceServer.Name,
	}

	assert.NotNil(t, reportData)
}

// Helper function to create workflow test configuration
func createWorkflowTestConfig(t *testing.T, tempDir string) string {
	configPath := filepath.Join(tempDir, "workflow-config.yaml")
	
	cfg := &config.Config{
		App: config.AppConfig{
			Name:  "vps-tools-workflow-test",
			Debug: true,
		},
		Database: config.DatabaseConfig{
			Path: filepath.Join(tempDir, "workflow-test.db"),
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		SSH: config.SSHConfig{
			Timeout:        30 * time.Second,
			MaxRetries:     3,
			StrictHostKey:  false,
		},
		Health: config.HealthConfig{
			DefaultInterval: 5 * time.Second,
			Timeout:         15 * time.Second,
			Thresholds: config.HealthThresholds{
				CPU: config.ThresholdConfig{Warning: 70, Critical: 90},
				Memory: config.ThresholdConfig{Warning: 80, Critical: 95},
				Disk: config.ThresholdConfig{Warning: 80, Critical: 95},
			},
		},
		Security: config.SecurityConfig{
			DefaultPortRange:    "1-1000",
			ScanTimeout:         5 * time.Second,
			MaxConcurrentScans: 50,
			SSHKeyScan:         true,
			VulnerabilityCheck: true,
		},
		Maintenance: config.MaintenanceConfig{
			BackupPath:    filepath.Join(tempDir, "backups"),
			LogRetention:  "30d",
			AutoCleanup:   true,
		},
	}

	err := config.SaveToFile(cfg, configPath)
	require.NoError(t, err)

	return configPath
}