package health

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/logger"
	"github.com/pgd1001/vps-tools/internal/server"
	"github.com/pgd1001/vps-tools/internal/ssh"
	"github.com/pgd1001/vps-tools/internal/store"
)

// Command represents the health command
type Command struct {
	configManager *config.ConfigManager
	store        store.Store
	logger       *logger.Logger
}

// NewCommand creates a new health command
func NewCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check server health",
		Long:  "Perform health checks on servers and display results",
	}

	// Add subcommands
	cmd.AddCommand(newCheckCommand(configManager, store, logger))
	cmd.AddCommand(newListCommand(configManager, store, logger))
	cmd.AddCommand(newMonitorCommand(configManager, store, logger))

	return cmd
}

// newCheckCommand creates the check command
func newCheckCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [server-id]",
		Short: "Run health check",
		Long:  "Run a health check on specific servers or all servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHealthCheck(configManager, store, logger, cmd, args)
		},
	}

	// Add flags
	cmd.Flags().StringSliceP("servers", "s", []string{}, "Server IDs to check (default: all)")
	cmd.Flags().StringSliceP("checks", "c", []string{}, "Specific checks to run (default: all)")
	cmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
	cmd.Flags().BoolP("verbose", "v", false, "Verbose output")

	return cmd
}

// newListCommand creates the list command
func newListCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List health checks",
		Long:  "List recent health check results",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHealthList(configManager, store, logger, cmd, args)
		},
	}
}

// newMonitorCommand creates the monitor command
func newMonitorCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "monitor [server-id]",
		Short: "Monitor server health",
		Long:  "Continuously monitor server health with real-time updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHealthMonitor(configManager, store, logger, cmd, args)
		},
	}
}

// runHealthCheck executes the health check command
func runHealthCheck(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	checks, _ := cmd.Flags().GetStringSlice("checks")
	outputJSON, _ := cmd.Flags().GetBool("json")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Get servers to check
	var servers []*server.Server
	if len(serverIDs) == 0 {
		// Check all servers
		allServers, err := store.ListServers(nil)
		if err != nil {
			return fmt.Errorf("failed to list servers: %w", err)
		}
		servers = allServers
	} else {
		// Check specific servers
		for _, id := range serverIDs {
			srv, err := store.GetServer(id)
			if err != nil {
				logger.WithError(err).Warnf("Server %s not found, skipping", id)
				continue
			}
			servers = append(servers, srv)
		}
	}

	if len(servers) == 0 {
		logger.Info("No servers to check")
		return nil
	}

	// Perform health checks
	results := make([]*HealthCheckResult, 0)
	for _, srv := range servers {
		result := performHealthCheck(srv, checks, verbose, logger)
		results = append(results, result)
	}

	// Output results
	if outputJSON {
		return outputHealthCheckJSON(results, logger)
	}

	return outputHealthCheckTable(results, logger)
}

