package dag

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// GraphNode is a DAG processor node that:
//
//  1. Extracts entity nodes and edges from each event via graph.ExtractEntities()
//  2. Updates the live GraphEngine so entity relationships are tracked in real time
//  3. Passes the original event downstream unmodified (never drops events)
//
// Placement in the production DAG (see pipeline.go buildProductionDAG):
//
//	WASMFilter → GraphNode → MultiDestinationNode → {SIEMNode, AnalyticsNode}
//
// Graph failure is non-fatal: any panic in UpdateFromContext is recovered by
// the pipeline worker's own defer/recover, and the event continues downstream.
type GraphNode struct {
	BaseNode
	engine *graph.GraphEngine
	log    *logger.Logger
}

// NewGraphNode creates a GraphNode that updates the entity graph for every
// event passing through the pipeline.
func NewGraphNode(engine *graph.GraphEngine, log *logger.Logger) *GraphNode {
	return &GraphNode{
		BaseNode: BaseNode{nodeName: "Graph_EntityExtractor"},
		engine:   engine,
		log:      log,
	}
}

// Process extracts entities from the event, updates the graph, and passes
// the original event to all downstream nodes unchanged.
func (n *GraphNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if n.engine == nil {
		return []*Event{evt}, nil
	}

	// Extract entity nodes and edges (pure function, concurrent-safe)
	graphCtx := graph.ExtractEntities(evt)

	// Update the live graph — upserts nodes, appends integrity-chained edges,
	// publishes graph.node_upserted / graph.edge_created bus events
	n.engine.UpdateFromContext(graphCtx)

	// Always pass the event through — graph processing is a side effect only
	return []*Event{evt}, nil
}
