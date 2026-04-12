package api

// explained_matches.go — REST endpoint for the detection explainability ring buffer.
//
// Surfaces "why did this rule fire?" data to the DecisionInspector frontend page.
// Subscribes to "detection.explained_match" on the event bus and keeps the last
// 500 entries in memory, accessible at GET /api/v1/detection/explained.

import (
	"net/http"
	"sync"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/eventbus"
)

const explainedMatchRingSize = 500

// explainedMatchRing is a process-global ring buffer of recent explained matches.
// We keep it package-level so it survives RESTServer reconstruction in tests.
var (
	explainedRingMu sync.RWMutex
	explainedRing   []detection.ExplainedMatch
)

// SubscribeExplainedMatches registers the ring buffer subscriber on the event bus.
// Call once during application startup, after the bus is initialised.
func SubscribeExplainedMatches(bus *eventbus.Bus) {
	bus.Subscribe("detection.explained_match", func(ev eventbus.Event) {
		m, ok := ev.Data.(detection.ExplainedMatch)
		if !ok {
			return
		}
		explainedRingMu.Lock()
		explainedRing = append(explainedRing, m)
		if len(explainedRing) > explainedMatchRingSize {
			// Trim oldest entries — keep the ring bounded
			explainedRing = explainedRing[len(explainedRing)-explainedMatchRingSize:]
		}
		explainedRingMu.Unlock()
	})
}

// handleExplainedMatches serves the ring buffer to the frontend.
//
// GET /api/v1/detection/explained
//
// Query params:
//   limit=N      — return at most N entries (default 100, max 500)
//   tenant=X     — filter by tenant ID (admin can pass any; non-admin is auto-scoped)
//   severity=X   — filter by severity (CRITICAL, HIGH, MEDIUM, LOW)
//   rule_id=X    — filter by rule ID
func (s *RESTServer) handleExplainedMatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// RBAC: Analysts and Admins only
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin {
		http.Error(w, "Forbidden: requires Analyst or Admin role", http.StatusForbidden)
		return
	}

	// Parse query params
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		_, _ = intFromString(v, &limit)
	}
	if limit > explainedMatchRingSize {
		limit = explainedMatchRingSize
	}

	filterSeverity := r.URL.Query().Get("severity")
	filterRuleID   := r.URL.Query().Get("rule_id")

	// Tenant scope: non-admin users only see their own tenant's matches
	callerTenant := TenantFromContext(r.Context())

	// Snapshot under read lock
	explainedRingMu.RLock()
	snapshot := make([]detection.ExplainedMatch, len(explainedRing))
	copy(snapshot, explainedRing)
	explainedRingMu.RUnlock()

	// Apply filters + reverse (most recent first)
	out := make([]detection.ExplainedMatch, 0, limit)
	for i := len(snapshot) - 1; i >= 0 && len(out) < limit; i-- {
		m := snapshot[i]

		// Tenant isolation
		if callerTenant != "" && m.TenantID != "" && m.TenantID != callerTenant {
			continue
		}
		// Severity filter
		if filterSeverity != "" && m.Severity != filterSeverity {
			continue
		}
		// Rule ID filter
		if filterRuleID != "" && m.RuleID != filterRuleID {
			continue
		}

		out = append(out, m)
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"count":   len(out),
		"matches": out,
	})
}

// intFromString parses an integer from a string, writing into *dst.
// Returns false if parsing failed (dst is unchanged).
func intFromString(s string, dst *int) (int, bool) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	*dst = n
	return n, true
}
