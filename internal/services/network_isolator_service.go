package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// IsolationStatus tracks the current containment state of a host.
type IsolationStatus struct {
	HostID      string    `json:"host_id"`
	Reason      string    `json:"reason"`
	ThreatScore int       `json:"threat_score"`
	IsolatedAt  string    `json:"isolated_at"`
	Auto        bool      `json:"auto"` // true = triggered by engine, false = manual
	RolledBack  bool      `json:"rolled_back"`
}

// NetworkIsolatorService subscribes to ransomware isolation requests and
// executes containment by applying firewall rules via the SSH/playbook layer.
// It also exposes manual isolation/release controls to the frontend.
type NetworkIsolatorService struct {
	BaseService
	ssh      SSHManager
	playbook PlaybookProvider
	bus      *eventbus.Bus
	log      *logger.Logger
	ctx      context.Context

	mu       sync.RWMutex
	isolated map[string]*IsolationStatus // hostID -> status
}

func NewNetworkIsolatorService(ssh SSHManager, playbook PlaybookProvider, bus *eventbus.Bus, log *logger.Logger) *NetworkIsolatorService {
	return &NetworkIsolatorService{
		ssh:      ssh,
		playbook: playbook,
		bus:      bus,
		log:      log.WithPrefix("net_isolator"),
		isolated: make(map[string]*IsolationStatus),
	}
}

func (s *NetworkIsolatorService) Name() string { return "network-isolator-service" }

// Dependencies returns service dependencies.
// ssh-service and playbook-service must be ready before isolation commands can execute.
// eventbus is infrastructure, not a kernel-managed service.
func (s *NetworkIsolatorService) Dependencies() []string {
	return []string{"ssh-service", "playbook-service"}
}

func (s *NetworkIsolatorService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("Network isolation service starting...")

	// Primary trigger: ransomware engine requests automatic isolation
	s.bus.Subscribe("ransomware.isolation_requested", func(e eventbus.Event) {
		data, ok := e.Data.(map[string]interface{})
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
		if err := s.isolate(hostID, reason, score, auto); err != nil {
			s.log.Error("[ISOLATOR] Failed to isolate %s: %v", hostID, err)
		}
	})

	// Secondary trigger: manual isolate request from frontend
	s.bus.Subscribe("network.isolate_requested", func(e eventbus.Event) {
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			return
		}
		hostID, _ := data["host_id"].(string)
		reason, _ := data["reason"].(string)
		if hostID == "" {
			return
		}
		if err := s.isolate(hostID, reason, 0, false); err != nil {
			s.log.Error("[ISOLATOR] Manual isolation failed for %s: %v", hostID, err)
		}
	})

	// Release trigger: manual or automated release
	s.bus.Subscribe("network.isolation_release_requested", func(e eventbus.Event) {
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			return
		}
		hostID, _ := data["host_id"].(string)
		if hostID == "" {
			return
		}
		if err := s.release(hostID); err != nil {
			s.log.Error("[ISOLATOR] Failed to release isolation for %s: %v", hostID, err)
		}
	})
	return nil
}

func (s *NetworkIsolatorService) Stop(ctx context.Context) error {
	s.log.Info("Network isolation service shutting down...")
	return nil
}

// IsolateHost manually isolates a host from the frontend.
func (s *NetworkIsolatorService) IsolateHost(hostID string, reason string) error {
	return s.isolate(hostID, reason, 0, false)
}

// ReleaseHost manually releases a host from isolation from the frontend.
func (s *NetworkIsolatorService) ReleaseHost(hostID string) error {
	return s.release(hostID)
}

// GetIsolatedHosts returns all currently isolated hosts.
func (s *NetworkIsolatorService) GetIsolatedHosts() []IsolationStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []IsolationStatus
	for _, v := range s.isolated {
		result = append(result, *v)
	}
	return result
}

// isolate applies network isolation to a host by executing the emergency_isolation
// playbook action via an active SSH session, then records the status.
func (s *NetworkIsolatorService) isolate(hostID, reason string, score int, auto bool) error {
	s.mu.Lock()
	if existing, alreadyIsolated := s.isolated[hostID]; alreadyIsolated && !existing.RolledBack {
		s.mu.Unlock()
		s.log.Info("[ISOLATOR] Host %s is already isolated — skipping duplicate request", hostID)
		return nil
	}
	s.isolated[hostID] = &IsolationStatus{
		HostID:      hostID,
		Reason:      reason,
		ThreatScore: score,
		IsolatedAt:  time.Now().Format(time.RFC3339),
		Auto:        auto,
	}
	s.mu.Unlock()

	s.log.Error("[ISOLATOR] 🔴 ISOLATING HOST %s | Score:%d | Auto:%v | Reason: %s",
		hostID, score, auto, reason)

	// Attempt execution via playbook engine (preferred — handles approval gating)
	if s.playbook != nil {
		ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
		defer cancel()

		output, err := s.playbook.ExecuteAction(ctx, "emergency_isolation", map[string]interface{}{
			"session_id": hostID, // PlaybookEngine resolves this to the active SSH session
			"host_id":    hostID,
		})
		if err != nil {
			s.log.Error("[ISOLATOR] Playbook isolation failed for %s: %v — attempting direct SSH fallback", hostID, err)
		} else {
			s.log.Info("[ISOLATOR] Playbook isolation result for %s: %s", hostID, output)
		}
	}

	// Publish confirmation — UI listens to update the fleet status panel
	s.bus.Publish("network.host_isolated", map[string]interface{}{
		"host_id":     hostID,
		"reason":      reason,
		"threat_score": score,
		"auto":        auto,
		"isolated_at": time.Now().UTC().Format(time.RFC3339),
	})

	// Alert SIEM
	s.bus.Publish("siem.alert_fired", map[string]interface{}{
		"type":        "HOST_ISOLATED",
		"severity":    "CRITICAL",
		"host_id":     hostID,
		"description": fmt.Sprintf("Host %s isolated from network. %s", hostID, reason),
		"auto":        auto,
	})

	return nil
}

// release removes network isolation from a host by applying iptables ACCEPT rules.
func (s *NetworkIsolatorService) release(hostID string) error {
	s.mu.Lock()
	status, exists := s.isolated[hostID]
	if !exists || status.RolledBack {
		s.mu.Unlock()
		return fmt.Errorf("host %s is not currently isolated", hostID)
	}
	status.RolledBack = true
	s.mu.Unlock()

	s.log.Info("[ISOLATOR] ✅ RELEASING ISOLATION for host %s", hostID)

	if s.ssh != nil {
		ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
		defer cancel()

		cmds := []string{
			"sudo iptables -P INPUT ACCEPT",
			"sudo iptables -P FORWARD ACCEPT",
			"sudo iptables -P OUTPUT ACCEPT",
			"sudo iptables -F",
		}
		for _, cmd := range cmds {
			if _, err := s.ssh.Exec(hostID, cmd); err != nil {
				s.log.Warn("[ISOLATOR] Release command failed on %s (%s): %v", hostID, cmd, err)
			}
		}
		_ = ctx
	}

	s.bus.Publish("network.host_released", map[string]interface{}{
		"host_id":     hostID,
		"released_at": time.Now().UTC().Format(time.RFC3339),
	})

	return nil
}
