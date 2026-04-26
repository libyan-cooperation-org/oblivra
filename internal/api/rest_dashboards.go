package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
)

// handleDashboards GET /api/v1/dashboards, POST /api/v1/dashboards
func (s *RESTServer) handleDashboards(w http.ResponseWriter, r *http.Request) {
	if s.dashboards == nil {
		http.Error(w, "Dashboard service not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		dashboards, err := s.dashboards.ListDashboards(r.Context())
		if err != nil {
			s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
			return
		}
		s.jsonResponse(w, http.StatusOK, dashboards)

	case http.MethodPost:
		var d database.Dashboard
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, "Invalid dashboard payload", http.StatusBadRequest)
			return
		}
		if err := s.dashboards.CreateDashboard(r.Context(), &d); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
			return
		}
		s.jsonResponse(w, http.StatusCreated, d)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDashboardByID GET/PUT/DELETE /api/v1/dashboards/{id}, GET /data, POST /widgets
func (s *RESTServer) handleDashboardByID(w http.ResponseWriter, r *http.Request) {
	if s.dashboards == nil {
		http.Error(w, "Dashboard service not available", http.StatusServiceUnavailable)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/dashboards/")
	if id == "" {
		http.Error(w, "Dashboard ID required", http.StatusBadRequest)
		return
	}

	// Sub-resources
	if strings.HasSuffix(id, "/data") {
		id = strings.TrimSuffix(id, "/data")
		s.handleDashboardData(w, r, id)
		return
	}
	if strings.HasSuffix(id, "/widgets") {
		id = strings.TrimSuffix(id, "/widgets")
		s.handleDashboardWidgets(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		d, err := s.dashboards.GetDashboard(r.Context(), id)
		if err != nil {
			http.Error(w, "Dashboard not found", http.StatusNotFound)
			return
		}
		s.jsonResponse(w, http.StatusOK, d)

	case http.MethodPut:
		var d database.Dashboard
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, "Invalid dashboard payload", http.StatusBadRequest)
			return
		}
		d.ID = id
		if err := s.dashboards.UpdateDashboard(r.Context(), &d); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
			return
		}
		s.jsonResponse(w, http.StatusOK, d)

	case http.MethodDelete:
		if err := s.dashboards.DeleteDashboard(r.Context(), id); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RESTServer) handleDashboardData(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := s.dashboards.GetDashboardData(r.Context(), id)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
		return
	}
	s.jsonResponse(w, http.StatusOK, data)
}

func (s *RESTServer) handleDashboardWidgets(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodPost:
		var widget database.DashboardWidget
		if err := json.NewDecoder(r.Body).Decode(&widget); err != nil {
			http.Error(w, "Invalid widget payload", http.StatusBadRequest)
			return
		}
		widget.DashboardID = id
		if err := s.dashboards.AddWidget(r.Context(), &widget); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
			return
		}
		s.jsonResponse(w, http.StatusCreated, widget)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
