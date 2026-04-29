package api

// Hot/Warm/Cold storage tiering REST surface (Phase 22.3).
//
// Exposes:
//   GET /api/v1/storage/tiering/stats — per-tier size estimate +
//     last migration cycle stats (hot→warm count, warm→cold count,
//     errors, duration). Read-only; admin-or-analyst-gated.
//   POST /api/v1/storage/tiering/promote — operator override that
//     fires a migration cycle right now (instead of waiting for the
//     next scheduled tick). Admin-only. Returns the resulting stats.
//
// The tier instances + migrator are owned by `internal/core` and
// injected via SetTieringProvider after server construction (same
// import-cycle-avoidance pattern as SuppressionProvider /
// SettingsProvider). Without the injection these endpoints return
// 503 — useful for the air-gap deployment that doesn't configure
// any tiering at all.

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
)

// TierStatProvider returns size estimates for a single tier. Mirrors
// `tiering.Tier.EstimatedSize` without forcing the api package to
// import the tiering subpackage directly.
type TierStatProvider interface {
	ID() string                                                    // "hot" | "warm" | "cold"
	EstimatedSize(ctx context.Context) (int64, error)              // bytes; -1 = unknown
}

// TierMigrationProvider returns the most recent migration cycle's
// stats, plus an on-demand RunOnce trigger for the operator's
// "Promote now" button.
type TierMigrationProvider interface {
	LastCycle() (TierMigrationStats, bool) // false if migrator has never run
	RunOnce(ctx context.Context) TierMigrationStats
}

// TierMigrationStats is the wire shape of a single cycle. Concrete
// `*tiering.Migrator` returns `tiering.MigrationStats`; the container
// wraps it through this lighter interface to keep the api package's
// dependency surface small.
type TierMigrationStats struct {
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	HotToWarm   int       `json:"hot_to_warm"`
	WarmToCold  int       `json:"warm_to_cold"`
	Errors      []string  `json:"errors,omitempty"`
}

// SetTieringProvider wires the tier-stat + migration accessors. Pass
// nil for either to disable the corresponding endpoint subset
// (e.g. air-gap with cold tier omitted).
func (s *RESTServer) SetTieringProvider(tiers []TierStatProvider, mig TierMigrationProvider) {
	s.tierStats = tiers
	s.tierMig = mig
}

// handleTieringStats serves GET /api/v1/storage/tiering/stats.
// Returns:
//
//	{
//	  "tiers": [
//	    { "id": "hot",  "size_bytes": 1234567 },
//	    { "id": "warm", "size_bytes":   45678 },
//	    { "id": "cold", "size_bytes":      0  }
//	  ],
//	  "last_cycle": { ...TierMigrationStats... } | null,
//	  "ok": true
//	}
//
// `size_bytes: -1` means "tier reports size unknown" (Badger may not
// have populated stats yet on a fresh install). Frontend renders "—"
// instead of "0 B" for that case.
func (s *RESTServer) handleTieringStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	// Read-only endpoint — analyst, admin, and read-only viewer all
	// see this for the storage dashboard.
	if role != auth.RoleAdmin && role != auth.RoleAnalyst && role != auth.RoleReadOnly {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if s.tierStats == nil && s.tierMig == nil {
		http.Error(w, "Storage tiering not configured", http.StatusServiceUnavailable)
		return
	}

	type tierEntry struct {
		ID        string `json:"id"`
		SizeBytes int64  `json:"size_bytes"`
	}
	tiers := make([]tierEntry, 0, len(s.tierStats))
	ctx := r.Context()
	for _, t := range s.tierStats {
		if t == nil {
			continue
		}
		size, err := t.EstimatedSize(ctx)
		if err != nil {
			// Don't fail the whole response for a single tier — log
			// the failure and report -1 so the UI can show "—".
			s.log.Warn("[tiering] EstimatedSize(%s) failed: %v", t.ID(), err)
			size = -1
		}
		tiers = append(tiers, tierEntry{ID: t.ID(), SizeBytes: size})
	}

	resp := map[string]any{
		"ok":    true,
		"tiers": tiers,
	}
	if s.tierMig != nil {
		if last, ok := s.tierMig.LastCycle(); ok {
			resp["last_cycle"] = last
		} else {
			resp["last_cycle"] = nil
		}
	}
	s.jsonResponse(w, http.StatusOK, resp)
}

// handleTieringPromote serves POST /api/v1/storage/tiering/promote.
// Admin-only — manual trigger of a migration cycle. Returns the
// resulting stats so the operator can confirm the cycle ran.
//
// This is the operator's "Promote now" button — useful when an
// operator just changed retention policy and wants the new threshold
// applied immediately instead of waiting up to an hour for the next
// scheduled cycle.
func (s *RESTServer) handleTieringPromote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden — admin only", http.StatusForbidden)
		return
	}
	if s.tierMig == nil {
		http.Error(w, "Storage tiering not configured", http.StatusServiceUnavailable)
		return
	}

	// Cap the request-driven cycle at 5 minutes so a stuck migration
	// doesn't hold an admin's HTTP request open forever; the Migrator
	// itself respects this context and returns whatever it managed.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	stats := s.tierMig.RunOnce(ctx)

	// Audit log the manual promotion — destructive class because it
	// moves data between storage tiers.
	actor := connectorActor(r)
	s.appendAuditEntry(actor, "storage.tiering.promote.manual",
		"hot+warm",
		jsonMustMarshal(map[string]any{
			"hot_to_warm":  stats.HotToWarm,
			"warm_to_cold": stats.WarmToCold,
			"errors":       stats.Errors,
		}),
		r,
	)
	if s.bus != nil {
		s.bus.Publish("storage:tiering_manual_promote", map[string]any{
			"actor":        actor,
			"hot_to_warm":  stats.HotToWarm,
			"warm_to_cold": stats.WarmToCold,
			"event_type":   "destructive_action",
		})
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"ok":    true,
		"cycle": stats,
	})
}

// jsonMustMarshal is a tiny helper for audit-detail formatting; we
// don't want json errors to bubble into the audit row.
func jsonMustMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
