// Package parsers turns raw log lines into canonical Event structs. Each
// parser is best-effort: a parse failure surfaces as an Event whose Raw is
// preserved and whose EventType is "parse.unknown" so nothing is dropped.
package parsers

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

type Format string

const (
	FormatAuto    Format = "auto"
	FormatJSON    Format = "json"
	FormatRFC5424 Format = "rfc5424"
	FormatRFC3164 Format = "rfc3164"
	FormatCEF     Format = "cef"
	FormatAuditd  Format = "auditd"
)

// Parse picks a parser by format (or auto-detects) and returns a populated
// event. The Raw field always carries the original line.
func Parse(raw string, format Format) (*events.Event, error) {
	raw = strings.TrimRight(raw, "\r\n")
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("parsers: empty line")
	}
	if format == "" || format == FormatAuto {
		format = sniff(raw)
	}
	switch format {
	case FormatJSON:
		return parseJSON(raw)
	case FormatRFC5424:
		return parseRFC5424(raw)
	case FormatRFC3164:
		return parseRFC3164(raw)
	case FormatCEF:
		return parseCEF(raw)
	case FormatAuditd:
		return parseAuditd(raw)
	default:
		return parsePlain(raw), nil
	}
}

// sniff peeks at the first non-space rune to decide a format.
func sniff(line string) Format {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return FormatAuto
	}
	switch trimmed[0] {
	case '{':
		return FormatJSON
	case '<':
		// <PRI>VERSION ... → RFC5424; otherwise RFC3164.
		if rfc5424VersionRE.MatchString(trimmed) {
			return FormatRFC5424
		}
		return FormatRFC3164
	}
	if strings.HasPrefix(trimmed, "CEF:") {
		return FormatCEF
	}
	// auditd lines always start with `type=` and contain `msg=audit(...)`.
	if strings.HasPrefix(trimmed, "type=") && strings.Contains(trimmed, "msg=audit(") {
		return FormatAuditd
	}
	return FormatAuto
}

// ---- JSON ----

func parseJSON(raw string) (*events.Event, error) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return parsePlain(raw), nil
	}
	ev := &events.Event{Source: events.SourceFile, Raw: raw}

	if v, ok := doc["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			ev.Timestamp = t
		}
	}
	if v, ok := doc["host"].(string); ok {
		ev.HostID = v
	} else if v, ok := doc["hostId"].(string); ok {
		ev.HostID = v
	}
	if v, ok := doc["severity"].(string); ok {
		ev.Severity = events.Severity(v)
	}
	if v, ok := doc["message"].(string); ok {
		ev.Message = v
	} else if v, ok := doc["msg"].(string); ok {
		ev.Message = v
	}
	if v, ok := doc["eventType"].(string); ok {
		ev.EventType = v
	}

	// Promote remaining scalar fields to Fields map (string-typed).
	fields := map[string]string{}
	for k, v := range doc {
		switch k {
		case "timestamp", "host", "hostId", "severity", "message", "msg", "eventType":
			continue
		}
		switch t := v.(type) {
		case string:
			fields[k] = t
		case float64:
			fields[k] = strconv.FormatFloat(t, 'f', -1, 64)
		case bool:
			fields[k] = strconv.FormatBool(t)
		}
	}
	if len(fields) > 0 {
		ev.Fields = fields
	}
	if ev.Message == "" {
		ev.Message = raw
	}
	if ev.EventType == "" {
		ev.EventType = "json"
	}
	return ev, nil
}

// ---- RFC 5424 ----
//
// <PRI>VERSION TIMESTAMP HOSTNAME APP-NAME PROCID MSGID [STRUCTURED-DATA] MSG
//
// e.g. <34>1 2026-04-01T12:34:56Z host app 1234 ID47 - hello world

var (
	rfc5424RE        = regexp.MustCompile(`^<(\d+)>(\d+) (\S+) (\S+) (\S+) (\S+) (\S+) (.*)$`)
	rfc5424VersionRE = regexp.MustCompile(`^<\d+>\d+ `)
)

func parseRFC5424(raw string) (*events.Event, error) {
	m := rfc5424RE.FindStringSubmatch(raw)
	if m == nil {
		return parsePlain(raw), nil
	}
	pri, _ := strconv.Atoi(m[1])
	ts, _ := time.Parse(time.RFC3339Nano, m[3])
	if ts.IsZero() {
		ts, _ = time.Parse(time.RFC3339, m[3])
	}
	ev := &events.Event{
		Source:    events.SourceSyslog,
		Severity:  syslogSeverity(pri),
		Timestamp: ts,
		HostID:    nilDash(m[4]),
		EventType: "syslog.5424",
		Message:   m[8],
		Raw:       raw,
		Fields: map[string]string{
			"app":    nilDash(m[5]),
			"pid":    nilDash(m[6]),
			"msgId":  nilDash(m[7]),
			"sd":     "",
			"facility": strconv.Itoa(pri >> 3),
		},
	}
	return ev, nil
}

// ---- RFC 3164 ----
//
// <PRI>MMM dd HH:MM:SS HOSTNAME TAG: MSG
//
// e.g. <34>Apr  1 12:34:56 host sshd[1234]: Failed password for root

var rfc3164RE = regexp.MustCompile(`^<(\d+)>([A-Z][a-z]{2}\s+\d+\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+([^:]+):\s*(.*)$`)

