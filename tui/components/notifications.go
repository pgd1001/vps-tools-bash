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

// NotificationList component for displaying notifications
type NotificationList struct {
	width  int
	height int
	model   *models.Model
}

// NewNotificationList creates a new notification list component
func NewNotificationList(width, height int, model *models.Model) *NotificationList {
	return &NotificationList{
		width:  width,
		height: height,
		model:   model,
	}
}

// Init initializes the component
func (n *NotificationList) Init() tea.Cmd {
	return n.model.Init()
}

// Update updates the component with new notifications
func (n *NotificationList) Update(msg tea.Msg) tea.Cmd {
	n.model.Update(msg)
}

// View renders the component
func (n *NotificationList) View() string {
	return n.view()
}

// view renders the current notifications
func (n *NotificationList) view() string {
	// Build notification list
	builder := strings.Builder{}
	builder.WriteString("NOTIFICATIONS\n")
	
	if len(n.model.notifications) == 0 {
		builder.WriteString("No notifications\n")
	} else {
		for _, notif := range n.model.notifications {
			status := getNotificationStatus(notif.Type)
			indicator := getNotificationIndicator(status)
			
			builder.WriteString(fmt.Sprintf("%s %s %s %s\n",
				indicator, notif.Title, notif.Message, notif.Timestamp))
		}
	}
	
	return builder.String()
}

// Helper functions
func getNotificationStatus(status models.NotificationType) string {
	switch status {
	case models.NotificationInfo:
		return "ℹ️"
	case models.NotificationWarning:
		return "⚠️"
	case models.NotificationError:
		return "❌"
	default:
		return "ℹ️"
	}
}

// getNotificationIndicator returns an indicator character for notification status
func getNotificationIndicator(status models.NotificationType) string {
	switch status {
	case models.NotificationInfo:
		return "✓"
	case models.NotificationWarning:
		return "⚠️"
	case models.NotificationError:
		return "✗"
	default:
		return "ℹ️"
	}
}