package search

import (
	"context"
	"sort"
	"sync"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// Peer represents a remote search shard
type Peer struct {
	ID       string
	Address  string // e.g. "http://10.0.0.5:8080"
	IsActive bool
}

// Federator coordinates distributed searches across multiple shards
type Federator struct {
	local *SearchEngine
	peers []Peer
	log   *logger.Logger
	mu    sync.RWMutex
}

// NewFederator creates a new query coordinator
func NewFederator(local *SearchEngine, log *logger.Logger) *Federator {
	return &Federator{
		local: local,
		peers: make([]Peer, 0),
		log:   log,
	}
}

// AddPeer adds a remote shard to the federation (idempotent)
func (f *Federator) AddPeer(id, address string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.peers {
		if p.ID == id || p.Address == address {
			return
		}
	}
	f.peers = append(f.peers, Peer{ID: id, Address: address, IsActive: true})
}

// ActivePeerAddresses returns a list of addresses for all currently active peers
func (f *Federator) ActivePeerAddresses() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var addrs []string
	for _, p := range f.peers {
		if p.IsActive {
			addrs = append(addrs, p.Address)
		}
	}
	return addrs
}

// Search coordinates a global search across all active peers and the local index
func (f *Federator) Search(ctx context.Context, tenantID, query string, limit, offset int) ([]SearchResult, error) {
	f.mu.RLock()
	peers := make([]Peer, len(f.peers))
	copy(peers, f.peers)
	f.mu.RUnlock()

	resultsChan := make(chan []SearchResult, len(peers)+1)
	errChan := make(chan error, len(peers)+1)

	var wg sync.WaitGroup

	// 1. Local Search
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := f.local.Search(tenantID, query, limit+offset, 0)
		if err != nil {
			errChan <- err
			return
		}
		resultsChan <- res
	}()

	// 2. Peer Search (Fan-out)
	for _, p := range peers {
		if !p.IsActive {
			continue
		}
		wg.Add(1)
		go func(peer Peer) {
			defer wg.Done()
			// Implement RPC call in rpc.go
			res, err := f.queryPeer(ctx, peer, tenantID, query, limit+offset)
			if err != nil {
				f.log.Warn("[FEDERATION] Peer %s search failed: %v", peer.ID, err)
				return // Continue with partial results
			}
			resultsChan <- res
		}(p)
	}

	// Wait for all or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All finished
	case <-ctx.Done():
		f.log.Warn("[FEDERATION] Search context cancelled/timeout")
	}

	close(resultsChan)
	close(errChan)

	// 3. Aggregate and Sort (Fan-in)
	var allResults []SearchResult
	for res := range resultsChan {
		allResults = append(allResults, res...)
	}

	// Sort by Score (or timestamp if available in data)
	sort.Slice(allResults, func(i, j int) bool {
		// Prefer Score for Lucene relevance
		if allResults[i].Score != allResults[j].Score {
			return allResults[i].Score > allResults[j].Score
		}
		// Fallback to timestamp if present in Data
		ti, _ := allResults[i].Data["timestamp"].(float64)
		tj, _ := allResults[j].Data["timestamp"].(float64)
		return ti > tj
	})

	// 4. Apply Limit/Offset
	start := offset
	if start > len(allResults) {
		return []SearchResult{}, nil
	}
	end := start + limit
	if end > len(allResults) {
		end = len(allResults)
	}

	return allResults[start:end], nil
}

// queryPeer is a placeholder for the actual RPC implementation
func (f *Federator) queryPeer(ctx context.Context, peer Peer, tenantID, query string, limit int) ([]SearchResult, error) {
	// This will be implemented in rpc.go using HTTP/JSON
	return CallPeerSearch(ctx, peer, tenantID, query, limit)
}
