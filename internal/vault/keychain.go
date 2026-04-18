package vault

import (
	"fmt"
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
	store := platformKeychain()
	if store != nil && store.Available() {
		return store
	}
	return &noopKeychain{}
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
