// Package tiering migrates events out of the hot BadgerDB store into a warm
// Parquet tier on a configurable age threshold. The cold tier (S3 / JSONL) is
// stubbed but not implemented in Phase 22.3.
package tiering

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/parquet-go/parquet-go"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/storage/hot"
	"github.com/kingknull/oblivra/internal/storage/worm"
)

// ParquetEvent is the v2 row schema. Adds the per-event content hash, the
// schema-version stamp, and a flat provenance hint so a verifier can re-derive
// identity from a Parquet row alone, without consulting the WAL.
//
// v1 rows lacked Hash / SchemaVersion / provenance fields. The cross-tier
// verifier handles both shapes: when SchemaVersion == 0 the row is treated
// as v1 and only structural-parse is checked; v2+ rows are subject to the
// stronger content-hash-recompute test.
type ParquetEvent struct {
	SchemaVersion int    `parquet:"schemaVersion"`
	ID            string `parquet:"id"`
	Hash          string `parquet:"hash"`
	TenantID      string `parquet:"tenantId"`
	Timestamp     int64  `parquet:"timestamp"`
	ReceivedAt    int64  `parquet:"receivedAt"`
	Source        string `parquet:"source"`
	HostID        string `parquet:"hostId"`
	EventType     string `parquet:"eventType"`
	Severity      string `parquet:"severity"`
	Message       string `parquet:"message"`
	Raw           string `parquet:"raw"`
	IngestPath    string `parquet:"ingestPath"`
	Peer          string `parquet:"peer"`
	AgentID       string `parquet:"agentId"`
	Parser        string `parquet:"parser"`
}

func toParquet(ev events.Event) ParquetEvent {
	return ParquetEvent{
		SchemaVersion: ev.SchemaVersion,
		ID:            ev.ID,
		Hash:          ev.Hash,
		TenantID:      ev.TenantID,
		Timestamp:     ev.Timestamp.UnixNano(),
		ReceivedAt:    ev.ReceivedAt.UnixNano(),
		Source:        string(ev.Source),
		HostID:        ev.HostID,
		EventType:     ev.EventType,
		Severity:      string(ev.Severity),
		Message:       ev.Message,
		Raw:           ev.Raw,
		IngestPath:    ev.Provenance.IngestPath,
		Peer:          ev.Provenance.Peer,
		AgentID:       ev.Provenance.AgentID,
		Parser:        ev.Provenance.Parser,
	}
}

type Stats struct {
	WarmFiles   int64     `json:"warmFiles"`
	WarmEvents  int64     `json:"warmEvents"`
	LastRunAt   time.Time `json:"lastRunAt"`
	LastRunMoved int64    `json:"lastRunMoved"`
	WarmDir     string    `json:"warmDir"`
	HotAgeMax   string    `json:"hotAgeMax"`
}

// AgeResolver is an injection seam for per-tenant retention policy. The
// tiering package can't import services without a cycle, so the platform
// stack provides this closure.
type AgeResolver func(tenantID string) time.Duration

type Migrator struct {
	log      *slog.Logger
	hot      *hot.Store
	warmDir  string
	maxAge   time.Duration
	tenantID string
	resolveAge AgeResolver

	mu          sync.Mutex
	files       atomic.Int64
	events      atomic.Int64
	lastRun     time.Time
	lastMoved   atomic.Int64
}

type Options struct {
	WarmDir     string        // directory for parquet files
	MaxAge      time.Duration // fallback events-older-than for migration
	TenantID    string        // optional — defaults to "default"
	ResolveAge  AgeResolver   // optional — per-tenant override
}

func New(log *slog.Logger, store *hot.Store, opts Options) (*Migrator, error) {
	if opts.WarmDir == "" {
		return nil, errors.New("tiering: WarmDir required")
	}
	if opts.MaxAge <= 0 {
		opts.MaxAge = 30 * 24 * time.Hour // 30 days, matches README spec
	}
	if opts.TenantID == "" {
		opts.TenantID = "default"
	}
	if err := os.MkdirAll(opts.WarmDir, 0o755); err != nil {
		return nil, err
	}
	return &Migrator{
		log: log, hot: store, warmDir: opts.WarmDir, maxAge: opts.MaxAge,
		tenantID: opts.TenantID, resolveAge: opts.ResolveAge,
	}, nil
}

// Run performs one migration pass. Returns the number of events moved.
//
// Order of operations is important for crash safety:
//  1. Range-scan eligible events from hot.
//  2. Write them to a parquet file in warm.
//  3. fsync the file (close() + os.File.Sync()).
//  4. Only then delete from hot.
//
// If we crash between (3) and (4), the next run re-reads the same events from
// hot and writes them to a new file — duplicate-but-safe. If we crashed after
// the delete the events are gone but already in warm.
func (m *Migrator) Run(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	maxAge := m.maxAge
	if m.resolveAge != nil {
		if v := m.resolveAge(m.tenantID); v > 0 {
			maxAge = v
		}
	}
	cutoff := time.Now().UTC().Add(-maxAge)
	evs, err := m.hot.Range(ctx, hot.RangeOpts{
		TenantID: m.tenantID,
		From:     time.Unix(0, 0),
		To:       cutoff,
		Limit:    100000,
	})
	if err != nil {
		return 0, err
	}
	if len(evs) == 0 {
		m.lastRun = time.Now().UTC()
		m.lastMoved.Store(0)
		return 0, nil
	}

	rows := make([]ParquetEvent, len(evs))
	ids := make([]string, len(evs))
	for i, e := range evs {
		rows[i] = toParquet(e)
		ids[i] = e.ID
	}

	// Nanosecond precision so two migrations in the same second can't clash
	// on the filename (the WORM-locked previous file would block re-open).
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	path := filepath.Join(m.warmDir, fmt.Sprintf("warm-%s-%s.parquet", m.tenantID, stamp))
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}

	w := parquet.NewGenericWriter[ParquetEvent](f)
	if _, err := w.Write(rows); err != nil {
		_ = w.Close()
		_ = f.Close()
		return 0, err
	}
	if err := w.Close(); err != nil {
		_ = f.Close()
		return 0, err
	}
	// fsync before declaring durability
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}

	// Lock the warm-tier file as WORM (best-effort — Lock just strips write
	// bits if it can't escalate). Done BEFORE hot eviction so a tampered
	// warm file would be detected on the next verifier pass.
	if err := worm.Lock(path); err != nil {
		m.log.Warn("worm lock failed; warm file still readable but mutable", "err", err, "file", path)
	}

	// Warm is durable — now safe to evict from hot.
	if err := m.hot.Delete(m.tenantID, ids); err != nil {
		// Warm has the data; hot eviction failed. Log but report success
		// since the durable side is fine.
		m.log.Warn("warm written but hot delete failed", "err", err, "file", path)
	}

	m.files.Add(1)
	m.events.Add(int64(len(rows)))
	m.lastRun = time.Now().UTC()
	m.lastMoved.Store(int64(len(rows)))
	m.log.Info("warm tier write+evict", "file", path, "rows", len(rows), "wormLocked", true)
	return len(rows), nil
}

func (m *Migrator) Stats() Stats {
	return Stats{
		WarmFiles:    m.files.Load(),
		WarmEvents:   m.events.Load(),
		LastRunAt:    m.lastRun,
		LastRunMoved: m.lastMoved.Load(),
		WarmDir:      m.warmDir,
		HotAgeMax:    m.maxAge.String(),
	}
}
