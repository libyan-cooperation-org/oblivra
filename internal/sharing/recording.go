package sharing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/vault"
)

// RecordingManager manages TTY recordings via AnalyticsEngine
type RecordingManager struct {
	mu         sync.RWMutex
	analytics  analytics.Engine
	vault      vault.Provider
	recordings map[string]*ActiveRecording
}

// ActiveRecording represents a currently running recording
type ActiveRecording struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	HostLabel  string    `json:"host_label"`
	StartedAt  time.Time `json:"started_at"`
	EventCount int       `json:"event_count"`
	IsActive   bool      `json:"is_active"`
	Cols       int       `json:"cols"`
	Rows       int       `json:"rows"`
}

// RecordingMetadata represents completed recording info
type RecordingMetadata struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	HostLabel  string    `json:"host_label"`
	StartedAt  time.Time `json:"started_at"`
	Duration   float64   `json:"duration"` // seconds
	EventCount int       `json:"event_count"`
	Cols       int       `json:"cols"`
	Rows       int       `json:"rows"`
	Status     string    `json:"status"`
}

func NewRecordingManager(ana analytics.Engine, v vault.Provider) *RecordingManager {
	return &RecordingManager{
		recordings: make(map[string]*ActiveRecording),
		analytics:  ana,
		vault:      v,
	}
}

// StartRecording begins a new session recording in the analytics store
func (m *RecordingManager) StartRecording(sessionID, hostLabel string, cols, rows int) (*ActiveRecording, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.New().String()

	recording := &ActiveRecording{
		ID:        id,
		SessionID: sessionID,
		HostLabel: hostLabel,
		StartedAt: time.Now(),
		IsActive:  true,
		Cols:      cols,
		Rows:      rows,
	}

	// Save initial metadata for crash recovery
	err := m.analytics.SaveRecording(id, sessionID, hostLabel, cols, rows, 0, 0, "in_progress")
	if err != nil {
		return nil, fmt.Errorf("pre-create recording: %w", err)
	}

	m.recordings[sessionID] = recording

	return recording, nil
}

// RecordOutput logs stdout/stderr to the analytics store
func (m *RecordingManager) RecordOutput(sessionID string, data []byte) {
	m.mu.RLock()
	recording, ok := m.recordings[sessionID]
	m.mu.RUnlock()

	if !ok || !recording.IsActive {
		return
	}

	timestamp := time.Since(recording.StartedAt).Seconds()
	m.analytics.IngestFrame(recording.ID, timestamp, "o", string(data))
	recording.EventCount++
}

// RecordInput logs stdin to the analytics store
func (m *RecordingManager) RecordInput(sessionID string, data []byte) {
	m.mu.RLock()
	recording, ok := m.recordings[sessionID]
	m.mu.RUnlock()

	if !ok || !recording.IsActive {
		return
	}

	timestamp := time.Since(recording.StartedAt).Seconds()
	m.analytics.IngestFrame(recording.ID, timestamp, "i", string(data))
	recording.EventCount++
}

// StopRecording finishes the recording and updates metadata
func (m *RecordingManager) StopRecording(sessionID string) (*RecordingMetadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	recording, ok := m.recordings[sessionID]
	if !ok {
		return nil, fmt.Errorf("no active recording for session")
	}

	if !recording.IsActive {
		return nil, fmt.Errorf("recording already stopped")
	}

	recording.IsActive = false
	duration := time.Since(recording.StartedAt).Seconds()

	// Update final metadata and mark as completed
	err := m.analytics.SaveRecording(recording.ID, recording.SessionID, recording.HostLabel, recording.Cols, recording.Rows, duration, recording.EventCount, "completed")
	if err != nil {
		return nil, fmt.Errorf("finalize recording: %w", err)
	}

	metadata := &RecordingMetadata{
		ID:         recording.ID,
		SessionID:  recording.SessionID,
		HostLabel:  recording.HostLabel,
		StartedAt:  recording.StartedAt,
		Duration:   duration,
		EventCount: recording.EventCount,
		Cols:       recording.Cols,
		Rows:       recording.Rows,
		Status:     "completed",
	}

	// Remove from active map
	delete(m.recordings, sessionID)

	return metadata, nil
}

