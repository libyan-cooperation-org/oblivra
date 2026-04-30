package parsers

import (
	"strings"
	"testing"

	"github.com/kingknull/oblivra/internal/events"
)

func TestSniff(t *testing.T) {
	cases := []struct {
		in   string
		want Format
	}{
		{`{"a":1}`, FormatJSON},
		{`<34>1 2026-04-30T12:00:00Z host app - - - hello`, FormatRFC5424},
		{`<34>Apr 30 12:00:00 host sshd: hi`, FormatRFC3164},
		{"CEF:0|Vendor|Product|1|100|Bad|7|src=1.2.3.4", FormatCEF},
		{"raw plain log line", FormatAuto},
	}
	for _, c := range cases {
		t.Run(c.in[:min(len(c.in), 40)], func(t *testing.T) {
			if got := sniff(c.in); got != c.want {
				t.Fatalf("sniff(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	raw := `{"timestamp":"2026-04-30T10:00:00Z","host":"web-01","severity":"warning","message":"hi","extra":"v","n":42}`
	ev, err := Parse(raw, FormatJSON)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Message != "hi" {
		t.Errorf("message = %q", ev.Message)
	}
	if ev.HostID != "web-01" {
		t.Errorf("host = %q", ev.HostID)
	}
	if ev.Severity != events.SeverityWarn {
		t.Errorf("severity = %q", ev.Severity)
	}
	if ev.Fields["extra"] != "v" {
		t.Errorf("fields.extra = %q", ev.Fields["extra"])
	}
	if ev.Fields["n"] != "42" {
		t.Errorf("fields.n = %q", ev.Fields["n"])
	}
	if ev.Raw != raw {
		t.Errorf("raw not preserved")
	}
}

func TestParseRFC5424(t *testing.T) {
	raw := `<34>1 2026-04-30T12:34:56Z dc-02 sshd 1234 ID47 - Failed password for root from 10.0.0.99`
	ev, err := Parse(raw, FormatRFC5424)
	if err != nil {
		t.Fatal(err)
	}
	if ev.HostID != "dc-02" {
		t.Errorf("host = %q", ev.HostID)
	}
	if !strings.Contains(ev.Message, "Failed password") {
		t.Errorf("message = %q", ev.Message)
	}
	// PRI 34 = facility 4 (auth), severity 2 (critical)
	if ev.Severity != events.SeverityCritical {
		t.Errorf("severity = %q (want critical)", ev.Severity)
	}
	if ev.Fields["app"] != "sshd" {
		t.Errorf("app = %q", ev.Fields["app"])
	}
	if ev.Fields["pid"] != "1234" {
		t.Errorf("pid = %q", ev.Fields["pid"])
	}
	if ev.Timestamp.Year() != 2026 {
		t.Errorf("ts not parsed: %v", ev.Timestamp)
	}
}

func TestParseRFC3164(t *testing.T) {
	raw := `<38>Apr 30 12:34:56 web-01 sshd[6789]: Accepted publickey for admin from 192.168.1.1`
	ev, err := Parse(raw, FormatRFC3164)
	if err != nil {
		t.Fatal(err)
	}
	if ev.HostID != "web-01" {
		t.Errorf("host = %q", ev.HostID)
	}
	if ev.Fields["app"] != "sshd" {
		t.Errorf("app = %q", ev.Fields["app"])
	}
	if ev.Fields["pid"] != "6789" {
		t.Errorf("pid = %q", ev.Fields["pid"])
	}
	if !strings.Contains(ev.Message, "Accepted publickey") {
		t.Errorf("message = %q", ev.Message)
	}
}

func TestParseCEF(t *testing.T) {
	raw := `CEF:0|Trend|DeepSecurity|10.0|600|Login Failure|6|src=10.0.0.1 act=blocked dpt=22`
	ev, err := Parse(raw, FormatCEF)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Message != "Login Failure" {
		t.Errorf("message = %q", ev.Message)
	}
	if ev.Severity != events.SeverityWarn {
		t.Errorf("severity = %q (want warning)", ev.Severity)
	}
	if ev.Fields["vendor"] != "Trend" {
		t.Errorf("vendor = %q", ev.Fields["vendor"])
	}
	if ev.Fields["src"] != "10.0.0.1" {
		t.Errorf("src = %q", ev.Fields["src"])
	}
	if ev.Fields["act"] != "blocked" {
		t.Errorf("act = %q", ev.Fields["act"])
	}
	if ev.HostID != "10.0.0.1" {
		t.Errorf("host populated from src expected, got %q", ev.HostID)
	}
}

func TestParseAuto(t *testing.T) {
	for _, raw := range []string{
		`{"message":"hi","host":"a"}`,
		`<13>1 2026-04-30T00:00:00Z host - - - - hello`,
		`<13>Apr 30 00:00:00 h sshd: hi`,
		`CEF:0|V|P|1|1|x|0|`,
	} {
		ev, err := Parse(raw, FormatAuto)
		if err != nil {
			t.Errorf("auto %q: %v", raw[:min(len(raw), 40)], err)
		}
		if ev == nil || ev.Raw == "" {
			t.Errorf("auto %q: empty event", raw[:min(len(raw), 40)])
		}
	}
}

func TestParseEmptyRejected(t *testing.T) {
	if _, err := Parse("", FormatJSON); err == nil {
		t.Error("expected error on empty input")
	}
	if _, err := Parse("   \n  \r", FormatAuto); err == nil {
		t.Error("expected error on whitespace-only input")
	}
}

func TestPlainFallback(t *testing.T) {
	ev, err := Parse("just a message", FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "plain" {
		t.Errorf("eventType = %q", ev.EventType)
	}
	if ev.Message != "just a message" {
		t.Errorf("message = %q", ev.Message)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
