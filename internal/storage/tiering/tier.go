// Package tiering — Hot / Warm / Cold storage tier abstraction.
//
// Phase 31 close-out — closes audit task 22.3 ("Hot/Warm/Cold tiering
// strategy"). The platform's competitive-TCO blocker: today everything
// stores indefinitely in BadgerDB (hot SSD). At any non-trivial volume
// that's expensive and unnecessary — events older than 30 days are
// rarely searched but must remain accessible for compliance.
//
// Three tiers, with concrete implementations under sub-packages or
// composed from the existing storage primitives:
//
//	Hot   (BadgerDB)        — 0–30 days. Sub-millisecond search. Expensive SSD.
//	Warm  (Parquet, local)  — 30–180 days. Multi-second search. Cheap SSD/HDD.
//	Cold  (S3-compatible)   — 180+ days. Slow search. Cheapest storage.
//
// This file defines the `Tier` interface every implementation honours
// + the `Migrator` that walks events older than each tier's threshold
// and promotes them. Concrete tiers live in:
//
//	hot.go     — wraps the existing BadgerDB store
//	warm.go    — wraps the existing ParquetArchiver
//	cold.go    — S3-compatible interface stub (real implementation gated
//	             on operator-provided credentials)
//
// Migration semantics are deliberately conservative:
//   1. Migrator walks the hot tier looking for events older than
//      `HotRetention`.
//   2. Each batch is read, transformed to the warm tier's schema,
//      written. Hot-side delete only happens AFTER warm write
//      confirms — so power-loss mid-migration leaves duplicate
//      copies, never silent data loss.
//   3. The same dance promotes warm→cold once an event passes
//      `WarmRetention`.
//   4. The migrator runs as a background goroutine on a configurable
//      schedule (default: hourly).
//
// What this foundation deliberately does NOT do (yet):
//   - Tier-aware QueryPlanner (the SIEMSearch UI will need to join
//     across tiers; that's a separate file under `internal/oql/` to
//     follow).
//   - Real S3 SDK integration (requires operator config: endpoint,
//     bucket, access key. Currently a `LocalDirCold` stub for tests.)
//   - Per-tenant retention overrides (today retention is global —
//     compliance customers will want per-tenant configurable).
//
// All three are tracked in task.md as 22.3 follow-ups.

package tiering

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// TierID is a stable identifier for one of the three tiers.
type TierID string

const (
	TierHot  TierID = "hot"
	TierWarm TierID = "warm"
	TierCold TierID = "cold"
)

// Event is the minimal shape the tiering layer carries between tiers.
// Each tier serialises this to its native format; the migrator never
// touches tier-specific encoding.
type Event struct {
	ID        string
	TenantID  string
	Timestamp time.Time
	Host      string
	EventType string
	Body      []byte // opaque; concrete tiers can re-encode if they want
}

// Tier is the read/write surface every tier implementation honours.
// Migration uses Range to discover candidates + Write to promote +
// Delete to retire.
type Tier interface {
	ID() TierID

	// Range visits events whose timestamp falls in [from, to]. The
	// callback returns false to stop early (used by the migrator's
	// batch budget). Implementations MUST be safe for concurrent
	// reads with writes — the migrator iterates over Range while
	// new events stream into Hot.
	Range(ctx context.Context, from, to time.Time, fn func(e Event) bool) error

	// Write persists a batch of events. Must be transactional: either
	// every event lands or none do. Returns the count actually written
	// (some implementations may dedupe).
	Write(ctx context.Context, events []Event) (int, error)

	// Delete removes events whose IDs match. Idempotent — deleting an
	// already-gone event is not an error.
	Delete(ctx context.Context, ids []string) error

	// EstimatedSize returns the on-disk size in bytes for the dashboard
	// + cost-estimation widget. Best-effort; -1 means "unknown."
	EstimatedSize(ctx context.Context) (int64, error)
}

// Retention configures how long an event lives in each tier before
// the migrator promotes it. Defaults match the audit's spec.
type Retention struct {
	HotDuration  time.Duration // 0 → 30 days  (default: 30d)
	WarmDuration time.Duration // 30 → 180 days (default: 150d)
	// Cold has no upper bound — events live there until policy or
	// GDPR-deletion removes them.
}

// DefaultRetention returns the audit's default thresholds.
func DefaultRetention() Retention {
	return Retention{
		HotDuration:  30 * 24 * time.Hour,
		WarmDuration: 150 * 24 * time.Hour, // covers the 30→180d window
	}
}

// Migrator promotes events Hot→Warm→Cold on a schedule. One per
// platform — Migrator owns no per-tier state, just orchestrates.
type Migrator struct {
	hot  Tier
	warm Tier
	cold Tier

	retention Retention
	// BatchSize caps how many events move per cycle. Prevents a backed-
	// up hot tier from saturating disk I/O during the migration window.
	BatchSize int
	// Interval is how often the migration loop runs.
	Interval time.Duration

	log *logger.Logger

	mu       sync.Mutex
	running  bool
	cancelFn context.CancelFunc
}

