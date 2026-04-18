package oql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FetchRequest represents the RPC payload for pushing a data fetch to a shard
type FetchRequest struct {
	TenantID  string    `json:"tenant_id"`
	Search    string    `json:"search_expr_json"` // Serialized expression for now, or just query string
	TimeRange TimeRange `json:"time_range"`
}

// FetchResponse represents the RPC response from a shard
type FetchResponse struct {
	Rows  []Row  `json:"rows"`
	Error string `json:"error,omitempty"`
}

// CallPeerFetch performs an HTTP RPC call to a remote shard to fetch raw rows
func CallPeerFetch(ctx context.Context, address, tenantID string, search SearchExpr, tr TimeRange) ([]Row, error) {
	// Simple serialization of search expr (in MVP we might just send the query string if available)
	// For now, we'll serialize the expression tree to JSON if possible, or just a placeholder
	searchJSON, _ := json.Marshal(search)

	reqBody, err := json.Marshal(FetchRequest{
		TenantID:  tenantID,
		Search:    string(searchJSON),
		TimeRange: tr,
	})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/oql/shard/fetch", address)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Oblivra-Cluster-Token", "oblivra-internal-secret")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned status %d", resp.StatusCode)
	}

	var fetchResp FetchResponse
	if err := json.NewDecoder(resp.Body).Decode(&fetchResp); err != nil {
		return nil, err
	}

	if fetchResp.Error != "" {
		return nil, fmt.Errorf("peer error: %s", fetchResp.Error)
	}

	return fetchResp.Rows, nil
}

// ShardFetchHandler provides an HTTP handler for shards to expose their local BadgerSource
func ShardFetchHandler(source DataSource) http.HandlerFunc {
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

		var req FetchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Reconstruct SearchExpr from JSON (In a full implementation, we'd need a registry or type field)
		// For MVP, we'll just pass nil or a basic text search if reconstruction fails
		var expr SearchExpr
		_ = json.Unmarshal([]byte(req.Search), &expr)

		rows, err := source.Fetch(r.Context(), expr, req.TimeRange)
		resp := FetchResponse{Rows: rows}
		if err != nil {
			resp.Error = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