// runHealthList executes the health list command
func runHealthList(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverID, _ := cmd.Flags().GetString("server")
	limit, _ := cmd.Flags().GetInt("limit")

	// Get health checks
	var checks []*server.HealthCheck
	var err error

	if serverID != "" {
		checks, err = store.ListHealthChecks(serverID, limit)
	} else {
		// Get latest check for each server
		servers, err := store.ListServers(nil)
		if err != nil {
			return fmt.Errorf("failed to list servers: %w", err)
		}

		for _, srv := range servers {
			serverChecks, err := store.ListHealthChecks(srv.ID, 1)
			if err != nil {
				logger.WithError(err).Warnf("Failed to get health checks for server %s", srv.ID)
				continue
			}
			if len(serverChecks) > 0 {
				checks = append(checks, serverChecks[0])
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to get health checks: %w", err)
	}

	if len(checks) == 0 {
		logger.Info("No health checks found")
		return nil
	}

	// Display results
	displayHealthList(checks, logger)
	return nil
}

// runHealthMonitor executes the health monitor command
func runHealthMonitor(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("server ID is required")
	}

	serverID := args[0]

	// Get server
	srv, err := store.GetServer(serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	logger.WithField("server_id", serverID).Info("Starting health monitoring")

	// Create SSH client
	sshConfig := &ssh.Config{
		Host:       srv.Host,
		Port:       srv.Port,
		User:       srv.User,
		AuthMethod: srv.AuthMethod,
		Timeout:    30 * time.Second,
	}

	sshClient, err := ssh.NewClient(sshConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer sshClient.Close()

	// Connect to server
	ctx := context.Background()
	conn, err := sshClient.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer conn.Close()

	// Continuous monitoring
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Health monitoring stopped")
			return nil
		case <-ticker.C:
			result := performHealthCheck(srv, []string{"disk", "memory", "cpu"}, true, logger)
			
			// Store health check
			healthCheck := &server.HealthCheck{
				ID:        generateHealthCheckID(serverID),
				ServerID:  serverID,
				Timestamp: time.Now(),
				Status:    result.OverallStatus,
				Metrics:   result.Metrics,
				Checks:     result.Checks,
				Duration:  result.Duration,
			}

			if err := store.CreateHealthCheck(healthCheck); err != nil {
				logger.WithError(err).Error("Failed to store health check")
			}

			// Display result
			displayLiveHealthCheck(serverID, result, logger)
		}
	}
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	ServerID      string                    `json:"server_id"`
	Timestamp     time.Time                 `json:"timestamp"`
	OverallStatus server.ServerStatus        `json:"overall_status"`
	Metrics       map[string]interface{}     `json:"metrics"`
	Checks        map[string]CheckResult     `json:"checks"`
	Duration      time.Duration              `json:"duration"`
	Issues        []string                  `json:"issues,omitempty"`
}

// CheckResult represents the result of a specific check
type CheckResult struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
	Unit    string      `json:"unit,omitempty"`
}

// performHealthCheck performs a health check on a server
func performHealthCheck(srv *server.Server, checks []string, verbose bool, logger *logger.Logger) *HealthCheckResult {
	result := &HealthCheckResult{
		ServerID:      srv.ID,
		Timestamp:     time.Now(),
		OverallStatus: server.StatusUnknown,
		Metrics:       make(map[string]interface{}),
		Checks:        make(map[string]CheckResult),
	}

	startTime := time.Now()

	// Create SSH client for this check
	sshConfig := &ssh.Config{
		Host:       srv.Host,
		Port:       srv.Port,
		User:       srv.User,
		AuthMethod: srv.AuthMethod,
		Timeout:    30 * time.Second,
	}

	sshClient, err := ssh.NewClient(sshConfig, logger)
	if err != nil {
		result.OverallStatus = server.StatusError
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to create SSH client: %v", err))
		result.Duration = time.Since(startTime)
		return result
	}
	defer sshClient.Close()

	// Connect to server
	ctx := context.Background()
	conn, err := sshClient.Connect(ctx)
	if err != nil {
		result.OverallStatus = server.StatusOffline
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to connect: %v", err))
		result.Duration = time.Since(startTime)
		return result
	}
	defer conn.Close()

	// Determine which checks to run
	if len(checks) == 0 {
		checks = []string{"disk", "memory", "cpu", "services"}
	}

	// Perform checks
	allPassed := true
	for _, check := range checks {
		checkResult := performSpecificCheck(conn, check, verbose, logger)
		result.Checks[check] = checkResult
		
		if checkResult.Status != "ok" {
			allPassed = false
		}
	}

	// Set overall status
	if allPassed {
		result.OverallStatus = server.StatusOnline
	} else {
		result.OverallStatus = server.StatusError
	}

	result.Duration = time.Since(startTime)

	return result
}

// performSpecificCheck performs a specific health check
func performSpecificCheck(conn *ssh.Connection, check string, verbose bool, logger *logger.Logger) CheckResult {
	switch check {
	case "disk":
		return checkDiskUsage(conn, verbose, logger)
	case "memory":
		return checkMemoryUsage(conn, verbose, logger)
	case "cpu":
		return checkCPUUsage(conn, verbose, logger)
	case "services":
		return checkServices(conn, verbose, logger)
	default:
		return CheckResult{
			Status:  "unknown",
			Message: fmt.Sprintf("Unknown check: %s", check),
		}
	}
}

