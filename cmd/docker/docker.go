package docker

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

// Command represents the docker command
type Command struct {
	configManager *config.ConfigManager
	store        store.Store
	logger       *logger.Logger
}

// NewCommand creates a new docker command
func NewCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Docker container management",
		Long:  "Manage Docker containers, images, volumes, and networks",
	}

	// Add subcommands
	cmd.AddCommand(newListCommand(configManager, store, logger))
	cmd.AddCommand(newHealthCommand(configManager, store, logger))
	cmd.AddCommand(newCleanupCommand(configManager, store, logger))
	cmd.AddCommand(newBackupCommand(configManager, store, logger))
	cmd.AddCommand(newRestoreCommand(configManager, store, logger))
	cmd.AddCommand(newLogsCommand(configManager, store, logger))
	cmd.AddCommand(newStatsCommand(configManager, store, logger))

	return cmd
}

// newListCommand creates the list subcommand
func newListCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Docker containers",
		Long:  "List all Docker containers with their status and information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerList(configManager, store, logger, cmd, args)
		},
	}
}

// newHealthCommand creates the health subcommand
func newHealthCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check container health",
		Long:  "Perform health checks on Docker containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerHealth(configManager, store, logger, cmd, args)
		},
	}
}

// newCleanupCommand creates the cleanup subcommand
func newCleanupCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up Docker resources",
		Long:  "Remove unused Docker images, containers, volumes, and networks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerCleanup(configManager, store, logger, cmd, args)
		},
	}
}

// newBackupCommand creates the backup subcommand
func newBackupCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "backup [container-id]",
		Short: "Backup Docker containers",
		Long:  "Create backups of Docker containers and volumes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerBackup(configManager, store, logger, cmd, args)
		},
	}
}

// newRestoreCommand creates the restore subcommand
func newRestoreCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "restore [backup-file]",
		Short: "Restore Docker containers",
		Long:  "Restore Docker containers from backup files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerRestore(configManager, store, logger, cmd, args)
		},
	}
}

// newLogsCommand creates the logs subcommand
func newLogsCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "logs [container-id]",
		Short: "Show container logs",
		Long:  "Display logs from Docker containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerLogs(configManager, store, logger, cmd, args)
		},
	}
}

// newStatsCommand creates the stats subcommand
func newStatsCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show Docker statistics",
		Long:  "Display Docker system statistics and resource usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerStats(configManager, store, logger, cmd, args)
		},
	}
}

// runDockerList executes the docker list command
func runDockerList(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	all, _ := cmd.Flags().GetBool("all")
	format, _ := cmd.Flags().GetString("format")

	// Get servers
	servers, err := getServers(store, cmd)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	// Get Docker containers for each server
	var allContainers []*DockerContainer
	for _, srv := range servers {
		containers, err := getDockerContainers(srv, logger)
		if err != nil {
			logger.WithError(err).Warnf("Failed to get containers for server %s", srv.ID)
			continue
		}
		allContainers = append(allContainers, containers...)
	}

	// Output results
	switch format {
	case "json":
		return outputDockerContainersJSON(allContainers, logger)
	case "table":
		return outputDockerContainersTable(allContainers, logger)
	default:
		return outputDockerContainersTable(allContainers, logger)
	}
}

// runDockerHealth executes the docker health command
func runDockerHealth(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	restart, _ := cmd.Flags().GetBool("restart")

	// Get servers
	servers, err := getServers(store, cmd)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	// Check health for each server
	var results []*HealthCheckResult
	for _, srv := range servers {
		result := checkDockerHealth(srv, restart, logger)
		results = append(results, result)
	}

	// Output results
	outputHealthCheckResults(results, logger)
	return nil
}

// runDockerCleanup executes the docker cleanup command
func runDockerCleanup(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	aggressive, _ := cmd.Flags().GetBool("aggressive")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Get servers
	servers, err := getServers(store, cmd)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	// Perform cleanup on each server
	var results []*CleanupResult
	for _, srv := range servers {
		result := performDockerCleanup(srv, aggressive, dryRun, logger)
		results = append(results, result)
	}

	// Output results
	outputCleanupResults(results, logger)
	return nil
}

// runDockerBackup executes the docker backup command
func runDockerBackup(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	backupDir, _ := cmd.Flags().GetString("backup-dir")
	compress, _ := cmd.Flags().GetBool("compress")

	// Get servers
	servers, err := getServers(store, cmd)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	// Perform backup for each server
	var results []*BackupResult
	for _, srv := range servers {
		result := performDockerBackup(srv, backupDir, compress, logger)
		results = append(results, result)
	}

	// Output results
	outputBackupResults(results, logger)
	return nil
}

