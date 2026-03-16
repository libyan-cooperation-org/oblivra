package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type SessionService struct {
	BaseService
	sessions database.SessionStore
	audit    database.AuditStore
	bus      *eventbus.Bus
	log      *logger.Logger
}

func (s *SessionService) Name() string { return "session-service" }

// Dependencies returns service dependencies.
func (s *SessionService) Dependencies() []string {
	return []string{}
}

func (s *SessionService) Start(ctx context.Context) error {
	return nil
}

func (s *SessionService) Stop(ctx context.Context) error {
	return nil
}

func NewSessionService(s database.SessionStore, a database.AuditStore, bus *eventbus.Bus, log *logger.Logger) *SessionService {
	return &SessionService{sessions: s, audit: a, bus: bus, log: log.WithPrefix("sessions")}
}

func (s *SessionService) GetHistory(limit int) ([]database.Session, error) {
	s.log.Debug("Fetching session history (limit: %d)", limit)
	return s.sessions.GetRecent(context.Background(), limit)
}

func (s *SessionService) GetAuditLogs(limit int) ([]database.AuditLog, error) {
	s.log.Debug("Fetching audit logs (limit: %d)", limit)
	return s.audit.GetRecent(context.Background(), limit)
}

func (s *SessionService) Create(sess database.Session) error {
	s.log.Info("Creating session record: %s (Host: %s)", sess.ID, sess.HostID)
	return s.sessions.Create(context.Background(), &sess)
}

func (s *SessionService) UpdateStatus(id string, status string) error {
	s.log.Info("Updating session %s status to %s", id, status)
	// We use End for status updates for now or we could add a specific UpdateStatus to repo.
	// For hardening, let's just use End with 0 bytes if it's just a status change.
	return s.sessions.End(context.Background(), id, status, 0, 0)
}
