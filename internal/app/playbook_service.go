package app

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/incident"
)

// PlaybookService exposes automated response capabilities to the frontend.
type PlaybookService struct {
	engine *incident.PlaybookEngine
}

func NewPlaybookService(engine *incident.PlaybookEngine) *PlaybookService {
	return &PlaybookService{
		engine: engine,
	}
}

func (s *PlaybookService) Name() string { return "PlaybookService" }

func (s *PlaybookService) Startup(ctx context.Context) {
}

func (s *PlaybookService) Shutdown() {
}

func (s *PlaybookService) ExecuteAction(ctx context.Context, action string, params map[string]interface{}) (string, error) {
	return s.engine.ExecuteAction(ctx, action, params)
}

func (s *PlaybookService) RunPlaybook(ctx context.Context, playbookID string, incidentID string) error {
	return s.engine.RunPlaybook(ctx, playbookID, incidentID)
}

func (s *PlaybookService) ListAvailableActions() []string {
	return s.engine.ListAvailableActions()
}
