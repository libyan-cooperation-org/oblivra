package ingest

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/engine/dag"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/storage"
)

// ReplayResult holds the aggregated outcome of a deterministic replay session.
type ReplayResult struct {
	TotalEvents     int           `json:"total_events"`
	ProcessedEvents int           `json:"processed_events"`
	Alerts          []eventbus.Event `json:"alerts"`
	Duration        time.Duration `json:"duration"`
	StartTime       time.Time     `json:"start_time"`
}

// EventReplayer provides high-fidelity, isolated event replay.
type EventReplayer struct {
	siem      *database.MockSIEMStore
	bus       *eventbus.CollectingBus
	log       *logger.Logger
	dag       *dag.Engine
}

// NewEventReplayer initializes a replayer with isolated infrastructure mocks.
func NewEventReplayer(log *logger.Logger) *EventReplayer {
	mockSIEM := database.NewMockSIEMStore()
	collectingBus := eventbus.NewCollectingBus()

	replayer := &EventReplayer{
		siem: mockSIEM,
		bus:  collectingBus,
		log:  log,
	}

	// Build a high-fidelity replica of the production DAG but wired to mocks.
	replayer.dag = replayer.buildReplayDAG()

	return replayer
}

func (r *EventReplayer) buildReplayDAG() *dag.Engine {
	// Replay DAG avoids the Analytics tier (which writes to large SQLite files)
	// and focuses on the SIEM detection logic.
	
	// Root Node (Processor)
	// In replay, we skip WASM for now unless requested (MVP focus is SIEM rules).
	
	// SIEM Branch: match everything (we want to see what happens to ALL replayed events)
	siemDest := &dag.Node{Processor: dag.NewSIEMNode(r.siem, r.bus, r.log)}

	return dag.NewEngine(siemDest)
}

// ReplayWAL reads from a specific WAL file and runs the isolated pipeline.
func (r *EventReplayer) ReplayWAL(ctx context.Context, walPath string) (*ReplayResult, error) {
	start := time.Now()
	
	if _, err := os.Stat(walPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("WAL file not found: %s", walPath)
	}

	result := &ReplayResult{
		StartTime: start,
		Alerts:    make([]eventbus.Event, 0),
	}

	// Use storage.WAL.Replay if possible. I'll need to use a dummy directory.
	// Better: Implement the reader here using the known format.
	
	count := 0
	err := storage.ReadWALManual(walPath, func(payload []byte) error {
		count++
		pCtx := events.EventProcessingContext{
			EventID:  fmt.Sprintf("evt-replay-%d", count),
			TenantID: "GLOBAL",
			Seed:     uint64(count),
			Now:      time.Unix(int64(1700000000+count), 0).UTC(),
		}
		evt := AutoParse(string(payload), pCtx)
		
		// Run DAG
		if err := r.dag.Execute(ctx, evt); err != nil {
			return err
		}
		
		return nil
	})

	if err != nil {
		return nil, err
	}

	result.TotalEvents = count
	result.ProcessedEvents = count
	result.Alerts = r.bus.GetEvents(eventbus.AllEvents)
	result.Duration = time.Since(start)

	return result, nil
}
