package services

import (
	"context"
	"sort"
	"time"

	"github.com/kingknull/oblivra/internal/storage/hot"
)

// TimelineService merges every observable signal we have for a host into a
// single chronological stream so analysts can see what happened around a
// suspicious moment.
type TimelineService struct {
	hot    *hot.Store
	alerts *AlertService
	foren  *ForensicsService
}

func NewTimelineService(h *hot.Store, alerts *AlertService, foren *ForensicsService) *TimelineService {
	return &TimelineService{hot: h, alerts: alerts, foren: foren}
}

func (s *TimelineService) ServiceName() string { return "TimelineService" }

type TimelineEntry struct {
	Kind      string    `json:"kind"` // "event" | "alert" | "gap" | "evidence"
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity,omitempty"`
	Title     string    `json:"title"`
	Detail    string    `json:"detail,omitempty"`
	RefID     string    `json:"refId,omitempty"`
}

type TimelineRequest struct {
	TenantID string
	HostID   string
	From     time.Time
	To       time.Time
	Limit    int
}

func (s *TimelineService) Build(ctx context.Context, req TimelineRequest) ([]TimelineEntry, error) {
	if req.To.IsZero() {
		req.To = time.Now().UTC()
	}
	if req.From.IsZero() {
		req.From = req.To.Add(-24 * time.Hour)
	}
	if req.Limit <= 0 {
		req.Limit = 200
	}

	out := make([]TimelineEntry, 0, req.Limit)

	// Events from the hot store, filtered to host.
	evs, err := s.hot.Range(ctx, hot.RangeOpts{
		TenantID: req.TenantID,
		From:     req.From,
		To:       req.To,
		Limit:    1000,
	})
	if err == nil {
		for _, e := range evs {
			if req.HostID != "" && e.HostID != req.HostID {
				continue
			}
			out = append(out, TimelineEntry{
				Kind:      "event",
				Timestamp: e.Timestamp,
				Severity:  string(e.Severity),
				Title:     orFallback(e.EventType, "event"),
				Detail:    e.Message,
				RefID:     e.ID,
			})
		}
	}

	// Alerts.
	if s.alerts != nil {
		for _, a := range s.alerts.Recent(500) {
			if req.HostID != "" && a.HostID != req.HostID {
				continue
			}
			if a.Triggered.Before(req.From) || a.Triggered.After(req.To) {
				continue
			}
			out = append(out, TimelineEntry{
				Kind:      "alert",
				Timestamp: a.Triggered,
				Severity:  string(a.Severity),
				Title:     a.RuleName,
				Detail:    a.Message,
				RefID:     a.ID,
			})
		}
	}

	// Log gaps.
	if s.foren != nil {
		for _, g := range s.foren.Gaps() {
			if req.HostID != "" && g.HostID != req.HostID {
				continue
			}
			if g.EndedAt.Before(req.From) || g.StartedAt.After(req.To) {
				continue
			}
			out = append(out, TimelineEntry{
				Kind:      "gap",
				Timestamp: g.EndedAt,
				Severity:  "warning",
				Title:     "Telemetry gap",
				Detail:    "host " + g.HostID + " silent for " + g.Duration,
			})
		}
		// Sealed evidence.
		for _, ev := range s.foren.List() {
			if req.HostID != "" && ev.HostID != req.HostID {
				continue
			}
			if ev.SealedAt.Before(req.From) || ev.SealedAt.After(req.To) {
				continue
			}
			out = append(out, TimelineEntry{
				Kind:      "evidence",
				Timestamp: ev.SealedAt,
				Title:     ev.Title,
				Detail:    "sha256 " + ev.Hash,
				RefID:     ev.ID,
			})
		}
	}

	// Deterministic ordering: timestamp DESC primary, but break ties on
	// (kind, refId) so two events that race the same nanosecond don't shuffle
	// across renders. The clock-drift correction is happening upstream in
	// Trust.Engine + Tamper.Service — by the time entries reach this sort,
	// suspect timestamps are already labeled, so we trust the value here.
	sort.Slice(out, func(i, j int) bool {
		ti, tj := out[i].Timestamp, out[j].Timestamp
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].RefID < out[j].RefID
	})
	if len(out) > req.Limit {
		out = out[:req.Limit]
	}
	return out, nil
}

// PivotWindow returns the timeline entries within ±delta of a pivot moment,
// scoped to a host. The Beta-1 task tracker calls this the "pivot engine":
// given an event, jump to "what else was happening within ±15 minutes".
func (s *TimelineService) PivotWindow(ctx context.Context, tenantID, hostID string, pivot time.Time, delta time.Duration, limit int) ([]TimelineEntry, error) {
	if pivot.IsZero() {
		pivot = time.Now().UTC()
	}
	if delta <= 0 {
		delta = 15 * time.Minute
	}
	if limit <= 0 {
		limit = 200
	}
	return s.Build(ctx, TimelineRequest{
		TenantID: tenantID, HostID: hostID,
		From: pivot.Add(-delta), To: pivot.Add(delta),
		Limit: limit,
	})
}

func orFallback(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
