package services

import (
	"context"
	"fmt"
	"time"

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
func (s *AnalyticsService) SearchLogs(ctx context.Context, query string, mode string, limit int, offset int) ([]map[string]interface{}, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("analytics engine not localized")
	}
	return s.engine.Search(ctx, query, mode, limit, offset)
}

// GetRecordingFrames retrieves the full TTY frame sequence for a session.
func (s *AnalyticsService) GetRecordingFrames(ctx context.Context, sessionID string) ([]map[string]interface{}, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("analytics engine not localized")
	}
	return s.engine.GetRecordingFrames(ctx, sessionID)
}

// SaveDashboard stores a dashboard layout as JSON.
func (s *AnalyticsService) SaveDashboard(ctx context.Context, id string, layoutJSON string) error {
	if s.engine == nil {
		return fmt.Errorf("analytics engine not localized")
	}
	return s.engine.SaveConfig(ctx, "dashboard_"+id, layoutJSON)
}

// LoadDashboard retrieves a saved dashboard layout.
func (s *AnalyticsService) LoadDashboard(ctx context.Context, id string) (string, error) {
	if s.engine == nil {
		return "", fmt.Errorf("analytics engine not localized")
	}
	return s.engine.LoadConfig(ctx, "dashboard_"+id)
}

// RunWidgetQuery executes a dashboard widget query.
func (s *AnalyticsService) RunWidgetQuery(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("analytics engine not localized")
	}
	return s.engine.Search(ctx, query, "sql", limit, 0)
}

// RunOsquery executes an osquery-style query (stub — osquery integration planned for Phase 6).
func (s *AnalyticsService) RunOsquery(ctx context.Context, query string) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("osquery integration not yet available — planned for Phase 6 Agent Framework")
}

// RunOQL executes an OQL query. It prefers BadgerDB for system-wide SIEM logs
// but falls back to SQLite for terminal-specific telemetry if needed.
func (s *AnalyticsService) RunOQL(ctx context.Context, query string) (*oql.QueryResult, error) {
	if s.oqlExecutor == nil {
		return nil, fmt.Errorf("OQL executor not initialized")
	}

	searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tenantID := "default_tenant" // Safest fallback
	if id, ok := ctx.Value("tenant_id").(string); ok {
		tenantID = id
	}

	// 1. If we have a HotStore, use the optimized BadgerSource
	if s.hotStore != nil {
		source := oql.NewBadgerSource(s.hotStore, tenantID)
		s.oqlExecutor.SetSource(source)
		
		return s.oqlExecutor.Execute(searchCtx, query, nil, nil)
	}

	// 2. Fallback to SQLite terminal logs for in-memory processing
	if s.engine == nil {
		return nil, fmt.Errorf("analytics engine not localized")
	}

	// Fetch raw logs from SQLite to feed into the OQL engine
	data, err := s.engine.Search(searchCtx, "SELECT timestamp, session_id, host, output FROM terminal_logs ORDER BY timestamp DESC LIMIT 50000", "sql", 50000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw data for OQL fallback: %w", err)
	}

	rows := make([]oql.Row, len(data))
	for i, d := range data {
		rows[i] = oql.Row(d)
	}

	return s.oqlExecutor.Execute(searchCtx, query, rows, nil)
}
// GetEntityEnrichment provides real-time context (GeoIP, Threat Intel) for a specific entity.
func (s *AnalyticsService) GetEntityEnrichment(entityID string, entityType string) (*EntityEnrichment, error) {
	enrichment := &EntityEnrichment{
		Location: "Unknown",
	}

	// 1. Threat Intel Check
	if s.matcher != nil {
		iocType := "ipv4-addr"
		switch entityType {
		case "user":
			iocType = "user" // Custom type if we have it
		case "domain":
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
