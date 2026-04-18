package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/notifications"
	"github.com/kingknull/oblivrashell/internal/osquery"
	"github.com/kingknull/oblivrashell/internal/platform"
	"path/filepath"
)

const (
	configKeyNotification = "notification_config"
	configKeyTriggers     = "custom_triggers"
	configKeyMetricTrigs  = "metric_triggers"
)

// persistedTrigger is the JSON-safe version of a trigger for storage
type persistedTrigger struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Pattern  string `json:"pattern"`
	Severity string `json:"severity"`
}

// AlertingService exposes alert and notification configuration to the frontend
type AlertingService struct {
	BaseService
	ctx           context.Context
	alerts        analytics.AlertProvider
	notifier      notifications.Notifier
	analytics     analytics.Engine
	siemRepo      database.SIEMStore
	incidents     IncidentManager
	evaluator     *detection.Evaluator
	bus           *eventbus.Bus
	log           *logger.Logger
	sigmaWatcher  *fsnotify.Watcher // hot-reload watcher; nil if sigma dir absent
	sigmaDir      string            // absolute path being watched
	corrHub       *detection.CorrelationHub
	suppress      *SuppressionService
}

func (s *AlertingService) Name() string { return "alerting-service" }

// Dependencies returns service dependencies.
// siem-service must be started before alerting can process SIEM events.
// eventbus is infrastructure wired at construction time, not a kernel-managed service.
func (s *AlertingService) Dependencies() []string {
	return []string{"siem-service"}
}

func NewAlertingService(alerts analytics.AlertProvider, notifier notifications.Notifier, ae analytics.Engine, sr database.SIEMStore, inc IncidentManager, evaluator *detection.Evaluator, bus *eventbus.Bus, log *logger.Logger) *AlertingService {
	return &AlertingService{
		alerts:    alerts,
		notifier:  notifier,
		analytics: ae,
		siemRepo:  sr,
		incidents: inc,
		evaluator: evaluator,
		bus:       bus,
		log:       log,
	}
}

func (s *AlertingService) SetSuppressionService(svc *SuppressionService) {
	s.suppress = svc
}

