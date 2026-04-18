package search

import (
"fmt"
"os"
"path/filepath"
"strings"
"sync"

"github.com/blevesearch/bleve/v2"
"github.com/blevesearch/bleve/v2/mapping"
"github.com/blevesearch/bleve/v2/search/query"
"github.com/kingknull/oblivrashell/internal/logger"
)

// SearchEngine provides full-text search capabilities over terminal logs and SIEM events
type SearchEngine struct {
	mu      sync.RWMutex
	indexes map[string]bleve.Index
	dataDir string
	log     *logger.Logger
}

// SearchResult represents a single hit from the search engine
type SearchResult struct {
	ID    string
	Score float64
	Data  map[string]interface{}
}

// NewSearchEngine creates a new search index or opens an existing one
func NewSearchEngine(dataDir string, log *logger.Logger) (*SearchEngine, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create search directory: %w", err)
	}

	return &SearchEngine{
		indexes: make(map[string]bleve.Index),
		dataDir: dataDir,
		log:     log,
	}, nil
}

func (s *SearchEngine) getIndex(tenantID string) (bleve.Index, error) {
	s.mu.RLock()
	idx, exists := s.indexes[tenantID]
	s.mu.RUnlock()
	if exists {
		return idx, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check
	idx, exists = s.indexes[tenantID]
	if exists {
		return idx, nil
	}

	indexPath := filepath.Join(s.dataDir, fmt.Sprintf("bleve_%s.idx", tenantID))

	if _, errStat := os.Stat(indexPath); os.IsNotExist(errStat) {
		idxMapping := buildIndexMapping()
		newIdx, err := bleve.New(indexPath, idxMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create bleve index for %s: %w", tenantID, err)
		}
		s.indexes[tenantID] = newIdx
		if s.log != nil {
			s.log.Info("[SEARCH] Created new Bleve index at %s", indexPath)
		}
		return newIdx, nil
	}

	existingIdx, err := bleve.Open(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open bleve index for %s: %w", tenantID, err)
	}
	s.indexes[tenantID] = existingIdx
	if s.log != nil {
		s.log.Info("[SEARCH] Opened existing Bleve index at %s", indexPath)
	}
	return existingIdx, nil
}

// buildIndexMapping configures how fields are indexed
func buildIndexMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	// Document mapping for logs
	logMapping := bleve.NewDocumentMapping()

	// Text fields (tokenized, indexed)
	outputFieldMapping := bleve.NewTextFieldMapping()
	outputFieldMapping.Analyzer = "standard"

	sessionFieldMapping := bleve.NewTextFieldMapping()
	sessionFieldMapping.Analyzer = "keyword" // exact match

	hostFieldMapping := bleve.NewTextFieldMapping()
	hostFieldMapping.Analyzer = "keyword"

	tenantFieldMapping := bleve.NewTextFieldMapping()
	tenantFieldMapping.Analyzer = "keyword"

	// Tie mappings to fields
	logMapping.AddFieldMappingsAt("output", outputFieldMapping)
	logMapping.AddFieldMappingsAt("session_id", sessionFieldMapping)
	logMapping.AddFieldMappingsAt("host", hostFieldMapping)
	logMapping.AddFieldMappingsAt("tenant", tenantFieldMapping)

	indexMapping.AddDocumentMapping("log_entry", logMapping)
	return indexMapping
}

// Close cleanly shuts down the search index
func (s *SearchEngine) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for tenantID, idx := range s.indexes {
		if s.log != nil {
			s.log.Info("[SEARCH] Closing Bleve index for %s", tenantID)
		}
		_ = idx.Close()
	}
	return nil
}

// Index adds or updates a document in the search index
func (s *SearchEngine) Index(tenantID string, id string, docType string, data map[string]interface{}) error {
	idx, err := s.getIndex(tenantID)
	if err != nil {
		return err
	}

	data["_type"] = docType
	return idx.Index(id, data)
}

// BatchIndex adds multiple documents to the index in a single batch for performance.
func (s *SearchEngine) BatchIndex(tenantID string, docs map[string]interface{}, docType string) error {
	idx, err := s.getIndex(tenantID)
	if err != nil {
		return err
	}

	batch := idx.NewBatch()
	for id, data := range docs {
		m, ok := data.(map[string]interface{})
		if !ok {
			continue
		}
		m["_type"] = docType
		batch.Index(id, m)
	}
	return idx.Batch(batch)
}

// Search executes a Lucene-style query against the index
func (s *SearchEngine) Search(tenantID string, queryStr string, limit, offset int) ([]SearchResult, error) {
	idx, err := s.getIndex(tenantID)
	if err != nil {
		return nil, err
	}

	var q query.Query
	if strings.Contains(queryStr, ":") {
		q = bleve.NewQueryStringQuery(queryStr)
	} else if queryStr != "" {
		matchQ := bleve.NewMatchQuery(queryStr)
		matchQ.SetField("output")
		q = matchQ
	} else {
		q = bleve.NewMatchAllQuery()
	}

	req := bleve.NewSearchRequestOptions(q, limit, offset, false)
	req.Fields = []string{"*"}
	req.SortBy([]string{"-timestamp"})

	res, err := idx.Search(req)
	if err != nil {
		return nil, fmt.Errorf("bleve search failed: %w", err)
	}

	var results []SearchResult
	for _, hit := range res.Hits {
		data := make(map[string]interface{})
		for k, v := range hit.Fields {
			data[k] = v
		}

		results = append(results, SearchResult{
ID:    hit.ID,
Score: hit.Score,
Data:  data,
})
	}

	return results, nil
}

// Aggregate performs a faceted search against the index to count occurrences of a specific field.
func (s *SearchEngine) Aggregate(tenantID, queryStr string, facetField string) (map[string]int, error) {
	idx, err := s.getIndex(tenantID)
	if err != nil {
		return nil, err
	}

	var q query.Query
	if strings.Contains(queryStr, ":") {
		q = bleve.NewQueryStringQuery(queryStr)
	} else if queryStr != "" {
		matchQ := bleve.NewMatchQuery(queryStr)
		matchQ.SetField("output")
		q = matchQ
	} else {
		q = bleve.NewMatchAllQuery()
	}

	req := bleve.NewSearchRequestOptions(q, 0, 0, false)
	facetReq := bleve.NewFacetRequest(facetField, 50)
	req.AddFacet("term_facet", facetReq)

	res, err := idx.Search(req)
	if err != nil {
		return nil, fmt.Errorf("bleve aggregate failed: %w", err)
	}

	results := make(map[string]int)
	if facet, exists := res.Facets["term_facet"]; exists {
		for _, term := range facet.Terms.Terms() {
			results[term.Term] = term.Count
		}
	}
	return results, nil
}

// Delete removes a document from the index
func (s *SearchEngine) Delete(tenantID, id string) error {
	idx, err := s.getIndex(tenantID)
	if err != nil {
		return err
	}
	return idx.Delete(id)
}

// Count returns the total number of documents in the index
func (s *SearchEngine) Count(tenantID string) (uint64, error) {
	idx, err := s.getIndex(tenantID)
	if err != nil {
		return 0, err
	}
	return idx.DocCount()
}
