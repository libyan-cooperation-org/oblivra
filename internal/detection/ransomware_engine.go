package detection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/security"
)

// RansomwareEngine correlated file system events to detect ransomware behavioral patterns.
type RansomwareEngine struct {
	bus       *eventbus.Bus
	incidents database.IncidentStore
	log       *logger.Logger

	// Track stats per host/directory to detect "mass" operations
	mu             sync.Mutex
	highEntropyOps map[string]int // hostID -> count
	lastReset      time.Time
}

func NewRansomwareEngine(bus *eventbus.Bus, incidents database.IncidentStore, log *logger.Logger) *RansomwareEngine {
	return &RansomwareEngine{
		bus:            bus,
		incidents:      incidents,
		log:            log.WithPrefix("ransomware"),
		highEntropyOps: make(map[string]int),
		lastReset:      time.Now(),
	}
}

func (e *RansomwareEngine) Name() string {
	return "RansomwareEngine"
}

func (e *RansomwareEngine) Startup(ctx context.Context) {
	e.log.Info("Ransomware detection engine starting...")

	e.bus.Subscribe(eventbus.EventFIMModified, e.handleFileEvent)
	e.bus.Subscribe(eventbus.EventFIMRenamed, e.handleFileEvent)

	go e.resetLoop(ctx)
}

func (e *RansomwareEngine) Shutdown() {
	e.log.Info("Ransomware detection engine shutting down...")
}

func (e *RansomwareEngine) resetLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			e.mu.Lock()
			e.highEntropyOps = make(map[string]int)
			e.lastReset = time.Now()
			e.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (e *RansomwareEngine) handleFileEvent(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}

	hostID, _ := data["host_id"].(string)
	filePath, _ := data["path"].(string)
	content, _ := data["content"].([]byte)

	if len(content) > 0 && security.IsHighEntropy(content) {
		e.mu.Lock()
		e.highEntropyOps[hostID]++
		count := e.highEntropyOps[hostID]
		e.mu.Unlock()

		e.log.Debug("[DETECTION] High entropy write detected on %s: %s (Count: %d)", hostID, filePath, count)

		// Threshold for "mass" encryption: 10 high-entropy writes within 1 minute
		if count >= 10 {
			e.triggerAlert(hostID, "Massive high-entropy file modifications detected. Potential ransomware activity.")
		}
	}
}

func (e *RansomwareEngine) triggerAlert(hostID string, reason string) {
	e.log.Error("[ALERT] RANSOMWARE PATTERN DETECTED on %s: %s", hostID, reason)

	incident := &database.Incident{
		ID:              fmt.Sprintf("RANS-%d", time.Now().Unix()),
		RuleID:          "Ransomware.Behavioral.Entropy",
		GroupKey:        hostID,
		Status:          "Active",
		Severity:        "Critical",
		Title:           "Potential Ransomware Activity Detected",
		Description:     reason,
		FirstSeenAt:     time.Now(),
		LastSeenAt:      time.Now(),
		EventCount:      1,
		MitreTactics:    []string{"Impact"},
		MitreTechniques: []string{"Data Encrypted for Impact"},
	}

	// Persist to DB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.incidents.Upsert(ctx, incident); err != nil {
		e.log.Error("Failed to persist ransomware incident: %v", err)
	}

	// Publish to alert bus
	e.bus.Publish(eventbus.EventSIEMAlert, incident)
}
