package vault

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/security"
)

var (
	ErrLocked        = errors.New("vault is locked")
	ErrNotSetup      = errors.New("vault not initialized")
	ErrWrongPassword = errors.New("incorrect password")
)

type Config struct {
	StorePath string
	Platform  platform.Platform
}

type metadata struct {
	Salt           []byte `json:"salt"`
	Canary         []byte `json:"canary"`
	YubiKeySerial  string `json:"yubikey_serial,omitempty"`
	TpmPcr         int    `json:"tpm_pcr,omitempty"`
	TpmFingerprint []byte `json:"tpm_fingerprint,omitempty"`
}

type Vault struct {
	mu        sync.RWMutex
	config    Config
	log       *logger.Logger
	keychain  KeychainStore
	unlocked  bool
	masterKey *SecureBytes
}

func New(cfg Config, log *logger.Logger) (*Vault, error) {
	return &Vault{
		config:   cfg,
		log:      log.WithPrefix("vault"),
		keychain: GetKeychainStore(),
	}, nil
}

func (v *Vault) IsSetup() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	_, err := os.Stat(filepath.Join(v.config.StorePath, "vault.json"))
	return err == nil
}

func (v *Vault) Setup(password string, yubiKeySerial string) error {
	return v.SetupWithTPM(password, yubiKeySerial, -1)
}

func (v *Vault) SetupWithTPM(password string, yubiKeySerial string, pcr int) error {
	v.log.Info("Vault.SetupWithTPM: started")
	v.mu.Lock()
	defer v.mu.Unlock()

	v.log.Info("Vault.SetupWithTPM: acquired lock, creating directory: %s", v.config.StorePath)
	if err := os.MkdirAll(v.config.StorePath, 0700); err != nil {
		return err
	}

	v.log.Info("Vault.SetupWithTPM: generating salt")
	salt, err := GenerateSalt()
	if err != nil {
		return err
	}

	v.log.Info("Vault.SetupWithTPM: deriving key")
	key := DeriveKey(password, salt)
	v.log.Info("Vault.SetupWithTPM: encrypting canary")
	canary, err := Encrypt(key, []byte("oblivra"))
	if err != nil {
		return err
	}

	meta := metadata{
		Salt:          salt,
		Canary:        canary,
		YubiKeySerial: yubiKeySerial,
		TpmPcr:        pcr,
	}

	// If TPM is requested, bind the vault to the current PCR state
	if pcr >= 0 {
		tpm, err := security.NewTPMManager(v.log)
		if err == nil {
			defer tpm.Close()
			fingerprint, err := tpm.GetPCRValue(pcr)
			if err == nil {
				meta.TpmFingerprint = fingerprint
			}
		}
	}

	v.log.Info("Vault.SetupWithTPM: marshaling meta")
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	v.log.Info("Vault.SetupWithTPM: writing vault.json")
	err = os.WriteFile(filepath.Join(v.config.StorePath, "vault.json"), data, 0600)
	v.log.Info("Vault.SetupWithTPM: finished with err=%v", err)
	return err
}

func (v *Vault) Unlock(password string, hardwareKey []byte, rememberMe bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := os.ReadFile(filepath.Join(v.config.StorePath, "vault.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotSetup
		}
		return err
	}

	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}

	key := DeriveKey(password, meta.Salt)

	// 3. Hardware-Rooted Integrity Verification (TPM)
	if meta.TpmPcr >= 0 && len(meta.TpmFingerprint) > 0 {
		tpm, err := security.NewTPMManager(v.log)
		if err != nil {
			return fmt.Errorf("TPM verification failed: hardware not accessible: %w", err)
		}
		defer tpm.Close()

		currentFingerprint, err := tpm.GetPCRValue(meta.TpmPcr)
		if err != nil {
			return fmt.Errorf("TPM verification failed: could not read PCR %d: %w", meta.TpmPcr, err)
		}

		// Strictly compare current PCR state with stored fingerprint
		if subtle.ConstantTimeCompare(currentFingerprint, meta.TpmFingerprint) != 1 {
			return fmt.Errorf("TPM PCR MISMATCH: system integrity compromised or environment changed")
		}
	}

	decrypted, err := Decrypt(key, meta.Canary)
	expectedCanary := []byte("oblivra")
	if err != nil || subtle.ConstantTimeCompare(decrypted, expectedCanary) != 1 {
		return ErrWrongPassword
	}

	v.masterKey = NewSecureBytesFromSlice(key)
	v.unlocked = true

	if rememberMe && v.keychain.Available() {
		// Store the key in the keychain for future auto-unlock
		_ = v.keychain.Set("master-key", key)
	}

	return nil
}

func (v *Vault) GetYubiKeySerial() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	data, err := os.ReadFile(filepath.Join(v.config.StorePath, "vault.json"))
	if err != nil {
		return ""
	}
	var meta metadata
	json.Unmarshal(data, &meta)
	return meta.YubiKeySerial
}

func (v *Vault) IsTPMBound() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	data, err := os.ReadFile(filepath.Join(v.config.StorePath, "vault.json"))
	if err != nil {
		return false
	}
	var meta metadata
	json.Unmarshal(data, &meta)
	return meta.TpmPcr >= 0 && len(meta.TpmFingerprint) > 0
}

