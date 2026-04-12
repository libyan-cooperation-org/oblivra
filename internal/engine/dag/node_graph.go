package dag

import (
	"context"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// GraphNode is a DAG processor node that:
//
//  1. Extracts entity nodes and edges from each event via graph.ExtractEntities()
//  2. Updates the live GraphEngine (UpdateFromContext) for relationship tracking
//  3. Optionally runs all "type: graph" detection rules via Evaluator.ProcessGraphContext()
//  4. Passes the original event downstream unmodified
//
// Placement in the production DAG (pipeline.go buildProductionDAG):
//
//	WASMFilter → [IdentityEnrichment] → GraphNode → MultiDestinationNode → {SIEMNode, …}
//
// Graph detection is non-fatal: any match error is logged and the event continues.
type GraphNode struct {
	BaseNode
	engine    *graph.GraphEngine
	evaluator *detection.Evaluator  // optional; nil = graph detection disabled
	bus       *eventbus.Bus         // optional; used to publish graph match alerts
	log       *logger.Logger
}

// NewGraphNode creates a GraphNode that updates the graph but does NOT run
// graph-aware detection rules (backward-compatible with existing pipeline wiring).
func NewGraphNode(engine *graph.GraphEngine, log *logger.Logger) *GraphNode {
	return &GraphNode{
		BaseNode: BaseNode{nodeName: "Graph_EntityExtractor"},
		engine:   engine,
		log:      log,
	}
}

// NewGraphNodeWithDetection creates a GraphNode that also runs graph detection rules.
// Pass the same Evaluator used for log-based rules so graph rules are co-located.
// bus is used to publish "siem.alert_fired" events for any matches.
func NewGraphNodeWithDetection(engine *graph.GraphEngine, ev *detection.Evaluator, bus *eventbus.Bus, log *logger.Logger) *GraphNode {
	return &GraphNode{
		BaseNode:  BaseNode{nodeName: "Graph_EntityExtractor_WithDetection"},
		engine:    engine,
		evaluator: ev,
		bus:       bus,
		log:       log,
	}
}

// Process extracts entities from the event, updates the graph, runs graph
// detection rules (if an evaluator is wired), then passes the event downstream.
// Never drops events — graph failure is logged and treated as non-fatal.
func (n *GraphNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if n.engine == nil {
		return []*Event{evt}, nil
	}

	// 1. Extract entity nodes and edges from this event (pure, concurrent-safe)
	graphCtx := graph.ExtractEntities(evt)

	// 2. Update the live graph
	n.engine.UpdateFromContext(graphCtx)

	// 3. Run graph detection rules if an evaluator is attached
	if n.evaluator != nil && graphCtx != nil {
		matches := n.evaluator.ProcessGraphContext(graphCtx)
		for _, m := range matches {
			n.log.Warn("[GRAPH-DETECT] Rule %q fired: severity=%s tenant=%s",
				m.RuleID, m.Severity, m.TenantID)

			// Publish a standard SIEM alert so the alert pipeline picks it up
			if n.bus != nil {
				n.bus.Publish("siem.alert_fired", map[string]interface{}{
					"rule_id":     m.RuleID,
					"description": m.Description,
					"severity":    m.Severity,
					"tactic":      firstNonEmpty(m.MitreTactics),
					"technique":   firstNonEmpty(m.MitreTechniques),
					"group_key":   graphCtx.EventID,
					"tenant_id":   m.TenantID,
					"triggered_at": m.TriggeredAt,
					"context":     m.Context,
					"source":      "graph_rule",
					"meta": map[string]interface{}{
						"path":       graphCtx.Path,
						"node_count": len(graphCtx.Nodes),
						"edge_count": len(graphCtx.Edges),
						"event_id":   graphCtx.EventID,
					},
				})
			}
		}
	}

	// 4. Pass original event downstream unchanged
	return []*Event{evt}, nil
}

// firstNonEmpty returns the first non-empty string in the slice, or "".
func firstNonEmpty(ss []string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// GraphMetricsNode — lightweight stats publisher (optional, append to DAG tail)
// ─────────────────────────────────────────────────────────────────────────────

// GraphMetricsNode periodically publishes graph size stats to the bus.
// Wire it after the GraphNode in the DAG if you want real-time cardinality
// metrics surfaced to the DiagnosticsService and frontend StatusBar.
//
// Usage (in pipeline buildProductionDAG, after graphNode is created):
//
//	metricsNode := dag.NewGraphMetricsNode(graphEngine, bus, log, 30*time.Second)
//	graphNode.Children = append(graphNode.Children, &dag.Node{Processor: metricsNode})
type GraphMetricsNode struct {
	BaseNode
	engine   *graph.GraphEngine
	bus      *eventbus.Bus
	log      *logger.Logger
	interval time.Duration
	lastPub  time.Time
}

// NewGraphMetricsNode creates a metrics node that publishes stats every interval.
func NewGraphMetricsNode(engine *graph.GraphEngine, bus *eventbus.Bus, log *logger.Logger, interval time.Duration) *GraphMetricsNode {
	if interval == 0 {
		interval = 30 * time.Second
	}
	return &GraphMetricsNode{
		BaseNode: BaseNode{nodeName: "Graph_MetricsPublisher"},
		engine:   engine,
		bus:      bus,
		log:      log,
		interval: interval,
	}
}

// Process passes events through and publishes graph stats on the configured interval.
func (n *GraphMetricsNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if n.engine != nil && n.bus != nil && time.Since(n.lastPub) >= n.interval {
		stats := n.engine.Stats()
		n.bus.Publish("graph.stats", map[string]interface{}{
			"node_count":      stats.NodeCount,
			"edge_count":      stats.EdgeCount,
			"rich_edge_count": stats.RichEdgeCount,
			"timestamp":       fmt.Sprintf("%d", time.Now().Unix()),
		})
		n.lastPub = time.Now()
	}
	return []*Event{evt}, nil
}
