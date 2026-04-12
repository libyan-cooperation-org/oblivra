package graph

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// NodeType represents the kind of entity in the graph.
type NodeType string

const (
	NodeUser    NodeType = "user"
	NodeHost    NodeType = "host"
	NodeProcess NodeType = "process"
	NodeFile    NodeType = "file"
	NodeIP      NodeType = "ip"
)

// EdgeType represents the relationship between two nodes.
type EdgeType string

const (
	EdgeAuthenticatedTo EdgeType = "authenticated_to"
	EdgeExecuted        EdgeType = "executed"
	EdgeAccessed        EdgeType = "accessed"
	EdgeConnectedTo     EdgeType = "connected_to"
	EdgeSpawned         EdgeType = "spawned"
)

// Node represents a single entity.
type Node struct {
	ID       string            `json:"id"`
	Type     NodeType          `json:"type"`
	Meta     map[string]string `json:"meta,omitempty"`
	LastSeen time.Time         `json:"last_seen"`
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Type      EdgeType  `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// ─────────────────────────────────────────────────────────────────────────────
// GraphConfig — resource limits and eviction policy
// ─────────────────────────────────────────────────────────────────────────────

// GraphConfig controls the memory bounds and TTL of the live graph.
// All limits have safe production defaults; zero values retain those defaults.
type GraphConfig struct {
	// NodeTTL is how long a node lives without being touched by a new event.
	// Default: 72 hours (covers a full soak window).
	NodeTTL time.Duration

	// EdgeTTL is how long a rich edge is retained.
	// Default: 24 hours (edges are recreated from new events; older ones become stale).
	EdgeTTL time.Duration

	// MaxNodes is the hard cap on live nodes. When exceeded, the oldest
	// (by LastSeen) nodes are evicted until we are below the cap.
	// Default: 500 000.
	MaxNodes int

	// MaxRichEdges is the hard cap on integrity-chained edges retained in memory.
	// Default: 2 000 000 (each RichEdge is ~400 bytes → ~800 MB at cap).
	MaxRichEdges int

	// EvictInterval is how often the background goroutine runs TTL eviction.
	// Default: 5 minutes.
	EvictInterval time.Duration
}

func (c *GraphConfig) withDefaults() GraphConfig {
	out := *c
	if out.NodeTTL == 0 {
		out.NodeTTL = 72 * time.Hour
	}
	if out.EdgeTTL == 0 {
		out.EdgeTTL = 24 * time.Hour
	}
	if out.MaxNodes == 0 {
		out.MaxNodes = 500_000
	}
	if out.MaxRichEdges == 0 {
		out.MaxRichEdges = 2_000_000
	}
	if out.EvictInterval == 0 {
		out.EvictInterval = 5 * time.Minute
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// GraphEngine
// ─────────────────────────────────────────────────────────────────────────────

// GraphEngine maintains cross-entity relationships for attack path analysis.
// It is safe for concurrent use. Call Start(ctx) to enable background eviction.
type GraphEngine struct {
	mu        sync.RWMutex
	nodes     map[string]Node
	edges     []Edge        // base edges for BFS/subgraph traversal
	richEdges []RichEdge    // integrity-chained edges (see entities.go)
	cfg       GraphConfig
	bus       *eventbus.Bus
	log       *logger.Logger

	stopEvict context.CancelFunc // nil until Start() is called
}

// NewGraphEngine creates a new GraphEngine with default resource limits.
func NewGraphEngine(bus *eventbus.Bus, log *logger.Logger) *GraphEngine {
	return NewGraphEngineWithConfig(bus, log, GraphConfig{})
}

// NewGraphEngineWithConfig creates a GraphEngine with caller-specified limits.
func NewGraphEngineWithConfig(bus *eventbus.Bus, log *logger.Logger, cfg GraphConfig) *GraphEngine {
	return &GraphEngine{
		nodes: make(map[string]Node),
		edges: make([]Edge, 0),
		cfg:   cfg.withDefaults(),
		bus:   bus,
		log:   log.WithPrefix("graph"),
	}
}

// Start launches the background TTL-eviction goroutine.
// It must be called once after construction; ctx cancellation stops it cleanly.
func (e *GraphEngine) Start(ctx context.Context) {
	evictCtx, cancel := context.WithCancel(ctx)
	e.stopEvict = cancel

	go func() {
		ticker := time.NewTicker(e.cfg.EvictInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				evicted := e.evict()
				if evicted > 0 {
					e.log.Info("[GRAPH] TTL eviction: removed %d stale nodes/edges", evicted)
				}
			case <-evictCtx.Done():
				return
			}
		}
	}()

	e.log.Info("[GRAPH] Started — NodeTTL=%s EdgeTTL=%s MaxNodes=%d MaxEdges=%d EvictInterval=%s",
		e.cfg.NodeTTL, e.cfg.EdgeTTL, e.cfg.MaxNodes, e.cfg.MaxRichEdges, e.cfg.EvictInterval)
}

// Stop halts the background eviction goroutine.
func (e *GraphEngine) Stop() {
	if e.stopEvict != nil {
		e.stopEvict()
	}
}

// evict removes nodes older than NodeTTL and edges older than EdgeTTL.
// If still over the node cap after TTL pass, it evicts the oldest nodes until
// we are within limits. Returns total number of items evicted.
func (e *GraphEngine) evict() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	removed := 0

	// ── Node TTL pass ────────────────────────────────────────────────────────
	for id, n := range e.nodes {
		if now.Sub(n.LastSeen) > e.cfg.NodeTTL {
			delete(e.nodes, id)
			removed++
		}
	}

	// ── Node cap: evict oldest if still over limit ───────────────────────────
	if len(e.nodes) > e.cfg.MaxNodes {
		// Collect and sort by LastSeen ascending (oldest first)
		type nodeAge struct {
			id       string
			lastSeen time.Time
		}
		aged := make([]nodeAge, 0, len(e.nodes))
		for id, n := range e.nodes {
			aged = append(aged, nodeAge{id, n.LastSeen})
		}
		// Simple insertion sort — MaxNodes eviction is rare; slice is bounded
		for i := 1; i < len(aged); i++ {
			for j := i; j > 0 && aged[j].lastSeen.Before(aged[j-1].lastSeen); j-- {
				aged[j], aged[j-1] = aged[j-1], aged[j]
			}
		}
		excess := len(e.nodes) - e.cfg.MaxNodes
		for i := 0; i < excess && i < len(aged); i++ {
			delete(e.nodes, aged[i].id)
			removed++
		}
	}

	// ── Build live node ID set for edge pruning ──────────────────────────────
	liveNodes := make(map[string]bool, len(e.nodes))
	for id := range e.nodes {
		liveNodes[id] = true
	}

	// ── Rich edge TTL + orphan pruning ───────────────────────────────────────
	if len(e.richEdges) > 0 {
		kept := e.richEdges[:0]
		for _, re := range e.richEdges {
			if now.Sub(re.Timestamp) <= e.cfg.EdgeTTL &&
				(liveNodes[re.From] || liveNodes[re.To]) {
				kept = append(kept, re)
			} else {
				removed++
			}
		}
		// Cap to MaxRichEdges: drop oldest if still over limit
		if len(kept) > e.cfg.MaxRichEdges {
			overflow := len(kept) - e.cfg.MaxRichEdges
			removed += overflow
			kept = kept[overflow:] // oldest are at the front (append order)
		}
		e.richEdges = kept
	}

	// ── Base edge TTL + orphan pruning ───────────────────────────────────────
	if len(e.edges) > 0 {
		kept := e.edges[:0]
		for _, ed := range e.edges {
			if now.Sub(ed.Timestamp) <= e.cfg.EdgeTTL &&
				(liveNodes[ed.From] || liveNodes[ed.To]) {
				kept = append(kept, ed)
			} else {
				removed++
			}
		}
		e.edges = kept
	}

	return removed
}

// Stats returns a snapshot of current graph size for diagnostics/metrics.
type GraphStats struct {
	NodeCount     int `json:"node_count"`
	EdgeCount     int `json:"edge_count"`
	RichEdgeCount int `json:"rich_edge_count"`
}

func (e *GraphEngine) Stats() GraphStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return GraphStats{
		NodeCount:     len(e.nodes),
		EdgeCount:     len(e.edges),
		RichEdgeCount: len(e.richEdges),
	}
}

// AddNode inserts or updates a node, refreshing its LastSeen timestamp.
func (e *GraphEngine) AddNode(node Node) {
	e.mu.Lock()
	defer e.mu.Unlock()
	node.LastSeen = time.Now()
	e.nodes[node.ID] = node
}

// AddEdge inserts a directed relationship between nodes.
func (e *GraphEngine) AddEdge(edge Edge) {
	e.mu.Lock()
	defer e.mu.Unlock()
	edge.Timestamp = time.Now()
	e.edges = append(e.edges, edge)
}

// FindPath finds the shortest path between two nodes using BFS.
func (e *GraphEngine) FindPath(startNodeID, endNodeID string) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, ok := e.nodes[startNodeID]; !ok {
		return nil, fmt.Errorf("start node not found")
	}

	queue := [][]string{{startNodeID}}
	visited := map[string]bool{startNodeID: true}

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		current := path[len(path)-1]

		if current == endNodeID {
			return path, nil
		}

		for _, edge := range e.edges {
			if edge.From == current && !visited[edge.To] {
				visited[edge.To] = true
				newPath := make([]string, len(path)+1)
				copy(newPath, path)
				newPath[len(path)] = edge.To
				queue = append(queue, newPath)
			}
		}
	}

	return nil, fmt.Errorf("no path found")
}

// GetSubGraph returns all nodes and edges within N hops of a starting node.
func (e *GraphEngine) GetSubGraph(startNodeID string, hops int) ([]Node, []Edge) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	resultNodes := make(map[string]Node)
	resultEdges := make([]Edge, 0)

	start, ok := e.nodes[startNodeID]
	if !ok {
		return nil, nil
	}
	resultNodes[startNodeID] = start

	currentLevel := []string{startNodeID}
	for i := 0; i < hops; i++ {
		var nextLevel []string
		for _, nodeID := range currentLevel {
			for _, edge := range e.edges {
				if edge.From == nodeID {
					if _, seen := resultNodes[edge.To]; !seen {
						if n, ok := e.nodes[edge.To]; ok {
							resultNodes[edge.To] = n
						}
						nextLevel = append(nextLevel, edge.To)
					}
					resultEdges = append(resultEdges, edge)
				} else if edge.To == nodeID {
					if _, seen := resultNodes[edge.From]; !seen {
						if n, ok := e.nodes[edge.From]; ok {
							resultNodes[edge.From] = n
						}
						nextLevel = append(nextLevel, edge.From)
					}
					resultEdges = append(resultEdges, edge)
				}
			}
		}
		currentLevel = nextLevel
	}

	nodes := make([]Node, 0, len(resultNodes))
	for _, n := range resultNodes {
		nodes = append(nodes, n)
	}
	return nodes, resultEdges
}

// CorrelateIncident builds a subgraph based on incident participants.
func (e *GraphEngine) CorrelateIncident(ctx context.Context, principalID string) error {
	e.log.Info("[GRAPH] Correlating entity %s for incident graph", principalID)
	return nil
}

// SubscribeToAlerts listens for SIEM alerts and automatically populates the graph.
func (e *GraphEngine) SubscribeToAlerts() {
	e.log.Info("[GRAPH] Subscribing to siem.alert_fired")
	e.bus.Subscribe("siem.alert_fired", func(event eventbus.Event) {
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return
		}

		hostID, _ := data["host_id"].(string)
		ip, _ := data["source_ip"].(string)
		user, _ := data["user"].(string)

		now := time.Now()
		if hostID != "" && ip != "" {
			e.AddNode(Node{ID: hostID, Type: NodeHost, LastSeen: now})
			e.AddNode(Node{ID: ip, Type: NodeIP, LastSeen: now})
			e.AddEdge(Edge{From: hostID, To: ip, Type: EdgeConnectedTo, Timestamp: now})
		}
		if user != "" && hostID != "" {
			e.AddNode(Node{ID: user, Type: NodeUser, LastSeen: now})
			e.AddEdge(Edge{From: user, To: hostID, Type: EdgeAuthenticatedTo, Timestamp: now})
		}
	})
}
