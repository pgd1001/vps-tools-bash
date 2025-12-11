package ssh

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"github.com/pgd1001/vps-tools/internal/server"
)

// SSHAgentAuth implements SSH agent authentication
type SSHAgentAuth struct {
	agent   agent.Agent
	agentConn agent.Conn
}

// PrivateKeyAuth implements private key authentication
type PrivateKeyAuth struct {
	key     ssh.Signer
	keyPath string
}

// PasswordAuth implements password authentication
type PasswordAuth struct {
	password string
}

// NewSSHAgentAuth creates a new SSH agent authenticator
func NewSSHAgentAuth() (*SSHAgentAuth, error) {
	// Connect to SSH agent
	conn, err := agent.New()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	// List keys from agent
	keys, err := conn.Signers()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get keys from SSH agent: %w", err)
	}

	if len(keys) == 0 {
		conn.Close()
		return nil, fmt.Errorf("no keys found in SSH agent")
	}

	return &SSHAgentAuth{
		agent:   agent.Agent{conn, keys},
		agentConn: conn,
	}, nil
}

// NewPrivateKeyAuth creates a new private key authenticator
func NewPrivateKeyAuth(keyPath string, privateKey string) (*PrivateKeyAuth, error) {
	var key ssh.Signer
	var err error

	if privateKey != "" {
		// Use provided private key content
		key, err = ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	} else if keyPath != "" {
		// Load private key from file
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}

		key, err = ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key from file: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either key path or private key content must be provided")
	}

	return &PrivateKeyAuth{
		key:     key,
		keyPath: keyPath,
	}, nil
}

// NewPasswordAuth creates a new password authenticator
func NewPasswordAuth(password string) (*PasswordAuth, error) {
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	return &PasswordAuth{
		password: password,
	}, nil
}

// Authenticate implements AuthMethod interface for SSH agent
func (a *SSHAgentAuth) Authenticate() (ssh.AuthMethod, error) {
	return ssh.PublicKeysCallback(a.agent.Signers), nil
}

// Type returns the authentication type
func (a *SSHAgentAuth) Type() string {
	return "ssh_agent"
}

// String returns string representation
func (a *SSHAgentAuth) String() string {
	return "SSH Agent Authentication"
}

// Close closes the SSH agent connection
func (a *SSHAgentAuth) Close() error {
	if a.agentConn != nil {
		return a.agentConn.Close()
	}
	return nil
}

// GetKeys returns available keys from SSH agent
func (a *SSHAgentAuth) GetKeys() ([]*agent.Key, error) {
	return a.agent.List(), nil
}

// Authenticate implements AuthMethod interface for private key
func (a *PrivateKeyAuth) Authenticate() (ssh.AuthMethod, error) {
	return ssh.PublicKeys(a.key), nil
}

// Type returns the authentication type
func (a *PrivateKeyAuth) Type() string {
	return "private_key"
}

// String returns string representation
func (a *PrivateKeyAuth) String() string {
	if a.keyPath != "" {
		return fmt.Sprintf("Private Key Authentication (%s)", a.keyPath)
	}
	return "Private Key Authentication (embedded)"
}

// GetKeyInfo returns information about the private key
func (a *PrivateKeyAuth) GetKeyInfo() (*KeyInfo, error) {
	if a.key == nil {
		return nil, fmt.Errorf("no key available")
	}

	// Get the public key
	pubKey, err := a.key.(ssh.AlgorithmSigner).PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Analyze key type and strength
	keyType := pubKey.Type()
	var strength KeyStrength
	var keySize int

	switch keyType {
	case ssh.KeyAlgoRSA:
		if rsaKey, ok := a.key.(*ssh.SignerImpl).PrivateKey.(*rsa.PrivateKey); ok {
			keySize = rsaKey.N.BitLen()
			strength = analyzeRSAKeyStrength(keySize)
		}
	case ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521:
		strength = KeyStrengthGood
	case ssh.KeyAlgoED25519:
		strength = KeyStrengthExcellent
	default:
		strength = KeyStrengthUnknown
	}

	return &KeyInfo{
		Type:     keyType,
		Size:     keySize,
		Strength: strength,
		Fingerprint: ssh.FingerprintSHA256(pubKey),
		KeyPath:  a.keyPath,
	}, nil
}

