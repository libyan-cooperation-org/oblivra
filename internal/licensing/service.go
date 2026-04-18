package licensing

import (
	"context"
	"fmt"
)

// Service wraps Manager as an app-lifecycle service compatible with the
// container's ServiceRegistry. It loads a persisted license key from the
// settings database on Startup and provides Wails-bindable methods for the
// frontend Settings > License page.
type Service struct {
	manager    *Manager
	getSetting func(key string) (string, error)
	setSetting func(key, value string) error
	log        interface {
		Info(string, ...interface{})
		Warn(string, ...interface{})
	}
}

const settingsKey = "license_key"

// NewService creates a LicensingService.
//
//   - pubKeyHex  — Ed25519 public key hex, injected at build time via ldflags
//   - getSetting — read from settings DB  (e.g. settingsService.Get)
//   - setSetting — write to settings DB   (e.g. settingsService.Set)
//   - log        — any logger with Info/Warn methods
func NewService(
	pubKeyHex string,
	getSetting func(string) (string, error),
	setSetting func(string, string) error,
	log interface {
		Info(string, ...interface{})
		Warn(string, ...interface{})
	},
) *Service {
	return &Service{
		manager:    New(pubKeyHex),
		getSetting: getSetting,
		setSetting: setSetting,
		log:        log,
	}
}

func (s *Service) Name() string { return "LicensingService" }

// Startup attempts to restore a previously activated license from the settings DB.
func (s *Service) Startup(_ context.Context) {
	raw, err := s.getSetting(settingsKey)
	if err != nil || raw == "" {
		s.log.Info("[licensing] No saved license found — running Community tier")
		return
	}
	if err := s.manager.Activate(raw); err != nil {
		s.log.Warn("[licensing] Saved license failed validation: %v — reverting to Community", err)
		return
	}
	s.log.Info("[licensing] License restored: %s", s.manager.Summary())
}

func (s *Service) Shutdown() {}

// ─── Wails-bindable methods ───────────────────────────────────────────────────

// ActivateLicense verifies a license token and persists it to the settings DB.
func (s *Service) ActivateLicense(token string) error {
	if err := s.manager.Activate(token); err != nil {
		return err
	}
	if err := s.setSetting(settingsKey, token); err != nil {
		s.log.Warn("[licensing] Could not persist license key: %v", err)
	}
	s.log.Info("[licensing] License activated: %s", s.manager.Summary())
	return nil
}

// DeactivateLicense removes the current license and resets to Community tier.
func (s *Service) DeactivateLicense() error {
	s.manager.Deactivate()
	return s.setSetting(settingsKey, "")
}

// GetLicenseStatus returns a JSON-serialisable map for the frontend.
func (s *Service) GetLicenseStatus() map[string]interface{} {
	c := s.manager.CurrentClaims()
	tier := s.manager.CurrentTier()
	status := map[string]interface{}{
		"tier":        tier.String(),
		"tier_int":    int(tier),
		"is_licensed": c != nil,
		"is_expired":  s.manager.IsExpired(),
		"summary":     s.manager.Summary(),
	}
	if c != nil {
		status["licensee"] = c.Licensee
		status["license_id"] = c.LicenseID
		status["max_seats"] = c.MaxSeats
		status["max_agents"] = c.MaxAgents
		if c.ExpiresAt > 0 {
			status["expires_at"] = c.ExpiresAt
		} else {
			status["expires_at"] = nil
		}
	}
	return status
}

// IsFeatureEnabled exposes IsFeatureEnabled to Wails bindings.
func (s *Service) IsFeatureEnabled(feature string) bool {
	return s.manager.IsFeatureEnabled(Feature(feature))
}

// Manager returns the underlying Manager for use by other services in the container.
func (s *Service) Manager() *Manager {
	return s.manager
}

// RequireFeature is a helper for internal service guard checks.
func (s *Service) RequireFeature(f Feature) error {
	if err := s.manager.RequireFeature(f); err != nil {
		return fmt.Errorf("OBLIVRA: %w", err)
	}
	return nil
}
