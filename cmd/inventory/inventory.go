package inventory

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/logger"
	"github.com/pgd1001/vps-tools/internal/server"
	"github.com/pgd1001/vps-tools/internal/store"
)

// Command represents the inventory command
type Command struct {
	configManager *config.ConfigManager
	store        store.Store
	logger       *logger.Logger
}

// NewCommand creates a new inventory command
func NewCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Manage server inventory",
		Long:  "Add, list, update, and delete servers from the inventory",
	}

	// Add subcommands
	cmd.AddCommand(newListCommand(configManager, store, logger))
	cmd.AddCommand(newAddCommand(configManager, store, logger))
	cmd.AddCommand(newUpdateCommand(configManager, store, logger))
	cmd.AddCommand(newDeleteCommand(configManager, store, logger))
	cmd.AddCommand(newShowCommand(configManager, store, logger))
	cmd.AddCommand(newImportCommand(configManager, store, logger))
	cmd.AddCommand(newExportCommand(configManager, store, logger))

	return cmd
}

// newListCommand creates the list command
func newListCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all servers",
		Long:  "List all servers in the inventory with optional filtering",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(configManager, store, logger, cmd, args)
		},
	}
}

// newAddCommand creates the add command
func newAddCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a new server",
		Long:  "Add a new server to the inventory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(configManager, store, logger, cmd, args)
		},
	}
}

// newUpdateCommand creates the update command
func newUpdateCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "update [id]",
		Short: "Update a server",
		Long:  "Update an existing server in the inventory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(configManager, store, logger, cmd, args)
		},
	}
}

// newDeleteCommand creates the delete command
func newDeleteCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a server",
		Long:  "Delete a server from the inventory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(configManager, store, logger, cmd, args)
		},
	}
}

// newShowCommand creates the show command
func newShowCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "show [id]",
		Short: "Show server details",
		Long:  "Show detailed information about a specific server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(configManager, store, logger, cmd, args)
		},
	}
}

// newImportCommand creates the import command
func newImportCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "import [file]",
		Short: "Import servers",
		Long:  "Import servers from a configuration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(configManager, store, logger, cmd, args)
		},
	}
}

// newExportCommand creates the export command
func newExportCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "export [format]",
		Short: "Export servers",
		Long:  "Export servers to various formats (yaml, json, csv)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(configManager, store, logger, cmd, args)
		},
	}
}

// runList executes the list command
func runList(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse filters
	filter := &server.ServerFilter{}
	
	// Check for tag filter
	if tags, err := cmd.Flags().GetStringSlice("tags"); err == nil && len(tags) > 0 {
		filter.Tags = tags
	}
	
	// Check for status filter
	if status, err := cmd.Flags().GetString("status"); err == nil && status != "" {
		filter.Status = status
	}
	
	// Check for search filter
	if search, err := cmd.Flags().GetString("search"); err == nil && search != "" {
		filter.Search = search
	}

	// Get servers
	servers, err := store.ListServers(filter)
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	// Display results
	if len(servers) == 0 {
		logger.Info("No servers found")
		return nil
	}

	displayServers(servers, logger)
	return nil
}

// runAdd executes the add command
func runAdd(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse server details
	name, _ := cmd.Flags().GetString("name")
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	user, _ := cmd.Flags().GetString("user")
	authType, _ := cmd.Flags().GetString("auth-type")
	keyPath, _ := cmd.Flags().GetString("key-path")
	tags, _ := cmd.Flags().GetStringSlice("tags")

	// Create server
	srv := &server.Server{
		ID:   generateServerID(host),
		Name:  name,
		Host:  host,
		Port:  port,
		User:  user,
		Tags:  tags,
		Status: server.StatusUnknown,
	}

	// Set authentication method
	switch authType {
	case "ssh-agent":
		srv.AuthMethod = server.AuthConfig{
			Type:    "ssh_agent",
			UseAgent: true,
		}
	case "private-key":
		srv.AuthMethod = server.AuthConfig{
			Type:    "private_key",
			KeyPath: keyPath,
		}
	case "password":
		password, _ := cmd.Flags().GetString("password")
		srv.AuthMethod = server.AuthConfig{
			Type:     "password",
			Password: password,
		}
	default:
		return fmt.Errorf("unsupported authentication type: %s", authType)
	}

	// Validate server
	if err := srv.Validate(); err != nil {
		return fmt.Errorf("invalid server configuration: %w", err)
	}

	// Save server
	if err := store.CreateServer(srv); err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"server_id": srv.ID,
		"name":      srv.Name,
		"host":      srv.Host,
	}).Info("Server added successfully")

	return nil
}

