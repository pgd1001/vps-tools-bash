package main

import (
	"os"

	"github.com/pgd1001/vps-tools/cmd"
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/logger"
	"github.com/pgd1001/vps-tools/internal/store"
	"github.com/pgd1001/vps-tools/tui"
)

func main() {
	// Create dependencies
	configManager := config.NewConfigManager()
	store, err := store.NewBoltStore(configManager.Get())
	if err != nil {
		logger := logger.NewDefaultLogger()
		logger.Error("Failed to initialize store:", err)
		os.Exit(1)
	}

	// Create TUI program
	program := tui.NewProgram(configManager, store, logger)
	if _, err := program(); err != nil {
		logger.Error("Failed to start TUI:", err)
		os.Exit(1)
	}

	logger.Info("vps-tools TUI started successfully")
}