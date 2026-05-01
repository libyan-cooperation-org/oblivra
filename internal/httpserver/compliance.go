package httpserver

import (
	"net/http"
	"strings"
	"time"
)

// Phase 46 — Compliance attestation adapters.
//
// We deliberately don't ship PDF/HTML compliance report packs (Phase 36
// scope cut). Instead, the platform exposes a machine-readable JSON-LD
// evidence feed per framework that external compliance tools (Drata,
// Vanta, Tugboat Logic) can consume on a schedule. They map control IDs
// to ATT&CK / log-evidence; we provide the audit-grade source material
// keyed to each control.
//
// Endpoint: GET /api/v1/compliance/feed/{framework}
//
// Frameworks supported (manifest at docs/compliance/<framework>.md):
//   pci-dss-4   — Requirements 10, 11, 12
//   soc2        — CC6, CC7, CC8, CC9
//   nist-800-53 — AU, AC, CA, CM, IA, IR, SI control families
//   iso-27001   — A.8, A.9, A.12, A.16
//   gdpr        — Articles 25, 30, 32 (logging + breach detection)
//   hipaa       — 164.312 (audit controls + integrity)

// complianceFeedHandler renders a JSON-LD document keyed to control IDs.
// Each control entry includes:
//   - controlId        : the framework's identifier (e.g. "pci-dss-4:10.2.1")
//   - title            : short human-readable name
//   - evidenceType     : what the entry represents ("audit-log", "alert", etc.)
//   - lastSeenAt       : timestamp of the most recent matching evidence
//   - count24h         : count over the last 24h (for freshness)
//   - sourceEndpoint   : which OBLIVRA endpoint provides the granular data
//
// Compliance tools poll this on their own cadence; "stale" controls
// (lastSeenAt > N days ago) trigger their existing remediation flow.
func (s *Server) complianceFeedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		framework := strings.TrimPrefix(r.URL.Path, "/api/v1/compliance/feed/")
		framework = strings.Trim(framework, "/")
		controls, ok := complianceFrameworks[framework]
		if !ok {
			writeError(w, http.StatusNotFound, "unknown framework: "+framework)
			return
		}
		now := time.Now().UTC()

		entries := make([]map[string]any, 0, len(controls))
		for _, c := range controls {
			lastSeen, count24h := s.evidenceFreshness(c.evidenceType, c.matcher, now)
			entry := map[string]any{
				"@type":          "ComplianceEvidence",
				"controlId":      c.id,
				"title":          c.title,
				"evidenceType":   c.evidenceType,
				"sourceEndpoint": c.endpoint,
				"count24h":       count24h,
			}
			if !lastSeen.IsZero() {
				entry["lastSeenAt"] = lastSeen.Format(time.RFC3339)
				entry["fresh"] = now.Sub(lastSeen) < 7*24*time.Hour
			} else {
				entry["fresh"] = false
			}
			entries = append(entries, entry)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"@context":    "https://oblivra.dev/compliance/v1",
			"@type":       "ComplianceFeed",
			"framework":   framework,
			"generatedAt": now.Format(time.RFC3339),
			"controls":    entries,
		})
	}
}

// evidenceFreshness computes (lastSeenAt, count24h) for a given evidence
// matcher. Cheap implementation — counts entries in the audit chain
// matching `action`. Frameworks that need richer matching should provide
// a custom matcher func (see complianceControl.matcher).
func (s *Server) evidenceFreshness(evidenceType string, matcher func(action string) bool, now time.Time) (time.Time, int) {
	if s.audit == nil {
		return time.Time{}, 0
	}
	since := now.Add(-24 * time.Hour)
	all := s.audit.Recent(2000)
	var lastSeen time.Time
	count := 0
	for _, e := range all {
		if matcher != nil && !matcher(e.Action) {
			continue
		}
		if e.Timestamp.After(lastSeen) {
			lastSeen = e.Timestamp
		}
		if e.Timestamp.After(since) {
			count++
		}
	}
	_ = evidenceType
	return lastSeen, count
}

