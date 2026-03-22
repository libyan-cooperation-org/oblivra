package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/oql"
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/threatintel"
)

// SIEMService exposes SIEM configurations to the frontend
type SIEMService struct {
	BaseService
	ctx         context.Context
	repo        database.SIEMStore
	forwarder   *security.SIEMForwarder
	ai          AIPrompter
	snippets    *SnippetService
	matcher     *threatintel.MatchEngine
	correlation *detection.CorrelationEngine
	bus         *eventbus.Bus
	log         *logger.Logger
	lastRiskCheck *sync.Map
	oqlExecutor   *oql.Executor
}

func (s *SIEMService) Name() string { return "siem-service" }

// Dependencies returns service dependencies.
// eventbus is infrastructure wired at construction time, not a kernel-managed service.
func (s *SIEMService) Dependencies() []string {
	return []string{"vault"}
}

func NewSIEMService(r database.SIEMStore, forwarder *security.SIEMForwarder, ai AIPrompter, snippets *SnippetService, matcher *threatintel.MatchEngine, bus *eventbus.Bus, log *logger.Logger) *SIEMService {
	correlationEngine := detection.NewCorrelationEngine(bus, log.WithPrefix("correlation"))
	return &SIEMService{
		repo:          r,
		forwarder:     forwarder,
		ai:            ai,
		snippets:      snippets,
		matcher:       matcher,
		correlation:   correlationEngine,
		bus:           bus,
		log:           log.WithPrefix("siem"),
		lastRiskCheck: &sync.Map{},
		oqlExecutor:   oql.NewExecutor(),
	}
}

func (s *SIEMService) Start(ctx context.Context) error {
	s.ctx = ctx

	// Stream new SIEM events to the frontend UI
	s.bus.Subscribe("siem.event_indexed", func(e eventbus.Event) {
		// e.Data is a database.HostEvent
		defer func() {
			if r := recover(); r != nil {
				s.log.Debug("Recovered from panic in siem.event_indexed: %v", r)
			}
		}()

		if s.ctx == nil {
			return
		}

		// Defensively check if we are in a Wails environment before emitting
		if s.ctx.Value("test") == "true" {
			return
		}

		evt, ok := e.Data.(database.HostEvent)
		if ok && s.matcher != nil {
			// Enrich with Threat Intel if available
			if evt.SourceIP != "" && evt.SourceIP != "127.0.0.1" && evt.SourceIP != "localhost" {
				if indicator, hit := s.matcher.Match("ipv4-addr", evt.SourceIP); hit {
					s.log.Warn("🚨 IOC MATCH ON INGEST: IP %s matched %s feed (%s)", evt.SourceIP, indicator.Source, indicator.Description)
					evt.EventType = "IOC_MATCH_" + evt.EventType

					// Trigger a high priority alert
					s.bus.Publish("security.alert", map[string]interface{}{
						"host_id":    evt.HostID,
						"session_id": "",
						"score":      100,
						"message":    fmt.Sprintf("IOC Match! IP %s appears in %s. Reason: %s", evt.SourceIP, indicator.Source, indicator.Description),
						"type":       "ioc_match",
					})
				}
			}
		}

		EmitEvent(s.ctx, "siem-stream", evt)
	})

	// Subscribe to SIEM audit completion events to perform heuristic cross-checks
	s.bus.Subscribe("siem.audit_completed", func(e eventbus.Event) {
		data, ok := e.Data.(map[string]string)
		if !ok {
			return
		}

		hostID := data["host_id"]
		sessionID := data["session_id"]

		if lastCheck, loaded := s.lastRiskCheck.Load(hostID); loaded {
			if time.Since(lastCheck.(time.Time)) < 5*time.Second {
				return // Debounce rapid SIEM events on the same host
			}
		}
		s.lastRiskCheck.Store(hostID, time.Now())

		// Calculate risk score after audit
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		score, err := s.repo.CalculateRiskScore(ctx, hostID)
		if err != nil {
			s.log.Error("Heuristic risk calculation failed for %s: %v", hostID, err)
			return
		}

		// Flag risky behavior if score > 70
		if score >= 70 {
			msg := "High risk anomaly detected. Multiple failed logins or root targeting observed."
			s.bus.Publish("security.alert", map[string]interface{}{
				"host_id":    hostID,
				"session_id": sessionID,
				"score":      score,
				"message":    msg,
				"type":       "brute_force_heuristic",
			})
			s.log.Warn("SECURITY ALERT: Host %s risk score is %d/100", hostID, score)

			// AUTONOMOUS SUGGESTION: Find relevant snippets for brute force protection
			go s.suggestRemediation(hostID, sessionID, "brute_force")
		}
	})

	s.bus.Subscribe("session.shared", func(event eventbus.Event) {
		// Log sharing events
		details := map[string]interface{}{
			"event_data": event.Data,
		}
		s.forwarder.RecordEvent("session_shared", "info", "terminal-user", "localhost", details)
	})

	s.bus.Subscribe("fido2.registration_completed", func(event eventbus.Event) {
		s.forwarder.RecordEvent("user_registered_fido2", "info", "terminal-user", "localhost", nil)
	})
	return nil
}

func (s *SIEMService) Store() database.SIEMStore {
	return s.repo
}

func (s *SIEMService) Stop(ctx context.Context) error {
	return nil
}

// Configure changes the forwarding destination
func (s *SIEMService) Configure(enabled bool, destination string, url string, token string, batchSize int) error {
	cfg := security.SIEMConfig{
		Enabled:   enabled,
		Type:      security.SIEMType(destination),
		Endpoint:  url,
		Token:     token,
		BatchSize: batchSize,
	}
	s.forwarder.Configure(cfg)
	s.log.Info("Configured SIEM forwarder: %s (Enabled: %v)", destination, enabled)
	return nil
}