func (s *AlertingService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.loadPersistedConfig()

	// Load community Sigma rules from the user's sigma/ directory alongside builtin rules.
	// We use the persistent data directory (e.g. %LOCALAPPDATA%/oblivra/data/sigma) 
	// to ensure rules survive restarts and binary movements.
	if s.evaluator != nil {
		sigmaDir := filepath.Join(platform.DataDir(), "sigma")
		s.sigmaDir = sigmaDir

		// Seeding phase: Copy core rules from the binary distribution to DataDir on first run
		s.seedSigmaRules(sigmaDir)

		if err := s.evaluator.LoadSigmaDirectory(sigmaDir); err != nil {
			s.log.Warn("[SIGMA] Failed to load sigma directory: %v", err)
		} else {
			s.log.Info("[SIGMA] Community Sigma rules loaded from %s (%d total rules active)",
				sigmaDir, len(s.evaluator.GetRules()))
			// Start filesystem watcher for hot-reload on .yml changes
			s.startSigmaWatcher(ctx)

			// Initialize and start the CorrelationHub for cross-shard rules
			// We clone the global evaluator and put it into HUB mode (IsLocal=false)
			s.corrHub = detection.NewCorrelationHub(s.evaluator.Clone(), s.bus, s.log)
			s.corrHub.Start(ctx)
		}
	}

	// Listen for heuristic security alerts from SIEMService
	s.bus.Subscribe("security.alert", func(e eventbus.Event) {
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			return
		}

		hostID, _ := data["host_id"].(string)
		score, _ := data["score"].(int)
		msg, _ := data["message"].(string)
		tenantID, _ := data["tenant_id"].(string)

		if tenantID == "" {
			tenantID = "default_tenant"
		}

		// Create a tenant-scoped context for incident generation
		ctx := database.WithTenant(s.ctx, tenantID)

		ruleID := "heuristic_risk"
		groupKey := hostID

		// Check suppression
		if s.suppress != nil {
			meta := map[string]string{
				"host_id": hostID,
				"message": msg,
				"score":   fmt.Sprintf("%d", score),
			}
			if suppressed, label := s.suppress.ShouldSuppress(ctx, ruleID, meta); suppressed {
				s.log.Info("Heuristic alert suppressed by rule: %s", label)
				return
			}
		}

		incident, err := s.incidents.GetByRuleAndGroup(ctx, ruleID, groupKey)
		if err != nil {
			s.log.Error("Failed to lookup incident: %v", err)
			return
		}

		isNew := false
		if incident == nil {
			isNew = true
			incident = &database.Incident{
				ID:          fmt.Sprintf("INC-%d", time.Now().UnixNano()),
				TenantID:    tenantID,
				RuleID:      ruleID,
				GroupKey:    groupKey,
				Status:      "New",
				Severity:    "high", // Could scale based on score
				Description: msg,
				Title:       fmt.Sprintf("Heuristic Security Alert: Host %s", hostID),
				FirstSeenAt: time.Now().Format(time.RFC3339),
				EventCount:  0,
			}
		}

		incident.LastSeenAt = time.Now().Format(time.RFC3339)
		incident.EventCount++

		if err := s.incidents.Upsert(ctx, incident); err != nil {
			s.log.Error("Failed to upsert heuristic incident: %v", err)
		}

		// Notify user only on net-new incidents to avoid spam
		if isNew {
			go s.notifier.SendAlert(incident.Title, fmt.Sprintf("%s\nCalculated Risk Score: %d/100", msg, score))
		}
	})

	// Listen for matches from parallel shard evaluators
	s.bus.Subscribe("detection.match", func(e eventbus.Event) {
		match, ok := e.Data.(detection.Match)
		if !ok {
			return
		}

		ctx := database.WithTenant(s.ctx, match.TenantID)
		groupKey := match.Context["group_key"]

		// Check suppression
		if s.suppress != nil {
			if suppressed, label := s.suppress.ShouldSuppress(ctx, match.RuleID, match.Context); suppressed {
				s.log.Info("Detection alert (%s) suppressed by rule: %s", match.RuleID, label)
				return
			}
		}

		incident, err := s.incidents.GetByRuleAndGroup(ctx, match.RuleID, groupKey)
		if err != nil {
			s.log.Error("Failed to lookup incident: %v", err)
			return
		}

		isNew := false
		if incident == nil {
			isNew = true
			incident = &database.Incident{
				ID:              fmt.Sprintf("INC-%d", time.Now().UnixNano()),
				TenantID:        match.TenantID,
				RuleID:          match.RuleID,
				GroupKey:        groupKey,
				Status:          "New",
				Severity:        match.Severity,
				Description:     match.Description,
				Title:           fmt.Sprintf("Detection Alert: %s (Entity: %s)", match.RuleName, groupKey),
				FirstSeenAt:     time.Now().Format(time.RFC3339),
				EventCount:      0,
				MitreTactics:    match.MitreTactics,
				MitreTechniques: match.MitreTechniques,
			}
		}

		incident.LastSeenAt = time.Now().Format(time.RFC3339)
		incident.EventCount++

		if err := s.incidents.Upsert(ctx, incident); err != nil {
			s.log.Error("Failed to upsert detection incident: %v", err)
		}

		if isNew {
			go s.notifier.SendAlert(incident.Title, fmt.Sprintf("Severity: %s\nRule: %s\nDetails: %s", match.Severity, match.RuleID, match.Description))
		}
	})

	// Start a background worker for global aggregate alerts (e.g. 50+ failures across fleet in 5m)
	go s.scanGlobalThreatsLoop(ctx)
	return nil
}

func (s *AlertingService) Stop(ctx context.Context) error {
	if s.sigmaWatcher != nil {
		_ = s.sigmaWatcher.Close()
		s.sigmaWatcher = nil
		s.log.Info("[SIGMA] Hot-reload watcher stopped")
	}
	return nil
}

