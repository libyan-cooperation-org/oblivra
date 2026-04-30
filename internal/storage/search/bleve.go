// Package search wraps Bleve to give us full-text + field queries over the
// events that pass through ingest. Per-tenant index instances keep tenants
// physically separated (no shared index → cross-tenant escape is impossible).
package search

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/kingknull/oblivra/internal/events"
)

// Index manages one Bleve index per tenant. Indices are created lazily when
// the first event for a tenant arrives.
type Index struct {
	dir      string
	inMemory bool

	mu       sync.RWMutex
	tenants  map[string]bleve.Index
}

type Options struct {
	Dir      string // root dir; per-tenant indices live under {dir}/{tenant}.bleve
	InMemory bool
}

func Open(opts Options) (*Index, error) {
	if opts.Dir == "" && !opts.InMemory {
		return nil, errors.New("search: Dir is required unless InMemory is set")
	}
	if !opts.InMemory {
		if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
			return nil, fmt.Errorf("search mkdir: %w", err)
		}
	}
	return &Index{
		dir:      opts.Dir,
		inMemory: opts.InMemory,
		tenants:  make(map[string]bleve.Index),
	}, nil
}

func (i *Index) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	var first error
	for _, idx := range i.tenants {
		if err := idx.Close(); err != nil && first == nil {
			first = err
		}
	}
	i.tenants = nil
	return first
}

// Index inserts/updates an event. Safe for concurrent calls.
func (i *Index) Index(ev *events.Event) error {
	idx, err := i.tenantIndex(ev.TenantID)
	if err != nil {
		return err
	}
	doc := map[string]any{
		"tenantId":   ev.TenantID,
		"timestamp":  ev.Timestamp.Format(time.RFC3339Nano),
		"timestampS": ev.Timestamp.Unix(),
		"source":     string(ev.Source),
		"hostId":     ev.HostID,
		"eventType":  ev.EventType,
		"severity":   string(ev.Severity),
		"message":    ev.Message,
		"raw":        ev.Raw,
	}
	for k, v := range ev.Fields {
		doc["f_"+k] = v
	}
	return idx.Index(ev.ID, doc)
}

// Hit is one search match — only IDs are returned; the caller looks up the
// full Event in the hot store.
type Hit struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

// QueryOpts narrows a search.
type QueryOpts struct {
	TenantID string
	Q        string    // free-text query string (Bleve syntax)
	From     time.Time // optional time bound (inclusive)
	To       time.Time // optional time bound (inclusive)
	Limit    int
	NewestFirst bool
}

// Result wraps the hit list plus paging metadata.
type Result struct {
	Hits     []Hit
	Total    uint64
	Took     time.Duration
}

// Query runs a Bleve search against the tenant's index.
func (i *Index) Query(opts QueryOpts) (Result, error) {
	if opts.TenantID == "" {
		opts.TenantID = "default"
	}
	if opts.Limit <= 0 || opts.Limit > 1000 {
		opts.Limit = 100
	}

	idx, err := i.tenantIndex(opts.TenantID)
	if err != nil {
		return Result{}, err
	}

	var q query.Query
	if opts.Q == "" || opts.Q == "*" {
		q = bleve.NewMatchAllQuery()
	} else {
		q = bleve.NewQueryStringQuery(opts.Q)
	}

	if !opts.From.IsZero() || !opts.To.IsZero() {
		from := opts.From.Unix()
		to := opts.To.Unix()
		if opts.To.IsZero() {
			to = time.Now().Unix()
		}
		fromF := float64(from)
		toF := float64(to)
		incl := true
		rng := bleve.NewNumericRangeInclusiveQuery(&fromF, &toF, &incl, &incl)
		rng.SetField("timestampS")
		q = bleve.NewConjunctionQuery(q, rng)
	}

	req := bleve.NewSearchRequestOptions(q, opts.Limit, 0, false)
	if opts.NewestFirst {
		req.SortBy([]string{"-timestampS"})
	}

	start := time.Now()
	res, err := idx.Search(req)
	if err != nil {
		return Result{}, err
	}

	hits := make([]Hit, 0, len(res.Hits))
	for _, h := range res.Hits {
		hits = append(hits, Hit{ID: h.ID, Score: h.Score})
	}
	return Result{Hits: hits, Total: res.Total, Took: time.Since(start)}, nil
}

func (i *Index) tenantIndex(tenantID string) (bleve.Index, error) {
	if tenantID == "" {
		tenantID = "default"
	}
	i.mu.RLock()
	idx, ok := i.tenants[tenantID]
	i.mu.RUnlock()
	if ok {
		return idx, nil
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	if idx, ok := i.tenants[tenantID]; ok {
		return idx, nil
	}

	mapping := buildMapping()
	var (
		bidx bleve.Index
		err  error
	)
	if i.inMemory {
		bidx, err = bleve.NewMemOnly(mapping)
	} else {
		path := filepath.Join(i.dir, sanitize(tenantID)+".bleve")
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			bidx, err = bleve.New(path, mapping)
		} else {
			bidx, err = bleve.Open(path)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("open tenant index %q: %w", tenantID, err)
	}
	i.tenants[tenantID] = bidx
	return bidx, nil
}

func buildMapping() *mapping.IndexMappingImpl {
	m := bleve.NewIndexMapping()
	doc := bleve.NewDocumentMapping()

	keyword := bleve.NewKeywordFieldMapping()
	text := bleve.NewTextFieldMapping()
	num := bleve.NewNumericFieldMapping()

	doc.AddFieldMappingsAt("tenantId", keyword)
	doc.AddFieldMappingsAt("source", keyword)
	doc.AddFieldMappingsAt("hostId", keyword)
	doc.AddFieldMappingsAt("eventType", keyword)
	doc.AddFieldMappingsAt("severity", keyword)
	doc.AddFieldMappingsAt("timestamp", keyword)
	doc.AddFieldMappingsAt("timestampS", num)
	doc.AddFieldMappingsAt("message", text)
	doc.AddFieldMappingsAt("raw", text)

	m.DefaultMapping = doc
	return m
}

// sanitize strips path-unsafe characters from tenant IDs.
func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '-', c == '_':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "default"
	}
	return string(out)
}
