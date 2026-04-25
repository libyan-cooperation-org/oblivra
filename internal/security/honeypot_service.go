package security

import (
	"context"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// DecoyType defines the category of honeypot/decoy element
type DecoyType string

const (
	DecoyCredential DecoyType = "credential"
	DecoyFile       DecoyType = "file"
	DecoyPort       DecoyType = "port"
)

// HoneypotDecoy represents a single decoy element monitored for interaction.
type HoneypotDecoy struct {
	ID          string     `json:"id"`
	Type        DecoyType  `json:"type"`
	Target      string     `json:"target"` // HostID or path
	Value       string     `json:"value"`  // e.g., Username or Port number
	DeployTime  string     `json:"deploy_time"`
	LastTrigger *string    `json:"last_trigger,omitempty"`
}

// HoneypotService manages decoy "honeypot" elements across the infrastructure.
type HoneypotService struct {
	log    *logger.Logger
	decoys map[string]HoneypotDecoy
	mu     sync.RWMutex
}

func NewHoneypotService(log *logger.Logger) *HoneypotService {
	return &HoneypotService{
		log:    log.WithPrefix("honeypot"),
		decoys: make(map[string]HoneypotDecoy),
	}
}

func (s *HoneypotService) Startup(ctx context.Context) {}
func (s *HoneypotService) Shutdown()                   {}
func (s *HoneypotService) Name() string                { return "HoneypotService" }

// InjectHoneypotCredential creates a decoy credential that should never be used.
func (s *HoneypotService) InjectHoneypotCredential(id string, username string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	decoy := HoneypotDecoy{
		ID:         id,
		Type:       DecoyCredential,
		Value:      username,
		DeployTime: time.Now().Format(time.RFC3339),
	}
	s.decoys[id] = decoy
	s.log.Info("Injected honeypot credential: [ID:%s]", id)
	return id
}

// RegisterTrigger records an interaction with a decoy element.
func (s *HoneypotService) RegisterTrigger(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if decoy, ok := s.decoys[id]; ok {
		now := time.Now().Format(time.RFC3339)
		decoy.LastTrigger = &now
		s.decoys[id] = decoy
		// Never log decoy.Value: it may be a plaintext honeypot credential
		// (username/password/token) and audit log readers must not exfiltrate trap secrets.
		s.log.Warn("HONEYPOT TRIGGERED: id=%s type=%s", decoy.ID, decoy.Type)
	}
}

// GetDecoyStatus returns all active decoys and their state.
func (s *HoneypotService) GetDecoyStatus() []HoneypotDecoy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []HoneypotDecoy
	for _, d := range s.decoys {
		list = append(list, d)
	}
	return list
}