// startSigmaWatcher watches sigmaDir for .yml file changes and reloads rules
// with a 500 ms debounce so rapid saves don't trigger multiple reloads.
func (s *AlertingService) startSigmaWatcher(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		s.log.Warn("[SIGMA] Could not create fsnotify watcher: %v", err)
		return
	}
	if err := watcher.Add(s.sigmaDir); err != nil {
		_ = watcher.Close()
		s.log.Warn("[SIGMA] Could not watch sigma dir %s: %v", s.sigmaDir, err)
		return
	}
	s.sigmaWatcher = watcher
	s.log.Info("[SIGMA] Hot-reload watcher active on %s", s.sigmaDir)

	go func() {
		var debounce *time.Timer
		defer func() {
			if debounce != nil {
				debounce.Stop()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Only react to .yml / .yaml files being written, created or removed
				n := event.Name
				if !(
					(len(n) > 4 && n[len(n)-4:] == ".yml") ||
					(len(n) > 5 && n[len(n)-5:] == ".yaml")) {
					continue
				}
				if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Remove) {
					continue
				}
				// Debounce: reset timer so rapid successive saves only trigger one reload
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(500*time.Millisecond, func() {
					s.reloadSigmaRules()
				})

			case watchErr, ok := <-watcher.Errors:
				if !ok {
					return
				}
				s.log.Warn("[SIGMA] Watcher error: %v", watchErr)
			}
		}
	}()
}

// reloadSigmaRules re-reads every .yml file in sigmaDir and atomically
// replaces the active rule set. Called by the debounced watcher and also
// exposed as a Wails-callable method so the UI can trigger manual reloads.
func (s *AlertingService) reloadSigmaRules() {
	if s.evaluator == nil || s.sigmaDir == "" {
		return
	}
	if err := s.evaluator.LoadSigmaDirectory(s.sigmaDir); err != nil {
		s.log.Warn("[SIGMA] Hot-reload failed: %v", err)
		s.bus.Publish("sigma:reload_error", map[string]string{"error": err.Error()})
		return
	}
	count := len(s.evaluator.GetRules())
	// Rebuild the EventType routing index so new rules are picked up immediately.
	s.evaluator.RebuildRouteIndex()
	s.log.Info("[SIGMA] Hot-reload complete — %d rules active", count)
	s.bus.Publish("sigma:rules_reloaded", map[string]interface{}{"rule_count": count, "dir": s.sigmaDir})
	EmitEvent("sigma:rules_reloaded", map[string]interface{}{"rule_count": count})
}

// ReloadSigmaRules is the Wails-facing manual reload trigger.
func (s *AlertingService) ReloadSigmaRules() int {
	s.reloadSigmaRules()
	if s.evaluator != nil {
		return len(s.evaluator.GetRules())
	}
	return 0
}

// seedSigmaRules copies foundational rules from the relative 'sigma/core' directory
// to the persistent application data directory if the destination is empty.
func (s *AlertingService) seedSigmaRules(destDir string) {
	// Check if the destination already has rules
	entries, err := os.ReadDir(destDir)
	if err == nil && len(entries) > 0 {
		return // Already seeded or user has added rules
	}

	// Source: looking for foundational rules bundled with the binary
	srcDir := filepath.Join("sigma", "core")
	srcEntries, err := os.ReadDir(srcDir)
	if err != nil {
		s.log.Warn("[SIGMA] Source core rules directory not found, skipping auto-seed: %v", err)
		return
	}

	s.log.Info("[SIGMA] First run detected: Seeding %d core rules to %s", len(srcEntries), destDir)
	for _, entry := range srcEntries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if err := copyFile(srcPath, destPath); err != nil {
			s.log.Warn("[SIGMA] Failed to seed rule %s: %v", entry.Name(), err)
		}
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (s *AlertingService) scanGlobalThreatsLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.CheckGlobalSecurityThresholds()
		}
	}
}

