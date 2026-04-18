package detection

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// CorrelationHub is a centralized detection engine for cross-shard (global) rules.
// It listens to indexed events from all shards and maintains a global state
// for rules grouping by User, IP, or other non-host entities.
type CorrelationHub struct {
	engine *Evaluator
	bus    *eventbus.Bus
	log    *logger.Logger
	done   chan struct{}
}

// NewCorrelationHub creates a centralized evaluator for global rules.
func NewCorrelationHub(e *Evaluator, bus *eventbus.Bus, log *logger.Logger) *CorrelationHub {
	// Ensure the engine is set to Global mode
	e.IsLocal = false
	
	ch := &CorrelationHub{
		engine: e,
		bus:    bus,
		log:    log,
		done:   make(chan struct{}),
	}

	return ch
}

// Start begins listening for events from all shards.
func (ch *CorrelationHub) Start(ctx context.Context) {
	ch.log.Info("[DETECTION] CorrelationHub started — monitoring global rules")

	ch.bus.Subscribe("siem.event_indexed", func(e eventbus.Event) {
		evt, ok := e.Data.(database.HostEvent)
		if !ok {
			return
		}

		detEvt := Event{
			TenantID:  evt.TenantID,
			EventType: evt.EventType,
			SourceIP:  evt.SourceIP,
			User:      evt.User,
			HostID:    evt.HostID,
			RawLog:    evt.RawLog,
			Location:  evt.Location,
			Timestamp: evt.Timestamp,
		}

		// Process only Global rules in this hub
		matches := ch.engine.ProcessEvent(detEvt)
		for _, match := range matches {
			ch.bus.Publish("detection.match", match)
		}
	})
}

// Stop shuts down the hub.
func (ch *CorrelationHub) Stop() {
	close(ch.done)
}
