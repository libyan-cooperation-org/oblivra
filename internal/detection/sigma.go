package detection

// sigma.go — Sigma rule transpiler for Oblivra
//
// Converts the Sigma community detection format (https://sigmahq.io) into the
// native Oblivra Rule format so that the entire SigmaHQ ruleset can be imported
// without manual rewriting.
//
// Supported Sigma constructs:
//   - title, id, description, status, level
//   - tags  (maps MITRE ATT&CK tags → mitre_tactics / mitre_techniques)
//   - detection: keywords, selection with contains/endswith/startswith/|re: modifiers
//   - condition: selection / keywords (and / or / not)
//   - timeframe → window_sec
//   - logsource  → EventType condition
//   - falsepositives → appended to Description
//
// Not yet supported (skipped / best-effort):
//   - Near/within operators
//   - Aggregate functions (count by, sum by)
//   - Multi-document correlated rules (related:)

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── Sigma YAML structures ─────────────────────────────────────────────────────

// SigmaRule is the top-level Sigma document.
type SigmaRule struct {
	Title          string         `yaml:"title"`
	ID             string         `yaml:"id"`
	Description    string         `yaml:"description"`
	Status         string         `yaml:"status"`      // stable | test | experimental | deprecated
	Level          string         `yaml:"level"`       // informational | low | medium | high | critical
	Tags           []string       `yaml:"tags"`        // attack.tXXXX, attack.tactname
	LogSource      SigmaLogSource `yaml:"logsource"`
	Detection      SigmaDetection `yaml:"detection"`
	Timeframe      string         `yaml:"timeframe"`   // e.g. "15m", "1h", "30s"
	FalsePositives []string       `yaml:"falsepositives"`
	References     []string       `yaml:"references"`
}

// SigmaLogSource describes the log category the rule targets.
type SigmaLogSource struct {
	Category   string `yaml:"category"`
	Product    string `yaml:"product"`
	Service    string `yaml:"service"`
	Definition string `yaml:"definition"`
}

// SigmaDetection holds detection identifiers and the condition expression.
type SigmaDetection struct {
	// All keys other than "condition" and "timeframe" are detection identifiers.
	Identifiers map[string]interface{} `yaml:",inline"`
	Condition   string                 `yaml:"condition"`
}

// ── Public API ────────────────────────────────────────────────────────────────

// TranspileSigma converts raw Sigma YAML bytes into an Oblivra Rule.
// Returns an error for fundamentally malformed documents or deprecated rules.
func TranspileSigma(data []byte) (*Rule, error) {
	var sigma SigmaRule
	if err := yaml.Unmarshal(data, &sigma); err != nil {
		return nil, fmt.Errorf("parse sigma yaml: %w", err)
	}

	if sigma.Title == "" {
		return nil, fmt.Errorf("sigma rule missing title")
	}
	if sigma.Detection.Condition == "" {
		return nil, fmt.Errorf("sigma rule '%s' missing detection.condition", sigma.Title)
	}
	if sigma.Status == "deprecated" {
		return nil, fmt.Errorf("sigma rule '%s' is deprecated, skipping", sigma.Title)
	}

	rule := &Rule{
		ID:          sigmaRuleID(sigma),
		Name:        sigma.Title,
		Description: buildSigmaDescription(sigma),
		Severity:    mapSigmaSeverity(sigma.Level),
		Type:        ThresholdRule,
		Threshold:   1,
	}

	rule.MitreTactics, rule.MitreTechniques = parseSigmaMitreTags(sigma.Tags)

	rule.WindowSec = parseSigmaTimeframe(sigma.Timeframe)
	rule.DedupWindowSec = rule.WindowSec * 2
	if rule.DedupWindowSec == 0 {
		rule.DedupWindowSec = 300 // 5-minute default dedup
	}

	conditions, err := buildSigmaConditions(sigma)
	if err != nil {
		return nil, fmt.Errorf("build conditions for '%s': %w", sigma.Title, err)
	}
	rule.Conditions = conditions
	rule.GroupBy = inferSigmaGroupBy(sigma)

	return rule, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func sigmaRuleID(s SigmaRule) string {
	if s.ID != "" {
		return "sigma-" + s.ID
	}
	slug := strings.ToLower(strings.ReplaceAll(s.Title, " ", "-"))
	slug = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(slug, "")
	return "sigma-" + slug
}

func buildSigmaDescription(s SigmaRule) string {
	desc := s.Description
	if len(s.FalsePositives) > 0 {
		desc += "\nFalse positives: " + strings.Join(s.FalsePositives, "; ")
	}
	if len(s.References) > 0 {
		desc += "\nReferences: " + strings.Join(s.References, ", ")
	}
	return strings.TrimSpace(desc)
}

// mapSigmaSeverity converts Sigma level strings to Oblivra severity strings.
func mapSigmaSeverity(level string) string {
	switch strings.ToLower(level) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low", "informational":
		return "low"
	default:
		return "medium"
	}
}

