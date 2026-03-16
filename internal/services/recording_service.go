package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/sharing"
)

// RecordingService exposes recording features to frontend over Wails
type RecordingService struct {
	BaseService
	ctx     context.Context
	manager sharing.RecordingProvider
	bus     *eventbus.Bus
	log     *logger.Logger
}

// NewRecordingService bounds the recording manager to the app context
func (s *RecordingService) Name() string { return "recording-service" }

// Dependencies returns service dependencies.
func (s *RecordingService) Dependencies() []string {
	return []string{}
}

func NewRecordingService(provider sharing.RecordingProvider, bus *eventbus.Bus, log *logger.Logger) *RecordingService {
	return &RecordingService{
		manager: provider,
		bus:     bus,
		log:     log.WithPrefix("recording_service"),
	}
}

func (s *RecordingService) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *RecordingService) Stop(ctx context.Context) error {
	return nil
}

// StartRecording starts a new recording
func (s *RecordingService) StartRecording(sessionID, hostLabel string, cols, rows int) error {
	if s.manager == nil {
		return fmt.Errorf("recording manager not initialized")
	}
	_, err := s.manager.StartRecording(sessionID, hostLabel, cols, rows)
	if err != nil {
		s.log.Error("Failed to start recording for session %s: %v", sessionID, err)
		return err
	}
	s.log.Info("Started recording for session %s", sessionID)
	s.bus.Publish("recording:started", sessionID)
	return nil
}

// StopRecording finishes the active recording
func (s *RecordingService) StopRecording(sessionID string) (*sharing.RecordingMetadata, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("recording manager not initialized")
	}
	meta, err := s.manager.StopRecording(sessionID)
	if err != nil {
		s.log.Error("Failed to stop recording for session %s: %v", sessionID, err)
		return nil, err
	}

	s.log.Info("Stopped recording %s (Duration: %.2fs)", meta.ID, meta.Duration)
	s.bus.Publish("recording:stopped", sessionID)
	return meta, nil
}

// ListRecordings fetches all completed local recordings
func (s *RecordingService) ListRecordings() ([]sharing.RecordingMetadata, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("recording manager not initialized")
	}
	return s.manager.ListRecordings()
}

// DeleteRecording removes a recorded session
func (s *RecordingService) DeleteRecording(id string) error {
	if s.manager == nil {
		return fmt.Errorf("recording manager not initialized")
	}
	return s.manager.DeleteRecording(id)
}

// GetRecordingFrames retrieves all frames for a specific recording
func (s *RecordingService) GetRecordingFrames(id string) ([]map[string]interface{}, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("recording manager not initialized")
	}
	return s.manager.GetRecordingFrames(id)
}

// SearchRecordings executes a forensic search across all sessions
func (s *RecordingService) SearchRecordings(query string) ([]map[string]interface{}, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("recording manager not initialized")
	}
	return s.manager.SearchRecordings(query)
}

// GetRecordingMeta retrieves metadata for a specific recording
func (s *RecordingService) GetRecordingMeta(id string) (map[string]interface{}, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("recording manager not initialized")
	}
	return s.manager.GetRecordingMeta(id)
}

// ExportRecording generates a signed export for a recording
func (s *RecordingService) ExportRecording(id, destPath string) error {
	if s.manager == nil {
		return fmt.Errorf("recording manager not initialized")
	}
	return s.manager.ExportRecording(id, destPath)
}
