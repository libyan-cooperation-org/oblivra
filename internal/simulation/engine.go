package simulation

import (
	"sync"
	"time"
)

// Campaign represents a series of simulation scenarios
type Campaign struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Scenarios   []string  `json:"scenarios"` // Slice of Scenario IDs
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time,omitempty"`
	Status      string    `json:"status"` // "running", "completed", "failed"
	TotalSteps  int       `json:"total_steps"`
	PassedSteps int       `json:"passed_steps"`
}

// CampaignManager handles multi-step attack simulations
type CampaignManager struct {
	mu        sync.RWMutex
	campaigns map[string]*Campaign
}

func NewCampaignManager() *CampaignManager {
	return &CampaignManager{
		campaigns: make(map[string]*Campaign),
	}
}

func (m *CampaignManager) StartCampaign(id, name string, scenarios []string) *Campaign {
	m.mu.Lock()
	defer m.mu.Unlock()

	c := &Campaign{
		ID:         id,
		Name:       name,
		Scenarios:  scenarios,
		StartTime:  time.Now().Format(time.RFC3339),
		Status:     "running",
		TotalSteps: len(scenarios),
	}
	m.campaigns[id] = c
	return c
}

func (m *CampaignManager) GetCampaign(id string) *Campaign {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.campaigns[id]
}

func (m *CampaignManager) ListCampaigns() []*Campaign {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Campaign, 0, len(m.campaigns))
	for _, c := range m.campaigns {
		list = append(list, c)
	}
	return list
}
