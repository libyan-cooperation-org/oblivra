package vault

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

const keychainService = "sovereign-terminal"

// KeychainStore abstracts OS keychain operations
type KeychainStore interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	Available() bool
}

// GetKeychainStore returns the appropriate keychain for the OS
func GetKeychainStore() KeychainStore {
	switch runtime.GOOS {
	case "darwin":
		return &macOSKeychain{}
	case "linux":
		return &linuxKeychain{}
	case "windows":
		return &windowsKeychain{}
	default:
		return &noopKeychain{}
	}
}

// macOS Keychain implementation using 'security' CLI
type macOSKeychain struct{}

func (k *macOSKeychain) Available() bool {
	_, err := exec.LookPath("security")
	return err == nil
}

func (k *macOSKeychain) Set(key string, value []byte) error {
	// First delete any existing entry (security add-generic-password -U can be flaky)
	_ = k.Delete(key)

	// Use -w with stdin pipe to avoid password exposure in process list
	cmd := exec.Command("security", "add-generic-password",
		"-s", keychainService,
		"-a", key,
		"-w", "-", // read from stdin
	)
	cmd.Stdin = strings.NewReader(string(value))
	if err := cmd.Run(); err != nil {
		// Fallback: some macOS versions don't support -w - for stdin
		cmd2 := exec.Command("security", "add-generic-password",
			"-s", keychainService,
			"-a", key,
			"-w", string(value),
			"-U",
		)
		return cmd2.Run()
	}
	return nil
}

func (k *macOSKeychain) Get(key string) ([]byte, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", keychainService,
		"-a", key,
		"-w",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("keychain get: %w", err)
	}
	return []byte(strings.TrimSpace(string(output))), nil
}

func (k *macOSKeychain) Delete(key string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", keychainService,
		"-a", key,
	)
	return cmd.Run()
}

// Linux Secret Service implementation using 'secret-tool'
type linuxKeychain struct{}

func (k *linuxKeychain) Available() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

func (k *linuxKeychain) Set(key string, value []byte) error {
	cmd := exec.Command("secret-tool", "store",
		"--label", fmt.Sprintf("Sovereign Terminal: %s", key),
		"service", keychainService,
		"account", key,
	)
	cmd.Stdin = strings.NewReader(string(value))
	return cmd.Run()
}

func (k *linuxKeychain) Get(key string) ([]byte, error) {
	cmd := exec.Command("secret-tool", "lookup",
		"service", keychainService,
		"account", key,
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("secret-tool lookup: %w", err)
	}
	return output, nil
}

func (k *linuxKeychain) Delete(key string) error {
	cmd := exec.Command("secret-tool", "clear",
		"service", keychainService,
		"account", key,
	)
	return cmd.Run()
}

// noopKeychain for unsupported platforms
type noopKeychain struct{}

func (k *noopKeychain) Available() bool { return false }
func (k *noopKeychain) Set(key string, value []byte) error {
	return fmt.Errorf("no keychain available")
}
func (k *noopKeychain) Get(key string) ([]byte, error) {
	return nil, fmt.Errorf("no keychain available")
}
func (k *noopKeychain) Delete(key string) error { return fmt.Errorf("no keychain available") }
