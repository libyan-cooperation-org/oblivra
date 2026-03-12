package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/gdpr"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type SettingsService struct {
	BaseService
	db        database.DatabaseStore
	bus       *eventbus.Bus
	log       *logger.Logger
	destroyer *gdpr.DataDestructionService
}

func (s *SettingsService) Name() string { return "settings-service" }

// Dependencies returns service dependencies
func (s *SettingsService) Dependencies() []string {
	return []string{"eventbus"}
}

func (s *SettingsService) Start(ctx context.Context) error {
	return nil
}

func (s *SettingsService) Stop(ctx context.Context) error {
	return nil
}

func NewSettingsService(db database.DatabaseStore, bus *eventbus.Bus, log *logger.Logger, destroyer *gdpr.DataDestructionService) *SettingsService {
	return &SettingsService{
		db:        db,
		bus:       bus,
		log:       log.WithPrefix("settings"),
		destroyer: destroyer,
	}
}

func (s *SettingsService) Get(key string) (string, error) {
	var val string
	err := s.db.DB().QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		return "", err
	}
	return val, nil
}

func (s *SettingsService) Set(key string, value string) error {
	s.log.Debug("Setting setting: %s=%s", key, value)
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

	tables := []string{
		"hosts",
		"snippets",
		"settings",
		"metrics",
		"recordings",
		"notes",
		"sessions",
		"audit_logs",
		"siem_events",
	}

	for _, table := range tables {
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
