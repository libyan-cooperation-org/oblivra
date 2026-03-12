package temporal

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// Policy defines the temporal validation boundaries.
type Policy struct {
	MaxFutureSkew time.Duration // Events this far in the future are flagged
	MaxPastAge    time.Duration // Events this old are flagged
	DriftAlertMs  int64         // Agent clock drift threshold in milliseconds
	GracePeriod   time.Duration // Allowed network jitter period
}

// DefaultPolicy returns conservative temporal boundaries.
func DefaultPolicy() Policy {
	return Policy{
		MaxFutureSkew: 1 * time.Hour,
		MaxPastAge:    30 * 24 * time.Hour, // 30 days
		DriftAlertMs:  5000,                // 5 seconds
		GracePeriod:   100 * time.Millisecond,
	}
}

// Violation represents a detected temporal anomaly.
type Violation struct {
	Timestamp string    `json:"timestamp"`
	HostID    string    `json:"host_id"`
	Type      string    `json:"type"` // "future_event", "stale_event", "clock_drift"
	Detail    string    `json:"detail"`
	DeltaMs   int64     `json:"delta_ms"`
}

// IntegrityService validates event timestamps and detects clock drift.
type IntegrityService struct {
	policy     Policy
	bus        *eventbus.Bus
	log        *logger.Logger
	violations []Violation
	agentDrift map[string]int64     // hostID -> last known drift in ms
	highWater  map[string]string    // hostID -> latest processed timestamp
	mu         sync.RWMutex
}

// NewIntegrityService creates a new temporal integrity checker.
func NewIntegrityService(policy Policy, bus *eventbus.Bus, log *logger.Logger) *IntegrityService {
	return &IntegrityService{
		policy:     policy,
		bus:        bus,
		log:        log.WithPrefix("temporal"),
		agentDrift: make(map[string]int64),
		highWater:  make(map[string]string),
	}
}

// ValidateTimestamp checks if an event timestamp is within acceptable bounds.
// Returns nil if valid, or a Violation if the timestamp is suspicious.
func (s *IntegrityService) ValidateTimestamp(hostID string, eventTime time.Time) *Violation {
	now := time.Now()
	delta := eventTime.Sub(now)

	// 1. Future event check
	if delta > s.policy.MaxFutureSkew {
		v := &Violation{
			Timestamp: now.Format(time.RFC3339),
			HostID:    hostID,
			Type:      "future_event",
			Detail:    fmt.Sprintf("Event timestamp %s is %.1f minutes in the future", eventTime.Format(time.RFC3339), delta.Minutes()),
			DeltaMs:   delta.Milliseconds(),
		}
		s.recordViolation(*v)
		return v
	}

	// 2. Stale event check
	if -delta > s.policy.MaxPastAge {
		v := &Violation{
			Timestamp: now.Format(time.RFC3339),
			HostID:    hostID,
			Type:      "stale_event",
			Detail:    fmt.Sprintf("Event timestamp %s is %.0f days old", eventTime.Format(time.RFC3339), (-delta).Hours()/24),
			DeltaMs:   delta.Milliseconds(),
		}
		s.recordViolation(*v)
		return v
	}

	// 3. Sequence manipulation (Monotonicity) check
	s.mu.RLock()
	lastTs, exists := s.highWater[hostID]
	lastTime := parseTime(lastTs)
	s.mu.RUnlock()

	if exists && eventTime.Before(lastTime.Add(-s.policy.GracePeriod)) {
		inversion := lastTime.Sub(eventTime).Milliseconds()
		v := &Violation{
			Timestamp: now.Format(time.RFC3339),
			HostID:    hostID,
			Type:      "sequence_manipulation",
			Detail:    fmt.Sprintf("Timestamp inversion detected: event is %dms older than host high-water mark", inversion),
			DeltaMs:   -inversion,
		}
		s.recordViolation(*v)
		return v
	}

	// Update High-Water Mark if this event moves time forward
	if !exists || eventTime.After(lastTime) {
		s.mu.Lock()
		s.highWater[hostID] = eventTime.Format(time.RFC3339)
		s.mu.Unlock()
	}

	return nil
}

// RecordHeartbeat tracks an agent's clock drift relative to the server.
func (s *IntegrityService) RecordHeartbeat(hostID string, agentTime time.Time) {
	drift := time.Since(agentTime).Milliseconds()

	s.mu.Lock()
	s.agentDrift[hostID] = drift
	s.mu.Unlock()

	if abs(drift) > s.policy.DriftAlertMs {
		v := Violation{
			Timestamp: time.Now().Format(time.RFC3339),
			HostID:    hostID,
			Type:      "clock_drift",
			Detail:    fmt.Sprintf("Agent clock drift: %dms (threshold: %dms)", drift, s.policy.DriftAlertMs),
			DeltaMs:   drift,
		}
		s.recordViolation(v)

		if s.bus != nil {
			s.bus.Publish("temporal.drift_detected", map[string]interface{}{
				"host_id":  hostID,
				"drift_ms": drift,
			})
		}
	}
}

// GetViolations returns all recorded temporal violations.
func (s *IntegrityService) GetViolations() []Violation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Violation, len(s.violations))
	copy(result, s.violations)
	return result
}

