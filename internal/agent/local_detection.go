// Local (edge) detection rules for the OBLIVRA agent.
//
// Closes the "Local detection rules" gap from the agent feature audit:
// the SIEM has a sophisticated detection engine, but EVERY rule fires
// after the round trip — collect → send → ingest → evaluate → alert
// → notify. For credential-stuffing or rapid-fire SSH brute force
// the latency budget is brutal: a 5-second mean detection delay can
// be the difference between catching the attacker mid-stuffing and
// finding their footprint after they've cleaned up.
//
// This file ships a small set of fast in-process rules that run on
// every event BEFORE the WAL/transport stage. When a rule fires, the
// matching event is tagged with `local_detection=<rule_id>` in its
// metadata so downstream consumers can boost priority, and the
// detector emits its own synthetic high-priority `local.detection`
// event that gets fast-tracked over the transport.
//
// Design notes:
//   - In-process, lock-free per rule (each rule maintains its own
//     local state via a small ring buffer / sliding-window counter).
//   - Stateless rules (e.g. "process discovery commands") match on
//     a single event and need no buffer.
//   - Stateful rules (e.g. "5 SSH failures from same IP in 60s")
//     keep a per-key sliding window. Memory bounded — keys are
//     evicted on TTL expiry.
//   - Rules are disable-able at runtime (`Detector.SetEnabled`) so
//     ToggleDebug from the UI can flip them off without restart.
//
// What it is NOT:
//   - A replacement for the server-side detection engine. The agent
//     can only see what its own collectors see; cross-host correlation
//     still requires the SIEM. Local rules are the "low-latency
//     first response" layer.
//   - A general-purpose YAML rule loader. The shipped rules are
//     hardcoded for performance and predictability. Adding more is
//     a code change (deliberately — agent-side rule packs are a
//     phase 32+ extension after WASM-on-agent lands).

package agent

import (
	"strings"
	"sync"
	"time"
)

// LocalRuleID is the stable identifier for an agent-side detection.
// Surfaces in event metadata under `local_detection`.
type LocalRuleID string

const (
	LocalRuleSSHBruteForce LocalRuleID = "agent.ssh_brute_force"
	LocalRuleSuspiciousSudo LocalRuleID = "agent.suspicious_sudo"
	LocalRuleDiscoveryCmds  LocalRuleID = "agent.discovery_commands"
)

// LocalDetection is the verdict returned when a local rule fires.
type LocalDetection struct {
	RuleID   LocalRuleID
	Severity string // "low" / "medium" / "high" / "critical"
	Message  string
	// Context carries rule-specific evidence (e.g. ip="1.2.3.4",
	// failure_count=8) — appended to the originating event's
	// metadata so server-side correlation has the bridge.
	Context map[string]string
}

// Detector runs every event through the registered local rules and
// returns the first match (rules are evaluated in priority order;
// most-specific first).
type Detector struct {
	mu      sync.RWMutex
	enabled bool

	sshBrute  *slidingCounter
	discovery map[string]struct{}
}

// NewDetector returns a Detector with the canonical rule set
// pre-loaded. Rules can be disabled at runtime via SetEnabled.
func NewDetector() *Detector {
	return &Detector{
		enabled: true,
		// 5 failures in 60s from the same SourceIP triggers brute-force.
		sshBrute: newSlidingCounter(60*time.Second, 5),
		discovery: map[string]struct{}{
			"whoami": {}, "id": {}, "uname": {}, "uname -a": {},
			"net user": {}, "net localgroup": {}, "ipconfig /all": {},
			"systeminfo": {}, "nltest /dclist": {},
			"hostname": {}, "ifconfig": {}, "arp -a": {},
			"netstat": {}, "ss -tnp": {},
		},
	}
}

// SetEnabled turns local detection on or off without restart. Called
// from the remote-control RPC `Toggle Debug` flow.
func (d *Detector) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// Evaluate runs all rules against the event. Returns the first match
// or nil. Performance budget: <100µs per event under steady-state load.
//
// We deliberately match on the event's normalized fields (EventType,
// RawLine, source-ip / process tokens lifted from metadata) rather
// than re-parsing — the ingest pipeline does heavy parsing
// downstream; the agent's local view is by design rough but fast.
func (d *Detector) Evaluate(evt *Event) *LocalDetection {
	if d == nil || evt == nil {
		return nil
	}
	d.mu.RLock()
	if !d.enabled {
		d.mu.RUnlock()
		return nil
	}
	d.mu.RUnlock()

	if det := d.evalSSHBruteForce(evt); det != nil {
		return det
	}
	if det := d.evalSuspiciousSudo(evt); det != nil {
		return det
	}
	if det := d.evalDiscoveryCommands(evt); det != nil {
		return det
	}
	return nil
}

// ── Rule: SSH brute force ───────────────────────────────────────────
//
// 5 "Failed password" lines from the same source IP within 60s.
// Source-IP extraction is best-effort regex over the raw syslog line.
// When it fires we DON'T zero the counter — repeat fires are useful
// to feed the SIEM with a continuous "still under attack" signal.
func (d *Detector) evalSSHBruteForce(evt *Event) *LocalDetection {
	if evt.Type != "sshd" && evt.Type != "syslog" {
		return nil
	}
	raw, ok := evt.Data["raw_line"].(string)
	if !ok {
		return nil
	}
	if !strings.Contains(raw, "Failed password") {
		return nil
	}
	ip := extractSourceIP(raw)
	if ip == "" {
		return nil
	}
	count := d.sshBrute.Increment(ip)
	if count < d.sshBrute.threshold {
		return nil
	}
	return &LocalDetection{
		RuleID:   LocalRuleSSHBruteForce,
		Severity: "high",
		Message:  "SSH brute-force pattern detected: " + ip,
		Context: map[string]string{
			"src_ip":        ip,
			"failure_count": itoa(count),
			"window":        "60s",
		},
	}
}

