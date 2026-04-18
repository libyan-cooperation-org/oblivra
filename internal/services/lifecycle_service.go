package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// RetentionPolicy defines TTL for a specific data category
type RetentionPolicy struct {
	Category     string `json:"category"`
	Description  string `json:"description"`
	RetainDays   int    `json:"retain_days"`
	ArchiveFirst bool   `json:"archive_first"` // Archive to Parquet before purging
	IsEnabled    bool   `json:"is_enabled"`
}

// LifecycleStats holds metrics from the last purge cycle
type LifecycleStats struct {
	LastRunAt        string           `json:"last_run_at"`
	NextRunAt        string           `json:"next_run_at"`
	CategoriesPurged map[string]int64 `json:"categories_purged"`
	TotalRowsPurged  int64            `json:"total_rows_purged"`
	LegalHoldActive  bool             `json:"legal_hold_active"`
	Errors           []string         `json:"errors"`
}

// DataLifecycleService manages data retention, TTL enforcement, and automated purge
type DataLifecycleService struct {
	BaseService
	db        database.DatabaseStore
	bus       *eventbus.Bus
	log       *logger.Logger
	policies  []RetentionPolicy
	legalHold bool
	stats     LifecycleStats
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func (s *DataLifecycleService) Name() string { return "lifecycle-service" }

// Dependencies returns service dependencies.
func (s *DataLifecycleService) Dependencies() []string {
	return []string{}
}

// DefaultPolicies returns the baseline retention policies
func DefaultPolicies() []RetentionPolicy {
	return []RetentionPolicy{
		{Category: "sessions", Description: "SSH session metadata", RetainDays: 90, ArchiveFirst: false, IsEnabled: true},
		{Category: "audit_logs", Description: "Audit trail entries", RetainDays: 365, ArchiveFirst: true, IsEnabled: true},
		{Category: "host_events", Description: "SIEM/host security events", RetainDays: 180, ArchiveFirst: true, IsEnabled: true},
		{Category: "incidents", Description: "Incident records", RetainDays: 730, ArchiveFirst: true, IsEnabled: true},
		{Category: "config_changes", Description: "Configuration change log", RetainDays: 365, ArchiveFirst: false, IsEnabled: true},
		{Category: "evidence_chain", Description: "Forensic chain of custody", RetainDays: 0, ArchiveFirst: false, IsEnabled: false}, // Never auto-purge evidence
		{Category: "recording_frames", Description: "Terminal recording data", RetainDays: 30, ArchiveFirst: true, IsEnabled: true},
	}
}

func NewDataLifecycleService(db database.DatabaseStore, bus *eventbus.Bus, log *logger.Logger) *DataLifecycleService {
	ctx, cancel := context.WithCancel(context.Background())
	return &DataLifecycleService{
		db:       db,
		bus:      bus,
		log:      log.WithPrefix("lifecycle"),
		policies: DefaultPolicies(),
		ctx:      ctx,
		cancel:   cancel,
		stats: LifecycleStats{
			CategoriesPurged: make(map[string]int64),
		},
	}
}

func (s *DataLifecycleService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.wg.Add(1)
	go s.runLoop()
	s.log.Info("Data Lifecycle engine started. %d policies configured.", len(s.policies))
	return nil
}

func (s *DataLifecycleService) Stop(ctx context.Context) error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *DataLifecycleService) runLoop() {
	defer s.wg.Done()

	// Run first purge 5 minutes after startup to avoid boot contention
	firstRun := time.After(5 * time.Minute)
	ticker := time.NewTicker(6 * time.Hour) // Run every 6 hours
	defer ticker.Stop()

	select {
	case <-s.ctx.Done():
		return
	case <-firstRun:
		s.executePurgeCycle()
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.executePurgeCycle()
		}
	}
}

