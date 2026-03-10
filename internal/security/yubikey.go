package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// YubiKeyInfo holds information about a connected YubiKey
type YubiKeyInfo struct {
	Serial      string `json:"serial"`
	Version     string `json:"version"`
	Model       string `json:"model"`
	PIVEnabled  bool   `json:"piv_enabled"`
	FIDOEnabled bool   `json:"fido_enabled"`
}

// YubiKeyManager handles YubiKey operations
type YubiKeyManager struct {
	mu sync.RWMutex
}

func NewYubiKeyManager() *YubiKeyManager {
	return &YubiKeyManager{}
}

// Detect checks for connected YubiKeys
func (m *YubiKeyManager) Detect() ([]YubiKeyInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try ykman (YubiKey Manager CLI)
	if isCommandAvailable("ykman") {
		return m.detectWithYkman()
	}

	// Fallback: try to detect via USB device listing (omitted for brevity)
	return nil, fmt.Errorf("ykman not installed; install via: pip install yubikey-manager")
}

func (m *YubiKeyManager) detectWithYkman() ([]YubiKeyInfo, error) {
	output, err := exec.Command("ykman", "list", "--serials").Output()
	if err != nil {
		return nil, fmt.Errorf("ykman list: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var keys []YubiKeyInfo

	for _, serial := range lines {
		serial = strings.TrimSpace(serial)
		if serial == "" {
			continue
		}

		info := YubiKeyInfo{
			Serial: serial,
		}

		// Get more info
		infoOutput, err := exec.Command("ykman", "--device", serial, "info").Output()
		if err == nil {
			infoStr := string(infoOutput)
			if strings.Contains(infoStr, "PIV") {
				info.PIVEnabled = true
			}
			if strings.Contains(infoStr, "FIDO2") {
				info.FIDOEnabled = true
			}
			// Parse version
			for _, line := range strings.Split(infoStr, "\n") {
				if strings.HasPrefix(line, "Firmware version:") {
					info.Version = strings.TrimSpace(strings.TrimPrefix(line, "Firmware version:"))
				}
				if strings.HasPrefix(line, "Device type:") {
					info.Model = strings.TrimSpace(strings.TrimPrefix(line, "Device type:"))
				}
			}
		}

		keys = append(keys, info)
	}

	return keys, nil
}

// GenerateSSHKey generates an SSH key on the YubiKey PIV slot
func (m *YubiKeyManager) GenerateSSHKey(serial string, slot string, pin string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if slot == "" {
		slot = "9a" // Default PIV authentication slot
	}

	// Generate key on device
	cmd := exec.Command("ykman", "--device", serial,
		"piv", "keys", "generate",
		"--algorithm", "ECCP256",
		"--pin-policy", "ONCE",
		"--touch-policy", "CACHED",
		slot, "-")

	cmd.Stdin = strings.NewReader(pin + "\n")
	pubKeyOutput, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	return string(pubKeyOutput), nil
}

// GetSSHPublicKey extracts the SSH public key from a YubiKey PIV slot
func (m *YubiKeyManager) GetSSHPublicKey(serial string, slot string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if slot == "" {
		slot = "9a"
	}

	output, err := exec.Command("ykman", "--device", serial,
		"piv", "keys", "export", slot, "-").Output()
	if err != nil {
		return "", fmt.Errorf("export key: %w", err)
	}

	// Convert PEM to SSH format
	sshPubKey, err := exec.Command("ssh-keygen", "-i", "-m", "PKCS8", "-f", "/dev/stdin").
		CombinedOutput()
	if err != nil {
		// Return PEM format instead
		return string(output), nil
	}

	return string(sshPubKey), nil
}

// ChallengeResponse performs a challenge-response authentication
func (m *YubiKeyManager) ChallengeResponse(serial string, challenge []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use HMAC-SHA1 challenge-response (slot 2)
	challengeHex := hex.EncodeToString(challenge)

	output, err := exec.Command("ykchalresp", "-2", "-H", challengeHex).Output()
	if err != nil {
		return nil, fmt.Errorf("challenge-response: %w", err)
	}

	response, err := hex.DecodeString(strings.TrimSpace(string(output)))
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return response, nil
}

// DeriveVaultKey derives an encryption key using YubiKey + password
func (m *YubiKeyManager) DeriveVaultKey(serial string, password string) ([]byte, error) {
	// Create challenge from password
	passwordHash := sha256.Sum256([]byte(password))

	// Get YubiKey response
	response, err := m.ChallengeResponse(serial, passwordHash[:])
	if err != nil {
		return nil, fmt.Errorf("yubikey challenge: %w", err)
	}

	// Combine password hash and YubiKey response
	combined := append(passwordHash[:], response...)
	finalKey := sha256.Sum256(combined)

	return finalKey[:], nil
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
