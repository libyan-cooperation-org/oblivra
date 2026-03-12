package app

import (
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

func (s *AnalyticsService) Name() string { return "AnalyticsService" }

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