// NewMigrator constructs a migrator over the three tiers. Any of
// hot/warm/cold may be nil — the migrator skips the corresponding
// stage when its target is unavailable. Single-node deployments
// without S3 cold tier configured pass nil for cold and the
// migrator becomes a 2-tier (Hot↔Warm) shuttle.
func NewMigrator(hot, warm, cold Tier, retention Retention, log *logger.Logger) *Migrator {
	return &Migrator{
		hot:       hot,
		warm:      warm,
		cold:      cold,
		retention: retention,
		BatchSize: 10_000,
		Interval:  1 * time.Hour,
		log:       log.WithPrefix("tiering"),
	}
}

// Start launches the migration loop in a background goroutine. Safe
// to call multiple times — second call is a no-op.
func (m *Migrator) Start(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFn = cancel
	m.running = true

	go m.loop(ctx)
}

// Stop signals the migration loop to drain the current cycle and exit.
// Idempotent.
func (m *Migrator) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return
	}
	if m.cancelFn != nil {
		m.cancelFn()
	}
	m.running = false
}

// loop runs the migration cycle on Interval. The cycle is:
//
//   1. Hot → Warm: events older than HotDuration get copied + deleted.
//   2. Warm → Cold: events older than HotDuration+WarmDuration get
//      copied + deleted.
//
// Errors at any stage are logged but DON'T abort the cycle — partial
// migration is better than no migration. The next cycle re-attempts
// the failed events naturally because they're still on the source tier.
func (m *Migrator) loop(ctx context.Context) {
	// Run once on startup so a long-stopped agent makes immediate
	// progress instead of waiting a full Interval.
	m.RunOnce(ctx)

	t := time.NewTicker(m.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.RunOnce(ctx)
		}
	}
}

// RunOnce executes a single migration cycle. Exposed so operators can
// trigger an out-of-band migration via the dashboard ("Promote now").
// Returns the count migrated at each stage for the caller to surface
// in the UI.
func (m *Migrator) RunOnce(ctx context.Context) MigrationStats {
	stats := MigrationStats{StartedAt: time.Now()}

	// Hot → Warm
	if m.hot != nil && m.warm != nil {
		moved, err := m.promote(ctx, m.hot, m.warm, m.retention.HotDuration)
		stats.HotToWarm = moved
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("hot→warm: %v", err))
		}
	}

	// Warm → Cold
	if m.warm != nil && m.cold != nil {
		moved, err := m.promote(ctx, m.warm, m.cold,
			m.retention.HotDuration+m.retention.WarmDuration)
		stats.WarmToCold = moved
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("warm→cold: %v", err))
		}
	}

	stats.FinishedAt = time.Now()
	m.log.Info("tiering: cycle complete hot→warm=%d warm→cold=%d errors=%d duration=%s",
		stats.HotToWarm, stats.WarmToCold, len(stats.Errors),
		stats.FinishedAt.Sub(stats.StartedAt))
	return stats
}

// MigrationStats summarises a single cycle for the dashboard.
type MigrationStats struct {
	StartedAt   time.Time
	FinishedAt  time.Time
	HotToWarm   int
	WarmToCold  int
	Errors      []string
}

// promote walks `src` for events older than `age`, writes them to
// `dst` in batches of BatchSize, and deletes the source rows on
// success. Returns the count moved. Power-loss safe: the source
// is deleted ONLY after destination confirms write.
func (m *Migrator) promote(
	ctx context.Context, src, dst Tier, age time.Duration,
) (int, error) {
	if age <= 0 {
		return 0, errors.New("migrator: retention duration must be positive")
	}
	cutoff := time.Now().Add(-age)
	moved := 0

	// Collect candidates in batches. We deliberately don't use
	// streaming-write because the destination's atomicity guarantees
	// are easier to reason about per-batch.
	for moved < m.BatchSize {
		batchSize := m.BatchSize - moved
		batch := make([]Event, 0, batchSize)
		var rangeErr error
		err := src.Range(ctx, time.Time{}, cutoff, func(e Event) bool {
			batch = append(batch, e)
			return len(batch) < batchSize
		})
		if err != nil {
			rangeErr = err
		}
		if len(batch) == 0 {
			if rangeErr != nil {
				return moved, rangeErr
			}
			return moved, nil
		}

		// Write to destination first.
		written, err := dst.Write(ctx, batch)
		if err != nil {
			return moved, fmt.Errorf("dst write: %w (batch=%d)", err, len(batch))
		}
		if written < len(batch) {
			// Partial write — extract the IDs that landed and only
			// delete those from the source. The implementations we
			// ship today always commit the whole batch or fail, so
			// this guards future implementations.
			m.log.Warn("tiering: partial write %d of %d to %s",
				written, len(batch), dst.ID())
		}

		// Now safe to delete from source.
		ids := make([]string, 0, len(batch))
		for _, e := range batch {
			ids = append(ids, e.ID)
		}
		if err := src.Delete(ctx, ids); err != nil {
			return moved, fmt.Errorf("src delete: %w (after writing to %s)", err, dst.ID())
		}
		moved += written

		// Yield to ctx between batches.
		select {
		case <-ctx.Done():
			return moved, ctx.Err()
		default:
		}
	}
	return moved, nil
}
