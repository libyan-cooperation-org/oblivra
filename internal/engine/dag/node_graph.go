package dag

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// GraphNode is a DAG processor node that:
//  1. Extracts entities from each event (the "missing layer" identified in the audit)
//  2. Updates the live GraphEngine with the extracted nodes and edges
//  3. Passes the event through unmodified so downstream nodes (SIEM, detection) still receive it
//
// Placement in the DAG (after WASM filter, before SIEM/detection fan-out):
//
//   WASMFilter → GraphNode → MultiDestinationNode → {SIEMNode, AnalyticsNode}
//
// This makes detection graph-aware without breaking any existing pipeline logic.
type GraphNode struct {
	BaseNode
	engine *graph.GraphEngine
	log    *logger.Logger
}

// NewGraphNode creates a new graph enrichment node.
func NewGraphNode(engine *graph.GraphEngine, log *logger.Logger) *GraphNode {
	return &GraphNode{
		BaseNode: BaseNode{nodeName: "Graph_EntityExtractor"},
		engine:   engine,
		log:      log,
	}
}

// Process extracts entities from the event, updates the graph, and passes
// the original event downstream. Never drops events — graph failure is non-fatal.
func (n *GraphNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if n.engine == nil {
		return []*Event{evt}, nil
	}

	// Extract entity nodes and edges from this event
	graphCtx := graph.ExtractEntities(evt)

	// Update the live graph (concurrent-safe, non-blocking path)
	n.engine.UpdateFromContext(graphCtx)

	// Pass event through to all downstream nodes unchanged
	return []*Event{evt}, nil
}
