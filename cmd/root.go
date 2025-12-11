package cmd

import (
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/inventory"
	"github.com/pgd1001/vps-tools/internal/health"
	"github.com/pgd1001/vps-tools/internal/run"
	"github.com/pgd1001/vps-tools/internal/security"
	"github.com/pgd1001/vps-tools/internal/docker"
	"github.com/pgd1001/vps-tools/internal/maintenance"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vps-tools",
	Short: "Modern terminal tool for managing Linux servers",
	Long:  "vps-tools replaces legacy bash scripts with a fast, secure, structured terminal application.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add all subcommands
	rootCmd.AddCommand(inventory.NewCommand(configManager, store, logger))
	rootCmd.AddCommand(health.NewCommand(configManager, store, logger))
	rootCmd.AddCommand(run.NewCommand(configManager, store, logger))
	rootCmd.AddCommand(security.NewCommand(configManager, store, logger))
	rootCmd.AddCommand(docker.NewCommand(configManager, store, logger))
	rootCmd.AddCommand(maintenance.NewCommand(configManager, store, logger))
}