package detection

import (
	"time"
)

// TimelineEvent represents a single point in an incident's progression.
type TimelineEvent struct {
	ID          string            `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	Type        string            `json:"type"` // "ALERT", "EVENT", "SYSTEM"
	Source      string            `json:"source"`
	Description string            `json:"description"`
	Tactic      string            `json:"tactic,omitempty"`
	Technique   string            `json:"technique,omitempty"`
	Severity    string            `json:"severity,omitempty"`
	EntityID    string            `json:"entity_id"`
	CausalityID string            `json:"causality_id,omitempty"` // ID of the parent/triggering event
	Metadata    map[string]interface{} `json:"metadata"`
}

// CampaignTimeline is the reconstructed story of an attack campaign.
type CampaignTimeline struct {
	CampaignID string          `json:"campaign_id"`
	Events     []TimelineEvent `json:"events"`
	Start      time.Time       `json:"start"`
	End        time.Time       `json:"end"`
}