// GetAgentDrift returns the latest clock drift readings per agent.
func (s *IntegrityService) GetAgentDrift() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]int64, len(s.agentDrift))
	for k, v := range s.agentDrift {
		result[k] = v
	}
	return result
}

func (s *IntegrityService) recordViolation(v Violation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.violations = append(s.violations, v)
	s.log.Warn("[TEMPORAL] %s on %s: %s", v.Type, v.HostID, v.Detail)

	// Cap at 1000 violations to prevent unbounded memory growth
	if len(s.violations) > 1000 {
		s.violations = s.violations[len(s.violations)-500:]
	}
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// FleetDriftReport provides statistical analysis of clock drift across the fleet.
type FleetDriftReport struct {
	TotalAgents int            `json:"total_agents"`
	MeanDriftMs float64        `json:"mean_drift_ms"`
	StdDevMs    float64        `json:"std_dev_ms"`
	Outliers    []DriftOutlier `json:"outliers"`
	Timestamp   string         `json:"timestamp"`
}

// DriftOutlier represents an agent with anomalous clock drift.
type DriftOutlier struct {
	HostID  string  `json:"host_id"`
	DriftMs int64   `json:"drift_ms"`
	ZScore  float64 `json:"z_score"`
}

// TimestampedEvent is a minimal event with timestamp and source for sequence analysis.
type TimestampedEvent struct {
	Timestamp string    `json:"timestamp"`
	HostID    string    `json:"host_id"`
	EventID   string    `json:"event_id"`
}

// SequenceAnomaly represents a detected timestamp inversion.
type SequenceAnomaly struct {
	HostID      string    `json:"host_id"`
	EventIDA    string    `json:"event_id_a"`
	EventIDB    string    `json:"event_id_b"`
	TimestampA  string    `json:"timestamp_a"`
	TimestampB  string    `json:"timestamp_b"`
	InversionMs int64     `json:"inversion_ms"`
	Detail      string    `json:"detail"`
}

// DetectFleetDrift calculates statistical outliers across agent clock drift values.
// Uses Z-score analysis: agents with |Z| > 2.0 are flagged as outliers.
func (s *IntegrityService) DetectFleetDrift() *FleetDriftReport {
	s.mu.RLock()
	drifts := make(map[string]int64, len(s.agentDrift))
	for k, v := range s.agentDrift {
		drifts[k] = v
	}
	s.mu.RUnlock()

	report := &FleetDriftReport{
		TotalAgents: len(drifts),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if len(drifts) < 2 {
		return report
	}

	// Calculate mean
	var sum float64
	for _, d := range drifts {
		sum += float64(d)
	}
	mean := sum / float64(len(drifts))
	report.MeanDriftMs = mean

	// Calculate standard deviation
	var variance float64
	for _, d := range drifts {
		diff := float64(d) - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(drifts)))
	report.StdDevMs = stdDev

	// Detect outliers (Z-score > 2.0)
	if stdDev > 0 {
		for host, drift := range drifts {
			z := math.Abs(float64(drift)-mean) / stdDev
			if z > 2.0 {
				report.Outliers = append(report.Outliers, DriftOutlier{
					HostID:  host,
					DriftMs: drift,
					ZScore:  math.Round(z*100) / 100,
				})
			}
		}
	}

	// Sort outliers by Z-score descending
	sort.Slice(report.Outliers, func(i, j int) bool {
		return report.Outliers[i].ZScore > report.Outliers[j].ZScore
	})

	return report
}

// DetectSequenceManipulation scans events from the same host for timestamp inversions.
// If event N+1 has a timestamp older than event N (from the same source), it's flagged.
func (s *IntegrityService) DetectSequenceManipulation(events []TimestampedEvent) []SequenceAnomaly {
	// Group by host
	byHost := make(map[string][]TimestampedEvent)
	for _, e := range events {
		byHost[e.HostID] = append(byHost[e.HostID], e)
	}

	var anomalies []SequenceAnomaly

	for hostID, hostEvents := range byHost {
		if len(hostEvents) < 2 {
			continue
		}

		// Sort by arrival order (preserve original order)
		for i := 1; i < len(hostEvents); i++ {
			prev := hostEvents[i-1]
			curr := hostEvents[i]

			// Timestamp inversion: current event's timestamp is older than previous
			if parseTime(curr.Timestamp).Before(parseTime(prev.Timestamp)) {
				inversion := parseTime(prev.Timestamp).Sub(parseTime(curr.Timestamp)).Milliseconds()
				anomalies = append(anomalies, SequenceAnomaly{
					HostID:      hostID,
					EventIDA:    prev.EventID,
					EventIDB:    curr.EventID,
					TimestampA:  prev.Timestamp,
					TimestampB:  curr.Timestamp,
					InversionMs: inversion,
					Detail:      fmt.Sprintf("Event %s timestamp is %dms before preceding event %s on host %s", curr.EventID, inversion, prev.EventID, hostID),
				})

				s.recordViolation(Violation{
					Timestamp: time.Now().Format(time.RFC3339),
					HostID:    hostID,
					Type:      "sequence_manipulation",
					Detail:    fmt.Sprintf("Timestamp inversion: %dms between events %s and %s", inversion, prev.EventID, curr.EventID),
					DeltaMs:   -inversion,
				})
			}
		}
	}

	return anomalies
}

func parseTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return t
}

