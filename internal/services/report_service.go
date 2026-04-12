package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"


	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/oql"
	"github.com/kingknull/oblivrashell/internal/platform"
)

// ReportService orchestrates the automated generation and scheduling of reports.
type ReportService struct {
	BaseService
	repo       database.ReportStore
	analytics  *AnalyticsService
	tenantRepo *database.TenantRepository
	log        *logger.Logger
	stop       chan struct{}
}

func NewReportService(
	repo database.ReportStore,
	analytics *AnalyticsService,
	tenantRepo *database.TenantRepository,
	log *logger.Logger,
) *ReportService {
	return &ReportService{
		repo:       repo,
		analytics:  analytics,
		tenantRepo: tenantRepo,
		log:        log.WithPrefix("report-factory"),
		stop:       make(chan struct{}),
	}
}

func (s *ReportService) Name() string { return "report-factory-service" }

func (s *ReportService) Start(ctx context.Context) error {
	s.log.Info("Report Factory starting...")
	go s.run(ctx)
	return nil
}

func (s *ReportService) Stop(ctx context.Context) error {
	s.log.Info("Report Factory shutting down...")
	close(s.stop)
	return nil
}

func (s *ReportService) run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processSchedules(ctx)
		case <-s.stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *ReportService) processSchedules(ctx context.Context) {
	schedules, err := s.repo.GetDueSchedules(ctx)
	if err != nil {
		s.log.Error("Failed to fetch due schedules: %v", err)
		return
	}

	for _, sch := range schedules {
		s.log.Info("Processing due schedule %s (%s) for tenant %s", sch.Name, sch.ID, sch.TenantID)
		
		tenantCtx := database.WithTenant(ctx, sch.TenantID)
		
		// Run in background to not block the ticker
		go s.executeScheduledReport(tenantCtx, sch)
	}
}

func (s *ReportService) executeScheduledReport(ctx context.Context, sch database.ReportSchedule) {
	start := time.Now().Add(-time.Duration(sch.IntervalMins) * time.Minute).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)

	instance := &database.GeneratedReport{
		ScheduleID:  sch.ID,
		TemplateID:  sch.TemplateID,
		Title:       fmt.Sprintf("%s - %s", sch.Name, time.Now().Format("2006-01-02")),
		PeriodStart: start,
		PeriodEnd:   end,
		Status:      "pending",
	}

	if err := s.repo.CreateReportInstance(ctx, instance); err != nil {
		s.log.Error("Failed to record report instance for %s: %v", sch.ID, err)
		return
	}

	filePath, err := s.GenerateReport(ctx, sch.TemplateID, start, end)
	if err != nil {
		s.log.Error("Report generation failed for %s: %v", sch.ID, err)
		return
	}

	// Update schedule metadata
	s.repo.MarkScheduleRun(ctx, sch.ID)
	
	s.log.Info("Successfully generated scheduled report: %s", filePath)
}

// Management Methods

func (s *ReportService) ListTemplates(ctx context.Context) ([]database.ReportTemplate, error) {
	return s.repo.ListTemplates(ctx)
}

func (s *ReportService) CreateTemplate(ctx context.Context, t *database.ReportTemplate) error {
	return s.repo.CreateTemplate(ctx, t)
}

func (s *ReportService) ListGeneratedReports(ctx context.Context, limit int) ([]database.GeneratedReport, error) {
	return s.repo.ListReports(ctx, limit)
}

func (s *ReportService) GenerateManualReport(ctx context.Context, templateID string, start, end string) (string, error) {
	// 1. Create instance record
	instance := &database.GeneratedReport{
		TemplateID:  templateID,
		Title:       fmt.Sprintf("Manual Report - %s", time.Now().Format("2006-01-02 15:04")),
		PeriodStart: start,
		PeriodEnd:   end,
		Status:      "pending",
	}
	s.repo.CreateReportInstance(ctx, instance)

	path, err := s.GenerateReport(ctx, templateID, start, end)
	if err != nil {
		return "", err
	}
	
	return path, nil
}

func (s *ReportService) GetReportPath(ctx context.Context, id string) (string, error) {
	// For MVP, we use the ID as the filename relative to the tenant's report dir
	// In a full implementation, we'd lookup the instance in the DB.
	tenantID := database.TenantFromContext(ctx)
	reportsDir := filepath.Join(platform.DataDir(), "reports", tenantID)
	
	// Security: validate ID doesn't contain path traversal
	if strings.Contains(id, "..") || strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return "", fmt.Errorf("invalid report ID")
	}

	return filepath.Join(reportsDir, id), nil
}

