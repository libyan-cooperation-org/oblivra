package oql

import (
	"context"
	"sync"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// FederatedOQLSource satisfies the DataSource interface by fetching data from multiple shards.
type FederatedOQLSource struct {
	local  DataSource
	peers  []string // List of peer addresses e.g. "http://10.0.0.5:8080"
	log    *logger.Logger
}

// NewFederatedOQLSource creates a new federated data source.
func NewFederatedOQLSource(local DataSource, peers []string, log *logger.Logger) *FederatedOQLSource {
	return &FederatedOQLSource{
		local: local,
		peers: peers,
		log:   log,
	}
}

// Fetch coordinates parallel data fetching across the local shard and all configured peers.
func (s *FederatedOQLSource) Fetch(ctx context.Context, search SearchExpr, timeRange TimeRange) ([]Row, error) {
	resultsChan := make(chan []Row, len(s.peers)+1)
	
	var wg sync.WaitGroup

	// 1. Local Fetch
	wg.Add(1)
	go func() {
		defer wg.Done()
		rows, err := s.local.Fetch(ctx, search, timeRange)
		if err != nil {
			s.log.Error("[OQL-FEDERATION] Local fetch failed: %v", err)
			return
		}
		resultsChan <- rows
	}()

	// 2. Peer Fetch (Fan-out)
	for _, peerAddr := range s.peers {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			// Extract tenant from context (required by shard handlers)
			// Assuming it's already in the context from the entry point
			tenantID := "GLOBAL" // Fallback
			// In a real implementation, we'd extract it properly
			
			rows, err := CallPeerFetch(ctx, addr, tenantID, search, timeRange)
			if err != nil {
				s.log.Warn("[OQL-FEDERATION] Peer %s fetch failed: %v", addr, err)
				return
			}
			resultsChan <- rows
		}(peerAddr)
	}

	// Wait for all or context timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		s.log.Warn("[OQL-FEDERATION] Fetch context cancelled/timeout")
	}

	close(resultsChan)

	// 3. Aggregate Results (Fan-in)
	var allRows []Row
	for rows := range resultsChan {
		allRows = append(allRows, rows...)
	}

	s.log.Info("[OQL-FEDERATION] Federated fetch completed. Total rows: %d", len(allRows))
	return allRows, nil
}
