package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// OperatorService provides the "Operator Mode" intelligence layer for the terminal.
// It surfaces SIEM alerts, host risk scores, and anomaly data for the currently
// active SSH session — enabling the operator to see security context without
// leaving the terminal.
type OperatorService struct {
	BaseService
	siem  database.SIEMStore
	hosts database.HostStore
	log   *logger.Logger
}

// OperatorContext is the security intelligence overlay for a terminal session.
type OperatorContext struct {
	HostID         string         `json:"host_id"`
	HostLabel      string         `json:"host_label"`
	RiskScore      int            `json:"risk_score"`        // 0-100
	RiskLevel      string         `json:"risk_level"`        // low, medium, high, critical
	AlertCount     int            `json:"alert_count"`       // Active alerts for this host
	RecentAlerts   []HostAlert    `json:"recent_alerts"`     // Last 5 alerts
	FailedLogins   int            `json:"failed_logins"`     // Last 24h
	SuspiciousIPs  []string       `json:"suspicious_ips"`    // Source IPs with failed logins
	LastEvent      string         `json:"last_event"`        // Timestamp of most recent SIEM event
	ThreatSummary  string         `json:"threat_summary"`    // One-line threat assessment
}

// HostAlert is a simplified alert for the operator banner.
type HostAlert struct {
	ID        string `json:"id"`
	Severity  string `json:"severity"`
	EventType string `json:"event_type"`
	Timestamp string `json:"timestamp"`
	SourceIP  string `json:"source_ip"`
	Summary   string `json:"summary"`
}

func (s *OperatorService) Name() string        { return "operator-service" }
func (s *OperatorService) Dependencies() []string { return []string{} }
func (s *OperatorService) Start(ctx context.Context) error { return nil }
func (s *OperatorService) Stop(ctx context.Context) error  { return nil }

// NewOperatorService creates the operator intelligence service.
func NewOperatorService(
	siem database.SIEMStore,
	hosts database.HostStore,
	log *logger.Logger,
) *OperatorService {
	return &OperatorService{
		siem:  siem,
		hosts: hosts,
		log:   log.WithPrefix("operator"),
	}
}

// GetContext returns the full security context for a host.
// This powers the "anomaly banner" on the terminal tab bar.
func (s *OperatorService) GetContext(hostID string) (*OperatorContext, error) {
	if hostID == "" || hostID == "local" {
		return &OperatorContext{
			HostID:    "local",
			HostLabel: "Local Shell",
			RiskLevel: "none",
		}, nil
	}

	ctx := context.Background()

	// Get host info
	host, err := s.hosts.GetByID(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}

	// Calculate risk score
	riskScore, _ := s.siem.CalculateRiskScore(ctx, hostID)

	// Get recent events
	events, _ := s.siem.GetHostEvents(ctx, hostID, 20)

	// Get failed logins
	failedLogins, _ := s.siem.GetFailedLoginsByHost(ctx, hostID)

	// Build alert list
	var recentAlerts []HostAlert
	var suspiciousIPs []string
	seenIPs := make(map[string]bool)
	failedCount := 0

	for _, fl := range failedLogins {
		if ip, ok := fl["source_ip"].(string); ok && !seenIPs[ip] {
			suspiciousIPs = append(suspiciousIPs, ip)
			seenIPs[ip] = true
		}
		if count, ok := fl["count"].(int64); ok {
			failedCount += int(count)
		}
	}

	for _, evt := range events {
		if len(recentAlerts) >= 5 {
			break
		}
		severity := "info"
		switch evt.EventType {
		case "failed_login", "brute_force":
			severity = "high"
		case "suspicious_process", "lateral_movement":
			severity = "critical"
		case "port_scan":
			severity = "medium"
		}

		recentAlerts = append(recentAlerts, HostAlert{
			ID:        fmt.Sprintf("evt-%d", evt.ID),
			Severity:  severity,
			EventType: evt.EventType,
			Timestamp: evt.Timestamp,
			SourceIP:  evt.SourceIP,
			Summary:   fmt.Sprintf("%s from %s (%s)", evt.EventType, evt.SourceIP, evt.User),
		})
	}

	// Generate threat summary
	riskLevel := "low"
	summary := "No significant threats detected"
	switch {
	case riskScore >= 80:
		riskLevel = "critical"
		summary = fmt.Sprintf("🔴 CRITICAL: %d alerts, %d failed logins from %d IPs", len(recentAlerts), failedCount, len(suspiciousIPs))
	case riskScore >= 50:
		riskLevel = "high"
		summary = fmt.Sprintf("🟠 HIGH RISK: %d recent events, investigate suspicious activity", len(events))
	case riskScore >= 25:
		riskLevel = "medium"
		summary = fmt.Sprintf("🟡 ELEVATED: %d events in monitoring window", len(events))
	default:
		if len(events) > 0 {
			summary = fmt.Sprintf("🟢 Normal: %d events, no anomalies", len(events))
		}
	}

	var lastEvent string
	if len(events) > 0 {
		lastEvent = events[0].Timestamp
	}

	return &OperatorContext{
		HostID:        hostID,
		HostLabel:     host.Label,
		RiskScore:     riskScore,
		RiskLevel:     riskLevel,
		AlertCount:    len(recentAlerts),
		RecentAlerts:  recentAlerts,
		FailedLogins:  failedCount,
		SuspiciousIPs: suspiciousIPs,
		LastEvent:     lastEvent,
		ThreatSummary: summary,
	}, nil
}

// GetBannerText returns a one-line status string for the terminal tab bar.
// This is the lightweight version of GetContext for frequent polling.
func (s *OperatorService) GetBannerText(hostID string) string {
	if hostID == "" || hostID == "local" {
		return ""
	}

	riskScore, _ := s.siem.CalculateRiskScore(context.Background(), hostID)
	events, _ := s.siem.GetHostEvents(context.Background(), hostID, 5)

	if riskScore >= 80 {
		return fmt.Sprintf("🔴 CRITICAL RISK (%d) — %d events", riskScore, len(events))
	}
	if riskScore >= 50 {
		return fmt.Sprintf("🟠 HIGH RISK (%d) — %d events", riskScore, len(events))
	}
	if riskScore >= 25 {
		return fmt.Sprintf("🟡 ELEVATED (%d)", riskScore)
	}
	if len(events) > 0 {
		return fmt.Sprintf("🟢 %d events", len(events))
	}
	return ""
}
