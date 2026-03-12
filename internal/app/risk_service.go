package app

import (
	"context"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/risk"
)

type RiskService struct {
	BaseService
	engine *risk.RiskEngine
	bus    *eventbus.Bus
	log    *logger.Logger
	ctx    context.Context

	history   []risk.RiskScore
	historyMu sync.RWMutex
}

func NewRiskService(engine *risk.RiskEngine, bus *eventbus.Bus, log *logger.Logger) *RiskService {
	return &RiskService{
		engine:  engine,
		bus:     bus,
		log:     log.WithPrefix("risk_svc"),
		history: make([]risk.RiskScore, 0),
	}
}

func (s *RiskService) Name() string { return "RiskService" }

func (s *RiskService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.engine.Start(ctx)

	// Subscribe to risk evaluations to keep history
	s.bus.Subscribe("risk.evaluated", func(event eventbus.Event) {
		score, ok := event.Data.(risk.RiskScore)
		if ok {
			s.historyMu.Lock()
			s.history = append([]risk.RiskScore{score}, s.history...) // Prepend
			if len(s.history) > 100 {
				s.history = s.history[:100]
			}
			s.historyMu.Unlock()
		}
	})
}

func (s *RiskService) GetRiskHistory() []risk.RiskScore {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()
	return s.history
}

func (s *RiskService) EvaluateManual(type_ string, target string, changes []string) risk.RiskScore {
	event := risk.ConfigChangeEvent{
		Type:      type_,
		Target:    target,
		Changes:   changes,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return s.engine.CalculateScore(event)
}