// Authenticate implements AuthMethod interface for password
func (a *PasswordAuth) Authenticate() (ssh.AuthMethod, error) {
	return ssh.Password(a.password), nil
}

// Type returns the authentication type
func (a *PasswordAuth) Type() string {
	return "password"
}

// String returns string representation
func (a *PasswordAuth) String() string {
	return "Password Authentication"
}

// KeyInfo represents information about an SSH key
type KeyInfo struct {
	Type        string      `json:"type"`
	Size        int         `json:"size"`
	Strength    KeyStrength `json:"strength"`
	Fingerprint string      `json:"fingerprint"`
	KeyPath     string      `json:"key_path,omitempty"`
}

// KeyStrength represents the strength of an SSH key
type KeyStrength string

const (
	KeyStrengthWeak       KeyStrength = "weak"
	KeyStrengthFair       KeyStrength = "fair"
	KeyStrengthGood       KeyStrength = "good"
	KeyStrengthExcellent  KeyStrength = "excellent"
	KeyStrengthUnknown    KeyStrength = "unknown"
)

// analyzeRSAKeyStrength analyzes the strength of an RSA key
func analyzeRSAKeyStrength(keySize int) KeyStrength {
	if keySize < 2048 {
		return KeyStrengthWeak
	} else if keySize < 3072 {
		return KeyStrengthFair
	} else if keySize < 4096 {
		return KeyStrengthGood
	} else {
		return KeyStrengthExcellent
	}
}

// GenerateKeyPair generates a new SSH key pair
func GenerateKeyPair(keyType string, keySize int, comment string) (*KeyPair, error) {
	var privateKey string
	var publicKey string
	var err error

	switch strings.ToLower(keyType) {
	case "rsa":
		privateKey, publicKey, err = generateRSAKeyPair(keySize, comment)
	case "ed25519":
		privateKey, publicKey, err = generateED25519KeyPair(comment)
	case "ecdsa":
		privateKey, publicKey, err = generateECDSAKeyPair(keySize, comment)
	default:
		return nil, fmt.Errorf("unsupported key type: %s (supported: rsa, ed25519, ecdsa)", keyType)
	}

	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Type:       strings.ToLower(keyType),
		Size:       keySize,
		Comment:    comment,
		CreatedAt:  time.Now(),
	}, nil
}

// generateRSAKeyPair generates an RSA key pair
func generateRSAKeyPair(keySize int, comment string) (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Generate private key PEM
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Headers: map[string]string{
			"Comment": comment,
		},
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	publicKeyStr := string(publicKeyBytes)

	if comment != "" {
		publicKeyStr += fmt.Sprintf(" %s", comment)
	}

	return string(privateKeyBytes), publicKeyStr, nil
}

// generateED25519KeyPair generates an ED25519 key pair
func generateED25519KeyPair(comment string) (string, string, error) {
	// ED25519 key generation
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ED25519 key: %w", err)
	}

	// Generate private key PEM
	privateKeyBytes, err := x509.MarshalED25519PrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Headers: map[string]string{
			"Comment": comment,
		},
		Bytes: privateKeyBytes,
	}
	privateKeyStr := string(pem.EncodeToMemory(privateKeyPEM))

	// Generate public key
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate SSH public key: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(sshPublicKey)
	publicKeyStr := string(publicKeyBytes)

	if comment != "" {
		publicKeyStr += fmt.Sprintf(" %s", comment)
	}

	return privateKeyStr, publicKeyStr, nil
}

