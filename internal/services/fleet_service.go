package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
)

type AgentRegistration struct {
	Hostname          string   `json:"hostname"`
	OS                string   `json:"os"`
	Arch              string   `json:"arch"`
	Version           string   `json:"version"`
	Tags              []string `json:"tags,omitempty"`
	PubKeyB64         string   `json:"pubkeyB64,omitempty"`
	PubKeyFingerprint string   `json:"pubkeyFingerprint,omitempty"`
}

// HeartbeatStats is the rich self-report the agent posts on a schedule.
// Fields all optional — older agents that don't populate them simply
// leave the fleet record's prior values untouched.
type HeartbeatStats struct {
	AgentID           string `json:"agentId"`
	Version           string `json:"version,omitempty"`
	PubKeyFingerprint string `json:"pubkeyFingerprint,omitempty"`
	InputCount        int    `json:"inputCount,omitempty"`
	SpillFiles        int    `json:"spillFiles,omitempty"`
	SpillBytes        int64  `json:"spillBytes,omitempty"`
	QueueDepth        int    `json:"queueDepth,omitempty"`
	DroppedEvents     int64  `json:"droppedEvents,omitempty"`
	BatchSize         int    `json:"batchSize,omitempty"`
}

type Agent struct {
	ID                string    `json:"id"`
	Token             string    `json:"token"`
	Hostname          string    `json:"hostname"`
	OS                string    `json:"os"`
	Arch              string    `json:"arch"`
	Version           string    `json:"version"`
	Tags              []string  `json:"tags,omitempty"`
	Registered        time.Time `json:"registered"`
	LastSeen          time.Time `json:"lastSeen"`
	Events            int64     `json:"events"`

	// Rich state — populated from the agent's heartbeat. The operator
	// uses PubKeyB64 to seed OBLIVRA_AGENT_PUBKEYS, and the per-agent
	// telemetry surfaces in the fleet detail panel so backpressure
	// (SpillBytes high, DroppedEvents > 0) is visible at a glance.
	PubKeyB64         string `json:"pubkeyB64,omitempty"`
	PubKeyFingerprint string `json:"pubkeyFingerprint,omitempty"`
	InputCount        int    `json:"inputCount,omitempty"`
	SpillFiles        int    `json:"spillFiles,omitempty"`
	SpillBytes        int64  `json:"spillBytes,omitempty"`
	QueueDepth        int    `json:"queueDepth,omitempty"`
	DroppedEvents     int64  `json:"droppedEvents,omitempty"`
	BatchSize         int    `json:"batchSize,omitempty"`
}

type FleetService struct {
	log      *slog.Logger
	pipeline *ingest.Pipeline
	mu       sync.RWMutex
	agents   map[string]*Agent
}

func NewFleetService(log *slog.Logger, p *ingest.Pipeline) *FleetService {
	return &FleetService{log: log, pipeline: p, agents: map[string]*Agent{}}
}

func (s *FleetService) ServiceName() string { return "FleetService" }

func (s *FleetService) Register(req AgentRegistration) (*Agent, error) {
	if req.Hostname == "" {
		return nil, errors.New("hostname required")
	}
	id := randomID(8)
	tok := randomID(16)
	a := &Agent{
		ID:                id,
		Token:             tok,
		Hostname:          req.Hostname,
		OS:                req.OS,
		Arch:              req.Arch,
		Version:           req.Version,
		Tags:              req.Tags,
		PubKeyB64:         req.PubKeyB64,
		PubKeyFingerprint: req.PubKeyFingerprint,
		Registered:        time.Now().UTC(),
		LastSeen:          time.Now().UTC(),
	}
	s.mu.Lock()
	s.agents[id] = a
	s.mu.Unlock()
	return a, nil
}

// Heartbeat records a rich self-report from the agent. Updates LastSeen
// and the telemetry fields; leaves identity columns alone so a typo'd
// heartbeat can't rename a host.
func (s *FleetService) Heartbeat(stats HeartbeatStats) error {
	if stats.AgentID == "" {
		return errors.New("agentId required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[stats.AgentID]
	if !ok {
		return errors.New("unknown agent")
	}
	a.LastSeen = time.Now().UTC()
	if stats.Version != "" {
		a.Version = stats.Version
	}
	if stats.PubKeyFingerprint != "" {
		a.PubKeyFingerprint = stats.PubKeyFingerprint
	}
	if stats.InputCount > 0 {
		a.InputCount = stats.InputCount
	}
	a.SpillFiles = stats.SpillFiles
	a.SpillBytes = stats.SpillBytes
	a.QueueDepth = stats.QueueDepth
	a.DroppedEvents = stats.DroppedEvents
	if stats.BatchSize > 0 {
		a.BatchSize = stats.BatchSize
	}
	return nil
}

// Get returns a single agent (or false). Drives the per-agent detail
// panel in the Fleet view.
func (s *FleetService) Get(id string) (Agent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.agents[id]
	if !ok {
		return Agent{}, false
	}
	return *a, true
}

func (s *FleetService) IngestFromAgent(ctx context.Context, agentID string, batch []events.Event) (int, error) {
	if agentID == "" {
		return 0, errors.New("agentId required")
	}
	s.mu.Lock()
	a, ok := s.agents[agentID]
	if !ok {
		s.mu.Unlock()
		return 0, errors.New("unknown agent")
	}
	a.LastSeen = time.Now().UTC()
	s.mu.Unlock()

	written := 0
	for i := range batch {
		batch[i].Source = events.SourceAgent
		if batch[i].HostID == "" {
			batch[i].HostID = a.Hostname
		}
		batch[i].Provenance.IngestPath = "agent"
		batch[i].Provenance.AgentID = agentID
		if err := s.pipeline.Submit(ctx, &batch[i]); err != nil {
			return written, err
		}
		written++
	}
	s.mu.Lock()
	a.Events += int64(written)
	s.mu.Unlock()
	return written, nil
}

func (s *FleetService) List() []Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Agent, 0, len(s.agents))
	for _, a := range s.agents {
		out = append(out, *a)
	}
	return out
}

func randomID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
