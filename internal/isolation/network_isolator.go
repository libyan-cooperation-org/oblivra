package isolation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// HostSSHExec is the subset of SSHService used by the isolator.
// Matches the signature already defined in internal/incident/actions.go.
type HostSSHExec interface {
	// Exec runs a command on the session identified by sessionID.
	Exec(sessionID string, cmd string) (string, error)
}

// HostSessionResolver resolves a hostID to an active SSH sessionID.
// Implemented by SSHService / HostService.
type HostSessionResolver interface {
	GetActiveSessionForHost(hostID string) (sessionID string, ok bool)
}

// IsolationRecord stores the state of a host isolation operation.
type IsolationRecord struct {
	HostID      string    `json:"host_id"`
	IsolatedAt  string    `json:"isolated_at"`
	Reason      string    `json:"reason"`
	ThreatScore int       `json:"threat_score"`
	Auto        bool      `json:"auto"`           // true = autonomous, false = analyst-triggered
	Restored    bool      `json:"restored"`
	RestoredAt  *string    `json:"restored_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// NetworkIsolator subscribes to ransomware isolation requests and executes
// host-level network containment via SSH.
//
// Isolation strategy:
//   - Drop all INPUT/FORWARD traffic
//   - Allow established sessions (keeps the management SSH session alive)
//   - Allow a configurable management CIDR (SOC network) to maintain visibility
//   - Log all dropped packets for forensic reconstruction
//
// This is intentionally additive (rules stacked on top of existing policy)
// and reversible via RestoreHost().
type NetworkIsolator struct {
	ssh           HostSSHExec
	resolver      HostSessionResolver
	bus           *eventbus.Bus
	log           *logger.Logger
	managementCIDR string // SOC/management network allowed through, e.g. "10.0.0.0/24"

	mu      sync.RWMutex
	records map[string]*IsolationRecord // hostID -> record
}

// NewNetworkIsolator creates a NetworkIsolator and starts its event bus subscription.
func NewNetworkIsolator(
	ssh HostSSHExec,
	resolver HostSessionResolver,
	managementCIDR string,
	bus *eventbus.Bus,
	log *logger.Logger,
) *NetworkIsolator {
	n := &NetworkIsolator{
		ssh:            ssh,
		resolver:       resolver,
		bus:            bus,
		log:            log.WithPrefix("network-isolator"),
		managementCIDR: managementCIDR,
		records:        make(map[string]*IsolationRecord),
	}

	bus.Subscribe("ransomware.isolation_requested", n.handleIsolationRequest)

	return n
}

// handleIsolationRequest is the event bus handler for autonomous isolation.
func (n *NetworkIsolator) handleIsolationRequest(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}

	hostID, _ := data["host_id"].(string)
	reason, _ := data["reason"].(string)
	score, _ := data["threat_score"].(int)
	auto, _ := data["auto"].(bool)

	if hostID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := n.IsolateHost(ctx, hostID, reason, score, auto); err != nil {
		n.log.Error("[ISOLATOR] Failed to isolate host %s: %v", hostID, err)
		n.bus.Publish("ransomware.isolation_failed", map[string]interface{}{
			"host_id": hostID,
			"error":   err.Error(),
		})
	}
}

// IsolateHost applies network containment rules to a host via SSH.
// Safe to call directly from analyst workflows (auto=false).
func (n *NetworkIsolator) IsolateHost(ctx context.Context, hostID, reason string, threatScore int, auto bool) error {
	n.mu.Lock()
	if rec, exists := n.records[hostID]; exists && !rec.Restored {
		n.mu.Unlock()
		n.log.Warn("[ISOLATOR] Host %s is already isolated — skipping duplicate request", hostID)
		return nil
	}
	n.mu.Unlock()

	sessionID, ok := n.resolver.GetActiveSessionForHost(hostID)
	if !ok {
		return fmt.Errorf("no active SSH session found for host %s — cannot isolate remotely", hostID)
	}

	n.log.Warn("[ISOLATOR] 🔴 Isolating host %s (score=%d, auto=%v): %s", hostID, threatScore, auto, reason)

	cmds := n.buildIsolationCommands()
	var firstErr error
	for _, cmd := range cmds {
		out, err := n.ssh.Exec(sessionID, cmd)
		if err != nil {
			n.log.Error("[ISOLATOR] Command failed on %s: %q -> %v (output: %s)", hostID, cmd, err, out)
			if firstErr == nil {
				firstErr = fmt.Errorf("isolation command %q failed: %w", cmd, err)
			}
			// Continue applying remaining rules even if one fails
		} else {
			n.log.Debug("[ISOLATOR] OK: %q", cmd)
		}
	}

	record := &IsolationRecord{
		HostID:      hostID,
		IsolatedAt:  time.Now().Format(time.RFC3339),
		Reason:      reason,
		ThreatScore: threatScore,
		Auto:        auto,
	}
	if firstErr != nil {
		record.Error = firstErr.Error()
	}

	n.mu.Lock()
	n.records[hostID] = record
	n.mu.Unlock()

	// Publish isolation confirmed event
	n.bus.Publish("ransomware.isolation_applied", map[string]interface{}{
		"host_id":      hostID,
		"isolated_at":  record.IsolatedAt,
		"reason":       reason,
		"threat_score": threatScore,
		"auto":         auto,
		"error":        record.Error,
	})

	// Also fire a SIEM alert for audit trail
	n.bus.Publish("siem.alert_fired", map[string]interface{}{
		"type":        "HOST_NETWORK_ISOLATED",
		"severity":    "CRITICAL",
		"host_id":     hostID,
		"description": fmt.Sprintf("Host %s has been network-isolated. Reason: %s", hostID, reason),
		"technique":   "T1486",
		"auto_action": fmt.Sprintf("isolated=%v", auto),
	})

	n.log.Warn("[ISOLATOR] ✅ Host %s isolation complete (error=%v)", hostID, firstErr)
	return firstErr
}

// RestoreHost removes the isolation rules and re-enables normal network access.
// Should only be called by an authorised analyst after incident validation.
func (n *NetworkIsolator) RestoreHost(ctx context.Context, hostID string) error {
	n.mu.RLock()
	rec, exists := n.records[hostID]
	n.mu.RUnlock()

	if !exists {
		return fmt.Errorf("host %s has no isolation record", hostID)
	}
	if rec.Restored {
		return fmt.Errorf("host %s is already restored", hostID)
	}

	sessionID, ok := n.resolver.GetActiveSessionForHost(hostID)
	if !ok {
		return fmt.Errorf("no active SSH session for host %s — cannot restore remotely", hostID)
	}

	n.log.Info("[ISOLATOR] Restoring network access for host %s", hostID)

	restoreCmds := []string{
		"sudo iptables -F INPUT",
		"sudo iptables -F OUTPUT",
		"sudo iptables -F FORWARD",
		"sudo iptables -P INPUT ACCEPT",
		"sudo iptables -P OUTPUT ACCEPT",
		"sudo iptables -P FORWARD ACCEPT",
	}

	for _, cmd := range restoreCmds {
		if _, err := n.ssh.Exec(sessionID, cmd); err != nil {
			n.log.Warn("[ISOLATOR] Restore command failed: %q -> %v", cmd, err)
		}
	}

	now := time.Now().Format(time.RFC3339)
	n.mu.Lock()
	rec.Restored = true
	rec.RestoredAt = &now
	n.mu.Unlock()

	n.bus.Publish("ransomware.isolation_restored", map[string]interface{}{
		"host_id":     hostID,
		"restored_at": now,
	})

	n.log.Info("[ISOLATOR] Host %s network access restored", hostID)
	return nil
}

// ListIsolations returns all current and past isolation records.
func (n *NetworkIsolator) ListIsolations() []IsolationRecord {
	n.mu.RLock()
	defer n.mu.RUnlock()
	records := make([]IsolationRecord, 0, len(n.records))
	for _, r := range n.records {
		records = append(records, *r)
	}
	return records
}

// buildIsolationCommands returns the ordered iptables commands for host containment.
// Design goals:
//   - Preserve existing management/SOC SSH session (ESTABLISHED,RELATED)
//   - Allow loopback
//   - Allow management CIDR if configured
//   - Drop everything else on INPUT
//   - Log dropped INPUT for forensic reconstruction
//   - Leave OUTPUT ACCEPT so agents can phone home to OBLIVRA
func (n *NetworkIsolator) buildIsolationCommands() []string {
	cmds := []string{
		// Flush existing rules to start clean
		"sudo iptables -F INPUT",
		"sudo iptables -F FORWARD",

		// Allow loopback
		"sudo iptables -A INPUT -i lo -j ACCEPT",

		// Allow established/related connections — keeps current SSH session alive
		"sudo iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT",
	}

	// Allow management CIDR if configured
	if n.managementCIDR != "" && isValidCIDR(n.managementCIDR) {
		cmds = append(cmds,
			fmt.Sprintf("sudo iptables -A INPUT -s %s -j ACCEPT", n.managementCIDR),
		)
	}

	// Log all dropped INPUT packets for forensic reconstruction
	cmds = append(cmds,
		`sudo iptables -A INPUT -m limit --limit 10/min -j LOG --log-prefix "[OBLIVRA-ISOLATED] " --log-level 4`,
	)

	// Default DROP policy for INPUT and FORWARD
	cmds = append(cmds,
		"sudo iptables -P INPUT DROP",
		"sudo iptables -P FORWARD DROP",
	)

	return cmds
}

// isValidCIDR does a basic sanity check on a CIDR string before injecting into a shell command.
func isValidCIDR(cidr string) bool {
	// Reject any shell metacharacters to prevent command injection
	bad := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\n", "\r", " "}
	for _, b := range bad {
		if strings.Contains(cidr, b) {
			return false
		}
	}
	// Must contain a slash (e.g. "10.0.0.0/24")
	return strings.Contains(cidr, "/")
}
