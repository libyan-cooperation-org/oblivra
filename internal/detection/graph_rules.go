package detection

// graph_rules.go — Graph-Aware Detection Rules (Phase 3.5)
//
// This file closes the final gap identified in the backend audit:
//
//   "Graph-based detection rules — the plumbing exists (GraphContext,
//    UpdateFromContext, the DAG GraphNode) but the Evaluator.ProcessEvent
//    still only accepts the flat Event struct. There is no rule type that
//    takes a GraphContext as input."
//
// ─────────────────────────────────────────────────────────────────────────────
// Design
// ─────────────────────────────────────────────────────────────────────────────
//
// A GraphRule is a YAML-native rule type ("type: graph") evaluated against the
// GraphContext produced by graph.ExtractEntities() rather than the flat Event.
//
// Rules express:
//   • which entity types must be present (node_types)
//   • which relationship types must appear  (edge_types)
//   • the minimum number of distinct hops in the traversal path (min_path_len)
//   • optional property constraints on any node  (node_conditions)
//   • optional per-edge-type count thresholds    (edge_thresholds)
//   • the same MITRE, severity, dedup fields as log-based rules
//
// Example YAML (saved to rules/graph_lateral_movement.yaml):
//
//   id:          G-LM-001
//   name:        Graph Lateral Movement via Authenticated Chain
//   type:        graph
//   severity:    high
//   description: User authenticated to 3+ distinct hosts within the time window.
//   mitre_tactics:    ["TA0008"]
//   mitre_techniques: ["T1021"]
//   node_types:  [user, host]
//   edge_types:  [authenticated_to]
//   min_path_len: 3
//   edge_thresholds:
//     authenticated_to: 3
//   window_sec:       300
//   dedup_window_sec: 120
//   is_global:        true
//
// ProcessGraphContext() feeds each GraphContext through all loaded GraphRules.
// Matches produce the same ExplainedMatch/Match types as log-based rules and
// are published on "detection.explained_match" for the API and frontend.
//
// ─────────────────────────────────────────────────────────────────────────────

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/kingknull/oblivrashell/internal/graph"
)

// ─────────────────────────────────────────────────────────────────────────────
// GraphRule — YAML schema
// ─────────────────────────────────────────────────────────────────────────────

// GraphRuleType is the RuleType constant for graph-aware rules.
const GraphRuleType RuleType = "graph"

// GraphRule is the compiled, in-memory representation of a "type: graph" rule.
// It is derived from the standard Rule struct at load time.
type GraphRule struct {
	// Core identity — mirrors Rule fields loaded from YAML
	ID          string
	Name        string
	Description string
	Severity    string
	Version     string

	// MITRE
	MitreTactics    []string
	MitreTechniques []string

	// ── Graph-specific constraints ────────────────────────────────────────────

	// NodeTypes lists entity types that must be present in the context.
	// ["user", "host"] means the event must involve at least one user and one host.
	NodeTypes []string `yaml:"node_types"`

	// EdgeTypes lists relationship types that must appear in the context.
	// ["authenticated_to", "spawned"] means both must be present.
	EdgeTypes []string `yaml:"edge_types"`

	// MinPathLen is the minimum number of distinct node IDs in GraphContext.Path.
	// Use 3 to detect A→B→C lateral movement (user→host1→host2).
	MinPathLen int `yaml:"min_path_len"`

	// EdgeThresholds is an optional per-edge-type minimum count.
	// {"authenticated_to": 3} fires only if at least 3 auth edges are present.
	// Evaluated against the stateful window (same grouping as log rules).
	EdgeThresholds map[graph.EdgeType]int `yaml:"edge_thresholds"`

	// NodeConditions is an optional map of property key→value constraints
	// that at least one node in the context must satisfy.
	// {"CommandLine": "powershell"} means a process node with that cmdline must exist.
	NodeConditions map[string]string `yaml:"node_conditions"`

	// Window and dedup
	WindowSec      int
	DedupWindowSec int

	// IsGlobal mirrors Rule.IsGlobal — global rules run in CorrelationHub,
	// not in local pipeline shards.
	IsGlobal bool
}

