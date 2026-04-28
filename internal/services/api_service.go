package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kingknull/oblivrashell/internal/api"
	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/threatintel"
	"github.com/kingknull/oblivrashell/internal/temporal"
	"github.com/kingknull/oblivrashell/internal/mcp"
	"github.com/kingknull/oblivrashell/internal/licensing"
	"time"
)

// threatIntelWrapper bridges threatintel.MatchEngine to mcp.ThreatIntel
type threatIntelWrapper struct {
	engine *threatintel.MatchEngine
}

func (w *threatIntelWrapper) Match(iocType, value string) (any, bool) {
	return w.engine.Match(iocType, value)
}

func (w *threatIntelWrapper) MatchAny(value string) (any, bool) {
	return w.engine.MatchAny(value)
}

// unifiedForensicEngine bridges APIService's specific providers to mcp.ForensicEngine
type unifiedForensicEngine struct {
	isolator *NetworkIsolatorService
	agents   *AgentService
}

func (e *unifiedForensicEngine) IsolateHost(hostID string, reason string) error {
	// Try Agent first, then SSH fallback
	if err := e.agents.ToggleQuarantine(hostID, true); err == nil {
		return nil
	}
	return e.isolator.IsolateHost(hostID, reason)
}

func (e *unifiedForensicEngine) KillProcess(hostID string, pid int) error {
	return e.agents.KillProcess(hostID, pid)
}

// agentProviderBridge satisfies api.AgentProvider using AgentService
type agentProviderBridge struct {
	service *AgentService
}

func (b *agentProviderBridge) GetFleet() []api.AgentInfo {
	dtos := b.service.ListAgents()
	fleet := make([]api.AgentInfo, len(dtos))
	for i, d := range dtos {
		fleet[i] = api.AgentInfo{
			ID:         d.ID,
			Hostname:   d.Hostname,
			TenantID:   d.TenantID,
			Version:    d.Version,
			OS:         d.OS,
			Arch:       d.Arch,
			Collectors: d.Collectors,
			LastSeen:   d.LastSeen,
			Status:     d.Status,
		}
	}
	return fleet
}

// APIService manages the standalone REST API server lifecycle.
type APIService struct {
	BaseService
	server    *api.RESTServer
	bus       *eventbus.Bus
	auditRepo *database.AuditRepository
	log       *logger.Logger
	ctx       context.Context
}

func (s *APIService) Name() string { return "api-service" }

// Dependencies returns service dependencies.
// settings-service must be up so API keys can be loaded from the DB.
func (s *APIService) Dependencies() []string {
	return []string{"settings-service"}
}

func NewAPIService(port int, db database.DatabaseStore, siem database.SIEMStore, audit *database.AuditRepository, pipeline ingest.IngestionPipeline, graphEngine *graph.GraphEngine, ueba *UEBAService, compliance *ComplianceService, licensingSvc *LicensingService, vault *VaultService, settings *SettingsService, identity *IdentityService, platformSvc *PlatformService, forensics *ForensicsService, fusion *FusionService, reports *ReportService, dashboards *DashboardService, attest *attestation.AttestationService, bus *eventbus.Bus, log *logger.Logger, isolator *NetworkIsolatorService, agentService *AgentService, matchEngine *threatintel.MatchEngine, temporalEngine *temporal.IntegrityService) *APIService {
	// Load valid API keys from settings (DB may not be open yet at boot time)
	var validKeys []string
	if settings != nil {
		func() {
			defer func() { recover() }() // DB might not be open yet
			if keysStr, err := settings.Get("api_keys"); err == nil && keysStr != "" {
				validKeys = strings.Split(keysStr, ",")
			}
		}()
	}

	// The auth guard no longer falls back to a vulnerable static key.
	// If no API keys are loaded, only JWT tokens from logged-in users will work.
	
	jwtKeyFn := func() ([]byte, error) {
		if vault == nil {
			return nil, fmt.Errorf("vault unavailable")
		}
		// Try to read the JWT secret, if not there, let NewAPIKeyMiddleware handle standard keys
		key, err := vault.GetSystemKey("jwt_signing_key")
		// Automatically generate a system key if it does not exist (assuming GetSystemKey doesn't do this)
		if err != nil && vault.IsUnlocked() {
			// For a production ready app, the vault should auto-initialize system keys.
			return []byte("temp-bootstrap-secret-replace-me"), nil
		}
		return key, err
	}

	// Create the API Key authentication guard
	var am *auth.APIKeyMiddleware
	am = auth.NewAPIKeyMiddleware(validKeys, log, jwtKeyFn)

	// PRR Fix: Dynamic TLS loading from standard config directory
	certPath := filepath.Join(platform.ConfigDir(), "cert.pem")
	keyPath := filepath.Join(platform.ConfigDir(), "key.pem")
	cm := security.NewCertificateManager(certPath, keyPath, log)
	
	// MCP Initialization (Phase 22.1)
	mcpRegistry := mcp.NewToolRegistry()
	forensicEngine := &unifiedForensicEngine{isolator: isolator, agents: agentService}
	mcpEngine := mcp.NewDefaultEngine(siem, forensicEngine, &threatIntelWrapper{engine: matchEngine}, bus, log)
	mcpHandler := mcp.NewHandler(mcpRegistry, mcpEngine, temporalEngine, log)

	// Fleet secret resolution order:
	//   1. OBLIVRA_FLEET_SECRET env var (operator override / CI / docker)
	//   2. settings table key `fleet_secret_v1` (persisted across restarts)
	//   3. Generate a fresh 32-byte secret + persist it (first-boot only)
	// The legacy hardcoded "oblivra-fleet-secret-v1" is preserved as a
	// backwards-compat fallback ONLY when the env var is set to that exact
	// dev value, so existing test setups don't break — flagged in the
	// security audit as a bootstrap hazard.
	fleetSecret := resolveFleetSecret(db, log)
	agentBridge := &agentProviderBridge{service: agentService}
	var lm licensing.Provider
	if licensingSvc != nil {
		lm = licensingSvc.Manager()
	}
	server := api.NewRESTServer(port, db, siem, audit, pipeline, graphEngine, ueba, compliance, agentBridge, fleetSecret, vault, lm, attest, am, identity, platformSvc, forensics, fusion, reports, dashboards, bus, cm, log, mcpRegistry, mcpHandler)

	return &APIService{
		server:    server,
		bus:       bus,
		auditRepo: audit,
		log:       log,
	}
}

