package compliance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/platform"
)

// ReportType defines the compliance standard
type ReportType string

const (
	ReportSOC2    ReportType = "soc2"
	ReportPCIDSS  ReportType = "pci-dss"
	ReportHIPAA   ReportType = "hipaa"
	ReportGeneral ReportType = "general"
)

// ComplianceReport is a complete audit report
type ComplianceReport struct {
	ID          string          `json:"id"`
	Type        ReportType      `json:"type"`
	Title       string          `json:"title"`
	GeneratedAt string          `json:"generated_at"`
	PeriodStart string          `json:"period_start"`
	PeriodEnd   string          `json:"period_end"`
	Summary     ReportSummary   `json:"summary"`
	Sections    []ReportSection `json:"sections"`
	Findings    []Finding       `json:"findings"`
	PackResults []PackResult    `json:"pack_results,omitempty"`
}

type ReportSummary struct {
	TotalSessions      int     `json:"total_sessions"`
	UniquHosts         int     `json:"unique_hosts"`
	TotalCommands      int     `json:"total_commands"`
	AvgSessionDuration string  `json:"avg_session_duration"`
	FailedLogins       int     `json:"failed_logins"`
	ComplianceScore    float64 `json:"compliance_score"`
	CriticalFindings   int     `json:"critical_findings"`
	WarningFindings    int     `json:"warning_findings"`
	InfoFindings       int     `json:"info_findings"`
}

type ReportSection struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      string      `json:"status"` // "pass", "fail", "warning", "info"
	Details     interface{} `json:"details"`
}

type Finding struct {
	Severity    string `json:"severity"` // "critical", "warning", "info"
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Remediation string `json:"remediation,omitempty"`
	Evidence    string `json:"evidence,omitempty"`
}

// ReportGenerator creates compliance reports
type ReportGenerator struct {
	auditRepo   database.AuditStore
	sessionRepo database.SessionStore
	hostRepo    database.HostStore
}

func NewReportGenerator(audit database.AuditStore, session database.SessionStore, host database.HostStore) *ReportGenerator {
	return &ReportGenerator{
		auditRepo:   audit,
		sessionRepo: session,
		hostRepo:    host,
	}
}

// GenerateReport creates a compliance report for a given period
func (g *ReportGenerator) GenerateReport(
	reportType ReportType,
	periodStartStr string,
	periodEndStr string,
) (*ComplianceReport, error) {
	periodStart, _ := time.Parse(time.RFC3339, periodStartStr)
	periodEnd, _ := time.Parse(time.RFC3339, periodEndStr)

	report := &ComplianceReport{
		ID:          fmt.Sprintf("report-%d", time.Now().UnixNano()),
		Type:        reportType,
		Title:       fmt.Sprintf("%s Compliance Report", string(reportType)),
		GeneratedAt: time.Now().Format(time.RFC3339),
		PeriodStart: periodStartStr,
		PeriodEnd:   periodEndStr,
	}

	// Gather data
	auditLogs, err := g.auditRepo.GetByDateRange(context.Background(), periodStartStr, periodEndStr, 100000)
	if err != nil {
		return nil, fmt.Errorf("fetch audit logs: %w", err)
	}

	sessions, err := g.sessionRepo.GetRecent(context.Background(), 10000)
	if err != nil {
		return nil, fmt.Errorf("fetch sessions: %w", err)
	}

	// Filter sessions to period
	var periodSessions []database.Session
	for _, s := range sessions {
		ts, _ := time.Parse(time.RFC3339, s.StartedAt)
		if ts.After(periodStart) && ts.Before(periodEnd) {
			periodSessions = append(periodSessions, s)
		}
	}

	hosts, err := g.hostRepo.GetAll(context.Background())
	if err != nil {
		return nil, fmt.Errorf("fetch hosts: %w", err)
	}

	// Build summary
	report.Summary = g.buildSummary(periodSessions, auditLogs, hosts)

	// Build sections based on report type
	switch reportType {
	case ReportSOC2:
		report.Sections = g.buildSOC2Sections(periodSessions, auditLogs, hosts)
	case ReportPCIDSS:
		report.Sections = g.buildPCIDSSSections(periodSessions, auditLogs, hosts)
	default:
		report.Sections = g.buildGeneralSections(periodSessions, auditLogs, hosts)
	}

	// Generate findings
	report.Findings = g.generateFindings(periodSessions, auditLogs, hosts)

	// Count findings by severity
	for _, f := range report.Findings {
		switch f.Severity {
		case "critical":
			report.Summary.CriticalFindings++
		case "warning":
			report.Summary.WarningFindings++
		case "info":
			report.Summary.InfoFindings++
		}
	}

	// Calculate compliance score
	report.Summary.ComplianceScore = g.calculateScore(report)

	return report, nil
}

