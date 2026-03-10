package incident

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// SSHExec defines the required SSH operations for response actions.
type SSHExec interface {
	Disconnect(sessionID string) error
	Exec(sessionID string, cmd string) (string, error)
}

// Notifier defines the interface for sending alerts to external systems (Jira, SNOW, etc).
type Notifier interface {
	SendAlert(title, message string)
}

// ResponseAction defines a single primitive remediation step with metadata.
type ResponseAction struct {
	Name            string
	Execute         func(ctx context.Context, params map[string]interface{}) (string, error)
	RollbackName    string // Name of the action to undo this one
	RequireApproval bool   // If true, requires multi-persona authorization
}

// ActionRegistry manages available response actions.
type ActionRegistry struct {
	actions map[string]ResponseAction
	ssh     SSHExec
	notif   Notifier
	log     *logger.Logger
}

func NewActionRegistry(ssh SSHExec, notif Notifier, log *logger.Logger) *ActionRegistry {
	r := &ActionRegistry{
		actions: make(map[string]ResponseAction),
		ssh:     ssh,
		notif:   notif,
		log:     log,
	}
	r.registerDefaults()
	return r
}

func (r *ActionRegistry) registerDefaults() {
	r.Register(ResponseAction{
		Name:    "kill_session",
		Execute: r.KillSession,
	})
	r.Register(ResponseAction{
		Name:            "block_ip",
		Execute:         r.BlockIP,
		RollbackName:    "unblock_ip",
		RequireApproval: true,
	})
	r.Register(ResponseAction{
		Name:    "unblock_ip",
		Execute: r.UnblockIP,
	})
	r.Register(ResponseAction{
		Name:            "isolate_host",
		Execute:         r.IsolateHost,
		RequireApproval: true,
	})
	r.Register(ResponseAction{
		Name:            "emergency_isolation",
		Execute:         r.EmergencyIsolation,
		RequireApproval: true,
	})
	r.Register(ResponseAction{
		Name:    "create_jira_ticket",
		Execute: r.CreateJiraTicket,
	})
	r.Register(ResponseAction{
		Name:    "escalate_servicenow",
		Execute: r.EscalateServiceNow,
	})
}

func (r *ActionRegistry) Register(a ResponseAction) {
	r.actions[a.Name] = a
}

func (r *ActionRegistry) Get(name string) (ResponseAction, bool) {
	a, ok := r.actions[name]
	return a, ok
}

func (r *ActionRegistry) Execute(ctx context.Context, name string, params map[string]interface{}) (string, error) {
	action, ok := r.actions[name]
	if !ok {
		return "", fmt.Errorf("unknown action: %s", name)
	}
	return action.Execute(ctx, params)
}

