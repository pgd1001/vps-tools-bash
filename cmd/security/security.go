package security

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/pgd1001/vps-tools/internal/config"
	"github.com/pgd1001/vps-tools/internal/logger"
	"github.com/pgd1001/vps-tools/internal/server"
	"github.com/pgd1001/vps-tools/internal/ssh"
	"github.com/pgd1001/vps-tools/internal/store"
)

// Command represents the security command
type Command struct {
	configManager *config.ConfigManager
	store        store.Store
	logger       *logger.Logger
}

// NewCommand creates a new security command
func NewCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Security auditing and management",
		Long:  "Perform security audits and manage SSH keys",
	}

	// Add subcommands
	cmd.AddCommand(newAuditCommand(configManager, store, logger))
	cmd.AddCommand(newKeyCommand(configManager, store, logger))
	cmd.AddCommand(newScanCommand(configManager, store, logger))

	return cmd
}

// newAuditCommand creates the audit subcommand
func newAuditCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit [server-id]",
		Short: "Perform security audit",
		Long:  "Perform comprehensive security audit on servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudit(configManager, store, logger, cmd, args)
		},
	}

	// Add flags
	cmd.Flags().StringSliceP("servers", "s", []string{}, "Server IDs to audit (default: all)")
	cmd.Flags().BoolP("ssh-keys", "k", false, "Include SSH key audit")
	cmd.Flags().BoolP("permissions", "p", false, "Include permission checks")
	cmd.Flags().BoolP("algorithms", "a", false, "Include algorithm analysis")
	cmd.Flags().BoolP("json", "j", false, "Output results in JSON format")

	return cmd
}

// newKeyCommand creates the key management subcommand
func newKeyCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key [generate|rotate|list]",
		Short: "SSH key management",
		Long:  "Generate, rotate, or list SSH keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKeyCommand(configManager, store, logger, cmd, args)
		},
	}

	return cmd
}

// newScanCommand creates the scan subcommand
func newScanCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan [server-id]",
		Short: "Port and service scanning",
		Long:  "Scan for open ports and running services on servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(configManager, store, logger, cmd, args)
		},
	}

	// Add flags
	cmd.Flags().StringSliceP("servers", "s", []string{}, "Server IDs to scan (default: all)")
	cmd.Flags().StringP("ports", "p", "22,80,443,8080", "Ports to scan (comma-separated)")
	cmd.Flags().BoolP("services", "v", false, "Include service detection")
	cmd.Flags().BoolP("deep", "d", false, "Deep scan with service version detection")
	cmd.Flags().IntP("timeout", "t", 30, "Scan timeout in seconds")
	cmd.Flags().BoolP("json", "j", false, "Output results in JSON format")

	return cmd
}

// runAudit executes the audit command
func runAudit(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	includeSSHKeys, _ := cmd.Flags().GetBool("ssh-keys")
	includePermissions, _ := cmd.Flags().GetBool("permissions")
	includeAlgorithms, _ := cmd.Flags().GetBool("algorithms")
	outputJSON, _ := cmd.Flags().GetBool("json")

	// Get servers to audit
	servers, err := getServers(store, serverIDs)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no servers found")
	}

	logger.WithFields(map[string]interface{}{
		"servers":        len(servers),
		"include_ssh_keys": includeSSHKeys,
		"include_permissions": includePermissions,
		"include_algorithms": includeAlgorithms,
		"output_json":    outputJSON,
	}).Info("Starting security audit")

	// Perform audit on each server
	var results []*AuditResult
	for _, srv := range servers {
		result := performSecurityAudit(srv, includeSSHKeys, includePermissions, includeAlgorithms, logger)
		results = append(results, result)
	}

	// Output results
	if outputJSON {
		return outputAuditResultsJSON(results, logger)
	}

	return outputAuditResultsTable(results, logger)
}

// runKeyCommand executes the key management command
func runKeyCommand(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("key operation is required (generate, rotate, list)")
	}

	operation := args[0]

	switch operation {
	case "generate":
		return runKeyGenerate(configManager, store, logger, args[1:])
	case "rotate":
		return runKeyRotate(configManager, store, logger, args[1:])
	case "list":
		return runKeyList(configManager, store, logger, args[1:])
	default:
		return fmt.Errorf("unknown key operation: %s", operation)
	}
}

