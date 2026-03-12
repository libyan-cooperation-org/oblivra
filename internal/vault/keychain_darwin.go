//go:build darwin

package vault

import (
	"fmt"
	"os/exec"
	"strings"
)

type macOSKeychain struct{}

func (k *macOSKeychain) Available() bool {
	_, err := exec.LookPath("security")
	return err == nil
}

func (k *macOSKeychain) Set(key string, value []byte) error {
	_ = k.Delete(key)

	cmd := exec.Command("security", "add-generic-password",
		"-s", keychainService,
		"-a", key,
		"-w", "-",
	)
	cmd.Stdin = strings.NewReader(string(value))
	if err := cmd.Run(); err != nil {
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

func platformKeychain() KeychainStore {
	return &macOSKeychain{}
}
