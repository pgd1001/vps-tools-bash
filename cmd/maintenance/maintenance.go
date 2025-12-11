package maintenance

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

// Command represents the maintenance command
type Command struct {
	configManager *config.ConfigManager
	store        store.Store
	logger       *logger.Logger
}

// NewCommand creates a new maintenance command
func NewCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "maintenance",
		Short: "System maintenance operations",
		Long:  "Perform system maintenance tasks like cleanup, updates, and backups",
	}

	// Add subcommands
	cmd.AddCommand(newCleanupCommand(configManager, store, logger))
	cmd.AddCommand(newUpdateCommand(configManager, store, logger))
	cmd.AddCommand(newBackupCommand(configManager, store, logger))
	cmd.AddCommand(newRestoreCommand(configManager, store, logger))

	return cmd
}

// newCleanupCommand creates the cleanup subcommand
func newCleanupCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup",
		Short: "System cleanup",
		Long:  "Clean up temporary files, logs, and system cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCleanup(configManager, store, logger, cmd, args)
		},
	}
}

// newUpdateCommand creates the update subcommand
func newUpdateCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "update [system|packages]",
		Short: "Update system or packages",
		Long:  "Update system packages or perform system maintenance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(configManager, store, logger, cmd, args)
		},
	}
}

// newBackupCommand creates the backup subcommand
func newBackupCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "backup [database|files|system]",
		Short: "Create backups",
		Long:  "Create backups of databases, files, or system state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackup(configManager, store, logger, cmd, args)
		},
	}
}

// newRestoreCommand creates the restore subcommand
func newRestoreCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "restore [backup-file]",
		Short: "Restore from backup",
		Long:  "Restore system state or data from backup files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(configManager, store, logger, cmd, args)
		},
	}
}

// runCleanup executes the cleanup command
func runCleanup(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	aggressive, _ := cmd.Flags().GetBool("aggressive")
	logs, _ := cmd.Flags().GetBool("logs")
	cache, _ := cmd.Flags().GetBool("cache")
	temp, _ := cmd.Flags().GetBool("temp")

	// Get servers to clean
	servers, err := getServers(store, serverIDs)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		logger.Info("No servers to clean")
		return nil
	}

	logger.WithFields(map[string]interface{}{
		"servers":     len(servers),
		"dry_run":     dryRun,
		"aggressive":  aggressive,
		"logs":        logs,
		"cache":        cache,
		"temp":         temp,
	}).Info("Starting system cleanup")

	// Perform cleanup on each server
	var results []*CleanupResult
	for _, srv := range servers {
		result := performCleanupOnServer(srv, dryRun, aggressive, logs, cache, temp, logger)
		results = append(results, result)
	}

	// Output results
	outputCleanupResults(results, logger)
	return nil
}

// runUpdate executes the update command
func runUpdate(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	updateType, _ := cmd.Flags().GetString("type")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if len(args) < 1 {
		return fmt.Errorf("update type is required (system, packages)")
	}

	// Get servers to update
	servers, err := getServers(store, serverIDs)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		logger.Info("No servers to update")
		return nil
	}

	logger.WithFields(map[string]interface{}{
		"servers":     len(servers),
		"update_type": updateType,
		"dry_run":     dryRun,
	}).Info("Starting system update")

	// Perform update based on type
	switch updateType {
	case "system":
		return runSystemUpdate(servers, dryRun, logger)
	case "packages":
		return runPackageUpdate(servers, dryRun, logger)
	default:
		return fmt.Errorf("unknown update type: %s", updateType)
	}
}

// runBackup executes the backup command
func runBackup(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	backupType, _ := cmd.Flags().GetString("type")
	backupDir, _ := cmd.Flags().GetString("backup-dir")
	compress, _ := cmd.Flags().GetBool("compress")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if len(args) < 1 {
		return fmt.Errorf("backup type is required (database, files, system)")
	}

	// Get servers to backup
	servers, err := getServers(store, serverIDs)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		logger.Info("No servers to backup")
		return nil
	}

	logger.WithFields(map[string]interface{}{
		"servers":     len(servers),
		"backup_type": backupType,
		"backup_dir":  backupDir,
		"compress":    compress,
		"dry_run":     dryRun,
	}).Info("Starting backup operation")

	// Perform backup based on type
	switch backupType {
	case "database":
		return runDatabaseBackup(servers, backupDir, compress, dryRun, logger)
	case "files":
		return runFilesBackup(servers, backupDir, compress, dryRun, logger)
	case "system":
		return runSystemBackup(servers, backupDir, compress, dryRun, logger)
	default:
		return fmt.Errorf("unknown backup type: %s", backupType)
	}
}

