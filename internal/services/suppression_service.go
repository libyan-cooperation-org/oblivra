package services

import (
	"context"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type SuppressionService struct {
	BaseService
	repo   database.SuppressionStore
	log    *logger.Logger
	cache  sync.Map // simple cache for active rules to avoid DB hits on every event
	// matchCounts is an in-memory tally of how many alerts each rule has
	// silenced since process start. Used by the UI to show operators which
	// rules are pulling weight (or which are stale).
	matchCounts sync.Map // ruleID -> *atomic.Int64
}

func NewSuppressionService(repo database.SuppressionStore, log *logger.Logger) *SuppressionService {
	return &SuppressionService{
		repo: repo,
		log:  log.WithPrefix("suppression_service"),
	}
}

// MatchCount returns the number of alerts a given rule has silenced since
// process start. Resets on restart by design — durable counts would require
// a schema column we don't have yet (Phase 26.9 follow-up).
func (s *SuppressionService) MatchCount(ruleID string) int64 {
	if v, ok := s.matchCounts.Load(ruleID); ok {
		if c, ok := v.(*atomic.Int64); ok {
			return c.Load()
		}
	}
	return 0
}

// MatchCounts returns a snapshot map of all in-memory match counts.
func (s *SuppressionService) MatchCounts() map[string]int64 {
	out := map[string]int64{}
	s.matchCounts.Range(func(k, v interface{}) bool {
		if id, ok := k.(string); ok {
			if c, ok := v.(*atomic.Int64); ok {
				out[id] = c.Load()
			}
		}
		return true
	})
	return out
}

// bumpMatch increments the in-memory hit counter for a rule.
func (s *SuppressionService) bumpMatch(ruleID string) {
	v, _ := s.matchCounts.LoadOrStore(ruleID, &atomic.Int64{})
	if c, ok := v.(*atomic.Int64); ok {
		c.Add(1)
	}
}

// SuggestFromEvidence builds a draft suppression rule from a false-positive
// evidence map. Looks at common fields (host_id, user_name, src_ip,
// rule_id, event_type) and prefers the most specific match.
//
// The returned rule is a draft — ID/CreatedAt are unset so the caller can
// either present it to an operator for approval (preferred) or call
// CreateRule directly. Phase 26.9's "automated feedback loop" — feedback
// (mark-as-FP) generates a suggestion, which the operator can promote to
// a suppression rule with one click.
func (s *SuppressionService) SuggestFromEvidence(evidence []map[string]interface{}) *database.SuppressionRule {
	if len(evidence) == 0 {
		return nil
	}

	// Aggregate common fields across all evidence rows. Pick the field
	// that has a single consistent value across every row — that's the
	// strongest "this attribute is what triggered the FP" signal.
	first := evidence[0]
	candidates := []string{"host_id", "user_name", "user", "src_ip", "source_ip", "event_type", "rule_id"}

	for _, field := range candidates {
		val, ok := first[field]
		if !ok {
			continue
		}
		valStr, ok := val.(string)
		if !ok || valStr == "" {
			continue
		}

		// Confirm the value is consistent across all evidence rows.
		consistent := true
		for _, row := range evidence[1:] {
			rv, ok := row[field]
			if !ok || rv != val {
				consistent = false
				break
			}
		}
		if !consistent {
			continue
		}

		// rule_id maps to the RuleID scope rather than the Field/Value pair.
		var ruleID, fieldName, fieldValue string
		if field == "rule_id" {
			ruleID = valStr
			fieldName, fieldValue = "host_id", "*" // global to that detection rule
		} else {
			fieldName, fieldValue = field, valStr
		}

		return &database.SuppressionRule{
			Label:       "Suggested from FP feedback",
			Description: "Auto-generated suppression candidate. Review evidence before activating.",
			RuleID:      ruleID,
			Field:       fieldName,
			Value:       fieldValue,
			IsRegex:     false,
			IsActive:    false, // operator must explicitly enable
		}
	}

	return nil
}

func (s *SuppressionService) Name() string { return "suppression-service" }

// ShouldSuppress checks if an alert with the given metadata should be silenced.
func (s *SuppressionService) ShouldSuppress(ctx context.Context, ruleID string, metadata map[string]string) (bool, string) {
	// 1. Get all active rules for this tenant and ruleID (or global)
	// For performance, we could cache these, but let's start simple.
	rules, err := s.repo.List(ctx)
	if err != nil {
		s.log.Error("Failed to list suppression rules: %v", err)
		return false, ""
	}

	now := time.Now()

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		// Check expiration
		if rule.ExpiresAt != "" {
			expires, err := time.Parse(time.RFC3339, rule.ExpiresAt)
			if err == nil && now.After(expires) {
				continue
			}
		}

		// Check RuleID scope
		if rule.RuleID != "" && rule.RuleID != ruleID {
			continue
		}

		// Check Field match
		strVal, ok := metadata[rule.Field]
		if !ok {
			continue
		}

		match := false
		if rule.IsRegex {
			re, err := regexp.Compile(rule.Value)
			if err == nil && re.MatchString(strVal) {
				match = true
			}
		} else if rule.Value == strVal {
			match = true
		}

		if match {
			// Update last matched time asynchronously
			go func(ruleID string) {
				_ = s.repo.MarkMatched(context.Background(), ruleID)
			}(rule.ID)
			s.bumpMatch(rule.ID)
			return true, rule.Label
		}
	}

	return false, ""
}

func (s *SuppressionService) CreateRule(ctx context.Context, rule *database.SuppressionRule) (string, error) {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	if rule.CreatedAt == "" {
		rule.CreatedAt = time.Now().Format(time.RFC3339)
	}
	rule.UpdatedAt = time.Now().Format(time.RFC3339)
	
	err := s.repo.Upsert(ctx, rule)
	if err != nil {
		return "", err
	}
	return rule.ID, nil
}

func (s *SuppressionService) ListRules(ctx context.Context) ([]database.SuppressionRule, error) {
	return s.repo.List(ctx)
}

func (s *SuppressionService) DeleteRule(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *SuppressionService) ToggleRule(ctx context.Context, id string, active bool) error {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil || rule == nil {
		return err
	}
	rule.IsActive = active
	return s.repo.Upsert(ctx, rule)
}
