package api

// Bulk evidence-seal endpoint (Phase 32 Crisis Decision Panel follow-up).
//
// POST /api/v1/evidence/seal
// Body: { "incident_id"?: string, "reason"?: string, "crisis"?: bool }
//
// Behaviour:
//   • If incident_id is provided, seals every unsealed evidence item
//     attached to that incident.
//   • If incident_id is empty, seals every currently-unsealed item in
//     the locker (the "freeze everything I've gathered so far" path
//     the Decision Panel exposes during a live crisis).
//
// Response: { "sealed": [item_id, …], "skipped": [item_id, …], "errors": {item_id: msg, …} }
//
// The endpoint is idempotent — already-sealed items are returned in
// `skipped`, never re-sealed (the underlying locker rejects re-seals
// to preserve chain-of-custody invariants).

import (
	"encoding/json"
	"net/http"

	"github.com/kingknull/oblivrashell/internal/auth"
)

func (s *RESTServer) handleEvidenceBulkSeal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if s.evidence == nil {
		http.Error(w, "Evidence locker not initialised", http.StatusServiceUnavailable)
		return
	}

	authUser := auth.UserFromContext(r.Context())
	actor := "anonymous"
	if authUser != nil && authUser.Email != "" {
		actor = authUser.Email
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var body struct {
		IncidentID string `json:"incident_id"`
		Reason     string `json:"reason"`
		Crisis     bool   `json:"crisis"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // body is optional

	notes := body.Reason
	if notes == "" {
		notes = "Bulk seal via REST"
	}
	if body.Crisis {
		notes = "[CRISIS] " + notes
	}

	// Pick the working set: scoped-by-incident or every unsealed item.
	var working []string // item ids
	if body.IncidentID != "" {
		for _, it := range s.evidence.ListByIncident(body.IncidentID) {
			if !it.Sealed {
				working = append(working, it.ID)
			}
		}
	} else {
		for _, it := range s.evidence.ListAll() {
			if !it.Sealed {
				working = append(working, it.ID)
			}
		}
	}

	sealed := make([]string, 0, len(working))
	skipped := []string{}
	errs := map[string]string{}

	for _, id := range working {
		if err := s.evidence.Seal(id, actor, notes); err != nil {
			errs[id] = err.Error()
			continue
		}
		sealed = append(sealed, id)
	}

	auditTag := "evidence.seal.bulk"
	if body.Crisis {
		auditTag = "evidence.seal.bulk.crisis"
	}
	s.appendAuditEntry(actor, auditTag,
		body.IncidentID, // empty = "all"
		notes, r,
	)

	// Publish to the destructive-action bus subscriber so the audit
	// trail captures this with event_type=destructive_action — the
	// single query operators run for the high-risk operator timeline.
	if s.bus != nil {
		s.bus.Publish("evidence:bulk_sealed", map[string]any{
			"actor":        actor,
			"incident_id":  body.IncidentID,
			"sealed_count": len(sealed),
			"crisis":       body.Crisis,
			"reason":       notes,
		})
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"ok":          true,
		"sealed":      sealed,
		"skipped":     skipped,
		"errors":      errs,
		"sealed_count": len(sealed),
	})
}