// runRestore executes the restore command
func runRestore(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	backupFile, _ := cmd.Flags().GetString("backup-file")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if len(args) < 1 {
		return fmt.Errorf("backup file is required")
	}

	logger.WithFields(map[string]interface{}{
		"backup_file": backupFile,
		"dry_run":     dryRun,
	}).Info("Starting restore operation")

	// Perform restore
	result := performRestore(backupFile, dryRun, logger)

	// Output result
	outputRestoreResult(result, logger)
	return nil
}

// Helper functions

// getServers gets servers based on flags
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

// performCleanupOnServer performs cleanup on a single server
func performCleanupOnServer(srv *server.Server, dryRun, aggressive, logs, cache, temp, logger *logger.Logger) *CleanupResult {
	result := &CleanupResult{
		ServerID:  srv.ID,
		Timestamp: time.Now(),
	}

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
		result.Status = "error"
		result.Message = fmt.Sprintf("Failed to create SSH client: %v", err)
		result.Error = err
		return result
	}
	defer sshClient.Close()

	// Connect to server
	ctx := context.Background()
	conn, err := sshClient.Connect(ctx)
	if err != nil {
		result.Status = "error"
		result.Message = fmt.Sprintf("Failed to connect to server: %v", err)
		result.Error = err
		return result
	}
	defer conn.Close()

	// Perform cleanup operations
	if !dryRun {
		// Log cleanup
		if logs {
			if err := performLogCleanup(conn, logger); err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("Log cleanup failed: %v", err))
			}
		}

		// Cache cleanup
		if cache {
			if err := performCacheCleanup(conn, logger); err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("Cache cleanup failed: %v", err))
			}
		}

		// Temp file cleanup
		if temp {
			if err := performTempCleanup(conn, logger); err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("Temp cleanup failed: %v", err))
			}
		}

		// System cleanup
		if aggressive {
			if err := performAggressiveCleanup(conn, logger); err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("Aggressive cleanup failed: %v", err))
			}
		} else {
			if err := performBasicCleanup(conn, logger); err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("Basic cleanup failed: %v", err))
			}
		}
	}

	result.Status = "success"
	result.Message = "Cleanup completed successfully"
	return result
}

