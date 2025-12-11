package ssh

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/pgd1001/vps-tools/internal/logger"
	"github.com/pgd1001/vps-tools/internal/server"
)

// Client represents an SSH client for server operations
type Client struct {
	config     *Config
	pool       *ConnectionPool
	logger     *logger.Logger
	sshConfig  *ssh.ClientConfig
	knownHosts ssh.HostKeyCallback
}

// Config represents SSH client configuration
type Config struct {
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	User           string        `yaml:"user"`
	AuthMethod     AuthMethod    `yaml:"auth_method"`
	Timeout        time.Duration `yaml:"timeout"`
	BastionHost    string        `yaml:"bastion_host"`
	KnownHostsFile string        `yaml:"known_hosts_file"`
	StrictHostKey  bool          `yaml:"strict_host_key"`
	MaxRetries     int           `yaml:"max_retries"`
	RetryDelay     time.Duration `yaml:"retry_delay"`
	KeepAlive      time.Duration `yaml:"keep_alive"`
}

// AuthMethod interface for SSH authentication
type AuthMethod interface {
	Authenticate() (ssh.AuthMethod, error)
	Type() string
	String() string
}

// Connection represents an active SSH connection
type Connection struct {
	client     *ssh.Client
	session    *ssh.Session
	server     string
	connectedAt time.Time
	lastUsed   time.Time
	logger     *logger.Logger
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Duration  time.Duration `json:"duration"`
	Error     error  `json:"error,omitempty"`
}

// NewClient creates a new SSH client
func NewClient(config *Config, logger *logger.Logger) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SSH config: %w", err)
	}

	client := &Client{
		config: config,
		logger: logger,
	}

	// Initialize SSH client config
	if err := client.initSSHConfig(); err != nil {
		return nil, fmt.Errorf("failed to initialize SSH config: %w", err)
	}

	// Initialize connection pool
	client.pool = NewConnectionPool(config.MaxConnections, logger)

	return client, nil
}

// initSSHConfig initializes the SSH client configuration
func (c *Client) initSSHConfig() error {
	// Get authentication method
	authMethod, err := c.config.AuthMethod.Authenticate()
	if err != nil {
		return fmt.Errorf("failed to get authentication method: %w", err)
	}

	// Create SSH client config
	c.sshConfig = &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: c.getKnownHostsCallback(),
		Timeout:         c.config.Timeout,
		Config:          c.getSSHConfig(),
	}

	return nil
}

// getKnownHostsCallback returns the appropriate host key callback
func (c *Client) getKnownHostsCallback() ssh.HostKeyCallback {
	if c.config.StrictHostKey {
		return ssh.InsecureIgnoreHostKey() // For testing - in production, use proper known hosts
	}
	
	// In production, you'd implement proper known hosts checking
	return ssh.InsecureIgnoreHostKey() // Temporary - implement proper known hosts
}

// getSSHConfig returns SSH configuration options
func (c *Client) getSSHConfig() map[string]string {
	config := make(map[string]string)
	
	// Set keep alive
	config["ServerAliveInterval"] = "60"
	config["ServerAliveCountMax"] = "3"
	
	// Disable strict host key checking if not strict
	if !c.config.StrictHostKey {
		config["StrictHostKeyChecking"] = "no"
	}
	
	return config
}

// Connect establishes an SSH connection to the server
func (c *Client) Connect(ctx context.Context) (*Connection, error) {
	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	
	c.logger.WithField("server", address).Info("Attempting SSH connection")
	
	var client *ssh.Client
	var err error
	
	// Try with bastion host if configured
	if c.config.BastionHost != "" {
		client, err = c.connectViaBastion(ctx)
	} else {
		client, err = ssh.Dial("tcp", address, c.sshConfig)
	}
	
	if err != nil {
		c.logger.WithError(err).Error("SSH connection failed")
		return nil, fmt.Errorf("SSH connection failed: %w", err)
	}
	
	c.logger.WithField("server", address).Info("SSH connection established")
	
	connection := &Connection{
		client:      client,
		server:      address,
		connectedAt: time.Now(),
		lastUsed:    time.Now(),
		logger:      c.logger,
	}
	
	// Add to pool
	c.pool.Add(connection)
	
	return connection, nil
}