// ── Rule: Suspicious sudo ───────────────────────────────────────────
//
// `sudo bash`, `sudo su -`, `sudo /bin/sh`, etc. Catches the
// privilege-escalation moment. Less false-positive-prone than a pure
// "any sudo" rule because admins rarely call `sudo bash` in scripts.
func (d *Detector) evalSuspiciousSudo(evt *Event) *LocalDetection {
	if evt.Type != "sudo" && evt.Type != "syslog" {
		return nil
	}
	raw, ok := evt.Data["raw_line"].(string)
	if !ok {
		return nil
	}
	lower := strings.ToLower(raw)
	// Match either the explicit "sudo <shell>" form OR the audit-log
	// syslog format ("COMMAND=/bin/bash") which records the resolved
	// command without the `sudo` keyword. Without the audit-log form,
	// sudoers configured through the audit subsystem evade the rule.
	for _, needle := range []string{
		"sudo bash", "sudo /bin/bash", "sudo /bin/sh",
		"sudo su -", "sudo su root", "sudo dash", "sudo zsh",
		"command=/bin/bash", "command=/bin/sh", "command=/bin/zsh",
	} {
		if strings.Contains(lower, needle) {
			return &LocalDetection{
				RuleID:   LocalRuleSuspiciousSudo,
				Severity: "medium",
				Message:  "Suspicious sudo invocation: " + needle,
				Context: map[string]string{
					"pattern": needle,
				},
			}
		}
	}
	return nil
}

// ── Rule: Process-discovery commands ────────────────────────────────
//
// Stateless allowlist match on canonical post-exploitation enum
// commands. Marked low/medium because they're noisy on admin hosts
// but priceless on user workstations where they almost never fire
// legitimately.
func (d *Detector) evalDiscoveryCommands(evt *Event) *LocalDetection {
	cmd, ok := evt.Data["command"].(string)
	if !ok {
		// Fall back to raw_line for shell history / syslog sources.
		raw, _ := evt.Data["raw_line"].(string)
		cmd = raw
	}
	if cmd == "" {
		return nil
	}
	low := strings.ToLower(strings.TrimSpace(cmd))
	d.mu.RLock()
	defer d.mu.RUnlock()
	for needle := range d.discovery {
		if strings.HasPrefix(low, needle) || low == needle {
			return &LocalDetection{
				RuleID:   LocalRuleDiscoveryCmds,
				Severity: "low",
				Message:  "Post-exploit discovery command: " + needle,
				Context: map[string]string{
					"command": needle,
				},
			}
		}
	}
	return nil
}

// ── Helpers ─────────────────────────────────────────────────────────

// extractSourceIP best-effort pulls a dotted IPv4 out of a syslog line.
// Stripped to the minimum that doesn't pull in regexp at hot path —
// scan for "from <ip>" which is the standard sshd format.
func extractSourceIP(line string) string {
	idx := strings.Index(line, "from ")
	if idx < 0 {
		return ""
	}
	rest := line[idx+5:]
	// Read until whitespace or ":port".
	end := 0
	for end < len(rest) {
		c := rest[end]
		if c == ' ' || c == '\t' || c == ':' || c == '\n' {
			break
		}
		end++
	}
	if end == 0 {
		return ""
	}
	return rest[:end]
}

func itoa(i int) string {
	// strconv.Itoa would do but this avoids the import overhead in
	// a file that already runs on the hot path.
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	buf := [20]byte{}
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// ── Sliding-window counter ──────────────────────────────────────────
//
// One slidingCounter per stateful rule. Keys map to a ring of
// timestamps; on Increment, expired entries fall off the front and
// the count is the remaining length. Bounded memory: keys with no
// activity for `window` are GC'd on the next Increment (lazy).
type slidingCounter struct {
	mu        sync.Mutex
	window    time.Duration
	threshold int
	hits      map[string][]time.Time
	lastSweep time.Time
}

func newSlidingCounter(window time.Duration, threshold int) *slidingCounter {
	return &slidingCounter{
		window:    window,
		threshold: threshold,
		hits:      make(map[string][]time.Time),
	}
}

// Increment records a hit for `key` and returns the count of hits
// within the active window.
func (c *slidingCounter) Increment(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-c.window)

	// Lazy cleanup of dead keys every 30s.
	if now.Sub(c.lastSweep) > 30*time.Second {
		for k, ts := range c.hits {
			if len(ts) == 0 || ts[len(ts)-1].Before(cutoff) {
				delete(c.hits, k)
			}
		}
		c.lastSweep = now
	}

	ts := c.hits[key]
	// Drop expired entries from the front.
	i := 0
	for i < len(ts) && ts[i].Before(cutoff) {
		i++
	}
	ts = append(ts[i:], now)
	c.hits[key] = ts
	return len(ts)
}
