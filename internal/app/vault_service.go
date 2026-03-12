package app

import (
	"context"
	"crypto/ed25519"
	"math/big"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/search"
	"github.com/kingknull/oblivrashell/internal/vault"
	"golang.org/x/crypto/ssh"
)

type VaultService struct {
	BaseService
	ctx          context.Context
	vault        vault.Provider
	db           database.DatabaseStore
	analytics    *analytics.AnalyticsEngine
	searchEngine **search.SearchEngine // pointer to pointer so we can initialize it later
	creds        database.CredentialStore
	audit        database.AuditStore
	fido2Manager FIDO2Provider
	bus          *eventbus.Bus
	log          *logger.Logger
	postMu       sync.Mutex // Guard against concurrent postUnlock
}

func (s *VaultService) Startup(ctx context.Context) {
	s.ctx = ctx
}

func (s *VaultService) Name() string { return "VaultService" }

func NewVaultService(v vault.Provider, db database.DatabaseStore, analytics *analytics.AnalyticsEngine, searchPtr **search.SearchEngine, credRepo database.CredentialStore, auditRepo database.AuditStore, fido FIDO2Provider, bus *eventbus.Bus, log *logger.Logger) *VaultService {
	return &VaultService{
		vault:        v,
		db:           db,
		analytics:    analytics,
		searchEngine: searchPtr,
		creds:        credRepo,
		audit:        auditRepo,
		fido2Manager: fido,
		bus:          bus,
		log:          log.WithPrefix("vault_service"),
	}
}

func (s *VaultService) Shutdown() {
	s.log.Info("Shutting down VaultService and closing databases...")
	if s.analytics != nil {
		s.analytics.Close()
	}
	if s.searchEngine != nil && *s.searchEngine != nil {
		(*s.searchEngine).Close()
	}
	if s.db != nil {
		s.db.Close()
	}
	s.log.Info("VaultService shutdown complete.")
}

func (s *VaultService) IsUnlocked() bool {
	if s == nil || s.vault == nil {
		return false
	}
	result := s.vault.IsUnlocked()
	s.log.Info("IsUnlocked() called -> %v", result)
	return result
}
func (s *VaultService) UnlockWithKeychain() error { return s.TryAutoUnlock() }
func (s *VaultService) IsSetup() bool {
	if s == nil || s.vault == nil {
		return false
	}
	result := s.vault.IsSetup()
	s.log.Info("IsSetup() called -> %v", result)
	return result
}

// GetPassword retrieves a password by ID and decrypts it into a volatile, mutable byte slice.
func (s *VaultService) GetPassword(id string) ([]byte, error) {
	if s.vault == nil || !s.vault.IsUnlocked() {
		return nil, vault.ErrLocked
	}
	cred, err := s.creds.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	decrypted, err := s.vault.Decrypt(cred.EncryptedData)
	if err != nil {
		return nil, err
	}
	return decrypted, nil
}

// GetPrivateKey retrieves a private key by ID and decrypts it
func (s *VaultService) GetPrivateKey(id string) ([]byte, string, error) {
	if s.vault == nil || !s.vault.IsUnlocked() {
		return nil, "", vault.ErrLocked
	}
	cred, err := s.creds.GetByID(context.Background(), id)
	if err != nil {
		return nil, "", err
	}
	decrypted, err := s.vault.Decrypt(cred.EncryptedData)
	if err != nil {
		return nil, "", err
	}
	return decrypted, "", nil // passphrase not handled here yet
}

func (s *VaultService) Encrypt(data []byte) ([]byte, error) {
	if s.vault == nil {
		return nil, vault.ErrLocked
	}
	return s.vault.Encrypt(data)
}
func (s *VaultService) Decrypt(data []byte) ([]byte, error) {
	if s.vault == nil {
		return nil, vault.ErrLocked
	}
	return s.vault.Decrypt(data)
}
func (s *VaultService) AccessMasterKey(fn func(key []byte) error) error {
	return s.vault.AccessMasterKey(fn)
}
func (s *VaultService) GetYubiKeySerial() string { return s.vault.GetYubiKeySerial() }
func (s *VaultService) IsTPMBound() bool         { return s.vault.IsTPMBound() }
func (s *VaultService) Unlock(password string, hardwareKey []byte, rememberMe bool) error {
	s.log.Info("Unlocking vault (hardware: %v, remember: %v)", hardwareKey != nil, rememberMe)
	// If the vault is already unlocked (e.g. auto-unlock ran first), skip re-deriving
	// the key and go straight to postUnlock which will emit the event and return fast.
	if s.vault.IsUnlocked() {
		s.log.Info("Vault already unlocked, skipping re-unlock")
		return s.postUnlock()
	}
	if err := s.vault.Unlock(password, hardwareKey, rememberMe); err != nil {
		s.log.Error("vault.Unlock returned error: %v", err)
		return err
	}
	return s.postUnlock()
}

