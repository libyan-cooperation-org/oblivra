package services

import (
	"context"
	"strconv"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/licensing"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// LicensingService is the Wails-bound wrapper around the licensing.Manager.
// It exposes tier management, feature gating, and per-agent metering to both
// the frontend Settings > License page and to internal service guard checks.
type LicensingService struct {
	BaseService
	ctx     context.Context
	inner   *licensing.Service
	bus     *eventbus.Bus
	log     *logger.Logger
	// agentCount tracks the number of registered agents for metering.
	agentCount int
}

func (s *LicensingService) Name() string         { return "licensing-service" }
func (s *LicensingService) Dependencies() []string { return []string{} }

// NewLicensingService creates the Wails-bound licensing service.
// pubKeyHex is injected at build time via -ldflags "-X main.licensePubKey=...".
// Pass an empty string for development/community builds.
func NewLicensingService(
	pubKeyHex string,
	bus *eventbus.Bus,
	log *logger.Logger,
	getSetting func(string) (string, error),
	setSetting func(string, string) error,
) *LicensingService {
	inner := licensing.NewService(pubKeyHex, getSetting, setSetting, log)
	return &LicensingService{
		inner: inner,
		bus:   bus,
		log:   log.WithPrefix("licensing"),
	}
}

func (s *LicensingService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.inner.Startup(ctx)
	s.log.Info("LicensingService started — tier: %s", s.inner.GetLicenseStatus()["tier"])
	return nil
}

func (s *LicensingService) Stop(_ context.Context) error {
	s.inner.Shutdown()
	return nil
}

// ── Wails-bindable methods ────────────────────────────────────────────────────

// ActivateLicense verifies and activates a license token, persisting it to the
// settings database. Returns an error string on failure.
func (s *LicensingService) ActivateLicense(token string) error {
	if err := s.inner.ActivateLicense(token); err != nil {
		return err
	}
	// Broadcast tier change so all UI components can react
	s.bus.Publish("licensing:activated", s.inner.GetLicenseStatus())
	return nil
}

// DeactivateLicense removes the current license and reverts to Community tier.
func (s *LicensingService) DeactivateLicense() error {
	if err := s.inner.DeactivateLicense(); err != nil {
		return err
	}
	s.bus.Publish("licensing:deactivated", nil)
	return nil
}

// GetLicenseStatus returns the current license state for the frontend.
func (s *LicensingService) GetLicenseStatus() map[string]interface{} {
	status := s.inner.GetLicenseStatus()
	// Inject live agent metering into the status payload
	status["active_agents"] = s.agentCount
	maxAgents := 0
	if c := s.inner.Manager().CurrentClaims(); c != nil {
		maxAgents = c.MaxAgents
	}
	status["max_agents"] = maxAgents
	if maxAgents > 0 {
		status["agents_at_limit"] = s.agentCount >= maxAgents
	} else {
		status["agents_at_limit"] = false
	}
	return status
}

// IsFeatureEnabled reports whether a named feature is available under the
// current license. Used by the frontend to conditionally render gated UI.
func (s *LicensingService) IsFeatureEnabled(feature string) bool {
	return s.inner.IsFeatureEnabled(feature)
}

// GetFeatureMap returns the full map of features and their enabled state.
// Used by the License page to render the feature matrix.
func (s *LicensingService) GetFeatureMap() map[string]bool {
	allFeatures := []licensing.Feature{
		licensing.FeatureSSH, licensing.FeatureTerminal, licensing.FeatureVault,
		licensing.FeatureSnippets, licensing.FeatureNotes, licensing.FeatureSIEM,
		licensing.FeatureAlerts, licensing.FeatureAgents, licensing.FeatureVaultFull,
		licensing.FeatureRecordings, licensing.FeatureTransfers, licensing.FeatureTunnels,
		licensing.FeatureMultiExec, licensing.FeatureAIAssistant, licensing.FeaturePlugins,
		licensing.FeatureDashboard, licensing.FeatureHealth, licensing.FeatureMetrics,
		licensing.FeatureTopology, licensing.FeatureUEBA, licensing.FeatureNDR,
		licensing.FeatureSOAR, licensing.FeaturePurpleTeam, licensing.FeatureCompliance,
		licensing.FeatureForensics, licensing.FeatureIdentity, licensing.FeatureGraph,
		licensing.FeatureTeam, licensing.FeatureSync, licensing.FeatureThreatHunt,
		licensing.FeatureSOC, licensing.FeatureRisk, licensing.FeatureGovernance,
		licensing.FeatureExecutive, licensing.FeatureCluster, licensing.FeatureWarMode,
		licensing.FeatureRansomware, licensing.FeatureSimulation, licensing.FeatureTemporal,
		licensing.FeatureLineage, licensing.FeatureDecisions, licensing.FeatureLedger,
		licensing.FeatureReplay, licensing.FeatureCounterfact, licensing.FeatureDeterministic,
		licensing.FeatureMemorySec, licensing.FeatureDisaster, licensing.FeatureOfflineUpdate,
	}
	result := make(map[string]bool, len(allFeatures))
	for _, f := range allFeatures {
		result[string(f)] = s.inner.IsFeatureEnabled(string(f))
	}
	return result
}

// ── Per-agent metering ────────────────────────────────────────────────────────

// RegisterAgent increments the live agent count and checks the license seat limit.
// Returns an error if the agent limit would be exceeded.
func (s *LicensingService) RegisterAgent() error {
	claims := s.inner.Manager().CurrentClaims()
	if claims != nil && claims.MaxAgents > 0 && s.agentCount >= claims.MaxAgents {
		return &agentLimitError{
			current: s.agentCount,
			max:     claims.MaxAgents,
			tier:    s.inner.Manager().CurrentTier().String(),
		}
	}
	s.agentCount++
	s.log.Info("[licensing] Agent registered — active: %d / max: %d",
		s.agentCount, func() int {
			if claims != nil {
				return claims.MaxAgents
			}
			return -1
		}())
	s.bus.Publish("licensing:agent_registered", map[string]interface{}{
		"active_agents": s.agentCount,
	})
	return nil
}

// UnregisterAgent decrements the live agent count.
func (s *LicensingService) UnregisterAgent() {
	if s.agentCount > 0 {
		s.agentCount--
	}
	s.bus.Publish("licensing:agent_unregistered", map[string]interface{}{
		"active_agents": s.agentCount,
	})
}

// ActiveAgentCount returns the current number of registered agents.
func (s *LicensingService) ActiveAgentCount() int {
	return s.agentCount
}

// Manager exposes the underlying licensing.Manager for use by other services.
func (s *LicensingService) Manager() *licensing.Manager {
	return s.inner.Manager()
}

// RequireFeature is a guard helper for internal service calls.
func (s *LicensingService) RequireFeature(f licensing.Feature) error {
	return s.inner.RequireFeature(f)
}

// ── Errors ────────────────────────────────────────────────────────────────────

type agentLimitError struct {
	current, max int
	tier         string
}

func (e *agentLimitError) Error() string {
	return "agent limit reached: " + e.tier + " license allows " +
		strconv.Itoa(e.max) + " agents (" + strconv.Itoa(e.current) + " active) — upgrade to add more"
}
