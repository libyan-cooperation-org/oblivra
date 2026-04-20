package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// GraphService exposes the Security Graph Engine to the Wails frontend.
type GraphService struct {
	engine          *graph.GraphEngine
	campaignBuilder interface{ GetActiveClusters() []detection.CampaignCluster }
	snapshotPath    string
	log             *logger.Logger
}

func NewGraphService(engine *graph.GraphEngine, log *logger.Logger) *GraphService {
	return &GraphService{
		engine: engine,
		log:    log,
	}
}

func (s *GraphService) SetSnapshotPath(path string) {
	s.snapshotPath = path
}

func (s *GraphService) Name() string { return "graph-service" }

// Dependencies returns service dependencies
func (s *GraphService) Dependencies() []string {
	return []string{}
}

func (s *GraphService) Start(ctx context.Context) error {
	// 1. Attempt to restore from persistent storage
	if s.snapshotPath != "" {
		if err := s.engine.LoadSnapshot(s.snapshotPath); err != nil {
			s.log.Warn("[GRAPH] Failed to load snapshot: %v", err)
		}
	}

	// 2. Subscribe to real-time events
	s.engine.SubscribeToAlerts()
	
	// 3. Seed sample data if graph is currently empty (Day Zero UI wow factor)
	stats := s.engine.Stats()
	if stats.NodeCount == 0 {
		s.engine.SeedSampleData()
	}

	return nil
}

func (s *GraphService) Stop(ctx context.Context) error {
	if s.snapshotPath != "" {
		s.log.Info("[GRAPH] Saving snapshot to %s", s.snapshotPath)
		if err := s.engine.SaveSnapshot(s.snapshotPath); err != nil {
			s.log.Error("[GRAPH] Failed to save snapshot: %v", err)
		}
	}
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

// GetFullGraph returns the entire live graph state.
func (s *GraphService) GetFullGraph() (map[string]interface{}, error) {
	nodes, edges := s.engine.GetAll()
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

// SetCampaignBuilder wires the campaign builder after container init.
// Called by the container once both GraphEngine and CampaignBuilder exist.
func (s *GraphService) SetCampaignBuilder(cb interface{ GetActiveClusters() []detection.CampaignCluster }) {
	s.campaignBuilder = cb
}

// GetActiveClusters returns all active entity clusters tracked by the
// CampaignBuilder. Each cluster represents a group of related entities
// (user, host, IP) that have interacted within the correlation window,
// along with which ATT&CK tactics have been observed across those edges.
// Wails-bound: called by FusionDashboard.tsx for the campaign cluster graph.
func (s *GraphService) GetActiveClusters() []map[string]interface{} {
	if s.campaignBuilder == nil {
		return []map[string]interface{}{}
	}

	clusters := s.campaignBuilder.GetActiveClusters()
	out := make([]map[string]interface{}, 0, len(clusters))
	for _, c := range clusters {
		entities := make([]string, 0, len(c.Entities))
		for e := range c.Entities {
			entities = append(entities, e)
		}
		tactics := make([]string, 0, len(c.TacticHits))
		for t := range c.TacticHits {
			tactics = append(tactics, t)
		}
		out = append(out, map[string]interface{}{
			"cluster_id": c.ClusterID,
			"entities":   entities,
			"tactics":    tactics,
			"tactic_hits": c.TacticHits,
			"edge_count":  c.EdgeCount,
			"first_seen":  c.FirstSeen,
			"last_seen":   c.LastSeen,
		})
	}
	return out
}

// GetRichEdges returns all integrity-chained graph edges for audit export.
// Each edge carries an EdgeHash = SHA-256(NodeA + NodeB + EdgeType + EventHash)
// allowing forensic verification that graph data was not tampered with.
func (s *GraphService) GetRichEdges() []graph.RichEdge {
	return s.engine.GetRichEdges()
}
