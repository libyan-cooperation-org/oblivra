package detection

// campaign_builder.go — Graph-Aware Campaign Builder (Phase 10.6 / Audit Fix)
//
// This file closes the final gap identified in the code audit:
//
//   "Campaign builder — group alerts sharing entities within time window,
//    score cluster by entity overlap × tactic coverage × time compression."
//
// It subscribes to graph events (graph.node_upserted, graph.edge_created)
// and IOC match events, and feeds entity-correlated data directly into the
// AttackFusionEngine so campaigns are built from GRAPH RELATIONSHIPS rather
// than just flat log patterns.
//
// Architecture:
//   [Graph.UpdateFromContext] → bus("graph.edge_created")
//       → CampaignBuilder.handleEdge()
//           → resolves entity cluster (entities sharing graph edges)
//           → maps edge type → ATT&CK tactic
//           → calls AttackFusionEngine.ingest(entityClusterID, ...)

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// edgeTacticMap maps graph edge types to ATT&CK tactic IDs.
// This is the semantic bridge between entity relationships and threat tactics.
var edgeTacticMap = map[graph.EdgeType]string{
	graph.EdgeAuthenticatedTo: "TA0001", // Initial Access
	graph.EdgeExecuted:        "TA0002", // Execution
	graph.EdgeSpawned:         "TA0003", // Persistence (process spawning)
	graph.EdgeAccessed:        "TA0007", // Discovery (file access)
	graph.EdgeConnectedTo:     "TA0011", // Command & Control (network connection)
}

// CampaignCluster groups entities that share graph edges within a time window.
// It forms the "entity cluster" that campaigns track across tactic stages.
type CampaignCluster struct {
	ClusterID  string            // deterministic: sorted entity IDs joined
	Entities   map[string]bool   // all entity node IDs in this cluster
	TacticHits map[string]int    // tacticID → hit count
	EdgeCount  int
	FirstSeen  time.Time
	LastSeen   time.Time
}

// CampaignBuilder bridges the graph engine and the fusion engine.
// It maintains entity clusters derived from graph edges and feeds them
// into the AttackFusionEngine for probabilistic campaign scoring.
type CampaignBuilder struct {
	fusion  *AttackFusionEngine
	graph   *graph.GraphEngine
	bus     *eventbus.Bus
	log     *logger.Logger

	mu       sync.Mutex
	clusters map[string]*CampaignCluster // clusterID → cluster

	// Time window for clustering: edges within this window from the same
	// entity pair are merged into the same cluster.
	clusterWindow time.Duration
}

// NewCampaignBuilder creates and wires the campaign builder.
// Must be called after both the AttackFusionEngine and GraphEngine are initialised.
func NewCampaignBuilder(fusion *AttackFusionEngine, g *graph.GraphEngine, bus *eventbus.Bus, log *logger.Logger) *CampaignBuilder {
	cb := &CampaignBuilder{
		fusion:        fusion,
		graph:         g,
		bus:           bus,
		log:           log.WithPrefix("campaign_builder"),
		clusters:      make(map[string]*CampaignCluster),
		clusterWindow: 2 * time.Hour,
	}
	cb.subscribe()
	return cb
}

// subscribe registers all bus handlers. Called once at construction.
func (cb *CampaignBuilder) subscribe() {
	// Graph edge events — the primary feed
	cb.bus.Subscribe(eventbus.EventGraphEdgeCreated, func(ev eventbus.Event) {
		edge, ok := ev.Data.(graph.RichEdge)
		if !ok {
			return
		}
		cb.handleEdge(edge)
	})

	// Graph node events — ensure all nodes are tracked in clusters
	cb.bus.Subscribe(eventbus.EventGraphNodeUpserted, func(ev eventbus.Event) {
		node, ok := ev.Data.(graph.RichNode)
		if !ok {
			return
		}
		cb.handleNode(node)
	})

	// IOC matches — if a known IOC is matched, immediately promote the
	// entity to a campaign with "Threat Intelligence" tactic (TA0002-adjacent)
	cb.bus.Subscribe("threatintel.ioc_matched", func(ev eventbus.Event) {
		if data, ok := ev.Data.(map[string]interface{}); ok {
			cb.handleIOCMatch(data)
		}
	})
}