// toGraphRule converts a standard Rule (type: graph) into a GraphRule.
// Returns nil if the rule is not a graph rule.
func toGraphRule(r Rule) *GraphRule {
	if r.Type != GraphRuleType {
		return nil
	}

	gr := &GraphRule{
		ID:              r.ID,
		Name:            r.Name,
		Description:     r.Description,
		Severity:        r.Severity,
		Version:         r.Version,
		MitreTactics:    r.MitreTactics,
		MitreTechniques: r.MitreTechniques,
		WindowSec:       r.WindowSec,
		DedupWindowSec:  r.DedupWindowSec,
		IsGlobal:        r.IsGlobal,
		EdgeThresholds:  make(map[graph.EdgeType]int),
		NodeConditions:  make(map[string]string),
	}

	// Extract graph-specific fields from the conditions map (YAML reuse)
	// We encode graph rule fields into the conditions map in YAML as:
	//   conditions:
	//     node_types: "user,host"
	//     edge_types: "authenticated_to,spawned"
	//     min_path_len: "3"
	//     node_condition.CommandLine: "powershell"
	//     edge_threshold.authenticated_to: "3"
	for k, v := range r.Conditions {
		val := fmt.Sprintf("%v", v)
		switch strings.ToLower(k) {
		case "node_types":
			for _, t := range strings.Split(val, ",") {
				if s := strings.TrimSpace(t); s != "" {
					gr.NodeTypes = append(gr.NodeTypes, s)
				}
			}
		case "edge_types":
			for _, t := range strings.Split(val, ",") {
				if s := strings.TrimSpace(t); s != "" {
					gr.EdgeTypes = append(gr.EdgeTypes, s)
				}
			}
		case "min_path_len":
			fmt.Sscanf(val, "%d", &gr.MinPathLen)
		default:
			if strings.HasPrefix(k, "node_condition.") {
				prop := strings.TrimPrefix(k, "node_condition.")
				gr.NodeConditions[prop] = val
			} else if strings.HasPrefix(k, "edge_threshold.") {
				et := graph.EdgeType(strings.TrimPrefix(k, "edge_threshold."))
				var n int
				fmt.Sscanf(val, "%d", &n)
				gr.EdgeThresholds[et] = n
			}
		}
	}

	return gr
}

// ─────────────────────────────────────────────────────────────────────────────
// GraphEvaluatorState — stateful window tracking for graph rules
// ─────────────────────────────────────────────────────────────────────────────

// graphEdgeCount tracks the per-edge-type count within the rule window for a
// given group key (typically tenant:entityID).
type graphEdgeCount struct {
	counts    map[graph.EdgeType]int
	updatedAt time.Time
}

// graphRuleState holds per-rule, per-group-key edge count windows.
type graphRuleState struct {
	mu     sync.Mutex
	lru    *expirable.LRU[string, *graphEdgeCount]
	alerts *expirable.LRU[string, time.Time]
}

