package ssh

import (
	"sync"
	"time"

	"github.com/pgd1001/vps-tools/internal/logger"
)

// Connection represents an active SSH connection
type Connection struct {
	client      *ssh.Client
	session     *ssh.Session
	server      string
	id          string
	connectedAt time.Time
	lastUsed    time.Time
	active      bool
	logger      *logger.Logger
	mutex       sync.RWMutex
	metrics     *ConnectionMetrics
}

// ConnectionMetrics represents metrics for a connection
type ConnectionMetrics struct {
	CommandsExecuted int64         `json:"commands_executed"`
	FilesUploaded    int64         `json:"files_uploaded"`
	FilesDownloaded  int64         `json:"files_downloaded"`
	BytesTransferred int64         `json:"bytes_transferred"`
	Errors           int64         `json:"errors"`
	LastActivity     time.Time      `json:"last_activity"`
	TotalDuration    time.Duration `json:"total_duration"`
}

// NewConnection creates a new connection
func NewConnection(client *ssh.Client, server, id string, logger *logger.Logger) *Connection {
	return &Connection{
		client:      client,
		server:      server,
		id:          id,
		connectedAt: time.Now(),
		lastUsed:    time.Now(),
		active:      true,
		logger:      logger,
		metrics: &ConnectionMetrics{
			LastActivity: time.Now(),
		},
	}
}

// Close closes the SSH connection and session
func (c *Connection) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.active {
		return nil
	}

	c.logger.WithFields(map[string]interface{}{
		"connection_id": c.id,
		"server":       c.server,
		"duration":     time.Since(c.connectedAt).String(),
	}).Info("Closing SSH connection")

	// Close session if exists
	if c.session != nil {
		if err := c.session.Close(); err != nil {
			c.logger.WithError(err).Error("Failed to close SSH session")
		}
		c.session = nil
	}

	// Close client if exists
	if c.client != nil {
		if err := c.client.Close(); err != nil {
			c.logger.WithError(err).Error("Failed to close SSH client")
		}
		c.client = nil
	}

	c.active = false
	return nil
}

// IsActive returns whether the connection is active
func (c *Connection) IsActive() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.active
}

// GetServer returns the server address
func (c *Connection) GetServer() string {
	return c.server
}

// GetID returns the connection ID
func (c *Connection) GetID() string {
	return c.id
}

// GetConnectedAt returns when the connection was established
func (c *Connection) GetConnectedAt() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connectedAt
}

// GetLastUsed returns when the connection was last used
func (c *Connection) GetLastUsed() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lastUsed
}

// UpdateLastUsed updates the last used timestamp
func (c *Connection) UpdateLastUsed() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lastUsed = time.Now()
	c.metrics.LastActivity = time.Now()
}

// GetMetrics returns connection metrics
func (c *Connection) GetMetrics() *ConnectionMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.metrics
}

// IncrementCommandsExecuted increments the command counter
func (c *Connection) IncrementCommandsExecuted() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metrics.CommandsExecuted++
	c.metrics.LastActivity = time.Now()
}

// IncrementFilesUploaded increments the file upload counter
func (c *Connection) IncrementFilesUploaded() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metrics.FilesUploaded++
	c.metrics.LastActivity = time.Now()
}

// IncrementFilesDownloaded increments the file download counter
func (c *Connection) IncrementFilesDownloaded() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metrics.FilesDownloaded++
	c.metrics.LastActivity = time.Now()
}

// AddBytesTransferred adds to the bytes transferred counter
func (c *Connection) AddBytesTransferred(bytes int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metrics.BytesTransferred += bytes
	c.metrics.LastActivity = time.Now()
}

// IncrementErrors increments the error counter
func (c *Connection) IncrementErrors() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metrics.Errors++
	c.metrics.LastActivity = time.Now()
}

// AddDuration adds to the total duration
func (c *Connection) AddDuration(duration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metrics.TotalDuration += duration
	c.metrics.LastActivity = time.Now()
}

// ResetMetrics resets all connection metrics
func (c *Connection) ResetMetrics() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metrics = &ConnectionMetrics{
		LastActivity: time.Now(),
	}
}

// GetUptime returns the connection uptime
func (c *Connection) GetUptime() time.Duration {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return time.Since(c.connectedAt)
}

// IsIdle returns true if connection has been idle for more than specified duration
func (c *Connection) IsIdle(idleThreshold time.Duration) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return time.Since(c.lastUsed) > idleThreshold
}

// String returns string representation of the connection
func (c *Connection) String() string {
	return fmt.Sprintf("SSH Connection %s to %s (active: %v, uptime: %s)", 
		c.id, c.server, c.active, c.GetUptime())
}