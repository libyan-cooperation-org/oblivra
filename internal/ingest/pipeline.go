// Package ingest is the entry point every event must pass through. It writes
// to the WAL first (durability), then to the hot store (queryability).
package ingest

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/storage/hot"
	"github.com/kingknull/oblivra/internal/storage/search"
	"github.com/kingknull/oblivra/internal/wal"
)

// latencyBuf is a tiny ring of durations used to compute p50/p95/p99 without
// pulling in a histogram library. Caller holds the parent mutex.
type latencyBuf struct {
	d   []time.Duration
	cap int
	idx int
	n   int
}

func newLatencyBuf(cap int) *latencyBuf { return &latencyBuf{d: make([]time.Duration, cap), cap: cap} }

func (b *latencyBuf) add(d time.Duration) {
	b.d[b.idx] = d
	b.idx = (b.idx + 1) % b.cap
	if b.n < b.cap {
		b.n++
	}
}

func (b *latencyBuf) percentiles() (p50, p95, p99 time.Duration) {
	if b.n == 0 {
		return
	}
	cp := make([]time.Duration, b.n)
	copy(cp, b.d[:b.n])
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	at := func(p int) time.Duration {
		i := (p * len(cp)) / 100
		if i >= len(cp) {
			i = len(cp) - 1
		}
		return cp[i]
	}
	return at(50), at(95), at(99)
}

type Pipeline struct {
	log    *slog.Logger
	wal    *wal.WAL
	hot    *hot.Store
	search *search.Index
	bus    *events.Bus
	total  atomic.Int64

	// EPS sampling — naive 1s window.
	lastSecond atomic.Int64
	lastCount  atomic.Int64
	currentEPS atomic.Int64

	// Stage latencies (rolling buffer of last 1024 measurements per stage).
	latencyBufMu sync.Mutex
	walLat       *latencyBuf
	hotLat       *latencyBuf
	indexLat     *latencyBuf
	totalLat     *latencyBuf
}

func New(log *slog.Logger, w *wal.WAL, h *hot.Store, idx *search.Index, bus *events.Bus) *Pipeline {
	return &Pipeline{
		log: log, wal: w, hot: h, search: idx, bus: bus,
		walLat:   newLatencyBuf(1024),
		hotLat:   newLatencyBuf(1024),
		indexLat: newLatencyBuf(1024),
		totalLat: newLatencyBuf(1024),
	}
}

// Submit accepts an event, validates it, and persists it through WAL → hot store.
func (p *Pipeline) Submit(ctx context.Context, ev *events.Event) error {
	if ev == nil {
		return errors.New("ingest: nil event")
	}
	if err := ev.Validate(); err != nil {
		return err
	}
	start := time.Now()

	t0 := time.Now()
	if err := p.wal.Append(ev); err != nil {
		return err
	}
	walDur := time.Since(t0)

	t1 := time.Now()
	if err := p.hot.Put(ev); err != nil {
		p.log.Error("hot store put failed; event durably in WAL", "id", ev.ID, "err", err)
		return err
	}
	hotDur := time.Since(t1)

	var indexDur time.Duration
	if p.search != nil {
		t2 := time.Now()
		if err := p.search.Index(ev); err != nil {
			p.log.Warn("search index failed", "id", ev.ID, "err", err)
		}
		indexDur = time.Since(t2)
	}
	if p.bus != nil {
		p.bus.Publish(*ev)
	}

	totalDur := time.Since(start)
	p.latencyBufMu.Lock()
	p.walLat.add(walDur)
	p.hotLat.add(hotDur)
	p.indexLat.add(indexDur)
	p.totalLat.add(totalDur)
	p.latencyBufMu.Unlock()

	p.total.Add(1)
	p.bumpEPS(time.Now().Unix())
	return nil
}

// LatencyStats holds rolling p50/p95/p99 per pipeline stage.
type LatencyStats struct {
	WAL   StagePercentiles `json:"wal"`
	Hot   StagePercentiles `json:"hot"`
	Index StagePercentiles `json:"index"`
	Total StagePercentiles `json:"total"`
}

type StagePercentiles struct {
	P50 string `json:"p50"`
	P95 string `json:"p95"`
	P99 string `json:"p99"`
}

// Stats summarises pipeline state.
type Stats struct {
	Total       int64        `json:"total"`
	HotCount    int64        `json:"hotCount"`
	WAL         wal.Stats    `json:"wal"`
	EPS         int64        `json:"eps"`
	Latency     LatencyStats `json:"latency"`
	GeneratedAt time.Time    `json:"generatedAt"`
}

func (p *Pipeline) Stats() Stats {
	p.latencyBufMu.Lock()
	wp50, wp95, wp99 := p.walLat.percentiles()
	hp50, hp95, hp99 := p.hotLat.percentiles()
	ip50, ip95, ip99 := p.indexLat.percentiles()
	tp50, tp95, tp99 := p.totalLat.percentiles()
	p.latencyBufMu.Unlock()
	return Stats{
		Total:    p.total.Load(),
		HotCount: p.hot.Count(),
		WAL:      p.wal.Stats(),
		EPS:      p.currentEPS.Load(),
		Latency: LatencyStats{
			WAL:   StagePercentiles{P50: wp50.String(), P95: wp95.String(), P99: wp99.String()},
			Hot:   StagePercentiles{P50: hp50.String(), P95: hp95.String(), P99: hp99.String()},
			Index: StagePercentiles{P50: ip50.String(), P95: ip95.String(), P99: ip99.String()},
			Total: StagePercentiles{P50: tp50.String(), P95: tp95.String(), P99: tp99.String()},
		},
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
