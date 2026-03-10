package graph

import (
	"context"
	"fmt"
	"sync"

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
	ID   string            `json:"id"`
	Type NodeType          `json:"type"`
	Meta map[string]string `json:"meta,omitempty"`
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Type EdgeType `json:"type"`
}

// GraphEngine maintains cross-entity relationships for attack path analysis.
type GraphEngine struct {
	mu    sync.RWMutex
	nodes map[string]Node
	edges []Edge
	bus   *eventbus.Bus
	log   *logger.Logger
}

func NewGraphEngine(bus *eventbus.Bus, log *logger.Logger) *GraphEngine {
	return &GraphEngine{
		nodes: make(map[string]Node),
		edges: make([]Edge, 0),
		bus:   bus,
		log:   log.WithPrefix("graph"),
	}
}

// AddNode inserts or updates a node in the graph.
func (e *GraphEngine) AddNode(node Node) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nodes[node.ID] = node
}

// AddEdge inserts a directed relationship between nodes.
func (e *GraphEngine) AddEdge(edge Edge) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.edges = append(e.edges, edge)
}

// FindPath finds the shortest path between two nodes (BFS).
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
		currentNode := path[len(path)-1]

		if currentNode == endNodeID {
			return path, nil
		}

		// Find neighbors
		for _, edge := range e.edges {
			if edge.From == currentNode {
				if !visited[edge.To] {
					visited[edge.To] = true
					newPath := append([]string(nil), path...)
					newPath = append(newPath, edge.To)
					queue = append(queue, newPath)
				}
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

	if start, ok := e.nodes[startNodeID]; ok {
		resultNodes[startNodeID] = start
	} else {
		return nil, nil
	}

	currentLevel := []string{startNodeID}
	for i := 0; i < hops; i++ {
		nextLevel := []string{}
		for _, nodeID := range currentLevel {
			for _, edge := range e.edges {
				if edge.From == nodeID {
					if _, ok := resultNodes[edge.To]; !ok {
						resultNodes[edge.To] = e.nodes[edge.To]
						nextLevel = append(nextLevel, edge.To)
					}
					resultEdges = append(resultEdges, edge)
				} else if edge.To == nodeID {
					// Also include incoming edges for subgraphs
					if _, ok := resultNodes[edge.From]; !ok {
						resultNodes[edge.From] = e.nodes[edge.From]
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
	// Integration logic would query SIEM/Audit logs to build the graph dynamically
	return nil
}

// SubscribeToAlerts listens for SIEM alerts and automatically populates the graph.
func (e *GraphEngine) SubscribeToAlerts() {
	e.log.Info("GraphEngine subscribing to alerts...")
	e.bus.Subscribe("siem.alert_fired", func(event eventbus.Event) {
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return
		}

		hostID, _ := data["host_id"].(string)
		ip, _ := data["source_ip"].(string)
		user, _ := data["user"].(string)

		if hostID != "" && ip != "" {
			e.AddNode(Node{ID: hostID, Type: NodeHost})
			e.AddNode(Node{ID: ip, Type: NodeIP})
			e.AddEdge(Edge{From: hostID, To: ip, Type: EdgeConnectedTo})
		}

		if user != "" && hostID != "" {
			e.AddNode(Node{ID: user, Type: NodeUser})
			e.AddEdge(Edge{From: user, To: hostID, Type: EdgeAuthenticatedTo})
		}
	})
}
