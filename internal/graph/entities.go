package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/events"
)

// ─────────────────────────────────────────────────────────────────────────────
// Extended Node/Edge types with timestamps, weights, and integrity hashing
// ─────────────────────────────────────────────────────────────────────────────

// RichNode extends the base Node with temporal and behavioural metadata.
type RichNode struct {
	Node
	FirstSeen  time.Time         `json:"first_seen"`
	LastSeen   time.Time         `json:"last_seen"`
	EventCount int               `json:"event_count"`
	RiskScore  float64           `json:"risk_score"`
	TenantID   string            `json:"tenant_id"`
	Properties map[string]string `json:"properties,omitempty"`
}

// RichEdge extends the base Edge with timestamps, weights, and a cryptographic
// integrity hash that chains back to the source event.
//
//	EdgeHash = SHA-256(FromID + ToID + EdgeType + SourceEventHash)
type RichEdge struct {
	Edge
	ID         string            `json:"id"`
	Timestamp  time.Time         `json:"timestamp"`
	Weight     float64           `json:"weight"`
	EventID    string            `json:"event_id"`   // source SovereignEvent.Id
	EventHash  string            `json:"event_hash"` // SHA-256 of the source event's RawLine
	EdgeHash   string            `json:"edge_hash"`  // integrity chain hash
	TenantID   string            `json:"tenant_id"`
	Properties map[string]string `json:"properties,omitempty"`
}

// computeEdgeHash produces the cryptographic chain hash for an edge.
func computeEdgeHash(from, to string, edgeType EdgeType, eventHash string) string {
	h := sha256.New()
	h.Write([]byte(from + "|" + to + "|" + string(edgeType) + "|" + eventHash))
	return hex.EncodeToString(h.Sum(nil))
}

// computeEventHash produces a SHA-256 fingerprint of a raw event line.
func computeEventHash(rawLine string) string {
	h := sha256.Sum256([]byte(rawLine))
	return hex.EncodeToString(h[:])
}

// ─────────────────────────────────────────────────────────────────────────────
// GraphContext — graph-native event context passed to the detection engine
// ─────────────────────────────────────────────────────────────────────────────

// GraphContext carries the entity nodes, relationships, and traversal paths
// extracted from a single SovereignEvent. It is the input type for graph-aware
// detection rules (see detection/graph_rules.go).
type GraphContext struct {
	// Source event identity
	EventID   string
	TenantID  string
	Timestamp time.Time

	// Extracted entities
	Nodes []RichNode

	// Extracted relationships
	Edges []RichEdge

	// Path: ordered node IDs representing the traversal implied by the event
	// (e.g. user→host→ip). Used by graph detection rules for path matching.
	Path []string

	// Original event retained for backward-compatible log-based rules
	SourceEvent *events.SovereignEvent
}

// ─────────────────────────────────────────────────────────────────────────────
// Entity Extractor
// ─────────────────────────────────────────────────────────────────────────────