// checkDiskUsage checks disk usage
func checkDiskUsage(conn *ssh.Connection, verbose bool, logger *logger.Logger) CheckResult {
	// Execute df command
	result, err := conn.ExecuteCommand(context.Background(), "df -h", "")
	if err != nil {
		return CheckResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to check disk usage: %v", err),
		}
	}

	if result.ExitCode != 0 {
		return CheckResult{
			Status:  "error",
			Message: fmt.Sprintf("Disk check failed with exit code %d", result.ExitCode),
			Stderr: result.Stderr,
		}
	}

	// Parse df output (simplified)
	// In production, you'd want more robust parsing
	usage := parseDiskUsage(result.Stdout)
	
	status := "ok"
	message := "Disk usage is normal"
	
	if usage > 80 {
		status = "critical"
		message = fmt.Sprintf("Disk usage is critical: %d%%", usage)
	} else if usage > 70 {
		status = "warning"
		message = fmt.Sprintf("Disk usage is high: %d%%", usage)
	}

	if verbose {
		message += fmt.Sprintf(" (Output: %s)", result.Stdout)
	}

	return CheckResult{
		Status:  status,
		Message: message,
		Value:   usage,
		Unit:    "%",
	}
}

// checkMemoryUsage checks memory usage
func checkMemoryUsage(conn *ssh.Connection, verbose bool, logger *logger.Logger) CheckResult {
	// Execute free command
	result, err := conn.ExecuteCommand(context.Background(), "free -h", "")
	if err != nil {
		return CheckResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to check memory usage: %v", err),
		}
	}

	if result.ExitCode != 0 {
		return CheckResult{
			Status:  "error",
			Message: fmt.Sprintf("Memory check failed with exit code %d", result.ExitCode),
			Stderr: result.Stderr,
		}
	}

	// Parse free output (simplified)
	usage := parseMemoryUsage(result.Stdout)
	
	status := "ok"
	message := "Memory usage is normal"
	
	if usage > 85 {
		status = "critical"
		message = fmt.Sprintf("Memory usage is critical: %d%%", usage)
	} else if usage > 75 {
		status = "warning"
		message = fmt.Sprintf("Memory usage is high: %d%%", usage)
	}

	if verbose {
		message += fmt.Sprintf(" (Output: %s)", result.Stdout)
	}

	return CheckResult{
		Status:  status,
		Message: message,
		Value:   usage,
		Unit:    "%",
	}
}

// checkCPUUsage checks CPU usage
func checkCPUUsage(conn *ssh.Connection, verbose bool, logger *logger.Logger) CheckResult {
	// Execute uptime command for load average
	result, err := conn.ExecuteCommand(context.Background(), "uptime", "")
	if err != nil {
		return CheckResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to check CPU usage: %v", err),
		}
	}

	if result.ExitCode != 0 {
		return CheckResult{
			Status:  "error",
			Message: fmt.Sprintf("CPU check failed with exit code %d", result.ExitCode),
			Stderr: result.Stderr,
		}
	}

	// Parse uptime output (simplified)
	loadAvg := parseLoadAverage(result.Stdout)
	
	status := "ok"
	message := "CPU load is normal"
	
	if loadAvg > 2.0 {
		status = "critical"
		message = fmt.Sprintf("CPU load is critical: %.2f", loadAvg)
	} else if loadAvg > 1.5 {
		status = "warning"
		message = fmt.Sprintf("CPU load is high: %.2f", loadAvg)
	}

	if verbose {
		message += fmt.Sprintf(" (Output: %s)", result.Stdout)
	}

	return CheckResult{
		Status:  status,
		Message: message,
		Value:   loadAvg,
		Unit:    "",
	}
}

// checkServices checks service status
func checkServices(conn *ssh.Connection, verbose bool, logger *logger.Logger) CheckResult {
	// Execute systemctl command
	result, err := conn.ExecuteCommand(context.Background(), "systemctl list-units --type=service --state=running", "")
	if err != nil {
		return CheckResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to check services: %v", err),
		}
	}

	if result.ExitCode != 0 {
		return CheckResult{
			Status:  "error",
			Message: fmt.Sprintf("Service check failed with exit code %d", result.ExitCode),
			Stderr: result.Stderr,
		}
	}

	// Count running services (simplified)
	serviceCount := countServices(result.Stdout)
	
	status := "ok"
	message := fmt.Sprintf("%d services running", serviceCount)
	
	if serviceCount == 0 {
		status = "critical"
		message = "No services running"
	}

	if verbose {
		message += fmt.Sprintf(" (Output: %s)", result.Stdout)
	}

	return CheckResult{
		Status:  status,
		Message: message,
		Value:   serviceCount,
		Unit:    "services",
	}
}