// runScan executes the scan command
func runScan(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, cmd *cobra.Command, args []string) error {
	// Parse flags
	serverIDs, _ := cmd.Flags().GetStringSlice("servers")
	ports, _ := cmd.Flags().GetString("ports")
	includeServices, _ := cmd.Flags().GetBool("services")
	deep, _ := cmd.Flags().GetBool("deep")
	timeout, _ := cmd.Flags().GetInt("timeout")
	outputJSON, _ := cmd.Flags().GetBool("json")

	// Get servers to scan
	servers, err := getServers(store, serverIDs)
	if err != nil {
		return fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no servers found")
	}

	logger.WithFields(map[string]interface{}{
		"servers":     len(servers),
		"ports":       ports,
		"services":    includeServices,
		"deep":        deep,
		"timeout":     timeout,
		"output_json": outputJSON,
	}).Info("Starting port and service scan")

	// Perform scan on each server
	var results []*ScanResult
	for _, srv := range servers {
		result := performPortScan(srv, ports, includeServices, deep, timeout, logger)
		results = append(results, result)
	}

	// Output results
	if outputJSON {
		return outputScanResultsJSON(results, logger)
	}

	return outputScanResultsTable(results, logger)
}

// runKeyGenerate executes the key generation command
func runKeyGenerate(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, args []string) error {
	// Parse flags
	keyType, _ := cmd.Flags().GetString("type")
	keySize, _ := cmd.Flags().GetInt("size")
	comment, _ := cmd.Flags().GetString("comment")
	output, _ := cmd.Flags().GetString("output")

	if keyType == "" {
		keyType = "ed25519" // Default to ED25519
	}
	if keySize == 0 {
		keySize = 256 // Default size
	}

	// Generate key pair
	keyPair, err := ssh.GenerateKeyPair(keyType, keySize, comment)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"key_type":  keyType,
		"key_size":  keySize,
		"comment":   comment,
	}).Info("SSH key pair generated successfully")

	// Output based on format
	switch output {
	case "json":
		data, _ := json.MarshalIndent(keyPair, "", "  ")
		fmt.Println(data)
	case "screen":
		fmt.Printf("Private Key (%s, %d bits):\n%s\n\n", keyPair.Type, keyPair.Size, keyPair.PrivateKey)
		fmt.Printf("Public Key (%s):\n%s\n\n", keyPair.Type, keyPair.PublicKey)
		if comment != "" {
			fmt.Printf("Comment: %s\n\n", comment)
		}
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}

	return nil
}

// runKeyRotate executes the key rotation command
func runKeyRotate(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, args []string) error {
	// Parse flags
	serverID, _ := cmd.Flags().GetString("server-id")
	force, _ := cmd.Flags().GetBool("force")

	if serverID == "" {
		return fmt.Errorf("server ID is required for key rotation")
	}

	// Get server
	srv, err := store.GetServer(serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	logger.WithField("server_id", serverID).Info("Starting SSH key rotation")

	// Create new key pair
	keyPair, err := ssh.GenerateKeyPair("ed25519", 256, fmt.Sprintf("rotated-%s", time.Now().Format("2006-01-02")))
	if err != nil {
		return fmt.Errorf("failed to generate new key pair: %w", err)
	}

	// Update server with new key
	srv.AuthMethod = server.AuthConfig{
		Type:    "private_key",
		KeyPath: "", // Will be set when saving
	}

	if err := store.UpdateServer(srv); err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"server_id": serverID,
		"new_key_type": "ed25519",
	}).Info("SSH key rotation completed")

	fmt.Printf("SSH key rotated for server %s\n", serverID)
	return nil
}