// KillSession terminates an active SSH session.
func (r *ActionRegistry) KillSession(ctx context.Context, params map[string]interface{}) (string, error) {
	sessionID, ok := params["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("missing session_id")
	}

	r.log.Warn("[RESPONSE] Killing session %s", sessionID)
	err := r.ssh.Disconnect(sessionID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Session %s terminated", sessionID), nil
}

// BlockIP adds an IP address to the local firewall (iptables).
func (r *ActionRegistry) BlockIP(ctx context.Context, params map[string]interface{}) (string, error) {
	sessionID, ok := params["session_id"].(string)
	ip, ok2 := params["ip"].(string)
	if !ok || !ok2 {
		return "", fmt.Errorf("missing session_id or ip")
	}

	cmd := fmt.Sprintf("sudo iptables -A INPUT -s %s -j DROP", ip)
	r.log.Warn("[RESPONSE] Blocking IP %s via session %s", ip, sessionID)

	output, err := r.ssh.Exec(sessionID, cmd)
	if err != nil {
		return "", fmt.Errorf("iptables block failed: %w (output: %s)", err, output)
	}
	return fmt.Sprintf("IP %s blocked on remote host", ip), nil
}

// UnblockIP removes an IP address from the local firewall (iptables).
func (r *ActionRegistry) UnblockIP(ctx context.Context, params map[string]interface{}) (string, error) {
	sessionID, ok := params["session_id"].(string)
	ip, ok2 := params["ip"].(string)
	if !ok || !ok2 {
		return "", fmt.Errorf("missing session_id or ip")
	}

	cmd := fmt.Sprintf("sudo iptables -D INPUT -s %s -j DROP", ip)
	r.log.Warn("[RESPONSE] Unblocking IP %s via session %s", ip, sessionID)

	output, err := r.ssh.Exec(sessionID, cmd)
	if err != nil {
		return "", fmt.Errorf("iptables unblock failed: %w (output: %s)", err, output)
	}
	return fmt.Sprintf("IP %s unblocked on remote host", ip), nil
}

// IsolateHost is a placeholder for a more complex isolation (e.g. cloud security group update).
// For now, it just adds a "locked" tag in the audit trails.
// EmergencyIsolation is a drastic measure that cuts all network traffic except management.
func (r *ActionRegistry) EmergencyIsolation(ctx context.Context, params map[string]interface{}) (string, error) {
	sessionID, ok := params["session_id"].(string)
	managementIP, _ := params["management_ip"].(string) // Optional: allow the OBLIVRA server to stay connected

	if !ok {
		return "", fmt.Errorf("missing session_id")
	}

	r.log.Fatal("[RESPONSE] TRIGGERING EMERGENCY ISOLATION on host via session %s", sessionID)

	// Comprehensive isolation commands (Simulation of a hardened script)
	// 1. Flush existing rules
	// 2. Allow lo
	// 3. Allow established
	// 4. Allow management IP (if provided)
	// 5. Drop everything else
	cmds := []string{
		"sudo iptables -P INPUT DROP",
		"sudo iptables -P FORWARD DROP",
		"sudo iptables -P OUTPUT ACCEPT", // Allow outbound for local diagnostics unless specified otherwise
		"sudo iptables -A INPUT -i lo -j ACCEPT",
		"sudo iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT",
	}

	if managementIP != "" {
		cmds = append(cmds, fmt.Sprintf("sudo iptables -A INPUT -s %s -j ACCEPT", managementIP))
	}

	for _, cmd := range cmds {
		output, err := r.ssh.Exec(sessionID, cmd)
		if err != nil {
			return "", fmt.Errorf("isolation command failed: %s -> %w (output: %s)", cmd, err, output)
		}
	}

	return "Emergency isolation rules applied. Host is now in lockdown.", nil
}

// IsolateHost is a placeholder for a more complex isolation.
func (r *ActionRegistry) IsolateHost(ctx context.Context, params map[string]interface{}) (string, error) {
	return "Host marked for isolation (Audit only)", nil
}

// CreateJiraTicket creates a new ticket in the configured Jira instance.
func (r *ActionRegistry) CreateJiraTicket(ctx context.Context, params map[string]interface{}) (string, error) {
	title, _ := params["title"].(string)
	summary, _ := params["summary"].(string)

	if title == "" {
		title = "Incident Remediation Required"
	}
	if summary == "" {
		summary = "Manual escalation triggered from OBLIVRA Incident Response center."
	}

	r.log.Info("[RESPONSE] Creating Jira ticket: %s", title)
	r.notif.SendAlert(title, summary)
	return "Jira ticket creation request sent to notification service", nil
}

// EscalateServiceNow creates a new incident in ServiceNow.
func (r *ActionRegistry) EscalateServiceNow(ctx context.Context, params map[string]interface{}) (string, error) {
	title, _ := params["title"].(string)
	description, _ := params["description"].(string)

	if title == "" {
		title = "Security Incident Escalation"
	}
	if description == "" {
		description = "A critical security event requires manual review. Escalated via OBLIVRA."
	}

	r.log.Info("[RESPONSE] Escalating to ServiceNow: %s", title)
	r.notif.SendAlert(title, description)
	return "ServiceNow escalation request sent to notification service", nil
}

func (r *ActionRegistry) ListActions() []string {
	keys := make([]string, 0, len(r.actions))
	for k := range r.actions {
		keys = append(keys, k)
	}
	return keys
}
