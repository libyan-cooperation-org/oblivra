package detection

import (
	"sort"
	"sync"
	"time"
)

// WindowBucket holds events for a specific grouping (e.g., source_ip).
type WindowBucket struct {
	Events []Event
	Latest time.Time
}

// WindowStateManager manages state for rules across multiple windows and watermarks.
type WindowStateManager struct {
	mu     sync.RWMutex
	buckets map[string]map[string]*WindowBucket // RuleID -> GroupKey -> Bucket
}

// NewWindowStateManager initializes a new manager for stream-oriented state.
func NewWindowStateManager() *WindowStateManager {
	return &WindowStateManager{
		buckets: make(map[string]map[string]*WindowBucket),
	}
}

// AddEvent injects an event into a rule's window, respecting window boundaries and watermarks.
func (m *WindowStateManager) AddEvent(rule Rule, groupKey string, event Event, eventTime time.Time) ([]Event, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.buckets[rule.ID]; !ok {
		m.buckets[rule.ID] = make(map[string]*WindowBucket)
	}

	bucket, ok := m.buckets[rule.ID][groupKey]
	if !ok {
		bucket = &WindowBucket{
			Events: make([]Event, 0),
		}
		m.buckets[rule.ID][groupKey] = bucket
	}

	// 1. Watermark Check: If event is older than (Latest - Watermark), drop it (unless replay is enabled).
	watermark := time.Duration(rule.Watermark) * time.Second
	if !bucket.Latest.IsZero() && eventTime.Before(bucket.Latest.Add(-watermark)) {
		// Late event - drop
		return nil, false
	}

	// Update latest observed time
	if eventTime.After(bucket.Latest) {
		bucket.Latest = eventTime
	}

	// 2. Add event
	bucket.Events = append(bucket.Events, event)

	// 3. Window Pruning
	windowDuration := time.Duration(rule.WindowSec) * time.Second
	
	if rule.Window == WindowTumbling {
		// Tumbling: All events must be within the same fixed block relative to bucket.Latest
		startTime := bucket.Latest.Truncate(windowDuration)
		filtered := bucket.Events[:0]
		for _, e := range bucket.Events {
			eTime, _ := time.Parse(time.RFC3339, e.Timestamp)
			if !eTime.Before(startTime) {
				filtered = append(filtered, e)
			}
		}
		bucket.Events = filtered
	} else {
		// Sliding (Default): Any event older than bucket.Latest - WindowSec is pruned
		startTime := bucket.Latest.Add(-windowDuration)
		filtered := bucket.Events[:0]
		for _, e := range bucket.Events {
			eTime, _ := time.Parse(time.RFC3339, e.Timestamp)
			if !eTime.Before(startTime) {
				filtered = append(filtered, e)
			}
		}
		bucket.Events = filtered
	}

	// Sort events by timestamp to ensure deterministic evaluation
	sort.Slice(bucket.Events, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, bucket.Events[i].Timestamp)
		tj, _ := time.Parse(time.RFC3339, bucket.Events[j].Timestamp)
		return ti.Before(tj)
	})

	return bucket.Events, true
}

// Clear removes state for a specific rule or group.
func (m *WindowStateManager) Clear(ruleID string, groupKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if groupKey == "" {
		delete(m.buckets, ruleID)
	} else if r, ok := m.buckets[ruleID]; ok {
		delete(r, groupKey)
	}
}