// generateECDSAKeyPair generates an ECDSA key pair
func generateECDSAKeyPair(keySize int, comment string) (string, string, error) {
	// ECDSA key generation
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	// Generate private key PEM
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Headers: map[string]string{
			"Comment": comment,
		},
		Bytes: privateKeyBytes,
	}
	privateKeyStr := string(pem.EncodeToMemory(privateKeyPEM))

	// Generate public key
	sshPublicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate SSH public key: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(sshPublicKey)
	publicKeyStr := string(publicKeyBytes)

	if comment != "" {
		publicKeyStr += fmt.Sprintf(" %s", comment)
	}

	return privateKeyStr, publicKeyStr, nil
}

// KeyPair represents a generated SSH key pair
type KeyPair struct {
	PrivateKey string    `json:"private_key"`
	PublicKey  string    `json:"public_key"`
	Type       string    `json:"type"`
	Size       int       `json:"size"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

// SaveKeyPair saves a key pair to files
func (kp *KeyPair) SaveKeyPair(privateKeyPath, publicKeyPath string) error {
	// Save private key
	if err := os.WriteFile(privateKeyPath, []byte(kp.PrivateKey), 0600); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	// Save public key
	if err := os.WriteFile(publicKeyPath, []byte(kp.PublicKey), 0644); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	return nil
}

// LoadAuthorizedKeys loads authorized keys from a file
func LoadAuthorizedKeys(authorizedKeysPath string) ([]*AuthorizedKey, error) {
	file, err := os.Open(authorizedKeysPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open authorized keys file: %w", err)
	}
	defer file.Close()

	var keys []*AuthorizedKey
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key line
		key, err := parseAuthorizedKeyLine(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
		}

		key.LineNumber = lineNum
		keys = append(keys, key)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading authorized keys file: %w", err)
	}

	return keys, nil
}

// parseAuthorizedKeyLine parses a single line from authorized_keys file
func parseAuthorizedKeyLine(line string) (*AuthorizedKey, error) {
	// Simple parsing - in production, you'd want more robust parsing
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid authorized key format")
	}

	key := &AuthorizedKey{
		KeyType:   parts[0],
		KeyData:   parts[1],
		RawLine:   line,
	}

	// Extract options if present
	if strings.HasPrefix(parts[0], "ssh-") {
		// This is a simplified parser - real implementation would be more complex
		key.Options = parts[:len(parts)-2]
		key.KeyType = parts[len(parts)-2]
		key.KeyData = parts[len(parts)-1]
	}

	return key, nil
}

// AuthorizedKey represents a key in authorized_keys file
type AuthorizedKey struct {
	KeyType   string   `json:"key_type"`
	KeyData   string   `json:"key_data"`
	Options   []string `json:"options,omitempty"`
	Comment   string   `json:"comment,omitempty"`
	RawLine   string   `json:"raw_line"`
	LineNumber int      `json:"line_number"`
}

// ValidateKeyPermissions validates SSH key file permissions
func ValidateKeyPermissions(keyPath string) error {
	// Check file permissions
	info, err := os.Stat(keyPath)
	if err != nil {
		return fmt.Errorf("failed to stat key file: %w", err)
	}

	mode := info.Mode().Perm()
	
	// Private key should be 600 or more restrictive
	if mode&0077 != 0 {
		return fmt.Errorf("private key file has too permissive permissions: %o", mode)
	}

	// Directory should not be writable by others
	dir := filepath.Dir(keyPath)
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("failed to stat key directory: %w", err)
	}

	dirMode := dirInfo.Mode().Perm()
	if dirMode&0002 != 0 {
		return fmt.Errorf("key directory is writable by others: %o", dirMode)
	}

	return nil
}

// FixKeyPermissions fixes SSH key file permissions
func FixKeyPermissions(keyPath string) error {
	// Set file permissions to 600
	if err := os.Chmod(keyPath, 0600); err != nil {
		return fmt.Errorf("failed to set key file permissions: %w", err)
	}

	// Set directory permissions to 700
	dir := filepath.Dir(keyPath)
	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("failed to set directory permissions: %w", err)
	}

	return nil
}