package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/isolation"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// RansomwareService exposes ransomware defense controls to the Wails frontend.
type RansomwareService struct {
	BaseService
	isolator *isolation.NetworkIsolator
	bus      *eventbus.Bus
	log      *logger.Logger
}

func NewRansomwareService(
	isolator *isolation.NetworkIsolator,
	bus *eventbus.Bus,
	log *logger.Logger,
) *RansomwareService {
	return &RansomwareService{
		isolator: isolator,
		bus:      bus,
		log:      log.WithPrefix("ransomware_svc"),
	}
}

func (s *RansomwareService) Name() string { return "ransomware-service" }

// Dependencies returns service dependencies.
func (s *RansomwareService) Dependencies() []string {
	return []string{}
}

func (s *RansomwareService) Start(ctx context.Context) error {
	s.log.Info("Ransomware response service started")
	return nil
}

func (s *RansomwareService) Stop(ctx context.Context) error {
	return nil
}

// IsolationRecordDTO is the frontend-safe view of an isolation record.
type IsolationRecordDTO struct {
	HostID      string  `json:"host_id"`
	IsolatedAt  string  `json:"isolated_at"`
	Reason      string  `json:"reason"`
	ThreatScore int     `json:"threat_score"`
	Auto        bool    `json:"auto"`
	Restored    bool    `json:"restored"`
	RestoredAt  *string `json:"restored_at,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// ListIsolations returns all current and historical isolation records.
func (s *RansomwareService) ListIsolations() []IsolationRecordDTO {
	if s.isolator == nil {
		return []IsolationRecordDTO{}
	}
	records := s.isolator.ListIsolations()
	dtos := make([]IsolationRecordDTO, 0, len(records))
	for _, r := range records {
		dto := IsolationRecordDTO{
			HostID:      r.HostID,
			IsolatedAt:  r.IsolatedAt,
			Reason:      r.Reason,
			ThreatScore: r.ThreatScore,
			Auto:        r.Auto,
			Restored:    r.Restored,
			Error:       r.Error,
		}
		if r.RestoredAt != nil {
			dto.RestoredAt = r.RestoredAt
		}
		dtos = append(dtos, dto)
	}
	return dtos
}

// IsolateHost manually triggers network isolation for a host (analyst action).
// `parentCtx` carries the operator's request context (Wails injects it as
// the first arg automatically); we still cap with a 30s deadline for the
// actual isolator call so a network hang doesn't pin the request goroutine.
func (s *RansomwareService) IsolateHost(parentCtx context.Context, hostID, reason string) error {
	if s.isolator == nil {
		return fmt.Errorf("isolator not available")
	}
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()
	return s.isolator.IsolateHost(ctx, hostID, reason, 0, false)
}

// RestoreHost removes isolation rules from a host (analyst action after validation).
func (s *RansomwareService) RestoreHost(parentCtx context.Context, hostID string) error {
	if s.isolator == nil {
		return fmt.Errorf("isolator not available")
	}
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()
	return s.isolator.RestoreHost(ctx, hostID)
}
