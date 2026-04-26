package agent

import (
	"testing"
	"time"
)

func mkLocalEvent(eventType string, data map[string]interface{}) *Event {
	return &Event{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Type:      eventType,
		Data:      data,
	}
}

// TestSSHBruteForce_FiresAfterThreshold: 5 failed-password events
// from the same IP within 60s trigger the rule.
func TestSSHBruteForce_FiresAfterThreshold(t *testing.T) {
	d := NewDetector()

	failedFrom := func(ip string) *Event {
		return mkLocalEvent("sshd", map[string]interface{}{
			"raw_line": "Failed password for root from " + ip + " port 12345 ssh2",
		})
	}

	// 4 hits — should NOT trigger (threshold=5).
	for i := 0; i < 4; i++ {
		if det := d.Evaluate(failedFrom("10.0.0.5")); det != nil {
			t.Errorf("hit %d: rule fired prematurely: %s", i+1, det.Message)
		}
	}

	// 5th hit — should trigger.
	det := d.Evaluate(failedFrom("10.0.0.5"))
	if det == nil {
		t.Fatal("rule should have fired on 5th failure")
	}
	if det.RuleID != LocalRuleSSHBruteForce {
		t.Errorf("wrong rule id: %s", det.RuleID)
	}
	if det.Context["src_ip"] != "10.0.0.5" {
		t.Errorf("src_ip context: got %q, want 10.0.0.5", det.Context["src_ip"])
	}
}

// TestSSHBruteForce_PerIPIsolation: failures from different IPs
// don't compound — each IP has its own counter.
func TestSSHBruteForce_PerIPIsolation(t *testing.T) {
	d := NewDetector()

	for i := 0; i < 4; i++ {
		_ = d.Evaluate(mkLocalEvent("sshd", map[string]interface{}{
			"raw_line": "Failed password for x from 1.1.1.1 port 1",
		}))
	}
	for i := 0; i < 4; i++ {
		_ = d.Evaluate(mkLocalEvent("sshd", map[string]interface{}{
			"raw_line": "Failed password for y from 2.2.2.2 port 1",
		}))
	}
	// Neither IP has crossed the threshold.
	det := d.Evaluate(mkLocalEvent("sshd", map[string]interface{}{
		"raw_line": "Failed password for x from 1.1.1.1 port 1",
	}))
	if det == nil {
		t.Fatal("expected fire on 5th failure for 1.1.1.1")
	}
	if det.Context["src_ip"] != "1.1.1.1" {
		t.Errorf("wrong IP fired: %s", det.Context["src_ip"])
	}
}

// TestSuspiciousSudo: catches the privilege-escalation patterns.
func TestSuspiciousSudo(t *testing.T) {
	d := NewDetector()

	cases := []struct {
		name    string
		raw     string
		fires   bool
	}{
		{"sudo bash", "user : USER=root ; COMMAND=/bin/bash", true},
		{"sudo su -", "user : COMMAND=/bin/su - ", false /* doesn't match — too noisy */},
		{"command=/bin/sh", "user : COMMAND=/bin/sh", true},
		{"sudo apt", "user : USER=root ; COMMAND=/usr/bin/apt update", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			det := d.Evaluate(mkLocalEvent("sudo", map[string]interface{}{
				"raw_line": c.raw,
			}))
			fired := det != nil
			if fired != c.fires {
				t.Errorf("got fired=%v, want %v (raw=%q)", fired, c.fires, c.raw)
			}
		})
	}
}

// TestDiscoveryCommands: stateless command match.
func TestDiscoveryCommands(t *testing.T) {
	d := NewDetector()

	cases := []struct {
		cmd   string
		fires bool
	}{
		{"whoami", true},
		{"net user", true},
		{"ipconfig /all", true},
		{"ls -la /tmp", false},
		{"vim README.md", false},
	}
	for _, c := range cases {
		t.Run(c.cmd, func(t *testing.T) {
			det := d.Evaluate(mkLocalEvent("shell", map[string]interface{}{
				"command": c.cmd,
			}))
			fired := det != nil
			if fired != c.fires {
				t.Errorf("cmd=%q: got fired=%v, want %v", c.cmd, fired, c.fires)
			}
		})
	}
}

// TestDetector_DisableHonored: when disabled, no rules fire.
func TestDetector_DisableHonored(t *testing.T) {
	d := NewDetector()
	d.SetEnabled(false)

	if det := d.Evaluate(mkLocalEvent("shell", map[string]interface{}{"command": "whoami"})); det != nil {
		t.Errorf("disabled detector should not fire, got %v", det)
	}

	d.SetEnabled(true)
	if det := d.Evaluate(mkLocalEvent("shell", map[string]interface{}{"command": "whoami"})); det == nil {
		t.Errorf("re-enabled detector should fire on whoami")
	}
}

// TestExtractSourceIP: best-effort IP parser.
func TestExtractSourceIP(t *testing.T) {
	cases := []struct {
		line, want string
	}{
		{"Failed password for root from 1.2.3.4 port 22 ssh2", "1.2.3.4"},
		{"... from 10.0.0.1:22", "10.0.0.1"},
		{"no ip here", ""},
	}
	for _, c := range cases {
		got := extractSourceIP(c.line)
		if got != c.want {
			t.Errorf("extractSourceIP(%q): got %q, want %q", c.line, got, c.want)
		}
	}
}
