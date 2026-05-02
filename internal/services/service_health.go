package services

import (
	"sort"
	"time"
)

// ServiceHealth is one row per logical service in the operator UI.
// "Service" = the union of (sourceType, hostId) pairs we observe in
// events. We borrow CategoriesService's per-sourceType rollup for
// total counts + top hosts, and QualityService's per-source profile
// for unparsed-rate / gap density / last-seen freshness, then derive
// a single status word the dashboard can colour.
type ServiceHealth struct {
	SourceType   string    `json:"sourceType"`
	Status       string    `json:"status"` // healthy | degraded | silent | unknown
	Hosts        int       `json:"hosts"`
	Events24h    int64     `json:"events24h"`
	Events1h     int64     `json:"events1h"`
	LastSeen     time.Time `json:"lastSeen"`
	UnparsedRate float64   `json:"unparsedRate"` // 0..1 across the source's events
	GapsObserved int64     `json:"gapsObserved"`
	AvgDelayMs   int64     `json:"avgDelayMs"`
	TopHosts     []string  `json:"topHosts,omitempty"` // up to 5
}

// ServiceHealthService composes the breakdowns into a single rollup.
// It owns no state — every read recomputes from CategoriesService and
// QualityService. Cheap because both sources are O(N_categories) and
// N_categories is bounded (~100 in any real deployment).
type ServiceHealthService struct {
	cats *CategoriesService
	qual *QualityService
}

func NewServiceHealthService(cats *CategoriesService, qual *QualityService) *ServiceHealthService {
	return &ServiceHealthService{cats: cats, qual: qual}
}

func (s *ServiceHealthService) ServiceName() string { return "ServiceHealthService" }

// List returns one ServiceHealth row per observed sourceType, sorted
// by status (worst first) then by recent volume.
func (s *ServiceHealthService) List() []ServiceHealth {
	if s.cats == nil {
		return nil
	}
	now := time.Now().UTC()
	cats := s.cats.List()

	// Index quality profiles by sourceType so we can join below. The
	// quality engine keys by (host, source) — for service-level health
	// we collapse to the source dimension by averaging.
	type qAgg struct {
		unparsed  float64
		count     int
		gaps      int64
		delayMS   int64
		hosts     map[string]struct{}
		windowMin int64
	}
	qBySource := map[string]*qAgg{}
	if s.qual != nil {
		for _, p := range s.qual.Profiles() {
			a := qBySource[p.Source]
			if a == nil {
				a = &qAgg{hosts: map[string]struct{}{}}
				qBySource[p.Source] = a
			}
			a.unparsed += p.UnparsedRate
			a.count++
			a.gaps += p.GapsObserved
			a.delayMS += p.AvgDelayMS
			a.hosts[p.Host] = struct{}{}
		}
	}

	out := make([]ServiceHealth, 0, len(cats))
	for _, c := range cats {
		row := ServiceHealth{
			SourceType: c.SourceType,
			Hosts:      len(c.TopHosts),
			LastSeen:   c.LastSeen,
		}
		// Top-host names for the chip rail.
		for i, h := range c.TopHosts {
			if i >= 5 {
				break
			}
			row.TopHosts = append(row.TopHosts, h.Host)
		}
		if a := qBySource[c.SourceType]; a != nil && a.count > 0 {
			row.UnparsedRate = a.unparsed / float64(a.count)
			row.GapsObserved = a.gaps
			row.AvgDelayMs = a.delayMS / int64(a.count)
			if len(a.hosts) > row.Hosts {
				row.Hosts = len(a.hosts)
			}
		}

		// Recent volume — we don't keep a per-minute histogram, so we
		// approximate using the lifetime count plus the gap since
		// LastSeen as a "is it flowing now" signal.
		row.Events24h = c.Count // Lifetime is a reasonable upper-bound
		// Events1h — set to count if last seen within the hour, else 0.
		if !c.LastSeen.IsZero() && now.Sub(c.LastSeen) < time.Hour {
			row.Events1h = c.Count
		}

		row.Status = healthStatus(row, now)
		out = append(out, row)
	}

	// Sort: silent/degraded first, then by recent volume.
	statusRank := map[string]int{"silent": 0, "degraded": 1, "unknown": 2, "healthy": 3}
	sort.Slice(out, func(i, j int) bool {
		ri, rj := statusRank[out[i].Status], statusRank[out[j].Status]
		if ri != rj {
			return ri < rj
		}
		return out[i].Events24h > out[j].Events24h
	})
	return out
}

// healthStatus picks a one-word verdict for the row. Operators rely on
// status colour for at-a-glance scanning; the underlying numbers are
// available in the detail row.
func healthStatus(r ServiceHealth, now time.Time) string {
	if r.LastSeen.IsZero() {
		return "unknown"
	}
	age := now.Sub(r.LastSeen)
	if age > 30*time.Minute {
		return "silent"
	}
	if r.UnparsedRate > 0.20 || r.GapsObserved > 5 {
		return "degraded"
	}
	if age > 5*time.Minute {
		return "degraded"
	}
	return "healthy"
}