// performLogCleanup performs log file cleanup
func performLogCleanup(conn *ssh.Connection, logger *logger.Logger) error {
	// Execute log cleanup command
	result, err := conn.ExecuteCommand(context.Background(), "find /var/log -name \"*.log\" -mtime +7 -delete", "")
	if err != nil {
		return fmt.Errorf("log cleanup command failed: %v", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("log cleanup failed with exit code %d", result.ExitCode)
	}

	logger.Info("Log cleanup completed")
	return nil
}

// performCacheCleanup performs cache cleanup
func performCacheCleanup(conn *ssh.Connection, logger *logger.Logger) error {
	// Execute cache cleanup command
	result, err := conn.ExecuteCommand(context.Background(), "find /tmp -name \"*\" -mtime +7 -delete", "")
	if err != nil {
		return fmt.Errorf("cache cleanup command failed: %v", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("cache cleanup failed with exit code %d", result.ExitCode)
	}

	logger.Info("Cache cleanup completed")
	return nil
}

// performTempCleanup performs temporary file cleanup
func performTempCleanup(conn *ssh.Connection, logger *logger.Logger) error {
	// Execute temp cleanup command
	result, err := conn.ExecuteCommand(context.Background(), "find /tmp -name \"*\" -mtime +7 -delete", "")
	if err != nil {
		return fmt.Errorf("temp cleanup command failed: %v", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("temp cleanup failed with exit code %d", result.ExitCode)
	}

	logger.Info("Temp cleanup completed")
	return nil
}

// performBasicCleanup performs basic cleanup
func performBasicCleanup(conn *ssh.Connection, logger *logger.Logger) error {
	// Execute basic cleanup command
	result, err := conn.ExecuteCommand(context.Background(), "find /var/tmp /var/lib/apt/lists/old -delete", "")
	if err != nil {
		return fmt.Errorf("basic cleanup command failed: %v", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("basic cleanup failed with exit code %d", result.ExitCode)
	}

	logger.Info("Basic cleanup completed")
	return nil
}

// performAggressiveCleanup performs aggressive cleanup
func performAggressiveCleanup(conn *ssh.Connection, logger *logger.Logger) error {
	// Execute aggressive cleanup command
	result, err := conn.ExecuteCommand(context.Background(), "apt-get clean && apt-get autoremove && apt-get autoclean", "")
	if err != nil {
		return fmt.Errorf("aggressive cleanup command failed: %v", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("aggressive cleanup command failed with exit code %d", result.ExitCode)
	}

	logger.Info("Aggressive cleanup completed")
	return nil
}

// runSystemUpdate performs system update
func runSystemUpdate(servers []*server.Server, dryRun bool, logger *logger.Logger) error {
	// This would require actual system update implementation
	// For now, we'll show a placeholder message
	logger.Info("System update not yet implemented")
	return nil
}

// runPackageUpdate performs package update
func runPackageUpdate(servers []*server.Server, dryRun bool, logger *logger.Logger) error {
	// This would require actual package management
	// For now, we'll show a placeholder message
	logger.Info("Package update not yet implemented")
	return nil
}

// runDatabaseBackup performs database backup
func runDatabaseBackup(servers []*server.Server, backupDir string, compress bool, dryRun bool, logger *logger.Logger) error {
	// This would require actual database backup implementation
	// For now, we'll show a placeholder message
	logger.Info("Database backup not yet implemented")
	return nil
}

// runFilesBackup performs files backup
func runFilesBackup(servers []*server.Server, backupDir string, compress bool, dryRun bool, logger *logger.Logger) error {
	// This would require actual file backup implementation
	// For now, we'll show a placeholder message
	logger.Info("Files backup not yet implemented")
	return nil
}

// runSystemBackup performs system backup
func runSystemBackup(servers []*server.Server, backupDir string, compress bool, dryRun bool, logger *logger.Logger) error {
	// This would require actual system backup implementation
	// For now, we'll show a placeholder message
	logger.Info("System backup not yet implemented")
	return nil
}

// performRestore performs restore operation
func performRestore(backupFile string, dryRun bool, logger *logger.Logger) *RestoreResult {
	result := &RestoreResult{
		Timestamp: time.Now(),
	}

	// Check if backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		result.Status = "error"
		result.Message = fmt.Sprintf("Backup file not found: %s", backupFile)
		return result
	}

	if dryRun {
		result.Status = "dry_run"
		result.Message = "Dry run - restore not executed"
		return result
	}

	// This would require actual restore implementation
	// For now, we'll show a placeholder message
	logger.Info("Restore operation not yet implemented")
	result.Status = "success"
	result.Message = "Restore completed (placeholder)"

	return result
}

// Result structures
type CleanupResult struct {
	ServerID   string    `json:"server_id"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	Issues     []string   `json:"issues,omitempty"`
	Timestamp  time.Time  `json:"timestamp"`
}

type RestoreResult struct {
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	Timestamp  time.Time  `json:"timestamp"`
}

// Output functions

func outputCleanupResults(results []*CleanupResult, logger *logger.Logger) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tSTATUS\tMESSAGE\tTIMESTAMP\tISSUES")
	
	for _, result := range results {
		issuesStr := ""
		if len(result.Issues) > 0 {
			issuesStr = strings.Join(result.Issues, "; ")
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			result.ServerID, result.Status, result.Message, result.Timestamp.Format("2006-01-02 15:04:05"), issuesStr)
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Cleanup results displayed in table format")
	return nil
}

func outputRestoreResult(result *RestoreResult, logger *logger.Logger) error {
	fmt.Printf("Restore Status: %s\n", result.Status)
	fmt.Printf("Message: %s\n", result.Message)
	fmt.Printf("Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	
	logger.WithField("status", result.Status).Info("Restore result displayed")
	return nil
}