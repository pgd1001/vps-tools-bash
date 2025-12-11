package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pgd1001/vps-tools/tui/models"
	"github.com/pgd1001/vps-tools/tui/styles"
)

	"github.com/pgd1001/vps-tools/tui/styles"
)

// Help component for displaying help information
type Help struct {
	width  int
	height int
	model   *models.Model
}

// NewHelp creates a new help component
func NewHelp(width, height int, model *models.Model) *Help {
	return &Help{
		width:  width,
		height: height,
		model:   model,
	}
}

// Init initializes the help component
func (h *Help) Init() tea.Cmd {
	return h.model.Init()
}

// Update updates the help component
func (h *Help) Update(msg tea.Msg) tea.Cmd {
	h.model.Update(msg)
}

// View renders the help content
func (h *Help) View() string {
	// Build help content
	content := `# vps-tools Help

## Global Shortcuts

### Navigation
- `q` / `Ctrl+C` - Quit application
- `1` / `Ctrl+P` - Switch to Servers view
- `2` / `Ctrl+H` - Switch to Health view
- `3` / `Ctrl+J` - Switch to Jobs view
- `4` / `Ctrl+L` - Switch to Notifications view
- `5` / `Ctrl+O` - Switch to Settings view
- `?` / `F1` - Show this help

## Views

### Servers View (1)
- List all servers with status and tags
- Filter by status, tags, or search
- Select a server for details
- Add new servers with interactive wizard
- Edit existing server configurations
- Delete servers with confirmation

### Health View (2)
- Real-time health monitoring dashboard
- View system metrics (CPU, memory, disk usage)
- Execute health checks on demand
- View historical health check results
- Configure health check thresholds and schedules

### Jobs View (3)
- List all jobs with status and filters
- View job details and logs
- Create new jobs with command wizard
- Monitor job execution in real-time
- Cancel running jobs
- View job history and statistics

### Notifications View (4)
- View system notifications and alerts
- Clear notifications
- Mark notifications as read
- Configure notification settings

### Settings View (5)
- Configure application settings
- Set preferences and themes
- Manage SSH and authentication
- Configure alerting and notifications
- Import/export configurations

## Commands

### Global Shortcuts
- `Ctrl+R` - Refresh all data
- `Ctrl+S` - Save configuration
- `Ctrl+Q` - Quit application

## Getting Started

1. Use arrow keys to navigate between views
2. Use number keys to select items
3. Use `Enter` to confirm actions
4. Use `Esc` to cancel operations
5. Use `Tab` to cycle through fields

## Need Help?

Press `?` to show this help screen.`

`

## Server Management

### Adding Servers
1. Use `n` to create a new server
2. Fill in the form that appears
3. Use `Enter` to save the server
4. Use `Esc` to cancel

### Health Monitoring

### Running Health Checks
1. Select servers and press `h` to run health checks
2. View real-time results as they update
3. Press `Ctrl+C` to stop monitoring

### Command Execution

### Job Management

### Running Jobs
1. Select servers and press `j` to create jobs
2. Monitor job execution in real-time
3. Press `Ctrl+C` to stop all jobs

### System Maintenance

### Getting Started

The TUI will automatically connect to servers and begin monitoring when started.`

Press any key to begin...`

`

	`

	return content
}

// Init initializes the help component
func (h *Help) Init() tea.Cmd {
	return h.model.Init()
}

// Update updates the help component
func (h *Help) Update(msg tea.Msg) tea.Cmd {
	h.model.Update(msg)
}

// View renders the help content
func (h *Help) View() string {
	return h.view()
}

// view renders the help content
func (h *Help) view() string {
	return h.view()
}

// view builds the help content
func (h *Help) view() string {
	content := `# vps-tools Help

## Global Shortcuts

### Navigation
- `q` / `Ctrl+C` - Quit application
- `1` / `Ctrl+P` - Switch to Servers view
- `2` / `Ctrl+H` - Switch to Health view
- `3` / `Ctrl+J` - Switch to Jobs view
- `4` / `Ctrl+L` - Switch to Notifications view
- `5` / `Ctrl+O` - Switch to Settings view
- `?` / `F1` - Show this help

## Views

### Servers View (1)
- List all servers with status and tags
- Filter by status, tags, or search
- Select a server for details
- Add new servers with interactive wizard
- Edit existing server configurations
- Delete servers with confirmation

### Health View (2)
- Real-time health monitoring dashboard
- View system metrics (CPU, memory, disk usage)
- Execute health checks on demand
- View historical health check results
- Configure health check thresholds and schedules

### Jobs View (3)
- List all jobs with status and filters
- View job details and logs
- Create new jobs with command wizard
- Monitor job execution in real-time
- Cancel running jobs
- View job history and statistics

### Notifications View (4)
- View system notifications and alerts
- Clear notifications
- Mark notifications as read
- Configure notification settings

### Settings View (5)
- Configure application settings
- Set preferences and themes
- Manage SSH and authentication
- Configure alerting and notifications
- Import/export configurations

## Commands

### Global Shortcuts
- `Ctrl+R` - Refresh all data
- `Ctrl+S` - Save configuration
- `Ctrl+Q` - Quit application

## Getting Started

1. Use arrow keys to navigate between views
2. Use number keys to select items
3. Use `Enter` to confirm actions
4. Use `Esc` to cancel operations
5. Use `Tab` to cycle through fields

## Need Help?

Press `?` to show this help screen.`

## Server Management

### Adding Servers
1. Use `n` to create a new server
2. Fill in the form that appears
3. Use `Enter` to save the server
4. Use `Esc` to cancel

### Health Monitoring

### Running Health Checks
1. Select servers and press `h` to run health checks
2. View real-time results as they update
3. Press `Ctrl+C` to stop monitoring

### Command Execution

### Job Management

### Running Jobs
1. Select servers and press `j` to create jobs
2. Monitor job execution in real-time
3. Press `Ctrl+C` to stop all jobs

### System Maintenance

### Getting Started

The TUI will automatically connect to servers and begin monitoring when started.

Press any key to begin...`

	`

	return content
}

// view builds the help content
func (h *Help) view() string {
	return h.view()
}

// view builds the help content
func (h *Help) view() string {
	return h.view()
}