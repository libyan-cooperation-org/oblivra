package oql

import (
	"math"
	"sync"
	"time"
)

type QueryHistoryDB struct {
	mu    sync.RWMutex
	stats map[string]*SelectivityStat
}

type SelectivityStat struct {
	count    int64
	mean     float64
	m2       float64
	lastSeen time.Time
	segments [4]segmentStat
}

type segmentStat struct {
	count int64
	mean  float64
	m2    float64
}

func NewQueryHistoryDB() *QueryHistoryDB {
	return &QueryHistoryDB{stats: make(map[string]*SelectivityStat)}
}

func (h *QueryHistoryDB) Record(key string, observed float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.stats[key]
	if !ok {
		s = &SelectivityStat{}
		h.stats[key] = s
	}
	now := time.Now()
	if age := now.Sub(s.lastSeen); age > 24*time.Hour && s.count > 0 {
		decay := math.Exp(-float64(age) / float64(7*24*time.Hour))
		s.count = int64(float64(s.count) * decay)
		if s.count < 1 {
			s.count = 1
		}
	}
	s.lastSeen = now
	s.count++
	delta := observed - s.mean
	s.mean += delta / float64(s.count)
	delta2 := observed - s.mean
	s.m2 += delta * delta2
	seg := &s.segments[now.Hour()/6]
	seg.count++
	sd := observed - seg.mean
	seg.mean += sd / float64(seg.count)
	sd2 := observed - seg.mean
	seg.m2 += sd * sd2
}

func (h *QueryHistoryDB) Lookup(key string) (float64, float64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s, ok := h.stats[key]
	if !ok || s.count < 3 {
		return -1, 0
	}
	seg := s.segments[time.Now().Hour()/6]
	if seg.count >= 5 {
		v := seg.m2 / float64(seg.count)
		c := 1.0 - math.Min(math.Sqrt(v)/math.Max(seg.mean, 0.001), 1.0)
		return seg.mean, c
	}
	v := s.m2 / float64(s.count)
	c := 1.0 - math.Min(math.Sqrt(v)/math.Max(s.mean, 0.001), 1.0)
	return s.mean, c
}

func (h *QueryHistoryDB) RecordProfile(p *QueryProfile) {
	for _, s := range p.Stages() {
		if s.Name == "WHERE" && s.RowsIn > 100 {
			h.Record(s.PlanNode, float64(s.RowsOut)/float64(s.RowsIn))
		}
	}
}

func (h *QueryHistoryDB) Prune(maxAge time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for k, s := range h.stats {
		if s.lastSeen.Before(cutoff) {
			delete(h.stats, k)
		}
	}
}