// runUpdate executes the update command
func runUpdate(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server ID is required")
	}

	serverID := args[0]

	// Get existing server
	srv, err := store.GetServer(serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Parse updates
	if name, err := cmd.Flags().GetString("name"); err == nil && name != "" {
		srv.Name = name
	}
	if host, err := cmd.Flags().GetString("host"); err == nil && host != "" {
		srv.Host = host
	}
	if port, err := cmd.Flags().GetInt("port"); err == nil && port > 0 {
		srv.Port = port
	}
	if user, err := cmd.Flags().GetString("user"); err == nil && user != "" {
		srv.User = user
	}
	if tags, err := cmd.Flags().GetStringSlice("tags"); err == nil && len(tags) > 0 {
		srv.Tags = tags
	}

	// Validate and save
	if err := srv.Validate(); err != nil {
		return fmt.Errorf("invalid server configuration: %w", err)
	}

	if err := store.UpdateServer(srv); err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"server_id": srv.ID,
		"name":      srv.Name,
	}).Info("Server updated successfully")

	return nil
}

// runDelete executes the delete command
func runDelete(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server ID is required")
	}

	serverID := args[0]

	// Confirm deletion
	confirm, _ := cmd.Flags().GetBool("confirm")
	if !confirm {
		fmt.Printf("Are you sure you want to delete server %s? [y/N]: ", serverID)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete server
	if err := store.DeleteServer(serverID); err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	logger.WithField("server_id", serverID).Info("Server deleted successfully")
	fmt.Printf("Server %s deleted successfully\n", serverID)

	return nil
}

// runShow executes the show command
func runShow(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server ID is required")
	}

	serverID := args[0]

	// Get server
	srv, err := store.GetServer(serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Display server details
	displayServerDetails(srv, logger)
	return nil
}

// runImport executes the import command
func runImport(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("import file is required")
	}

	filename := args[0]
	format, _ := cmd.Flags().GetString("format")

	logger.WithFields(map[string]interface{}{
		"filename": filename,
		"format":   format,
	}).Info("Starting server import")

	// Implementation would depend on the migration package
	// For now, we'll show a placeholder message
	fmt.Printf("Import from %s (format: %s) - not yet implemented\n", filename, format)
	return nil
}

// runExport executes the export command
func runExport(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("export format is required")
	}

	format := args[0]

	// Get all servers
	servers, err := store.ListServers(nil)
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	// Export based on format
	switch format {
	case "yaml":
		return exportServersYAML(servers, logger)
	case "json":
		return exportServersJSON(servers, logger)
	case "csv":
		return exportServersCSV(servers, logger)
	default:
		return fmt.Errorf("unsupported export format: %s (supported: yaml, json, csv)", format)
	}
}

