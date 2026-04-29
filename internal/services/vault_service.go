package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
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
	"github.com/kingknull/oblivrashell/internal/auth"
	"golang.org/x/crypto/ssh"
)

type VaultService struct {
	BaseService
	ctx          context.Context
	vault        vault.Provider
	db           database.DatabaseStore
	analytics    *analytics.AnalyticsEngine
	searchEngine **search.SearchEngine // pointer to pointer so we can initialize it later
	federator    **search.Federator    // pointer to pointer for federated search
	creds        database.CredentialStore
	audit        database.AuditStore
	fido2Manager FIDO2Provider
	bus          *eventbus.Bus
	log          *logger.Logger
	rbac         *auth.RBACEngine
	postMu       sync.Mutex // Guard against concurrent postUnlock
}

func (s *VaultService) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *VaultService) Name() string { return "vault-service" }

// Dependencies returns service dependencies
func (s *VaultService) Dependencies() []string {
	return []string{"vault"}
}

func NewVaultService(v vault.Provider, db database.DatabaseStore, analytics *analytics.AnalyticsEngine, searchPtr **search.SearchEngine, federatorPtr **search.Federator, credRepo database.CredentialStore, auditRepo database.AuditStore, fido FIDO2Provider, rbac *auth.RBACEngine, bus *eventbus.Bus, log *logger.Logger) *VaultService {
	return &VaultService{
		vault:        v,
		db:           db,
		analytics:    analytics,
		searchEngine: searchPtr,
		federator:    federatorPtr,
		creds:        credRepo,
		audit:        auditRepo,
		fido2Manager: fido,
		rbac:         rbac,
		bus:          bus,
		log:          log.WithPrefix("vault_service"),
	}
}

// SetContext updates the Wails runtime context after DomReady.
// Called by app.Startup so EventsEmit uses the correct context.
func (s *VaultService) SetContext(ctx context.Context) {
	if ctx != nil {
		s.ctx = ctx
	}
}

// HasKeychainEntry returns true if the OS keychain has a stored vault credential,
// meaning a headless auto-unlock attempt is likely to succeed.
// Returns false on any error — never panics.
func (s *VaultService) HasKeychainEntry() bool {
	if s == nil || s.vault == nil {
		return false
	}
	defer func() { recover() }() // keychain probe must never crash the app
	return s.vault.HasKeychainEntry()
}

