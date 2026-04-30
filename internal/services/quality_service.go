package services

import (
	"context"
	"log/slog"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/quality"
)

type QualityService struct {
	log    *slog.Logger
	engine *quality.Engine
}

func NewQualityService(log *slog.Logger) *QualityService {
	return &QualityService{log: log, engine: quality.New()}
}

func (s *QualityService) ServiceName() string                       { return "QualityService" }
func (s *QualityService) Observe(_ context.Context, ev events.Event) { s.engine.Observe(ev) }
func (s *QualityService) Profiles() []quality.SourceProfile         { return s.engine.Profiles() }
func (s *QualityService) Coverage() []quality.Coverage              { return s.engine.Coverage() }