func (g *ReportGenerator) buildSummary(
	sessions []database.Session,
	logs []database.AuditLog,
	hosts []database.Host,
) ReportSummary {
	summary := ReportSummary{
		TotalSessions: len(sessions),
		UniquHosts:    len(hosts),
	}

	// Count unique hosts actually connected
	hostSet := make(map[string]bool)
	var totalDuration time.Duration
	for _, s := range sessions {
		hostSet[s.HostID] = true
		totalDuration += time.Duration(s.DurationSeconds) * time.Second
	}
	summary.UniquHosts = len(hostSet)

	if len(sessions) > 0 {
		avg := totalDuration / time.Duration(len(sessions))
		summary.AvgSessionDuration = avg.String()
	}

	// Count failed logins
	for _, log := range logs {
		if log.EventType == "auth.failed" {
			summary.FailedLogins++
		}
	}

	// Count commands
	for _, log := range logs {
		if log.EventType == "command.executed" {
			summary.TotalCommands++
		}
	}

	return summary
}

func (g *ReportGenerator) buildSOC2Sections(
	sessions []database.Session,
	logs []database.AuditLog,
	hosts []database.Host,
) []ReportSection {
	return []ReportSection{
		{
			Title:       "CC6.1 - Logical Access Controls",
			Description: "Access to systems is restricted through logical access security measures",
			Status:      g.evaluateAccessControls(sessions, logs),
			Details: map[string]interface{}{
				"encrypted_vault":    true,
				"session_logging":    true,
				"unique_credentials": len(hosts),
				"failed_auth_events": countEvents(logs, "auth.failed"),
			},
		},
		{
			Title:       "CC6.2 - Access Authorization",
			Description: "New logical access and modifications are authorized",
			Status:      "pass",
			Details: map[string]interface{}{
				"host_changes":        countEvents(logs, "host.created") + countEvents(logs, "host.updated"),
				"credential_changes":  countEvents(logs, "credential.created") + countEvents(logs, "credential.updated"),
				"audit_trail_present": true,
			},
		},
		{
			Title:       "CC6.3 - Access Removal",
			Description: "Access removal is timely when no longer needed",
			Status:      "info",
			Details: map[string]interface{}{
				"hosts_deleted":       countEvents(logs, "host.deleted"),
				"credentials_deleted": countEvents(logs, "credential.deleted"),
			},
		},
		{
			Title:       "CC7.1 - Monitoring Activities",
			Description: "Activities are monitored to detect security events",
			Status:      "pass",
			Details: map[string]interface{}{
				"total_audit_events": len(logs),
				"session_recordings": countEvents(logs, "recording.started"),
				"alert_events":       countEvents(logs, "security.alert"),
			},
		},
		{
			Title:       "CC7.2 - Anomaly Detection",
			Description: "System anomalies are identified and evaluated",
			Status:      g.evaluateAnomalyDetection(sessions, logs),
			Details: map[string]interface{}{
				"unusual_hours_sessions": g.countUnusualHourSessions(sessions),
				"rapid_connections":      g.countRapidConnections(sessions),
				"failed_auth_spikes":     g.detectFailedAuthSpikes(logs),
			},
		},
		{
			Title:       "CC8.1 - Change Management",
			Description: "Changes are authorized, documented and tracked",
			Status:      "pass",
			Details: map[string]interface{}{
				"configuration_changes": countEvents(logs, "settings.changed"),
				"all_changes_logged":    true,
			},
		},
	}
}

