package services

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// EdgeKind enumerates the graph relationships we record.
type EdgeKind string

const (
	EdgeAlertOf       EdgeKind = "alert-of"        // alert was raised by event
	EdgeMatchesIOC    EdgeKind = "matches-ioc"     // event hit a threat-intel indicator
	EdgeAttachedToCase EdgeKind = "attached-to-case"
	EdgeSession       EdgeKind = "session-of"     // event participates in a session
	EdgeCorroborates  EdgeKind = "corroborates"   // event seen via another path
	EdgeSealedIn      EdgeKind = "sealed-in"      // event captured in an evidence package
)

// Node identifies any anchor in the graph: events, alerts, cases, sessions,
// indicators, evidence packages.
type Node struct {
	Kind string `json:"kind"` // event | alert | case | session | indicator | evidence
	ID   string `json:"id"`
	Hash string `json:"hash,omitempty"` // tamper anchor when available
}

type Edge struct {
	From      Node      `json:"from"`
	To        Node      `json:"to"`
	Kind      EdgeKind  `json:"kind"`
	Evidence  string    `json:"evidence,omitempty"` // free-form note
	Timestamp time.Time `json:"timestamp"`
}

// EvidenceGraphService stores cross-references in memory. Phase 7 will back
// this with SQLite — the in-memory shape is identical so the migration is
// a one-line schema swap.
type EvidenceGraphService struct {
	log *slog.Logger
	mu  sync.RWMutex
	out map[string][]Edge // from-node-key → edges
	in  map[string][]Edge // to-node-key → edges
}

func NewEvidenceGraphService(log *slog.Logger) *EvidenceGraphService {
	return &EvidenceGraphService{log: log, out: map[string][]Edge{}, in: map[string][]Edge{}}
}

func (s *EvidenceGraphService) ServiceName() string { return "EvidenceGraphService" }

func (s *EvidenceGraphService) AddEdge(_ context.Context, e Edge) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fk := nodeKey(e.From)
	tk := nodeKey(e.To)
	s.out[fk] = append(s.out[fk], e)
	s.in[tk] = append(s.in[tk], e)
}

// Neighbors returns every edge touching a node (in or out).
func (s *EvidenceGraphService) Neighbors(n Node) []Edge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k := nodeKey(n)
	out := append([]Edge(nil), s.out[k]...)
	out = append(out, s.in[k]...)
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	return out
}

// Subgraph returns the BFS subgraph rooted at n, bounded by depth.
func (s *EvidenceGraphService) Subgraph(n Node, depth int) []Edge {
	if depth <= 0 {
		depth = 2
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]struct{}{}
	frontier := []Node{n}
	out := []Edge{}
	for d := 0; d < depth && len(frontier) > 0; d++ {
		next := []Node{}
		for _, node := range frontier {
			k := nodeKey(node)
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			for _, e := range s.out[k] {
				out = append(out, e)
				next = append(next, e.To)
			}
			for _, e := range s.in[k] {
				out = append(out, e)
				next = append(next, e.From)
			}
		}
		frontier = next
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	return out
}

// Stats — counts per kind, used by the UI tile.
type GraphStats struct {
	Edges int            `json:"edges"`
	Nodes int            `json:"nodes"`
	ByKind map[string]int `json:"byKind"`
}

func (s *EvidenceGraphService) Stats() GraphStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := GraphStats{ByKind: map[string]int{}}
	st.Edges = 0
	uniqueNodes := map[string]struct{}{}
	for _, edges := range s.out {
		for _, e := range edges {
			st.Edges++
			st.ByKind[string(e.Kind)]++
			uniqueNodes[nodeKey(e.From)] = struct{}{}
			uniqueNodes[nodeKey(e.To)] = struct{}{}
		}
	}
	st.Nodes = len(uniqueNodes)
	return st
}

func nodeKey(n Node) string { return n.Kind + ":" + n.ID }
