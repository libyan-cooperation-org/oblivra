package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/decision"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type DeterministicResponseService struct {
	ctx      context.Context
	log      *logger.Logger
	executor *decision.DeterministicExecutor
}

func NewDeterministicResponseService(executor *decision.DeterministicExecutor, log *logger.Logger) *DeterministicResponseService {
	return &DeterministicResponseService{
		executor: executor,
		log:      log,
	}
}

func (s *DeterministicResponseService) RegisterCtx(ctx context.Context, logger *logger.Logger) {
	s.ctx = ctx
}

func (s *DeterministicResponseService) Name() string {
	return "deterministic-service"
}

// Dependencies returns service dependencies
func (s *DeterministicResponseService) Dependencies() []string {
	return []string{}
}

func (s *DeterministicResponseService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("DeterministicResponseService started")
	return nil
}

func (s *DeterministicResponseService) Stop(ctx context.Context) error {
	return nil
}

// MapResponse generates and stores the cryptographic execution signature.
func (s *DeterministicResponseService) MapResponse(action, eventPayload, policyStateHash string) decision.ExecutionSignature {
	return s.executor.ExecuteAndSign(action, eventPayload, policyStateHash)
}

// GetSignatures returns the history of immutable response signatures.
func (s *DeterministicResponseService) GetSignatures() []decision.ExecutionSignature {
	return s.executor.GetSignatures()
}

// Replay computes the theoretical hash for an event map and checks equality against historical truth.
func (s *DeterministicResponseService) Replay(inputHash, policyHash, action string) map[string]interface{} {
	finalHash, matched := s.executor.Replay(inputHash, policyHash, action)
	return map[string]interface{}{
		"computed_hash": finalHash,
		"matched_past":  matched,
	}
}
