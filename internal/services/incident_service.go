package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// IncidentService handles the lifecycle of security incidents and forensic cases.
type IncidentService struct {
	repo     database.IncidentStore
	audit    database.AuditStore
	evidence database.EvidenceStore
	bus      *eventbus.Bus
	log      *logger.Logger
}

// NewIncidentService creates a new incident management service.
func NewIncidentService(
	repo database.IncidentStore,
	audit database.AuditStore,
	evidence database.EvidenceStore,
	bus *eventbus.Bus,
	log *logger.Logger,
) *IncidentService {
	return &IncidentService{
		repo:     repo,
		audit:    audit,
		evidence: evidence,
		bus:      bus,
		log:      log,
	}
}

func (s *IncidentService) Name() string { return "incident-service" }

// Dependencies returns service dependencies.
func (s *IncidentService) Dependencies() []string {
	return []string{}
}

func (s *IncidentService) Start(ctx context.Context) error {
	s.log.Info("Incident management service starting...")
	return nil
}

func (s *IncidentService) Stop(ctx context.Context) error {
	s.log.Info("Incident management service shutting down...")
	return nil
}

// ListIncidents retrieves incidents with optional filtering.
func (s *IncidentService) ListIncidents(ctx context.Context, status string, owner string, limit int) ([]database.Incident, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.Search(ctx, status, owner, limit)
}

// GetIncident retrieves a specific case by ID.
func (s *IncidentService) GetIncident(ctx context.Context, id string) (*database.Incident, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateIncidentStatus changes the workflow state of a case.
func (s *IncidentService) UpdateIncidentStatus(ctx context.Context, id string, status string, reason string) error {
	s.log.Info("[INCIDENT] Updating status for %s to %s", id, status)
	err := s.repo.UpdateStatus(ctx, id, status, reason)
	if err == nil {
		s.bus.Publish("incident.updated", map[string]interface{}{
			"id":     id,
			"status": status,
		})
	}
	return err
}

// AssignIncident assigns a case to a specific analyst/owner.
func (s *IncidentService) AssignIncident(ctx context.Context, id string, owner string) error {
	s.log.Info("[INCIDENT] Assigning %s to %s", id, owner)
	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	inc.Owner = owner
	return s.repo.Upsert(ctx, inc)
}

// GetTimeline reconstructs the event sequence for an incident.
// For now, it returns the most recent audit logs.
// In a follow-up, we can add specific tag filtering to AuditStore.
func (s *IncidentService) GetTimeline(ctx context.Context, incidentID string) ([]database.AuditLog, error) {
	return s.audit.GetRecent(context.Background(), 100)
}

// GetEvidence retrieves all forensic artifacts linked to an incident.
func (s *IncidentService) GetEvidence(ctx context.Context, incidentID string) ([]database.EvidenceItem, error) {
	return s.evidence.ListByIncident(context.Background(), incidentID)
}

// GetByRuleAndGroup proxy for detection engine.
func (s *IncidentService) GetByRuleAndGroup(ctx context.Context, ruleID string, groupKey string) (*database.Incident, error) {
	return s.repo.GetByRuleAndGroup(ctx, ruleID, groupKey)
}

// Upsert proxy for incident storage.
func (s *IncidentService) Upsert(ctx context.Context, incident *database.Incident) error {
	return s.repo.Upsert(ctx, incident)
}

// Search proxy for backward compatibility with AlertingService.
func (s *IncidentService) Search(ctx context.Context, status string, owner string, limit int) ([]database.Incident, error) {
	return s.repo.Search(ctx, status, owner, limit)
}

// UpdateStatus proxy for backward compatibility with AlertingService.
func (s *IncidentService) UpdateStatus(ctx context.Context, id string, status string, reason string) error {
	return s.UpdateIncidentStatus(ctx, id, status, reason)
}
