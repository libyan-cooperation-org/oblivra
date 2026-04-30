// Package quality scores log-source reliability and coverage. Per
// `(host, source)` pair we track:
//
//   - lines parsed cleanly vs lines that fell back to "plain"
//   - the gap-density (long silences in the per-host stream)
//   - the ingestion delay (receivedAt - timestamp)
//
// These feed two analyst-facing surfaces: a "noisy / unreliable source"
// list, and a "coverage" view that shows which hosts the platform has heard
// from in the last hour, day, week.
package quality

import (
	"sort"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// SourceProfile is the per-(host, source) reliability snapshot.
type SourceProfile struct {
	Host         string        `json:"host"`
	Source       string        `json:"source"`
	Total        int64         `json:"total"`
	Parsed       int64         `json:"parsed"`
	UnparsedRate float64       `json:"unparsedRate"`
	GapsObserved int64         `json:"gapsObserved"`
	AvgDelayMS   int64         `json:"avgDelayMs"`
	LastSeen     time.Time     `json:"lastSeen"`
	FirstSeen    time.Time     `json:"firstSeen"`
}

// Coverage gives a per-host roll-up of recent activity.
type Coverage struct {
	Host         string    `json:"host"`
	LastSeen     time.Time `json:"lastSeen"`
	EventsLastHr int64     `json:"eventsLastHour"`
	EventsLastDay int64    `json:"eventsLastDay"`
	Sources      []string  `json:"sources"`
}

// Engine accumulates the metrics and is safe for concurrent calls.
type Engine struct {
	mu       sync.RWMutex
	profiles map[string]*SourceProfile
	hostBuckets map[string]*hostStats
	gapThreshold time.Duration
}

type hostStats struct {
	host   string
	last   time.Time
	first  time.Time
	bucketHr  []time.Time
	bucketDay []time.Time
	sources   map[string]struct{}
}

func New() *Engine {
	return &Engine{
		profiles:     map[string]*SourceProfile{},
		hostBuckets:  map[string]*hostStats{},
		gapThreshold: 5 * time.Minute,
	}
}

// Observe is called per-event from the bus fan-out.
func (e *Engine) Observe(ev events.Event) {
	if ev.HostID == "" {
		return
	}
	now := time.Now().UTC()

	key := ev.HostID + "|" + string(ev.Source)
	e.mu.Lock()
	defer e.mu.Unlock()

	p, ok := e.profiles[key]
	if !ok {
		p = &SourceProfile{Host: ev.HostID, Source: string(ev.Source), FirstSeen: ev.Timestamp}
		e.profiles[key] = p
	}
	p.Total++
	if ev.EventType != "plain" && ev.EventType != "" {
		p.Parsed++
	}
	delay := ev.ReceivedAt.Sub(ev.Timestamp)
	if delay < 0 {
		delay = 0
	}
	// Running average update — n*avg + new / (n+1)
	p.AvgDelayMS = (p.AvgDelayMS*int64(p.Total-1) + delay.Milliseconds()) / int64(p.Total)
	if !p.LastSeen.IsZero() && ev.Timestamp.Sub(p.LastSeen) > e.gapThreshold {
		p.GapsObserved++
	}
	p.LastSeen = ev.Timestamp
	if p.Total == 0 {
		p.UnparsedRate = 0
	} else {
		p.UnparsedRate = float64(p.Total-p.Parsed) / float64(p.Total)
	}

	// Host-level coverage roll-up.
	hs, ok := e.hostBuckets[ev.HostID]
	if !ok {
		hs = &hostStats{host: ev.HostID, first: ev.Timestamp, sources: map[string]struct{}{}}
		e.hostBuckets[ev.HostID] = hs
	}
	hs.last = ev.Timestamp
	hs.sources[string(ev.Source)] = struct{}{}
	hs.bucketHr = append(hs.bucketHr, ev.Timestamp)
	hs.bucketDay = append(hs.bucketDay, ev.Timestamp)
	hs.bucketHr = trimBefore(hs.bucketHr, now.Add(-time.Hour))
	hs.bucketDay = trimBefore(hs.bucketDay, now.Add(-24*time.Hour))
}

func trimBefore(xs []time.Time, cut time.Time) []time.Time {
	for i, t := range xs {
		if !t.Before(cut) {
			return xs[i:]
		}
	}
	return nil
}

// Profiles returns per-source reliability snapshots, worst-first
// (highest unparsed rate / longest delay).
func (e *Engine) Profiles() []SourceProfile {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]SourceProfile, 0, len(e.profiles))
	for _, p := range e.profiles {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UnparsedRate != out[j].UnparsedRate {
			return out[i].UnparsedRate > out[j].UnparsedRate
		}
		return out[i].AvgDelayMS > out[j].AvgDelayMS
	})
	return out
}

// Coverage returns one entry per known host with bucketed recent activity.
func (e *Engine) Coverage() []Coverage {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Coverage, 0, len(e.hostBuckets))
	for _, hs := range e.hostBuckets {
		c := Coverage{
			Host:          hs.host,
			LastSeen:      hs.last,
			EventsLastHr:  int64(len(hs.bucketHr)),
			EventsLastDay: int64(len(hs.bucketDay)),
		}
		for s := range hs.sources {
			c.Sources = append(c.Sources, s)
		}
		sort.Strings(c.Sources)
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	return out
}
