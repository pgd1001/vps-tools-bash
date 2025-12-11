package run

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/logger"
	"github.com/pgd1001/vps-tools/internal/server"
	"github.com/pgd1001/vps-tools/internal/ssh"
	"github.com/pgd1001/vps-tools/internal/store"
)

// Command represents the run command
type Command struct {
	configManager *config.ConfigManager
	store        store.Store
	logger       *logger.Logger
}

// NewCommand creates a new run command
func NewCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute commands on servers",
		Long:  "Execute commands on one or more servers with various options",
	}

	// Add subcommands
	cmd.AddCommand(newExecuteCommand(configManager, store, logger))
	cmd.AddCommand(newBatchCommand(configManager, store, logger))
	cmd.AddCommand(newScheduleCommand(configManager, store, logger))

	return cmd
}

// newExecuteCommand creates the execute subcommand
func newExecuteCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute [command]",
		Short: "Execute a command on servers",
		Long:  "Execute a specific command on target servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExecute(configManager, store, logger, cmd, args)
		},
	}

	// Add flags
	cmd.Flags().StringSliceP("servers", "s", []string{}, "Server IDs to run command on")
	cmd.Flags().StringP("command", "c", "", "Command to execute")
	cmd.Flags().StringP("working-dir", "d", "", "Working directory for command execution")
	cmd.Flags().StringP("user", "u", "", "User to run command as")
	cmd.Flags().IntP("timeout", "t", 300, "Command timeout in seconds")
	cmd.Flags().BoolP("parallel", "p", false, "Run commands in parallel")
	cmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
	cmd.Flags().BoolP("dry-run", "n", false, "Show what would be executed without running")

	return cmd
}

// newBatchCommand creates the batch subcommand
func newBatchCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch [file]",
		Short: "Execute commands from a batch file",
		Long:  "Execute multiple commands from a batch file on target servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatch(configManager, store, logger, cmd, args)
		},
	}

	// Add flags
	cmd.Flags().StringP("file", "f", "", "Batch file containing commands")
	cmd.Flags().StringSliceP("servers", "s", []string{}, "Server IDs to run batch on")
	cmd.Flags().BoolP("parallel", "p", false, "Run commands in parallel")
	cmd.Flags().IntP("concurrency", "c", 5, "Maximum concurrent commands")
	cmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
	cmd.Flags().BoolP("dry-run", "n", false, "Show what would be executed without running")

	return cmd
}

// newScheduleCommand creates the schedule subcommand
func newScheduleCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule [command]",
		Short: "Schedule a command for later execution",
		Long:  "Schedule a command to run at a specified time on target servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchedule(configManager, store, logger, cmd, args)
		},
	}

	// Add flags
	cmd.Flags().StringP("command", "c", "", "Command to schedule")
	cmd.Flags().StringSliceP("servers", "s", []string{}, "Server IDs to run command on")
	cmd.Flags().StringP("schedule", "s", "", "Schedule time (cron format)")
	cmd.Flags().StringP("name", "n", "", "Job name for identification")
	cmd.Flags().StringP("description", "d", "", "Job description")

	return cmd
}

// runExecute executes the execute command
func runExecute(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	command, _ := cmd.Flags().GetString("command")
	workingDir, _ := cmd.Flags().GetString("working-dir")
	user, _ := cmd.Flags().GetString("user")
	timeout, _ := cmd.Flags().GetInt("timeout")
	parallel, _ := cmd.Flags().GetBool("parallel")
	outputJSON, _ := cmd.Flags().GetBool("json")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if command == "" {
		return fmt.Errorf("command is required")
	}

	// Get servers to run on
	servers, err := getServers(store, serverIDs)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no servers found")
	}

	logger.WithFields(map[string]interface{}{
		"command":     command,
		"servers":     len(servers),
		"parallel":    parallel,
		"dry_run":    dryRun,
	}).Info("Starting command execution")

	// Execute command on servers
	results := executeCommandOnServers(servers, command, workingDir, user, timeout, parallel, dryRun, logger)

	// Output results
	if outputJSON {
		return outputResultsJSON(results, logger)
	}

	return outputResultsTable(results, logger)
}

// runBatch executes the batch command
func runBatch(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	batchFile, _ := cmd.Flags().GetString("file")
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	parallel, _ := cmd.Flags().GetBool("parallel")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	outputJSON, _ := cmd.Flags().GetBool("json")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if batchFile == "" {
		return fmt.Errorf("batch file is required")
	}

	// Parse batch file
	commands, err := parseBatchFile(batchFile)
	if err != nil {
		return fmt.Errorf("failed to parse batch file: %w", err)
	}

	// Get servers to run on
	servers, err := getServers(store, serverIDs)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no servers found")
	}

	logger.WithFields(map[string]interface{}{
		"batch_file": batchFile,
		"commands":   len(commands),
		"servers":    len(servers),
		"parallel":   parallel,
		"concurrency": concurrency,
		"dry_run":   dryRun,
	}).Info("Starting batch execution")

	// Execute batch commands
	results := executeBatchCommands(servers, commands, parallel, dryRun, logger)

	// Output results
	if outputJSON {
		return outputResultsJSON(results, logger)
	}

	return outputResultsTable(results, logger)
}

