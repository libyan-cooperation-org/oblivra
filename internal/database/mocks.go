package database

import (
	"context"
	"sync"
)

// MockSIEMStore is a thread-safe in-memory store for replaying events without database contamination.
type MockSIEMStore struct {
	mu           sync.RWMutex
	Events       []*HostEvent
	SavedSearches []SavedSearch
}

func NewMockSIEMStore() *MockSIEMStore {
	return &MockSIEMStore{
		Events: make([]*HostEvent, 0),
	}
}

func (m *MockSIEMStore) InsertHostEvent(ctx context.Context, event *HostEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, event)
	return nil
}

func (m *MockSIEMStore) GetHostEvents(ctx context.Context, hostID string, limit int) ([]HostEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []HostEvent
	for _, e := range m.Events {
		if e.HostID == hostID {
			result = append(result, *e)
		}
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *MockSIEMStore) SearchHostEvents(ctx context.Context, query string, limit int) ([]HostEvent, error) {
	return nil, nil // Not required for replay match checking
}

func (m *MockSIEMStore) GetFailedLoginsByHost(ctx context.Context, hostID string) ([]map[string]interface{}, error) {
	return nil, nil
}

func (m *MockSIEMStore) CalculateRiskScore(ctx context.Context, hostID string) (int, error) {
	return 0, nil
}

func (m *MockSIEMStore) GetGlobalThreatStats(ctx context.Context) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockSIEMStore) GetEventTrend(ctx context.Context, days int) ([]map[string]interface{}, error) {
	return nil, nil
}

func (m *MockSIEMStore) AggregateHostEvents(ctx context.Context, query string, facetField string) (map[string]int, error) {
	return nil, nil
}

func (m *MockSIEMStore) CreateSavedSearch(ctx context.Context, s *SavedSearch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SavedSearches = append(m.SavedSearches, *s)
	return nil
}

func (m *MockSIEMStore) GetSavedSearches(ctx context.Context) ([]SavedSearch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.SavedSearches, nil
}
