package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pgd1001/vps-tools/tui/models"
	"github.com/pgd1001/vps-tools/tui/styles"
)

// Styles for TUI components
type Styles struct {
	Normal  lipgloss.Style
	Header  lipgloss.Style
	Selected lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style
	Muted   lipgloss.Style
}

// Default styles
func DefaultStyles() Styles {
	return Styles{
		Normal:   lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		Header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")),
		Selected: lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("205")),
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
		Warning: lipgloss.NewStyle().Foreground(lipg.Color("220")),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		Info:    lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		Muted:   lipgloss.NewStyle().Faint(true),
	}
	}
}