package services

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/storage/hot"
	"github.com/kingknull/oblivra/internal/storage/search"
)

type SearchRequest struct {
	TenantID    string `json:"tenantId,omitempty"`
	Query       string `json:"query,omitempty"` // Bleve query string; empty = match all
	FromUnix    int64  `json:"fromUnix,omitempty"`
	ToUnix      int64  `json:"toUnix,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	NewestFirst bool   `json:"newestFirst,omitempty"`
}

type SearchResponse struct {
	Events []events.Event `json:"events"`
	Total  int            `json:"total"`
	Took   string         `json:"took"`
	Mode   string         `json:"mode"` // "chrono" | "fulltext"
}

type SiemService struct {
	log      *slog.Logger
	pipeline *ingest.Pipeline
}

func NewSiemService(log *slog.Logger, p *ingest.Pipeline) *SiemService {
	return &SiemService{log: log, pipeline: p}
}

func (s *SiemService) ServiceName() string { return "SiemService" }

// Ingest accepts a single event and pushes it through the pipeline.
func (s *SiemService) Ingest(ctx context.Context, ev events.Event) (events.Event, error) {
	if err := s.pipeline.Submit(ctx, &ev); err != nil {
		return events.Event{}, err
	}
	return ev, nil
}

// IngestBatch accepts a slice of events. Returns the count successfully written
// and the first error if any failed.
func (s *SiemService) IngestBatch(ctx context.Context, evs []events.Event) (int, error) {
	if len(evs) == 0 {
		return 0, errors.New("siem: empty batch")
	}
	written := 0
	for i := range evs {
		if err := s.pipeline.Submit(ctx, &evs[i]); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// Search returns events within a window. With Query set it goes through the
// Bleve full-text index; without one it does an ordered scan of the hot store.
// Defaults to the last hour, newest first, capped at 200.
func (s *SiemService) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	start := time.Now()
	to := time.Now().UTC()
	from := to.Add(-1 * time.Hour)
	if req.FromUnix > 0 {
		from = time.Unix(req.FromUnix, 0).UTC()
	}
	if req.ToUnix > 0 {
		to = time.Unix(req.ToUnix, 0).UTC()
	}
	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 200
	}

	if req.Query != "" && s.pipeline.Search() != nil {
		return s.searchFullText(ctx, req, from, to, start)
	}
	return s.searchChrono(ctx, req, from, to, start)
}

func (s *SiemService) searchChrono(ctx context.Context, req SearchRequest, from, to time.Time, start time.Time) (SearchResponse, error) {
	hits, err := s.pipeline.HotStore().Range(ctx, hot.RangeOpts{
		TenantID: req.TenantID,
		From:     from,
		To:       to,
		Limit:    req.Limit,
		Reverse:  req.NewestFirst || (req.FromUnix == 0 && req.ToUnix == 0),
	})
	if err != nil {
		return SearchResponse{}, err
	}
	return SearchResponse{
		Events: hits,
		Total:  len(hits),
		Took:   time.Since(start).String(),
		Mode:   "chrono",
	}, nil
}

func (s *SiemService) searchFullText(_ context.Context, req SearchRequest, from, to time.Time, start time.Time) (SearchResponse, error) {
	idx := s.pipeline.Search()
	tenant := req.TenantID
	if tenant == "" {
		tenant = "default"
	}
	res, err := idx.Query(search.QueryOpts{
		TenantID:    tenant,
		Q:           req.Query,
		From:        from,
		To:          to,
		Limit:       req.Limit,
		NewestFirst: req.NewestFirst,
	})
	if err != nil {
		return SearchResponse{}, err
	}
	ids := make([]string, 0, len(res.Hits))
	for _, h := range res.Hits {
		ids = append(ids, h.ID)
	}
	hydrated, err := s.pipeline.HotStore().Lookup(tenant, ids)
	if err != nil {
		return SearchResponse{}, err
	}
	// Preserve Bleve's ranking order. Without sort metadata Bleve already gives
	// us NewestFirst when requested; otherwise sort by score descending so the
	// highest-relevance hits float up.
	if req.NewestFirst {
		sort.Slice(hydrated, func(i, j int) bool {
			return hydrated[i].Timestamp.After(hydrated[j].Timestamp)
		})
	}
	return SearchResponse{
		Events: hydrated,
		Total:  int(res.Total),
		Took:   time.Since(start).String(),
		Mode:   "fulltext",
	}, nil
}

// Stats returns pipeline counters for the dashboard.
func (s *SiemService) Stats() ingest.Stats {
	return s.pipeline.Stats()
}
