// Package ingest is the entry point every event must pass through. It writes
// to the WAL first (durability), then to the hot store (queryability).
package ingest

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/storage/hot"
	"github.com/kingknull/oblivra/internal/storage/search"
	"github.com/kingknull/oblivra/internal/wal"
)

type Pipeline struct {
	log    *slog.Logger
	wal    *wal.WAL
	hot    *hot.Store
	search *search.Index
	bus    *events.Bus
	total  atomic.Int64

	// EPS sampling — naive 1s window, replaced in Phase 1b with a ring.
	lastSecond atomic.Int64
	lastCount  atomic.Int64
	currentEPS atomic.Int64
}

func New(log *slog.Logger, w *wal.WAL, h *hot.Store, idx *search.Index, bus *events.Bus) *Pipeline {
	return &Pipeline{log: log, wal: w, hot: h, search: idx, bus: bus}
}

// Submit accepts an event, validates it, and persists it through WAL → hot store.
func (p *Pipeline) Submit(ctx context.Context, ev *events.Event) error {
	if ev == nil {
		return errors.New("ingest: nil event")
	}
	if err := ev.Validate(); err != nil {
		return err
	}
	if err := p.wal.Append(ev); err != nil {
		return err
	}
	if err := p.hot.Put(ev); err != nil {
		// WAL succeeded but hot store didn't — caller can retry from WAL replay.
		p.log.Error("hot store put failed; event durably in WAL", "id", ev.ID, "err", err)
		return err
	}
	if p.search != nil {
		if err := p.search.Index(ev); err != nil {
			// Index miss is non-fatal — search degrades gracefully, hot store still has it.
			p.log.Warn("search index failed", "id", ev.ID, "err", err)
		}
	}
	if p.bus != nil {
		p.bus.Publish(*ev)
	}
	p.total.Add(1)
	p.bumpEPS(time.Now().Unix())
	return nil
}

// Stats summarises pipeline state.
type Stats struct {
	Total       int64     `json:"total"`
	HotCount    int64     `json:"hotCount"`
	WAL         wal.Stats `json:"wal"`
	EPS         int64     `json:"eps"`
	GeneratedAt time.Time `json:"generatedAt"`
}

func (p *Pipeline) Stats() Stats {
	return Stats{
		Total:       p.total.Load(),
		HotCount:    p.hot.Count(),
		WAL:         p.wal.Stats(),
		EPS:         p.currentEPS.Load(),
		GeneratedAt: time.Now().UTC(),
	}
}

// HotStore exposes the underlying read-side for query handlers.
func (p *Pipeline) HotStore() *hot.Store { return p.hot }

// Search exposes the full-text index for query handlers.
func (p *Pipeline) Search() *search.Index { return p.search }

// Bus exposes the live-event broadcaster for live-tail handlers.
func (p *Pipeline) Bus() *events.Bus { return p.bus }

func (p *Pipeline) bumpEPS(nowSec int64) {
	prev := p.lastSecond.Load()
	if prev == nowSec {
		p.lastCount.Add(1)
		p.currentEPS.Store(p.lastCount.Load())
		return
	}
	// New second — store the previous bucket as the published EPS.
	p.currentEPS.Store(p.lastCount.Load())
	p.lastSecond.Store(nowSec)
	p.lastCount.Store(1)
}