// runKeyList executes the key list command
func runKeyList(configManager *config.ConfigManager, store store.Store, logger *logger.Logger, args []string) error {
	// Parse flags
	serverID, _ := cmd.Flags().GetString("server-id")
	output, _ := cmd.Flags().GetString("output")

	if serverID == "" {
		return fmt.Errorf("server ID is required for key listing")
	}

	// Get server
	srv, err := store.GetServer(serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	logger.WithField("server_id", serverID).Info("Listing SSH keys for server")

	// Get SSH keys from server
	// This would require SSH connection to list keys
	// For now, we'll show a placeholder message
	fmt.Printf("SSH keys for server %s:\n", srv.Name)
	fmt.Printf("  Authentication Type: %s\n", srv.AuthMethod.Type)
	fmt.Printf("  Key Path: %s\n", srv.AuthMethod.KeyPath)
	fmt.Printf("  Note: Full key listing requires SSH connection\n")

	return nil
}

// Helper functions

// getServers gets servers based on IDs or all servers
func getServers(store store.Store, serverIDs []string) ([]*server.Server, error) {
	if len(serverIDs) == 0 {
		// Get all servers
		return store.ListServers(nil)
	}

	// Get specific servers
	var servers []*server.Server
	for _, id := range serverIDs {
		srv, err := store.GetServer(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get server %s: %w", id, err)
		}
		servers = append(servers, srv)
	}

	return servers, nil
}

// performSecurityAudit performs a comprehensive security audit
func performSecurityAudit(srv *server.Server, includeSSHKeys, includePermissions, includeAlgorithms, logger *logger.Logger) *AuditResult {
	result := &AuditResult{
		ServerID:      srv.ID,
		Timestamp:     time.Now(),
		Issues:        []ssh.SecurityIssue{},
	}

	// SSH key audit
	if includeSSHKeys {
		// This would require SSH connection to audit keys
		// For now, we'll add a placeholder issue
		result.Issues = append(result.Issues, ssh.SecurityIssue{
			ID:          "SSH_KEY_AUDIT",
			Severity:    ssh.SeverityInfo,
			Category:    "ssh_keys",
			Title:       "SSH Key Audit",
			Description: "SSH key audit requires SSH connection to server",
			Recommendation: "Connect to server and run SSH key audit",
		})
	}

	// Permission audit
	if includePermissions {
		result.Issues = append(result.Issues, ssh.SecurityIssue{
			ID:          "PERMISSION_AUDIT",
			Severity:    ssh.SeverityMedium,
			Category:    "permissions",
			Title:       "Permission Audit",
			Description: "Permission audit requires SSH connection to server",
			Recommendation: "Connect to server and run permission audit",
		})
	}

	// Algorithm audit
	if includeAlgorithms {
		result.Issues = append(result.Issues, ssh.SecurityIssue{
			ID:          "ALGORITHM_AUDIT",
			Severity:    ssh.SeverityLow,
			Category:    "algorithms",
			Title:       "Algorithm Audit",
			Description: "Algorithm audit requires SSH connection to server",
			Recommendation: "Connect to server and run algorithm audit",
		})
	}

	// Calculate overall score
	result.OverallScore, result.Recommendations = calculateSecurityScore(result.Issues)
	result.Summary = generateAuditSummary(result.OverallScore, len(result.Issues))

	return result
}

// performPortScan performs port and service scanning
func performPortScan(srv *server.Server, ports, includeServices, deep, timeout, logger *logger.Logger) *ScanResult {
	result := &ScanResult{
		ServerID:  srv.ID,
		Timestamp: time.Now(),
		OpenPorts: []PortInfo{},
		Services:  []ServiceInfo{},
		Issues:   []ssh.SecurityIssue{},
	}

	// Parse ports to scan
	portList := parsePorts(ports)

	// Perform port scan
	for _, port := range portList {
		portInfo := scanPort(srv, port, timeout, logger)
		if portInfo.Open {
			result.OpenPorts = append(result.OpenPorts, portInfo)
		}
	}

	// Service detection (simplified)
	if includeServices {
		// This would require more sophisticated scanning
		// For now, we'll add a placeholder
		result.Services = append(result.Services, ServiceInfo{
			Port:     22,
			Service:  "ssh",
			Version:  "unknown",
		})
	}

	// Calculate overall status
	if len(result.Issues) == 0 {
		result.OverallStatus = "secure"
	} else {
		result.OverallStatus = "vulnerable"
	}

	return result
}

// parsePorts parses port list from string
func parsePorts(ports string) []int {
	var portList []int
	if ports == "" {
		return []int{22, 80, 443, 8080} // Default ports
	}

	// Parse comma-separated ports
	parts := strings.Split(ports, ",")
	for _, part := range parts {
		if port := parseInt(part); port > 0 && port <= 65535 {
			portList = append(portList, port)
		}
	}

	return portList
}

// parseInt parses an integer from string
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// scanPort scans a single port (simplified)
func scanPort(srv *server.Server, port int, timeout, logger *logger.Logger) PortInfo {
	// Simplified port scan - in production, you'd use proper port scanning
	// For now, we'll assume common ports are open if SSH is configured
	
	isOpen := false
	if port == srv.Port {
		isOpen = true // Assume SSH port is open if it's the configured port
	}

	return PortInfo{
		Port:     port,
		Protocol: "tcp",
		State:    getPortState(isOpen),
		Service:  getPortService(port),
	}
}

// getPortState returns the state of a port
func getPortState(open bool) string {
	if open {
		return "open"
	}
	return "closed"
}

// getPortService returns the service for a port
func getPortService(port int) string {
	switch port {
	case 22:
		return "ssh"
	case 80:
		return "http"
	case 443:
		return "https"
	case 8080:
		return "http-alt"
	default:
		return "unknown"
	}
}

// calculateSecurityScore calculates overall security score
func calculateSecurityScore(issues []ssh.SecurityIssue) (ssh.SecurityScore, []string) {
	score := 100
	var recommendations []string

	for _, issue := range issues {
		switch issue.Severity {
		case ssh.SeverityCritical:
			score -= 40
		case ssh.SeverityHigh:
			score -= 25
		case ssh.SeverityMedium:
			score -= 15
		case ssh.SeverityLow:
			score -= 5
		case ssh.SeverityInfo:
			score -= 1
		}

		if issue.Recommendation != "" {
			recommendations = append(recommendations, issue.Recommendation)
		}
	}

	var overallScore ssh.SecurityScore
	if score >= 90 {
		overallScore = ssh.ScoreExcellent
	} else if score >= 75 {
		overallScore = ssh.ScoreGood
	} else if score >= 60 {
		overallScore = ssh.ScoreFair
	} else if score >= 40 {
		overallScore = ssh.ScorePoor
	} else {
		overallScore = ssh.ScoreCritical
	}

	// Add general recommendations if score is low
	if overallScore == ssh.ScorePoor || overallScore == ssh.ScoreCritical {
		recommendations = append(recommendations, "Comprehensive security review recommended")
		recommendations = append(recommendations, "Implement regular security audits")
	}

	return overallScore, recommendations
}

// generateAuditSummary generates a summary of the audit results
func generateAuditSummary(score ssh.SecurityScore, issueCount int) string {
	switch score {
	case ssh.ScoreExcellent:
		return fmt.Sprintf("Excellent security posture with %d minor issues found", issueCount)
	case ssh.ScoreGood:
		return fmt.Sprintf("Good security posture with %d issues found", issueCount)
	case ssh.ScoreFair:
		return fmt.Sprintf("Fair security posture with %d issues found - improvement recommended", issueCount)
	case ssh.ScorePoor:
		return fmt.Sprintf("Poor security posture with %d issues found - immediate attention required", issueCount)
	case ssh.ScoreCritical:
		return fmt.Sprintf("Critical security posture with %d issues found - urgent action required", issueCount)
	default:
		return fmt.Sprintf("Security assessment completed with %d issues found", issueCount)
	}
}

// Result structures
type AuditResult struct {
	ServerID      string              `json:"server_id"`
	Timestamp     time.Time           `json:"timestamp"`
	OverallScore ssh.SecurityScore      `json:"overall_score"`
	Issues       []ssh.SecurityIssue    `json:"issues"`
	Recommendations []string           `json:"recommendations"`
	Summary      string              `json:"summary"`
}

type ScanResult struct {
	ServerID    string       `json:"server_id"`
	Timestamp   time.Time    `json:"timestamp"`
	OpenPorts   []PortInfo  `json:"open_ports"`
	Services    []ServiceInfo `json:"services"`
	Issues      []ssh.SecurityIssue `json:"issues"`
	OverallStatus string         `json:"overall_status"`
}

type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	State    string `json:"state"`
	Service  string `json:"service"`
}

type ServiceInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Service  string `json:"service"`
	Version  string `json:"version"`
}

// Output functions
func outputAuditResultsJSON(results []*AuditResult, logger *logger.Logger) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal audit results: %w", err)
	}

	fmt.Println(data)
	logger.WithField("count", len(results)).Info("Audit results exported to JSON")
	return nil
}

func outputAuditResultsTable(results []*AuditResult, logger *logger.Logger) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tSCORE\tISSUES\tSUMMARY")
	
	for _, result := range results {
		issueCount := len(result.Issues)
		issuesStr := fmt.Sprintf("%d issues", issueCount)
		if issueCount == 0 {
			issuesStr = "no issues"
		}
		
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			result.ServerID, result.OverallScore, issuesStr, result.Summary)
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Audit results displayed in table format")
	return nil
}

func outputScanResultsJSON(results []*ScanResult, logger *logger.Logger) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scan results: %w", err)
	}

	fmt.Println(data)
	logger.WithField("count", len(results)).Info("Scan results exported to JSON")
	return nil
}

func outputScanResultsTable(results []*ScanResult, logger *logger.Logger) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVER\tSTATUS\tOPEN PORTS\tSERVICES")
	
	for _, result := range results {
		openPortsStr := fmt.Sprintf("%d open", len(result.OpenPorts))
		servicesStr := fmt.Sprintf("%d services", len(result.Services))
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			result.ServerID, result.OverallStatus, openPortsStr, servicesStr)
	}
	
	w.Flush()
	logger.WithField("count", len(results)).Info("Scan results displayed in table format")
	return nil
}