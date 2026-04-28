package api

// Crisis-lifecycle audit endpoint (Phase 32).
//
// POST /api/v1/crisis/state
// Body: { "active": bool, "reason"?: string }
//
// The frontend's crisisStore arms/stands down locally (operator-driven
// or alert-spike-driven). The bus subscriber in api_service.go listens
// for `crisis:armed` / `crisis:stand_down` events and writes an
// audit_log row with event_type="destructive_action" — but only if
// SOMETHING publishes those events. This handler is that something:
// the frontend pings it on every state transition, the handler
// republishes onto the in-process bus, and the audit subscriber seals
// it.
//
// Why a REST surface and not a direct bus.Publish from the frontend?
// Because the frontend can't reach the in-process eventbus — the only
// IPC channel is HTTP. This is the dedicated frontend→audit seam for
// state changes the platform wants permanently recorded.
//
// We deliberately keep this endpoint forgiving — never error on a
// duplicate state-change ping (the operator's network blip shouldn't
// break crisis arming). We just record the call and move on.

import (
	"encoding/json"
	"net/http"

	"github.com/kingknull/oblivrashell/internal/auth"
)

func (s *RESTServer) handleCrisisState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var body struct {
		Active bool   `json:"active"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	actor := connectorActor(r)

	if body.Active {
		s.appendAuditEntry(actor, "crisis.armed", "", body.Reason, r)
		if s.bus != nil {
			s.bus.Publish("crisis:armed", map[string]any{
				"actor":  actor,
				"reason": body.Reason,
			})
		}
	} else {
		s.appendAuditEntry(actor, "crisis.stand_down", "", body.Reason, r)
		if s.bus != nil {
			s.bus.Publish("crisis:stand_down", map[string]any{
				"actor":  actor,
				"reason": body.Reason,
			})
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"ok": true})
}