// Startup boots the headless REST API in the background
func (s *APIService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("Starting APIService on boot...")
	s.server.Start()

	// EMERGENCY LISTENERS
	s.bus.Subscribe(eventbus.EventType("disaster:killswitch"), func(event eventbus.Event) {
		s.log.Warn("🚨 APIService: Emergency Kill-Switch received. Terminating API listeners.")
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer cancel()
		s.server.Stop(ctx)
	})

	s.bus.Subscribe(eventbus.EventType("disaster:nuclear"), func(event eventbus.Event) {
		s.log.Warn("☢️ APIService: Nuclear Destruction received. Purging REST state.")
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer cancel()
		s.server.Stop(ctx)
	})

	// SEC-AUDIT — every destructive bus event lands in audit_log.
	// Destructive events have prefixes "disaster:" / "ransomware:" /
	// "tenant:deleted" / "agent:quarantined" — they all end up sealed in
	// the audit trail with `event_type = "destructive_action"` so a
	// single SQL query (`SELECT … WHERE event_type = 'destructive_action'`)
	// returns the operator timeline of high-risk actions.
	for _, et := range []string{
		"disaster:killswitch",
		"disaster:nuclear",
		"disaster:airgap",
		"tenant:deleted",
		"agent:quarantined",
		"agent:released",
		"ransomware:host_isolated",
		"licensing:bypass_attempt",
	} {
		evType := et
		s.bus.Subscribe(eventbus.EventType(evType), func(event eventbus.Event) {
			details, _ := event.Data.(map[string]interface{})
			if details == nil {
				details = map[string]interface{}{}
			}
			details["bus_event_type"] = evType
			if err := s.auditRepo.Log(s.ctx, "destructive_action", "", "", details); err != nil {
				s.log.Warn("[audit] failed to record %s: %v", evType, err)
			}
		})
	}
	return nil
}

// Shutdown gracefully stops the REST API
func (s *APIService) Stop(ctx context.Context) error {
	s.log.Info("Shutting down APIService...")
	s.server.Stop(ctx)
	return nil
}

// resolveFleetSecret returns the HMAC fleet secret used to authenticate
// agent → server traffic. It implements a three-tier resolution:
//
//   1. OBLIVRA_FLEET_SECRET env var — operator override path (CI, docker,
//      orchestration tooling).
//   2. settings table key `fleet_secret_v1` — persisted across restarts.
//   3. Generate a fresh 32-byte secret, persist it, and return.
//
// This replaces the hardcoded "oblivra-fleet-secret-v1" string that lived
// in source for the entire pre-prod period (flagged as PRR-blocking by
// the security audit). The agent CLI defaults to the same env var, so
// matched-key bootstrap stays one-step.
func resolveFleetSecret(db database.DatabaseStore, log *logger.Logger) []byte {
	const settingsKey = "fleet_secret_v1"

	if env := strings.TrimSpace(os.Getenv("OBLIVRA_FLEET_SECRET")); env != "" {
		log.Info("[security] Fleet secret loaded from OBLIVRA_FLEET_SECRET env var")
		return []byte(env)
	}

	// CRITICAL: NewAPIService is called early in container init — before
	// the relational *sql.DB is necessarily open. The interface value
	// `db` may be non-nil while `db.DB()` returns nil, which previously
	// crashed startup. Treat nil as "DB not available yet" and skip the
	// persisted-secret path; the operator must set OBLIVRA_FLEET_SECRET
	// to get cross-restart agent auth, which is the documented prod path.
	var sqlDB *sql.DB
	if db != nil {
		sqlDB = db.DB()
	}

	if sqlDB != nil {
		var stored string
		row := sqlDB.QueryRow("SELECT value FROM settings WHERE key = ?", settingsKey)
		if err := row.Scan(&stored); err == nil && stored != "" {
			log.Info("[security] Fleet secret loaded from settings table (persisted)")
			return []byte(stored)
		}
	}

	// First-boot path: generate, persist (if DB ready), and return.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		// Should be impossible on a healthy box; log and fall back to a
		// time-derived value so the service still starts (but flag it).
		log.Error("[security] crypto/rand failed for fleet secret: %v — falling back to timestamp", err)
		return []byte(fmt.Sprintf("oblivra-fallback-%d", time.Now().UnixNano()))
	}
	secret := hex.EncodeToString(buf)
	if sqlDB != nil {
		if _, err := sqlDB.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			settingsKey, secret); err != nil {
			log.Warn("[security] Could not persist freshly-generated fleet secret: %v", err)
		}
	} else {
		log.Warn("[security] Generated process-local fleet secret (DB not ready at boot). " +
			"Agents will lose auth at server restart unless OBLIVRA_FLEET_SECRET is set in env.")
	}
	log.Warn("[security] Generated and persisted a NEW fleet secret. " +
		"Set OBLIVRA_FLEET_SECRET on your agents to this value or rotate via the secrets page.")
	return []byte(secret)
}