func (s *VaultService) UnlockWithPassword(password string, remember bool) error {
	s.log.Info("Unlocking vault with password (remember: %v)", remember)
	serial := s.vault.GetYubiKeySerial()
	if serial != "" {
		return fmt.Errorf("vault requires hardware security key (%s)", serial)
	}
	if err := s.vault.Unlock(password, nil, remember); err != nil {
		return err
	}
	return s.postUnlock()
}

// UnlockWithHardware unlocks the vault using a password combined with a hardware key.
// This is the Wails-facing method for hardware key unlock.
func (s *VaultService) UnlockWithHardware(password string, hardwareKey []byte, remember bool) error {
	s.log.Info("Unlocking vault with hardware key")
	serial := s.vault.GetYubiKeySerial()
	if serial == "" && s.fido2Manager != nil && !s.fido2Manager.HasCredentials() {
		return fmt.Errorf("vault is not linked to any hardware key or FIDO2 device")
	}
	if err := s.vault.Unlock(password, hardwareKey, remember); err != nil {
		return err
	}
	return s.postUnlock()
}

func (s *VaultService) postUnlock() error {
	// Non-blocking: if postUnlock is already running (e.g. from auto-unlock),
	// return immediately. The in-progress call will emit vault:unlocked when done.
	if !s.postMu.TryLock() {
		s.log.Info("postUnlock already in progress, skipping duplicate call")
		return nil
	}
	defer s.postMu.Unlock()

	// If database is already open, avoid redundant initialization
	if s.db != nil && !s.db.IsLocked() {
		// Still emit the event so the frontend can transition if it missed the first one
		EmitEvent(s.ctx, "vault:unlocked", nil)
		return nil
	}

	analyticsPath := platform.DataDir() + "/analytics.db"
	err := s.vault.AccessMasterKey(func(key []byte) error {
		if err := s.db.Open(platform.DatabasePath(), key); err != nil {
			s.log.Error("Failed to unlock database: %v", err)
			return fmt.Errorf("unlock database: %w", err)
		}
		if s.analytics != nil {
			if err := s.analytics.Open(analyticsPath, key); err != nil {
				s.log.Error("Failed to unlock analytics: %v", err)
			}
		}
		if s.searchEngine != nil {
			se, err := search.NewSearchEngine(platform.DataDir(), s.log)
			if err != nil {
				s.log.Error("Failed to initialize search engine: %v", err)
			} else {
				*s.searchEngine = se
				if s.analytics != nil {
					s.analytics.SetSearchEngine(se)
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := s.db.Migrate(); err != nil {
		s.log.Error("Failed to migrate database after unlock: %v", err)
	}
	if s.audit != nil {
		if err := s.audit.InitIntegrity(context.Background()); err != nil {
			s.log.Error("Failed to initialize audit integrity tree after unlock: %v", err)
		}
	}
	s.bus.Publish(eventbus.EventVaultUnlocked, nil)

	// CRITICAL: Emit to Wails frontend so the UI can load hosts and enable features.
	EmitEvent(s.ctx, "vault:unlocked", nil)

	return nil
}

func (s *VaultService) TryAutoUnlock() error {
	s.log.Info("Attempting auto-unlock from keychain")
	if err := s.vault.UnlockWithKeychain(); err != nil {
		return err
	}
	return s.postUnlock()
}

func (s *VaultService) Lock() {
	s.log.Info("Locking vault")
	s.vault.Lock()
	if s.searchEngine != nil && *s.searchEngine != nil {
		(*s.searchEngine).Close()
		*s.searchEngine = nil
	}
	if s.analytics != nil {
		s.analytics.SetSearchEngine(nil)
	}
	s.bus.Publish(eventbus.EventVaultLocked, nil)
}

func (s *VaultService) Setup(password string, yubiKeySerial string) error {
	s.log.Info("Setting up new vault")
	return s.vault.Setup(password, yubiKeySerial)
}

// SetupVault is a convenience alias used by the Wails frontend binding.
func (s *VaultService) SetupVault(password string) error {
	return s.Setup(password, "")
}

// SetupVaultWithYubiKey sets up the vault with a YubiKey serial.
func (s *VaultService) SetupVaultWithYubiKey(password string, serial string) error {
	return s.Setup(password, serial)
}

// ResetVault wipes the vault file so the user can set up a fresh vault.
// ALL ENCRYPTED DATA IS PERMANENTLY LOST. This is an emergency escape hatch.
func (s *VaultService) ResetVault() error {
	s.log.Warn("ResetVault called — permanently destroying vault")
	return s.vault.NuclearDestruction()
}

func (s *VaultService) NuclearDestruction() error {
	return s.vault.NuclearDestruction()
}

// SetupVaultWithTPM sets up the vault with TPM 2.0 PCR binding.
func (s *VaultService) SetupVaultWithTPM(password string, pcr int) error {
	s.log.Info("Setting up new vault with TPM binding (PCR %d)", pcr)
	return s.vault.SetupWithTPM(password, "", pcr)
}

// SetupWithTPM satisfies the vault.Provider interface.
func (s *VaultService) SetupWithTPM(password string, yubiKeySerial string, pcr int) error {
	return s.vault.SetupWithTPM(password, yubiKeySerial, pcr)
}

// Credential Management

func (s *VaultService) ListCredentials(typeFilter string) ([]database.Credential, error) {
	s.log.Debug("Listing credentials (filter: %s)", typeFilter)
	return s.creds.List(context.Background(), typeFilter)
}

func (s *VaultService) AddCredential(label, credType, rawData string) (string, error) {
	if !s.vault.IsUnlocked() {
		return "", vault.ErrLocked
	}

	s.log.Info("Adding new credential: %s (%s)", label, credType)

	encrypted, err := s.vault.Encrypt([]byte(rawData))
	if err != nil {
		return "", fmt.Errorf("encrypt credential: %w", err)
	}

	id := uuid.New().String()
	cred := &database.Credential{
		ID:            id,
		Label:         label,
		Type:          credType,
		EncryptedData: encrypted,
	}

	if err := s.creds.Create(context.Background(), cred); err != nil {
		return "", err
	}

	s.bus.Publish(eventbus.EventCredentialCreated, id)
	return id, nil
}

func (s *VaultService) GenerateEd25519Key(label string) (string, error) {
	if !s.vault.IsUnlocked() {
		return "", vault.ErrLocked
	}

	s.log.Info("Generating new Ed25519 key pair: %s", label)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	// Marshal private key to PKCS8 PEM
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", fmt.Errorf("marshal private key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	})

	_, err = s.AddCredential(label, "key", string(privPEM))
	if err != nil {
		return "", err
	}

	// Return OpenSSH format public key
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("create ssh public key: %w", err)
	}

	return string(ssh.MarshalAuthorizedKey(sshPub)), nil
}

func (s *VaultService) GetDecryptedCredential(id string) (string, error) {
	if !s.vault.IsUnlocked() {
		return "", vault.ErrLocked
	}

	cred, err := s.creds.GetByID(context.Background(), id)
	if err != nil {
		return "", err
	}

	decrypted, err := s.vault.Decrypt(cred.EncryptedData)
	if err != nil {
		return "", fmt.Errorf("decrypt credential: %w", err)
	}

	s.bus.Publish(eventbus.EventCredentialAccessed, map[string]string{
		"id":    id,
		"label": cred.Label,
		"type":  cred.Type,
	})

	return string(decrypted), nil
}

func (s *VaultService) DeleteCredential(id string) error {
	s.log.Info("Deleting credential: %s", id)
	if err := s.creds.Delete(context.Background(), id); err != nil {
		return err
	}
	s.bus.Publish(eventbus.EventCredentialDeleted, id)
	return nil
}

// GeneratePassword creates a cryptographically secure random password.
// Uses rejection sampling via math/big to eliminate modulo bias.
func (s *VaultService) GeneratePassword(length int, includeSymbols bool) string {
	if length < 8 {
		length = 16
	}
	if length > 128 {
		length = 128
	}

	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if includeSymbols {
		chars += "!@#$%^&*()-_=+[]{}|;:,.<>?"
	}

	charsetSize := big.NewInt(int64(len(chars)))
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsetSize)
		if err != nil {
			// Fallback — should never happen with crypto/rand
			n = big.NewInt(int64(i) % int64(len(chars)))
		}
		result[i] = chars[n.Int64()]
	}

	return string(result)
}