// connectViaBastion connects to a server via a bastion host
func (c *Client) connectViaBastion(ctx context.Context) (*ssh.Client, error) {
	// Connect to bastion host first
	bastionAddress := fmt.Sprintf("%s:22", c.config.BastionHost)
	bastionConfig := &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            []ssh.AuthMethod{c.config.AuthMethod.Authenticate()},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.config.Timeout,
	}
	
	bastionClient, err := ssh.Dial("tcp", bastionAddress, bastionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bastion host: %w", err)
	}
	
	// Connect from bastion to target server
	targetAddress := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	conn, err := bastionClient.Dial("tcp", targetAddress)
	if err != nil {
		bastionClient.Close()
		return nil, fmt.Errorf("failed to connect to target via bastion: %w", err)
	}
	
	// Create SSH client connection
	ncc, chans, reqs, err := ssh.NewClientConn(conn, targetAddress)
	if err != nil {
		conn.Close()
		bastionClient.Close()
		return nil, fmt.Errorf("failed to create SSH client connection: %w", err)
	}
	
	go ssh.DiscardRequests(reqs)
	go ssh.DiscardChannels(chans)
	
	client, err := ssh.NewClient(ncc, chans, reqs)
	if err != nil {
		conn.Close()
		bastionClient.Close()
		return nil, fmt.Errorf("failed to create SSH client: %w", err)
	}
	
	return client, nil
}

// ExecuteCommand executes a command on the server
func (c *Client) ExecuteCommand(ctx context.Context, command string, workingDir string) (*CommandResult, error) {
	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer c.pool.Return(conn)
	
	startTime := time.Now()
	
	// Create session
	session, err := conn.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()
	
	// Set working directory if specified
	if workingDir != "" {
		err = session.RequestSubsystem("exec", fmt.Sprintf("cd %s && %s", workingDir, command))
	} else {
		err = session.RequestSubsystem("exec", command)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}
	
	// Capture output
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	
	// Run command
	err = session.Run()
	duration := time.Since(startTime)
	
	result := &CommandResult{
		ExitCode: 0,
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Duration:  duration,
	}
	
	if err != nil {
		if exitError, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitError.ExitStatus()
			result.Error = fmt.Errorf("command exited with code %d", result.ExitCode)
		} else {
			result.Error = err
		}
	}
	
	// Log command execution
	c.logger.WithFields(map[string]interface{}{
		"server":      conn.server,
		"command":     command,
		"working_dir": workingDir,
		"exit_code":   result.ExitCode,
		"duration":    duration.String(),
	}).Info("Command executed")
	
	if result.Error != nil {
		c.logger.WithError(result.Error).Error("Command execution failed")
	}
	
	return result, nil
}

// ExecuteCommandWithOutput executes a command and returns structured output
func (c *Client) ExecuteCommandWithOutput(ctx context.Context, command string, workingDir string) (map[string]interface{}, error) {
	result, err := c.ExecuteCommand(ctx, command, workingDir)
	if err != nil {
		return nil, err
	}
	
	output := map[string]interface{}{
		"exit_code": result.ExitCode,
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"duration":  result.Duration.String(),
		"success":   result.ExitCode == 0,
	}
	
	return output, nil
}

// StartInteractiveShell starts an interactive shell session
func (c *Client) StartInteractiveShell(ctx context.Context) error {
	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	
	// Create session
	session, err := conn.client.NewSession()
	if err != nil {
		c.pool.Return(conn)
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	
	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	
	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", 80, 40, modes); err != nil {
		session.Close()
		c.pool.Return(conn)
		return fmt.Errorf("failed to request pseudo terminal: %w", err)
	}
	
	// Start shell
	if err := session.Shell(); err != nil {
		session.Close()
		c.pool.Return(conn)
		return fmt.Errorf("failed to start shell: %w", err)
	}
	
	c.logger.WithField("server", conn.server).Info("Interactive shell started")
	
	// Wait for session to end
	<-ctx.Done()
	
	session.Close()
	c.pool.Return(conn)
	
	c.logger.WithField("server", conn.server).Info("Interactive shell ended")
	
	return nil
}

