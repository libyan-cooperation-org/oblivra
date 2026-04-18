package detection

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// Campaign represents an active multi-stage attack sequence linked to an entity.
type Campaign struct {
	EntityID      string            `json:"entity_id"`
	Tactics       map[string]bool   `json:"tactics"` // TacticID -> true
	Alerts        []FusionAlertInfo `json:"alerts"`
	Probability   float64           `json:"probability"`
	FirstSeen     time.Time         `json:"first_seen"`
	LastSeen      time.Time         `json:"last_seen"`
	IsTriggered   bool              `json:"is_triggered"`
}

type FusionAlertInfo struct {
	RuleID    string    `json:"rule_id"`
	Name      string    `json:"name"`
	Tactic    string    `json:"tactic"`
	Timestamp time.Time `json:"timestamp"`
}

// AttackFusionEngine correlates alerts across the MITRE ATT&CK kill chain.
type AttackFusionEngine struct {
	// key: EntityID (user, host, or IP)
	campaigns *expirable.LRU[string, *Campaign]
	mu        sync.Mutex
	bus       *eventbus.Bus
	log       *logger.Logger

	// Probabilistic parameters
	BaseProb       float64
	TacticWeight   float64
	StageThreshold int
}

// NewAttackFusionEngine initialises the fusion engine.
func NewAttackFusionEngine(bus *eventbus.Bus, log *logger.Logger) *AttackFusionEngine {
	// Window of 2 hours for campaign correlation
	window := 2 * time.Hour
	fe := &AttackFusionEngine{
		campaigns: expirable.NewLRU[string, *Campaign](5000, nil, window),
		bus:       bus,
		log:       log,
		// Defaults
		BaseProb:       0.01, // 1% chance any single alert is a real attack
		TacticWeight:   0.40, // Significant boost per unique tactic
		StageThreshold: 3,    // Alert on 3+ distinct MITRE tactics
	}

	// Subscribe to SIEM alerts
	bus.Subscribe("siem.alert_fired", func(ev eventbus.Event) {
		if data, ok := ev.Data.(map[string]interface{}); ok {
			fe.HandleAlert(data)
		}
	})

	// Also subscribe to Correlation Engine matches
	bus.Subscribe("detection.correlation_match", func(ev eventbus.Event) {
		if match, ok := ev.Data.(CorrelationMatch); ok {
			fe.HandleCorrelationMatch(match)
		}
	})

	return fe
}

// HandleAlert processes a standard SIEM alert.
func (fe *AttackFusionEngine) HandleAlert(data map[string]interface{}) {
	ruleID, _ := data["rule_id"].(string)
	name, _ := data["description"].(string)
	tactic, _ := data["tactic"].(string) // We expect TAxxxx or "Initial Access"
	groupKey, _ := data["group_key"].(string)

	if tactic == "" || groupKey == "" {
		return
	}

	fe.ingest(groupKey, ruleID, name, tactic)
}

// HandleCorrelationMatch processes a match from the CorrelationEngine.
func (fe *AttackFusionEngine) HandleCorrelationMatch(match CorrelationMatch) {
	fe.ingest(match.GroupKey, match.RuleID, match.RuleName, match.MitreTechnique)
}

