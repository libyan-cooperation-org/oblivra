package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// SavedSearch is a stored Bleve / OQL query an operator runs on demand
// or on a schedule. When scheduled, the runner fires the search every
// IntervalMinutes and surfaces a one-line summary as an alert if the
// result count meets/exceeds AlertOnAtLeast.
//
// Designed to be cheap: queries against the existing search path; no
// new storage layer. Persisted to <dataDir>/saved_searches.json.
type SavedSearch struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`              // Bleve query string (OQL also accepted via QueryKind)
	QueryKind string    `json:"queryKind"`          // "bleve" (default) | "oql"
	TenantID  string    `json:"tenantId,omitempty"` // optional scope
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy,omitempty"`

	// Scheduling — IntervalMinutes=0 means "manual only".
	IntervalMinutes int    `json:"intervalMinutes,omitempty"`
	AlertOnAtLeast  int    `json:"alertOnAtLeast,omitempty"` // raise alert if hit count >=
	Severity        string `json:"severity,omitempty"`       // alert severity (default high)

	// Log-to-metric — when EmitMetric is true, every scheduled run also
	// pushes a metric event into the pipeline named MetricName (or
	// "saved_search_<id>_hits" if blank). Lets operators turn any log
	// pattern into a queryable counter / time series — the same trick
	// Splunk's stats command does, but native to OBLIVRA.
	EmitMetric bool   `json:"emitMetric,omitempty"`
	MetricName string `json:"metricName,omitempty"`

	// Last-run telemetry — overwritten by each scheduled run.
	LastRunAt    *time.Time `json:"lastRunAt,omitempty"`
	LastHitCount int        `json:"lastHitCount,omitempty"`
	LastError    string     `json:"lastError,omitempty"`
}

// SavedSearchRunner is the abstraction the scheduler uses to actually
// execute a saved search. We accept any function so the wiring stays
// out of internal/services — platform.New plugs in SiemService.Search.
type SavedSearchRunner func(ctx context.Context, q SavedSearch) (hits int, err error)

// MetricEmitter is the abstraction the scheduler uses to push a
// derived metric event into the pipeline. Plugged in by platform.New.
type MetricEmitter func(ctx context.Context, name string, value float64, labels map[string]string)

type SavedSearchService struct {
	log    *slog.Logger
	alerts *AlertService

	mu      sync.RWMutex
	saved   map[string]*SavedSearch
	runner  SavedSearchRunner
	emit    MetricEmitter
}

func NewSavedSearchService(log *slog.Logger, alerts *AlertService) *SavedSearchService {
	return &SavedSearchService{log: log, alerts: alerts, saved: map[string]*SavedSearch{}}
}

func (s *SavedSearchService) ServiceName() string { return "SavedSearchService" }

// AttachRunner wires the actual search executor. Called once from the
// platform stack after SiemService is constructed.
func (s *SavedSearchService) AttachRunner(r SavedSearchRunner) {
	s.mu.Lock()
	s.runner = r
	s.mu.Unlock()
}

// AttachMetricEmitter plugs in the function that pushes a metric event
// into the pipeline when EmitMetric is true on a saved search. Optional
// — without it, EmitMetric is a no-op.
func (s *SavedSearchService) AttachMetricEmitter(e MetricEmitter) {
	s.mu.Lock()
	s.emit = e
	s.mu.Unlock()
}

func (s *SavedSearchService) List() []SavedSearch {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SavedSearch, 0, len(s.saved))
	for _, q := range s.saved {
		out = append(out, *q)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *SavedSearchService) Get(id string) (SavedSearch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.saved[id]
	if !ok {
		return SavedSearch{}, false
	}
	return *q, true
}

