package app

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/decision"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// DecisionService exposes decision traceability to the frontend via Wails.
type DecisionService struct {
	engine *decision.DecisionEngine
	log    *logger.Logger
}

func NewDecisionService(engine *decision.DecisionEngine, log *logger.Logger) *DecisionService {
	return &DecisionService{
		engine: engine,
		log:    log,
	}
}

func (s *DecisionService) Name() string { return "DecisionService" }

func (s *DecisionService) Startup(ctx context.Context) {
	s.log.Info("[DecisionService] Started")
}

func (s *DecisionService) Shutdown() {
	s.log.Info("[DecisionService] Stopped")
}

// GetDecisionTrace returns a single decision trace by ID.
func (s *DecisionService) GetDecisionTrace(id string) *decision.DecisionTrace {
	return s.engine.GetTrace(id)
}

// ListRecentDecisions returns the N most recent decision traces.
func (s *DecisionService) ListRecentDecisions(limit int) []decision.DecisionTrace {
	return s.engine.ListRecent(limit)
}

// GetExplanation returns a human-readable explanation for a decision.
func (s *DecisionService) GetExplanation(id string) string {
	return s.engine.GetExplanation(id)
}

// GetProof returns the cryptographic proof chain for a decision.
func (s *DecisionService) GetProof(id string) string {
	return s.engine.GetProof(id)
}

// GetStats returns summary statistics.
func (s *DecisionService) GetStats() map[string]interface{} {
	return s.engine.Stats()
}
