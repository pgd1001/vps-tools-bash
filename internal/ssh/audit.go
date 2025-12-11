package ssh

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/pgd1001/vps-tools/internal/logger"
	"github.com/pgd1001/vps-tools/internal/server"
)

// Auditor performs SSH security audits
type Auditor struct {
	logger     *logger.Logger
	knownHosts string
}

// NewAuditor creates a new SSH auditor
func NewAuditor(logger *logger.Logger, knownHostsFile string) *Auditor {
	return &Auditor{
		logger:     logger,
		knownHosts: knownHostsFile,
	}
}

// AuditResult represents the result of an SSH audit
type AuditResult struct {
	ServerID      string              `json:"server_id"`
	Timestamp     time.Time           `json:"timestamp"`
	OverallScore  SecurityScore      `json:"overall_score"`
	Issues       []SecurityIssue    `json:"issues"`
	Recommendations []string           `json:"recommendations"`
	Summary      string              `json:"summary"`
}

// SecurityIssue represents a security issue found during audit
type SecurityIssue struct {
	ID             string       `json:"id"`
	Severity       Severity     `json:"severity"`
	Category       string       `json:"category"`
	Title          string       `json:"title"`
	Description     string       `json:"description"`
	Impact         string       `json:"impact"`
	Recommendation string       `json:"recommendation"`
	AffectedKeys  []string     `json:"affected_keys,omitempty"`
}

// SecurityScore represents overall security score
type SecurityScore string

const (
	ScoreExcellent SecurityScore = "excellent"
	ScoreGood      SecurityScore = "good"
	ScoreFair      SecurityScore = "fair"
	ScorePoor      SecurityScore = "poor"
	ScoreCritical  SecurityScore = "critical"
)

// Severity represents severity of a security issue
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// AuditSSHKeys performs a comprehensive SSH key audit
func (a *Auditor) AuditSSHKeys(serverID string, sshDir string) (*AuditResult, error) {
	a.logger.WithField("server_id", serverID).Info("Starting SSH key audit")

	result := &AuditResult{
		ServerID:  serverID,
		Timestamp: time.Now(),
		Issues:    []SecurityIssue{},
	}

	// Scan for SSH keys
	keys, err := a.scanSSHKeys(sshDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan SSH keys: %w", err)
	}

	// Analyze each key
	for _, key := range keys {
		issues := a.analyzeKey(key)
		result.Issues = append(result.Issues, issues...)
	}

	// Check for duplicate keys
	duplicateIssues := a.checkDuplicateKeys(keys)
	result.Issues = append(result.Issues, duplicateIssues...)

	// Check file permissions
	permissionIssues := a.checkKeyPermissions(keys)
	result.Issues = append(result.Issues, permissionIssues...)

	// Check for weak algorithms
	algorithmIssues := a.checkWeakAlgorithms(keys)
	result.Issues = append(result.Issues, algorithmIssues...)

	// Calculate overall score and recommendations
	result.OverallScore, result.Recommendations = a.calculateSecurityScore(result.Issues)
	result.Summary = a.generateSummary(result.OverallScore, len(result.Issues))

	a.logger.WithFields(map[string]interface{}{
		"server_id":     serverID,
		"keys_scanned":  len(keys),
		"issues_found":  len(result.Issues),
		"overall_score": result.OverallScore,
	}).Info("SSH key audit completed")

	return result, nil
}

// SSHKey represents an SSH key found during scanning
type SSHKey struct {
	Path         string       `json:"path"`
	Type         string       `json:"type"`
	Size         int          `json:"size"`
	Fingerprint  string       `json:"fingerprint"`
	Comment      string       `json:"comment"`
	Permissions  os.FileMode `json:"permissions"`
	LastModified time.Time    `json:"last_modified"`
	IsPrivate    bool         `json:"is_private"`
	Strength     KeyStrength  `json:"strength"`
}

// KeyStrength represents strength of an SSH key
type KeyStrength string

const (
	KeyStrengthWeak       KeyStrength = "weak"
	KeyStrengthFair       KeyStrength = "fair"
	KeyStrengthGood       KeyStrength = "good"
	KeyStrengthExcellent  KeyStrength = "excellent"
	KeyStrengthUnknown    KeyStrength = "unknown"
)

