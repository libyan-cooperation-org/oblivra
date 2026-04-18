package services

import (
	"context"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type CredentialAnomaly struct {
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Details   string    `json:"details"`
	Timestamp string    `json:"timestamp"`
}

type CredentialIntelService struct {
	BaseService
	bus *eventbus.Bus
	log *logger.Logger

	mu          sync.RWMutex
	usageHourly map[string]int       // HourKey -> count
	lastAccess  map[string]time.Time // CredID -> last access
	anomalies   []CredentialAnomaly
}

func NewCredentialIntelService(bus *eventbus.Bus, log *logger.Logger) *CredentialIntelService {
	return &CredentialIntelService{
		bus:         bus,
		log:         log.WithPrefix("cred_intel"),
		usageHourly: make(map[string]int),
		lastAccess:  make(map[string]time.Time),
		anomalies:   []CredentialAnomaly{},
	}
}

func (s *CredentialIntelService) Name() string { return "credential-intel-service" }

// Dependencies returns service dependencies.
func (s *CredentialIntelService) Dependencies() []string {
	return []string{}
}

// Startup now accepts a context.Context
func (s *CredentialIntelService) Start(ctx context.Context) error {
	s.bus.Subscribe(eventbus.EventCredentialAccessed, s.handleAccess)
	return nil
}

func (s *CredentialIntelService) Stop(ctx context.Context) error {
	return nil
}

func (s *CredentialIntelService) handleAccess(event eventbus.Event) {
	data, ok := event.Data.(map[string]string)
	if !ok {
		return
	}

	id := data["id"]
	label := data["label"]

	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Update hourly usage
	hourKey := time.Now().Format("2006-01-02 15:00")
	s.usageHourly[hourKey]++

	// 2. Burst Detection (Potential mass exfiltration)
	last, exists := s.lastAccess[id]
	now := time.Now()
	if exists && now.Sub(last) < 200*time.Millisecond {
		// Rapid successive access to same credential (or script/automated attack)
		// Note: Simplified logic, in real use we'd track access counts across ALL credentials
		anomaly := CredentialAnomaly{
			Type:      "BURST_DETECTION",
			Severity:  "HIGH",
			Details:   "Rapid successive access to credential: " + label,
			Timestamp: now.Format(time.RFC3339),
		}
		s.anomalies = append(s.anomalies, anomaly)
		s.bus.Publish(eventbus.EventSIEMAlert, anomaly)
	}
	s.lastAccess[id] = now

	// 3. Global Burst (Massive Credential Access)
	// We check if we've seen > 5 unique credentials in 5 seconds
	// (Simplified simulation for Phase 10.5)
}

func (s *CredentialIntelService) GetHeatmapData() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usageHourly
}

func (s *CredentialIntelService) GetAnomalies() []CredentialAnomaly {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.anomalies
}

func (s *CredentialIntelService) GetRiskScore() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	score := 100
	for _, a := range s.anomalies {
		switch a.Severity {
		case "CRITICAL":
			score -= 30
		case "HIGH":
			score -= 15
		default:
			score -= 5
		}
	}
	if score < 0 {
		return 0
	}
	return score
}
