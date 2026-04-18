package ssh

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
)

// SessionStatus represents session lifecycle
type SessionStatus string

const (
	SessionActive SessionStatus = "active"
	SessionClosed SessionStatus = "closed"
	SessionError  SessionStatus = "error"
)

// Session wraps an SSH client with metadata
type Session struct {
	ID        string        `json:"id"`
	HostID    string        `json:"host_id"`
	HostLabel string        `json:"host_label"`
	Status    SessionStatus `json:"status"`
	StartedAt string        `json:"started_at"`
	EndedAt   *string       `json:"ended_at,omitempty"`

	client     *Client
	sftpClient *sftp.Client
	mu         sync.RWMutex
	onData     func(sessionID string, data []byte)
	onClose    func(sessionID string)
	onError    func(sessionID string, err error)

	Ctx    context.Context
	cancel context.CancelFunc
}

// NewSession creates a new managed session with a random UUID
func NewSession(hostID string, hostLabel string, cfg ConnectionConfig) *Session {
	return NewSessionWithID(uuid.New().String(), hostID, hostLabel, cfg)
}

// NewSessionWithID creates a new managed session with a specific ID (e.g. for SOC pivoting)
func NewSessionWithID(id string, hostID string, hostLabel string, cfg ConnectionConfig) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		ID:        id,
		HostID:    hostID,
		HostLabel: hostLabel,
		Status:    SessionActive,
		StartedAt: time.Now().Format(time.RFC3339),
		client:    NewClient(cfg),
		Ctx:       ctx,
		cancel:    cancel,
	}
}

// SetCallbacks sets event callbacks
func (s *Session) SetCallbacks(
	onData func(string, []byte),
	onClose func(string),
	onError func(string, error),
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onData = onData
	s.onClose = onClose
	s.onError = onError
}

// GetClient returns the underlying ssh client
func (s *Session) GetClient() *Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

// GetSftpClient returns (and initializes if needed) the sftp client
func (s *Session) GetSftpClient() (*sftp.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sftpClient != nil {
		return s.sftpClient, nil
	}

	if s.client == nil || s.client.RawClient() == nil {
		return nil, fmt.Errorf("ssh client not connected")
	}

	sc, err := sftp.NewClient(s.client.RawClient())
	if err != nil {
		return nil, fmt.Errorf("create sftp client: %w", err)
	}

	s.sftpClient = sc
	return s.sftpClient, nil
}

// Start connects and starts the shell
func (s *Session) Start() error {
	if err := s.client.Connect(); err != nil {
		s.setStatus(SessionError)
		return fmt.Errorf("connect: %w", err)
	}

	if err := s.client.StartShell(); err != nil {
		s.client.Close()
		s.setStatus(SessionError)
		return fmt.Errorf("start shell: %w", err)
	}

	go s.processEvents()
	return nil
}

// processEvents handles SSH client events
func (s *Session) processEvents() {
	for event := range s.client.Events() {
		switch event.Type {
		case EventDataReceived:
			s.mu.RLock()
			cb := s.onData
			s.mu.RUnlock()
			if cb != nil {
				cb(s.ID, event.Data)
			}

		case EventError:
			s.mu.RLock()
			cb := s.onError
			s.mu.RUnlock()
			if cb != nil && event.Error != nil {
				cb(s.ID, event.Error)
			}

		case EventClosed:
			s.setStatus(SessionClosed)
			s.mu.RLock()
			cb := s.onClose
			s.mu.RUnlock()
			if cb != nil {
				cb(s.ID)
			}
			return
		}
	}
}

// Write sends input to the session
func (s *Session) Write(data []byte) error {
	_, err := s.client.Write(data)
	return err
}

// Resize changes the terminal dimensions
func (s *Session) Resize(cols, rows int) error {
	return s.client.Resize(cols, rows)
}

// Close terminates the session
func (s *Session) Close() error {
	s.cancel()

	s.mu.Lock()
	if s.sftpClient != nil {
		s.sftpClient.Close()
		s.sftpClient = nil
	}
	s.mu.Unlock()

	s.setStatus(SessionClosed)
	return s.client.Close()
}

// GetStatus returns current status
func (s *Session) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// Metrics returns session metrics
func (s *Session) Metrics() (bytesIn int64, bytesOut int64, uptime time.Duration) {
	return s.client.Metrics()
}

// setStatus updates the session status
func (s *Session) setStatus(status SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	if status == SessionClosed || status == SessionError {
		now := time.Now().Format(time.RFC3339)
		s.EndedAt = &now
	}
}
