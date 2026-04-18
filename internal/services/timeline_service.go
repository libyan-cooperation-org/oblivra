package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type TimelineStep struct {
	Timestamp   string            `json:"timestamp"`
	Type        string            `json:"type"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Meta        map[string]string `json:"meta"`
	RawEvent    interface{}       `json:"raw_event"`
}

type Timeline struct {
	IncidentID  string         `json:"incident_id"`
	PrincipalID string         `json:"principal_id"`
	Steps       []TimelineStep `json:"steps"`
	GeneratedAt time.Time      `json:"generated_at"`
}

// TimelineService automates incident narrative reconstruction.
type TimelineService struct {
	repo database.SIEMStore
	log  *logger.Logger
}

func NewTimelineService(repo database.SIEMStore, log *logger.Logger) *TimelineService {
	return &TimelineService{
		repo: repo,
		log:  log.WithPrefix("services:timeline"),
	}
}

func (s *TimelineService) Name() string { return "timeline-service" }

func (s *TimelineService) Start(ctx context.Context) error { return nil }
func (s *TimelineService) Stop(ctx context.Context) error  { return nil }

// ReconstructTimeline builds a story around a principal within a time window.
func (s *TimelineService) ReconstructTimeline(ctx context.Context, principalID string, principalType string, targetTime string) (*Timeline, error) {
	s.log.Info("[TIMELINE] Reconstructing for %s (%s) around %s", principalID, principalType, targetTime)

	// 1. Calculate window: -10m to +20m
	t, err := time.Parse(time.RFC3339, targetTime)
	if err != nil {
		// Try fallback format if RFC3339 fails
		t, err = time.Parse("2006-01-02 15:04:05", targetTime)
		if err != nil {
			t = time.Now()
		}
	}

	start := t.Add(-10 * time.Minute).Format("2006-01-02 15:04:05")
	end := t.Add(20 * time.Minute).Format("2006-01-02 15:04:05")

	events, err := s.repo.GetTimelineEvents(ctx, principalID, principalType, start, end)
	if err != nil {
		return nil, err
	}

	timeline := &Timeline{
		IncidentID:  fmt.Sprintf("INC-%d", time.Now().Unix()),
		PrincipalID: principalID,
		Steps:       []TimelineStep{},
		GeneratedAt: time.Now(),
	}

	for _, e := range events {
		step := TimelineStep{
			Timestamp:   e.Timestamp,
			Type:        e.EventType,
			Title:       s.generateTitle(e),
			Description: s.generateDescription(e),
			Severity:    s.inferSeverity(e),
			Meta: map[string]string{
				"host": e.HostID,
				"user": e.User,
				"ip":   e.SourceIP,
			},
			RawEvent: e,
		}
		timeline.Steps = append(timeline.Steps, step)
	}

	return timeline, nil
}

func (s *TimelineService) generateTitle(e database.HostEvent) string {
	switch e.EventType {
	case "failed_login":
		return "Unauthorized Access Attempt"
	case "process_spawn":
		return "Process Execution"
	case "file_modified":
		return "File System Alteration"
	case "connection_established":
		return "Outbound Network Connection"
	case "privilege_escalation":
		return "Privilege Escalation Detected"
	default:
		return fmt.Sprintf("Security Event: %s", e.EventType)
	}
}

func (s *TimelineService) generateDescription(e database.HostEvent) string {
	if e.User != "" && e.SourceIP != "" {
		return fmt.Sprintf("User %s attempted action from %s on host %s", e.User, e.SourceIP, e.HostID)
	}
	return e.RawLog
}

func (s *TimelineService) inferSeverity(e database.HostEvent) string {
	if e.EventType == "privilege_escalation" {
		return "CRITICAL"
	}
	if e.EventType == "failed_login" {
		return "HIGH"
	}
	return "MEDIUM"
}
