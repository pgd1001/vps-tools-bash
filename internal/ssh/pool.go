package ssh

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pgd1001/vps-tools/internal/logger"
)

// ConnectionPool manages a pool of SSH connections
type ConnectionPool struct {
	connections map[string]*Connection
	mutex       sync.RWMutex
	maxSize     int
	logger       *logger.Logger
	metrics     *PoolMetrics
}

// PoolMetrics represents connection pool metrics
type PoolMetrics struct {
	Created     int64     `json:"created"`
	Acquired    int64     `json:"acquired"`
	Released    int64     `json:"released"`
	Destroyed   int64     `json:"destroyed"`
	Active      int64     `json:"active"`
	MaxActive   int64     `json:"max_active"`
	TotalErrors  int64     `json:"total_errors"`
	LastActivity time.Time  `json:"last_activity"`
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int, logger *logger.Logger) *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*Connection),
		maxSize:     maxSize,
		logger:       logger,
		metrics: &PoolMetrics{
			LastActivity: time.Now(),
		},
	}
}

// Get gets a connection from the pool or creates a new one
func (p *ConnectionPool) Get(ctx context.Context) (*Connection, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if we have reached maximum connections
	if len(p.connections) >= p.maxSize {
		return nil, fmt.Errorf("connection pool is full (max: %d)", p.maxSize)
	}

	// Find an existing connection for the same server
	// For simplicity, we'll create a new connection each time
	// In production, you'd want to reuse existing connections
	p.metrics.Acquired++
	p.metrics.Active++
	p.metrics.LastActivity = time.Now()

	if p.metrics.Active > p.metrics.MaxActive {
		p.metrics.MaxActive = p.metrics.Active
	}

	// Create new connection
	// This is a placeholder - in real implementation, you'd create actual SSH connection
	connection := &Connection{
		ID:         fmt.Sprintf("conn_%d", p.metrics.Created),
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		Active:      true,
		logger:      p.logger,
	}

	p.connections[connection.ID] = connection
	p.metrics.Created++

	p.logger.WithFields(map[string]interface{}{
		"connection_id": connection.ID,
		"active_count":  len(p.connections),
		"max_size":     p.maxSize,
	}).Debug("Connection created and acquired from pool")

	return connection, nil
}

// Return returns a connection to the pool
func (p *ConnectionPool) Return(conn *Connection) {
	if conn == nil {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Update connection
	conn.LastUsed = time.Now()
	conn.Active = false

	// Update metrics
	p.metrics.Released++
	if p.metrics.Active > 0 {
		p.metrics.Active--
	}

	p.logger.WithFields(map[string]interface{}{
		"connection_id": conn.ID,
		"active_count":  len(p.connections),
	}).Debug("Connection returned to pool")
}

// Add adds an existing connection to the pool
func (p *ConnectionPool) Add(conn *Connection) {
	if conn == nil {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if connection already exists
	if _, exists := p.connections[conn.ID]; exists {
		p.logger.WithField("connection_id", conn.ID).Warn("Connection already exists in pool")
		return
	}

	// Add connection
	p.connections[conn.ID] = conn
	p.metrics.Created++
	p.metrics.Active++
	p.metrics.LastActivity = time.Now()

	p.logger.WithFields(map[string]interface{}{
		"connection_id": conn.ID,
		"active_count":  len(p.connections),
	}).Debug("Connection added to pool")
}

// Remove removes a connection from the pool
func (p *ConnectionPool) Remove(conn *Connection) {
	if conn == nil {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Remove connection
	delete(p.connections, conn.ID)
	p.metrics.Destroyed++
	if p.metrics.Active > 0 {
		p.metrics.Active--
	}

	p.logger.WithFields(map[string]interface{}{
		"connection_id": conn.ID,
		"active_count":  len(p.connections),
	}).Debug("Connection removed from pool")
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for id, conn := range p.connections {
		if err := conn.Close(); err != nil {
			p.logger.WithError(err).WithField("connection_id", id).Error("Failed to close connection")
		}
	}

	// Clear connections
	p.connections = make(map[string]*Connection)
	p.metrics.Active = 0

	p.logger.Info("Connection pool closed")
}

// Cleanup removes inactive connections
func (p *ConnectionPool) Cleanup(maxIdleTime time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	toRemove := []string{}

	for id, conn := range p.connections {
		// Remove connections that have been idle too long
		if !conn.Active && now.Sub(conn.LastUsed) > maxIdleTime {
			toRemove = append(toRemove, id)
		}
	}

	// Remove inactive connections
	for _, id := range toRemove {
		if conn, exists := p.connections[id]; exists {
			if err := conn.Close(); err != nil {
				p.logger.WithError(err).WithField("connection_id", id).Error("Failed to close inactive connection")
			}
			delete(p.connections, id)
			p.metrics.Destroyed++
			if p.metrics.Active > 0 {
				p.metrics.Active--
			}
		}
	}

	if len(toRemove) > 0 {
		p.logger.WithField("removed_count", len(toRemove)).Info("Cleaned up inactive connections")
	}
}

// GetMetrics returns current pool metrics
func (p *ConnectionPool) GetMetrics() *PoolMetrics {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Return a copy of metrics
	return &PoolMetrics{
		Created:     p.metrics.Created,
		Acquired:    p.metrics.Acquired,
		Released:    p.metrics.Released,
		Destroyed:   p.metrics.Destroyed,
		Active:      p.metrics.Active,
		MaxActive:   p.metrics.MaxActive,
		TotalErrors:  p.metrics.TotalErrors,
		LastActivity: p.metrics.LastActivity,
	}
}

// GetActiveCount returns the number of active connections
func (p *ConnectionPool) GetActiveCount() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	count := 0
	for _, conn := range p.connections {
		if conn.Active {
			count++
		}
	}

	return count
}

// GetTotalCount returns the total number of connections in the pool
func (p *ConnectionPool) GetTotalCount() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return len(p.connections)
}

// IsFull returns true if the pool is at maximum capacity
func (p *ConnectionPool) IsFull() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return len(p.connections) >= p.maxSize
}

