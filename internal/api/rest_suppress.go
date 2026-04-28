package api

// Alert suppression REST endpoint (Phase 32 / 33 follow-up).
//
// POST /api/v1/alerts/{id}/suppress
//
// Body: { "reason": string, "scope": "this_alert" | "this_pattern" }
//
// "this_alert"   — creates a one-off suppression rule keyed on alert id
//                  (low-blast-radius; rare).
// "this_pattern" — extracts a pattern from the alert (host_id by default;
//                  caller can override with `field`) and creates a
//                  durable rule that suppresses similar alerts going
//                  forward. Default for the operator's `x` keystroke.
//
// The endpoint persists the rule via SuppressionService.CreateRule,
// which writes to the database — that's how cross-device sync works
// once we move the operator's `x` keystroke off pure localStorage.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
)

type suppressBody struct {
	Reason string `json:"reason"`
	// Scope decides whether we suppress only this alert id or every
	// future alert that shares the chosen field+value.
	Scope string `json:"scope"`
	// Field/Value override the default host_id pattern. If empty we
	// fall back to host_id from the alert metadata.
	Field string `json:"field"`
	Value string `json:"value"`
	// Optional: which detection rule's alerts this suppression applies
	// to. Empty means "any rule" (global suppression on the field/value).
	RuleID string `json:"rule_id"`
	// Optional: ISO-8601 expiration. Empty means permanent.
	ExpiresAt string `json:"expires_at"`
}

// handleAlertSubresource dispatches /api/v1/alerts/{id}/{action}.
// Today only "suppress" is wired; future mutations (acknowledge,
// reassign, comment) hang off the same prefix.
func (s *RESTServer) handleAlertSubresource(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/")
	if strings.HasSuffix(rest, "/suppress") {
		s.handleAlertSuppress(w, r)
		return
	}
	http.Error(w, "Not found", http.StatusNotFound)
}

func (s *RESTServer) handleAlertSuppress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if s.suppression == nil {
		http.Error(w, "Suppression service not available", http.StatusServiceUnavailable)
		return
	}

	// Path: /api/v1/alerts/{id}/suppress
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/")
	id := strings.TrimSuffix(rest, "/suppress")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "alert id required in path", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var body suppressBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Defaults: 'this_alert' is the safer choice when the operator's
	// `x` keystroke fires without a thoughtful pattern selection. The
	// frontend can promote to 'this_pattern' via the future "Edit
	// suppression" affordance.
	if body.Scope == "" {
		body.Scope = "this_alert"
	}
	if body.Reason == "" {
		body.Reason = "Operator-initiated FP suppression"
	}

	rule := &database.SuppressionRule{
		Label:       fmt.Sprintf("FP suppression for %s", id),
		Description: body.Reason,
		IsActive:    true,
		ExpiresAt:   body.ExpiresAt,
	}

	switch body.Scope {
	case "this_alert":
		// Bind to the alert id. Field is `alert_id` so future events
		// matching exactly this id are silenced (use case: alert
		// re-fires after a routing flap).
		rule.Field = "alert_id"
		rule.Value = id
		rule.IsRegex = false
	case "this_pattern":
		// Caller-provided field/value; fall back to host_id.
		if body.Field == "" {
			body.Field = "host_id"
		}
		if body.Value == "" {
			http.Error(w, "value required when scope=this_pattern", http.StatusBadRequest)
			return
		}
		rule.Field = body.Field
		rule.Value = body.Value
		rule.IsRegex = false
		rule.RuleID = body.RuleID
	default:
		http.Error(w, "scope must be 'this_alert' or 'this_pattern'", http.StatusBadRequest)
		return
	}

	ruleID, err := s.suppression.CreateRule(r.Context(), rule)
	if err != nil {
		http.Error(w, "suppression create failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.appendAuditEntry(connectorActor(r), "alert.suppress", id,
		fmt.Sprintf("scope=%s rule=%s reason=%s", body.Scope, ruleID, body.Reason), r)

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"ok":               true,
		"alert_id":         id,
		"suppression_rule": ruleID,
		"scope":            body.Scope,
	})
}