// ExtractEntities parses a SovereignEvent and returns a GraphContext with all
// implied entity nodes and relationships.
//
// Pipeline stage:
//
//	[Ingest] → [Normalize] → [ExtractEntities] → [GraphBuilder] → [Detection]
//
// Pure function (no side effects) — safe to call concurrently from any worker.
func ExtractEntities(evt *events.SovereignEvent) *GraphContext {
	if evt == nil {
		return nil
	}
	now := time.Now()

	ctx := &GraphContext{
		EventID:     evt.Id,
		TenantID:    evt.TenantID,
		Timestamp:   now,
		SourceEvent: evt,
	}

	eventHash := computeEventHash(evt.RawLine)
	var nodeIDs []string

	// ── Host node ────────────────────────────────────────────────────────────
	if evt.Host != "" {
		ctx.Nodes = append(ctx.Nodes, RichNode{
			Node:       Node{ID: hostNodeID(evt.TenantID, evt.Host), Type: NodeHost, LastSeen: now},
			FirstSeen:  now, LastSeen: now,
			TenantID:   evt.TenantID,
			Properties: map[string]string{"hostname": evt.Host},
		})
		nodeIDs = append(nodeIDs, hostNodeID(evt.TenantID, evt.Host))
	}

	// ── User node ────────────────────────────────────────────────────────────
	if evt.User != "" {
		ctx.Nodes = append(ctx.Nodes, RichNode{
			Node:       Node{ID: userNodeID(evt.TenantID, evt.User), Type: NodeUser, LastSeen: now},
			FirstSeen:  now, LastSeen: now,
			TenantID:   evt.TenantID,
			Properties: map[string]string{"username": evt.User},
		})
		nodeIDs = append(nodeIDs, userNodeID(evt.TenantID, evt.User))
	}

	// ── IP node ──────────────────────────────────────────────────────────────
	if evt.SourceIp != "" {
		ctx.Nodes = append(ctx.Nodes, RichNode{
			Node:       Node{ID: ipNodeID(evt.SourceIp), Type: NodeIP, LastSeen: now},
			FirstSeen:  now, LastSeen: now,
			TenantID:   evt.TenantID,
			Properties: map[string]string{"ip": evt.SourceIp},
		})
		nodeIDs = append(nodeIDs, ipNodeID(evt.SourceIp))
	}

	// ── Process node (from metadata) ─────────────────────────────────────────
	if cmdLine, ok := evt.Metadata["CommandLine"]; ok && cmdLine != "" {
		procID := processNodeID(evt.TenantID, evt.Host, cmdLine)
		ctx.Nodes = append(ctx.Nodes, RichNode{
			Node:       Node{ID: procID, Type: NodeProcess, LastSeen: now},
			FirstSeen:  now, LastSeen: now,
			TenantID:   evt.TenantID,
			Properties: map[string]string{"CommandLine": cmdLine, "host": evt.Host},
		})
		nodeIDs = append(nodeIDs, procID)
	}

	// ── File node (from metadata) ─────────────────────────────────────────────
	if filePath, ok := evt.Metadata["FilePath"]; ok && filePath != "" {
		fileID := fileNodeID(evt.TenantID, filePath)
		ctx.Nodes = append(ctx.Nodes, RichNode{
			Node:       Node{ID: fileID, Type: NodeFile, LastSeen: now},
			FirstSeen:  now, LastSeen: now,
			TenantID:   evt.TenantID,
			Properties: map[string]string{"path": filePath},
		})
		nodeIDs = append(nodeIDs, fileID)
	}

	// ── Derive edges from event type semantics ────────────────────────────────
	hostID := hostNodeID(evt.TenantID, evt.Host)
	userID := userNodeID(evt.TenantID, evt.User)
	ipID := ipNodeID(evt.SourceIp)

	switch evt.EventType {
	case "failed_login", "successful_login", "ssh_login":
		if evt.User != "" && evt.Host != "" {
			ctx.Edges = append(ctx.Edges, makeEdge(userID, hostID, EdgeAuthenticatedTo, evt, eventHash))
		}
		if evt.SourceIp != "" && evt.Host != "" {
			ctx.Edges = append(ctx.Edges, makeEdge(ipID, hostID, EdgeConnectedTo, evt, eventHash))
		}

	case "sudo_exec", "process_create", "windows_process_creation":
		if evt.User != "" && evt.Host != "" {
			ctx.Edges = append(ctx.Edges, makeEdge(userID, hostID, EdgeAuthenticatedTo, evt, eventHash))
		}
		if cmdLine, ok := evt.Metadata["CommandLine"]; ok && cmdLine != "" {
			procID := processNodeID(evt.TenantID, evt.Host, cmdLine)
			if evt.User != "" {
				ctx.Edges = append(ctx.Edges, makeEdge(userID, procID, EdgeExecuted, evt, eventHash))
			}
			ctx.Edges = append(ctx.Edges, makeEdge(hostID, procID, EdgeSpawned, evt, eventHash))
		}

	case "file_access", "file_create", "windows_file_access":
		if filePath, ok := evt.Metadata["FilePath"]; ok && filePath != "" {
			fileID := fileNodeID(evt.TenantID, filePath)
			if evt.User != "" {
				ctx.Edges = append(ctx.Edges, makeEdge(userID, fileID, EdgeAccessed, evt, eventHash))
			}
		}

	case "network_connection", "dns_query", "netflow":
		if evt.SourceIp != "" && evt.Host != "" {
			ctx.Edges = append(ctx.Edges, makeEdge(hostID, ipID, EdgeConnectedTo, evt, eventHash))
		}

	default:
		if evt.User != "" && evt.Host != "" {
			ctx.Edges = append(ctx.Edges, makeEdge(userID, hostID, EdgeAuthenticatedTo, evt, eventHash))
		}
		if evt.SourceIp != "" && evt.Host != "" {
			ctx.Edges = append(ctx.Edges, makeEdge(hostID, ipID, EdgeConnectedTo, evt, eventHash))
		}
	}

	ctx.Path = nodeIDs
	return ctx
}