// displayServers displays servers in a table format
func displayServers(servers []*server.Server, logger *logger.Logger) {
	if len(servers) == 0 {
		logger.Info("No servers to display")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tHOST\tPORT\tUSER\tSTATUS\tTAGS")
	
	for _, srv := range servers {
		tags := ""
		if len(srv.Tags) > 0 {
			tags = srv.Tags[0]
			for i, tag := range srv.Tags[1:] {
				tags += ", " + tag
			}
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			srv.ID, srv.Name, srv.Host, srv.Port, srv.User, srv.Status, tags)
	}
	
	w.Flush()
}

// displayServerDetails displays detailed information about a single server
func displayServerDetails(srv *server.Server, logger *logger.Logger) {
	fmt.Printf("Server Details:\n")
	fmt.Printf("  ID:       %s\n", srv.ID)
	fmt.Printf("  Name:     %s\n", srv.Name)
	fmt.Printf("  Host:     %s\n", srv.Host)
	fmt.Printf("  Port:     %d\n", srv.Port)
	fmt.Printf("  User:     %s\n", srv.User)
	fmt.Printf("  Status:   %s\n", srv.Status)
	fmt.Printf("  Tags:     %s\n", srv.Tags)
	fmt.Printf("  Created:  %s\n", srv.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Updated:  %s\n", srv.UpdatedAt.Format("2006-01-02 15:04:05"))
}

// exportServersYAML exports servers to YAML format
func exportServersYAML(servers []*server.Server, logger *logger.Logger) error {
	fmt.Println("# vps-tools Server Inventory")
	fmt.Println("# Generated by vps-tools inventory export")
	fmt.Println()
	
	for _, srv := range servers {
		fmt.Printf("- id: %s\n", srv.ID)
		fmt.Printf("  name: %s\n", srv.Name)
		fmt.Printf("  host: %s\n", srv.Host)
		fmt.Printf("  port: %d\n", srv.Port)
		fmt.Printf("  user: %s\n", srv.User)
		fmt.Printf("  tags: [%s]\n", strings.Join(srv.Tags, ", "))
		fmt.Printf("  status: %s\n", srv.Status)
		fmt.Printf("  created_at: %s\n", srv.CreatedAt.Format("2006-01-02T15:04:05Z"))
		fmt.Printf("  updated_at: %s\n", srv.UpdatedAt.Format("2006-01-02T15:04:05Z"))
		fmt.Println()
	}
	
	logger.WithField("count", len(servers)).Info("Servers exported to YAML")
	return nil
}

// exportServersJSON exports servers to JSON format
func exportServersJSON(servers []*server.Server, logger *logger.Logger) error {
	fmt.Println("[")
	for i, srv := range servers {
		if i > 0 {
			fmt.Println(",")
		}
		fmt.Printf("  {\n")
		fmt.Printf("    \"id\": \"%s\",\n", srv.ID)
		fmt.Printf("    \"name\": \"%s\",\n", srv.Name)
		fmt.Printf("    \"host\": \"%s\",\n", srv.Host)
		fmt.Printf("    \"port\": %d,\n", srv.Port)
		fmt.Printf("    \"user\": \"%s\",\n", srv.User)
		fmt.Printf("    \"tags\": [%s],\n", strings.Join(srv.Tags, "\", \""))
		fmt.Printf("    \"status\": \"%s\",\n", srv.Status)
		fmt.Printf("    \"created_at\": \"%s\",\n", srv.CreatedAt.Format("2006-01-02T15:04:05Z"))
		fmt.Printf("    \"updated_at\": \"%s\"\n", srv.UpdatedAt.Format("2006-01-02T15:04:05Z"))
		fmt.Printf("  }")
	}
	fmt.Println("]")
	
	logger.WithField("count", len(servers)).Info("Servers exported to JSON")
	return nil
}

// exportServersCSV exports servers to CSV format
func exportServersCSV(servers []*server.Server, logger *logger.Logger) error {
	fmt.Println("id,name,host,port,user,status,tags,created_at,updated_at")
	
	for _, srv := range servers {
		tags := strings.Join(srv.Tags, ";")
		fmt.Printf("%s,%s,%s,%d,%s,%s,%s,%s\n",
			srv.ID, srv.Name, srv.Host, srv.Port, srv.User, srv.Status, tags,
			srv.CreatedAt.Format("2006-01-02T15:04:05Z"), srv.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}
	
	logger.WithField("count", len(servers)).Info("Servers exported to CSV")
	return nil
}

// generateServerID generates a unique server ID
func generateServerID(host string) string {
	// Simple ID generation - in production, you'd want something more robust
	return fmt.Sprintf("srv-%s", strings.ReplaceAll(host, ".", "-"))
}