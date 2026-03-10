package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// BroadcastService manages real-time input duplication across sessions
type BroadcastService struct {
	BaseService
	executors []SessionExecutor
	log       *logger.Logger
	mu        *sync.RWMutex
	groups    map[string][]string // Map of groupID to sessionIDs
}

func (s *BroadcastService) Name() string { return "BroadcastService" }

func NewBroadcastService(executors []SessionExecutor, log *logger.Logger) *BroadcastService {
	return &BroadcastService{
		executors: executors,
		log:       log.WithPrefix("broadcast"),
		mu:        &sync.RWMutex{},
		groups:    make(map[string][]string),
	}
}

func (s *BroadcastService) Startup(ctx context.Context) {}

// SendInput implements sharing.SessionExecutor
func (s *BroadcastService) SendInput(sessionID, data string) error {
	return s.BroadcastInput([]string{sessionID}, data)
}

// BroadcastInput sends the same input to all specified session IDs concurrently
func (s *BroadcastService) BroadcastInput(sessionIDs []string, inputBase64 string) error {
	if len(sessionIDs) == 0 {
		return nil
	}

	s.log.Debug("Broadcasting input to %d sessions", len(sessionIDs))

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, id := range sessionIDs {
		wg.Add(1)
		go func(sid string) {
			defer wg.Done()

			var success bool
			var lastErr error

			for _, exec := range s.executors {
				if err := exec.SendInput(sid, inputBase64); err == nil {
					success = true
					break
				} else {
					lastErr = err
				}
			}

			if !success {
				mu.Lock()
				if lastErr != nil {
					errs = append(errs, lastErr)
				} else {
					errs = append(errs, fmt.Errorf("no executor found for session %s", sid))
				}
				mu.Unlock()
				s.log.Error("Failed to broadcast to session %s", sid)
			}
		}(id)
	}

	// Use a helper goroutine to close the wait group for the timeout check
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All finished
	case <-time.After(5 * time.Second):
		return fmt.Errorf("broadcast timeout: some sessions failed to respond")
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to broadcast to %d sessions", len(errs))
	}

	return nil
}

// CreateGroup establishes a named group of sessions for broadcasting
func (s *BroadcastService) CreateGroup(name string, sessionIDs []string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups[name] = sessionIDs
	return name
}

// GetGroup returns the session IDs in a group
func (s *BroadcastService) GetGroup(name string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.groups[name]
}

// RemoveGroup deletes a broadcast group
func (s *BroadcastService) RemoveGroup(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.groups, name)
}