// runDockerRestore executes the docker restore command
func runDockerRestore(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	backupFile, _ := cmd.Flags().GetString("backup-file")

	// Get servers
	servers, err := getServers(store, cmd)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	// Perform restore for each server
	var results []*RestoreResult
	for _, srv := range servers {
		result := performDockerRestore(srv, backupFile, logger)
		results = append(results, result)
	}

	// Output results
	outputRestoreResults(results, logger)
	return nil
}

// runDockerLogs executes the docker logs command
func runDockerLogs(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	tail, _ := cmd.Flags().GetBool("tail")
	lines, _ := cmd.Flags().GetInt("lines")

	// Get servers
	servers, err := getServers(store, cmd)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	// Get logs for each server
	var results []*LogsResult
	for _, srv := range servers {
		result := getDockerLogs(srv, tail, lines, logger)
		results = append(results, result)
	}

	// Output results
	outputLogsResults(results, logger)
	return nil
}

// runDockerStats executes the docker stats command
func runDockerStats(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")

	// Get servers
	servers, err := getServers(store, cmd)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	// Get stats for each server
	var results []*StatsResult
	for _, srv := range servers {
		result := getDockerStats(srv, logger)
		results = append(results, result)
	}

	// Output results
	outputStatsResults(results, logger)
	return nil
}

// Helper functions

// getServers gets servers based on flags
func getServers(store store.Store, cmd *cobra.Command) ([]*server.Server, error) {
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
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

// Docker container structure
type DockerContainer struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Status     string            `json:"status"`
	State      string            `json:"state"`
	Created    time.Time         `json:"created"`
	Started    time.Time         `json:"started"`
	Ports      []PortInfo       `json:"ports"`
	Labels     map[string]string  `json:"labels"`
	ServerID   string            `json:"server_id"`
}

// PortInfo represents port information
type PortInfo struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol     string  `json:"protocol"`
	Type         string  `json:"type"`
}

// Health check result
type HealthCheckResult struct {
	ServerID   string    `json:"server_id"`
	ContainerID string  `json:"container_id"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
}

// Cleanup result
type CleanupResult struct {
	ServerID   string    `json:"server_id"`
	ImagesRemoved int       `json:"images_removed"`
	ContainersRemoved int    `json:"containers_removed"`
	VolumesRemoved int       `json:"volumes_removed"`
	NetworksRemoved int       `json:"networks_removed"`
	SpaceFreed   int64     `json:"space_freed"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
}

// Backup result
type BackupResult struct {
	ServerID   string    `json:"server_id"`
	BackupFile string    `json:"backup_file"`
	Size       int64     `json:"size"`
	Compressed bool       `json:"compressed"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
}

// Restore result
type RestoreResult struct {
	ServerID   string    `json:"server_id"`
	BackupFile string    `json:"backup_file"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
}

// Logs result
type LogsResult struct {
	ServerID   string    `json:"server_id"`
	ContainerID string  `json:"container_id"`
	Logs       []string  `json:"logs"`
	Tail       bool    `json:"tail"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
}

// Stats result
type StatsResult struct {
	ServerID   string    `json:"server_id"`
	Containers int    `json:"containers"`
	Images     int     `json:"images"`
	Volumes    int     `json:"volumes"`
	SpaceUsed  int64   `json:"space_used"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
}

// Output functions

func outputDockerContainersJSON(containers []*DockerContainer, logger *logger.Logger) error {
	data, err := json.MarshalIndent(containers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal containers: %w", err)
	}

	fmt.Println(data)
	logger.WithField("count", len(containers)).Info("Docker containers exported to JSON")
	return nil
}

func outputDockerContainersTable(containers []*DockerContainer, logger *logger.Logger) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tIMAGE\tSTATUS\tSTATE\tCREATED\tPORTS\tSERVER")
	
	for _, container := range containers {
		portsStr := ""
		for _, port := range container.Ports {
			portsStr += fmt.Sprintf("%d->%d ", port.ContainerPort, port.HostPort)
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			container.ID, container.Name, container.Image, container.Status, container.State,
			container.Created.Format("2006-01-02 15:04:05"), portsStr, container.ServerID)
	}
	
	w.Flush()
	logger.WithField("count", len(containers)).Info("Docker containers displayed in table format")
	return nil
}

func outputHealthCheckResults(results []*HealthCheckResult, logger *logger.Logger) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tCONTAINER\tSTATUS\tMESSAGE\tTIMESTAMP")
	
	for _, result := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			result.ServerID, result.ContainerID, result.Status, result.Message, result.Timestamp.Format("2006-01-02 15:04:05"))
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Docker health check results displayed")
	return nil
}

func outputCleanupResults(results []*CleanupResult, logger *logger.Logger) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tIMAGES\tCONTAINERS\tVOLUMES\tNETWORKS\tSPACE\tMESSAGE")
	
	for _, result := range results {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n",
			result.ServerID, result.ImagesRemoved, result.ContainersRemoved, result.VolumesRemoved, result.NetworksRemoved, result.SpaceFreed, result.Message)
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Docker cleanup results displayed")
	return nil
}