func newGraphRuleState(windowSec, dedupSec int) *graphRuleState {
	window := time.Duration(windowSec) * time.Second
	if window == 0 {
		window = 5 * time.Minute
	}
	dedup := time.Duration(dedupSec) * time.Second
	if dedup == 0 {
		dedup = 2 * time.Minute
	}
	return &graphRuleState{
		lru:    expirable.NewLRU[string, *graphEdgeCount](50_000, nil, window),
		alerts: expirable.NewLRU[string, time.Time](50_000, nil, dedup),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GraphRuleEngine — extends Evaluator with graph rule state
// ─────────────────────────────────────────────────────────────────────────────

// graphRulesMu guards graphRules and graphRuleStates on the Evaluator.
// We extend Evaluator with a side-channel rather than touching the core struct
// so this file stays self-contained and the existing engine.go is unchanged.
var (
	graphRulesMu    sync.RWMutex
	graphRulesMap   = make(map[*Evaluator][]GraphRule)
	graphStatesMap  = make(map[*Evaluator]map[string]*graphRuleState)
)

// registerGraphRules compiles all "type: graph" rules from the evaluator's
// rule set into GraphRule structs and initialises their state windows.
// Called once from ProcessGraphContext on first use (lazy init).
func registerGraphRules(ev *Evaluator) {
	graphRulesMu.Lock()
	defer graphRulesMu.Unlock()

	if _, already := graphRulesMap[ev]; already {
		return
	}

	rules := make([]GraphRule, 0)
	states := make(map[string]*graphRuleState)

	for _, r := range ev.GetRules() {
		if r.Type != GraphRuleType {
			continue
		}
		gr := toGraphRule(r)
		if gr == nil {
			continue
		}
		rules = append(rules, *gr)
		states[gr.ID] = newGraphRuleState(gr.WindowSec, gr.DedupWindowSec)
	}

	graphRulesMap[ev] = rules
	graphStatesMap[ev] = states

	ev.log.Info("[GRAPH-RULES] Compiled %d graph rules", len(rules))
}

// ─────────────────────────────────────────────────────────────────────────────
// ProcessGraphContext — the new entry point for graph-native detection
// ─────────────────────────────────────────────────────────────────────────────

// ProcessGraphContext evaluates all loaded GraphRules against the provided
// GraphContext and returns any matches. It is called from the pipeline DAG
// GraphNode immediately after graph.UpdateFromContext().
//
// Callers should use the same Evaluator instance that handles log-based events.
// Matches are published on "detection.explained_match" (same as log matches).
func (ev *Evaluator) ProcessGraphContext(ctx *graph.GraphContext) []Match {
	if ctx == nil {
		return nil
	}

	// Lazy-init graph rules (idempotent, guarded by mutex)
	graphRulesMu.RLock()
	_, ready := graphRulesMap[ev]
	graphRulesMu.RUnlock()
	if !ready {
		registerGraphRules(ev)
	}

	graphRulesMu.RLock()
	rules := graphRulesMap[ev]
	states := graphStatesMap[ev]
	graphRulesMu.RUnlock()

	var matches []Match

	for _, gr := range rules {
		// Sharding scope check — same logic as log-based rules
		if ev.IsLocal && gr.IsGlobal {
			continue
		}
		if !ev.IsLocal && !gr.IsGlobal {
			continue
		}

		if m := evaluateGraphRule(ev, &gr, ctx, states[gr.ID]); m != nil {
			matches = append(matches, *m)
		}
	}

	return matches
}

// evaluateGraphRule checks a single GraphRule against a GraphContext.
func evaluateGraphRule(ev *Evaluator, gr *GraphRule, ctx *graph.GraphContext, state *graphRuleState) *Match {
	if state == nil {
		return nil
	}

	// ── 1. Structural checks (fast, no state) ────────────────────────────────

	if !hasRequiredNodeTypes(ctx, gr.NodeTypes) {
		return nil
	}
	if !hasRequiredEdgeTypes(ctx, gr.EdgeTypes) {
		return nil
	}
	if gr.MinPathLen > 0 && len(ctx.Path) < gr.MinPathLen {
		return nil
	}
	if len(gr.NodeConditions) > 0 && !nodeConditionsMet(ctx, gr.NodeConditions) {
		return nil
	}

	// ── 2. Stateful edge-count window ────────────────────────────────────────

	groupKey := buildGraphGroupKey(ctx)

	state.mu.Lock()
	defer state.mu.Unlock()

	entry, ok := state.lru.Get(groupKey)
	if !ok {
		entry = &graphEdgeCount{
			counts:    make(map[graph.EdgeType]int),
			updatedAt: time.Now(),
		}
	}

	// Accumulate edge type counts from this context
	for _, re := range ctx.Edges {
		entry.counts[re.Type]++
	}
	entry.updatedAt = time.Now()
	state.lru.Add(groupKey, entry)

	// Check per-edge-type thresholds
	for et, required := range gr.EdgeThresholds {
		if entry.counts[et] < required {
			return nil // threshold not yet reached
		}
	}

	// ── 3. Deduplication ─────────────────────────────────────────────────────

	if lastAlert, fired := state.alerts.Get(groupKey); fired {
		dedupWindow := time.Duration(gr.DedupWindowSec) * time.Second
		if dedupWindow == 0 {
			dedupWindow = 2 * time.Minute
		}
		if time.Since(lastAlert) < dedupWindow {
			return nil // still in dedup window
		}
	}
	state.alerts.Add(groupKey, time.Now())

	// Reset accumulated counts after firing to avoid continuous re-triggering
	entry.counts = make(map[graph.EdgeType]int)
	state.lru.Add(groupKey, entry)

	// ── 4. Build and publish the Match ───────────────────────────────────────

	m := buildGraphMatch(gr, ctx, groupKey)
	explained := ExplainGraphEdgeMatch(
		ctx.Path[0], ctx.Path[len(ctx.Path)-1],
		edgeTypeSummary(ctx.Edges),
		firstTactic(gr.MitreTactics),
		ctx.TenantID,
		0.85, // graph structural matches are high-confidence
	)

	ev.log.Info("[GRAPH-RULES:WHY] rule=%q severity=%s confidence=%.2f group=%q summary=%q",
		explained.RuleID,
		explained.Severity,
		explained.WhyFired.Confidence,
		explained.WhyFired.GroupKey,
		explained.WhyFired.Summary,
	)

	if ev.bus != nil {
		ev.bus.Publish("detection.explained_match", explained)
	}

	return &m
}

// ─────────────────────────────────────────────────────────────────────────────
// Structural match helpers
// ─────────────────────────────────────────────────────────────────────────────

// hasRequiredNodeTypes returns true if all required node types exist in the context.
func hasRequiredNodeTypes(ctx *graph.GraphContext, required []string) bool {
	if len(required) == 0 {
		return true
	}
	present := make(map[string]bool, len(ctx.Nodes))
	for _, n := range ctx.Nodes {
		present[string(n.Type)] = true
	}
	for _, req := range required {
		if !present[req] {
			return false
		}
	}
	return true
}

// hasRequiredEdgeTypes returns true if all required edge types exist in the context.
func hasRequiredEdgeTypes(ctx *graph.GraphContext, required []string) bool {
	if len(required) == 0 {
		return true
	}
	present := make(map[string]bool, len(ctx.Edges))
	for _, e := range ctx.Edges {
		present[string(e.Type)] = true
	}
	for _, req := range required {
		if !present[req] {
			return false
		}
	}
	return true
}

// nodeConditionsMet returns true if at least one node satisfies ALL property constraints.
func nodeConditionsMet(ctx *graph.GraphContext, conditions map[string]string) bool {
	for _, n := range ctx.Nodes {
		if n.Properties == nil {
			continue
		}
		allMatch := true
		for k, v := range conditions {
			prop, ok := n.Properties[k]
			if !ok || !strings.Contains(strings.ToLower(prop), strings.ToLower(v)) {
				allMatch = false
				break
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}

// buildGraphGroupKey produces a stable group key for stateful windowing.
// Keyed on tenant + the principal entity (first user/host in the path).
func buildGraphGroupKey(ctx *graph.GraphContext) string {
	principal := ""
	for _, n := range ctx.Nodes {
		if n.Type == graph.NodeUser || n.Type == graph.NodeHost {
			principal = n.ID
			break
		}
	}
	if ctx.TenantID != "" {
		return fmt.Sprintf("t:%s|%s", ctx.TenantID, principal)
	}
	return principal
}

// buildGraphMatch constructs a Match from a fired GraphRule and its context.
func buildGraphMatch(gr *GraphRule, ctx *graph.GraphContext, groupKey string) Match {
	pathSummary := strings.Join(ctx.Path, "→")
	return Match{
		TenantID:        ctx.TenantID,
		RuleID:          gr.ID,
		RuleName:        gr.Name,
		Description:     gr.Description,
		Severity:        gr.Severity,
		MitreTactics:    gr.MitreTactics,
		MitreTechniques: gr.MitreTechniques,
		TriggeredAt:     time.Now().Format(time.RFC3339),
		Context: map[string]string{
			"group_key":    groupKey,
			"path":         pathSummary,
			"node_count":   fmt.Sprintf("%d", len(ctx.Nodes)),
			"edge_count":   fmt.Sprintf("%d", len(ctx.Edges)),
			"source_event": ctx.EventID,
		},
	}
}

// edgeTypeSummary returns a comma-joined summary of unique edge types for logging.
func edgeTypeSummary(edges []graph.RichEdge) string {
	seen := make(map[string]bool)
	var out []string
	for _, e := range edges {
		k := string(e.Type)
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return strings.Join(out, ",")
}

// firstTactic returns the first tactic ID or an empty string.
func firstTactic(tactics []string) string {
	if len(tactics) > 0 {
		return tactics[0]
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// ReloadGraphRules — called after a live rule-engine reload
// ─────────────────────────────────────────────────────────────────────────────

// ReloadGraphRules re-compiles the GraphRule index for the given Evaluator.
// Call this after Evaluator.RebuildRouteIndex() on a hot-reload so new or
// updated "type: graph" rules take effect without a restart.
func ReloadGraphRules(ev *Evaluator) {
	graphRulesMu.Lock()
	delete(graphRulesMap, ev)
	delete(graphStatesMap, ev)
	graphRulesMu.Unlock()
	registerGraphRules(ev)
	ev.log.Info("[GRAPH-RULES] Reloaded graph rule index")
}
