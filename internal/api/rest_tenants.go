package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

	if err := s.tenantRepo.CryptographicWipe(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to wipe tenant: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "wiped"})
}
