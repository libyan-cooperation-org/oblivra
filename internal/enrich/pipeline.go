package enrich

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// Enricher defines the interface for all SIEM enrichment providers
type Enricher interface {
	Name() string
	Enrich(event *database.HostEvent) error
}

// Pipeline orchestrates the application of multiple enrichers sequentially
type Pipeline struct {
	enrichers []Enricher
	bus       *eventbus.Bus
	log       *logger.Logger
}

func NewPipeline(bus *eventbus.Bus, log *logger.Logger) *Pipeline {
	return &Pipeline{
		enrichers: make([]Enricher, 0),
		bus:       bus,
		log:       log.WithPrefix("enrich"),
	}
}

// Add appends an enricher to the chain
func (p *Pipeline) Add(e Enricher) {
	p.enrichers = append(p.enrichers, e)
	p.log.Info("Registered enricher: %s", e.Name())
}

// Process passes an event through all registered enrichers
func (p *Pipeline) Process(event *database.HostEvent) {
	// Skip enrichment if no actionable IP or Host is present
	if event.SourceIP == "" && event.HostID == "" {
		return
	}

	for _, e := range p.enrichers {
		if err := e.Enrich(event); err != nil {
			p.log.Warn("Enricher %s failed for event %d: %v", e.Name(), event.ID, err)
			continue
		}
	}
}

// Start listens for raw ingestion events, enriches them before they hit the frontend
func (p *Pipeline) Start(ctx context.Context) {
	// Let SIEMService emit `siem.event_indexed`, we'll listen for it.
	// We use priority/synchronous mechanisms in real life, but for now we consume the same bus metric.
	p.bus.Subscribe("siem.event_indexed", func(e eventbus.Event) {
		defer func() {
			if r := recover(); r != nil {
				p.log.Debug("Recovered from panic in Enrichment: %v", r)
			}
		}()
		evt, ok := e.Data.(database.HostEvent)
		if !ok {
			return
		}

		p.Process(&evt)

		// Re-emit enriched event to a secondary channel the UI actually listens to
		p.bus.Publish("siem.event_enriched", evt)
	})

	p.log.Info("Enrichment Pipeline started with %d providers", len(p.enrichers))
}
