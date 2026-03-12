package risk

import (
	"context"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type ConfigChangeEvent struct {
	Type      string    `json:"type"`
	Target    string    `json:"target"` // host_id or "fleet"
	Changes   []string  `json:"changes"`
	Timestamp string    `json:"timestamp"`
}

type RiskScore struct {
	ID        string    `json:"id"`
	Score     int       `json:"score"` // 0-100
	Level     string    `json:"level"` // "Low", "Medium", "High", "Critical"
	Impact    string    `json:"impact"`
	Reason    string    `json:"reason"`
	Timestamp string    `json:"timestamp"`
}

type RiskEngine struct {
	bus   *eventbus.Bus
	log   *logger.Logger
	db    database.DatabaseStore
	hosts database.HostStore
}

func NewRiskEngine(bus *eventbus.Bus, db database.DatabaseStore, h database.HostStore, log *logger.Logger) *RiskEngine {
	return &RiskEngine{
		bus:   bus,
		db:    db,
		hosts: h,
		log:   log.WithPrefix("risk"),
	}
}

func (e *RiskEngine) Start(ctx context.Context) {
	e.log.Info("RiskEngine starting...")
	// Subscribe to configuration change events
	e.bus.Subscribe("config.changed", e.handleConfigChange)
}

func (e *RiskEngine) handleConfigChange(event eventbus.Event) {
	change, ok := event.Data.(ConfigChangeEvent)
	if !ok {
		return
	}

	score := e.CalculateScore(change)
	e.log.Info("Config change risk evaluated: %d (%s) - %s", score.Score, score.Level, score.Reason)

	// Publish risk score event
	e.bus.Publish("risk.evaluated", score)
}

func (e *RiskEngine) CalculateScore(event ConfigChangeEvent) RiskScore {
	score := 10 // Base score
	reason := "Routine configuration update"

	// Heuristic 1: Blast Radius
	if event.Target == "fleet" {
		score += 40
		reason = "Fleet-wide configuration change (High Blast Radius)"
	}

	// Heuristic 2: Sensitive Collectors
	for _, change := range event.Changes {
		if change == "ebpf" || change == "fim" {
			score += 20
			reason += " | Enabling kernel/filesystem collectors"
		}
	}

	// Heuristic 3: Impacted Hosts
	if event.Target != "fleet" {
		host, err := e.hosts.GetByID(context.Background(), event.Target)
		if err == nil {
			if host.Label == "Production" || host.Label == "Domain Controller" {
				score += 30
				reason += " | High-value target impacted"
			}
		}
	}

	if score > 100 {
		score = 100
	}

	level := "Low"
	if score > 75 {
		level = "Critical"
	} else if score > 50 {
		level = "High"
	} else if score > 25 {
		level = "Medium"
	}

	return RiskScore{
		ID:        fmt.Sprintf("risk_%d", time.Now().UnixNano()),
		Score:     score,
		Level:     level,
		Reason:    reason,
		Impact:    "Performance overhead / Security posture shift",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}
