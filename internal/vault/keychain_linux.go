//go:build linux

package vault

import (
	"fmt"
	"os/exec"
	"strings"
)

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

func platformKeychain() KeychainStore {
	return &linuxKeychain{}
}
