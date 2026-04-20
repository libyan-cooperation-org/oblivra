package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/identity"
)

// GET/POST /api/scim/v2/Users
func (s *RESTServer) handleSCIMUsers(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "Identity service not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		users, err := s.identity.ListUsers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		baseURL := "http://" + r.Host // Should use real base URL in production
		resources := make([]*identity.UserResource, 0, len(users))
		for _, u := range users {
			resources = append(resources, identity.ToUserResource(&u, baseURL))
		}

		resp := identity.ListResponse{
			Schemas:      []string{identity.SchemaList},
			TotalResults: len(resources),
			StartIndex:   1,
			ItemsPerPage: len(resources),
			Resources:    resources,
		}
		s.jsonResponse(w, http.StatusOK, resp)

	case http.MethodPost:
		var res identity.UserResource
		if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
			http.Error(w, "Invalid SCIM payload", http.StatusBadRequest)
			return
		}

		user := &database.User{
			AuthProvider: "scim",
		}
		identity.FromUserResource(&res, user)

		// Quota Enforcement (Phase 25.5)
		if s.license != nil {
			max := s.license.MaxSeats()
			if max > 0 {
				users, err := s.identity.ListUsers(r.Context())
				if err == nil && len(users) >= max {
					s.log.Warn("[licensing] User provisioning DENIED for %s: seat quota exceeded (max=%d)", user.Email, max)
					http.Error(w, "Payment Required: User seat quota exceeded", http.StatusPaymentRequired)
					return
				}
			}
		}

		if err := s.identity.ProvisionSCIMUser(r.Context(), user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		baseURL := "http://" + r.Host
		s.jsonResponse(w, http.StatusCreated, identity.ToUserResource(user, baseURL))

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET/PUT/PATCH/DELETE /api/scim/v2/Users/{id}
func (s *RESTServer) handleSCIMUserByID(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "Identity service not available", http.StatusServiceUnavailable)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/scim/v2/Users/")
	if id == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := s.identity.GetUser(id)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		s.jsonResponse(w, http.StatusOK, identity.ToUserResource(user, "http://"+r.Host))

	case http.MethodPut:
		var res identity.UserResource
		if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
			http.Error(w, "Invalid SCIM payload", http.StatusBadRequest)
			return
		}

		user, err := s.identity.GetUser(id)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		identity.FromUserResource(&res, user)
		if err := s.identity.ProvisionSCIMUser(r.Context(), user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, http.StatusOK, identity.ToUserResource(user, "http://"+r.Host))

	case http.MethodPatch:
		// Simplified PATCH implementation - for MVP we just handle 'active' toggle
		var req struct {
			Operations []struct {
				Op    string          `json:"op"`
				Path  string          `json:"path"`
				Value json.RawMessage `json:"value"`
			} `json:"Operations"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid patch request", http.StatusBadRequest)
			return
		}

		user, err := s.identity.GetUser(id)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		for _, op := range req.Operations {
			if strings.ToLower(op.Op) == "replace" && strings.ToLower(op.Path) == "active" {
				var active bool
				json.Unmarshal(op.Value, &active)
				user.Active = active
			}
		}

		if err := s.identity.ProvisionSCIMUser(r.Context(), user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, http.StatusOK, identity.ToUserResource(user, "http://"+r.Host))

	case http.MethodDelete:
		if err := s.identity.DeleteUser(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSCIMGroups handles SCIM 2.0 Group operations (Stub)
func (s *RESTServer) handleSCIMGroups(w http.ResponseWriter, r *http.Request) {
	resp := identity.ListResponse{
		Schemas:      []string{identity.SchemaList},
		TotalResults: 0,
		StartIndex:   1,
		ItemsPerPage: 0,
		Resources:    []*identity.UserResource{},
	}
	s.jsonResponse(w, http.StatusOK, resp)
}
