package ingest

import (
	"sync/atomic"
	"time"
)

// EnrichmentLimiter protects the ingestion pipeline from enrichment stalls.
// GeoIP, DNS, and Threat Intel lookups are all time-bounded and capacity-limited.
// If the per-second budget is exceeded the event is tagged with
// `enrichment_skipped:true` but processing continues unblocked — ingestion
// never waits on enrichment.
//
// Design: token-bucket using a single atomic counter reset every second by a
// background goroutine. Zero-allocation on the hot path.
type EnrichmentLimiter struct {
	MaxPerSecond int32
	remaining    atomic.Int32
	stopCh       chan struct{}
}

// NewEnrichmentLimiter creates a limiter that allows up to maxPerSecond
// enrichment calls per second across the entire pipeline.
func NewEnrichmentLimiter(maxPerSecond int) *EnrichmentLimiter {
	l := &EnrichmentLimiter{
		MaxPerSecond: int32(maxPerSecond),
		stopCh:       make(chan struct{}),
	}
	l.remaining.Store(int32(maxPerSecond))
	go l.refill()
	return l
}

// Allow returns true if an enrichment call is permitted under the budget.
// Returns false when the budget is exhausted — callers must skip enrichment
// and tag the event as enrichment_skipped.
func (l *EnrichmentLimiter) Allow() bool {
	for {
		cur := l.remaining.Load()
		if cur <= 0 {
			return false
		}
		if l.remaining.CompareAndSwap(cur, cur-1) {
			return true
		}
	}
}

// Stop terminates the background refill goroutine. Call on pipeline shutdown.
func (l *EnrichmentLimiter) Stop() {
	close(l.stopCh)
}

// refill resets the token bucket every second.
func (l *EnrichmentLimiter) refill() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.remaining.Store(l.MaxPerSecond)
		case <-l.stopCh:
			return
		}
	}
}