// parseSigmaMitreTags extracts MITRE tactic and technique identifiers.
// Sigma encodes these as "attack.tXXXX" (technique) or "attack.tactname" (tactic).
func parseSigmaMitreTags(tags []string) (tactics []string, techniques []string) {
	tacticMap := map[string]string{
		"initial-access":       "TA0001",
		"execution":            "TA0002",
		"persistence":          "TA0003",
		"privilege-escalation": "TA0004",
		"defense-evasion":      "TA0005",
		"credential-access":    "TA0006",
		"discovery":            "TA0007",
		"lateral-movement":     "TA0008",
		"collection":           "TA0009",
		"exfiltration":         "TA0010",
		"command-and-control":  "TA0011",
		"impact":               "TA0040",
		"resource-development": "TA0042",
		"reconnaissance":       "TA0043",
	}
	techRE := regexp.MustCompile(`(?i)^attack\.(t\d{4}(?:\.\d{3})?)$`)

	for _, tag := range tags {
		lower := strings.ToLower(tag)
		if m := techRE.FindStringSubmatch(lower); m != nil {
			techniques = append(techniques, strings.ToUpper(m[1]))
			continue
		}
		if strings.HasPrefix(lower, "attack.") {
			slug := strings.ReplaceAll(strings.TrimPrefix(lower, "attack."), "_", "-")
			if ta, ok := tacticMap[slug]; ok {
				tactics = append(tactics, ta)
			}
		}
	}
	return
}

// parseSigmaTimeframe converts Sigma timeframe strings (e.g. "15m", "1h") to seconds.
func parseSigmaTimeframe(tf string) int {
	if tf == "" {
		return 0
	}
	tf = strings.TrimSpace(strings.ToLower(tf))
	if len(tf) < 2 {
		return 0
	}
	unit := tf[len(tf)-1]
	var n int
	fmt.Sscanf(tf[:len(tf)-1], "%d", &n)
	switch unit {
	case 's':
		return n
	case 'm':
		return n * 60
	case 'h':
		return n * 3600
	case 'd':
		return n * 86400
	}
	return 300
}

// buildSigmaConditions converts a Sigma detection block into an Oblivra condition map.
func buildSigmaConditions(sigma SigmaRule) (map[string]string, error) {
	conditions := make(map[string]string)

	// Add logsource as an EventType hint
	if et := sigmaLogsourceToEventType(sigma.LogSource); et != "" {
		conditions["EventType"] = et
	}

	cond := strings.ToLower(sigma.Detection.Condition)

	for key, value := range sigma.Detection.Identifiers {
		lkey := strings.ToLower(key)
		if lkey == "condition" || lkey == "timeframe" {
			continue
		}
		// Only process identifiers actually referenced in the condition expression
		if !strings.Contains(cond, lkey) {
			continue
		}
		if err := mergeSigmaSelectionConditions(conditions, value); err != nil {
			return nil, fmt.Errorf("identifier '%s': %w", key, err)
		}
	}

	return conditions, nil
}

// mergeSigmaSelectionConditions flattens a Sigma detection identifier into Oblivra conditions.
func mergeSigmaSelectionConditions(out map[string]string, raw interface{}) error {
	switch v := raw.(type) {
	case []interface{}:
		// Keyword list — match against RawLog
		var keywords []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				keywords = append(keywords, regexp.QuoteMeta(s))
			}
		}
		if len(keywords) > 0 {
			appendSigmaPattern(out, "output_contains", "regex:(?i)"+strings.Join(keywords, "|"))
		}

	case map[string]interface{}:
		for field, val := range v {
			// Parse field modifiers: field|contains, field|startswith, etc.
			parts := strings.SplitN(field, "|", 2)
			fieldName := parts[0]
			modifier := ""
			if len(parts) == 2 {
				modifier = strings.ToLower(parts[1])
			}
			oblivraKey := sigmaFieldToOblivra(fieldName)
			pattern := buildSigmaPattern(val, modifier)
			if pattern != "" {
				appendSigmaPattern(out, oblivraKey, pattern)
			}
		}

	case string:
		appendSigmaPattern(out, "output_contains", "regex:(?i)"+regexp.QuoteMeta(v))
	}
	return nil
}