type complianceControl struct {
	id           string
	title        string
	evidenceType string
	endpoint     string
	matcher      func(action string) bool
}

func actionPrefix(prefix string) func(string) bool {
	return func(a string) bool { return strings.HasPrefix(a, prefix) }
}

// complianceFrameworks is the lookup table. Each entry maps a control to
// the audit-action prefix that proves it. Frameworks expand over time;
// the map is kept here so adding one is a single PR.
var complianceFrameworks = map[string][]complianceControl{
	"pci-dss-4": {
		{"pci-dss-4:10.2.1", "Audit logs for all individual user accesses", "audit-log",
			"/api/v1/audit/log", actionPrefix("auth.")},
		{"pci-dss-4:10.2.2", "Audit logs for all admin actions", "audit-log",
			"/api/v1/audit/log", actionPrefix("admin.")},
		{"pci-dss-4:10.5.5", "Audit log integrity (file integrity monitoring)", "audit-verify",
			"/api/v1/audit/verify", actionPrefix("audit.daily-anchor")},
		{"pci-dss-4:11.5.1", "Detection of unauthorized network changes", "detection-alert",
			"/api/v1/alerts", actionPrefix("alert.")},
	},
	"soc2": {
		{"CC6.1", "Logical access controls — auth events", "audit-log",
			"/api/v1/audit/log", actionPrefix("auth.")},
		{"CC6.6", "Detection of anomalous access patterns", "detection-alert",
			"/api/v1/alerts", actionPrefix("alert.")},
		{"CC7.2", "Monitoring of system components for anomalies", "detection-alert",
			"/api/v1/alerts", actionPrefix("alert.")},
		{"CC7.3", "Evaluation of security events to determine impact", "case-investigation",
			"/api/v1/cases", actionPrefix("investigation.")},
		{"CC8.1", "Change management — administrative actions", "audit-log",
			"/api/v1/audit/log", actionPrefix("admin.")},
	},
	"nist-800-53": {
		{"AU-2", "Event logging — actions that should be auditable", "audit-log",
			"/api/v1/audit/log", nil}, // every entry
		{"AU-9", "Protection of audit information (Merkle chain)", "audit-verify",
			"/api/v1/audit/verify", actionPrefix("audit.daily-anchor")},
		{"AC-2", "Account management — auth events", "audit-log",
			"/api/v1/audit/log", actionPrefix("auth.")},
		{"IR-4", "Incident handling — investigation cases", "case-investigation",
			"/api/v1/cases", actionPrefix("investigation.")},
		{"SI-4", "Information system monitoring — alerts", "detection-alert",
			"/api/v1/alerts", actionPrefix("alert.")},
	},
	"iso-27001": {
		{"A.8.15", "Logging — event recording", "audit-log",
			"/api/v1/audit/log", nil},
		{"A.8.16", "Monitoring — anomaly detection", "detection-alert",
			"/api/v1/alerts", actionPrefix("alert.")},
		{"A.5.27", "Learning from information security incidents", "case-investigation",
			"/api/v1/cases", actionPrefix("investigation.")},
	},
	"gdpr": {
		{"art-25", "Data protection by design — query log integrity", "audit-verify",
			"/api/v1/audit/verify", actionPrefix("audit.daily-anchor")},
		{"art-30", "Records of processing activities — auth events", "audit-log",
			"/api/v1/audit/log", actionPrefix("auth.")},
		{"art-32", "Security of processing — anomaly detection", "detection-alert",
			"/api/v1/alerts", actionPrefix("alert.")},
		{"art-33", "Notification of personal data breach (case timeline)", "case-investigation",
			"/api/v1/cases", actionPrefix("investigation.")},
	},
	"hipaa": {
		{"164.312(b)", "Audit controls — record activity in information systems", "audit-log",
			"/api/v1/audit/log", nil},
		{"164.312(c)(1)", "Integrity — protect ePHI from improper alteration", "audit-verify",
			"/api/v1/audit/verify", actionPrefix("audit.daily-anchor")},
		{"164.312(d)", "Person or entity authentication", "audit-log",
			"/api/v1/audit/log", actionPrefix("auth.")},
	},
}
