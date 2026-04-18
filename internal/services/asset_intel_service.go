package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// AssetIntelService calculates and maintains asset criticality scores.
type AssetIntelService struct {
	BaseService
	hosts database.HostStore
	users database.UserStore
	log   *logger.Logger
}

func NewAssetIntelService(hosts database.HostStore, users database.UserStore, log *logger.Logger) *AssetIntelService {
	return &AssetIntelService{
		hosts: hosts,
		users: users,
		log:   log.WithPrefix("asset_intel"),
	}
}

func (s *AssetIntelService) Name() string { return "asset-intel-service" }

func (s *AssetIntelService) Start(ctx context.Context) error {
	s.log.Info("Asset Intelligence service starting...")
	
	// Initial refresh
	go func() {
		// Wait for DB to be ready
		time.Sleep(5 * time.Second)
		if err := s.RefreshAll(ctx); err != nil {
			s.log.Error("Initial criticality refresh failed: %v", err)
		}
	}()

	// Periodic refresh ticker (every 6 hours)
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.RefreshAll(ctx); err != nil {
					s.log.Error("Periodic criticality refresh failed: %v", err)
				}
			}
		}
	}()

	return nil
}

// CalculateHostCriticality determines the score (1-10) for a host.
func (s *AssetIntelService) CalculateHostCriticality(ctx context.Context, host *database.Host) (int, string) {
	score := 1
	var reasons []string

	// 1. Label-based scoring
	label := strings.ToLower(host.Label)
	if strings.Contains(label, "domain controller") || strings.Contains(label, "dc") {
		score = 10
		reasons = append(reasons, "Domain Controller detected via label")
	} else if strings.Contains(label, "production") || strings.Contains(label, "prod") {
		score = 8
		reasons = append(reasons, "Production environment")
	} else if strings.Contains(label, "critical") {
		score = 9
		reasons = append(reasons, "Explicitly marked as Critical")
	}

	// 2. Tag-based scoring
	for _, tag := range host.Tags {
		t := strings.ToLower(tag)
		if t == "database" || t == "db" {
			if score < 7 { score = 7 }
			reasons = append(reasons, "Database role via tag")
		}
		if t == "gateway" || t == "bastion" || t == "vpn" {
			if score < 8 { score = 8 }
			reasons = append(reasons, "Infrastructure bottleneck/gateway")
		}
	}

	// 3. Category-based
	if host.Category == "Server" && score < 3 {
		score = 3
		reasons = append(reasons, "Server infrastructure")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "Default low criticality")
	}

	return score, strings.Join(reasons, "; ")
}

// CalculateUserCriticality determines the score (1-10) for a user identity.
func (s *AssetIntelService) CalculateUserCriticality(ctx context.Context, user *database.User) (int, string) {
	score := 1
	var reasons []string

	// 1. Role/Type based
	uType := strings.ToLower(user.UserType)
	switch uType {
	case "admin", "superuser":
		score = 9
		reasons = append(reasons, "Privileged administrative account")
	case "service":
		score = 6
		reasons = append(reasons, "Service account with automated access")
	case "executive", "vip":
		score = 8
		reasons = append(reasons, "High-profile identity (Executive/VIP)")
	}

	// 2. Department based
	dept := strings.ToLower(user.Department)
	switch dept {
	case "it", "security", "engineering":
		if score < 5 {
			score = 5
		}
		reasons = append(reasons, fmt.Sprintf("Sensitive department access (%s)", user.Department))
	case "finance", "hr":
		if score < 6 {
			score = 6
		}
		reasons = append(reasons, "Access to sensitive organizational data")
	}

	// 3. MFA Status
	if !user.IsMFAEnabled && score >= 5 {
		reasons = append(reasons, "High criticality asset WITHOUT MFA enabled")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "Standard user identity")
	}

	return score, strings.Join(reasons, "; ")
}

func (s *AssetIntelService) RefreshAll(ctx context.Context) error {
	s.log.Info("Re-evaluating all asset criticality scores...")
	
	// Refresh Hosts
	hosts, err := s.hosts.GetAll(ctx)
	if err == nil {
		for i := range hosts {
			h := &hosts[i]
			score, reason := s.CalculateHostCriticality(ctx, h)
			if h.CriticalityScore != score || h.CriticalityReason != reason {
				h.CriticalityScore = score
				h.CriticalityReason = reason
				s.hosts.Update(ctx, h)
			}
		}
	}

	// Refresh Users
	users, err := s.users.ListUsers(ctx)
	if err == nil {
		for i := range users {
			u := &users[i]
			score, reason := s.CalculateUserCriticality(ctx, u)
			if u.CriticalityScore != score || u.CriticalityReason != reason {
				u.CriticalityScore = score
				u.CriticalityReason = reason
				s.users.UpdateUser(ctx, u)
			}
		}
	}

	s.log.Info("Asset criticality refresh complete.")
	return nil
}