// appendSigmaPattern merges a new pattern into an existing Oblivra condition, combining with OR.
func appendSigmaPattern(out map[string]string, key, pattern string) {
	existing, ok := out[key]
	if !ok {
		out[key] = pattern
		return
	}
	// Both should start with "regex:" — combine as OR alternatives
	existingBody := strings.TrimPrefix(existing, "regex:(?i)")
	newBody := strings.TrimPrefix(pattern, "regex:(?i)")
	out[key] = "regex:(?i)" + existingBody + "|" + newBody
}

// buildSigmaPattern converts a Sigma field value with optional modifier into an Oblivra pattern.
func buildSigmaPattern(val interface{}, modifier string) string {
	var values []string
	switch v := val.(type) {
	case string:
		values = []string{v}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				values = append(values, s)
			}
		}
	default:
		return ""
	}
	if len(values) == 0 {
		return ""
	}

	var patterns []string
	for _, s := range values {
		switch modifier {
		case "contains", "":
			patterns = append(patterns, regexp.QuoteMeta(s))
		case "startswith":
			patterns = append(patterns, "^"+regexp.QuoteMeta(s))
		case "endswith":
			patterns = append(patterns, regexp.QuoteMeta(s)+"$")
		case "re":
			patterns = append(patterns, s) // already a regex
		case "all":
			// ALL modifier: every value must appear. Go's RE2 has no lookaheads,
			// so we use .* between terms as a best-effort approximation.
			var quoted []string
			for _, sv := range values {
				quoted = append(quoted, regexp.QuoteMeta(sv))
			}
			return "regex:(?i)" + strings.Join(quoted, ".*")
		default:
			patterns = append(patterns, regexp.QuoteMeta(s))
		}
	}
	return "regex:(?i)" + strings.Join(patterns, "|")
}

// sigmaFieldToOblivra maps Sigma field names to Oblivra Event field names.
func sigmaFieldToOblivra(field string) string {
	switch strings.ToLower(field) {
	case "user", "username", "account", "subjectusername", "targetusername":
		return "user"
	case "sourceip", "src_ip", "ipaddress", "c-ip", "remoteaddress",
		"sourceaddress", "ipv4", "clientip":
		return "source_ip"
	case "hostname", "computername", "computer", "host", "desthost":
		return "host"
	case "eventid", "event_id", "eventcode":
		return "EventType"
	default:
		// Everything else (CommandLine, FileName, Image, etc.) → search RawLog
		return "output_contains"
	}
}

// sigmaLogsourceToEventType converts a Sigma logsource block to an Oblivra EventType hint.
func sigmaLogsourceToEventType(ls SigmaLogSource) string {
	switch strings.ToLower(ls.Category) {
	case "process_creation":
		return "process_creation"
	case "network_connection", "network_traffic":
		return "network_connection"
	case "dns_query":
		return "dns_query"
	case "file_event", "file_change", "file_delete":
		return "file_event"
	case "registry_event", "registry_add", "registry_set", "registry_delete":
		return "registry_event"
	case "authentication", "logon":
		return "authentication"
	}

	product := strings.ToLower(ls.Product)
	service := strings.ToLower(ls.Service)

	switch {
	case product == "windows" && service == "security":
		return "windows_security"
	case product == "windows" && service == "system":
		return "windows_system"
	case product == "windows" && (service == "powershell" || service == "powershell-classic"):
		return "powershell"
	case product == "linux":
		return "syslog"
	case product == "aws":
		return "cloudtrail"
	case product == "azure":
		return "azure_log"
	case product == "gcp":
		return "gcp_log"
	case service == "sshd":
		return "sshd"
	case service == "sudo":
		return "sudo"
	}
	return ""
}

// inferSigmaGroupBy derives grouping keys from the logsource / condition context.
func inferSigmaGroupBy(sigma SigmaRule) []string {
	cat := strings.ToLower(sigma.LogSource.Category)
	svc := strings.ToLower(sigma.LogSource.Service)
	cond := strings.ToLower(sigma.Detection.Condition)

	if strings.Contains(cat, "network") || strings.Contains(svc, "ssh") ||
		strings.Contains(cond, "count") {
		return []string{"source_ip"}
	}
	if strings.Contains(cat, "auth") || strings.Contains(cat, "logon") {
		return []string{"user", "source_ip"}
	}
	return nil
}
