package services

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kingknull/oblivra/internal/vault"
)

// VaultService is a thin wrapper around the vault package that fits the rest
// of the service registry. It tracks locked/unlocked state and surfaces a
// minimal API to the UI: status, unlock, lock, names, set, delete.
type VaultService struct {
	log  *slog.Logger
	dir  string
	mu   sync.RWMutex
	open *vault.Vault
}

func NewVaultService(log *slog.Logger, dataDir string) *VaultService {
	return &VaultService{log: log, dir: dataDir}
}

func (s *VaultService) ServiceName() string { return "VaultService" }

func (s *VaultService) path() string { return filepath.Join(s.dir, "oblivra.vault") }

type VaultStatus struct {
	Exists   bool   `json:"exists"`
	Unlocked bool   `json:"unlocked"`
	Names    []string `json:"names,omitempty"`
}

func (s *VaultService) Status() VaultStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := VaultStatus{}
	if _, err := os.Stat(s.path()); err == nil {
		st.Exists = true
	}
	if s.open != nil {
		st.Unlocked = true
		st.Names = s.open.Names()
	}
	return st
}

// Initialize creates a brand-new vault. Errors if one already exists.
func (s *VaultService) Initialize(passphrase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := vault.Create(s.path(), passphrase)
	if err != nil {
		return err
	}
	s.open = v
	s.log.Info("vault initialised", "path", s.path())
	return nil
}

// Unlock opens the existing vault. Returns ErrInvalidKey on bad passphrase.
func (s *VaultService) Unlock(passphrase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open != nil {
		return nil
	}
	v, err := vault.Open(s.path(), passphrase)
	if err != nil {
		return err
	}
	s.open = v
	return nil
}

func (s *VaultService) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open != nil {
		s.open.Lock()
		s.open = nil
	}
}

func (s *VaultService) Set(name, value string) error {
	s.mu.RLock()
	v := s.open
	s.mu.RUnlock()
	if v == nil {
		return errors.New("vault locked")
	}
	return v.Set(name, value)
}

func (s *VaultService) Get(name string) (string, error) {
	s.mu.RLock()
	v := s.open
	s.mu.RUnlock()
	if v == nil {
		return "", errors.New("vault locked")
	}
	val, ok := v.Get(name)
	if !ok {
		return "", errors.New("not found")
	}
	return val, nil
}

func (s *VaultService) Delete(name string) error {
	s.mu.RLock()
	v := s.open
	s.mu.RUnlock()
	if v == nil {
		return errors.New("vault locked")
	}
	return v.Delete(name)
}

