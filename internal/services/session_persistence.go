package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// SessionPersistence saves and restores active terminal sessions across app restarts.
// On graceful shutdown, it snapshots all active session host IDs and tab order.
// On startup, it provides the saved state for re-connection.
type SessionPersistence struct {
	filePath string
	log      *logger.Logger
	mu       sync.Mutex
}

// PersistedSession represents a saved session that can be restored.
type PersistedSession struct {
	HostID    string `json:"host_id"`
	HostLabel string `json:"host_label"`
	TabIndex  int    `json:"tab_index"`
	IsLocal   bool   `json:"is_local"`
}

// PersistedState is the full snapshot saved to disk.
type PersistedState struct {
	Sessions     []PersistedSession `json:"sessions"`
	ActiveTabIdx int                `json:"active_tab_index"`
	SavedAt      string             `json:"saved_at"`
	AppVersion   string             `json:"app_version"`
}

// NewSessionPersistence creates a new session persistence manager.
// State is saved to ~/.oblivrashell/session_state.json.
func NewSessionPersistence(log *logger.Logger) *SessionPersistence {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".oblivrashell")
	os.MkdirAll(dir, 0700)

	return &SessionPersistence{
		filePath: filepath.Join(dir, "session_state.json"),
		log:      log.WithPrefix("session-persist"),
	}
}

// SaveState persists the current active sessions to disk.
// Called on graceful app shutdown (SSHService.Stop).
func (sp *SessionPersistence) SaveState(sessions []PersistedSession, activeTabIdx int) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	state := PersistedState{
		Sessions:     sessions,
		ActiveTabIdx: activeTabIdx,
		SavedAt:      time.Now().Format(time.RFC3339),
		AppVersion:   "1.0.0",
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.WriteFile(sp.filePath, data, 0600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	sp.log.Info("[PERSIST] Saved %d sessions to %s", len(sessions), sp.filePath)
	return nil
}

// LoadState reads the persisted session state from disk.
// Returns nil state if no state file exists (first launch).
func (sp *SessionPersistence) LoadState() (*PersistedState, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	data, err := os.ReadFile(sp.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // First launch — no state to restore
		}
		return nil, fmt.Errorf("read state: %w", err)
	}

	var state PersistedState
	if err := json.Unmarshal(data, &state); err != nil {
		sp.log.Warn("[PERSIST] Corrupt state file, ignoring: %v", err)
		return nil, nil
	}

	// Don't restore if state is older than 24 hours
	savedAt, err := time.Parse(time.RFC3339, state.SavedAt)
	if err == nil && time.Since(savedAt) > 24*time.Hour {
		sp.log.Info("[PERSIST] State file older than 24h, ignoring")
		return nil, nil
	}

	sp.log.Info("[PERSIST] Loaded %d sessions from %s (saved %s)", len(state.Sessions), sp.filePath, state.SavedAt)
	return &state, nil
}

// ClearState removes the persisted state file (e.g., after successful restore).
func (sp *SessionPersistence) ClearState() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if err := os.Remove(sp.filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// HasState returns true if there's a persisted state file to restore.
func (sp *SessionPersistence) HasState() bool {
	_, err := os.Stat(sp.filePath)
	return err == nil
}

// GetRestorable returns the host IDs and labels that can be restored.
// Used by the frontend to show "Restore previous sessions?" prompt.
func (sp *SessionPersistence) GetRestorable() ([]PersistedSession, error) {
	state, err := sp.LoadState()
	if err != nil || state == nil {
		return nil, err
	}
	return state.Sessions, nil
}
