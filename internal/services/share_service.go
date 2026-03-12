package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/sharing"
)

// ShareService exposes session sharing functionality to Wails
type ShareService struct {
	BaseService
	ctx          context.Context
	shareManager *sharing.ShareManager
	recManager   sharing.RecordingProvider
	bus          *eventbus.Bus
	log          *logger.Logger
}

func (s *ShareService) Name() string { return "share-service" }

// Dependencies returns service dependencies
func (s *ShareService) Dependencies() []string {
	return []string{"eventbus"}
}

func NewShareService(sm *sharing.ShareManager, rm sharing.RecordingProvider, bus *eventbus.Bus, log *logger.Logger) *ShareService {
	return &ShareService{
		shareManager: sm,
		recManager:   rm,
		bus:          bus,
		log:          log.WithPrefix("share_service"),
	}
}

func (s *ShareService) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *ShareService) Stop(ctx context.Context) error {
	return nil
}

// CreateShare makes a new session link
func (s *ShareService) CreateShare(sessionID, hostLabel, mode, createdBy string, expiresInMinutes, maxViewers int) (string, error) {
	fromMode := sharing.ShareMode(mode)
	if fromMode == "" {
		fromMode = sharing.ShareObserve
	}

	_, link, err := s.shareManager.CreateShare(sessionID, hostLabel, fromMode, createdBy, 0, maxViewers) // TODO correct duration
	if err != nil {
		s.log.Error("Failed to create share for session %s: %v", sessionID, err)
		return "", err
	}

	s.log.Info("Created new session share for %s", hostLabel)
	// We use strings for generic broadcast events that don't need typed bus topics if undefined
	s.bus.Publish("session.shared", map[string]interface{}{
		"session_id": sessionID,
		"link":       link,
	})

	return link, nil
}

// GetActiveShares lists current shared links
func (s *ShareService) GetActiveShares() []sharing.SessionShare {
	return s.shareManager.GetActiveShares()
}

// GetSharesBySession returns shares for a specific session
func (s *ShareService) GetSharesBySession(sessionID string) []sharing.SessionShare {
	return s.shareManager.GetSharesBySession(sessionID)
}

// RevokeShare forcefully kills a shared session
func (s *ShareService) RevokeShare(shareID string) error {
	err := s.shareManager.RevokeShare(shareID)
	if err == nil {
		s.log.Info("Revoked share %s", shareID)
	}
	return err
}

// GetViewers returns connected viewers for a share
func (s *ShareService) GetViewers(shareID string) []sharing.ShareViewer {
	return s.shareManager.GetViewers(shareID)
}

// HandleViewerInput allows viewers with write access to interact with the terminal
func (s *ShareService) HandleViewerInput(shareID, viewerID, token, data string) error {
	return s.shareManager.HandleViewerInput(shareID, viewerID, token, data)
}

// GetTotalViewers returns sum of viewers across all active shares for a session
func (s *ShareService) GetTotalViewers(sessionID string) int {
	return s.shareManager.GetTotalViewersForSession(sessionID)
}