// scanSSHKeys scans a directory for SSH keys
func (a *Auditor) scanSSHKeys(sshDir string) ([]*SSHKey, error) {
	var keys []*SSHKey

	// Scan for common SSH key files
	keyFiles := []string{
		"id_rsa", "id_ecdsa", "id_ed25519",
		"id_dsa",
		"authorized_keys",
		"known_hosts",
	}

	for _, filename := range keyFiles {
		filePath := filepath.Join(sshDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		// Scan the file
		fileKeys, err := a.scanKeyFile(filePath, filename)
		if err != nil {
			a.logger.WithError(err).Warnf("Failed to scan key file: %s", filePath)
			continue
		}

		keys = append(keys, fileKeys...)
	}

	return keys, nil
}

// scanKeyFile scans a single SSH key file
func (a *Auditor) scanKeyFile(filePath, filename string) ([]*SSHKey, error) {
	var keys []*SSHKey

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Try to parse as SSH key
		if key := a.parseSSHKeyLine(line, filePath, info.ModTime()); key != nil {
			keys = append(keys, key)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

// parseSSHKeyLine parses a line that might contain an SSH key
func (a *Auditor) parseSSHKeyLine(line, filePath string, modTime time.Time) *SSHKey {
	// Check if it looks like a private key
	if strings.Contains(line, "BEGIN") && strings.Contains(line, "PRIVATE KEY") {
		return a.parsePrivateKey(line, filePath, modTime, true)
	}

	// Check if it looks like a public key
	if strings.Contains(line, "ssh-") || strings.Contains(line, "ssh-rsa") || strings.Contains(line, "ssh-dss") {
		return a.parsePublicKey(line, filePath, modTime, false)
	}

	return nil
}

// parsePrivateKey parses a private key line
func (a *Auditor) parsePrivateKey(line, filePath string, modTime time.Time, isPrivate bool) *SSHKey {
	// Extract key type
	var keyType string
	if strings.Contains(line, "RSA") {
		keyType = "RSA"
	} else if strings.Contains(line, "ECDSA") {
		keyType = "ECDSA"
	} else if strings.Contains(line, "ED25519") {
		keyType = "ED25519"
	} else if strings.Contains(line, "DSS") {
		keyType = "DSS"
	} else {
		keyType = "UNKNOWN"
	}

	// Get file info
	info, _ := os.Stat(filePath)

	// Determine strength based on key type
	var strength KeyStrength
	switch keyType {
	case "ED25519":
		strength = KeyStrengthExcellent
	case "ECDSA":
		strength = KeyStrengthGood
	case "RSA":
		strength = KeyStrengthFair // Would need actual key size analysis
	case "DSS":
		strength = KeyStrengthWeak
	default:
		strength = KeyStrengthUnknown
	}

	return &SSHKey{
		Path:         filePath,
		Type:         keyType,
		Strength:     strength,
		Permissions:  info.Mode(),
		LastModified: modTime,
		IsPrivate:    isPrivate,
	}
}

// parsePublicKey parses a public key line
func (a *Auditor) parsePublicKey(line, filePath string, modTime time.Time, isPrivate bool) *SSHKey {
	// Extract key type
	var keyType string
	if strings.HasPrefix(line, "ssh-rsa") {
		keyType = "RSA"
	} else if strings.HasPrefix(line, "ssh-dss") {
		keyType = "DSS"
	} else if strings.HasPrefix(line, "ssh-ed25519") {
		keyType = "ED25519"
	} else if strings.HasPrefix(line, "ecdsa-") {
		keyType = "ECDSA"
	} else {
		keyType = "UNKNOWN"
	}

	// Generate fingerprint (simplified)
	fingerprint := a.generateFingerprint(line)

	// Get file info
	info, _ := os.Stat(filePath)

	// Determine strength
	var strength KeyStrength
	switch keyType {
	case "ED25519":
		strength = KeyStrengthExcellent
	case "ECDSA":
		strength = KeyStrengthGood
	case "RSA":
		strength = KeyStrengthFair
	case "DSS":
		strength = KeyStrengthWeak
	default:
		strength = KeyStrengthUnknown
	}

	return &SSHKey{
		Path:         filePath,
		Type:         keyType,
		Fingerprint:  fingerprint,
		Strength:     strength,
		Permissions:  info.Mode(),
		LastModified: modTime,
		IsPrivate:    isPrivate,
	}
}

// generateFingerprint generates a fingerprint for a key
func (a *Auditor) generateFingerprint(keyData string) string {
	// Simplified fingerprint generation
	// In production, you'd use proper SSH key parsing
	hash := sha256.Sum256([]byte(keyData))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// analyzeKey analyzes a single SSH key for security issues
func (a *Auditor) analyzeKey(key *SSHKey) []SecurityIssue {
	var issues []SecurityIssue

	// Check key strength
	if key.Strength == KeyStrengthWeak {
		issues = append(issues, SecurityIssue{
			ID:          "WEAK_KEY_ALGORITHM",
			Severity:    SeverityHigh,
			Category:    "algorithm",
			Title:       "Weak SSH Key Algorithm",
			Description: fmt.Sprintf("SSH key uses weak algorithm: %s", key.Type),
			Impact:      "Key can be compromised with modern computing power",
			Recommendation: "Replace with ED25519 or at least 4096-bit RSA key",
			AffectedKeys: []string{key.Path},
		})
	}

	// Check file permissions
	if key.IsPrivate && key.Permissions&0077 != 0 {
		issues = append(issues, SecurityIssue{
			ID:          "PERMISSIVE_KEY_PERMISSIONS",
			Severity:    SeverityMedium,
			Category:    "permissions",
			Title:       "Permissive SSH Key File Permissions",
			Description: fmt.Sprintf("SSH private key file has permissive permissions: %o", key.Permissions),
			Impact:      "Other users can read the private key",
			Recommendation: "Set permissions to 600 (read/write by owner only)",
			AffectedKeys: []string{key.Path},
		})
	}

	// Check for old keys
	if time.Since(key.LastModified) > 365*24*time.Hour {
		issues = append(issues, SecurityIssue{
			ID:          "OLD_KEY",
			Severity:    SeverityLow,
			Category:    "age",
			Title:       "Old SSH Key",
			Description: fmt.Sprintf("SSH key is older than 1 year: %s", key.LastModified.Format("2006-01-02")),
			Impact:      "Key may have been compromised over time",
			Recommendation: "Consider rotating the SSH key",
			AffectedKeys: []string{key.Path},
		})
	}

	return issues
}

// checkDuplicateKeys checks for duplicate SSH keys
func (a *Auditor) checkDuplicateKeys(keys []*SSHKey) []SecurityIssue {
	var issues []SecurityIssue

	// Group keys by fingerprint
	fingerprints := make(map[string][]*SSHKey)
	for _, key := range keys {
		if key.Fingerprint != "" {
			fingerprints[key.Fingerprint] = append(fingerprints[key.Fingerprint], key)
		}
	}

	// Check for duplicates
	for fingerprint, keyList := range fingerprints {
		if len(keyList) > 1 {
			var affectedKeys []string
			for _, key := range keyList {
				affectedKeys = append(affectedKeys, key.Path)
			}

			issues = append(issues, SecurityIssue{
				ID:          "DUPLICATE_KEYS",
				Severity:    SeverityMedium,
				Category:    "duplication",
				Title:       "Duplicate SSH Keys",
				Description: fmt.Sprintf("Found duplicate SSH keys with fingerprint: %s", fingerprint),
				Impact:      "Key duplication can lead to confusion and potential security issues",
				Recommendation: "Remove duplicate keys and keep only unique keys",
				AffectedKeys: affectedKeys,
			})
		}
	}

	return issues
}

// checkKeyPermissions checks permissions of SSH key files
func (a *Auditor) checkKeyPermissions(keys []*SSHKey) []SecurityIssue {
	var issues []SecurityIssue

	for _, key := range keys {
		if !key.IsPrivate {
			continue // Skip public keys for permission checks
		}

		// Check if file is readable by others
		if key.Permissions&0004 != 0 {
			issues = append(issues, SecurityIssue{
				ID:          "KEY_READABLE_BY_OTHERS",
				Severity:    SeverityHigh,
				Category:    "permissions",
				Title:       "SSH Key Readable by Others",
				Description: fmt.Sprintf("SSH private key is readable by others: %o", key.Permissions),
				Impact:      "Other users can steal the private key",
				Recommendation: "Set permissions to 600 (read/write by owner only)",
				AffectedKeys: []string{key.Path},
			})
		}

		// Check if file is writable by group
		if key.Permissions&0020 != 0 {
			issues = append(issues, SecurityIssue{
				ID:          "KEY_WRITABLE_BY_GROUP",
				Severity:    SeverityMedium,
				Category:    "permissions",
				Title:       "SSH Key Writable by Group",
				Description: fmt.Sprintf("SSH private key is writable by group: %o", key.Permissions),
				Impact:      "Group members can modify the private key",
				Recommendation: "Set permissions to 600 (read/write by owner only)",
				AffectedKeys: []string{key.Path},
			})
		}
	}

	return issues
}

// checkWeakAlgorithms checks for weak cryptographic algorithms
func (a *Auditor) checkWeakAlgorithms(keys []*SSHKey) []SecurityIssue {
	var issues []SecurityIssue

	for _, key := range keys {
		switch key.Type {
		case "DSS":
			issues = append(issues, SecurityIssue{
				ID:          "DSS_KEY_ALGORITHM",
				Severity:    SeverityHigh,
				Category:    "algorithm",
				Title:       "DSS SSH Key Algorithm",
				Description: "DSS algorithm is deprecated and considered weak",
				Impact:      "DSS keys are vulnerable to attacks",
				Recommendation: "Replace with ED25519 or RSA keys",
				AffectedKeys: []string{key.Path},
			})
		case "RSA":
			// Would need to check actual key size for accurate assessment
			issues = append(issues, SecurityIssue{
				ID:          "RSA_KEY_SIZE",
				Severity:    SeverityMedium,
				Category:    "algorithm",
				Title:       "RSA Key Size",
				Description: "RSA keys should be at least 2048 bits, preferably 4096 bits",
				Impact:      "Small RSA keys can be factored with modern computing",
				Recommendation: "Generate a new RSA key with at least 4096 bits or use ED25519",
				AffectedKeys: []string{key.Path},
			})
		}
	}

	return issues
}

// calculateSecurityScore calculates overall security score
func (a *Auditor) calculateSecurityScore(issues []SecurityIssue) (SecurityScore, []string) {
	var recommendations []string
	score := 100

	// Deduct points based on issues
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityCritical:
			score -= 40
		case SeverityHigh:
			score -= 25
		case SeverityMedium:
			score -= 15
		case SeverityLow:
			score -= 5
		case SeverityInfo:
			score -= 1
		}

		// Add recommendations
		if issue.Recommendation != "" {
			recommendations = append(recommendations, issue.Recommendation)
		}
	}

	// Determine overall score
	var overallScore SecurityScore
	if score >= 90 {
		overallScore = ScoreExcellent
	} else if score >= 75 {
		overallScore = ScoreGood
	} else if score >= 60 {
		overallScore = ScoreFair
	} else if score >= 40 {
		overallScore = ScorePoor
	} else {
		overallScore = ScoreCritical
	}

	// Add general recommendations if score is low
	if overallScore == ScorePoor || overallScore == ScoreCritical {
		recommendations = append(recommendations, "Consider comprehensive SSH security review")
		recommendations = append(recommendations, "Implement regular key rotation policy")
		recommendations = append(recommendations, "Use SSH agent instead of password authentication")
	}

	return overallScore, recommendations
}

// generateSummary generates a summary of the audit results
func (a *Auditor) generateSummary(score SecurityScore, issueCount int) string {
	switch score {
	case ScoreExcellent:
		return fmt.Sprintf("Excellent security posture with %d minor issues found", issueCount)
	case ScoreGood:
		return fmt.Sprintf("Good security posture with %d issues found", issueCount)
	case ScoreFair:
		return fmt.Sprintf("Fair security posture with %d issues found - improvement recommended", issueCount)
	case ScorePoor:
		return fmt.Sprintf("Poor security posture with %d issues found - immediate attention required", issueCount)
	case ScoreCritical:
		return fmt.Sprintf("Critical security posture with %d issues found - urgent action required", issueCount)
	default:
		return fmt.Sprintf("Security assessment completed with %d issues found", issueCount)
	}
}

// SortIssuesBySeverity sorts security issues by severity
func SortIssuesBySeverity(issues []SecurityIssue) []SecurityIssue {
	severityOrder := map[Severity]int{
		SeverityCritical: 5,
		SeverityHigh:     4,
		SeverityMedium:   3,
		SeverityLow:      2,
		SeverityInfo:     1,
	}

	sort.Slice(issues, func(i, j int) bool {
		return severityOrder[issues[i].Severity] > severityOrder[issues[j].Severity]
	})

	return issues
}

// GetSeverityColor returns ANSI color code for severity
func GetSeverityColor(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "\033[31m" // Red
	case SeverityHigh:
		return "\033[91m" // Bright Red
	case SeverityMedium:
		return "\033[33m" // Yellow
	case SeverityLow:
		return "\033[93m" // Bright Yellow
	case SeverityInfo:
		return "\033[36m" // Cyan
	default:
		return "\033[0m"  // Reset
	}
}

// ResetColor resets terminal color
func ResetColor() string {
	return "\033[0m"
}