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
)

type ParquetEvent struct {
	ID         string `parquet:"id"`
	TenantID   string `parquet:"tenantId"`
	Timestamp  int64  `parquet:"timestamp"`
	ReceivedAt int64  `parquet:"receivedAt"`
	Source     string `parquet:"source"`
	HostID     string `parquet:"hostId"`
	EventType  string `parquet:"eventType"`
	Severity   string `parquet:"severity"`
	Message    string `parquet:"message"`
	Raw        string `parquet:"raw"`
}

func toParquet(ev events.Event) ParquetEvent {
	return ParquetEvent{
		ID:         ev.ID,
		TenantID:   ev.TenantID,
		Timestamp:  ev.Timestamp.UnixNano(),
		ReceivedAt: ev.ReceivedAt.UnixNano(),
		Source:     string(ev.Source),
		HostID:     ev.HostID,
		EventType:  ev.EventType,
		Severity:   string(ev.Severity),
		Message:    ev.Message,
		Raw:        ev.Raw,
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

type Migrator struct {
	log      *slog.Logger
	hot      *hot.Store
	warmDir  string
	maxAge   time.Duration
	tenantID string

	mu          sync.Mutex
	files       atomic.Int64
	events      atomic.Int64
	lastRun     time.Time
	lastMoved   atomic.Int64
}

type Options struct {
	WarmDir  string        // directory for parquet files
	MaxAge   time.Duration // events older than this migrate to warm
	TenantID string        // optional — defaults to "default"
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
	return &Migrator{log: log, hot: store, warmDir: opts.WarmDir, maxAge: opts.MaxAge, tenantID: opts.TenantID}, nil
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

	cutoff := time.Now().UTC().Add(-m.maxAge)
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

	stamp := time.Now().UTC().Format("20060102T150405Z")
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
	m.log.Info("warm tier write+evict", "file", path, "rows", len(rows))
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