// executePurgeCycle runs all enabled retention policies
func (s *DataLifecycleService) executePurgeCycle() {
	s.mu.Lock()
	if s.legalHold {
		s.log.Warn("LEGAL HOLD active — skipping purge cycle")
		s.stats.LegalHoldActive = true
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.log.Info("Starting data lifecycle purge cycle...")
	start := time.Now()
	var totalPurged int64
	errors := []string{}
	categoriesPurged := make(map[string]int64)

	for _, policy := range s.policies {
		if !policy.IsEnabled || policy.RetainDays <= 0 {
			continue
		}

		cutoff := time.Now().AddDate(0, 0, -policy.RetainDays)
		purged, err := s.purgeCategory(policy.Category, cutoff)
		if err != nil {
			errMsg := fmt.Sprintf("%s: %v", policy.Category, err)
			errors = append(errors, errMsg)
			s.log.Error("Purge failed for %s: %v", policy.Category, err)
			continue
		}

		if purged > 0 {
			categoriesPurged[policy.Category] = purged
			totalPurged += purged
			s.log.Info("Purged %d rows from %s (cutoff: %s)", purged, policy.Category, cutoff.Format("2006-01-02"))
		}
	}

	s.mu.Lock()
	s.stats = LifecycleStats{
		LastRunAt:        start.Format(time.RFC3339),
		NextRunAt:        start.Add(6 * time.Hour).Format(time.RFC3339),
		CategoriesPurged: categoriesPurged,
		TotalRowsPurged:  totalPurged,
		LegalHoldActive:  s.legalHold,
		Errors:           errors,
	}
	s.mu.Unlock()

	elapsed := time.Since(start)
	s.log.Info("Purge cycle complete. Total: %d rows across %d categories in %v", totalPurged, len(categoriesPurged), elapsed)
	s.bus.Publish("lifecycle.purge_complete", map[string]interface{}{
		"total_purged": totalPurged,
		"elapsed_ms":   elapsed.Milliseconds(),
	})
}

// purgeCategory removes expired rows from a specific table
func (s *DataLifecycleService) purgeCategory(category string, cutoff time.Time) (int64, error) {
	s.db.Lock()
	defer s.db.Unlock()

	// SECURITY: Whitelist valid categories to prevent SQL injection
	validCategories := map[string]bool{
		"sessions": true, "audit_logs": true, "host_events": true,
		"incidents": true, "config_changes": true, "evidence_chain": true,
		"recording_frames": true,
	}
	if !validCategories[category] {
		return 0, fmt.Errorf("invalid purge category: %q", category)
	}

	// Determine the timestamp column (most tables use created_at)
	tsCol := "created_at"
	switch category {
	case "sessions":
		tsCol = "started_at"
	case "host_events":
		tsCol = "timestamp"
	case "recording_frames":
		tsCol = "timestamp"
	case "config_changes":
		tsCol = "timestamp"
	case "evidence_chain":
		tsCol = "timestamp"
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s < ?", category, tsCol)
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	result, err := s.db.ReplicatedExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()
	return affected, nil
}

// --- Wails API ---

// GetPolicies returns all retention policies
func (s *DataLifecycleService) GetPolicies() []RetentionPolicy {
	return s.policies
}

// UpdatePolicy modifies a retention policy by category
func (s *DataLifecycleService) UpdatePolicy(category string, retainDays int, enabled bool) error {
	for i, p := range s.policies {
		if p.Category == category {
			s.policies[i].RetainDays = retainDays
			s.policies[i].IsEnabled = enabled
			s.log.Info("Policy updated: %s → %d days, enabled=%v", category, retainDays, enabled)
			s.bus.Publish("lifecycle.policy_updated", category)
			return nil
		}
	}
	return fmt.Errorf("unknown category: %s", category)
}

// GetStats returns the last purge cycle metrics
func (s *DataLifecycleService) GetStats() LifecycleStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// SetLegalHold enables or disables legal hold (prevents all purge)
func (s *DataLifecycleService) SetLegalHold(active bool) {
	s.mu.Lock()
	s.legalHold = active
	s.mu.Unlock()

	status := "DISABLED"
	if active {
		status = "ENABLED"
	}
	s.log.Warn("Legal hold %s — all data purge is %s", status, map[bool]string{true: "SUSPENDED", false: "RESUMED"}[active])
	s.bus.Publish("lifecycle.legal_hold", active)
}

// TriggerPurge manually triggers a purge cycle
func (s *DataLifecycleService) TriggerPurge() {
	s.log.Info("Manual purge triggered")
	go s.executePurgeCycle()
}
