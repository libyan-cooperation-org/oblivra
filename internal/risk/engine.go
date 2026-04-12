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
	users database.UserStore
}

func NewRiskEngine(bus *eventbus.Bus, db database.DatabaseStore, h database.HostStore, u database.UserStore, log *logger.Logger) *RiskEngine {
	return &RiskEngine{
		bus:   bus,
		db:    db,
		hosts: h,
		users: u,
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

// TriageIncident performs automated triage scoring (Phase 20.9)
func (e *RiskEngine) TriageIncident(ctx context.Context, inc *database.Incident) (RiskScore, error) {
	score := 0
	reason := "Automated Triage: "

	// 1. Asset Criticality
	host, err := e.hosts.GetByID(ctx, inc.GroupKey)
	if err == nil && host != nil {
		criticality := host.CriticalityScore * 4 // Map 1-10 to 4-40
		score += criticality
		reason += fmt.Sprintf("Asset Criticality Level %d (%s); ", host.CriticalityScore, host.CriticalityReason)
	}

	// 2. Identity Risk
	user, err := e.users.GetUserByEmail(ctx, inc.GroupKey)
	if err == nil && user != nil {
		privilege := user.CriticalityScore * 3 // Map 1-10 to 3-30
		score += privilege
		reason += fmt.Sprintf("Identity Criticality Level %d (%s); ", user.CriticalityScore, user.CriticalityReason)
	}

	// 3. MITRE Tactic Weighting
	for _, tactic := range inc.MitreTactics {
		switch tactic {
		case "Exfiltration", "Impact", "Command and Control":
			score += 25
			reason += fmt.Sprintf("Critical tactic (%s); ", tactic)
		case "Persistence", "Privilege Escalation":
			score += 15
			reason += fmt.Sprintf("High-risk tactic (%s); ", tactic)
		}
	}

	// 4. Volume Bonus
	if inc.EventCount > 100 {
		score += 10
		reason += "High event volume; "
	}

	if score > 100 {
		score = 100
	}

	level := "Low"
	if score > 85 {
		level = "Critical"
	} else if score > 60 {
		level = "High"
	} else if score > 30 {
		level = "Medium"
	}

	return RiskScore{
		ID:        inc.ID,
		Score:     score,
		Level:     level,
		Reason:    reason,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}
