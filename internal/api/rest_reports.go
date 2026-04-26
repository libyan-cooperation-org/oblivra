package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
)

// handleReportTemplates GET /api/v1/reports/templates, POST /api/v1/reports/templates
func (s *RESTServer) handleReportTemplates(w http.ResponseWriter, r *http.Request) {
	if s.reports == nil {
		http.Error(w, "Report service not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		templates, err := s.reports.ListTemplates(r.Context())
		if err != nil {
			s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
			return
		}
		s.jsonResponse(w, http.StatusOK, templates)

	case http.MethodPost:
		var t database.ReportTemplate
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, "Invalid template payload", http.StatusBadRequest)
			return
		}
		if err := s.reports.CreateTemplate(r.Context(), &t); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
			return
		}
		s.jsonResponse(w, http.StatusCreated, t)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGeneratedReports GET /api/v1/reports/generated
func (s *RESTServer) handleGeneratedReports(w http.ResponseWriter, r *http.Request) {
	if s.reports == nil {
		http.Error(w, "Report service not available", http.StatusServiceUnavailable)
		return
	}

	reports, err := s.reports.ListGeneratedReports(r.Context(), 50)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
		return
	}
	s.jsonResponse(w, http.StatusOK, reports)
}

// handleReportGenerate POST /api/v1/reports/generate
func (s *RESTServer) handleReportGenerate(w http.ResponseWriter, r *http.Request) {
	if s.reports == nil {
		http.Error(w, "Report service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		TemplateID string `json:"template_id"`
		Start      string `json:"start"`
		End        string `json:"end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	path, err := s.reports.GenerateManualReport(r.Context(), req.TemplateID, req.Start, req.End)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
		return
	}

	s.jsonResponse(w, http.StatusAccepted, map[string]string{"path": path})
}

// handleReportView GET /api/v1/reports/view/{id}
func (s *RESTServer) handleReportView(w http.ResponseWriter, r *http.Request) {
	if s.reports == nil {
		http.Error(w, "Report service not available", http.StatusServiceUnavailable)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/reports/view/")
	if id == "" {
		http.Error(w, "Report ID required", http.StatusBadRequest)
		return
	}

	// For security, normally we'd fetch the instance to get the path
	// but for this implementation we assume the ID is safe or we use a dedicated viewer
	path, err := s.reports.GetReportPath(r.Context(), id)
	if err != nil {
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "Failed to read report file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(content)
}
