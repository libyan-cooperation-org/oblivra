package services

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/oql"
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

	// Aggregation output (OQL v2 stats/top/timechart). When set, Events is
	// empty — the aggregation replaces the raw result set.
	Aggregation *AggResult `json:"aggregation,omitempty"`
}

// AggResult carries the output of a terminal OQL aggregation stage.
type AggResult struct {
	Kind    string      `json:"kind"` // stats | top | timechart
	Fn      string      `json:"fn"`   // count | sum | avg | min | max | dc
	Field   string      `json:"field,omitempty"`
	By      string      `json:"by,omitempty"`
	Span    string      `json:"span,omitempty"` // timechart bucket width
	Buckets []AggBucket `json:"buckets"`
}

// AggBucket is one aggregation group. For timechart, Key is the RFC3339
// bucket start and Series carries per-by-field counts; otherwise Value is
// the aggregate for the group Key.
type AggBucket struct {
	Key    string             `json:"key"`
	Value  float64            `json:"value"`
	Series map[string]float64 `json:"series,omitempty"`
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

// SearchOQL parses an OQL pipe-syntax query, runs the expression through the
// existing search path, and applies stage filters / sort / limit / tail in Go.
func (s *SiemService) SearchOQL(ctx context.Context, raw string, tenantID string, fromUnix, toUnix int64) (SearchResponse, error) {
	plan, err := oql.Parse(raw)
	if err != nil {
		return SearchResponse{}, err
	}
	req := SearchRequest{
		TenantID:    tenantID,
		Query:       plan.Expr,
		FromUnix:    fromUnix,
		ToUnix:      toUnix,
		Limit:       plan.Limit,
		NewestFirst: plan.SortDesc && plan.SortField == "timestamp",
	}
	if req.Limit == 0 {
		req.Limit = 500 // pre-filter generously; we'll trim after where/sort/tail
	}
	if plan.Agg != nil {
		// Aggregations reduce over the full window, not one page — scan wide.
		req.Limit = 10000
	}
	resp, err := s.Search(ctx, req)
	if err != nil {
		return resp, err
	}
	out := resp.Events

	for _, f := range plan.Filters {
		out = applyWhere(out, f)
	}
	if plan.Agg != nil {
		agg := applyAgg(out, plan.Agg)
		return SearchResponse{
			Events:      []events.Event{}, // keep JSON "events" an array, not null
			Total:       len(out),
			Took:        resp.Took,
			Mode:        "oql/" + resp.Mode,
			Aggregation: agg,
		}, nil
	}
	if plan.SortField != "" {
		applySort(out, plan.SortField, plan.SortDesc)
	}
	if plan.Tail > 0 && plan.Tail < len(out) {
		out = out[len(out)-plan.Tail:]
	}
	if plan.Limit > 0 && plan.Limit < len(out) {
		out = out[:plan.Limit]
	}
	return SearchResponse{
		Events: out,
		Total:  len(out),
		Took:   resp.Took,
		Mode:   "oql/" + resp.Mode,
	}, nil
}

func applyWhere(in []events.Event, f oql.Filter) []events.Event {
	out := in[:0]
	want := strings.ToLower(f.Value)
	for _, ev := range in {
		val := fieldValue(ev, f.Field)
		if strings.Contains(strings.ToLower(val), want) {
			out = append(out, ev)
		}
	}
	return out
}

func applySort(out []events.Event, field string, desc bool) {
	sort.SliceStable(out, func(i, j int) bool {
		a := fieldValue(out[i], field)
		b := fieldValue(out[j], field)
		if field == "timestamp" {
			ta, tb := out[i].Timestamp, out[j].Timestamp
			if desc {
				return ta.After(tb)
			}
			return ta.Before(tb)
		}
		if desc {
			return a > b
		}
		return a < b
	})
}

func fieldValue(ev events.Event, field string) string {
	switch field {
	case "id":
		return ev.ID
	case "tenantId":
		return ev.TenantID
	case "host", "hostId":
		return ev.HostID
	case "source":
		return string(ev.Source)
	case "severity":
		return string(ev.Severity)
	case "eventType":
		return ev.EventType
	case "message":
		return ev.Message
	case "raw":
		return ev.Raw
	case "timestamp":
		return ev.Timestamp.Format(time.RFC3339Nano)
	default:
		if v, ok := ev.Fields[field]; ok {
			return v
		}
		return ""
	}
}

// applyAgg reduces a filtered event set through a terminal OQL aggregation.
func applyAgg(in []events.Event, a *oql.Agg) *AggResult {
	res := &AggResult{Kind: a.Kind, Fn: string(a.Fn), Field: a.Field, By: a.By}

	switch a.Kind {
	case "timechart":
		res.Span = a.Span.String()
		res.Buckets = timechartBuckets(in, a)
		return res

	default: // stats | top — group by a.By ("" = single bucket)
		groups := map[string][]events.Event{}
		var order []string
		for _, ev := range in {
			key := ""
			if a.By != "" {
				key = fieldValue(ev, a.By)
				if key == "" {
					key = "(missing)"
				}
			}
			if _, seen := groups[key]; !seen {
				order = append(order, key)
			}
			groups[key] = append(groups[key], ev)
		}
		for _, key := range order {
			res.Buckets = append(res.Buckets, AggBucket{Key: key, Value: reduce(groups[key], a)})
		}
		// stats and top both read best sorted by value desc; ties by key for
		// deterministic output (forensic reports must render identically).
		sort.SliceStable(res.Buckets, func(i, j int) bool {
			if res.Buckets[i].Value != res.Buckets[j].Value {
				return res.Buckets[i].Value > res.Buckets[j].Value
			}
			return res.Buckets[i].Key < res.Buckets[j].Key
		})
		if a.Kind == "top" && a.TopN > 0 && a.TopN < len(res.Buckets) {
			res.Buckets = res.Buckets[:a.TopN]
		}
		return res
	}
}

// reduce computes one aggregate value over a group.
func reduce(evs []events.Event, a *oql.Agg) float64 {
	switch a.Fn {
	case oql.AggCount:
		return float64(len(evs))
	case oql.AggDC:
		distinct := map[string]struct{}{}
		for _, ev := range evs {
			if v := fieldValue(ev, a.Field); v != "" {
				distinct[v] = struct{}{}
			}
		}
		return float64(len(distinct))
	default: // sum / avg / min / max over numeric field values
		var sum, minV, maxV float64
		n := 0
		for _, ev := range evs {
			v, err := strconv.ParseFloat(fieldValue(ev, a.Field), 64)
			if err != nil {
				continue // non-numeric values are skipped, not zero-counted
			}
			if n == 0 {
				minV, maxV = v, v
			} else {
				if v < minV {
					minV = v
				}
				if v > maxV {
					maxV = v
				}
			}
			sum += v
			n++
		}
		switch a.Fn {
		case oql.AggSum:
			return sum
		case oql.AggAvg:
			if n == 0 {
				return 0
			}
			return sum / float64(n)
		case oql.AggMin:
			return minV
		case oql.AggMax:
			return maxV
		}
		return 0
	}
}

// timechartBuckets buckets events into fixed spans; per-by-field series when
// requested. Buckets are emitted oldest-first with gaps preserved as absent
// keys (the frontend renders missing buckets as zero).
func timechartBuckets(in []events.Event, a *oql.Agg) []AggBucket {
	type cell struct {
		total  float64
		series map[string]float64
	}
	cells := map[int64]*cell{}
	for _, ev := range in {
		b := ev.Timestamp.UTC().Truncate(a.Span).Unix()
		c := cells[b]
		if c == nil {
			c = &cell{series: map[string]float64{}}
			cells[b] = c
		}
		c.total++
		if a.By != "" {
			key := fieldValue(ev, a.By)
			if key == "" {
				key = "(missing)"
			}
			c.series[key]++
		}
	}
	keys := make([]int64, 0, len(cells))
	for k := range cells {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make([]AggBucket, 0, len(keys))
	for _, k := range keys {
		b := AggBucket{Key: time.Unix(k, 0).UTC().Format(time.RFC3339), Value: cells[k].total}
		if a.By != "" {
			b.Series = cells[k].series
		}
		out = append(out, b)
	}
	return out
}
