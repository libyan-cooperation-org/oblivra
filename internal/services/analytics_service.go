package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/oql"
	"github.com/kingknull/oblivrashell/internal/storage"
	"github.com/kingknull/oblivrashell/internal/threatintel"
)

// EntityEnrichment holds geographic and threat context for an entity.
type EntityEnrichment struct {
	Location  string `json:"location"`
	ASN       string `json:"asn"`
	Country   string `json:"country"`
	IOCMatch  bool   `json:"ioc_match"`
	IOCSource string `json:"ioc_source"`
	IOCDesc   string `json:"ioc_desc"`
	Severity  string `json:"severity"`
}

// AnalyticsService provides log searching and dashboard configuration via Wails.
type AnalyticsService struct {
	BaseService
	engine      analytics.Engine
	oqlExecutor *oql.Executor
	matcher     *threatintel.MatchEngine
	hotStore    *storage.HotStore
}

func NewAnalyticsService(engine analytics.Engine, matcher *threatintel.MatchEngine, hotStore *storage.HotStore) *AnalyticsService {
	return &AnalyticsService{
		engine:      engine,
		oqlExecutor: oql.NewExecutor(),
		matcher:     matcher,
		hotStore:    hotStore,
	}
}

func (s *AnalyticsService) Name() string { return "analytics-service" }

// Dependencies returns service dependencies
func (s *AnalyticsService) Dependencies() []string {
	return []string{}
}

func (s *AnalyticsService) Start(ctx context.Context) error {
	return nil
}

func (s *AnalyticsService) Stop(ctx context.Context) error {
	return nil
}

// SearchLogs executes queries against the local Analytics Engine.
func (s *AnalyticsService) SearchLogs(query string, mode string, limit int, offset int) ([]map[string]interface{}, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("analytics engine not localized")
	}
	return s.engine.Search(query, mode, limit, offset)
}

// GetRecordingFrames retrieves the full TTY frame sequence for a session.
func (s *AnalyticsService) GetRecordingFrames(sessionID string) ([]map[string]interface{}, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("analytics engine not localized")
	}
	return s.engine.GetRecordingFrames(sessionID)
}

// SaveDashboard stores a dashboard layout as JSON.
func (s *AnalyticsService) SaveDashboard(id string, layoutJSON string) error {
	if s.engine == nil {
		return fmt.Errorf("analytics engine not localized")
	}
	return s.engine.SaveConfig("dashboard_"+id, layoutJSON)
}

// LoadDashboard retrieves a saved dashboard layout.
func (s *AnalyticsService) LoadDashboard(id string) (string, error) {
	if s.engine == nil {
		return "", fmt.Errorf("analytics engine not localized")
	}
	return s.engine.LoadConfig("dashboard_" + id)
}

// RunWidgetQuery executes a dashboard widget query.
func (s *AnalyticsService) RunWidgetQuery(query string, limit int) ([]map[string]interface{}, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("analytics engine not localized")
	}
	return s.engine.Search(query, "sql", limit, 0)
}

// RunOsquery executes an osquery-style query (stub — osquery integration planned for Phase 6).
func (s *AnalyticsService) RunOsquery(query string) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("osquery integration not yet available — planned for Phase 6 Agent Framework")
}

// RunOQL executes an OQL query. It prefers BadgerDB for system-wide SIEM logs
// but falls back to SQLite for terminal-specific telemetry if needed.
func (s *AnalyticsService) RunOQL(query string) (*oql.QueryResult, error) {
	if s.oqlExecutor == nil {
		return nil, fmt.Errorf("OQL executor not initialized")
	}

	// 1. If we have a HotStore, use the optimized BadgerSource
	if s.hotStore != nil {
		// Create a source for the current execution (Tenant context should be used in production)
		source := oql.NewBadgerSource(s.hotStore, "GLOBAL")
		s.oqlExecutor.SetSource(source)
		
		// Passing nil for data triggers the Source.Fetch() path in the executor
		return s.oqlExecutor.Execute(context.Background(), query, nil, nil)
	}

	// 2. Fallback to SQLite terminal logs for in-memory processing
	if s.engine == nil {
		return nil, fmt.Errorf("analytics engine not localized")
	}

	// Fetch raw logs from SQLite to feed into the OQL engine
	data, err := s.engine.Search("SELECT timestamp, session_id, host, output FROM terminal_logs ORDER BY timestamp DESC LIMIT 50000", "sql", 50000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw data for OQL fallback: %w", err)
	}

	rows := make([]oql.Row, len(data))
	for i, d := range data {
		rows[i] = oql.Row(d)
	}

	return s.oqlExecutor.Execute(context.Background(), query, rows, nil)
}
// GetEntityEnrichment provides real-time context (GeoIP, Threat Intel) for a specific entity.
func (s *AnalyticsService) GetEntityEnrichment(entityID string, entityType string) (*EntityEnrichment, error) {
	enrichment := &EntityEnrichment{
		Location: "Unknown",
	}

	// 1. Threat Intel Check
	if s.matcher != nil {
		iocType := "ipv4-addr"
		if entityType == "user" {
			iocType = "user" // Custom type if we have it
		} else if entityType == "domain" {
			iocType = "domain-name"
		}

		if indicator, hit := s.matcher.Match(iocType, entityID); hit {
			enrichment.IOCMatch = true
			enrichment.IOCSource = indicator.Source
			enrichment.IOCDesc = indicator.Description
			enrichment.Severity = indicator.Severity
		}
	}

	// 2. GeoIP Check (IP entities only)
	// For now, we return placeholder as we need to instantiate GeoIPEnricher correctly
	// but the UI expects this structure.
	if entityType == "ip" {
		// Mock for now, will integrate full GeoIP in next step
		enrichment.Location = "Determining..."
	}

	return enrichment, nil
}
