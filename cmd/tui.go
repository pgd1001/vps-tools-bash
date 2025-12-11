package cmd

import (
	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pgd1001/vps-tools/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(tui.NewModel())
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}