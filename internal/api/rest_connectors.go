package api

// Browser-mode REST endpoints for identity connector management.
// The desktop app uses Wails-bound IdentityService directly; web /
// remote operators get the same shape via these handlers.
//
// Routes:
//   GET    /api/v1/identity/connectors        — list (admin-gated)
//   POST   /api/v1/identity/connectors        — create
//   GET    /api/v1/identity/connectors/:id    — fetch one
//   PATCH  /api/v1/identity/connectors/:id    — update
//   DELETE /api/v1/identity/connectors/:id    — delete

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
)

// errAdminOnly is returned by connectorsRequireAdmin when the request
// context lacks RoleAdmin. Local sentinel — connector endpoints are
// the only callers.
var errAdminOnly = errors.New("admin-only endpoint")

// connectorsRequireAdmin guards every connector endpoint — federated
// auth config is admin-only.
func (s *RESTServer) connectorsRequireAdmin(r *http.Request) error {
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		return errAdminOnly
	}
	return nil
}

// handleConnectorsCollection dispatches GET /list + POST /create.
func (s *RESTServer) handleConnectorsCollection(w http.ResponseWriter, r *http.Request) {
	if err := s.connectorsRequireAdmin(r); err != nil {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleConnectorsList(w, r)
	case http.MethodPost:
		s.handleConnectorsCreate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConnectorByID dispatches GET/PATCH/DELETE /:id.
func (s *RESTServer) handleConnectorByID(w http.ResponseWriter, r *http.Request) {
	if err := s.connectorsRequireAdmin(r); err != nil {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/identity/connectors/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		c, err := s.identity.GetConnector(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		s.jsonResponse(w, http.StatusOK, c)
	case http.MethodPatch, http.MethodPut:
		s.handleConnectorsUpdate(w, r, id)
	case http.MethodDelete:
		if err := s.identity.DeleteConnector(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.appendAuditEntry(connectorActor(r), "identity.connector.delete", id, "ok", r)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RESTServer) handleConnectorsList(w http.ResponseWriter, r *http.Request) {
	connectors, err := s.identity.ListConnectors(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if connectors == nil {
		connectors = []database.IdentityConnector{}
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{"connectors": connectors})
}

func (s *RESTServer) handleConnectorsCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var c database.IdentityConnector
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if c.Name == "" || c.Type == "" {
		http.Error(w, "name and type are required", http.StatusBadRequest)
		return
	}
	if t := strings.ToLower(c.Type); t != "oidc" && t != "saml" && t != "okta" && t != "azure_ad" && t != "ldap" {
		http.Error(w, "type must be one of: oidc, saml, okta, azure_ad, ldap", http.StatusBadRequest)
		return
	}
	if err := s.identity.CreateConnector(r.Context(), &c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.appendAuditEntry(connectorActor(r), "identity.connector.create", c.ID, c.Type+":"+c.Name, r)
	s.jsonResponse(w, http.StatusCreated, c)
}

func (s *RESTServer) handleConnectorsUpdate(w http.ResponseWriter, r *http.Request, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var c database.IdentityConnector
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	c.ID = id // path-id is canonical; body's id is ignored to prevent rename-via-PATCH
	if err := s.identity.UpdateConnector(r.Context(), &c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.appendAuditEntry(connectorActor(r), "identity.connector.update", c.ID, c.Type+":"+c.Name, r)
	s.jsonResponse(w, http.StatusOK, c)
}

// connectorActor returns the operator email for audit-log attribution.
func connectorActor(r *http.Request) string {
	if u := auth.UserFromContext(r.Context()); u != nil && u.Email != "" {
		return u.Email
	}
	return "anonymous"
}
