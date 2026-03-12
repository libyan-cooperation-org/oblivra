package ueba

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// EntityProfile represents the behavioral baseline for a user or host.
type EntityProfile struct {
	ID             string             `json:"id"`
	EntityType     string             `json:"type"` // "user" or "host"
	LastSeen       string             `json:"last_seen"`
	RiskScore      float64            `json:"risk_score"`
	FeatureVectors map[string]float64 `json:"features"`
	Observations   int64              `json:"observations"`
	PeerGroupID    string             `json:"peer_group_id"`
	RiskHistory    []RiskPoint        `json:"risk_history"`
	mu             sync.RWMutex
}

const (
	// EMAAlpha defines the weight of new observations (0.1 = slow decay, 0.9 = fast)
	EMAAlpha = 0.2
)

type RiskPoint struct {
	Timestamp string  `json:"timestamp"`
	Score     float64   `json:"score"`
}

// KVStore defines the interface for persisting UEBA profiles (backed by BadgerDB)
type KVStore interface {
	Put(key []byte, value []byte, ttl time.Duration) error
	Get(key []byte) ([]byte, error)
	IteratePrefix(prefix []byte, fn func(key, value []byte) error) error
}

// BaselineStore manages a collection of entity profiles.
type BaselineStore struct {
	profiles map[string]*EntityProfile
	store    KVStore
	mu       sync.RWMutex
}

func NewBaselineStore(store KVStore) *BaselineStore {
	return &BaselineStore{
		profiles: make(map[string]*EntityProfile),
		store:    store,
	}
}

func (s *BaselineStore) GetProfile(id string) *EntityProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profiles[id]
}

func (s *BaselineStore) GetOrCreateProfile(id string, entityType string) *EntityProfile {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p, ok := s.profiles[id]; ok {
		return p
	}

	p := &EntityProfile{
		ID:             id,
		EntityType:     entityType,
		FeatureVectors: make(map[string]float64),
		RiskHistory:    make([]RiskPoint, 0),
		LastSeen:       time.Now().Format(time.RFC3339),
	}
	s.profiles[id] = p
	return p
}

func (p *EntityProfile) UpdateFeature(name string, value float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Use Exponential Moving Average (EMA) to prevent counter drift
	// and prioritize recent behavior.
	// If it's a new feature, initialize it.
	current, ok := p.FeatureVectors[name]
	if !ok {
		p.FeatureVectors[name] = value
	} else {
		p.FeatureVectors[name] = (EMAAlpha * value) + ((1.0 - EMAAlpha) * current)
	}

	p.Observations++
	p.LastSeen = time.Now().Format(time.RFC3339)
}

func (p *EntityProfile) SetRiskScore(score float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.RiskScore = score

	p.RiskHistory = append(p.RiskHistory, RiskPoint{
		Timestamp: time.Now().Format(time.RFC3339),
		Score:     score,
	})

	// Keep last 100 points
	if len(p.RiskHistory) > 100 {
		p.RiskHistory = p.RiskHistory[len(p.RiskHistory)-100:]
	}
}

func (s *BaselineStore) GetAllProfiles() []*EntityProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*EntityProfile
	for _, p := range s.profiles {
		list = append(list, p)
	}
	return list
}

// Save persists a specific profile to the KV store.
func (s *BaselineStore) Save(id string) error {
	if s.store == nil {
		return nil
	}

	s.mu.RLock()
	p, ok := s.profiles[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("profile %s not found", id)
	}

	p.mu.RLock()
	data, err := json.Marshal(p)
	p.mu.RUnlock()
	if err != nil {
		return err
	}

	key := []byte("ueba:profile:" + id)
	return s.store.Put(key, data, 0)
}

// LoadAll restores all profiles from the KV store.
func (s *BaselineStore) LoadAll() error {
	if s.store == nil {
		return nil
	}

	prefix := []byte("ueba:profile:")
	return s.store.IteratePrefix(prefix, func(key, value []byte) error {
		var p EntityProfile
		if err := json.Unmarshal(value, &p); err != nil {
			return err
		}

		s.mu.Lock()
		s.profiles[p.ID] = &p
		s.mu.Unlock()
		return nil
	})
}

func parseTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return t
}
