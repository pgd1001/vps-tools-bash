package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pgd1001/vps-tools/tui/models"
	"github.com/pgd1001/vps-tools/tui/styles"
	"github.com/pgd1001/vps-tools/internal/server"
)

// JobList component for displaying jobs
type JobList struct {
	width  int
	height int
	model   *models.Model
}

// NewJobList creates a new job list component
func NewJobList(width, height int, model *models.Model) *JobList {
	return &JobList{
		width:  width,
		height: height,
		model:   model,
	}
}

// Init initializes the component
func (j *JobList) Init() tea.Cmd {
	return j.model.Init()
}

// Update updates the component with new data
func (j *JobList) Update(msg tea.Msg) tea.Cmd {
	j.model.Update(msg)
}

// View renders the component
func (j *JobList) View() string {
	return j.view()
}

// view renders the current state
func (j *JobList) view() string {
	// Build table header
	builder := strings.Builder{}
	builder.WriteString("JOBS\n")
	builder.WriteString("ID\tSERVER\tCOMMAND\tSTATUS\tSTART\tDURATION\tEXIT\tMESSAGE\n")
	
	// Add job rows
	for _, job := range j.model.Jobs {
		status := getJobStatus(job.Status)
		builder.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			job.ID, job.ServerID, job.Command, status, job.StartTime.Format("2006-01-02 15:04:05"), job.Duration.String(), job.ExitCode, job.Message)
	}
	}
	
	return builder.String()
}

// Helper functions
func getJobStatus(status server.JobStatus) string {
	switch status {
	case server.JobStatusPending:
		return "pending"
	case server.JobStatusRunning:
		return "running"
	case server.JobStatusCompleted:
		return "completed"
	case server.JobStatusFailed:
		return "failed"
	case server.JobStatusTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// getJobStatusColor returns color for job status
func getJobStatusColor(status server.JobStatus) string {
	switch status {
	case server.JobStatusPending:
		return "33" // Yellow
	case server.JobStatusRunning:
		return "36" // Blue
	case server.JobStatusCompleted:
		return "32" // Green
	case server.JobStatusFailed:
		return "31" // Red
	case server.JobStatusTimeout:
		return "31" // Red
	default:
		return "37" // White
	}
}