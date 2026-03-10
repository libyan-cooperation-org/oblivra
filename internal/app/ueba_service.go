package app

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/ueba"
	
)

// UEBAManager defines the interface for the backend UEBA engine.
type UEBAManager interface {
	GetProfiles() []*ueba.EntityProfile
}

// UEBAService exposes behavioral analytics to the frontend.
type UEBAService struct {
	ctx    context.Context
	engine UEBAManager
	bus    *eventbus.Bus
	log    *logger.Logger
}

func NewUEBAService(engine UEBAManager, bus *eventbus.Bus, log *logger.Logger) *UEBAService {
	return &UEBAService{
		engine: engine,
		bus:    bus,
		log:    log.WithPrefix("app-ueba"),
	}
}

func (s *UEBAService) Name() string { return "UEBAService" }

func (s *UEBAService) Startup(ctx context.Context) {
	s.ctx = ctx

	// Stream anomalies to frontend
	s.bus.Subscribe(eventbus.EventType("siem.anomaly_detected"), func(e eventbus.Event) {
		if s.ctx != nil {
			EmitEvent(s.ctx, "siem:anomaly", e.Data)
		}
	})
}

func (s *UEBAService) Shutdown() {}

// GetProfiles returns the current behavioral profiles.
func (s *UEBAService) GetProfiles() []*ueba.EntityProfile {
	return s.engine.GetProfiles()
}