func outputBackupResults(results []*BackupResult, logger *logger.Logger) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tBACKUP FILE\tSIZE\tCOMPRESSED\tMESSAGE")
	
	for _, result := range results {
		fmt.Fprintf(w, "%s\t%s\t%d\t%t\t%s\n",
			result.ServerID, result.BackupFile, result.Size, result.Compressed, result.Message)
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Docker backup results displayed")
	return nil
}

func outputRestoreResults(results []*RestoreResult, logger *logger.Logger) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tBACKUP FILE\tMESSAGE")
	
	for _, result := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			result.ServerID, result.BackupFile, result.Message)
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Docker restore results displayed")
	return nil
}

func outputLogsResults(results []*LogsResult, logger *logger.Logger) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tCONTAINER\tLOGS\tTAIL\tMESSAGE")
	
	for _, result := range results {
		logsStr := ""
		if len(result.Logs) > 0 {
			for i, log := range result.Logs {
				if i > 0 {
					logsStr += ", "
				}
				logsStr += log
			}
		}
		
		fmt.Fprintf(w, "%s\t%s\t%t\t%s\n",
			result.ServerID, result.ContainerID, logsStr, result.Tail, result.Message)
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Docker logs results displayed")
	return nil
}

func outputStatsResults(results []*StatsResult, logger *logger.Logger) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tCONTAINERS\tIMAGES\tVOLUMES\tSPACE USED\tMESSAGE")
	
	for _, result := range results {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%d\t%s\n",
			result.ServerID, result.Containers, result.Images, result.Volumes, result.SpaceUsed, result.Message)
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Docker stats results displayed")
	return nil
}

// Placeholder functions for Docker operations (would require Docker client library)
func getDockerContainers(srv *server.Server, logger *logger.Logger) ([]*DockerContainer, error) {
	// This would require actual Docker client integration
	// For now, return empty slice
	logger.WithField("server", srv.ID).Warn("Docker container listing not yet implemented")
	return []*DockerContainer{}, nil
}

func checkDockerHealth(srv *server.Server, restart bool, logger *logger.Logger) *HealthCheckResult {
	// This would require actual Docker client integration
	// For now, return a placeholder result
	return &HealthCheckResult{
		ServerID:   srv.ID,
		ContainerID: "unknown",
		Status:     "unknown",
		Message:    "Docker health check not yet implemented",
		Timestamp:  time.Now(),
	}
}

func performDockerCleanup(srv *server.Server, aggressive, dryRun bool, logger *logger.Logger) *CleanupResult {
	// This would require actual Docker client integration
	// For now, return a placeholder result
	return &CleanupResult{
		ServerID:   srv.ID,
		ImagesRemoved: 0,
		ContainersRemoved: 0,
		VolumesRemoved: 0,
		NetworksRemoved: 0,
		SpaceFreed:   0,
		Message:    "Docker cleanup not yet implemented",
		Timestamp:  time.Now(),
	}
}

func performDockerBackup(srv *server.Server, backupDir string, compress bool, logger *logger.Logger) *BackupResult {
	// This would require actual Docker client integration
	// For now, return a placeholder result
	return &BackupResult{
		ServerID:   srv.ID,
		BackupFile:  "",
		Size:       0,
		Compressed: false,
		Message:    "Docker backup not yet implemented",
		Timestamp:  time.Now(),
	}
}

func performDockerRestore(srv *server.Server, backupFile string, logger *logger.Logger) *RestoreResult {
	// This would require actual Docker client integration
	// For now, return a placeholder result
	return &RestoreResult{
		ServerID:   srv.ID,
		BackupFile:  backupFile,
		Message:    "Docker restore not yet implemented",
		Timestamp:  time.Now(),
	}
}

func getDockerLogs(srv *server.Server, tail bool, lines int, logger *logger.Logger) *LogsResult {
	// This would require actual Docker client integration
	// For now, return a placeholder result
	return &LogsResult{
		ServerID:   srv.ID,
		ContainerID: "unknown",
		Logs:       []string{"Docker logs not yet implemented"},
		Tail:       tail,
		Message:    "Docker logs not yet implemented",
		Timestamp:  time.Now(),
	}
}

func getDockerStats(srv *server.Server, logger *logger.Logger) *StatsResult {
	// This would require actual Docker client integration
	// For now, return a placeholder result
	return &StatsResult{
		ServerID:   srv.ID,
		Containers: 0,
		Images:     0,
		Volumes:    0,
		SpaceUsed:  0,
		Message:    "Docker stats not yet implemented",
		Timestamp:  time.Now(),
	}
}