func (s *SavedSearchService) Save(q SavedSearch) (SavedSearch, error) {
	if q.Name == "" {
		return q, errors.New("name required")
	}
	if q.Query == "" {
		return q, errors.New("query required")
	}
	if q.QueryKind == "" {
		q.QueryKind = "bleve"
	}
	if q.IntervalMinutes < 0 {
		return q, errors.New("intervalMinutes must be >= 0")
	}
	if q.IntervalMinutes > 0 && q.IntervalMinutes < 5 {
		return q, errors.New("intervalMinutes must be 0 (manual) or >= 5")
	}
	if q.ID == "" {
		q.ID = randomID(8)
		q.CreatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	s.saved[q.ID] = &q
	s.mu.Unlock()
	return q, nil
}

func (s *SavedSearchService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.saved[id]; !ok {
		return errors.New("not found")
	}
	delete(s.saved, id)
	return nil
}

// Run executes a saved search now, regardless of schedule. Returns
// (hits, error). Used by the "Run" button in the UI and by Tick().
func (s *SavedSearchService) Run(ctx context.Context, id string) (int, error) {
	s.mu.RLock()
	q, ok := s.saved[id]
	runner := s.runner
	s.mu.RUnlock()
	if !ok {
		return 0, errors.New("not found")
	}
	if runner == nil {
		return 0, errors.New("runner not attached")
	}
	now := time.Now().UTC()
	hits, err := runner(ctx, *q)
	s.mu.Lock()
	q.LastRunAt = &now
	q.LastHitCount = hits
	if err != nil {
		q.LastError = err.Error()
	} else {
		q.LastError = ""
	}
	s.mu.Unlock()

	// Log-to-metric emission: every successful run produces one metric
	// event named after the saved search. Operators can then graph the
	// hit-count time series in Grafana (via the platform's /metrics
	// scrape) or query it directly via search.
	if err == nil && q.EmitMetric {
		s.mu.RLock()
		emit := s.emit
		s.mu.RUnlock()
		if emit != nil {
			name := q.MetricName
			if name == "" {
				name = "saved_search_" + q.ID + "_hits"
			}
			labels := map[string]string{
				"savedSearchId":   q.ID,
				"savedSearchName": q.Name,
			}
			if q.TenantID != "" {
				labels["tenantId"] = q.TenantID
			}
			emit(ctx, name, float64(hits), labels)
		}
	}

	// If alerting is configured and the hit count meets the threshold,
	// raise an alert (which fans out to webhooks + email channels via
	// the alert subscribe loop).
	if err == nil && q.AlertOnAtLeast > 0 && hits >= q.AlertOnAtLeast {
		sev := AlertSeverityHigh
		switch q.Severity {
		case "low":
			sev = AlertSeverityLow
		case "medium":
			sev = AlertSeverityMedium
		case "critical":
			sev = AlertSeverityCritical
		}
		if s.alerts != nil {
			s.alerts.Raise(ctx, Alert{
				TenantID: q.TenantID,
				RuleID:   "saved-search:" + q.ID,
				RuleName: "Saved search: " + q.Name,
				Severity: sev,
				Message:  fmt.Sprintf("saved search %s returned %d hit(s) (threshold %d)", q.Name, hits, q.AlertOnAtLeast),
			})
		}
	}
	return hits, err
}

// Tick is called by the scheduler every minute. Walks every saved
// search and runs the ones whose IntervalMinutes have elapsed since
// the last run. Cheap when nothing's due.
func (s *SavedSearchService) Tick(ctx context.Context) {
	s.mu.RLock()
	due := make([]string, 0)
	now := time.Now().UTC()
	for id, q := range s.saved {
		if q.IntervalMinutes <= 0 {
			continue
		}
		if q.LastRunAt == nil || now.Sub(*q.LastRunAt) >= time.Duration(q.IntervalMinutes)*time.Minute {
			due = append(due, id)
		}
	}
	s.mu.RUnlock()

	for _, id := range due {
		if _, err := s.Run(ctx, id); err != nil {
			s.log.Warn("saved-search run", "id", id, "err", err)
		}
	}
}