// runSchedule executes the schedule command
func runSchedule(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	command, _ := cmd.Flags().GetString("command")
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	schedule, _ := cmd.Flags().GetString("schedule")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")

	if command == "" {
		return fmt.Errorf("command is required")
	}
	if schedule == "" {
		return fmt.Errorf("schedule is required")
	}

	// Get servers to run on
	servers, err := getServers(store, serverIDs)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no servers found")
	}

	// Create scheduled job
	job := &server.Job{
		ID:       generateJobID(),
		ServerID: servers[0].ID, // Use first server for now
		Command:  command,
		Status:   server.JobStatusPending,
		CreatedAt: time.Now(),
	}

	// Store job
	if err := store.CreateJob(job); err != nil {
		return fmt.Errorf("failed to create scheduled job: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"job_id":      job.ID,
		"command":     command,
		"schedule":    schedule,
		"servers":     len(servers),
		"name":        name,
		"description": description,
	}).Info("Command scheduled successfully")

	fmt.Printf("Command scheduled with ID: %s\n", job.ID)
	return nil
}

// getServers gets servers based on IDs or all servers
func getServers(store store.Store, serverIDs []string) ([]*server.Server, error) {
	if len(serverIDs) == 0 {
		// Get all servers
		return store.ListServers(nil)
	}

	// Get specific servers
	var servers []*server.Server
	for _, id := range serverIDs {
		srv, err := store.GetServer(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get server %s: %w", id, err)
		}
		servers = append(servers, srv)
	}

	return servers, nil
}

// executeCommandOnServers executes a command on multiple servers
func executeCommandOnServers(servers []*server.Server, command, workingDir, user string, timeout int, parallel, dryRun bool, logger *logger.Logger) []*ExecutionResult {
	var results []*ExecutionResult

	if parallel {
		// Execute in parallel
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, srv := range servers {
			wg.Add(1)
			go func(srv *server.Server) {
				defer wg.Done()
				result := executeCommandOnServer(srv, command, workingDir, user, timeout, dryRun, logger)
				
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}(srv)
		}

		wg.Wait()
	} else {
		// Execute sequentially
		for _, srv := range servers {
			result := executeCommandOnServer(srv, command, workingDir, user, timeout, dryRun, logger)
			results = append(results, result)
		}
	}

	return results
}

// executeCommandOnServer executes a command on a single server
func executeCommandOnServer(srv *server.Server, command, workingDir, user string, timeout int, dryRun bool, logger *logger.Logger) *ExecutionResult {
	result := &ExecutionResult{
		ServerID: srv.ID,
		Server:   srv.Name,
		Command:  command,
		Status:   "pending",
		StartTime: time.Now(),
	}

	if dryRun {
		result.Status = "dry_run"
		result.Message = "Dry run - command not executed"
		return result
	}

	// Create SSH client
	sshConfig := &ssh.Config{
		Host:       srv.Host,
		Port:       srv.Port,
		User:       user,
		AuthMethod: srv.AuthMethod,
		Timeout:    time.Duration(timeout) * time.Second,
	}

	sshClient, err := ssh.NewClient(sshConfig, logger)
	if err != nil {
		result.Status = "error"
		result.Message = fmt.Sprintf("Failed to create SSH client: %v", err)
		result.Error = err
		return result
	}
	defer sshClient.Close()

	// Connect and execute command
	ctx := context.Background()
	cmdResult, err := sshClient.ExecuteCommand(ctx, command, workingDir)
	if err != nil {
		result.Status = "error"
		result.Message = fmt.Sprintf("Command execution failed: %v", err)
		result.Error = err
		return result
	}

	result.Status = "success"
	result.Message = "Command executed successfully"
	result.ExitCode = cmdResult.ExitCode
	result.Stdout = cmdResult.Stdout
	result.Stderr = cmdResult.Stderr
	result.Duration = cmdResult.Duration
	result.EndTime = time.Now()

	logger.WithFields(map[string]interface{}{
		"server_id":   srv.ID,
		"command":     command,
		"exit_code":   result.ExitCode,
		"duration":    result.Duration.String(),
	}).Info("Command executed successfully")

	return result
}

// executeBatchCommands executes multiple commands on servers
func executeBatchCommands(servers []*server.Server, commands []string, parallel, dryRun bool, logger *logger.Logger) []*ExecutionResult {
	var results []*ExecutionResult

	for _, command := range commands {
		serverResults := executeCommandOnServers(servers, command, "", "", 0, dryRun, logger)
		results = append(results, serverResults...)
	}

	return results
}

// ExecutionResult represents the result of a command execution
type ExecutionResult struct {
	ServerID   string        `json:"server_id"`
	Server     string        `json:"server"`
	Command    string        `json:"command"`
	Status     string        `json:"status"`
	Message    string        `json:"message"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	ExitCode   int           `json:"exit_code"`
	Stdout     string        `json:"stdout"`
	Stderr     string        `json:"stderr"`
	Error      error         `json:"error,omitempty"`
}

// parseBatchFile parses a batch file containing commands
func parseBatchFile(filename string) ([]string, error) {
	// Simplified batch file parser
	// In production, you'd want more robust parsing
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open batch file: %w", err)
	}
	defer file.Close()

	var commands []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" && !strings.HasPrefix(line, "#") {
			commands = append(commands, strings.TrimSpace(line))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading batch file: %w", err)
	}

	return commands, nil
}

// outputResultsJSON outputs results in JSON format
func outputResultsJSON(results []*ExecutionResult, logger *logger.Logger) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	fmt.Println(data)
	logger.WithField("count", len(results)).Info("Results exported to JSON")
	return nil
}

// outputResultsTable outputs results in table format
func outputResultsTable(results []*ExecutionResult, logger *logger.Logger) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tCOMMAND\tSTATUS\tEXIT\tDURATION\tMESSAGE")
	
	for _, result := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			result.Server, result.Command, result.Status, result.ExitCode, result.Duration.String(), result.Message)
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Results displayed in table format")
	return nil
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job-%d", time.Now().Unix())
}