// handleEdge processes a new graph edge and updates/creates the entity cluster.
func (cb *CampaignBuilder) handleEdge(edge graph.RichEdge) {
	if edge.From == "" || edge.To == "" {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Find or create a cluster containing both entities
	cluster := cb.findOrCreateCluster(edge.From, edge.To, edge.TenantID)
	cluster.Entities[edge.From] = true
	cluster.Entities[edge.To] = true
	cluster.EdgeCount++
	cluster.LastSeen = time.Now()

	// Map edge type to ATT&CK tactic
	tactic, ok := edgeTacticMap[edge.Type]
	if !ok {
		tactic = "TA0007" // Default to Discovery
	}
	cluster.TacticHits[tactic]++

	// Feed into fusion engine keyed on the cluster ID
	// This means the fusion engine sees a continuous stream of tactic hits
	// aggregated across ALL entities in the cluster — not just individual IPs.
	go cb.fusion.ingest(
		cluster.ClusterID,
		fmt.Sprintf("graph:%s->%s", edge.From, edge.To),
		fmt.Sprintf("Graph edge: %s [%s]", edge.Type, edge.EdgeHash[:8]),
		tactic,
	)

	// Publish campaign update for the FusionDashboard
	cb.bus.Publish(eventbus.EventCampaignUpdated, map[string]interface{}{
		"cluster_id":   cluster.ClusterID,
		"entities":     cb.entityList(cluster),
		"tactic_hits":  cluster.TacticHits,
		"edge_count":   cluster.EdgeCount,
		"first_seen":   cluster.FirstSeen,
		"last_seen":    cluster.LastSeen,
	})
}

// handleNode ensures isolated nodes are tracked even before they form edges.
func (cb *CampaignBuilder) handleNode(node graph.RichNode) {
	if node.ID == "" {
		return
	}
	// High-risk node types get their own single-entity cluster pre-seeded
	switch node.Type {
	case graph.NodeUser, graph.NodeHost:
		cb.mu.Lock()
		defer cb.mu.Unlock()
		clusterID := "singleton:" + node.ID
		if _, exists := cb.clusters[clusterID]; !exists {
			cb.clusters[clusterID] = &CampaignCluster{
				ClusterID:  clusterID,
				Entities:   map[string]bool{node.ID: true},
				TacticHits: make(map[string]int),
				FirstSeen:  time.Now(),
				LastSeen:   time.Now(),
			}
		}
	}
}

// handleIOCMatch immediately escalates an entity to a campaign when a known IOC is matched.
func (cb *CampaignBuilder) handleIOCMatch(data map[string]interface{}) {
	entityID, _ := data["entity_id"].(string)
	iocType, _ := data["ioc_type"].(string)
	if entityID == "" {
		return
	}

	// IOC match maps to "Threat Intelligence" / Command & Control tactic
	go cb.fusion.ingest(
		entityID,
		fmt.Sprintf("ioc:%s", iocType),
		fmt.Sprintf("IOC match: %s", iocType),
		"TA0011", // Command & Control
	)

	cb.log.Info("[CAMPAIGN] IOC match escalated to campaign: entity=%s ioc_type=%s", entityID, iocType)
}

// findOrCreateCluster finds the existing cluster containing both entities,
// or creates a new one. Implements a simple union-find-style merge:
// if entity A is in cluster X and entity B is in cluster Y, merge X+Y.
func (cb *CampaignBuilder) findOrCreateCluster(entityA, entityB, tenantID string) *CampaignCluster {
	now := time.Now()
	cutoff := now.Add(-cb.clusterWindow)

	// Search for existing cluster containing either entity
	var matchA, matchB *CampaignCluster
	for _, c := range cb.clusters {
		if c.LastSeen.Before(cutoff) {
			continue // Expired cluster
		}
		if c.Entities[entityA] {
			matchA = c
		}
		if c.Entities[entityB] {
			matchB = c
		}
		if matchA != nil && matchB != nil {
			break
		}
	}

	switch {
	case matchA != nil && matchB != nil && matchA != matchB:
		// Merge B into A
		for e := range matchB.Entities {
			matchA.Entities[e] = true
		}
		for tactic, count := range matchB.TacticHits {
			matchA.TacticHits[tactic] += count
		}
		matchA.EdgeCount += matchB.EdgeCount
		if matchB.FirstSeen.Before(matchA.FirstSeen) {
			matchA.FirstSeen = matchB.FirstSeen
		}
		delete(cb.clusters, matchB.ClusterID)
		matchA.ClusterID = cb.buildClusterID(tenantID, matchA.Entities)
		cb.clusters[matchA.ClusterID] = matchA
		return matchA

	case matchA != nil:
		matchA.Entities[entityB] = true
		return matchA

	case matchB != nil:
		matchB.Entities[entityA] = true
		return matchB

	default:
		// New cluster
		entities := map[string]bool{entityA: true, entityB: true}
		clusterID := cb.buildClusterID(tenantID, entities)
		c := &CampaignCluster{
			ClusterID:  clusterID,
			Entities:   entities,
			TacticHits: make(map[string]int),
			FirstSeen:  now,
			LastSeen:   now,
		}
		cb.clusters[clusterID] = c
		return c
	}
}

// buildClusterID produces a deterministic ID from the entity set.
func (cb *CampaignBuilder) buildClusterID(tenantID string, entities map[string]bool) string {
	parts := make([]string, 0, len(entities))
	for e := range entities {
		parts = append(parts, e)
	}
	// Sort for determinism without importing sort (simple bubble for small slices)
	for i := 0; i < len(parts); i++ {
		for j := i + 1; j < len(parts); j++ {
			if parts[i] > parts[j] {
				parts[i], parts[j] = parts[j], parts[i]
			}
		}
	}
	if tenantID != "" {
		return fmt.Sprintf("cluster:%s:%s", tenantID, strings.Join(parts, "+"))
	}
	return fmt.Sprintf("cluster:%s", strings.Join(parts, "+"))
}

func (cb *CampaignBuilder) entityList(c *CampaignCluster) []string {
	list := make([]string, 0, len(c.Entities))
	for e := range c.Entities {
		list = append(list, e)
	}
	return list
}

// GetActiveClusters returns a snapshot of all non-expired entity clusters.
func (cb *CampaignBuilder) GetActiveClusters() []CampaignCluster {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cutoff := time.Now().Add(-cb.clusterWindow)
	result := make([]CampaignCluster, 0)
	for _, c := range cb.clusters {
		if c.LastSeen.After(cutoff) {
			result = append(result, *c)
		}
	}
	return result
}