// GetConnection returns a specific connection by ID
func (p *ConnectionPool) GetConnection(id string) (*Connection, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	conn, exists := p.connections[id]
	if !exists {
		return nil, fmt.Errorf("connection not found: %s", id)
	}

	return conn, nil
}

// ListConnections returns all connections in the pool
func (p *ConnectionPool) ListConnections() []*Connection {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	connections := make([]*Connection, 0, len(p.connections))
	for _, conn := range p.connections {
		connections = append(connections, conn)
	}

	return connections
}

// HealthCheck performs a health check on the pool
func (p *ConnectionPool) HealthCheck() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Check if pool is empty
	if len(p.connections) == 0 {
		return nil // Empty pool is healthy
	}

	// Check for stale connections
	now := time.Now()
	staleCount := 0
	for _, conn := range p.connections {
		// Consider connections stale if they haven't been used in 10 minutes
		if now.Sub(conn.LastUsed) > 10*time.Minute {
			staleCount++
		}
	}

	// Log warning if too many stale connections
	if staleCount > len(p.connections)/2 {
		p.logger.WithField("stale_count", staleCount).Warn("High number of stale connections in pool")
	}

	return nil
}

// ResetMetrics resets the pool metrics
func (p *ConnectionPool) ResetMetrics() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.metrics = &PoolMetrics{
		LastActivity: time.Now(),
	}

	p.logger.Info("Connection pool metrics reset")
}

// SetMaxSize updates the maximum pool size
func (p *ConnectionPool) SetMaxSize(maxSize int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	oldSize := p.maxSize
	p.maxSize = maxSize

	p.logger.WithFields(map[string]interface{}{
		"old_size": oldSize,
		"new_size": maxSize,
	}).Info("Connection pool max size updated")

	// If we're over the new limit, remove excess connections
	if len(p.connections) > maxSize {
		toRemove := len(p.connections) - maxSize
		removed := 0
		for id, conn := range p.connections {
			if removed >= toRemove {
				break
			}
			if err := conn.Close(); err != nil {
				p.logger.WithError(err).WithField("connection_id", id).Error("Failed to close excess connection")
			}
			delete(p.connections, id)
			removed++
		}

		p.logger.WithField("removed_count", removed).Info("Removed excess connections due to size limit")
	}
}

// GetMaxSize returns the current maximum pool size
func (p *ConnectionPool) GetMaxSize() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.maxSize
}