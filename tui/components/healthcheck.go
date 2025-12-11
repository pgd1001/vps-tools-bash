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

	"github.com/pgd1001/vps-tools/internal/server"
)

// HealthCheck component for displaying health check results
type HealthCheck struct {
	width  int
	height int
	model   *models.Model
}

// NewHealthCheck creates a new health check component
func NewHealthCheck(width, height int, model *models.Model) *HealthCheck {
	return &HealthCheck{
		width:  width,
		height: height,
		model:   model,
	}
}

// Init initializes the component
func (h *HealthCheck) Init() tea.Cmd {
	return h.model.Init()
}

// Update updates the component with new data
func (h *HealthCheck) Update(msg tea.Msg) tea.Cmd {
	h.model.Update(msg)
}

// View renders the component
func (h *HealthCheck) View() string {
	return h.view()
}

// view renders the current state
func (h *HealthCheck) view() string {
	if h.model.SelectedServer == nil {
		return "No server selected"
	}

	// Build table view
	var builder strings.Builder
	builder.WriteString("Health Check Results\n\n")
	builder.WriteString("Server: ")
	if h.model.SelectedServer != nil {
		builder.WriteString(h.model.SelectedServer.Name)
	}
	builder.WriteString("\n\n")

	// Add health checks
	if len(h.model.HealthChecks[h.model.SelectedServer.ID]) > 0 {
		for _, check := range h.model.HealthChecks[h.model.SelectedServer.ID] {
			status := getHealthStatus(check.Status)
			builder.WriteString(fmt.Sprintf("  %s: %s\n", check.Name))
			builder.WriteString(fmt.Sprintf("    Status: %s\n", status))
			builder.WriteString(fmt.Sprintf("    Message: %s\n", check.Message))
			if check.Value != nil {
				builder.WriteString(fmt.Sprintf("    Value: %v\n", check.Value))
			}
			builder.WriteString(fmt.Sprintf("    Timestamp: %s\n", check.Timestamp.Format("2006-01-02 15:04:05Z")))
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// Helper functions
func getHealthStatus(status string) string {
	switch status {
	case "ok":
		return "✅"
	case "warning":
		return "⚠️"
	case "critical":
		return "❌"
	default:
		return "❓"
	}
}

// getHealthStatusColor returns color for health status
func getHealthStatusColor(status string) string {
	switch status {
	case "ok":
		return "42" // Green
	case "warning":
		return "43" // Yellow
	case "critical":
		return "41" // Red
	default:
		return "41" // Red
	}
}