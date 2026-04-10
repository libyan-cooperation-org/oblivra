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
func (s *RecordingService) StartRecording(ctx context.Context, sessionID, hostLabel string, cols, rows int) error {
	if s.manager == nil {
		return fmt.Errorf("recording manager not initialized")
	}
	tenantID := s.resolveTenant(ctx)
	_, err := s.manager.StartRecording(tenantID, sessionID, hostLabel, cols, rows)
	if err != nil {
		s.log.Error("Failed to start recording for session %s: %v", sessionID, err)
		return err
	}
	s.log.Info("Started recording for session %s (Tenant: %s)", sessionID, tenantID)
	s.bus.Publish("recording:started", sessionID)
	return nil
}

// StopRecording finishes the active recording
func (s *RecordingService) StopRecording(ctx context.Context, sessionID string) (*sharing.RecordingMetadata, error) {
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
func (s *RecordingService) ListRecordings(ctx context.Context) ([]sharing.RecordingMetadata, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("recording manager not initialized")
	}
	tenantID := s.resolveTenant(ctx)
	return s.manager.ListRecordings(tenantID)
}

// DeleteRecording removes a recorded session
func (s *RecordingService) DeleteRecording(ctx context.Context, id string) error {
	if s.manager == nil {
		return fmt.Errorf("recording manager not initialized")
	}
	tenantID := s.resolveTenant(ctx)
	return s.manager.DeleteRecording(tenantID, id)
}

// GetRecordingFrames retrieves all frames for a specific recording
func (s *RecordingService) GetRecordingFrames(ctx context.Context, id string) ([]map[string]interface{}, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("recording manager not initialized")
	}
	tenantID := s.resolveTenant(ctx)
	return s.manager.GetRecordingFrames(tenantID, id)
}

// SearchRecordings executes a forensic search across all sessions
func (s *RecordingService) SearchRecordings(ctx context.Context, query string) ([]map[string]interface{}, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("recording manager not initialized")
	}
	tenantID := s.resolveTenant(ctx)
	return s.manager.SearchRecordings(tenantID, query)
}

// GetRecordingMeta retrieves metadata for a specific recording
func (s *RecordingService) GetRecordingMeta(ctx context.Context, id string) (map[string]interface{}, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("recording manager not initialized")
	}
	tenantID := s.resolveTenant(ctx)
	return s.manager.GetRecordingMeta(tenantID, id)
}

// ExportRecording generates a signed export for a recording
func (s *RecordingService) ExportRecording(ctx context.Context, id, destPath string) error {
	if s.manager == nil {
		return fmt.Errorf("recording manager not initialized")
	}
	tenantID := s.resolveTenant(ctx)
	return s.manager.ExportRecording(tenantID, id, destPath)
}

func (s *RecordingService) resolveTenant(ctx context.Context) string {
	if tenantID, ok := ctx.Value("tenant_id").(string); ok && tenantID != "" {
		return tenantID
	}
	return "default_tenant"
}