func (v *Vault) UnlockWithKeychain() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.keychain.Available() {
		return errors.New("keychain not available")
	}

	key, err := v.keychain.Get("master-key")
	if err != nil {
		return err
	}

	// Verify key with canary
	data, err := os.ReadFile(filepath.Join(v.config.StorePath, "vault.json"))
	if err != nil {
		return err
	}

	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}

	decrypted, err := Decrypt(key, meta.Canary)
	expectedCanary := []byte("oblivra")
	if err != nil || subtle.ConstantTimeCompare(decrypted, expectedCanary) != 1 {
		return ErrWrongPassword
	}

	v.masterKey = NewSecureBytesFromSlice(key)
	v.unlocked = true
	return nil
}

// Lock clears the master key from memory and locks the vault.
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.masterKey != nil {
		v.masterKey.Release()
		v.masterKey = nil
	}
	v.unlocked = false
}

// RotateMasterKey changes the master password and re-encrypts the vault metadata.
func (v *Vault) RotateMasterKey(oldPassword, newPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.unlocked {
		return ErrLocked
	}

	// 1. Verify old password by re-deriving the key
	data, err := os.ReadFile(filepath.Join(v.config.StorePath, "vault.json"))
	if err != nil {
		return err
	}
	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}

	oldKey := DeriveKey(oldPassword, meta.Salt)
	decrypted, err := Decrypt(oldKey, meta.Canary)
	expectedCanary := []byte("oblivra")
	if err != nil || subtle.ConstantTimeCompare(decrypted, expectedCanary) != 1 {
		return ErrWrongPassword
	}

	// 2. Generate new salt and derive new key
	newSalt, err := GenerateSalt()
	if err != nil {
		return err
	}
	newKey := DeriveKey(newPassword, newSalt)

	// 3. Re-encrypt canary
	newCanary, err := Encrypt(newKey, []byte("oblivra"))
	if err != nil {
		return err
	}

	// 4. Update metadata and save
	meta.Salt = newSalt
	meta.Canary = newCanary
	newData, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(v.config.StorePath, "vault.json"), newData, 0600); err != nil {
		return err
	}

	// 5. Update active master key
	if v.masterKey != nil {
		v.masterKey.Release()
	}
	v.masterKey = NewSecureBytesFromSlice(newKey)

	// 6. Update keychain if active
	if v.keychain.Available() {
		_ = v.keychain.Set("master-key", newKey)
	}

	return nil
}

// NuclearDestruction performs a forensic wipe of the vault and all volatile secrets.
func (v *Vault) NuclearDestruction() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 1. Shred the vault file
	vaultPath := filepath.Join(v.config.StorePath, "vault.json")
	if info, err := os.Stat(vaultPath); err == nil {
		// Securely overwrite with random data then zeros before deleting
		size := info.Size()
		junk := make([]byte, size)
		os.WriteFile(vaultPath, junk, 0600) // junk (randomish if initialized)
		ZeroSlice(junk)
		os.WriteFile(vaultPath, junk, 0600) // true zeros
		os.Remove(vaultPath)
	}

	// 2. Clear keychain
	if v.keychain.Available() {
		_ = v.keychain.Delete("master-key")
	}

	// 3. Purge memory
	if v.masterKey != nil {
		v.masterKey.Release()
		v.masterKey = nil
	}
	v.unlocked = false

	return nil
}

func (v *Vault) IsUnlocked() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.unlocked
}

func (v *Vault) Encrypt(data []byte) ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.unlocked {
		return nil, ErrLocked
	}
	return Encrypt(v.masterKey.Bytes(), data)
}

func (v *Vault) Decrypt(data []byte) ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.unlocked {
		return nil, ErrLocked
	}
	return Decrypt(v.masterKey.Bytes(), data)
}

// AccessMasterKey provides temporary access to the master key for database initialization.
// The key must not be stored or used outside the callback.
// We copy the key bytes under RLock, then release the lock before calling fn so that
// slow I/O inside fn (e.g. db.Open) does not block concurrent Unlock calls.
func (v *Vault) AccessMasterKey(fn func(key []byte) error) error {
	v.mu.RLock()
	if !v.unlocked || v.masterKey == nil {
		v.mu.RUnlock()
		return ErrLocked
	}
	// Copy the key bytes so we can release the lock before the (potentially slow) callback
	keyBytes := v.masterKey.Bytes()
	keyCopy := make([]byte, len(keyBytes))
	copy(keyCopy, keyBytes)
	v.mu.RUnlock()

	err := fn(keyCopy)
	// Zero the copy when done
	for i := range keyCopy {
		keyCopy[i] = 0
	}
	return err
}

func (v *Vault) GetPassword(id string) ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.unlocked {
		return nil, ErrLocked
	}
	// Note: Vault itself doesn't have the DB/Repo.
	// This method is primarily for the Provider interface.
	// Actual retrieval happens in VaultService or SSHService using Decrypt.
	return nil, errors.New("method not implemented in base vault")
}

func (v *Vault) GetPrivateKey(id string) ([]byte, string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.unlocked {
		return nil, "", ErrLocked
	}
	return nil, "", errors.New("method not implemented in base vault")
}