// makeEdge constructs a RichEdge with a full integrity chain hash.
func makeEdge(from, to string, edgeType EdgeType, evt *events.SovereignEvent, eventHash string) RichEdge {
	now := time.Now()
	return RichEdge{
		Edge:      Edge{From: from, To: to, Type: edgeType, Timestamp: now},
		ID:        fmt.Sprintf("%s|%s|%s", from, to, edgeType),
		Timestamp: now,
		Weight:    1.0,
		EventID:   evt.Id,
		EventHash: eventHash,
		EdgeHash:  computeEdgeHash(from, to, edgeType, eventHash),
		TenantID:  evt.TenantID,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Node ID helpers — deterministic, tenant-scoped
// ─────────────────────────────────────────────────────────────────────────────

func hostNodeID(tenantID, host string) string {
	if tenantID != "" {
		return fmt.Sprintf("host:%s:%s", tenantID, host)
	}
	return fmt.Sprintf("host:%s", host)
}

func userNodeID(tenantID, user string) string {
	if tenantID != "" {
		return fmt.Sprintf("user:%s:%s", tenantID, user)
	}
	return fmt.Sprintf("user:%s", user)
}

func ipNodeID(ip string) string {
	return fmt.Sprintf("ip:%s", ip)
}

func processNodeID(tenantID, host, cmdLine string) string {
	h := sha256.Sum256([]byte(tenantID + host + cmdLine))
	return fmt.Sprintf("proc:%s", hex.EncodeToString(h[:8]))
}

func fileNodeID(tenantID, path string) string {
	h := sha256.Sum256([]byte(tenantID + path))
	return fmt.Sprintf("file:%s", hex.EncodeToString(h[:8]))
}

// ─────────────────────────────────────────────────────────────────────────────
// UpdateFromContext — integrates a GraphContext into the live GraphEngine
// ─────────────────────────────────────────────────────────────────────────────

// UpdateFromContext applies all nodes and edges from a GraphContext into the
// live GraphEngine. Nodes are upserted (LastSeen refreshed if existing),
// edges are appended. Called from the pipeline DAG GraphNode.
func (e *GraphEngine) UpdateFromContext(ctx *GraphContext) {
	if ctx == nil {
		return
	}

	e.mu.Lock()

	for _, rn := range ctx.Nodes {
		if existing, ok := e.nodes[rn.ID]; ok {
			// Refresh LastSeen and merge metadata on existing nodes
			existing.LastSeen = rn.LastSeen
			existing.Meta = mergeMap(existing.Meta, rn.Properties)
			e.nodes[rn.ID] = existing
		} else {
			// New node — stamp LastSeen
			n := rn.Node
			n.LastSeen = rn.LastSeen
			e.nodes[rn.ID] = n
		}
	}

	for _, re := range ctx.Edges {
		// Stamp Timestamp on the base edge so TTL eviction works
		baseEdge := re.Edge
		baseEdge.Timestamp = re.Timestamp
		e.richEdges = append(e.richEdges, re)
		e.edges = append(e.edges, baseEdge)
	}

	e.mu.Unlock()

	// Publish outside the lock — bus.Publish acquires its own lock
	if e.bus != nil {
		for _, rn := range ctx.Nodes {
			e.bus.Publish("graph.node_upserted", rn)
		}
		for _, re := range ctx.Edges {
			e.bus.Publish("graph.edge_created", re)
		}
	}
}

// GetRichEdges returns all integrity-chained edges for audit export.
func (e *GraphEngine) GetRichEdges() []RichEdge {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]RichEdge, len(e.richEdges))
	copy(out, e.richEdges)
	return out
}

// mergeMap merges overlay into base, returning base.
func mergeMap(base, overlay map[string]string) map[string]string {
	if base == nil {
		base = make(map[string]string, len(overlay))
	}
	for k, v := range overlay {
		base[k] = v
	}
	return base
}