// Helper functions for parsing command output (simplified)
func parseDiskUsage(output string) float64 {
	// Very simplified parsing - in production, use proper parsing
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "/") && strings.Contains(line, "%") {
			// Extract percentage (simplified)
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasSuffix(part, "%") {
					var usage float64
					fmt.Sscanf(part, "%f", &usage)
					return usage
				}
			}
		}
	}
	return 0
}

func parseMemoryUsage(output string) float64 {
	// Very simplified parsing - in production, use proper parsing
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Mem:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.HasSuffix(part, "%") {
					var usage float64
					fmt.Sscanf(part, "%f", &usage)
					return usage
				}
			}
		}
	}
	return 0
}

func parseLoadAverage(output string) float64 {
	// Very simplified parsing - in production, use proper parsing
	parts := strings.Fields(output)
	for _, part := range parts {
		if strings.Contains(part, "load average:") {
			// Find the load average value
			for i := 1; i < len(parts); i++ {
				if strings.Contains(parts[i], ":") {
					var loadAvg float64
					fmt.Sscanf(strings.TrimSpace(parts[i+1]), "%f", &loadAvg)
					return loadAvg
				}
			}
		}
	}
	return 0
}

func countServices(output string) int {
	return strings.Count(output, ".service")
}

// outputHealthCheckTable outputs health check results in table format
func outputHealthCheckTable(results []*HealthCheckResult, logger *logger.Logger) error {
	if len(results) == 0 {
		logger.Info("No health check results to display")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tSTATUS\tDISK\tMEMORY\tCPU\tSERVICES\tDURATION\tISSUES")
	
	for _, result := range results {
		diskStatus := getCheckStatus(result.Checks, "disk")
		memoryStatus := getCheckStatus(result.Checks, "memory")
		cpuStatus := getCheckStatus(result.Checks, "cpu")
		servicesStatus := getCheckStatus(result.Checks, "services")
		
		issues := ""
		if len(result.Issues) > 0 {
			issues = strings.Join(result.Issues, "; ")
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			result.ServerID, result.OverallStatus, diskStatus, memoryStatus, cpuStatus, servicesStatus, result.Duration.String(), issues)
	}
	
	w.Flush()
	return nil
}

// outputHealthCheckJSON outputs health check results in JSON format
func outputHealthCheckJSON(results []*HealthCheckResult, logger *logger.Logger) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	fmt.Println(data)
	logger.WithField("count", len(results)).Info("Health check results exported to JSON")
	return nil
}

// displayHealthList displays health check results in a list format
func displayHealthList(checks []*server.HealthCheck, logger *logger.Logger) {
	if len(checks) == 0 {
		logger.Info("No health checks found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSERVER\tTIMESTAMP\tSTATUS\tDURATION")
	
	for _, check := range checks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			check.ID, check.ServerID, check.Timestamp.Format("2006-01-02 15:04:05"), check.Status, check.Duration.String())
	}
	
	w.Flush()
}

// displayLiveHealthCheck displays live health check results
func displayLiveHealthCheck(serverID string, result *HealthCheckResult, logger *logger.Logger) {
	fmt.Printf("\r=== Health Check for %s ===\n", serverID)
	fmt.Printf("Status: %s\n", result.OverallStatus)
	fmt.Printf("Time: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %s\n", result.Duration.String())
	
	for checkName, checkResult := range result.Checks {
		fmt.Printf("%s: %s - %s\n", checkName, checkResult.Status, checkResult.Message)
	}
	
	fmt.Printf("Press Ctrl+C to stop monitoring...\n")
}

// getCheckStatus gets the status of a specific check
func getCheckStatus(checks map[string]CheckResult, checkName string) string {
	if checkResult, exists := checks[checkName]; exists {
		return checkResult.Status
	}
	return "unknown"
}

// generateHealthCheckID generates a unique health check ID
func generateHealthCheckID(serverID string) string {
	return fmt.Sprintf("health-%s-%d", serverID, time.Now().Unix())
}