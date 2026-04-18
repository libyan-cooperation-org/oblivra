package ssh

import (
	"fmt"
	"sync"
)

// SessionManager manages all active SSH sessions
type SessionManager struct {
	mu          sync.RWMutex
	sessions    map[string]*Session
	maxSessions int
}

// NewSessionManager creates a new session manager
func NewSessionManager(maxSessions int) *SessionManager {
	if maxSessions <= 0 {
		maxSessions = 50
	}
	return &SessionManager{
		sessions:    make(map[string]*Session),
		maxSessions: maxSessions,
	}
}

// Add registers a new session
func (m *SessionManager) Add(session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sessions) >= m.maxSessions {
		return fmt.Errorf("max sessions (%d) reached", m.maxSessions)
	}
	m.sessions[session.ID] = session
	return nil
}

// Get retrieves a session by ID
func (m *SessionManager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// Remove removes a session
func (m *SessionManager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

// GetAll returns all sessions
func (m *SessionManager) GetAll() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// ActiveCount returns the number of active sessions
func (m *SessionManager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, s := range m.sessions {
		if s.GetStatus() == SessionActive {
			count++
		}
	}
	return count
}

// CloseAll closes all active sessions
func (m *SessionManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, session := range m.sessions {
		session.Close()
		delete(m.sessions, id)
	}
}

// CloseSession closes a specific session
func (m *SessionManager) CloseSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	err := session.Close()
	delete(m.sessions, id)
	return err
}
