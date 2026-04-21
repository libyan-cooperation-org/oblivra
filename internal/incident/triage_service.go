package incident

import (
	"context"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// TriageResult contains the outcome of an automated incident assessment.
type TriageResult struct {
	Score     int
	Level     string
	Reason    string
	Timestamp string
}

// TriageService orchestrates automated incident assessment and escalation logic.
type TriageService struct {
	hosts database.HostStore
	users database.UserStore
	log   *logger.Logger
}

func NewTriageService(h database.HostStore, u database.UserStore, log *logger.Logger) *TriageService {
	return &TriageService{
		hosts: h,
		users: u,
		log:   log.WithPrefix("triage"),
	}
}

// Triage assesses an incident based on asset criticality, identity risk, and MITRE tactics.
func (s *TriageService) Triage(ctx context.Context, inc *database.Incident) (TriageResult, error) {
	score := 0
	reason := "Automated Triage: "

	// 1. Asset Criticality
	host, err := s.hosts.GetByID(ctx, inc.GroupKey)
	if err == nil && host != nil {
		criticality := host.CriticalityScore * 4 // Map 1-10 to 4-40
		score += criticality
		reason += fmt.Sprintf("Asset Criticality Level %d (%s); ", host.CriticalityScore, host.CriticalityReason)
	}

	// 2. Identity Risk
	user, err := s.users.GetUserByEmail(ctx, inc.GroupKey)
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

	return TriageResult{
		Score:     score,
		Level:     level,
		Reason:    reason,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// AutoEscalate applies severity updates based on triage results.
func (s *TriageService) AutoEscalate(inc *database.Incident, triage TriageResult) {
	inc.TriageScore = triage.Score
	inc.TriageReason = triage.Reason

	if triage.Score > 85 {
		inc.Severity = "Critical"
	} else if triage.Score > 60 && inc.Severity != "Critical" {
		inc.Severity = "High"
	} else if triage.Score > 30 && inc.Severity == "Low" {
		inc.Severity = "Medium"
	}
}
