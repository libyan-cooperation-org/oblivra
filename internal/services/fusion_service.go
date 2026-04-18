package services

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// FusionService bridges the AttackFusionEngine with the Wails frontend.
type FusionService struct {
	engine    *detection.AttackFusionEngine
	analytics *analytics.AnalyticsEngine
	bus       *eventbus.Bus
	log       *logger.Logger
	mu        sync.RWMutex
}

// NewFusionService creates a new fusion bridge service.
func NewFusionService(engine *detection.AttackFusionEngine, analytics *analytics.AnalyticsEngine, bus *eventbus.Bus, log *logger.Logger) *FusionService {
	return &FusionService{
		engine:    engine,
		analytics: analytics,
		bus:       bus,
		log:       log,
	}
}

// Name returns the service identifier.
func (s *FusionService) Name() string {
	return "FusionService"
}

// GetActiveCampaigns returns all currently tracked attack campaigns.
func (s *FusionService) GetActiveCampaigns() []detection.Campaign {
	// We need to reach into the engine's LRU. 
	// Since Campaign is usually internal, we'll need an Export method in engine 
	// or just return the slice if engine provides it.
	return s.engine.GetActiveCampaigns()
}

// GetCampaign returns details for a specific entity.
func (s *FusionService) GetCampaign(entityID string) *detection.Campaign {
	return s.engine.GetCampaign(entityID)
}

// GetCampaignTimeline reconstructs the causality-linked story of an attack.
func (s *FusionService) GetCampaignTimeline(ctx context.Context, entityID string) (*detection.CampaignTimeline, error) {
	camp := s.engine.GetCampaign(entityID)
	if camp == nil {
		return nil, fmt.Errorf("campaign not found for entity: %s", entityID)
	}

	timeline := &detection.CampaignTimeline{
		CampaignID: entityID,
		Start:      camp.FirstSeen,
		End:        camp.LastSeen,
		Events:     []detection.TimelineEvent{},
	}

	// 1. Add Fusion Alerts as anchor points
	for i, alert := range camp.Alerts {
		timeline.Events = append(timeline.Events, detection.TimelineEvent{
			ID:          fmt.Sprintf("alert-%d", i),
			Timestamp:   alert.Timestamp,
			Type:        "ALERT",
			Source:      "AttackFusionEngine",
			Description: alert.Name,
			Tactic:      alert.Tactic,
			EntityID:    entityID,
			Severity:    "HIGH",
			Metadata:    map[string]interface{}{"rule_id": alert.RuleID},
		})
	}

	// 2. Query raw events from AnalyticsEngine for this entity in the time window
	// OQL query: "host = 'ID' OR user = 'ID' last 2h" (simplified)
	query := fmt.Sprintf("output contains '%s'", entityID)
	rawEvents, err := s.analytics.Search(ctx, query, "sql", 100, 0)
	if err == nil {
		for i, raw := range rawEvents {
			tsStr, _ := raw["timestamp"].(string)
			ts, _ := time.Parse(time.RFC3339, tsStr)
			
			// Only include events within the campaign window (with 5m padding)
			if ts.After(camp.FirstSeen.Add(-5*time.Minute)) && ts.Before(camp.LastSeen.Add(5*time.Minute)) {
				timeline.Events = append(timeline.Events, detection.TimelineEvent{
					ID:          fmt.Sprintf("evt-%d", i),
					Timestamp:   ts,
					Type:        "EVENT",
					Source:      raw["host"].(string),
					Description: raw["output"].(string),
					EntityID:    entityID,
					Metadata:    raw,
				})
			}
		}
	}

	// 3. Sort chronologically
	sort.Slice(timeline.Events, func(i, j int) bool {
		return timeline.Events[i].Timestamp.Before(timeline.Events[j].Timestamp)
	})

	return timeline, nil
}

// Dependencies returns naming dependencies.
func (s *FusionService) Dependencies() []string {
	return []string{}
}

// Start matches the platform.Service interface.
func (s *FusionService) Start(ctx context.Context) error {
	s.log.Info("[FUSION] Attack Fusion Engine started.")
	return nil
}

// Stop matches the platform.Service interface.
func (s *FusionService) Stop(ctx context.Context) error {
	return nil
}
