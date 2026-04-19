package services

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/gdpr"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/vault"
)

type SettingsService struct {
	BaseService
	db        database.DatabaseStore
	vault     vault.Provider
	bus       *eventbus.Bus
	log       *logger.Logger
	destroyer *gdpr.DataDestructionService
}

func (s *SettingsService) Name() string { return "settings-service" }

// Dependencies returns service dependencies.
func (s *SettingsService) Dependencies() []string {
	return []string{}
}

func (s *SettingsService) Start(ctx context.Context) error {
	return nil
}

func (s *SettingsService) Stop(ctx context.Context) error {
	return nil
}

func NewSettingsService(db database.DatabaseStore, v vault.Provider, bus *eventbus.Bus, log *logger.Logger, destroyer *gdpr.DataDestructionService) *SettingsService {
	return &SettingsService{
		db:        db,
		vault:     v,
		bus:       bus,
		log:       log.WithPrefix("settings"),
		destroyer: destroyer,
	}
}

// isSensitiveKey returns true for settings that must be vault-encrypted at rest.
func isSensitiveKey(key string) bool {
	sensitiveKeys := map[string]bool{
		"smtp_password":   true,
		"api_key":         true,
		"secret_key":      true,
		"vault_key":       true,
		"token":           true,
		"slack_webhook":   true,
		"discord_webhook": true,
	}
	return sensitiveKeys[key]
}

func (s *SettingsService) Get(key string) (string, error) {
	if s.db == nil || s.db.DB() == nil {
		return "", database.ErrLocked
	}
	var val string
	err := s.db.DB().QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		return "", err
	}
	// SEC-24: Decrypt sensitive values that were encrypted via vault
	if isSensitiveKey(key) && s.vault != nil && s.vault.IsUnlocked() {
		if strings.HasPrefix(val, "v1:") {
			if decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(val, "v1:")); err == nil {
				if plaintext, err := s.vault.Decrypt(decoded); err == nil {
					return string(plaintext), nil
				}
			}
		}
	}
	return val, nil
}

func (s *SettingsService) Set(key string, value string) error {
	if s.db == nil || s.db.DB() == nil {
		return database.ErrLocked
	}

	displayValue := value
	if isSensitiveKey(key) {
		displayValue = "[REDACTED]"
		// SEC-24: Encrypt sensitive values before persisting to SQLite
		if s.vault != nil && s.vault.IsUnlocked() {
			if ciphertext, err := s.vault.Encrypt([]byte(value)); err == nil {
				value = "v1:" + base64.StdEncoding.EncodeToString(ciphertext)
			}
		}
	}

	s.log.Debug("Setting setting: %s=%s", key, displayValue)
	_, err := s.db.DB().Exec("INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)", key, value)
	if err == nil {
		s.bus.Publish(eventbus.EventSettingsChanged, key)
	}
	return err
}

func (s *SettingsService) GetConfig() (*AppConfig, error)  { return LoadConfig() }
func (s *SettingsService) SaveConfig(cfg *AppConfig) error { return SaveConfig(cfg) }

func (s *SettingsService) ClearDatabase() error {
	s.log.Info("Executing DoD-compliant CryptoWipe on all database tables...")

	// SEC-28: Hard-coded allowlist prevents SQL injection via table name concatenation
	allowedTables := map[string]bool{
		"hosts":       true,
		"snippets":    true,
		"settings":    true,
		"metrics":     true,
		"recordings":  true,
		"notes":       true,
		"sessions":    true,
		"audit_logs":  true,
		"siem_events": true,
	}

	tables := []string{
		"hosts", "snippets", "settings", "metrics",
		"recordings", "notes", "sessions", "audit_logs", "siem_events",
	}

	for _, table := range tables {
		if !allowedTables[table] {
			continue
		}
		s.log.Debug("Wiping table: %s", table)
		// Use CryptoWipe for secure erasure
		if err := s.destroyer.CryptoWipe(table, "1=1"); err != nil {
			s.log.Error("Failed to crypto-wipe table %s: %v. Falling back to simple DELETE.", table, err)
			_, _ = s.db.DB().Exec("DELETE FROM " + table)
		}
	}

	s.bus.Publish("database.cleared", nil)
	return nil
}