// ListRecordings returns all saved recordings from the analytics store
func (m *RecordingManager) ListRecordings() ([]RecordingMetadata, error) {
	metas, err := m.analytics.ListRecordings()
	if err != nil {
		return nil, err
	}

	var recordings []RecordingMetadata
	for _, raw := range metas {
		recordings = append(recordings, RecordingMetadata{
			ID:         raw["id"].(string),
			SessionID:  raw["session_id"].(string),
			HostLabel:  raw["host_label"].(string),
			StartedAt:  parseTime(raw["started_at"].(string)),
			Duration:   raw["duration"].(float64),
			EventCount: int(raw["event_count"].(int64)),
			Cols:       int(raw["cols"].(int64)),
			Rows:       int(raw["rows"].(int64)),
			Status:     raw["status"].(string),
		})
	}

	return recordings, nil
}

// DeleteRecording removes a recording and all its frames from the analytics store
func (m *RecordingManager) DeleteRecording(id string) error {
	return m.analytics.DeleteRecording(id)
}

// GetRecordingFrames retrieves all frames for a recording ordered by timestamp
func (m *RecordingManager) GetRecordingFrames(id string) ([]map[string]interface{}, error) {
	return m.analytics.GetRecordingFrames(id)
}

// SearchRecordings executes a forensic search across all sessions
func (m *RecordingManager) SearchRecordings(query string) ([]map[string]interface{}, error) {
	return m.analytics.SearchRecordings(query)
}

// ExportRecording generates a signed Asciinema v2 file for compliance
func (m *RecordingManager) ExportRecording(id, destPath string) error {
	meta, err := m.analytics.GetRecordingMeta(id)
	if err != nil {
		return fmt.Errorf("get meta: %w", err)
	}

	frames, err := m.analytics.GetRecordingFrames(id)
	if err != nil {
		return fmt.Errorf("get frames: %w", err)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// 1. Write Header
	header := map[string]interface{}{
		"version":   2,
		"width":     meta["cols"],
		"height":    meta["rows"],
		"timestamp": parseTime(meta["started_at"].(string)).Unix(),
		"env": map[string]string{
			"TERM": "xterm-256color",
		},
		"title": fmt.Sprintf("OblivraShell Audit - %s", meta["host_label"]),
	}
	headerJSON, _ := json.Marshal(header)
	file.Write(headerJSON)
	file.WriteString("\n")

	// 2. Write Frames
	for _, frame := range frames {
		// [time, type, data]
		event := []interface{}{frame["timestamp"], frame["type"], frame["data"]}
		eventJSON, _ := json.Marshal(event)
		file.Write(eventJSON)
		file.WriteString("\n")
	}

	// 3. Add Signature (HMAC-SHA256)
	if m.vault != nil && m.vault.IsUnlocked() {
		// We use the master key to sign the export
		err = m.vault.AccessMasterKey(func(key []byte) error {
			h := hmac.New(sha256.New, key)

			// Sign everything written so far
			if _, err := file.Seek(0, 0); err != nil {
				return err
			}
			data, err := os.ReadFile(destPath)
			if err != nil {
				return err
			}
			h.Write(data)
			sig := hex.EncodeToString(h.Sum(nil))

			// Append signature as a trailer
			sigBlock := map[string]string{
				"signature": sig,
				"method":    "HMAC-SHA256",
				"signer":    "OblivraShell-Vault",
			}
			sigJSON, _ := json.Marshal(sigBlock)
			file.Seek(0, 2) // End of file
			file.WriteString("\n")
			_, err = file.Write(sigJSON)
			return err
		})
		if err != nil {
			return fmt.Errorf("sign recording: %w", err)
		}
	}

	return nil
}

// GetRecordingMeta retrieves metadata for a specific recording
func (m *RecordingManager) GetRecordingMeta(id string) (map[string]interface{}, error) {
	return m.analytics.GetRecordingMeta(id)
}

func parseTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	if t.IsZero() {
		t, _ = time.Parse("2006-01-02 15:04:05", ts)
	}
	return t
}