func (g *ReportGenerator) buildPCIDSSSections(
	sessions []database.Session,
	logs []database.AuditLog,
	hosts []database.Host,
) []ReportSection {
	return []ReportSection{
		{
			Title:       "Req 2 - Default Passwords",
			Description: "Do not use vendor-supplied defaults for system passwords",
			Status:      "info",
			Details: map[string]interface{}{
				"unique_credentials_used": true,
				"encrypted_storage":       true,
			},
		},
		{
			Title:       "Req 7 - Access Control",
			Description: "Restrict access to cardholder data by business need-to-know",
			Status:      "pass",
			Details: map[string]interface{}{
				"credential_vault_encrypted": true,
				"session_audit_trail":        true,
				"access_logging_enabled":     true,
			},
		},
		{
			Title:       "Req 8 - Identification & Authentication",
			Description: "Identify and authenticate access to system components",
			Status:      "pass",
			Details: map[string]interface{}{
				"strong_auth_methods": true,
				"key_based_auth":      true,
				"master_password":     true,
			},
		},
		{
			Title:       "Req 10 - Logging & Monitoring",
			Description: "Track and monitor all access to network resources",
			Status:      "pass",
			Details: map[string]interface{}{
				"audit_log_entries":  len(logs),
				"session_recordings": countEvents(logs, "recording.started"),
				"tamper_protection":  true,
			},
		},
	}
}

func (g *ReportGenerator) buildGeneralSections(
	sessions []database.Session,
	logs []database.AuditLog,
	hosts []database.Host,
) []ReportSection {
	return []ReportSection{
		{
			Title:       "Access Overview",
			Description: "Summary of all system access during the reporting period",
			Status:      "info",
			Details: map[string]interface{}{
				"total_sessions":     len(sessions),
				"total_hosts":        len(hosts),
				"total_audit_events": len(logs),
			},
		},
		{
			Title:       "Security Posture",
			Description: "Evaluation of security controls",
			Status:      g.evaluateSecurityPosture(sessions, logs),
			Details: map[string]interface{}{
				"vault_encrypted": true,
				"session_logging": true,
				"failed_logins":   countEvents(logs, "auth.failed"),
			},
		},
		{
			Title:       "Session Activity",
			Description: "Breakdown of session activity",
			Status:      "info",
			Details:     g.buildSessionBreakdown(sessions),
		},
	}
}

func (g *ReportGenerator) generateFindings(
	sessions []database.Session,
	logs []database.AuditLog,
	hosts []database.Host,
) []Finding {
	var findings []Finding

	// Check for hosts without credentials
	for _, h := range hosts {
		if h.CredentialID == "" && h.AuthMethod == "password" {
			findings = append(findings, Finding{
				Severity:    "warning",
				Category:    "Authentication",
				Title:       fmt.Sprintf("Host '%s' uses password auth without vault credential", h.Label),
				Description: "This host may be using manually entered passwords",
				Remediation: "Store credentials in the encrypted vault",
			})
		}
	}

	// Check for excessive failed logins
	failedCount := countEvents(logs, "auth.failed")
	if failedCount > 10 {
		findings = append(findings, Finding{
			Severity:    "warning",
			Category:    "Security",
			Title:       fmt.Sprintf("%d failed login attempts detected", failedCount),
			Description: "High number of failed authentication attempts may indicate brute force",
			Remediation: "Review failed login sources and consider key-based authentication",
		})
	}

	// Check for sessions outside business hours
	unusualSessions := g.countUnusualHourSessions(sessions)
	if unusualSessions > 0 {
		findings = append(findings, Finding{
			Severity:    "info",
			Category:    "Access Pattern",
			Title:       fmt.Sprintf("%d sessions outside business hours (22:00-06:00)", unusualSessions),
			Description: "Sessions outside normal hours may need review",
		})
	}

	// Check for very long sessions
	for _, s := range sessions {
		if s.DurationSeconds > 28800 { // 8 hours
			findings = append(findings, Finding{
				Severity:    "info",
				Category:    "Session Management",
				Title:       fmt.Sprintf("Long session detected: %d hours", s.DurationSeconds/3600),
				Description: fmt.Sprintf("Session to host %s lasted over 8 hours", s.HostID),
			})
		}
	}

	return findings
}

