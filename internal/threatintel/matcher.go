package threatintel

import (
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// MatchEngine provides O(1) in-memory lookups for high-velocity IOC matching
type MatchEngine struct {
	// Maps type -> value -> Indicator
	// e.g., "ipv4-addr" -> "192.168.1.1" -> {Indicator struct}
	store     map[string]map[string]Indicator
	campaigns map[string]Campaign
	mu        sync.RWMutex
	log       *logger.Logger
}

func NewMatchEngine(log *logger.Logger) *MatchEngine {
	return &MatchEngine{
		store:     make(map[string]map[string]Indicator),
		campaigns: make(map[string]Campaign),
		log:       log,
	}
}

// Load syncs a list of parsed indicators into the high-speed maps
func (m *MatchEngine) Load(indicators []Indicator) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	loaded := 0
	now := time.Now()

	for _, ind := range indicators {
		// Drop expired indicators
		if ind.ExpiresAt != "" && parseTime(ind.ExpiresAt).Before(now) {
			continue
		}

		if m.store[ind.Type] == nil {
			m.store[ind.Type] = make(map[string]Indicator)
		}

		// Insert or update
		m.store[ind.Type][ind.Value] = ind
		loaded++
	}

	m.log.Info("IOC Match Engine loaded %d valid indicators", loaded)
	return loaded
}

// Match evaluates a candidate string against a specific IOC type map
// Returns the indicator template containing source/severity if found
func (m *MatchEngine) Match(iocType, value string) (*Indicator, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	typeMap, exists := m.store[iocType]
	if !exists {
		return nil, false
	}

	ind, match := typeMap[value]
	if !match {
		return nil, false
	}

	// Check expiry lazily during match to avoid heavy GC sweeper routines
	if ind.ExpiresAt != "" && parseTime(ind.ExpiresAt).Before(time.Now()) {
		return nil, false
	}

	return &ind, true
}

// MatchAny checks all known spaces (IPs, domains, hashes) for a specific string
// Useful for unstructured log parsing
func (m *MatchEngine) MatchAny(value string) (*Indicator, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, typeMap := range m.store {
		ind, match := typeMap[value]
		if match {
			if ind.ExpiresAt != "" && parseTime(ind.ExpiresAt).Before(time.Now()) {
				return nil, false
			}
			return &ind, true
		}
	}

	return nil, false
}

// Stats returns the total count of indicators per type category
func (m *MatchEngine) Stats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]int)
	for k, v := range m.store {
		stats[k] = len(v)
	}
	return stats
}

// Clear removes all loaded IOCs from memory
func (m *MatchEngine) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = make(map[string]map[string]Indicator)
	m.campaigns = make(map[string]Campaign)
}

// LoadCampaigns registers campaign metadata for correlation
func (m *MatchEngine) LoadCampaigns(campaigns []Campaign) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range campaigns {
		m.campaigns[c.ID] = c
	}
}

// GetCampaign returns metadata for a specific campaign ID
func (m *MatchEngine) GetCampaign(id string) (Campaign, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.campaigns[id]
	return c, ok
}
