package reconstruction

import (
	"context"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

func mk(host, msg, etype string, ts time.Time) events.Event {
	ev := events.Event{
		Source:    events.SourceSyslog,
		HostID:    host,
		Message:   msg,
		EventType: etype,
		Timestamp: ts,
	}
	_ = ev.Validate()
	return ev
}

func TestSshSessionLifecycle(t *testing.T) {
	se := NewSessionEngine()
	now := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)

	se.Observe(context.Background(), mk("web-01",
		"sshd[1234]: Failed password for root from 10.0.0.5 port 22 ssh2", "", now))
	se.Observe(context.Background(), mk("web-01",
		"sshd[1234]: Accepted publickey for alice from 10.0.0.5 port 22 ssh2", "", now.Add(time.Second)))
	se.Observe(context.Background(), mk("web-01",
		"sshd[1234]: pam_unix(sshd:session): session closed for user alice", "", now.Add(2*time.Second)))

	got := se.Sessions("web-01")
	// We should see two distinct sessions: root (failed) and alice (closed).
	if len(got) != 2 {
		t.Fatalf("expected 2 sessions, got %d: %+v", len(got), got)
	}
	var alice, root *Session
	for i := range got {
		if got[i].User == "alice" {
			alice = &got[i]
		}
		if got[i].User == "root" {
			root = &got[i]
		}
	}
	if alice == nil || alice.State != SessionClosed {
		t.Errorf("alice session: %+v", alice)
	}
	if alice == nil || alice.Method != "ssh-publickey" {
		t.Errorf("alice method: %+v", alice)
	}
	if root == nil || root.State != SessionFailed {
		t.Errorf("root failed session: %+v", root)
	}
	if root == nil || root.FailedAttempts != 1 {
		t.Errorf("root failed count: %+v", root)
	}
}

func TestExplicitEventTypeFastPath(t *testing.T) {
	se := NewSessionEngine()
	ev := events.Event{
		Source: events.SourceAgent, HostID: "h", Message: "x",
		EventType: "login_success",
		Fields:    map[string]string{"user": "carol", "src_ip": "1.2.3.4", "method": "rdp"},
		Timestamp: time.Now().UTC(),
	}
	_ = ev.Validate()
	se.Observe(context.Background(), ev)

	got := se.Sessions("h")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].User != "carol" || got[0].SourceIP != "1.2.3.4" || got[0].Method != "rdp" {
		t.Errorf("session = %+v", got[0])
	}
	if got[0].State != SessionOpen {
		t.Errorf("state = %s", got[0].State)
	}
}

func TestUnclassifiedEventIgnored(t *testing.T) {
	se := NewSessionEngine()
	se.Observe(context.Background(), mk("h", "totally unrelated message", "", time.Now()))
	if got := se.Sessions(""); len(got) != 0 {
		t.Errorf("expected 0 sessions, got %v", got)
	}
}

func TestSessionsScopedByHost(t *testing.T) {
	se := NewSessionEngine()
	now := time.Now().UTC()
	se.Observe(context.Background(), mk("a",
		"sshd[1]: Accepted password for u from 1.1.1.1 port 22 ssh2", "", now))
	se.Observe(context.Background(), mk("b",
		"sshd[2]: Accepted password for u from 1.1.1.1 port 22 ssh2", "", now))

	if a := se.Sessions("a"); len(a) != 1 {
		t.Errorf("host-a sessions: %d", len(a))
	}
	if b := se.Sessions("b"); len(b) != 1 {
		t.Errorf("host-b sessions: %d", len(b))
	}
	if all := se.Sessions(""); len(all) != 2 {
		t.Errorf("all sessions: %d", len(all))
	}
}
