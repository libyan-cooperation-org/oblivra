package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// CertificateInfo holds metadata about an SSH certificate
type CertificateInfo struct {
	Type            string            `json:"type"` // "user" or "host"
	KeyID           string            `json:"key_id"`
	Serial          uint64            `json:"serial"`
	ValidPrincipals []string          `json:"valid_principals"`
	ValidAfter      string            `json:"valid_after"`
	ValidBefore     string            `json:"valid_before"`
	IsExpired       bool              `json:"is_expired"`
	ExpiresIn       string            `json:"expires_in"`
	CriticalOptions map[string]string `json:"critical_options"`
	Extensions      map[string]string `json:"extensions"`
	SignatureType   string            `json:"signature_type"`
	CAFingerprint   string            `json:"ca_fingerprint"`
}

// CertificateManager handles SSH certificate operations
type CertificateManager struct{}

func NewCertificateManager() *CertificateManager {
	return &CertificateManager{}
}

// ParseCertificate parses an SSH certificate file
func (m *CertificateManager) ParseCertificate(certPath string) (*CertificateInfo, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read certificate: %w", err)
	}

	return m.ParseCertificateData(data)
}

// ParseCertificateData parses SSH certificate from bytes
func (m *CertificateManager) ParseCertificateData(data []byte) (*CertificateInfo, error) {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	cert, ok := pubKey.(*ssh.Certificate)
	if !ok {
		return nil, fmt.Errorf("not an SSH certificate")
	}

	now := time.Now()
	// G115: Prevent integer overflow when converting uint64 to int64.
	// SSH certificates use 0xffffffffffffffff for 'forever'.
	var validAfter, validBefore time.Time
	if cert.ValidAfter == 0xffffffffffffffff {
		validAfter = time.Unix(0, 0)
	} else {
		validAfter = time.Unix(int64(cert.ValidAfter), 0)
	}
	if cert.ValidBefore == 0xffffffffffffffff {
		validBefore = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	} else {
		validBefore = time.Unix(int64(cert.ValidBefore), 0)
	}

	info := &CertificateInfo{
		KeyID:           cert.KeyId,
		Serial:          cert.Serial,
		ValidPrincipals: cert.ValidPrincipals,
		ValidAfter:      validAfter.Format(time.RFC3339),
		ValidBefore:     validBefore.Format(time.RFC3339),
		IsExpired:       now.After(validBefore),
		CriticalOptions: cert.CriticalOptions,
		Extensions:      cert.Extensions,
	}

	// Determine type
	switch cert.CertType {
	case ssh.UserCert:
		info.Type = "user"
	case ssh.HostCert:
		info.Type = "host"
	}

	// Calculate expiry
	if !info.IsExpired {
		remaining := validBefore.Sub(now)
		if remaining.Hours() > 24 {
			info.ExpiresIn = fmt.Sprintf("%.0f days", remaining.Hours()/24)
		} else if remaining.Hours() > 1 {
			info.ExpiresIn = fmt.Sprintf("%.0f hours", remaining.Hours())
		} else {
			info.ExpiresIn = fmt.Sprintf("%.0f minutes", remaining.Minutes())
		}
	} else {
		info.ExpiresIn = "EXPIRED"
	}

	// CA fingerprint
	if cert.SignatureKey != nil {
		info.CAFingerprint = ssh.FingerprintSHA256(cert.SignatureKey)
	}

	info.SignatureType = cert.Signature.Format

	return info, nil
}

// ListCertificates scans for SSH certificates in ~/.ssh/
func (m *CertificateManager) ListCertificates() ([]CertificateInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sshDir := filepath.Join(home, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, err
	}

	var certs []CertificateInfo

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "-cert.pub") {
			continue
		}

		certPath := filepath.Join(sshDir, entry.Name())
		info, err := m.ParseCertificate(certPath)
		if err != nil {
			continue
		}

		certs = append(certs, *info)
	}

	return certs, nil
}

// BuildCertAuthMethod creates an SSH auth method from a certificate
func (m *CertificateManager) BuildCertAuthMethod(
	privateKeyPath string,
	certPath string,
	passphrase string,
) (ssh.AuthMethod, error) {
	// Read private key
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	// Parse private key
	var signer ssh.Signer
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	// Read certificate
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read certificate: %w", err)
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(certData)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	cert, ok := pubKey.(*ssh.Certificate)
	if !ok {
		return nil, fmt.Errorf("not a certificate")
	}

	// Create certificate signer
	certSigner, err := ssh.NewCertSigner(cert, signer)
	if err != nil {
		return nil, fmt.Errorf("create cert signer: %w", err)
	}

	return ssh.PublicKeys(certSigner), nil
}

// RequestCertificate requests a certificate from a CA (using ssh-keygen)
func (m *CertificateManager) RequestCertificate(
	caKeyPath string,
	publicKeyPath string,
	identity string,
	principals []string,
	validityDuration time.Duration,
	certType string, // "user" or "host"
) (string, error) {
	args := []string{
		"-s", caKeyPath, // CA key
		"-I", identity, // Key identity
		"-n", strings.Join(principals, ","), // Principals
		"-V", fmt.Sprintf("+%ds", int(validityDuration.Seconds())), // Validity
	}

	if certType == "host" {
		args = append(args, "-h")
	}

	args = append(args, publicKeyPath)

	cmd := exec.Command("ssh-keygen", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ssh-keygen: %s: %w", string(output), err)
	}

	// Certificate is written to publicKeyPath with -cert.pub suffix
	certPath := strings.TrimSuffix(publicKeyPath, ".pub") + "-cert.pub"
	return certPath, nil
}

// CheckExpiry returns certificates that are expiring within a threshold
func (m *CertificateManager) CheckExpiry(threshold time.Duration) ([]CertificateInfo, error) {
	certs, err := m.ListCertificates()
	if err != nil {
		return nil, err
	}

	var expiring []CertificateInfo
	cutoff := time.Now().Add(threshold)

	for _, cert := range certs {
		if parseTime(cert.ValidBefore).Before(cutoff) {
			expiring = append(expiring, cert)
		}
	}

	return expiring, nil
}

func parseTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return t
}
