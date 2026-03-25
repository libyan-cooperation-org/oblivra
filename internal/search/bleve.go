package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// SearchEngine provides full-text search capabilities over terminal logs and SIEM events
type SearchEngine struct {
	index bleve.Index
	log   *logger.Logger
}

// SearchResult represents a single hit from the search engine
type SearchResult struct {
	ID    string
	Score float64
	Data  map[string]interface{}
}

// NewSearchEngine creates a new search index or opens an existing one
func NewSearchEngine(dataDir string, log *logger.Logger) (*SearchEngine, error) {
	indexPath := filepath.Join(dataDir, "bleve.idx")

	var index bleve.Index
	var err error

	if _, errStat := os.Stat(indexPath); os.IsNotExist(errStat) {
		// Create new index with custom mapping
		idxMapping := buildIndexMapping()
		index, err = bleve.New(indexPath, idxMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create bleve index: %w", err)
		}
		if log != nil {
			log.Info("[SEARCH] Created new Bleve index at %s", indexPath)
		}
	} else {
		// Open existing index
		index, err = bleve.Open(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open bleve index: %w", err)
		}
		if log != nil {
			log.Info("[SEARCH] Opened existing Bleve index at %s", indexPath)
		}
	}

	return &SearchEngine{
		index: index,
		log:   log,
	}, nil
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
	if s.index != nil {
		if s.log != nil {
			s.log.Info("[SEARCH] Closing Bleve index")
		}
		return s.index.Close()
	}
	return nil
}

// Index adds or updates a document in the search index
func (s *SearchEngine) Index(id string, docType string, data map[string]interface{}) error {
	if s.index == nil {
		return fmt.Errorf("search index not initialized")
	}

	// Bleve doesn't have a strict "document type" concept out of the box unless we structure it this way
	// We inject _type to help with queries later if needed
	data["_type"] = docType

	return s.index.Index(id, data)
}

// BatchIndex adds multiple documents to the index in a single batch for performance.
func (s *SearchEngine) BatchIndex(docs map[string]interface{}, docType string) error {
	if s.index == nil {
		return fmt.Errorf("search index not initialized")
	}

	batch := s.index.NewBatch()
	for id, data := range docs {
		m, ok := data.(map[string]interface{})
		if !ok {
			continue
		}
		m["_type"] = docType
		batch.Index(id, m)
	}
	return s.index.Batch(batch)
}

// Search executes a Lucene-style query against the index
func (s *SearchEngine) Search(queryStr string, limit, offset int) ([]SearchResult, error) {
	if s.index == nil {
		return nil, fmt.Errorf("search index not initialized")
	}

	// For simple queries that don't specify fields, default to searching the 'output' field
	// If the query contains a colon (like host:foo), we use the query string parser directly
	var q query.Query
	if strings.Contains(queryStr, ":") {
		q = bleve.NewQueryStringQuery(queryStr)
	} else if queryStr != "" {
		// By default, match against output text
		matchQ := bleve.NewMatchQuery(queryStr)
		matchQ.SetField("output")
		q = matchQ
	} else {
		// Empty query returns everything
		q = bleve.NewMatchAllQuery()
	}

	req := bleve.NewSearchRequestOptions(q, limit, offset, false)
	req.Fields = []string{"*"}         // Return all stored fields
	req.SortBy([]string{"-timestamp"}) // Sort newest first (requires timestamp field)

	res, err := s.index.Search(req)
	if err != nil {
		return nil, fmt.Errorf("bleve search failed: %w", err)
	}

	var results []SearchResult
	for _, hit := range res.Hits {
		// Reconstruct the data map
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
func (s *SearchEngine) Aggregate(queryStr string, facetField string) (map[string]int, error) {
	if s.index == nil {
		return nil, fmt.Errorf("search index not initialized")
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

	// Create a facet request for the top 50 terms of the given field
	facetReq := bleve.NewFacetRequest(facetField, 50)
	req.AddFacet("term_facet", facetReq)

	res, err := s.index.Search(req)
	if err != nil {
		return nil, fmt.Errorf("bleve aggregate failed: %w", err)
	}

	results := make(map[string]int)
	if facet, exists := res.Facets["term_facet"]; exists {
		// facet.Terms is actually a pointer or type alias.
		// If it's a pointer to a struct/slice, we handle it depending on the framework version.
		// Wait, go doc says we can pull the literal terms.
		for _, term := range facet.Terms.Terms() {
			results[term.Term] = term.Count
		}
	}
	return results, nil
}

// Delete removes a document from the index
func (s *SearchEngine) Delete(id string) error {
	if s.index == nil {
		return fmt.Errorf("search index not initialized")
	}
	return s.index.Delete(id)
}

// Count returns the total number of documents in the index
func (s *SearchEngine) Count() (uint64, error) {
	if s.index == nil {
		return 0, fmt.Errorf("search index not initialized")
	}
	return s.index.DocCount()
}