// CheckGlobalSecurityThresholds evaluates fleet-wide security posture
func (s *AlertingService) CheckGlobalSecurityThresholds() {
	if s.siemRepo == nil {
		return
	}

	// For global threats, we iterate through all tenants or use a system-level context.
	// Since thresholds like "100 failures across fleet" are infrastructure-level,
	// we use a global search context.
	ctx, cancel := context.WithTimeout(database.WithGlobalSearch(s.ctx), 10*time.Second)
	defer cancel()

	stats, err := s.siemRepo.GetGlobalThreatStats(ctx)
	if err != nil {
		return
	}

	totalFailures := 0
	if tf, ok := stats["total_failed_logins"].(int); ok {
		totalFailures = tf
	}

	// Threshold: Alert if more than 100 failed logins currently detected globally
	if totalFailures > 100 {
		s.notifier.SendAlert(
			"Fleet-Wide Critical Threat",
			fmt.Sprintf("Global failed logins exceed threshold: %d total attempts detected across fleet.", totalFailures),
		)
	}
}

// loadPersistedConfig restores notification config and custom triggers from SQLite
func (s *AlertingService) loadPersistedConfig() {
	if s.analytics == nil {
		return
	}

	// Configuration loading is considered a system-level operation on start.
	// We use the service's lifetime context (usually default tenant)
	ctx := database.WithGlobalSearch(s.ctx)

	// Load notification config
	if data, err := s.analytics.LoadConfig(ctx, configKeyNotification); err == nil {
		var cfg notifications.NotificationConfig
		if json.Unmarshal([]byte(data), &cfg) == nil {
			s.notifier.UpdateConfig(cfg)
			s.log.Info("Restored notification config from database")
		}
	}

	// Load metric triggers
	if data, err := s.analytics.LoadConfig(ctx, configKeyMetricTrigs); err == nil {
		var mTriggers []analytics.MetricTrigger
		if json.Unmarshal([]byte(data), &mTriggers) == nil {
			s.alerts.SetMetricTriggers(mTriggers)
			s.log.Info("Restored %d metric triggers from database", len(mTriggers))
		}
	}

	// Load custom triggers
	if data, err := s.analytics.LoadConfig(ctx, configKeyTriggers); err == nil {
		var triggers []persistedTrigger
		if json.Unmarshal([]byte(data), &triggers) == nil {
			for _, t := range triggers {
				s.alerts.AddTrigger(t.ID, t.Name, t.Pattern, t.Severity)
			}
			s.log.Info("Restored %d custom triggers from database", len(triggers))
		}
	}
}

func (s *AlertingService) persistTriggers(ctx context.Context) {
	if s.analytics == nil {
		return
	}
	triggers := s.alerts.GetTriggers()
	var custom []persistedTrigger
	for _, t := range triggers {
		// Only persist non-builtin triggers
		if len(t.ID) > 8 && t.ID[:8] == "builtin-" {
			continue
		}
		custom = append(custom, persistedTrigger{
			ID: t.ID, Name: t.Name, Pattern: t.RawExpr, Severity: t.Severity,
		})
	}
	if data, err := json.Marshal(custom); err == nil {
		s.analytics.SaveConfig(ctx, configKeyTriggers, string(data))
	}
}

func (s *AlertingService) persistNotificationConfig(ctx context.Context) {
	if s.analytics == nil {
		return
	}
	cfg := s.notifier.GetConfig()
	if data, err := json.Marshal(cfg); err == nil {
		s.analytics.SaveConfig(ctx, configKeyNotification, string(data))
	}
}

// --- Alert Triggers ---

func (s *AlertingService) AddTrigger(ctx context.Context, id, name, pattern, severity string) error {
	err := s.alerts.AddTrigger(id, name, pattern, severity)
	if err != nil {
		return err
	}
	s.log.Info("Alert trigger added: %s (%s)", name, severity)
	s.persistTriggers(ctx)
	return nil
}

func (s *AlertingService) RemoveTrigger(ctx context.Context, id string) {
	s.alerts.RemoveTrigger(id)
	s.log.Info("Alert trigger removed: %s", id)
	s.persistTriggers(ctx)
}

