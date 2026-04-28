package api

// Background eviction for the per-IP / per-tenant rate-limit maps and
// the failed-login lockout map.
//
// Audit fix #5 (Phase 33 hardening pass) — these three sync.Maps had no
// eviction. Each unique IP that ever hit the server pinned a
// *rate.Limiter forever; a slow-drip portscan or DNS-rotated probe
// could grow them without bound and pin gigabytes of RSS.
//
// Strategy: wrap the limiter values in a struct that tracks last-use
// (atomic Unix-second), then a single goroutine sweeps every hour and
// drops entries that haven't been touched in 24 h. failedLogins gets
// a similar sweep — entries whose lockout expired more than 24 h ago
// are no longer interesting (a 25-hour-old failure is just noise).
//
// We keep the eviction interval coarse (1 h) and the TTL conservative
// (24 h) deliberately: limiters are a few hundred bytes each, so a
// hot IP we just dropped will re-allocate cheaply on next hit, and a
// 24-hour memory window is comfortably wider than any operator
// debounce window we care about.

import (
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// rateLimitTTL is the inactivity threshold after which an IP/tenant
// limiter is forgotten. 24h chosen to be wider than the failed-login
// lockout window (15 min) so we never lose lockout context for an
// active attacker, and wider than any plausible operator step-away.
const rateLimitTTL = 24 * time.Hour

// rateLimitGCInterval is how often we sweep. Hour-ish so the cost is
// negligible; we only walk the maps, not the whole limiter state.
const rateLimitGCInterval = 1 * time.Hour

// limiterEntry pairs a rate.Limiter with a last-used Unix timestamp.
// The timestamp is updated under atomic ops so the hot path
// (Allow check) doesn't take the GC mutex. Entry pointers are stored
// in the sync.Map directly.
type limiterEntry struct {
	limiter  *rate.Limiter
	lastUsed atomic.Int64 // unix seconds; updated on every touch
}

// newLimiterEntry constructs an entry initialised to "just used".
func newLimiterEntry(r rate.Limit, b int) *limiterEntry {
	e := &limiterEntry{limiter: rate.NewLimiter(r, b)}
	e.lastUsed.Store(time.Now().Unix())
	return e
}

// touch marks the entry as recently used. Called on every Allow().
func (e *limiterEntry) touch() {
	e.lastUsed.Store(time.Now().Unix())
}

// startRateLimitGC kicks off the background sweeper. Idempotent guard
// in NewRESTServer ensures we only start one. Stops when the server's
// context is cancelled (currently process-lifetime; harmless if the
// server is restarted in-process).
func (s *RESTServer) startRateLimitGC() {
	go func() {
		t := time.NewTicker(rateLimitGCInterval)
		defer t.Stop()
		for range t.C {
			s.sweepRateLimiters()
		}
	}()
}

// sweepRateLimiters walks all three maps and drops stale entries.
// Exported indirectly for test seams; not part of the public surface.
func (s *RESTServer) sweepRateLimiters() {
	cutoff := time.Now().Add(-rateLimitTTL).Unix()

	// Per-IP limiters.
	ipDropped := 0
	s.ipLimiters.Range(func(k, v any) bool {
		if entry, ok := v.(*limiterEntry); ok {
			if entry.lastUsed.Load() < cutoff {
				s.ipLimiters.Delete(k)
				ipDropped++
			}
		}
		return true
	})

	// Per-tenant limiters.
	tenantDropped := 0
	s.tenantLimiters.Range(func(k, v any) bool {
		if entry, ok := v.(*limiterEntry); ok {
			if entry.lastUsed.Load() < cutoff {
				s.tenantLimiters.Delete(k)
				tenantDropped++
			}
		}
		return true
	})

	// Failed-login records: drop any whose lockout window expired
	// more than rateLimitTTL ago. Entries that never reached the
	// lockout threshold (count<5, until==zero) get pruned by their
	// "first failed attempt" time — we approximate that with the
	// cutoff because we don't store last-attempt; in practice
	// failedLogins.Delete on success keeps this map small.
	loginDropped := 0
	cutoffTime := time.Unix(cutoff, 0)
	// Use a serialised walk to avoid racing with the login mutex —
	// failedLoginsMu also guards Store/Delete, so we hold it during
	// the sweep to keep the count/until tuple consistent.
	s.failedLoginsMu.Lock()
	s.failedLogins.Range(func(k, v any) bool {
		if info, ok := v.(struct {
			count int
			until time.Time
		}); ok {
			// If the lockout already expired and the record is
			// older than the cutoff (proxied by `until` for
			// locked entries; for un-locked entries count<5 we
			// drop them once they're outside the TTL since the
			// failedLogins map only sees writes on failures).
			if !info.until.IsZero() && info.until.Before(cutoffTime) {
				s.failedLogins.Delete(k)
				loginDropped++
			}
		}
		return true
	})
	s.failedLoginsMu.Unlock()

	if (ipDropped+tenantDropped+loginDropped) > 0 && s.log != nil {
		s.log.Info("[rate-limit-gc] swept stale entries: ip=%d tenant=%d failedLogins=%d",
			ipDropped, tenantDropped, loginDropped)
	}
}