// TestConnection pushes a test event
func (s *SIEMService) TestConnection() error {
	s.forwarder.RecordEvent("test_connection", "info", "system", "localhost", map[string]interface{}{"msg": "Hello from OblivraShell"})
	return nil
}

// GetFailedLoginsByHost fetches aggregated login failure analytics for the ThreatMap ECharts visualization
func (s *SIEMService) GetFailedLoginsByHost(hostID string) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	return s.repo.GetFailedLoginsByHost(ctx, hostID)
}

// GetHostEvents grabs raw parsed anomalies mapped to a specific internal SSH server UUID
func (s *SIEMService) GetHostEvents(hostID string, limit int) ([]database.HostEvent, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	return s.repo.GetHostEvents(ctx, hostID, limit)
}

// GetRiskScoreByHost calculates a 0-100 score of how compromised this host might be
func (s *SIEMService) GetRiskScoreByHost(hostID string) (int, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	return s.repo.CalculateRiskScore(ctx, hostID)
}

// SearchHostEvents performs a global search across all host anomaly events
func (s *SIEMService) SearchHostEvents(query string, limit int) ([]database.HostEvent, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 20*time.Second) // Search gets a bit more time
	defer cancel()
	return s.repo.SearchHostEvents(ctx, query, limit)
}

// GetGlobalThreatStats returns high-level dashboard metrics
func (s *SIEMService) GetGlobalThreatStats() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	return s.repo.GetGlobalThreatStats(ctx)
}

// GetEventTrend returns security event counts over time
func (s *SIEMService) GetEventTrend(days int) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	return s.repo.GetEventTrend(ctx, days)
}

// AnalyzeEvent uses AI to analyze a specific log entry
func (s *SIEMService) AnalyzeEvent(rawLog string) (*AIResponse, error) {
	s.log.Info("Analyzing event via AI...")
	prompt := "Analyze this security log and explain any potential threats and suggests a fix:\n\n" + rawLog
	return s.ai.ExplainError(prompt) // ExplainError is close enough for general log analysis
}

// ExecuteOQL parses and executes a Sovereign Query Language string
func (s *SIEMService) ExecuteOQL(query string) (*oql.QueryResult, error) {
	s.log.Info("Executing OQL: %s", query)
	
	// We need a DataSource for the executor.
	// We'll use a bridge that queries our repo.
	// For now, we'll try to find if we can use BadgerSource if repo is Badger-backed.
	// But according to internal/database/siem.go, it's SQL-backed (likely SQLite).
	
	// If it's SQL-backed, we might need a SQLSource for OQL.
	// Let's check if SQLSource exists.
	
	// For MVP, we'll use the InMemSource but populated with recent events from the repo.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events, err := s.repo.SearchHostEvents(ctx, "", 1000) // Get last 1000 events to query against
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events for OQL: %w", err)
	}

	// Convert database.HostEvent to oql.Row
	rows := make([]oql.Row, len(events))
	for i, e := range events {
		rows[i] = oql.Row{
			"id":         e.ID,
			"host_id":    e.HostID,
			"timestamp":  e.Timestamp,
			"event_type": e.EventType,
			"source_ip":  e.SourceIP,
			"user":       e.User,
			"raw_log":    e.RawLog,
		}
	}

	return s.oqlExecutor.Execute(ctx, query, rows, nil)
}

func (s *SIEMService) suggestRemediation(hostID, sessionID, threatType string) {
	if s.snippets == nil {
		return
	}

	snippets, err := s.snippets.List()
	if err != nil {
		return
	}

	var bestMatch *database.Snippet
	for _, snip := range snippets {
		for _, tag := range snip.Tags {
			if tag == threatType || tag == "security" || tag == "incident_response" {
				bestMatch = &snip
				break
			}
		}
		if bestMatch != nil {
			break
		}
	}

	if bestMatch != nil {
		s.log.Info("Found autonomous remediation suggestion for %s: %s", threatType, bestMatch.Title)
		s.bus.Publish("security.suggestion", map[string]interface{}{
			"host_id":    hostID,
			"session_id": sessionID,
			"snippet_id": bestMatch.ID,
			"title":      bestMatch.Title,
			"command":    bestMatch.Command,
			"reason":     fmt.Sprintf("Automatic match for threat type: %s", threatType),
		})
	}
}

// AggregateHostEvents exposes the Bleve/SQL facet aggregation capability to the UI
func (s *SIEMService) AggregateHostEvents(query string, facetField string) (map[string]int, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	defer cancel()
	return s.repo.AggregateHostEvents(ctx, query, facetField)
}

// CreateSavedSearch persists a given query so the user can recall it later
func (s *SIEMService) CreateSavedSearch(search *database.SavedSearch) error {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	return s.repo.CreateSavedSearch(ctx, search)
}

// GetSavedSearches retrieves all persisted SIEM queries
func (s *SIEMService) GetSavedSearches() ([]database.SavedSearch, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	return s.repo.GetSavedSearches(ctx)
}

// GetThreatIntelStats returns matcher statistics for the Threat Intel dashboard
func (s *SIEMService) GetThreatIntelStats() map[string]int {
	if s.matcher != nil {
		return s.matcher.Stats()
	}
	return map[string]int{}
}

// LoadOfflineIOCs provides an endpoint for the UI to inject new offline observables
func (s *SIEMService) LoadOfflineIOCs(indicators []threatintel.Indicator) int {
	if s.matcher != nil {
		count := s.matcher.Load(indicators)
		s.log.Info("Loaded %d observables into memory matcher", count)
		return count
	}
	return 0
}

