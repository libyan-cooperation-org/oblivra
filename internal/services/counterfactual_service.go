package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// CounterfactualService exposes counterfactual simulation to the frontend via Wails.
type CounterfactualService struct {
	engine    *detection.CounterfactualEngine
	evaluator *detection.Evaluator
	log       *logger.Logger
}

func NewCounterfactualService(engine *detection.CounterfactualEngine, evaluator *detection.Evaluator, log *logger.Logger) *CounterfactualService {
	return &CounterfactualService{
		engine:    engine,
		evaluator: evaluator,
		log:       log,
	}
}

func (s *CounterfactualService) Name() string { return "counterfactual-service" }

// Dependencies returns service dependencies
func (s *CounterfactualService) Dependencies() []string {
	return []string{}
}

func (s *CounterfactualService) Start(ctx context.Context) error {
	s.log.Info("[CounterfactualService] Started")
	return nil
}

func (s *CounterfactualService) Stop(ctx context.Context) error {
	s.log.Info("[CounterfactualService] Stopped")
	return nil
}

// RunCounterfactual runs a counterfactual simulation with the specified rules disabled.
func (s *CounterfactualService) RunCounterfactual(disabledRuleIDs []string) *detection.CounterfactualResult {
	// For the Wails API, we run against an empty event set since we don't have
	// direct access to stored events here. In production, this would pull
	// recent events from the SIEM hot store.
	events := []detection.Event{}
	return s.engine.RunSimulation(events, disabledRuleIDs)
}

// AnalyzeRuleImpact runs a focused simulation for a single rule.
func (s *CounterfactualService) AnalyzeRuleImpact(ruleID string) map[string]interface{} {
	events := []detection.Event{}
	return s.engine.AnalyzeRuleImpact(events, ruleID)
}

// ListRules returns the current rule IDs available for simulation.
func (s *CounterfactualService) ListRules() []string {
	if s.evaluator == nil || s.evaluator.RuleEngine == nil {
		return nil
	}
	var ids []string
	for _, r := range s.evaluator.GetRules() {
		ids = append(ids, r.ID)
	}
	return ids
}
