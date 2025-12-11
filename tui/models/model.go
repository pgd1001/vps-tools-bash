package models

import (
	"time"

	"github.com/pgd1001/vps-tools/internal/server"
)

// Model represents the main TUI application state
type Model struct {
	ready     bool
	currentView View
	servers    []*server.Server
	selectedServer *server.Server
	healthChecks map[string][]*server.HealthCheck
	jobs       []*server.Job
	notifications []Notification
	loading    bool
	searchQuery string
	filter      Filter
	width      int
	height     int
	helpVisible bool
	lastUpdate  time.Time
}

// View represents different views in the TUI
type View int

const (
	ViewServers View = iota
	ViewHealth View = iota
	ViewJobs View = iota
	ViewNotifications View = iota
	ViewHelp View = iota
)

// Filter represents filtering options
type Filter struct {
	Tags    []string
	Status  string
	Search  string
}

// Notification represents a notification in the TUI
type Notification struct {
	ID        string
	Type      NotificationType
	Title     string
	Message   string
	Timestamp time.Time
	Read      bool
}

// NotificationType represents the type of notification
type NotificationType int

const (
	NotificationInfo NotificationType = iota
	NotificationWarning NotificationType = iota
	NotificationError NotificationType = iota
	NotificationSuccess NotificationType = iota
)

// NewModel creates a new TUI model
func NewModel() Model {
	return Model{
		ready:     false,
		currentView: ViewServers,
		servers:    []*server.Server{},
		healthChecks: make(map[string][]*server.HealthCheck),
		jobs:       []*server.Job{},
		notifications: []Notification{},
		searchQuery: "",
		filter:      Filter{},
		width:      80,
		height:     24,
		helpVisible: false,
		lastUpdate:  time.Now(),
	}
}

// UpdateServers updates the servers list in the model
func (m *Model) UpdateServers(servers []*server.Server) {
	m.servers = servers
	m.lastUpdate = time.Now()
}

// AddNotification adds a notification to the model
func (m *Model) AddNotification(notification Notification) {
	notification.Timestamp = time.Now()
	notification.ID = generateNotificationID()
	m.notifications = append(m.notifications, notification)
}

// RemoveNotification removes a notification by ID
func (m *Model) RemoveNotification(id string) {
	for i, notif := range m.notifications {
		if notif.ID == id {
			m.notifications = append(m.notifications[:i], m.notifications[i+1:]...)
			break
		}
	}
}

// GetNotifications returns all notifications
func (m *Model) GetNotifications() []Notification {
	return m.notifications
}

// ClearNotifications removes all notifications
func (m *Model) ClearNotifications() {
	m.notifications = []Notification{}
}

// SetView changes the current view
func (m *Model) SetView(view View) {
	m.currentView = view
}

// GetView returns the current view
func (m *Model) GetView() View {
	return m.currentView
}

// SetSearchQuery sets the search query
func (m *Model) SetSearchQuery(query string) {
	m.searchQuery = query
}

// GetSearchQuery returns the current search query
func (m *Model) GetSearchQuery() string {
	return m.searchQuery
}

// SetFilter sets the filter
func (m *Model) SetFilter(filter Filter) {
	m.filter = filter
}

// GetFilter returns the current filter
func (m *Model) GetFilter() Filter {
	return m.filter
}

// SetSize sets the terminal size
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// GetSize returns the terminal size
func (m *Model) GetSize() (int, int) {
	return m.width, m.height
}

// ToggleHelp toggles help visibility
func (m *Model) ToggleHelp() {
	m.helpVisible = !m.helpVisible
}

// IsHelpVisible returns whether help is visible
func (m *Model) IsHelpVisible() bool {
	return m.helpVisible
}

// SelectServer selects a server
func (m *Model) SelectServer(server *server.Server) {
	m.selectedServer = server
}

// GetSelectedServer returns the currently selected server
func (m *Model) GetSelectedServer() *server.Server {
	return m.selectedServer
}

// ClearSelection clears the server selection
func (m *Model) ClearSelection() {
	m.selectedServer = nil
}

// IsServerSelected returns whether a server is selected
func (m *Model) IsServerSelected() bool {
	return m.selectedServer != nil
}

// AddHealthCheck adds a health check result
func (m *Model) AddHealthCheck(serverID string, check *server.HealthCheck) {
	if m.healthChecks[serverID] == nil {
		m.healthChecks[serverID] = []*server.HealthCheck{}
	}
	m.healthChecks[serverID] = append(m.healthChecks[serverID], check)
}

// GetHealthChecks returns health checks for a server
func (m *Model) GetHealthChecks(serverID string) []*server.HealthCheck {
	if checks, exists := m.healthChecks[serverID]; exists {
		return checks
	}
	return []*server.HealthCheck{}
}

// AddJob adds a job to the model
func (m *Model) AddJob(job *server.Job) {
	m.jobs = append(m.jobs, job)
}

// GetJobs returns all jobs
func (m *Model) GetJobs() []*server.Job {
	return m.jobs
}

// GetJobsByServer returns jobs for a specific server
func (m *Model) GetJobsByServer(serverID string) []*server.Job {
	var jobs []*server.Job
	for _, job := range m.jobs {
		if job.ServerID == serverID {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// ClearJobs removes all jobs
func (m *Model) ClearJobs() {
	m.jobs = []*server.Job{}
}

// SetLoading sets the loading state
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// IsLoading returns whether the model is loading
func (m *Model) IsLoading() bool {
	return m.loading
}

// SetReady sets the ready state
func (m *Model) SetReady(ready bool) {
	m.ready = ready
}

// IsReady returns whether the model is ready
func (m *Model) IsReady() bool {
	return m.ready
}

// Helper functions
func generateNotificationID() string {
	return fmt.Sprintf("notif-%d", time.Now().Unix())
}

// UpdateLastUpdate updates the last update timestamp
func (m *Model) UpdateLastUpdate() {
	m.lastUpdate = time.Now()
}