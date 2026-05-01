package services

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// TamperService surfaces log-level tampering signals: auditd self-disable,
// log-rotation gaps that line up with attacker activity, journal-truncation
// keywords, and clock-rollback patterns. Emits alerts for high-confidence
// signals; everything else is reported via /api/v1/tamper/findings.
type TamperFinding struct {
	HostID    string    `json:"hostId"`
	Kind      string    `json:"kind"` // "auditd-disabled" | "logrotate-gap" | "journal-truncate" | "clock-rollback"
	Detail    string    `json:"detail"`
	EventID   string    `json:"eventId,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type TamperService struct {
	log    *slog.Logger
	alerts *AlertService

	mu       sync.RWMutex
	findings []TamperFinding
	cap      int

	// Per-host clock watermark — used to detect clock-rollback at host level.
	hostClock map[string]time.Time
}

func NewTamperService(log *slog.Logger, alerts *AlertService) *TamperService {
	return &TamperService{
		log: log, alerts: alerts, cap: 5000,
		hostClock: map[string]time.Time{},
	}
}

func (s *TamperService) ServiceName() string { return "TamperService" }

var (
	rxAuditdStop  = regexp.MustCompile(`(?i)\b(auditd|auditctl)\b.*\b(stopped|disabled|halted|exiting)\b`)
	rxAuditdDel   = regexp.MustCompile(`(?i)auditctl\s+-D\b|auditctl\s+--delete`)
	rxJournalCut  = regexp.MustCompile(`(?i)\b(systemd-journald|journalctl)\b.*\b(rotated|deleted|vacuum-time|vacuum-size)\b`)
	rxClearLog    = regexp.MustCompile(`(?i)wevtutil\s+cl\b|Clear-EventLog|fsutil\s+usn\s+deletejournal`)
	rxBigGap      = regexp.MustCompile(`(?i)logrotate.*completed`)

	// Self-disable substrings — checked case-insensitively. An attacker
	// silencing OBLIVRA itself before pivoting is a strictly red signal,
	// because legitimate operator actions go through the audit chain.
	selfDisableNeedles = []string{
		"systemctl stop oblivra",
		"systemctl disable oblivra",
		"systemctl mask oblivra",
		"pkill -9 oblivra",
		"killall -9 oblivra",
		"sc stop oblivraagent", "sc.exe stop oblivraagent",
		"sc stop oblivraserver", "sc.exe stop oblivraserver",
		"sc delete oblivraagent", "sc.exe delete oblivraagent",
		"sc delete oblivraserver", "sc.exe delete oblivraserver",
		"taskkill /im oblivra-agent.exe",
		"taskkill /im oblivra-server.exe",
		"stop-service oblivraagent",
		"stop-service oblivraserver",
		"rm -rf /var/lib/oblivra",
	}
)

// Observe inspects each event for tampering markers.
func (s *TamperService) Observe(ctx context.Context, ev events.Event) {
	if ev.HostID == "" {
		return
	}
	src := ev.Message + " " + ev.Raw
	now := time.Now().UTC()

	add := func(kind, detail string, severity AlertSeverity) {
		f := TamperFinding{
			HostID: ev.HostID, Kind: kind, Detail: detail,
			EventID: ev.ID, Timestamp: ev.Timestamp,
		}
		s.mu.Lock()
		s.findings = append(s.findings, f)
		if len(s.findings) > s.cap {
			s.findings = s.findings[len(s.findings)-s.cap:]
		}
		s.mu.Unlock()
		if s.alerts != nil {
			s.alerts.Raise(ctx, Alert{
				TenantID: ev.TenantID, RuleID: "tamper-" + kind,
				RuleName: "Log tampering signal: " + kind,
				Severity: severity, HostID: ev.HostID,
				Message:  detail, MITRE: []string{"T1562.001"},
				EventIDs: []string{ev.ID},
			})
		}
		_ = now // kept for future "first-seen" tracking
	}

	switch {
	case rxAuditdStop.MatchString(src) || rxAuditdDel.MatchString(src):
		add("auditd-disabled", "auditd appears to have been stopped or its rules cleared", AlertSeverityHigh)
	case rxJournalCut.MatchString(src):
		add("journal-truncate", "systemd-journald rotation/vacuum executed", AlertSeverityMedium)
	case rxClearLog.MatchString(src):
		add("eventlog-clear", "Windows event log clear / USN journal delete observed", AlertSeverityHigh)
	case containsAnyFold(src, selfDisableNeedles):
		add("self-disable-attempt", "attempt to stop, mask, or remove the OBLIVRA agent or server itself", AlertSeverityCritical)
	}

	// Clock rollback: a fresh event whose timestamp is more than 5 minutes
	// behind the host's previous high watermark.
	s.mu.Lock()
	prev, seen := s.hostClock[ev.HostID]
	if seen && ev.Timestamp.Before(prev.Add(-5*time.Minute)) {
		f := TamperFinding{
			HostID: ev.HostID, Kind: "clock-rollback",
			Detail: "host clock rewound by " + prev.Sub(ev.Timestamp).Round(time.Second).String(),
			EventID: ev.ID, Timestamp: ev.Timestamp,
		}
		s.findings = append(s.findings, f)
		s.mu.Unlock()
		if s.alerts != nil {
			s.alerts.Raise(ctx, Alert{
				TenantID: ev.TenantID, RuleID: "tamper-clock-rollback",
				RuleName: "Log tampering signal: clock rollback",
				Severity: AlertSeverityHigh, HostID: ev.HostID,
				Message: f.Detail, MITRE: []string{"T1070.006"},
				EventIDs: []string{ev.ID},
			})
		}
	} else {
		if ev.Timestamp.After(prev) {
			s.hostClock[ev.HostID] = ev.Timestamp
		}
		s.mu.Unlock()
	}
	_ = strings.HasPrefix // keep strings imported even if rxes change later
}

// containsAnyFold returns true if any of `needles` appears in `haystack`,
// case-insensitively. Cheap fallback for substring lists where a regex
// would be overkill.
func containsAnyFold(haystack string, needles []string) bool {
	low := strings.ToLower(haystack)
	for _, n := range needles {
		if strings.Contains(low, n) {
			return true
		}
	}
	return false
}

func (s *TamperService) Findings(host string, limit int) []TamperFinding {
	if limit <= 0 {
		limit = 100
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TamperFinding, 0, limit)
	for i := len(s.findings) - 1; i >= 0 && len(out) < limit; i-- {
		f := s.findings[i]
		if host != "" && f.HostID != host {
			continue
		}
		out = append(out, f)
	}
	return out
}
