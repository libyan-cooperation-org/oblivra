package services

import (
	"context"
	"encoding/base64"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// TailEvent represents a single aggregated log line
type TailEvent struct {
	SessionID string    `json:"session_id"`
	HostLabel string    `json:"host_label"`
	Data      string    `json:"data"`
	Timestamp string    `json:"timestamp"`
}

// TailingService aggregates output from multiple SSH sessions for a unified feed
type TailingService struct {
	mu           *sync.RWMutex
	activeTails  map[string]bool // sessionIDs currently being tailed
	log          *logger.Logger
	bus          *eventbus.Bus
	ctx          context.Context
	onEvent      func(event TailEvent)
}

func NewTailingService(bus *eventbus.Bus, log *logger.Logger) *TailingService {
	return &TailingService{
		mu:          &sync.RWMutex{},
		activeTails: make(map[string]bool),
		log:         log.WithPrefix("tail-svc"),
		bus:         bus,
	}
}

func (s *TailingService) Name() string {
	return "tailing-service"
}

// Dependencies returns service dependencies.
func (s *TailingService) Dependencies() []string {
	return []string{}
}

func (s *TailingService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("Tailing service started")
	return nil
}

func (s *TailingService) Stop(ctx context.Context) error {
	s.StopAll()
	return nil
}

// SetCallback registers a handler for aggregated tail events (e.g. for UI streaming)
func (s *TailingService) SetCallback(cb func(event TailEvent)) {
	s.onEvent = cb
}

// RegisterOutput processes raw output from a session and emits a TailEvent
func (s *TailingService) RegisterOutput(sessionID, hostLabel, data string) {
	s.mu.RLock()
	active := s.activeTails[sessionID]
	s.mu.RUnlock()

	if !active {
		return
	}

	event := TailEvent{
		SessionID: sessionID,
		HostLabel: hostLabel,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if s.onEvent != nil {
		s.onEvent(event)
	}

	// Also emit via Wails if UI is active
	if s.ctx != nil {
		encoded := base64.StdEncoding.EncodeToString([]byte(data))
		EmitEvent(s.ctx, "tail:update", map[string]interface{}{
			"session_id": sessionID,
			"host_label": hostLabel,
			"data":       encoded,
			"timestamp":  event.Timestamp,
		})
	}
}

// StartTailing adds a session to the aggregation feed
func (s *TailingService) StartTailing(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeTails[sessionID] = true
	s.log.Info("Started tailing session: %s", sessionID)
}

// StopTailing removes a session from the aggregation feed
func (s *TailingService) StopTailing(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeTails, sessionID)
	s.log.Info("Stopped tailing session: %s", sessionID)
}

// StopAll clears the tailing list
func (s *TailingService) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeTails = make(map[string]bool)
	s.log.Info("Stopped all tailing sessions")
}