// UpdateCredential modifies an existing credential's label, type, or data
func (s *VaultService) UpdateCredential(id, label, credType, rawData string) error {
	if !s.vault.IsUnlocked() {
		return vault.ErrLocked
	}

	cred, err := s.creds.GetByID(context.Background(), id)
	if err != nil {
		return err
	}

	if label != "" {
		cred.Label = label
	}
	if credType != "" {
		cred.Type = credType
	}
	if rawData != "" {
		encrypted, err := s.vault.Encrypt([]byte(rawData))
		if err != nil {
			return fmt.Errorf("encrypt credential: %w", err)
		}
		cred.EncryptedData = encrypted
	}

	if err := s.creds.Update(context.Background(), cred); err != nil {
		return err
	}

	s.bus.Publish("credential.updated", id)
	s.log.Info("Credential updated: %s", id)
	return nil
}

// CredentialHealth represents the health status of a single credential
type CredentialHealth struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Age      int      `json:"age_days"`
	Issues   []string `json:"issues"`
	Score    int      `json:"score"`    // 0-100
	Severity string   `json:"severity"` // critical, warning, good
}

// PasswordHealthAudit scans all credentials and returns a health report
func (s *VaultService) PasswordHealthAudit() ([]CredentialHealth, error) {
	if !s.vault.IsUnlocked() {
		return nil, vault.ErrLocked
	}

	creds, err := s.creds.List(context.Background(), "")
	if err != nil {
		return nil, err
	}

	var results []CredentialHealth
	now := time.Now()

	for _, cred := range creds {
		health := CredentialHealth{
			ID:    cred.ID,
			Label: cred.Label,
			Type:  cred.Type,
			Score: 100,
		}

		// Age check
		if cred.CreatedAt != "" {
			parsed, err := time.Parse(time.RFC3339, cred.CreatedAt)
			if err == nil {
				health.Age = int(now.Sub(parsed).Hours() / 24)
			}
		}
		if health.Age > 365 {
			health.Issues = append(health.Issues, "Password older than 1 year")
			health.Score -= 30
		} else if health.Age > 180 {
			health.Issues = append(health.Issues, "Password older than 6 months")
			health.Score -= 15
		} else if health.Age > 90 {
			health.Issues = append(health.Issues, "Password older than 90 days")
			health.Score -= 5
		}

		// Decrypt and analyze (only for password type).
		// SECURITY: IIFE so defer vault.ZeroSlice runs per-iteration, not at function return.
		if cred.Type == "password" {
			func() {
				decrypted, err := s.vault.Decrypt(cred.EncryptedData)
				if err != nil {
					return
				}
				defer vault.ZeroSlice(decrypted)

				pwLen := len(decrypted)
				if pwLen < 8 {
					health.Issues = append(health.Issues, "Password too short (< 8 chars)")
					health.Score -= 40
				} else if pwLen < 12 {
					health.Issues = append(health.Issues, "Password could be longer (< 12 chars)")
					health.Score -= 10
				}

				hasUpper, hasLower, hasDigit, hasSymbol := false, false, false, false
				for _, c := range decrypted {
					switch {
					case c >= 'A' && c <= 'Z':
						hasUpper = true
					case c >= 'a' && c <= 'z':
						hasLower = true
					case c >= '0' && c <= '9':
						hasDigit = true
					default:
						hasSymbol = true
					}
				}
				if !hasUpper {
					health.Issues = append(health.Issues, "Missing uppercase letters")
					health.Score -= 10
				}
				if !hasLower {
					health.Issues = append(health.Issues, "Missing lowercase letters")
					health.Score -= 10
				}
				if !hasDigit {
					health.Issues = append(health.Issues, "Missing digits")
					health.Score -= 10
				}
				if !hasSymbol {
					health.Issues = append(health.Issues, "Missing special characters")
					health.Score -= 5
				}
			}()
		}

		if health.Score < 0 {
			health.Score = 0
		}

		if health.Score >= 80 {
			health.Severity = "good"
		} else if health.Score >= 50 {
			health.Severity = "warning"
		} else {
			health.Severity = "critical"
		}

		results = append(results, health)
	}

	s.log.Info("Password health audit completed: %d credentials scanned", len(results))
	return results, nil
}
