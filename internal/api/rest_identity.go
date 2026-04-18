package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
)

// handleIdentityConnectors GET /api/v1/identity/connectors, POST /api/v1/identity/connectors
func (s *RESTServer) handleIdentityConnectors(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "Identity service not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		connectors, err := s.identity.ListConnectors(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, http.StatusOK, connectors)

	case http.MethodPost:
		var c database.IdentityConnector
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, "Invalid connector payload", http.StatusBadRequest)
			return
		}
		if c.ID == "" {
			c.ID = uuid.New().String()
		}
		c.Status = "new"
		if err := s.identity.CreateConnector(r.Context(), &c); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, http.StatusCreated, c)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleIdentityConnectorByID GET/PUT/DELETE /api/v1/identity/connectors/{id}, POST /sync
func (s *RESTServer) handleIdentityConnectorByID(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "Identity service not available", http.StatusServiceUnavailable)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/identity/connectors/")
	if id == "" {
		http.Error(w, "Connector ID required", http.StatusBadRequest)
		return
	}

	// Handle /sync suffix if present
	if strings.HasSuffix(id, "/sync") {
		id = strings.TrimSuffix(id, "/sync")
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.identity.TriggerSync(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}

	switch r.Method {
	case http.MethodGet:
		c, err := s.identity.GetConnector(r.Context(), id)
		if err != nil {
			http.Error(w, "Connector not found", http.StatusNotFound)
			return
		}
		// Security: Strip config_json for single-item view too (it contains secrets)
		c.ConfigJSON = ""
		s.jsonResponse(w, http.StatusOK, c)

	case http.MethodPut:
		var c database.IdentityConnector
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, "Invalid connector payload", http.StatusBadRequest)
			return
		}
		c.ID = id
		if err := s.identity.UpdateConnector(r.Context(), &c); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, http.StatusOK, c)

	case http.MethodDelete:
		if err := s.identity.DeleteConnector(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
