package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/security"
)

func (s *RESTServer) handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	// Require Admin role
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}

	if s.tenantRepo == nil {
		http.Error(w, "Tenant functionality not initialized", http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name string `json:"name"`
			Tier string `json:"tier"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}

		salt := security.GenerateTenantSalt()
		tenant := &database.Tenant{
			Name:       req.Name,
			Tier:       req.Tier,
			Status:     "Active",
			CryptoSalt: salt,
		}

		if err := s.tenantRepo.CreateTenant(r.Context(), tenant); err != nil {
			http.Error(w, fmt.Sprintf("Failed to create tenant: %v", err), http.StatusInternalServerError)
			return
		}

		// Conceptually derived keys and index map exist when indexing starts.
		// Idempotent keyspace/index creation happens implicitly in Badger/Bleve on first write.

		s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
			"id":     tenant.ID,
			"name":   tenant.Name,
			"status": tenant.Status,
		})
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RESTServer) handleAdminTenantWipe(w http.ResponseWriter, r *http.Request) {
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.tenantRepo == nil {
		http.Error(w, "Tenant functionality not initialized", http.StatusInternalServerError)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/tenants/")
	if id == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	// Capture identifying state BEFORE the wipe so the audit record can name
	// what was deleted. Once CryptographicWipe runs, name+tier are no longer
	// recoverable from the deleted tenant alone.
	pre, getErr := s.tenantRepo.GetTenant(r.Context(), id)
	if getErr != nil {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	// Optional body: { "reason": "GDPR Art. 17 request from data subject ABC123" }
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // body is optional

	actor := auth.UserFromContext(r.Context())
	actorID, actorEmail := "", ""
	if actor != nil {
		actorID = actor.ID
		actorEmail = actor.Email
	}

	wipeErr := s.tenantRepo.CryptographicWipe(r.Context(), id)

	// Always emit an audit record — both success and failure paths must leave
	// evidence (GDPR Art. 30 records of processing; an attempted-and-failed
	// erasure is itself a regulator-relevant event).
	if s.audit != nil {
		eventType := "tenant.deleted"
		if wipeErr != nil {
			eventType = "tenant.delete_failed"
		}
		details := map[string]interface{}{
			"tenant_id":     id,
			"tenant_name":   pre.Name,
			"tenant_tier":   pre.Tier,
			"deleted_at":    time.Now().UTC().Format(time.RFC3339),
			"actor_user_id": actorID,
			"actor_email":   actorEmail,
			"actor_ip":      getClientIP(r),
			"reason":        body.Reason,
			"basis":         "operator-initiated cryptographic wipe (GDPR Art. 17)",
		}
		if wipeErr != nil {
			details["error"] = wipeErr.Error()
		}
		// audit.Log records a Merkle-chained, append-only log entry. The chain
		// hash makes any later tampering with this record detectable on
		// InitIntegrity replay at next boot.
		if logErr := s.audit.Log(r.Context(), eventType, "", "", details); logErr != nil {
			s.log.Error("[REST] tenant wipe: audit log failed: %v", logErr)
		}
	}

	if s.bus != nil {
		s.bus.Publish("tenant:deleted", map[string]interface{}{
			"tenant_id":   id,
			"tenant_name": pre.Name,
			"actor":       actorEmail,
			"deleted_at":  time.Now().UTC().Format(time.RFC3339),
			"success":     wipeErr == nil,
		})
	}

	if wipeErr != nil {
		s.log.Error("[REST] tenant wipe failed for tenant=%s actor=%s: %v", id, actorEmail, wipeErr)
		http.Error(w, "Failed to wipe tenant", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"status":     "wiped",
		"tenant_id":  id,
		"deleted_at": time.Now().UTC().Format(time.RFC3339),
	})
}
