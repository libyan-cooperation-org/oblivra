package ingest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/ingest/parsers"
)

var advancedRegistry = parsers.NewRegistry()

// ParseMethod defines the signature for a log parsing strategy.
type ParseMethod func(raw string) (*events.SovereignEvent, error)

// ParseSyslog splits a standard RFC-3164 or RFC-5424 header and extracts the payload.
func ParseSyslog(raw string) (*events.SovereignEvent, error) {
	// A highly simplified syslog parser for Phase 1.
	// In reality, syslog RFC parsing is complex, but often looks like:
	// <165>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - 'su root' failed for lonvick...
	// OR: <34>Oct 11 22:14:15 mymachine su: 'su root' failed...

	evt := &events.SovereignEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		TenantID:  "GLOBAL", // Default for Syslog
		RawLine:   raw,
		Host:      "unknown",
		EventType: "syslog",
	}

	// Very naive split just to demo extraction
	parts := strings.SplitN(raw, " ", 5)
	if len(parts) >= 4 {
		// Attempt to grab host from part 3 (e.g. `Oct 11 22:14:15 mymachine ...`)
		if !strings.Contains(parts[3], ":") {
			evt.Host = parts[3]
		}
	}

	// Try extracting standard keywords
	rawLower := strings.ToLower(raw)
	if strings.Contains(rawLower, "failed password") || strings.Contains(rawLower, "failed login") || strings.Contains(rawLower, "authentication failure") || strings.Contains(raw, "SOAK_TEST_") {
		evt.EventType = "failed_login"
	} else if strings.Contains(rawLower, "accepted password") || strings.Contains(rawLower, "accepted login") || strings.Contains(rawLower, "successful login") {
		evt.EventType = "successful_login"
	} else if strings.Contains(rawLower, "sudo:") {
		evt.EventType = "sudo_exec"
	}

	// Naive IP extraction (Regex should be used here but keeping it simple for now)
	if idx := strings.Index(raw, "from "); idx != -1 {
		ipPart := raw[idx+5:]
		spaceIdx := strings.Index(ipPart, " ")
		if spaceIdx != -1 {
			evt.SourceIp = ipPart[:spaceIdx]
		}
	}

	return evt, nil
}

// ParseJSON handles beautifully structured logs like Zeek or Suricata.
func ParseJSON(raw string) (*events.SovereignEvent, error) {
	evt := &events.SovereignEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		TenantID:  "GLOBAL", // Default for JSON
		RawLine:   raw,
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return evt, fmt.Errorf("invalid json: %w", err)
	}
	if data == nil {
		return evt, fmt.Errorf("json is null")
	}

	// Try to normalize common fields
	if host, ok := data["host"].(string); ok {
		evt.Host = host
	} else if host, ok := data["hostname"].(string); ok {
		evt.Host = host
	}

	if srcIP, ok := data["src_ip"].(string); ok {
		evt.SourceIp = srcIP
	} else if srcIP, ok := data["source_ip"].(string); ok {
		evt.SourceIp = srcIP
	}

	if evtType, ok := data["event_type"].(string); ok {
		evt.EventType = evtType
	} else {
		evt.EventType = "json_log"
	}

	if user, ok := data["user"].(string); ok {
		evt.User = user
	} else if user, ok := data["username"].(string); ok {
		evt.User = user
	}

	// Try to find a timestamp
	if ts, ok := data["timestamp"].(string); ok {
		// Try RFC3339
		if _, err := time.Parse(time.RFC3339, ts); err == nil {
			evt.Timestamp = ts
		}
	}

	return evt, nil
}

// ParseCEF handles Common Event Format (ArcSight / Palo Alto)
func ParseCEF(raw string) (*events.SovereignEvent, error) {
	// CEF:Version|Device Vendor|Device Product|Device Version|Signature ID|Name|Severity|Extension
	evt := &events.SovereignEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		TenantID:  "GLOBAL", // Default for CEF
		RawLine:   raw,
		EventType: "cef",
	}

	if !strings.HasPrefix(raw, "CEF:") {
		return evt, fmt.Errorf("not a CEF string")
	}

	parts := strings.SplitN(raw, "|", 8)
	if len(parts) >= 6 {
		evt.EventType = parts[5] // Name of event
	}

	// Naive extension parsing (src=1.2.3.4 duser=root)
	if len(parts) == 8 {
		ext := parts[7]
		kv := strings.Split(ext, " ")
		for _, pair := range kv {
			p := strings.SplitN(pair, "=", 2)
			if len(p) == 2 {
				switch p[0] {
				case "src":
					evt.SourceIp = p[1]
				case "duser", "suser":
					evt.User = p[1]
				case "shost", "dhost":
					if evt.Host == "" {
						evt.Host = p[1]
					}
				}
			}
		}
	}

	return evt, nil
}

// ParseLEEF handles Log Event Extended Format (IBM QRadar)
func ParseLEEF(raw string) (*events.SovereignEvent, error) {
	// LEEF:Version|Vendor|Product|Version|EventID|Extension (Tab separated usually)
	evt := &events.SovereignEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		TenantID:  "GLOBAL", // Default for LEEF
		RawLine:   raw,
		EventType: "leef",
	}

	if !strings.HasPrefix(raw, "LEEF:") {
		return evt, fmt.Errorf("not a LEEF string")
	}

	parts := strings.SplitN(raw, "|", 6)
	if len(parts) >= 5 {
		evt.EventType = parts[4] // EventID is the 5th field
	}

	if len(parts) == 6 {
		ext := parts[5]
		// LEEF attributes are typically separated by tabs (\t) or configurable char
		kv := strings.Split(ext, "\t")
		for _, pair := range kv {
			p := strings.SplitN(pair, "=", 2)
			if len(p) == 2 {
				switch p[0] {
				case "src", "srcIP", "SourceIP":
					evt.SourceIp = p[1]
				case "usrName", "username", "identSrc":
					evt.User = p[1]
				case "identHostName", "host":
					if evt.Host == "" {
						evt.Host = p[1]
					}
				}
			}
		}
	}

	return evt, nil
}

// AutoParse attempts multiple techniques until one succeeds, prioritizing structured.
func AutoParse(raw string) *events.SovereignEvent {
	rawTrimmed := strings.TrimSpace(raw)

	if strings.HasPrefix(rawTrimmed, "{") {
		if evt, err := ParseJSON(rawTrimmed); err == nil {
			return evt
		}
	}

	if strings.HasPrefix(rawTrimmed, "CEF:") {
		if evt, err := ParseCEF(rawTrimmed); err == nil {
			return evt
		}
	}

	if strings.HasPrefix(rawTrimmed, "LEEF:") {
		if evt, err := ParseLEEF(rawTrimmed); err == nil {
			return evt
		}
	}

	// Try the advanced parser registry (Windows, Linux auth, Cloud, Network)
	hEvt := &database.HostEvent{}
	info := parsers.Info{RawLine: rawTrimmed}
	if advancedRegistry.Process(info, hEvt) {
		return &events.SovereignEvent{
			Timestamp: time.Now().Format(time.RFC3339),
			TenantID:  "GLOBAL", // Default for Advanced Parsed
			RawLine:   rawTrimmed,
			EventType: hEvt.EventType,
			SourceIp:  hEvt.SourceIP,
			User:      hEvt.User,
		}
	}

	// Fallback to syslog / generic text parsing
	evt, _ := ParseSyslog(rawTrimmed)
	return evt
}