func (s *VaultService) Stop(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			if s.log != nil {
				s.log.Error("VaultService.Stop PANIC: %v", r)
			} else {
				fmt.Printf("VaultService.Stop PANIC (no log): %v\n", r)
			}
		}
	}()

	if s.log != nil {
		s.log.Info("Shutting down VaultService and closing databases...")
	}
	if s.analytics != nil {
		s.analytics.Close()
	}
	if s.searchEngine != nil {
		se := *s.searchEngine
		if se != nil {
			if s.log != nil {
				s.log.Debug("Closing search engine...")
			}
			se.Close()
		}
	}
	if s.db != nil {
		s.db.Close()
	}
	return nil
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
func (s *VaultService) GetPassword(ctx context.Context, id string) ([]byte, error) {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermVaultRead); err != nil {
		return nil, err
	}
	if s.vault == nil || !s.vault.IsUnlocked() {
		return nil, vault.ErrLocked
	}
	cred, err := s.creds.GetByID(ctx, id)
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
func (s *VaultService) GetPrivateKey(ctx context.Context, id string) ([]byte, string, error) {
	if s.vault == nil || !s.vault.IsUnlocked() {
		return nil, "", vault.ErrLocked
	}
	cred, err := s.creds.GetByID(ctx, id)
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
	// Normalize: treat empty slice as nil so frontend passing [] doesn't affect key derivation
	if len(hardwareKey) == 0 {
		hardwareKey = nil
	}
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

func (s *VaultService) postUnlock() (retErr error) {
	// Catch any unexpected panic so a crash inside postUnlock never terminates the app.
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("postUnlock PANIC recovered: %v", r)
			retErr = fmt.Errorf("postUnlock panic: %v", r)
		}
	}()

	// Non-blocking: if postUnlock is already running (e.g. from auto-unlock),
	// return immediately. The in-progress call will emit vault:unlocked when done.
	if !s.postMu.TryLock() {
		s.log.Info("postUnlock already in progress, skipping duplicate call")
		return nil
	}
	defer s.postMu.Unlock()

	// Guard against nil service fields — can happen if called before Start()
	if s.vault == nil {
		return fmt.Errorf("postUnlock: vault is nil")
	}
	if s.db == nil {
		return fmt.Errorf("postUnlock: database is nil")
	}

	// If database is already open, avoid redundant initialization
	if !s.db.IsLocked() {
		// Still emit the event so the frontend can transition if it missed the first one
		s.bus.Publish(eventbus.EventVaultUnlocked, nil)
		EmitEvent("vault:unlocked", nil)
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
				fmt.Printf("[DEBUG] postUnlock: setting *s.searchEngine (%p) to %p\n", s.searchEngine, se)
				*s.searchEngine = se
				if s.analytics != nil {
					s.analytics.SetSearchEngine(se)
				}
				// Initialize Federator
				if s.federator != nil {
					*s.federator = search.NewFederator(se, s.log)
					s.log.Info("[VAULT] Search federation initialized")
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
		// Audit integrity uses the global search or system context since
		// it's infrastructure. `s.ctx` is set by Start() / SetContext();
		// fall back to Background when neither has fired (unit tests, or
		// a postUnlock racing the Wails DomReady hook). Without this
		// fallback `context.WithValue(nil, ...)` inside WithGlobalSearch
		// panics with "cannot create context from nil parent" — the
		// recover above catches it but the audit tree never initialises.
		auditCtx := s.ctx
		if auditCtx == nil {
			auditCtx = context.Background()
		}
		if err := s.audit.InitIntegrity(database.WithGlobalSearch(auditCtx)); err != nil {
			s.log.Error("Failed to initialize audit integrity tree after unlock: %v", err)
		}
	}
	s.bus.Publish(eventbus.EventVaultUnlocked, nil)

	// CRITICAL: Emit to Wails frontend so the UI can load hosts and enable features.
	EmitEvent("vault:unlocked", nil)

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
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.log.Error("Failed to close database: %v", err)
		}
	}
	if s.searchEngine != nil && *s.searchEngine != nil {
		(*s.searchEngine).Close()
		*s.searchEngine = nil
	}
	if s.analytics != nil {
		s.analytics.SetSearchEngine(nil)
		s.analytics.Close()
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

func (s *VaultService) ListCredentials(ctx context.Context, typeFilter string) ([]database.Credential, error) {
	s.log.Debug("Listing credentials (filter: %s)", typeFilter)
	return s.creds.List(ctx, typeFilter)
}

func (s *VaultService) AddCredential(ctx context.Context, label, credType, rawData string) (string, error) {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermVaultWrite); err != nil {
		return "", err
	}
	if !s.vault.IsUnlocked() {
		return "", vault.ErrLocked
	}

	s.log.Info("Adding new credential: %s (%s)", label, credType)

	encrypted, err := s.vault.Encrypt([]byte(rawData))
	if err != nil {
		return "", fmt.Errorf("encrypt credential: %w", err)
	}

	id := uuid.New().String()
	now := time.Now().Format(time.RFC3339)
	cred := &database.Credential{
		ID:            id,
		Label:         label,
		Type:          credType,
		EncryptedData: encrypted,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.creds.Create(ctx, cred); err != nil {
		return "", err
	}

	s.bus.Publish(eventbus.EventCredentialCreated, id)
	return id, nil
}

func (s *VaultService) GenerateEd25519Key(ctx context.Context, label string) (string, error) {
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

	_, err = s.AddCredential(ctx, label, "key", string(privPEM))
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

func (s *VaultService) GetDecryptedCredential(ctx context.Context, id string) (string, error) {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermVaultRead); err != nil {
		return "", err
	}
	if !s.vault.IsUnlocked() {
		return "", vault.ErrLocked
	}

	cred, err := s.creds.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	decrypted, err := s.vault.Decrypt(cred.EncryptedData)
	if err != nil {
		return "", fmt.Errorf("decrypt credential: %w", err)
	}
	defer vault.ZeroSlice(decrypted)

	s.bus.Publish(eventbus.EventCredentialAccessed, map[string]string{
		"id":    id,
		"label": cred.Label,
		"type":  cred.Type,
	})

	return string(decrypted), nil
}

func (s *VaultService) DeleteCredential(ctx context.Context, id string) error {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermVaultWrite); err != nil {
		return err
	}
	s.log.Info("Deleting credential: %s", id)
	if err := s.creds.Delete(ctx, id); err != nil {
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
func (s *VaultService) UpdateCredential(ctx context.Context, id, label, credType, rawData string) error {
	if !s.vault.IsUnlocked() {
		return vault.ErrLocked
	}

	cred, err := s.creds.GetByID(ctx, id)
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

	if err := s.creds.Update(ctx, cred); err != nil {
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
func (s *VaultService) PasswordHealthAudit(ctx context.Context) ([]CredentialHealth, error) {
	if !s.vault.IsUnlocked() {
		return nil, vault.ErrLocked
	}

	creds, err := s.creds.List(ctx, "")
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

		if health.Score >= 90 {
			health.Severity = "good"
		} else if health.Score >= 70 {
			health.Severity = "warning"
		} else {
			health.Severity = "critical"
		}

		results = append(results, health)
	}

	s.log.Info("Password health audit completed: %d credentials scanned", len(results))
	return results, nil
}

// GetSystemKey derives a 32-byte key for a specific system purpose (e.g. "forensic_hmac").
// Implements api.SystemKeyProvider.
func (s *VaultService) GetSystemKey(purpose string) ([]byte, error) {
	if s.vault == nil {
		return nil, fmt.Errorf("vault provider not initialized")
	}
	return s.vault.GetSystemKey(purpose)
}
