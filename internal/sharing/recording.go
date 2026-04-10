package sharing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
	"github.com/kingknull/oblivrashell/internal/database"

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
	TenantID   string    `json:"tenant_id"`
	SessionID  string    `json:"session_id"`
	HostLabel  string    `json:"host_label"`
	StartedAt  string    `json:"started_at"`
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
	StartedAt  string    `json:"started_at"`
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
func (m *RecordingManager) StartRecording(tenantID, sessionID, hostLabel string, cols, rows int) (*ActiveRecording, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.New().String()

	recording := &ActiveRecording{
		ID:        id,
		TenantID:  tenantID,
		SessionID: sessionID,
		HostLabel: hostLabel,
		StartedAt: time.Now().Format(time.RFC3339),
		IsActive:  true,
		Cols:      cols,
		Rows:      rows,
	}

	// Save initial metadata for crash recovery
	ctx := database.WithTenant(context.Background(), tenantID)
	err := m.analytics.SaveRecording(ctx, id, sessionID, hostLabel, cols, rows, 0, 0, "in_progress")
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

	timestamp := time.Since(parseTime(recording.StartedAt)).Seconds()
	ctx := database.WithTenant(context.Background(), recording.TenantID)
	m.analytics.IngestFrame(ctx, recording.ID, timestamp, "o", string(data))
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

	timestamp := time.Since(parseTime(recording.StartedAt)).Seconds()
	ctx := database.WithTenant(context.Background(), recording.TenantID)
	m.analytics.IngestFrame(ctx, recording.ID, timestamp, "i", string(data))
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
	duration := time.Since(parseTime(recording.StartedAt)).Seconds()

	// Update final metadata and mark as completed
	ctx := database.WithTenant(context.Background(), recording.TenantID)
	err := m.analytics.SaveRecording(ctx, recording.ID, recording.SessionID, recording.HostLabel, recording.Cols, recording.Rows, duration, recording.EventCount, "completed")
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
func (m *RecordingManager) ListRecordings(tenantID string) ([]RecordingMetadata, error) {
	ctx := database.WithTenant(context.Background(), tenantID)
	metas, err := m.analytics.ListRecordings(ctx)
	if err != nil {
		return nil, err
	}

	var recordings []RecordingMetadata
	for _, raw := range metas {
		recordings = append(recordings, RecordingMetadata{
			ID:         raw["id"].(string),
			SessionID:  raw["session_id"].(string),
			HostLabel:  raw["host_label"].(string),
			StartedAt:  raw["started_at"].(string),
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
func (m *RecordingManager) DeleteRecording(tenantID, id string) error {
	ctx := database.WithTenant(context.Background(), tenantID)
	return m.analytics.DeleteRecording(ctx, id)
}

// GetRecordingFrames retrieves all frames for a recording ordered by timestamp
func (m *RecordingManager) GetRecordingFrames(tenantID, id string) ([]map[string]interface{}, error) {
	ctx := database.WithTenant(context.Background(), tenantID)
	return m.analytics.GetRecordingFrames(ctx, id)
}

// SearchRecordings executes a forensic search across all sessions
func (m *RecordingManager) SearchRecordings(tenantID, query string) ([]map[string]interface{}, error) {
	ctx := database.WithTenant(context.Background(), tenantID)
	return m.analytics.SearchRecordings(ctx, query)
}

// ExportRecording generates a signed Asciinema v2 file for compliance
func (m *RecordingManager) ExportRecording(tenantID, id, destPath string) error {
	ctx := database.WithTenant(context.Background(), tenantID)
	meta, err := m.analytics.GetRecordingMeta(ctx, id)
	if err != nil {
		return fmt.Errorf("get meta: %w", err)
	}

	frames, err := m.analytics.GetRecordingFrames(ctx, id)
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
func (m *RecordingManager) GetRecordingMeta(tenantID, id string) (map[string]interface{}, error) {
	ctx := database.WithTenant(context.Background(), tenantID)
	return m.analytics.GetRecordingMeta(ctx, id)
}


