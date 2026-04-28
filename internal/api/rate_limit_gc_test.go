package api

// Tests for the rate-limit GC (audit fix #5).
//
// We exercise sweepRateLimiters directly — easier than poking the hour
// ticker — and assert:
//   1. Recently-touched IP/tenant entries survive the sweep
//   2. Stale (lastUsed before cutoff) entries are dropped
//   3. failedLogins entries with expired lockout are dropped
//   4. failedLogins entries still inside the lockout window survive
//   5. Mixed map (some stale, some fresh) only drops the stale ones
//
// We construct a minimal RESTServer — just enough fields for the
// sweep code path — to avoid pulling in the full container.

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func newTestServerForGC() *RESTServer {
	// Zero-value sync.Map and sync.Mutex are usable. We leave log
	// nil — sweepRateLimiters guards on `s.log != nil`.
	return &RESTServer{}
}

func TestRateLimitGC_FreshEntriesSurvive(t *testing.T) {
	s := newTestServerForGC()
	now := time.Now().Unix()

	freshIP := newLimiterEntry(rate.Limit(5), 10)
	freshIP.lastUsed.Store(now) // just used
	s.ipLimiters.Store("192.0.2.1", freshIP)

	freshTenant := newLimiterEntry(rate.Limit(20), 50)
	freshTenant.lastUsed.Store(now)
	s.tenantLimiters.Store("tenant-A", freshTenant)

	s.sweepRateLimiters()

	if _, ok := s.ipLimiters.Load("192.0.2.1"); !ok {
		t.Fatal("fresh IP limiter must survive sweep")
	}
	if _, ok := s.tenantLimiters.Load("tenant-A"); !ok {
		t.Fatal("fresh tenant limiter must survive sweep")
	}
}

func TestRateLimitGC_StaleEntriesAreDropped(t *testing.T) {
	s := newTestServerForGC()
	stale := time.Now().Add(-2 * rateLimitTTL).Unix() // well past cutoff

	staleIP := newLimiterEntry(rate.Limit(5), 10)
	staleIP.lastUsed.Store(stale)
	s.ipLimiters.Store("192.0.2.99", staleIP)

	staleTenant := newLimiterEntry(rate.Limit(20), 50)
	staleTenant.lastUsed.Store(stale)
	s.tenantLimiters.Store("dead-tenant", staleTenant)

	s.sweepRateLimiters()

	if _, ok := s.ipLimiters.Load("192.0.2.99"); ok {
		t.Fatal("stale IP limiter must be evicted")
	}
	if _, ok := s.tenantLimiters.Load("dead-tenant"); ok {
		t.Fatal("stale tenant limiter must be evicted")
	}
}

func TestRateLimitGC_MixedSurvivalAndEviction(t *testing.T) {
	s := newTestServerForGC()
	now := time.Now().Unix()
	stale := time.Now().Add(-2 * rateLimitTTL).Unix()

	for ip, ts := range map[string]int64{
		"alive-1": now,
		"alive-2": now,
		"dead-1":  stale,
		"dead-2":  stale,
	} {
		e := newLimiterEntry(rate.Limit(5), 10)
		e.lastUsed.Store(ts)
		s.ipLimiters.Store(ip, e)
	}

	s.sweepRateLimiters()

	for _, ip := range []string{"alive-1", "alive-2"} {
		if _, ok := s.ipLimiters.Load(ip); !ok {
			t.Fatalf("expected %q to survive", ip)
		}
	}
	for _, ip := range []string{"dead-1", "dead-2"} {
		if _, ok := s.ipLimiters.Load(ip); ok {
			t.Fatalf("expected %q to be evicted", ip)
		}
	}
}

func TestRateLimitGC_FailedLoginsExpiredLockoutDropped(t *testing.T) {
	s := newTestServerForGC()
	// Lockout expired more than rateLimitTTL ago.
	expired := time.Now().Add(-2 * rateLimitTTL)
	s.failedLogins.Store("attacker@example.com", struct {
		count int
		until time.Time
	}{count: 5, until: expired})

	s.sweepRateLimiters()

	if _, ok := s.failedLogins.Load("attacker@example.com"); ok {
		t.Fatal("expired failedLogins entry must be dropped")
	}
}

func TestRateLimitGC_FailedLoginsActiveLockoutSurvives(t *testing.T) {
	s := newTestServerForGC()
	// Lockout still active (10 minutes from now).
	active := time.Now().Add(10 * time.Minute)
	s.failedLogins.Store("locked@example.com", struct {
		count int
		until time.Time
	}{count: 5, until: active})

	s.sweepRateLimiters()

	if _, ok := s.failedLogins.Load("locked@example.com"); !ok {
		t.Fatal("active-lockout failedLogins entry must survive sweep")
	}
}

func TestRateLimitGC_FailedLoginsZeroUntilSurvives(t *testing.T) {
	// Sub-threshold failures (count<5) have until=zero. The sweeper's
	// drop condition only fires for entries with a non-zero `until`
	// that's already past. Entries with zero until shouldn't be
	// touched here — they get cleared on success or by future writes.
	s := newTestServerForGC()
	s.failedLogins.Store("user@example.com", struct {
		count int
		until time.Time
	}{count: 2, until: time.Time{}})

	s.sweepRateLimiters()

	if _, ok := s.failedLogins.Load("user@example.com"); !ok {
		t.Fatal("sub-threshold failedLogins entry (until=zero) must survive sweep")
	}
}

func TestLimiterEntry_TouchUpdatesTimestamp(t *testing.T) {
	e := newLimiterEntry(rate.Limit(5), 10)
	original := e.lastUsed.Load()

	// Sleep just enough to push the unix-second forward. We use 1.1s
	// so even on a slow filesystem we cross the second boundary.
	time.Sleep(1100 * time.Millisecond)
	e.touch()

	if e.lastUsed.Load() <= original {
		t.Fatalf("touch() must advance lastUsed: before=%d after=%d",
			original, e.lastUsed.Load())
	}
}