func parseRFC3164(raw string) (*events.Event, error) {
	m := rfc3164RE.FindStringSubmatch(raw)
	if m == nil {
		return parsePlain(raw), nil
	}
	pri, _ := strconv.Atoi(m[1])
	// 3164 timestamps lack year; assume current.
	ts, err := time.Parse("Jan _2 15:04:05", m[2])
	if err == nil {
		now := time.Now().UTC()
		ts = time.Date(now.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), ts.Second(), 0, time.UTC)
	} else {
		ts = time.Now().UTC()
	}
	tag := strings.TrimSpace(m[4])
	app, pid := splitTag(tag)
	return &events.Event{
		Source:    events.SourceSyslog,
		Severity:  syslogSeverity(pri),
		Timestamp: ts,
		HostID:    m[3],
		EventType: "syslog.3164",
		Message:   m[5],
		Raw:       raw,
		Fields: map[string]string{
			"app":      app,
			"pid":      pid,
			"facility": strconv.Itoa(pri >> 3),
		},
	}, nil
}

// ---- CEF ----
//
// CEF:0|Vendor|Product|Version|SignatureID|Name|Severity|Extension
//
// e.g. CEF:0|Trend|DeepSecurity|10.0|600|Login Failure|6|src=10.0.0.1 act=blocked

func parseCEF(raw string) (*events.Event, error) {
	if !strings.HasPrefix(raw, "CEF:") {
		return parsePlain(raw), nil
	}
	parts := splitN(raw, '|', 8)
	if len(parts) < 8 {
		return parsePlain(raw), nil
	}
	sevAtoi, _ := strconv.Atoi(parts[6])
	ev := &events.Event{
		Source:    events.SourceFile,
		EventType: "cef",
		Severity:  cefSeverity(sevAtoi),
		Message:   parts[5],
		Raw:       raw,
		Fields: map[string]string{
			"cefVersion":  strings.TrimPrefix(parts[0], "CEF:"),
			"vendor":      parts[1],
			"product":     parts[2],
			"productVer":  parts[3],
			"signatureId": parts[4],
		},
	}
	for k, v := range parseCEFExt(parts[7]) {
		ev.Fields[k] = v
	}
	if v, ok := ev.Fields["src"]; ok {
		ev.HostID = v
	}
	return ev, nil
}

func parseCEFExt(s string) map[string]string {
	out := map[string]string{}
	// Split on space-separated key=value, but allow values to contain spaces if
	// followed by another key=. Cheap state machine.
	tokens := strings.Fields(s)
	for i := 0; i < len(tokens); i++ {
		eq := strings.IndexByte(tokens[i], '=')
		if eq <= 0 {
			continue
		}
		key := tokens[i][:eq]
		val := tokens[i][eq+1:]
		// Stitch back tokens until we see another key= pattern.
		for j := i + 1; j < len(tokens); j++ {
			if strings.Contains(tokens[j], "=") && !strings.HasPrefix(tokens[j], "=") {
				break
			}
			val += " " + tokens[j]
			i = j
		}
		out[key] = val
	}
	return out
}

// ---- Plain fallback ----

func parsePlain(raw string) *events.Event {
	return &events.Event{
		Source:    events.SourceFile,
		Severity:  events.SeverityInfo,
		EventType: "plain",
		Message:   raw,
		Raw:       raw,
	}
}

// ---- helpers ----

func syslogSeverity(pri int) events.Severity {
	sev := pri & 0x07
	switch sev {
	case 0:
		return events.SeverityAlert // emerg → alert tier
	case 1:
		return events.SeverityAlert
	case 2:
		return events.SeverityCritical
	case 3:
		return events.SeverityError
	case 4:
		return events.SeverityWarn
	case 5:
		return events.SeverityNotice
	case 6:
		return events.SeverityInfo
	default:
		return events.SeverityDebug
	}
}

func cefSeverity(n int) events.Severity {
	switch {
	case n >= 9:
		return events.SeverityCritical
	case n >= 7:
		return events.SeverityError
	case n >= 4:
		return events.SeverityWarn
	case n >= 1:
		return events.SeverityInfo
	default:
		return events.SeverityDebug
	}
}

func nilDash(s string) string {
	if s == "-" {
		return ""
	}
	return s
}

func splitTag(tag string) (app, pid string) {
	open := strings.IndexByte(tag, '[')
	if open < 0 {
		return tag, ""
	}
	close := strings.IndexByte(tag, ']')
	if close <= open {
		return tag, ""
	}
	return tag[:open], tag[open+1 : close]
}

// splitN splits s by separator into at most n parts; CEF uses '|' inside fields
// which would break strings.Split — but the spec forbids '|' in fields without
// escaping, so a simple split is fine for now.
func splitN(s string, sep byte, n int) []string {
	parts := make([]string, 0, n)
	start := 0
	for i := 0; i < len(s) && len(parts) < n-1; i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// Format selectors for unknown raw bytes; mostly here so callers can handle
// untyped errors uniformly.
func ParseFormat(raw string, format Format) (*events.Event, error) {
	ev, err := Parse(raw, format)
	if err != nil {
		return nil, fmt.Errorf("parsers(%s): %w", format, err)
	}
	return ev, nil
}
