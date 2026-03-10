package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
)

type BiasEntry struct {
	AnomalyID string                   `json:"anomaly_id"`
	Reason    string                   `json:"reason"`
	Evidence  []map[string]interface{} `json:"evidence"`
	Timestamp time.Time                `json:"timestamp"`
	User      string                   `json:"user"` // Operator who marked it
}

type GovernanceService struct {
	BaseService
	bus *eventbus.Bus
	log *logger.Logger

	mu       sync.Mutex
	biasLogs []BiasEntry
}

func NewGovernanceService(bus *eventbus.Bus, log *logger.Logger) *GovernanceService {
	return &GovernanceService{
		bus: bus,
		log: log.WithPrefix("governance"),
	}
}

func (s *GovernanceService) Name() string { return "GovernanceService" }

func (s *GovernanceService) MarkFalsePositive(anomalyID string, reason string, evidence []map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := BiasEntry{
		AnomalyID: anomalyID,
		Reason:    reason,
		Evidence:  evidence,
		Timestamp: time.Now(),
		User:      "admin", // Placeholder
	}

	s.biasLogs = append(s.biasLogs, entry)
	s.log.Info("Marked anomaly %s as False Positive. Reason: %s", anomalyID, reason)

	// Persist to disk for audit/retraining
	if err := s.saveBiasLogs(); err != nil {
		s.log.Error("Failed to save bias logs: %v", err)
	}

	// Publish event for Merkle audit trail integration
	s.bus.Publish("governance.fp_marked", entry)

	return nil
}

func (s *GovernanceService) saveBiasLogs() error {
	path := filepath.Join(platform.DataDir(), "governance", "bias_logs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.biasLogs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func (s *GovernanceService) GetBiasLogs() []BiasEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.biasLogs
}
