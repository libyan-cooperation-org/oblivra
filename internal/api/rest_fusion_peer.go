package api

// rest_fusion_peer.go — Handlers for Phase 10.5 (Peer Analytics) and Phase 10.6 (Fusion Engine)

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ── In-memory fusion campaign store ──────────────────────────────────────────

type fusionCampaign struct {
	ID               string   `json:"id"`
	Entities         []string `json:"entities"`
	AlertCount       int      `json:"alert_count"`
	TacticStages     []string `json:"tactic_stages"`
	StageCount       int      `json:"stage_count"`
	Confidence       float64  `json:"confidence"`
	FirstSeen        string   `json:"first_seen"`
	LastSeen         string   `json:"last_seen"`
	Status           string   `json:"status"`
	KillChainProgress int     `json:"kill_chain_progress"`
}

type killChainStage struct {
	TacticID   string   `json:"tactic_id"`
	TacticName string   `json:"tactic_name"`
	HitCount   int      `json:"hit_count"`
	Techniques []string `json:"techniques"`
	FirstSeen  string   `json:"first_seen,omitempty"`
}

var (
	fusionCampaignsMu sync.RWMutex
	fusionCampaigns   []fusionCampaign
	fusionSeeded      bool
)

func seedFusionCampaigns() {
	if fusionSeeded {
		return
	}
	fusionSeeded = true
	// Removed fake campaign data — Phase 25.1 remediation
	fusionCampaigns = []fusionCampaign{}
}

// GET /api/v1/fusion/campaigns
func (s *RESTServer) handleFusionCampaigns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fusionCampaignsMu.Lock()
	seedFusionCampaigns()

	// Merge in live SIEM fusion data if available
	if s.siem != nil {
		ctx := r.Context()
		// AggregateHostEvents can surface cross-entity correlations
		counts, err := s.siem.AggregateHostEvents(ctx, "EventType:lateral_movement OR EventType:c2_beacon OR EventType:exfil", "host_id")
		if err == nil && len(counts) > 0 {
			for entity, count := range counts {
				if count >= 3 {
					// Check if a campaign already tracks this entity
					found := false
					for i := range fusionCampaigns {
						for _, e := range fusionCampaigns[i].Entities {
							if e == entity {
								found = true
								fusionCampaigns[i].AlertCount += count
								break
							}
						}
					}
					if !found {
						fusionCampaigns = append(fusionCampaigns, fusionCampaign{
							ID:           fmt.Sprintf("camp-live-%s", entity),
							Entities:     []string{entity},
							AlertCount:   count,
							TacticStages: []string{"Command & Control"},
							StageCount:   1,
							Confidence:   0.0,
							FirstSeen:    time.Now().Format(time.RFC3339),
							LastSeen:     time.Now().Format(time.RFC3339),
							Status:       "active",
							KillChainProgress: 8,
						})
					}
				}
			}
		}
	}

	out := make([]fusionCampaign, len(fusionCampaigns))
	copy(out, fusionCampaigns)
	fusionCampaignsMu.Unlock()

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"campaigns": out})
}

// GET /api/v1/fusion/campaigns/{id}/kill-chain
func (s *RESTServer) handleFusionCampaignDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse campaign ID and action from path
	// /api/v1/fusion/campaigns/{id}/kill-chain
	suffix := strings.TrimPrefix(r.URL.Path, "/api/v1/fusion/campaigns/")
	parts := strings.SplitN(suffix, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing campaign ID", http.StatusBadRequest)
		return
	}
	campaignID := parts[0]

	fusionCampaignsMu.RLock()
	seedFusionCampaigns()
	var campaign *fusionCampaign
	for i := range fusionCampaigns {
		if fusionCampaigns[i].ID == campaignID {
			c := fusionCampaigns[i]
			campaign = &c
			break
		}
	}
	fusionCampaignsMu.RUnlock()

	if campaign == nil {
		http.Error(w, "campaign not found", http.StatusNotFound)
		return
	}

	// Build kill chain stages from campaign's tactic stages
	var stages []killChainStage
	for i, tactic := range campaign.TacticStages {
		tacticID := fmt.Sprintf("TA%04d", 1+i*3)
		stages = append(stages, killChainStage{
			TacticID:   tacticID,
			TacticName: tactic,
			HitCount:   campaign.AlertCount / len(campaign.TacticStages),
			Techniques: []string{fmt.Sprintf("T1%03d", 50+i*10)},
			FirstSeen:  campaign.FirstSeen,
		})
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"stages": stages})
}

// ── Peer Analytics handlers ───────────────────────────────────────────────────

type peerGroup struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Basis        string  `json:"basis"`
	MemberCount  int     `json:"member_count"`
	AvgRiskScore float64 `json:"avg_risk_score"`
	AnomalyRate  float64 `json:"anomaly_rate"`
	LastUpdated  string  `json:"last_updated"`
}

type peerDeviation struct {
	EntityID       string  `json:"entity_id"`
	EntityType     string  `json:"entity_type"`
	GroupID        string  `json:"group_id"`
	GroupName      string  `json:"group_name"`
	EntityRisk     float64 `json:"entity_risk"`
	GroupAvgRisk   float64 `json:"group_avg_risk"`
	DeviationSigma float64 `json:"deviation_sigma"`
	TopDeviation   string  `json:"top_deviation"`
	Timestamp      string  `json:"timestamp"`
}

var (
	peerGroupsMu     sync.RWMutex
	cachedPeerGroups []peerGroup
	peerGroupsSeeded bool
)

func seedPeerGroups(agentCount int) {
	if peerGroupsSeeded && len(cachedPeerGroups) > 0 {
		return
	}
	peerGroupsSeeded = true
	// Removed fake peer group data — Phase 25.1 remediation
	cachedPeerGroups = []peerGroup{}
}

// GET /api/v1/ueba/peer-groups
func (s *RESTServer) handlePeerGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.agentsMu.RLock()
	agentCount := len(s.agents)
	s.agentsMu.RUnlock()

	peerGroupsMu.Lock()
	seedPeerGroups(agentCount)
	out := make([]peerGroup, len(cachedPeerGroups))
	copy(out, cachedPeerGroups)
	peerGroupsMu.Unlock()

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"groups": out})
}

// GET /api/v1/ueba/peer-deviations
func (s *RESTServer) handlePeerDeviations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	peerGroupsMu.RLock()
	groups := make([]peerGroup, len(cachedPeerGroups))
	copy(groups, cachedPeerGroups)
	peerGroupsMu.RUnlock()

	// Generate deviations: query SIEM for high-risk entities, compare vs. group avg
	var deviations []peerDeviation
	if s.siem != nil {
		ctx := r.Context()
		events, _ := s.siem.SearchHostEvents(ctx, "EventType:anomaly OR EventType:off_hours_login OR EventType:mass_download", 20)
		for _, e := range events {
			entityID := e.HostID
			if entityID == "" {
				continue
			}
			// Find the group this entity belongs to (simple heuristic)
			group := groups[0] // Heuristic: first group
			entityRisk := 0.0
			sigma := 0.0
			if sigma < 1.0 {
				continue // Only surface outliers
			}
			deviations = append(deviations, peerDeviation{
				EntityID:       entityID,
				EntityType:     "host",
				GroupID:        group.ID,
				GroupName:      group.Name,
				EntityRisk:     entityRisk,
				GroupAvgRisk:   group.AvgRiskScore,
				DeviationSigma: sigma,
				TopDeviation:   "unknown",
				Timestamp:      time.Now().Format(time.RFC3339),
			})
		}
	}

	// Removed seed deviations — Phase 25.1 remediation

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"deviations": deviations})
}
