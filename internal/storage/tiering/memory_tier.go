package tiering

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MemoryTier is an in-memory implementation of Tier used by tests
// AND single-node deployments that don't need durability for the
// (small) hot tier. Production hot is the BadgerDB-backed tier in
// `hot.go`; this file gives us a deterministic, dependency-free
// implementation for the migrator tests.
type MemoryTier struct {
	id    TierID
	mu    sync.Mutex
	store map[string]Event
}

// NewMemoryTier constructs an empty tier with the given ID.
func NewMemoryTier(id TierID) *MemoryTier {
	return &MemoryTier{
		id:    id,
		store: make(map[string]Event),
	}
}

// ID implements Tier.
func (t *MemoryTier) ID() TierID { return t.id }

// Range implements Tier. Visits events in deterministic timestamp
// order — easier to reason about in tests, and the migrator's batch
// budget benefits from oldest-first ordering.
func (t *MemoryTier) Range(_ context.Context, from, to time.Time, fn func(e Event) bool) error {
	t.mu.Lock()
	events := make([]Event, 0, len(t.store))
	for _, e := range t.store {
		if !from.IsZero() && e.Timestamp.Before(from) {
			continue
		}
		if !to.IsZero() && e.Timestamp.After(to) {
			continue
		}
		events = append(events, e)
	}
	t.mu.Unlock()

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	for _, e := range events {
		if !fn(e) {
			return nil
		}
	}
	return nil
}

// Write implements Tier.
func (t *MemoryTier) Write(_ context.Context, events []Event) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, e := range events {
		t.store[e.ID] = e
	}
	return len(events), nil
}

// Delete implements Tier. Idempotent — missing IDs are not an error.
func (t *MemoryTier) Delete(_ context.Context, ids []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, id := range ids {
		delete(t.store, id)
	}
	return nil
}

// EstimatedSize implements Tier — sums the body bytes of every stored
// event for a rough on-disk-size approximation.
func (t *MemoryTier) EstimatedSize(_ context.Context) (int64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	var total int64
	for _, e := range t.store {
		total += int64(len(e.Body)) + int64(len(e.ID)) + int64(len(e.Host)) + int64(len(e.EventType))
	}
	return total, nil
}

// Count returns the number of events in the tier — test-only helper,
// not on the Tier interface.
func (t *MemoryTier) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.store)
}