// GenerateReport creates a report based on a template and date range. Return file path.
func (s *ReportService) GenerateReport(ctx context.Context, templateID string, start, end string) (string, error) {
	tmpl, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return "", err
	}

	var sections []database.GenericReportSection
	if err := json.Unmarshal([]byte(tmpl.SectionsJSON), &sections); err != nil {
		return "", fmt.Errorf("malformed template sections: %w", err)
	}

	type SectionResult struct {
		Title string
		Type  string
		Data  *oql.QueryResult
	}

	var results []SectionResult
	for _, sec := range sections {
		res, err := s.analytics.RunOQL(ctx, sec.Query)
		if err != nil {
			s.log.Warn("OQL Error in report section %s: %v", sec.Title, err)
			continue
		}
		results = append(results, SectionResult{
			Title: sec.Title,
			Type:  sec.Type,
			Data:  res,
		})
	}

	// Render HTML
	htmlData, err := s.renderHTML(tmpl.Name, start, end, results)
	if err != nil {
		return "", err
	}

	// Save to disk
	reportsDir := filepath.Join(platform.DataDir(), "reports", tmpl.TenantID)
	os.MkdirAll(reportsDir, 0700)
	
	filename := fmt.Sprintf("%s_%s.html", templateID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(reportsDir, filename)
	
	if err := os.WriteFile(filePath, htmlData, 0600); err != nil {
		return "", err
	}

	return filePath, nil
}

func (s *ReportService) renderHTML(title, start, end string, results interface{}) ([]byte, error) {
	const reportTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }} - OBLIVRA Report</title>
    <style>
        body { font-family: sans-serif; background: #0f1117; color: #e4e6f0; padding: 40px; }
        .header { border-bottom: 2px solid #6366f1; margin-bottom: 30px; padding-bottom: 20px; }
        .section { background: #1a1d27; border-radius: 8px; padding: 20px; margin-bottom: 20px; border: 1px solid #2a2d3a; }
        h1 { color: #6366f1; }
        h2 { border-left: 4px solid #6366f1; padding-left: 10px; font-size: 1.2rem; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        th, td { text-align: left; padding: 10px; border-bottom: 1px solid #2a2d3a; }
        th { background: #2a2d3a; font-size: 0.8rem; text-transform: uppercase; color: #8b8fa3; }
        .meta { color: #8b8fa3; font-size: 0.9rem; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{ .Title }}</h1>
        <div class="meta">OBLIVRA Master Report Factory • Period: {{ .Start }} to {{ .End }}</div>
    </div>

    {{ range .Results }}
    <div class="section">
        <h2>{{ .Title }}</h2>
        {{ if eq .Type "table" }}
            <table>
                <thead>
                    <tr>
                        {{ range .Data.Columns }}
                        <th>{{ .Name }}</th>
                        {{ end }}
                    </tr>
                </thead>
                <tbody>
                    {{ range .Data.Rows }}
                    <tr>
                        {{ $row := . }}
                        {{ range $.Data.Columns }}
                        <td>{{ index $row .Name }}</td>
                        {{ end }}
                    </tr>
                    {{ end }}
                </tbody>
            </table>
        {{ else if eq .Type "summary" }}
            {{ range .Data.Rows }}
               <div style="font-size: 1.2rem; font-weight: bold; margin-top: 10px;">{{ index . "count" }} Total Events</div>
            {{ end }}
        {{ end }}
    </div>
    {{ end }}

    <div style="text-align: center; margin-top: 40px; color: #8b8fa3; font-size: 0.8rem;">
        Generated by OBLIVRA Security Platform • {{ now }}
    </div>
</body>
</html>`

	tmpl, err := template.New("factory-report").Funcs(template.FuncMap{
		"now": func() string { return time.Now().Format(time.RFC3339) },
	}).Parse(reportTemplate)
	if err != nil {
		return nil, err
	}

	data := struct {
		Title   string
		Start   string
		End     string
		Results interface{}
	}{
		Title:   title,
		Start:   start,
		End:     end,
		Results: results,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExportPDF placeholder using gofpdf
func (s *ReportService) ExportPDF(htmlPath string) (string, error) {
	// In production, we'd use a headless browser or library to convert HTML to PDF
	// For OBLIVRA MVP, we provide HTML viewing which is superior for interactive charts.
	return htmlPath, nil
}
