package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// GraphService exposes the Security Graph Engine to the Wails frontend.
type GraphService struct {
	engine *graph.GraphEngine
	log    *logger.Logger
}

func NewGraphService(engine *graph.GraphEngine, log *logger.Logger) *GraphService {
	return &GraphService{
		engine: engine,
		log:    log,
	}
}

func (s *GraphService) Name() string { return "graph-service" }

// Dependencies returns service dependencies
func (s *GraphService) Dependencies() []string {
	return []string{}
}

func (s *GraphService) Start(ctx context.Context) error {
	s.engine.SubscribeToAlerts()
	return nil
}

func (s *GraphService) Stop(ctx context.Context) error {
	return nil
}

// GetSubGraph returns a subset of the graph centered on a target entity.
func (s *GraphService) GetSubGraph(nodeID string, hops int) (map[string]interface{}, error) {
	nodes, edges := s.engine.GetSubGraph(nodeID, hops)
	return map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}, nil
}

// FindAttackPath calculates the shortest path between two nodes.
func (s *GraphService) FindAttackPath(startID, endID string) ([]string, error) {
	return s.engine.FindPath(startID, endID)
}

// AddNode allows the frontend or other services to manually inject graph data.
func (s *GraphService) AddNode(id string, nodeType string, meta map[string]string) error {
	s.engine.AddNode(graph.Node{
		ID:   id,
		Type: graph.NodeType(nodeType),
		Meta: meta,
	})
	return nil
}

func (s *GraphService) AddEdge(from, to string, edgeType string) error {
	s.engine.AddEdge(graph.Edge{
		From: from,
		To:   to,
		Type: graph.EdgeType(edgeType),
	})
	return nil
}
