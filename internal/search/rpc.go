package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SearchRequest represents the RPC payload for a peer search
type SearchRequest struct {
	TenantID string `json:"tenant_id"`
	Query    string `json:"query"`
	Limit    int    `json:"limit"`
}

// SearchResponse represents the RPC response from a peer
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Error   string         `json:"error,omitempty"`
}

// CallPeerSearch performs an HTTP RPC call to a remote shard
func CallPeerSearch(ctx context.Context, peer Peer, tenantID, query string, limit int) ([]SearchResult, error) {
	reqBody, err := json.Marshal(SearchRequest{
		TenantID: tenantID,
		Query:    query,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/search/shard", peer.Address)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// Add internal cluster token here in production
	req.Header.Set("X-Oblivra-Cluster-Token", "oblivra-internal-secret")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned status %d", resp.StatusCode)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	if searchResp.Error != "" {
		return nil, fmt.Errorf("peer error: %s", searchResp.Error)
	}

	return searchResp.Results, nil
}

// ShardSearchHandler provides an HTTP handler for peers to expose their local search index
func ShardSearchHandler(engine *SearchEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify cluster token
		token := r.Header.Get("X-Oblivra-Cluster-Token")
		if token != "oblivra-internal-secret" {
			http.Error(w, "Unauthorized cluster access", http.StatusUnauthorized)
			return
		}

		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		results, err := engine.Search(req.TenantID, req.Query, req.Limit, 0)
		resp := SearchResponse{Results: results}
		if err != nil {
			resp.Error = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