// UploadFile uploads a file to the server
func (c *Client) UploadFile(ctx context.Context, localPath, remotePath string, permissions string) error {
	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer c.pool.Return(conn)
	
	// Create session
	session, err := conn.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()
	
	// Open SFTP connection
	// Note: This would require implementing SFTP or using external library
	// For now, we'll use SCP via SSH
	return c.uploadViaSCP(session, localPath, remotePath, permissions)
}

// uploadViaSCP uploads file using SCP protocol
func (c *Client) uploadViaSCP(session *ssh.Session, localPath, remotePath, permissions string) error {
	// This is a simplified SCP implementation
	// In production, you'd want to use a proper SFTP library
	command := fmt.Sprintf("scp -p %s %s@%s:%s", localPath, c.config.User, c.config.Host, remotePath)
	
	c.logger.WithFields(map[string]interface{}{
		"local_path":  localPath,
		"remote_path": remotePath,
		"server":      c.config.Host,
	}).Info("Starting file upload")
	
	return session.Run(command)
}

// DownloadFile downloads a file from the server
func (c *Client) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer c.pool.Return(conn)
	
	// Create session
	session, err := conn.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()
	
	// Download via SCP
	command := fmt.Sprintf("scp %s@%s:%s %s", c.config.User, c.config.Host, remotePath, localPath)
	
	c.logger.WithFields(map[string]interface{}{
		"remote_path": remotePath,
		"local_path":  localPath,
		"server":      c.config.Host,
	}).Info("Starting file download")
	
	return session.Run(command)
}

// TestConnection tests the SSH connection
func (c *Client) TestConnection(ctx context.Context) error {
	conn, err := c.Connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	c.logger.WithField("server", c.config.Host).Info("SSH connection test successful")
	
	return nil
}

// Close closes the SSH client and all connections
func (c *Client) Close() error {
	c.logger.Info("Closing SSH client")
	
	// Close all connections in pool
	if c.pool != nil {
		c.pool.Close()
	}
	
	return nil
}

// GetServerInfo returns information about the server
func (c *Client) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	commands := map[string]string{
		"hostname":     "hostname",
		"uname":        "uname -a",
		"uptime":       "uptime",
		"whoami":       "whoami",
		"pwd":          "pwd",
		"df":          "df -h",
		"free":         "free -h",
		"ps":          "ps aux",
	}
	
	info := &ServerInfo{
		Host: c.config.Host,
		Port: c.config.Port,
		User: c.config.User,
	}
	
	for key, command := range commands {
		result, err := c.ExecuteCommand(ctx, command, "")
		if err != nil {
			c.logger.WithError(err).Warnf("Failed to get %s", key)
			continue
		}
		
		switch key {
		case "hostname":
			info.Hostname = result.Stdout
		case "uname":
			info.Uname = result.Stdout
		case "uptime":
			info.Uptime = result.Stdout
		case "whoami":
			info.CurrentUser = result.Stdout
		case "pwd":
			info.CurrentDir = result.Stdout
		case "df":
			info.DiskUsage = result.Stdout
		case "free":
			info.MemoryUsage = result.Stdout
		case "ps":
			info.Processes = result.Stdout
		}
	}
	
	return info, nil
}

// ServerInfo represents information about a server
type ServerInfo struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	Hostname     string `json:"hostname"`
	Uname        string `json:"uname"`
	Uptime       string `json:"uptime"`
	CurrentUser string `json:"current_user"`
	CurrentDir   string `json:"current_dir"`
	DiskUsage    string `json:"disk_usage"`
	MemoryUsage  string `json:"memory_usage"`
	Processes    string `json:"processes"`
	ConnectedAt  time.Time `json:"connected_at"`
}

// Validate validates the SSH configuration
func (c *Config) Validate() error {
	if c.Host == "" {
		return ErrSSHHostRequired
	}
	if c.Port <= 0 || c.Port > 65535 {
		return ErrInvalidSSHPort
	}
	if c.User == "" {
		return ErrSSHUserRequired
	}
	if c.AuthMethod == nil {
		return ErrSSHAuthRequired
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second // default timeout
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3 // default retries
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = 5 * time.Second // default retry delay
	}
	
	return c.AuthMethod.Validate()
}

// GetAddress returns the full server address
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// String returns string representation of the SSH configuration
func (c *Config) String() string {
	return fmt.Sprintf("SSH://%s@%s:%d", c.User, c.Host, c.Port)
}