func (s *AlertingService) GetTriggers() []analytics.Trigger {
	return s.alerts.GetTriggers()
}

// --- Metric Triggers ---

func (s *AlertingService) GetMetricTriggers() []analytics.MetricTrigger {
	return s.alerts.GetMetricTriggers()
}

func (s *AlertingService) UpdateMetricTrigger(ctx context.Context, mt analytics.MetricTrigger) {
	s.alerts.UpdateMetricTrigger(mt)
	s.persistMetricTriggers(ctx)
}

func (s *AlertingService) RemoveMetricTrigger(ctx context.Context, id string) {
	s.alerts.RemoveMetricTrigger(id)
	s.persistMetricTriggers(ctx)
}

func (s *AlertingService) persistMetricTriggers(ctx context.Context) {
	if s.analytics == nil {
		return
	}
	trigs := s.alerts.GetMetricTriggers()
	var custom []analytics.MetricTrigger
	for _, t := range trigs {
		if len(t.ID) > 3 && t.ID[:3] == "mt-" { // builtin prefix check
			continue
		}
		custom = append(custom, t)
	}
	if data, err := json.Marshal(custom); err == nil {
		s.analytics.SaveConfig(ctx, configKeyMetricTrigs, string(data))
	}
}

// --- Alert History (now from SQLite) ---

func (s *AlertingService) GetAlertHistory(ctx context.Context) []map[string]interface{} {
	if s.analytics == nil {
		return nil
	}
	history, err := s.analytics.GetAlertHistory(ctx, 500)
	if err != nil {
		s.log.Error("Failed to load alert history: %v", err)
		return nil
	}
	return history
}

// GetDetectionRules exposes active YAML rules (including MITRE info) to the UI
func (s *AlertingService) GetDetectionRules() []detection.Rule {
	if s.evaluator != nil {
		return s.evaluator.GetRules()
	}
	return nil
}

// GetRuleVerifications exposes the AST static analysis results to the Policy Verifier UI
func (s *AlertingService) GetRuleVerifications() []detection.ValidationResult {
	if s.evaluator != nil {
		return s.evaluator.GetVerificationResults()
	}
	return nil
}

// --- Notification Config ---

func (s *AlertingService) UpdateNotificationConfig(ctx context.Context, cfg notifications.NotificationConfig) {
	s.notifier.UpdateConfig(cfg)
	s.persistNotificationConfig(ctx)
	s.log.Info("Notification config updated and persisted")
}

func (s *AlertingService) GetNotificationConfig() notifications.NotificationConfig {
	return s.notifier.GetConfig()
}

// TestNotification sends a test alert through all enabled channels
func (s *AlertingService) TestNotification() {
	s.notifier.SendAlert("Test Alert", "This is a test notification from OblivraShell. If you're reading this, notifications are working correctly!")
	s.log.Info("Test notification dispatched")
}

// --- Incident Management ---

// ListIncidents retrieves security incidents filtered by status
func (s *AlertingService) ListIncidents(ctx context.Context, status string, limit int) ([]database.Incident, error) {
	if s.incidents == nil {
		return nil, fmt.Errorf("incident store not initialized")
	}

	return s.incidents.Search(ctx, status, "", limit)
}

// UpdateIncidentStatus transitions an incident to a new status
func (s *AlertingService) UpdateIncidentStatus(ctx context.Context, id string, status string, reason string) error {
	if s.incidents == nil {
		return fmt.Errorf("incident store not initialized")
	}

	err := s.incidents.UpdateStatus(ctx, id, status, reason)
	if err == nil {
		s.log.Info("Incident %s status updated to %s: %s", id, status, reason)
	}
	return err
}

// --- Osquery Library ---

func (s *AlertingService) GetOsqueryTemplates() []osquery.QueryTemplate {
	return osquery.GetDefaultQueries()
}

// GetEvaluator returns the underlying detection evaluator.
// Used by integration tests to directly exercise sigma loading.
func (s *AlertingService) GetEvaluator() *detection.Evaluator {
	return s.evaluator
}
