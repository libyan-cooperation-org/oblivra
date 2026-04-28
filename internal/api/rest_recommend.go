package api

// Next-Best-Action recommender REST endpoint (Phase 32).
//
// POST /api/v1/alerts/recommend — body: { alert_id, severity, category,
//   ... optional facts }, response: detection.RecommendedAction.
//
// We accept POST only because the request body carries the fact-set;
// using GET with query params would force operators to URL-encode JSON
// blobs. The handler is read-only — no DB writes — and 64 KB body cap
// is plenty for the small fact-set struct.

import (
	"encoding/json"
	"net/http"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/detection"
)

// handleAlertRecommend produces a Next-Best-Action recommendation for
// the supplied alert facts. Stateless, fast (<1 ms), safe to call on
// every alert-row render.
func (s *RESTServer) handleAlertRecommend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// RBAC: same gate as the alerts list — analysts and admins only.
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var f detection.NBAFacts
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if f.Severity == "" {
		http.Error(w, "severity is required", http.StatusBadRequest)
		return
	}
	rec := detection.Recommend(f)
	s.jsonResponse(w, http.StatusOK, rec)
}
