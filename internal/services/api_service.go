package services

import (
	"context"
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
	if err := e.agents.ToggleQuarantine(context.Background(), hostID, true); err == nil {
		return nil
	}
	return e.isolator.IsolateHost(hostID, reason)
}

func (e *unifiedForensicEngine) KillProcess(hostID string, pid int) error {
	return e.agents.KillProcess(context.Background(), hostID, pid)
}

// APIService manages the standalone REST API server lifecycle.
type APIService struct {
	BaseService
	server *api.RESTServer
	bus    *eventbus.Bus
	log    *logger.Logger
	ctx    context.Context
}

func (s *APIService) Name() string { return "api-service" }

// Dependencies returns service dependencies.
// settings-service must be up so API keys can be loaded from the DB.
func (s *APIService) Dependencies() []string {
	return []string{"settings-service"}
}

func NewAPIService(port int, db database.DatabaseStore, siem database.SIEMStore, audit *database.AuditRepository, pipeline ingest.IngestionPipeline, graphEngine *graph.GraphEngine, settings *SettingsService, identity *IdentityService, reports *ReportService, dashboards *DashboardService, attest *attestation.AttestationService, bus *eventbus.Bus, log *logger.Logger, isolator *NetworkIsolatorService, agentService *AgentService, matchEngine *threatintel.MatchEngine, temporalEngine *temporal.IntegrityService) *APIService {
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

	// Fallback to development key if no keys configured
	if len(validKeys) == 0 {
		validKeys = []string{"oblivra-dev-key"}
	}

	// Create the API Key authentication guard
	var am *auth.APIKeyMiddleware
	am = auth.NewAPIKeyMiddleware(validKeys, log)

	// PRR Fix: Dynamic TLS loading from standard config directory
	certPath := filepath.Join(platform.ConfigDir(), "cert.pem")
	keyPath := filepath.Join(platform.ConfigDir(), "key.pem")
	cm := security.NewCertificateManager(certPath, keyPath, log)
	
	// MCP Initialization (Phase 22.1)
	mcpRegistry := mcp.NewToolRegistry()
	forensicEngine := &unifiedForensicEngine{isolator: isolator, agents: agentService}
	mcpEngine := mcp.NewDefaultEngine(siem, forensicEngine, &threatIntelWrapper{engine: matchEngine}, bus, log)
	mcpHandler := mcp.NewHandler(mcpRegistry, mcpEngine, temporalEngine, log)

	server := api.NewRESTServer(port, db, siem, audit, pipeline, graphEngine, attest, am, identity, reports, dashboards, bus, cm, log, mcpRegistry, mcpHandler)

	return &APIService{
		server: server,
		bus:    bus,
		log:    log,
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
	return nil
}

// Shutdown gracefully stops the REST API
func (s *APIService) Stop(ctx context.Context) error {
	s.log.Info("Shutting down APIService...")
	s.server.Stop(ctx)
	return nil
}
