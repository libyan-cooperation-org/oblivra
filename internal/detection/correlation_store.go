package detection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/storage"
)

// correlationStateRecord is what we serialize to Badger.
type correlationStateRecord struct {
	Seen      map[string][]Event `json:"seen"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// correlationStateKey returns the namespaced Badger key for a rule's group-key state.
// Schema: tenant:{tenantID}:correlation:{ruleID}:{window}:{groupKey}
func correlationStateKey(tenantID, ruleID string, windowSec int, groupKey string) []byte {
	return []byte(fmt.Sprintf("tenant:%s:correlation:%s:%d:%s", tenantID, ruleID, windowSec, groupKey))
}

// CorrelationStore wraps HotStore for persistent cross-restart correlation state.
// This satisfies audit requirement #6 — in-memory LRU is replaced on restart.
type CorrelationStore struct {
	hot *storage.HotStore
}

// NewCorrelationStore creates a store backed by the shared BadgerDB hot store.
func NewCorrelationStore(hot *storage.HotStore) *CorrelationStore {
	return &CorrelationStore{hot: hot}
}

// SaveState persists a correlation group's seen events to Badger.
func (cs *CorrelationStore) SaveState(tenantID, ruleID string, windowSec int, groupKey string, seen map[string][]Event, ttl time.Duration) error {
	if cs.hot == nil {
		return nil
	}
	rec := correlationStateRecord{
		Seen:      seen,
		UpdatedAt: time.Now().UTC(),
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("correlation persist marshal: %w", err)
	}
	key := correlationStateKey(tenantID, ruleID, windowSec, groupKey)
	return cs.hot.Put(key, b, ttl)
}

// LoadState retrieves a previously persisted correlation state.
// Returns nil, nil if not found (first-time or expired).
func (cs *CorrelationStore) LoadState(tenantID, ruleID string, windowSec int, groupKey string) (map[string][]Event, error) {
	if cs.hot == nil {
		return nil, nil
	}
	key := correlationStateKey(tenantID, ruleID, windowSec, groupKey)
	b, err := cs.hot.Get(key)
	if err != nil || b == nil {
		return nil, err
	}
	var rec correlationStateRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, fmt.Errorf("correlation persist unmarshal: %w", err)
	}
	return rec.Seen, nil
}

// DeleteState removes a correlation state once a rule has fired.
func (cs *CorrelationStore) DeleteState(tenantID, ruleID string, windowSec int, groupKey string) error {
	if cs.hot == nil {
		return nil
	}
	key := correlationStateKey(tenantID, ruleID, windowSec, groupKey)
	return cs.hot.Delete(key)
}