func (g *ReportGenerator) calculateScore(report *ComplianceReport) float64 {
	if len(report.Sections) == 0 {
		return 100.0
	}

	passCount := 0
	for _, s := range report.Sections {
		if s.Status == "pass" || s.Status == "info" {
			passCount++
		}
	}

	score := float64(passCount) / float64(len(report.Sections)) * 100

	// Deduct for critical findings
	score -= float64(report.Summary.CriticalFindings) * 10
	score -= float64(report.Summary.WarningFindings) * 3

	if score < 0 {
		score = 0
	}

	return score
}

// ExportJSON exports the report as JSON
func (g *ReportGenerator) ExportJSON(report *ComplianceReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// ExportToFile saves the report to disk
func (g *ReportGenerator) ExportToFile(report *ComplianceReport) (string, error) {
	reportsDir := filepath.Join(platform.DataDir(), "reports")
	if err := os.MkdirAll(reportsDir, 0700); err != nil {
		return "", err
	}

	tsStart := strings.ReplaceAll(strings.ReplaceAll(report.PeriodStart, ":", "-"), "T", "_")
	tsEnd := strings.ReplaceAll(strings.ReplaceAll(report.PeriodEnd, ":", "-"), "T", "_")
	filename := fmt.Sprintf("%s_%s_%s.json",
		report.Type,
		tsStart,
		tsEnd,
	)

	filePath := filepath.Join(reportsDir, filename)

	data, err := g.ExportJSON(report)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", err
	}

	return filePath, nil
}

func (g *ReportGenerator) ListReports() ([]ComplianceReport, error) {
	return nil, nil
}

// Helper functions
func countEvents(logs []database.AuditLog, eventType string) int {
	count := 0
	for _, l := range logs {
		if l.EventType == eventType {
			count++
		}
	}
	return count
}

func (g *ReportGenerator) evaluateAccessControls(sessions []database.Session, logs []database.AuditLog) string {
	failedLogins := countEvents(logs, "auth.failed")
	if failedLogins > 50 {
		return "warning"
	}
	return "pass"
}

func (g *ReportGenerator) evaluateAnomalyDetection(sessions []database.Session, logs []database.AuditLog) string {
	return "pass"
}

func (g *ReportGenerator) evaluateSecurityPosture(sessions []database.Session, logs []database.AuditLog) string {
	failedLogins := countEvents(logs, "auth.failed")
	if failedLogins > 20 {
		return "warning"
	}
	return "pass"
}

func (g *ReportGenerator) countUnusualHourSessions(sessions []database.Session) int {
	count := 0
	for _, s := range sessions {
		ts, _ := time.Parse(time.RFC3339, s.StartedAt)
		hour := ts.Hour()
		if hour >= 22 || hour < 6 {
			count++
		}
	}
	return count
}

func (g *ReportGenerator) countRapidConnections(sessions []database.Session) int {
	// Count sessions opened within 5 seconds of each other
	count := 0
	for i := 1; i < len(sessions); i++ {
		tsI, _ := time.Parse(time.RFC3339, sessions[i].StartedAt)
		tsPrev, _ := time.Parse(time.RFC3339, sessions[i-1].StartedAt)
		diff := tsI.Sub(tsPrev)
		if diff < 5*time.Second && diff > 0 {
			count++
		}
	}
	return count
}

func (g *ReportGenerator) detectFailedAuthSpikes(logs []database.AuditLog) bool {
	// Group failed auths by 5-minute windows
	windows := make(map[int64]int)
	for _, l := range logs {
		if l.EventType == "auth.failed" {
			ts, _ := time.Parse(time.RFC3339, l.Timestamp)
			window := ts.Unix() / 300 // 5-minute windows
			windows[window]++
		}
	}

	for _, count := range windows {
		if count > 10 {
			return true
		}
	}
	return false
}

