package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/analytics"
)

// AnalyticsService provides log searching and dashboard configuration via Wails.
type AnalyticsService struct {
	BaseService
	engine analytics.Engine
}

func NewAnalyticsService(engine analytics.Engine) *AnalyticsService {
	return &AnalyticsService{
		engine: engine,
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
