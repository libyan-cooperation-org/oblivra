package services

import (
	"context"
	"regexp"
	"sync"
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
}

func NewSuppressionService(repo database.SuppressionStore, log *logger.Logger) *SuppressionService {
	return &SuppressionService{
		repo: repo,
		log:  log.WithPrefix("suppression_service"),
	}
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
