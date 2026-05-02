package services

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// CategoriesService rolls events up by sourceType so the operator UI
// can show "what kinds of logs are flowing through this OBLIVRA". It's
// a read-side aggregate — the ingest path stays untouched and we
// observe the same event bus the other processors consume.
//
// We track:
//   - lifetime count per sourceType
//   - last-seen timestamp per sourceType
//   - top-5 hosts per sourceType (so a category like "linux:auth"
//     surfaces "web-01 → 3,200 events, db-01 → 1,800 events")
//
// Memory is bounded: per-category top-N is capped, and the category
// map only grows when a new sourceType appears (typically <100 in any
// real deployment).

type CategoryStat struct {
	SourceType string         `json:"sourceType"`
	Count      int64          `json:"count"`
	LastSeen   time.Time      `json:"lastSeen"`
	TopHosts   []CategoryHost `json:"topHosts,omitempty"`
}

type CategoryHost struct {
	Host  string `json:"host"`
	Count int64  `json:"count"`
}

type CategoriesService struct {
	mu     sync.RWMutex
	byType map[string]*categoryAgg
}

type categoryAgg struct {
	count    int64
	lastSeen time.Time
	hostCt   map[string]int64
}

func NewCategoriesService() *CategoriesService {
	return &CategoriesService{byType: map[string]*categoryAgg{}}
}

func (c *CategoriesService) ServiceName() string { return "CategoriesService" }

// Observe is wired into the platform processor fan-out so every event
// updates the rollup. Cheap: one map lookup + a couple of counters.
func (c *CategoriesService) Observe(_ context.Context, ev events.Event) {
	st := categorize(ev)
	if st == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	agg, ok := c.byType[st]
	if !ok {
		agg = &categoryAgg{hostCt: map[string]int64{}}
		c.byType[st] = agg
	}
	agg.count++
	if ev.ReceivedAt.After(agg.lastSeen) {
		agg.lastSeen = ev.ReceivedAt
	} else if agg.lastSeen.IsZero() {
		agg.lastSeen = time.Now().UTC()
	}
	if ev.HostID != "" {
		agg.hostCt[ev.HostID]++
		// Cap host map at 64 — drop the smallest contributor when over.
		if len(agg.hostCt) > 64 {
			var minHost string
			var minCount int64 = -1
			for h, n := range agg.hostCt {
				if minCount == -1 || n < minCount {
					minHost = h
					minCount = n
				}
			}
			if minHost != "" {
				delete(agg.hostCt, minHost)
			}
		}
	}
}

// List returns one row per sourceType, sorted by total count descending.
func (c *CategoriesService) List() []CategoryStat {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]CategoryStat, 0, len(c.byType))
	for st, agg := range c.byType {
		row := CategoryStat{
			SourceType: st,
			Count:      agg.count,
			LastSeen:   agg.lastSeen,
		}
		// Top-5 hosts per category.
		hosts := make([]CategoryHost, 0, len(agg.hostCt))
		for h, n := range agg.hostCt {
			hosts = append(hosts, CategoryHost{Host: h, Count: n})
		}
		sort.Slice(hosts, func(i, j int) bool { return hosts[i].Count > hosts[j].Count })
		if len(hosts) > 5 {
			hosts = hosts[:5]
		}
		row.TopHosts = hosts
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

// categorize picks the field that best classifies an event for the
// operator UI. SourceType is preferred (it's what the agent sets
// explicitly); EventType is the fallback; finally Source.
func categorize(ev events.Event) string {
	if ev.Fields != nil {
		if st, ok := ev.Fields["sourceType"]; ok && st != "" {
			return st
		}
	}
	if ev.EventType != "" {
		return ev.EventType
	}
	if ev.Source != "" {
		return string(ev.Source)
	}
	return ""
}
