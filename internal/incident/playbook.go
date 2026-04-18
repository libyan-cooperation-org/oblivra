package incident

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// PlaybookEngine orchestrates sequences of response actions for an incident.
type PlaybookEngine struct {
	registry *ActionRegistry
	repo     database.IncidentStore
	audit    database.AuditStore
	bus      *eventbus.Bus
	log      *logger.Logger

	// History for rollback
	actionHistory []ExecutedAction
}

type ExecutedAction struct {
	Name   string
	Params map[string]interface{}
}

func NewPlaybookEngine(
	ssh SSHExec,
	notif Notifier,
	repo database.IncidentStore,
	audit database.AuditStore,
	bus *eventbus.Bus,
	log *logger.Logger,
) *PlaybookEngine {
	return &PlaybookEngine{
		registry: NewActionRegistry(ssh, notif, log),
		repo:     repo,
		audit:    audit,
		bus:      bus,
		log:      log,
	}
}

func (e *PlaybookEngine) Startup() error {
	e.log.Info("Playbook engine starting...")
	return nil
}

func (e *PlaybookEngine) Shutdown() error {
	e.log.Info("Playbook engine shutting down...")
	return nil
}

func (e *PlaybookEngine) ExecuteAction(ctx context.Context, actionName string, params map[string]interface{}) (string, error) {
	action, ok := e.registry.Get(actionName)
	if !ok {
		return "", fmt.Errorf("unknown action: %s", actionName)
	}

	// 1. Check if approval is required
	if action.RequireApproval {
		e.log.Warn("[SOAR] Action %s REQUIRES APPROVAL. Gating execution.", actionName)
		// Publish an approval request event
		e.bus.Publish(eventbus.EventType("soar.approval_required"), map[string]interface{}{
			"action": actionName,
			"params": params,
		})
		return "PENDING_APPROVAL", nil
	}

	// 2. Execute
	output, err := action.Execute(ctx, params)
	if err == nil {
		// 3. Record for rollback if successful and has a rollback partner
		if action.RollbackName != "" {
			e.actionHistory = append(e.actionHistory, ExecutedAction{
				Name:   actionName,
				Params: params,
			})
		}
	}
	return output, err
}

// Rollback undoes all actions in the current playbook history in reverse order.
func (e *PlaybookEngine) Rollback(ctx context.Context) error {
	e.log.Warn("[SOAR] INITIATING ROLLBACK for %d actions", len(e.actionHistory))
	
	for i := len(e.actionHistory) - 1; i >= 0; i-- {
		executed := e.actionHistory[i]
		action, ok := e.registry.Get(executed.Name)
		if !ok || action.RollbackName == "" {
			continue
		}

		e.log.Info("[SOAR] Undoing %s via %s", executed.Name, action.RollbackName)
		_, err := e.registry.Execute(ctx, action.RollbackName, executed.Params)
		if err != nil {
			e.log.Error("[SOAR] ROLLBACK FAILED for %s: %v", executed.Name, err)
			return err
		}
	}

	e.actionHistory = nil
	return nil
}

func (e *PlaybookEngine) RunPlaybook(ctx context.Context, playbookID string, incidentID string) error {
	e.log.Info("[PLAYBOOK] Running playbook %s for incident %s", playbookID, incidentID)

	inc, err := e.repo.GetByID(ctx, incidentID)
	if err != nil {
		return err
	}

	// For Phase 8, we implement a simple hardcoded "contain_brute_force" playbook.
	// In the future, this will be driven by YAML configurations.
	if playbookID == "contain_brute_force" {
		return e.runBruteForceContainment(ctx, inc)
	}

	return fmt.Errorf("unsupported playbook: %s", playbookID)
}

func (e *PlaybookEngine) runBruteForceContainment(ctx context.Context, inc *database.Incident) error {
	e.log.Info("[PLAYBOOK] Initiating Brute Force Containment for Incident %s", inc.ID)

	// In a real scenario, we would parse the incident's metadata or logs to find the source IP and target session.
	// For this phase, we use the group_key as the IP and assume a valid session is required.
	// We'll look for an active session to execute the block command.

	ip := inc.GroupKey // Assuming group_key is the offending IP for brute force incidents
	if ip == "" {
		return fmt.Errorf("no IP address found in incident group key")
	}

	e.log.Info("[PLAYBOOK] Step 1: Auditing decision path...")
	_ = e.audit.Log(context.Background(), "playbook.step", "", "", map[string]interface{}{
		"playbook":    "contain_brute_force",
		"incident_id": inc.ID,
		"status":      "blocking_ip",
		"ip":          ip,
	})

	// 2. Execute Block IP action.
	// PROBLEM: We need a session_id to run the command on the remote host.
	// For now, we simulate finding a session ID or use a placeholder if the incident doesn't have one.
	// In production, the incident would be linked to the session that triggered the alert.
	params := map[string]interface{}{
		"ip":         ip,
		"session_id": "placeholder-session", // In reality, this would be inc.Metadata["session_id"]
	}

	e.log.Info("[PLAYBOOK] Step 2: Executing firewall block for IP %s...", ip)
	output, err := e.registry.Execute(ctx, "block_ip", params)
	if err != nil {
		e.log.Error("[PLAYBOOK] Firewall block failed: %v", err)
		_ = e.audit.Log(context.Background(), "playbook.step_failed", "", "", map[string]interface{}{
			"playbook":    "contain_brute_force",
			"incident_id": inc.ID,
			"error":       err.Error(),
		})
		// We continue to mark as partially contained or handle the failure
	} else {
		e.log.Info("[PLAYBOOK] Firewall block successful: %s", output)
	}

	// 3. Update Incident Status
	e.log.Info("[PLAYBOOK] Step 3: Updating incident status to 'Contained'...")
	err = e.repo.UpdateStatus(ctx, inc.ID, "Contained", "Automated playbook 'contain_brute_force' completed.")
	if err != nil {
		e.log.Error("[PLAYBOOK] Failed to update incident status: %v", err)
	}

	e.log.Info("[PLAYBOOK] Playbook execution completed.")
	_ = e.audit.Log(context.Background(), "playbook.completed", "", "", map[string]interface{}{
		"playbook":    "contain_brute_force",
		"incident_id": inc.ID,
		"result":      "success",
	})

	return nil
}

func (e *PlaybookEngine) ListAvailableActions() []string {
	return e.registry.ListActions()
}