func (fe *AttackFusionEngine) ingest(entityID, ruleID, name, tactic string) {
	fe.mu.Lock()

	// 1. Resolve tactic to ID if it's a name
	tacticID := fe.resolveTacticID(tactic)
	if tacticID == "" {
		fe.mu.Unlock()
		return
	}

	// 2. Get or create campaign
	camp, ok := fe.campaigns.Get(entityID)
	if !ok {
		camp = &Campaign{
			EntityID:  entityID,
			Tactics:   make(map[string]bool),
			FirstSeen: time.Now(),
			Alerts:    []FusionAlertInfo{},
		}
		fe.campaigns.Add(entityID, camp)
	}

	// 3. Update state
	camp.LastSeen = time.Now()
	camp.Tactics[tacticID] = true
	camp.Alerts = append(camp.Alerts, FusionAlertInfo{
		RuleID:    ruleID,
		Name:      name,
		Tactic:    tacticID,
		Timestamp: time.Now(),
	})

	// 4. Calculate Bayesian Probability
	// Simple accumulation: P = 1 - (1-base) * (1-weight)^n
	// But we'll use a slightly more nuanced approach for "Phase 10.6"
	n := float64(len(camp.Tactics))
	fe.calculateProbability(camp, n)

	// 5. Check Threshold
	var shouldTrigger bool
	if len(camp.Tactics) >= fe.StageThreshold && !camp.IsTriggered {
		camp.IsTriggered = true
		shouldTrigger = true
	}

	// Copy data for async publish (Deep copy map!)
	campCopy := *camp
	campCopy.Tactics = make(map[string]bool, len(camp.Tactics))
	for k, v := range camp.Tactics {
		campCopy.Tactics[k] = v
	}
	// Note: Alerts is a slice, could technically race if appended without a lock. We should copy it too.
	campCopy.Alerts = make([]FusionAlertInfo, len(camp.Alerts))
	copy(campCopy.Alerts, camp.Alerts)

	fe.mu.Unlock() // Release lock early

	if shouldTrigger {
		fe.triggerFusionAlert(&campCopy)
	}

	// 6. Publish campaign update for UI live view
	fe.bus.Publish("fusion.campaign_updated", &campCopy)
}

func (fe *AttackFusionEngine) calculateProbability(camp *Campaign, n float64) {
	// P(A|T) = 1 - (1-P_base) * (1-P_weight)^n
	// As n increases, P approaches 1.0
	prob := 1.0 - (1.0-fe.BaseProb)*math.Pow(1.0-fe.TacticWeight, n)
	camp.Probability = prob
}

func (fe *AttackFusionEngine) triggerFusionAlert(camp *Campaign) {
	fe.log.Warn("[FUSION] MULTI-STAGE CAMPAIGN DETECTED! Entity: %s | Probability: %.2f | Stages: %d",
		camp.EntityID, camp.Probability, len(camp.Tactics))

	fe.bus.Publish("siem.alert_fired", map[string]interface{}{
		"type":        "FUSION_CAMPAIGN_DETECTED",
		"severity":    "CRITICAL",
		"description": fmt.Sprintf("Coordinated multi-stage attack detected for entity %s. Probabilistic Score: %.2f", camp.EntityID, camp.Probability),
		"technique":   "T1506", // Multi-Stage Attack
		"group_key":   camp.EntityID,
		"meta": map[string]interface{}{
			"probability": camp.Probability,
			"stages":      len(camp.Tactics),
			"tactics":     camp.Tactics,
		},
	})
}

func (fe *AttackFusionEngine) resolveTacticID(t string) string {
	t = strings.ToUpper(t)
	// If it's already an ID
	if strings.HasPrefix(t, "TA") {
		return t
	}
	// Try to map name to ID using existing Tactics map
	for id, name := range Tactics {
		if strings.EqualFold(name, t) {
			return id
		}
	}
	// Fallback to searching technique map if passed a technique instead
	if strings.HasPrefix(t, "T") {
		// Just return the technique if we can't find a tactic, 
		// but ideally handled by the caller.
		return t 
	}
	return ""
}

// GetActiveCampaigns returns a snapshot of all currently tracked campaigns.
func (fe *AttackFusionEngine) GetActiveCampaigns() []Campaign {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	keys := fe.campaigns.Keys()
	res := make([]Campaign, 0, len(keys))
	for _, k := range keys {
		if c, ok := fe.campaigns.Peek(k); ok {
			res = append(res, *c)
		}
	}
	return res
}

// GetCampaign returns a specific campaign by entity ID.
func (fe *AttackFusionEngine) GetCampaign(id string) *Campaign {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	if c, ok := fe.campaigns.Get(id); ok {
		return c
	}
	return nil
}