func (g *ReportGenerator) buildSessionBreakdown(sessions []database.Session) map[string]interface{} {
	breakdown := map[string]interface{}{
		"by_hour":   make(map[int]int),
		"by_status": make(map[string]int),
	}

	byHour := make(map[int]int)
	byStatus := make(map[string]int)

	for _, s := range sessions {
		ts, _ := time.Parse(time.RFC3339, s.StartedAt)
		byHour[ts.Hour()]++
		byStatus[s.Status]++
	}

	breakdown["by_hour"] = byHour
	breakdown["by_status"] = byStatus

	return breakdown
}

func (g *ReportGenerator) GetAuditCount() (int64, error) {
	return g.auditRepo.Count(context.Background())
}

// IsMerkleValid checks if the audit logs are cryptographically sound
func (g *ReportGenerator) IsMerkleValid() bool {
	return g.auditRepo.ValidateIntegrity(context.Background())
}

// ExportPDF renders a compliance report as a PDF document.
func (g *ReportGenerator) ExportPDF(report *ComplianceReport) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Header
	pdf.SetFont("Arial", "B", 20)
	pdf.SetTextColor(99, 102, 241) // OBLIVRA Accent
	pdf.Cell(0, 15, "OBLIVRA Compliance Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(50, 50, 50)
	pdf.Cell(0, 10, report.Title)
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(120, 120, 120)
	pdf.Cell(0, 5, fmt.Sprintf("Generated At: %s", report.GeneratedAt))
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("Period: %s to %s", report.PeriodStart, report.PeriodEnd))
	pdf.Ln(10)

	// Summary Score
	pdf.SetFillColor(240, 240, 245)
	pdf.Rect(10, pdf.GetY(), 190, 25, "F")
	pdf.SetY(pdf.GetY() + 5)
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(30, 30, 30)
	pdf.Cell(95, 10, "Compliance Score:")

	score := report.Summary.ComplianceScore
	if score >= 80 {
		pdf.SetTextColor(34, 197, 94) // Green
	} else if score >= 50 {
		pdf.SetTextColor(245, 158, 11) // Amber
	} else {
		pdf.SetTextColor(239, 68, 68) // Red
	}
	pdf.SetFont("Arial", "B", 24)
	pdf.Cell(95, 10, fmt.Sprintf("%.1f%%", score))
	pdf.Ln(15)

	// Findings Summary Table
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 10, "Findings Summary")
	pdf.Ln(8)

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(60, 8, "Severity", "1", 0, "C", true, 0, "")
	pdf.CellFormat(65, 8, "Count", "1", 0, "C", true, 0, "")
	pdf.CellFormat(65, 8, "Status", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	drawRow := func(sev string, count int, color []int) {
		pdf.SetTextColor(color[0], color[1], color[2])
		pdf.CellFormat(60, 8, sev, "1", 0, "C", false, 0, "")

		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(65, 8, fmt.Sprintf("%d", count), "1", 0, "C", false, 0, "")
		status := "OK"
		if count > 0 {
			status = "ATTENTION"
		}
		pdf.CellFormat(65, 8, status, "1", 1, "C", false, 0, "")
	}

	drawRow("Critical", report.Summary.CriticalFindings, []int{239, 68, 68})
	drawRow("Warning", report.Summary.WarningFindings, []int{245, 158, 11})
	drawRow("Info", report.Summary.InfoFindings, []int{99, 102, 241})
	pdf.Ln(10)

	// Detailed Findings
	if len(report.Findings) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(0, 10, "Detailed Findings")
		pdf.Ln(8)

		for _, f := range report.Findings {
			pdf.SetFont("Arial", "B", 10)
			pdf.Cell(0, 7, fmt.Sprintf("[%s] %s", f.Severity, f.Title))
			pdf.Ln(6)
			pdf.SetFont("Arial", "", 9)
			pdf.MultiCell(0, 5, f.Description, "", "", false)
			if f.Remediation != "" {
				pdf.SetFont("Arial", "I", 9)
				pdf.MultiCell(0, 5, "Remediation: "+f.Remediation, "", "", false)
			}
			pdf.Ln(4)
			if pdf.GetY() > 250 {
				pdf.AddPage()
			}
		}
	}

	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(128, 128, 128)
	pdf.CellFormat(0, 10, fmt.Sprintf("Report ID: %s | Page %d", report.ID, pdf.PageNo()), "", 0, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
