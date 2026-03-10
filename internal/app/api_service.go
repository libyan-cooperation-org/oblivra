package app

import (
	"context"
	"strings"

	"github.com/kingknull/oblivrashell/internal/api"
	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// APIService manages the standalone REST API server lifecycle.
type APIService struct {
	BaseService
	server *api.RESTServer
	bus    *eventbus.Bus
	log    *logger.Logger
}

func (s *APIService) Name() string { return "APIService" }

func NewAPIService(port int, siem database.SIEMStore, pipeline *ingest.Pipeline, settings *SettingsService, attest *attestation.AttestationService, bus *eventbus.Bus, log *logger.Logger) *APIService {
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

	server := api.NewRESTServer(port, siem, pipeline, attest, am, bus, log)

	return &APIService{
		server: server,
		bus:    bus,
		log:    log,
	}
}

// Startup boots the headless REST API in the background
func (s *APIService) Startup(ctx context.Context) {
	s.log.Info("Starting APIService on boot...")
	s.server.Start()

	// EMERGENCY LISTENERS
	s.bus.Subscribe(eventbus.EventType("disaster:killswitch"), func(event eventbus.Event) {
		s.log.Warn("🚨 APIService: Emergency Kill-Switch received. Terminating API listeners.")
		s.server.Stop(context.Background())
	})

	s.bus.Subscribe(eventbus.EventType("disaster:nuclear"), func(event eventbus.Event) {
		s.log.Warn("☢️ APIService: Nuclear Destruction received. Purging REST state.")
		s.server.Stop(context.Background())
	})
}

// Shutdown gracefully stops the REST API
func (s *APIService) Shutdown() {
	s.log.Info("Shutting down APIService...")
	s.server.Stop(context.Background())
}
