package trust

import (
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

func mk(host, msg, ingestPath string, tlsFp string, ts time.Time) events.Event {
	ev := events.Event{
		Source:    events.SourceSyslog,
		HostID:    host,
		Message:   msg,
		Timestamp: ts,
		Provenance: events.Provenance{IngestPath: ingestPath, TLSFingerprint: tlsFp},
	}
	_ = ev.Validate()
	return ev
}

func TestVerifiedNeedsAgentOrTLS(t *testing.T) {
	e := New()
	now := time.Now().UTC()
	e.Observe(mk("h", "x", "agent", "", now))
	r, _ := e.Of(lastIDFromEngine(e))
	if r == nil || r.Grade != GradeVerified {
		t.Fatalf("expected agent → verified, got %+v", r)
	}
}

func TestUntrustedAnonymous(t *testing.T) {
	e := New()
	now := time.Now().UTC()
	e.Observe(mk("h", "x", "rest", "", now))
	r, _ := e.Of(lastIDFromEngine(e))
	if r == nil || r.Grade != GradeUntrusted {
		t.Fatalf("expected rest → untrusted, got %+v", r)
	}
}

func TestSuspiciousFutureTimestamp(t *testing.T) {
	e := New()
	future := time.Now().UTC().Add(1 * time.Hour)
	e.Observe(mk("h", "x", "agent", "", future))
	r, _ := e.Of(lastIDFromEngine(e))
	if r == nil || r.Grade != GradeSuspicious {
		t.Fatalf("expected future ts → suspicious, got %+v", r)
	}
}

func TestCorroborationUpgrades(t *testing.T) {
	e := New()
	now := time.Now().UTC().Truncate(time.Minute)
	a := mk("h", "same message", "rest", "", now)
	b := mk("h", "same message", "syslog-udp", "", now.Add(time.Second))
	e.Observe(a)
	e.Observe(b)
	rA, _ := e.Of(a.ID)
	rB, _ := e.Of(b.ID)
	if rA == nil || rA.Grade != GradeConsistent {
		t.Errorf("a should be upgraded to consistent, got %+v", rA)
	}
	if rB == nil || rB.Grade != GradeConsistent {
		t.Errorf("b should be upgraded to consistent, got %+v", rB)
	}
	if !contains(rA.CorrobBy, b.ID) {
		t.Errorf("a should list b as corroborator, got %v", rA.CorrobBy)
	}
}

func TestSummary(t *testing.T) {
	e := New()
	now := time.Now().UTC()
	e.Observe(mk("h", "a", "agent", "", now))
	e.Observe(mk("h", "b", "rest", "", now))
	e.Observe(mk("h", "c", "agent", "", now.Add(time.Hour*2))) // future → suspicious
	s := e.Summary()
	if s.Verified < 1 || s.Untrusted < 1 || s.Suspicious < 1 {
		t.Errorf("summary mix wrong: %+v", s)
	}
}

// lastIDFromEngine is a test helper — picks any record (we only use it when
// exactly one event has been observed).
func lastIDFromEngine(e *Engine) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for id := range e.records {
		return id
	}
	return ""
}
