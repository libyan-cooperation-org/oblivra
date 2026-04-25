package vault

import (
	"context"
	crand "crypto/rand"
	"crypto/sha256"
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
	ErrLocked           = errors.New("vault is locked")
	ErrNotSetup         = errors.New("vault not initialized")
	ErrWrongPassword    = errors.New("incorrect password")
	// ErrNotImplemented is returned by base Vault methods that must be overridden
	// by a concrete VaultService. SA-07: typed sentinel for misconfiguration detection.
	ErrNotImplemented   = errors.New("not implemented in base vault")
)

type Config struct {
	StorePath string
	Platform  platform.Platform
}

type metadata struct {
	Salt           []byte `json:"salt"`
	Canary         []byte `json:"canary"`
	CanaryHash     []byte `json:"canary_hash,omitempty"` // SHA256 of the plain canary
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

func (v *Vault) Name() string { return "vault" }

func (v *Vault) Dependencies() []string { return nil }

func (v *Vault) Start(ctx context.Context) error {
	v.log.Info("Vault service started (In-Process Mode)")
	return nil
}

func (v *Vault) Stop(ctx context.Context) error {
	v.Lock() // Ensure memory is wiped on stop
	return nil
}

func (v *Vault) Ping(ctx context.Context) error {
	return nil // In-process vault is always alive if the object exists
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

	// Convert password to byte slice for zeroing
	pwBytes := []byte(password)
	defer ZeroSlice(pwBytes)

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
	key := DeriveKey(pwBytes, salt)
	defer ZeroSlice(key)

	v.log.Info("Vault.SetupWithTPM: generating random canary")
	canaryPlain := make([]byte, 32)
	if _, err := crand.Read(canaryPlain); err != nil {
		return fmt.Errorf("generate canary: %w", err)
	}

	canary, err := Encrypt(key, canaryPlain)
	if err != nil {
		return err
	}

	// Hash the plain canary for constant-time verification during Unlock
	hasher := sha256.New()
	hasher.Write(canaryPlain)
	canaryHash := hasher.Sum(nil)

	meta := metadata{
		Salt:       salt,
		Canary:     canary,
		CanaryHash: canaryHash,
		TpmPcr:     pcr,
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

	pwBytes := []byte(password)
	defer ZeroSlice(pwBytes)

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

	key := DeriveKey(pwBytes, meta.Salt)
	defer ZeroSlice(key)

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
	if err != nil {
		return ErrWrongPassword
	}

	// Verify decrypted canary
	if len(meta.CanaryHash) > 0 {
		hasher := sha256.New()
		hasher.Write(decrypted)
		actualHash := hasher.Sum(nil)
		if subtle.ConstantTimeCompare(actualHash, meta.CanaryHash) != 1 {
			return ErrWrongPassword
		}
	} else {
		// Fallback for legacy static canary
		expectedCanary := []byte("oblivra")
		if subtle.ConstantTimeCompare(decrypted, expectedCanary) != 1 {
			return ErrWrongPassword
		}
			// SEC-20: One-time migration to random canary
		v.log.Info("Vault.Unlock: Migrating legacy static canary to random canary")
		canaryPlain := make([]byte, 32)
		if _, err := crand.Read(canaryPlain); err != nil {
			return fmt.Errorf("SEC-20 migration: generate canary: %w", err)
		}
		newCanary, _ := Encrypt(key, canaryPlain)
		hasher := sha256.New()
		hasher.Write(canaryPlain)
		meta.CanaryHash = hasher.Sum(nil)
		meta.Canary = newCanary
		if newData, err := json.Marshal(meta); err == nil {
			if err := os.WriteFile(filepath.Join(v.config.StorePath, "vault.json"), newData, 0600); err != nil {
				v.log.Error("[SECURITY] Failed to persist migrated vault metadata: %v", err)
			}
		}
	}

	v.masterKey = NewSecureBytesFromSlice(key)
	v.unlocked = true

	if rememberMe && v.keychain.Available() {
		// Store the key in the keychain for future auto-unlock
		if err := v.keychain.Set("master-key", key); err != nil {
			v.log.Warn("[VAULT] Failed to store master-key in keychain: %v", err)
		}
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
	if err != nil {
		return ErrWrongPassword
	}

	// Verify decrypted canary
	if len(meta.CanaryHash) > 0 {
		hasher := sha256.New()
		hasher.Write(decrypted)
		actualHash := hasher.Sum(nil)
		if subtle.ConstantTimeCompare(actualHash, meta.CanaryHash) != 1 {
			return ErrWrongPassword
		}
	} else {
		// Fallback for legacy static canary
		expectedCanary := []byte("oblivra")
		if subtle.ConstantTimeCompare(decrypted, expectedCanary) != 1 {
			return ErrWrongPassword
		}
			// SEC-20: One-time migration to random canary
		v.log.Info("Vault.UnlockWithKeychain: Migrating legacy static canary to random canary")
		canaryPlain := make([]byte, 32)
		if _, err := crand.Read(canaryPlain); err != nil {
			return fmt.Errorf("SEC-20 migration: generate canary: %w", err)
		}
		newCanary, _ := Encrypt(key, canaryPlain)
		hasher := sha256.New()
		hasher.Write(canaryPlain)
		meta.CanaryHash = hasher.Sum(nil)
		meta.Canary = newCanary
		if newData, err := json.Marshal(meta); err == nil {
			if err := os.WriteFile(filepath.Join(v.config.StorePath, "vault.json"), newData, 0600); err != nil {
				v.log.Error("[SECURITY] Failed to persist migrated vault metadata (keychain flow): %v", err)
			}
		}
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

	oldPwBytes := []byte(oldPassword)
	defer ZeroSlice(oldPwBytes)
	newPwBytes := []byte(newPassword)
	defer ZeroSlice(newPwBytes)

	// 1. Verify old password by re-deriving the key
	data, err := os.ReadFile(filepath.Join(v.config.StorePath, "vault.json"))
	if err != nil {
		return err
	}
	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}

	oldKey := DeriveKey(oldPwBytes, meta.Salt)
	defer ZeroSlice(oldKey)
	decrypted, err := Decrypt(oldKey, meta.Canary)
	if err != nil {
		return ErrWrongPassword
	}
	// Use the same verification path as Unlock: prefer CanaryHash (random canary),
	// fall back to legacy static string only for vaults that predate the random canary.
	if len(meta.CanaryHash) > 0 {
		hasher := sha256.New()
		hasher.Write(decrypted)
		if subtle.ConstantTimeCompare(hasher.Sum(nil), meta.CanaryHash) != 1 {
			return ErrWrongPassword
		}
	} else {
		if subtle.ConstantTimeCompare(decrypted, []byte("oblivra")) != 1 {
			return ErrWrongPassword
		}
	}

	// 2. Generate new salt and derive new key
	newSalt, err := GenerateSalt()
	if err != nil {
		return err
	}
	newKey := DeriveKey(newPwBytes, newSalt)
	defer ZeroSlice(newKey)

	// 3. Re-derive new random canary
	v.log.Info("Vault.RotateMasterKey: generating new random canary")
	canaryPlain := make([]byte, 32)
	if _, err := crand.Read(canaryPlain); err != nil {
		return fmt.Errorf("generate canary: %w", err)
	}

	newCanary, err := Encrypt(newKey, canaryPlain)
	if err != nil {
		return err
	}

	// Hash the new plain canary
	hasher := sha256.New()
	hasher.Write(canaryPlain)
	canaryHash := hasher.Sum(nil)

	// 4. Update metadata and save
	meta.Salt = newSalt
	meta.Canary = newCanary
	meta.CanaryHash = canaryHash
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
		if err := v.keychain.Set("master-key", newKey); err != nil {
			v.log.Warn("[VAULT] Failed to update master-key in keychain after rotation: %v", err)
		}
	}

	return nil
}

// NuclearDestruction performs a forensic wipe of the vault and all volatile secrets.
// SA-09: Note that on SSDs with wear-levelling, overwriting the file cannot guarantee
// physical erasure of the original sectors. This operation is best-effort on flash
// storage. For full assurance on SSDs, use full-disk encryption (e.g. BitLocker, FileVault)
// so that discarding the encryption key renders all data irrecoverable.
func (v *Vault) NuclearDestruction() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 1. Shred the vault file with cryptographically random data, then zeros.
	// Note: on SSDs with wear levelling this cannot guarantee physical erasure,
	// but it prevents trivial file-recovery of the ciphertext.
	vaultPath := filepath.Join(v.config.StorePath, "vault.json")
	if info, err := os.Stat(vaultPath); err == nil {
		size := info.Size()
		randomPass := make([]byte, size)
		// Fill with real random bytes for the first pass
		if _, err := crand.Read(randomPass); err == nil {
			os.WriteFile(vaultPath, randomPass, 0600)
		}
		// Second pass: zeros
		ZeroSlice(randomPass)
		os.WriteFile(vaultPath, randomPass, 0600)
		os.Remove(vaultPath)
	}

	// 2. Clear keychain
	if v.keychain.Available() {
		if err := v.keychain.Delete("master-key"); err != nil {
			v.log.Warn("[VAULT] Failed to clear keychain during nuclear destruction: %v", err)
		}
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

// HasKeychainEntry returns true if the OS keychain has a stored credential for
// this vault, meaning TryAutoUnlock / UnlockWithKeychain is likely to succeed.
func (v *Vault) HasKeychainEntry() bool {
	if v.keychain == nil || !v.keychain.Available() {
		return false
	}
	_, err := v.keychain.Get(keychainService)
	return err == nil
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

// GetTenantKey derives a 32-byte AES-256 key for a specific tenant from the master key.
func (v *Vault) GetTenantKey(tenantID string) ([]byte, error) {
	v.mu.RLock()
	if !v.unlocked || v.masterKey == nil {
		v.mu.RUnlock()
		return nil, ErrLocked
	}
	master := v.masterKey.Bytes()
	v.mu.RUnlock()

	return DeriveSubKey(master, "tenant:"+tenantID, 32)
}

// GetSystemKey derives a 32-byte key for a specific system purpose (e.g. "forensic_hmac").
func (v *Vault) GetSystemKey(purpose string) ([]byte, error) {
	v.mu.RLock()
	if !v.unlocked || v.masterKey == nil {
		v.mu.RUnlock()
		return nil, ErrLocked
	}
	master := v.masterKey.Bytes()
	v.mu.RUnlock()

	return DeriveSubKey(master, "system:"+purpose, 32)
}

func (v *Vault) GetPassword(id string) ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.unlocked {
		return nil, ErrLocked
	}
	// SA-07: Return a clearly typed sentinel error so callers can distinguish
	// "vault locked" from "method not implemented in this vault implementation".
	// Concrete VaultService implementations override this via the repo layer.
	return nil, fmt.Errorf("%w: GetPassword must be implemented by a concrete VaultService, not the base Vault", ErrNotImplemented)
}

func (v *Vault) GetPrivateKey(id string) ([]byte, string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.unlocked {
		return nil, "", ErrLocked
	}
	// SA-07: Same as GetPassword — return typed sentinel so callers can detect misconfiguration.
	return nil, "", fmt.Errorf("%w: GetPrivateKey must be implemented by a concrete VaultService, not the base Vault", ErrNotImplemented)
}
