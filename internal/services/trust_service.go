package services

import (
	"context"
	"log/slog"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/trust"
)

type TrustService struct {
	log    *slog.Logger
	engine *trust.Engine
}

func NewTrustService(log *slog.Logger) *TrustService {
	return &TrustService{log: log, engine: trust.New()}
}

func (s *TrustService) ServiceName() string { return "TrustService" }

// Observe is plugged into the bus fan-out from the platform stack.
func (s *TrustService) Observe(_ context.Context, ev events.Event) {
	s.engine.Observe(ev)
}

func (s *TrustService) Of(id string) (*trust.Record, bool) { return s.engine.Of(id) }
func (s *TrustService) Summary() trust.Summary             { return s.engine.Summary() }
