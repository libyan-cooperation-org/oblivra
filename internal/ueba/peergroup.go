package ueba

import (
	"math"
	"sync"
)

// PeerGroup tracks the aggregate behavior of a cohort of entities.
type PeerGroup struct {
	ID           string
	FeatureStats map[string]*StatSummary
	mu           sync.RWMutex
}

// StatSummary tracks running mean and variance for a feature.
type StatSummary struct {
	Mean  float64
	M2    float64
	Count int64
}

func NewPeerGroup(id string) *PeerGroup {
	return &PeerGroup{
		ID:           id,
		FeatureStats: make(map[string]*StatSummary),
	}
}

func (g *PeerGroup) Update(features map[string]float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for name, val := range features {
		s, ok := g.FeatureStats[name]
		if !ok {
			s = &StatSummary{}
			g.FeatureStats[name] = s
		}

		// Welford's online algorithm for mean and variance
		s.Count++
		delta := val - s.Mean
		s.Mean += delta / float64(s.Count)
		delta2 := val - s.Mean
		s.M2 += delta * delta2
	}
}

func (g *PeerGroup) GetDeviation(features map[string]float64) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.FeatureStats) == 0 {
		return 0.0
	}

	var totalDev float64
	var count int

	for name, val := range features {
		s, ok := g.FeatureStats[name]
		if !ok || s.Count < 2 {
			continue
		}

		variance := s.M2 / float64(s.Count-1)
		if variance == 0 {
			continue
		}

		stdDev := math.Sqrt(variance)
		zScore := math.Abs(val-s.Mean) / stdDev

		// Normalize Z-score to 0-1 (3.0+ Z-score is 1.0)
		dev := zScore / 3.0
		if dev > 1.0 {
			dev = 1.0
		}
		totalDev += dev
		count++
	}

	if count == 0 {
		return 0.0
	}
	return totalDev / float64(count)
}

// PeerGroupManager manages the lifecycle and mapping of groups.
type PeerGroupManager struct {
	groups map[string]*PeerGroup
	mu     sync.RWMutex
}

func NewPeerGroupManager() *PeerGroupManager {
	return &PeerGroupManager{
		groups: make(map[string]*PeerGroup),
	}
}

func (m *PeerGroupManager) GetOrCreateGroup(id string) *PeerGroup {
	m.mu.Lock()
	defer m.mu.Unlock()

	if g, ok := m.groups[id]; ok {
		return g
	}

	g := NewPeerGroup(id)
	m.groups[id] = g
	return g
}
