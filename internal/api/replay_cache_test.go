package api

// Tests for the ReplayCache (audit fix #1).
//
// We exercise:
//   1. First-seen → false, second-seen → true (basic dedup)
//   2. Different agent_id, same ts+body → both first-seen=false
//   3. Same agent_id+ts, different body bytes → both first-seen=false
//   4. TTL expiry: an entry older than TTL stops counting as a replay
//   5. Bounded growth: > maxEntries triggers oldest-eviction
//
// We do NOT test the background goroutine timing directly (flaky); we
// drive eviction via the synchronous path that runs inside Seen().

import (
	"fmt"
	"testing"
	"time"
)

func TestReplayCache_FirstSeenIsFalse(t *testing.T) {
	c := NewReplayCache(60 * time.Second)
	if c.Seen("agent-A", "1700000000", []byte("payload")) {
		t.Fatal("first call must report Seen=false")
	}
}

func TestReplayCache_DuplicateIsDetected(t *testing.T) {
	c := NewReplayCache(60 * time.Second)
	c.Seen("agent-A", "1700000000", []byte("payload"))
	if !c.Seen("agent-A", "1700000000", []byte("payload")) {
		t.Fatal("second identical call must report Seen=true")
	}
}

func TestReplayCache_DifferentAgentIDIsDistinct(t *testing.T) {
	c := NewReplayCache(60 * time.Second)
	c.Seen("agent-A", "1700000000", []byte("payload"))
	if c.Seen("agent-B", "1700000000", []byte("payload")) {
		t.Fatal("different agent_id must be a new fingerprint")
	}
}

func TestReplayCache_DifferentBodyIsDistinct(t *testing.T) {
	c := NewReplayCache(60 * time.Second)
	c.Seen("agent-A", "1700000000", []byte("payload-1"))
	if c.Seen("agent-A", "1700000000", []byte("payload-2")) {
		t.Fatal("different body must be a new fingerprint")
	}
}

func TestReplayCache_DifferentTimestampIsDistinct(t *testing.T) {
	c := NewReplayCache(60 * time.Second)
	c.Seen("agent-A", "1700000000", []byte("payload"))
	if c.Seen("agent-A", "1700000001", []byte("payload")) {
		t.Fatal("different timestamp must be a new fingerprint")
	}
}

func TestReplayCache_TTLExpiry(t *testing.T) {
	// 50ms TTL keeps the test fast. The synchronous path in Seen()
	// re-checks the TTL even before the background sweeper fires —
	// that's the path we exercise here.
	c := NewReplayCache(50 * time.Millisecond)
	c.Seen("agent-A", "1700000000", []byte("payload"))

	// Manually back-date the entry so we don't sleep flakily.
	c.mu.Lock()
	for k := range c.entries {
		c.entries[k] = time.Now().Add(-1 * time.Second) // > TTL
	}
	c.mu.Unlock()

	if c.Seen("agent-A", "1700000000", []byte("payload")) {
		t.Fatal("expired entry must be treated as fresh")
	}
}

func TestReplayCache_BoundedEviction(t *testing.T) {
	c := NewReplayCache(60 * time.Second)
	// Lower the cap so the test is fast and deterministic. The
	// production cap (100k) doesn't change the behaviour we're
	// asserting — only the size at which it fires.
	c.maxEntries = 100

	for i := 0; i < 200; i++ {
		c.Seen("agent-A", fmt.Sprintf("ts-%d", i), []byte("payload"))
	}

	c.mu.Lock()
	got := len(c.entries)
	c.mu.Unlock()

	// After overflow, evictOldest(maxEntries/10)=10 fires when the map
	// hits 101. We loop past 200 inserts, so multiple eviction passes
	// run. The post-condition we care about: the map never grows
	// unboundedly past the cap by more than one trigger window.
	if got > c.maxEntries+10 {
		t.Fatalf("bounded eviction failed: got %d entries with cap %d", got, c.maxEntries)
	}
}

func TestReplayCache_FingerprintDeterministic(t *testing.T) {
	a := fingerprint("agent-A", "1700000000", []byte("payload"))
	b := fingerprint("agent-A", "1700000000", []byte("payload"))
	if a != b {
		t.Fatal("fingerprint must be deterministic for identical inputs")
	}
	c := fingerprint("agent-B", "1700000000", []byte("payload"))
	if a == c {
		t.Fatal("fingerprint must differ when agent_id differs")
	}
}

// Fingerprint length sanity — sha256 hex == 64 chars. Catches a regression
// where someone swaps in a different hash and silently truncates.
func TestReplayCache_FingerprintShape(t *testing.T) {
	fp := fingerprint("agent-A", "1700000000", []byte("payload"))
	if len(fp) != 64 {
		t.Fatalf("expected 64-char sha256 hex, got %d (%q)", len(fp), fp)
	}
}
