package app

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

func (s *RansomwareService) Name() string { return "RansomwareService" }

func (s *RansomwareService) Startup(ctx context.Context) {
	s.log.Info("Ransomware response service started")
}

func (s *RansomwareService) Shutdown() {}

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
func (s *RansomwareService) IsolateHost(hostID, reason string) error {
	if s.isolator == nil {
		return fmt.Errorf("isolator not available")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.isolator.IsolateHost(ctx, hostID, reason, 0, false)
}

// RestoreHost removes isolation rules from a host (analyst action after validation).
func (s *RansomwareService) RestoreHost(hostID string) error {
	if s.isolator == nil {
		return fmt.Errorf("isolator not available")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.isolator.RestoreHost(ctx, hostID)
}
