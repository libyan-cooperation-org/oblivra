package api

// Replay-attack cache for agent endpoints.
//
// HMAC + a 30-second timestamp window doesn't prevent replay within
// the window: an attacker who captures a valid request can re-POST
// it bit-for-bit and the server accepts it. The fix is to track
// (agent_id, timestamp, body_hash) tuples and reject duplicates.
//
// We keep this in-memory rather than persisting to SQLite because:
//
//  • The replay window is 30s. Anything older than 60s is moot —
//    the HMAC validator will reject it on timestamp drift.
//  • A bounded LRU keyed by sha256(agent_id|ts|body) fits in <1 MB
//    even at 10k events/sec.
//  • Memory survives a single-process REST server's lifetime;
//    after a restart the timestamp window naturally re-protects
//    the first 30s.
//
// Cluster deployments (multiple REST servers behind a LB) need a
// shared cache (Redis / etcd). For Phase 33's single-binary single-
// instance deployment, in-process is sufficient. Phase 34 follow-up
// adds a pluggable backend if/when we ship clustering.

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// ReplayCache holds recently-seen (agent, timestamp, body) fingerprints.
// 60-second TTL — comfortably wider than the 30s HMAC window so we
// never miss a replay due to clock-edge effects.
type ReplayCache struct {
	mu      sync.Mutex
	entries map[string]time.Time // fingerprint → first seen
	ttl     time.Duration

	// Bounded growth: evict oldest when len > maxEntries. Otherwise a
	// burst of malformed-but-distinct payloads (one per ms from one
	// agent) could pin memory until the TTL ticker fires.
	maxEntries int
}

// NewReplayCache constructs the cache. ttl is how long to remember a
// fingerprint; the TTL must be ≥ the HMAC timestamp window to be
// useful (we use 60s; HMAC window is 30s).
func NewReplayCache(ttl time.Duration) *ReplayCache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	c := &ReplayCache{
		entries:    map[string]time.Time{},
		ttl:        ttl,
		maxEntries: 100_000,
	}
	go c.evictLoop()
	return c
}

// Seen returns true if this (agent, ts, body) fingerprint has been
// recorded already. Records the fingerprint as a side effect — so a
// caller's normal usage is:
//
//	if cache.Seen(agentID, ts, body) {
//	    return 409
//	}
//	// proceed
func (c *ReplayCache) Seen(agentID, timestamp string, body []byte) bool {
	fp := fingerprint(agentID, timestamp, body)
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if seen, exists := c.entries[fp]; exists {
		// If TTL has expired, we treat it as fresh. (The evict loop
		// usually catches this first, but the sync path is the
		// authoritative one.)
		if now.Sub(seen) <= c.ttl {
			return true
		}
	}
	c.entries[fp] = now

	// Bounded growth: if we're over the cap, evict the oldest 10% in
	// one pass to amortise.
	if len(c.entries) > c.maxEntries {
		c.evictOldest(c.maxEntries / 10)
	}
	return false
}

func (c *ReplayCache) evictLoop() {
	t := time.NewTicker(c.ttl / 2)
	defer t.Stop()
	for now := range t.C {
		c.mu.Lock()
		for k, v := range c.entries {
			if now.Sub(v) > c.ttl {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

// evictOldest removes the n entries with the oldest first-seen time.
// Caller MUST hold c.mu.
func (c *ReplayCache) evictOldest(n int) {
	if n <= 0 || len(c.entries) == 0 {
		return
	}
	type pair struct {
		k string
		t time.Time
	}
	all := make([]pair, 0, len(c.entries))
	for k, v := range c.entries {
		all = append(all, pair{k, v})
	}
	// Partial sort would be cheaper; we do a full sort for simplicity
	// and because eviction only fires when the map is past 100k.
	for i := 0; i < n && i < len(all); i++ {
		oldestIdx := i
		for j := i + 1; j < len(all); j++ {
			if all[j].t.Before(all[oldestIdx].t) {
				oldestIdx = j
			}
		}
		all[i], all[oldestIdx] = all[oldestIdx], all[i]
		delete(c.entries, all[i].k)
	}
}

// fingerprint builds the cache key. SHA256 over the concatenation of
// the three fields means even if two agents happen to send identical
// bodies at the identical timestamp, they're treated as distinct.
func fingerprint(agentID, timestamp string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(agentID))
	h.Write([]byte{0})
	h.Write([]byte(timestamp))
	h.Write([]byte{0})
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
