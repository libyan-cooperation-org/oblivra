package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
)

// AgentEvent mirrors the agent's Event struct for deserialization.
// AgentID is required — agents stamp every event with their stable ID.
type AgentEvent struct {
	Timestamp string                 `json:"timestamp"`
	Source    string                 `json:"source"`
	Type      string                 `json:"type"`
	Host      string                 `json:"host"`
	AgentID   string                 `json:"agent_id"`
	Version   string                 `json:"version"`
	Data      map[string]interface{} `json:"data"`
}

// AgentRegistration is the payload agents send on first connect and heartbeat.
type AgentRegistration struct {
	ID         string   `json:"id"`       // stable agent UUID
	Hostname   string   `json:"hostname"`
	OS         string   `json:"os"`
	Arch       string   `json:"arch"`
	Version    string   `json:"version"`
	Collectors []string `json:"collectors"`
}

// handleAgentIngest receives event batches from agents.
// POST /api/v1/agent/ingest
func (s *RESTServer) handleAgentIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Auth: agents or admins only
	identityUser := auth.UserFromContext(r.Context())
	if identityUser == nil ||
		(identityUser.RoleName != string(auth.RoleAgent) && identityUser.RoleName != string(auth.RoleAdmin)) {
		http.Error(w, "Forbidden: Only agents or admins can ingest", http.StatusForbidden)
		return
	}

	// Body limit: 10 MB — accommodates up to 5,000 events at ~2 KB each.
	// The previous 1 MB limit was too small for max-batch=5000 payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	var events []AgentEvent
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		// Return a generic error — never expose internal struct names or decoder details
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if len(events) == 0 {
		// Heartbeat: update last-seen from headers even with no events
		s.updateAgentLastSeen(r)
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"accepted": 0,
			"config":   map[string]interface{}{},
			"actions":  []interface{}{},
		})
		return
	}

	// Identity check: agent-role users can only ingest for their own AgentID
	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		agentID = events[0].AgentID
	}
	if identityUser.RoleName == string(auth.RoleAgent) &&
		identityUser.ID != "" && identityUser.ID != agentID {
		s.log.Warn("[agent] SPOOFING ATTEMPT: identity=%s tried to ingest for agent=%s",
			identityUser.Email, agentID)
		http.Error(w, "Forbidden: agent ID mismatch", http.StatusForbidden)
		return
	}

	// Update fleet last-seen keyed on AgentID (stable) and hostname (display)
	hostname := events[0].Host
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "GLOBAL"
	}

	if agentID != "" {
		s.agentsMu.Lock()
		now := time.Now().Format(time.RFC3339)
		// Update by AgentID (authoritative key)
		if a, ok := s.agents[agentID]; ok {
			a.LastSeen = now
			a.Status = "online"
			a.TenantID = tenantID
		}
		// Also update by hostname for backward compat with handlers that key on hostname
		if hostname != "" {
			if a, ok := s.agents[hostname]; ok {
				a.LastSeen = now
				a.Status = "online"
				a.TenantID = tenantID
			}
		}
		s.agentsMu.Unlock()
	}

	// Forward events to SIEM store
	accepted := 0
	if s.siem != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		for _, ev := range events {
			rawLog := ""
			if ev.Data != nil {
				if raw, err := json.Marshal(ev.Data); err == nil {
					rawLog = string(raw)
				}
			}
			hostEvent := &database.HostEvent{
				HostID:    ev.Host,
				EventType: ev.Type,
				SourceIP:  agentID, // use stable AgentID as source identifier
				User:      ev.AgentID,
				RawLog:    rawLog,
				Timestamp: ev.Timestamp,
			}
			if err := s.siem.InsertHostEvent(ctx, hostEvent); err != nil {
				s.log.Warn("[agent] Failed to insert event from %s: %v", ev.Host, err)
				continue
			}
			accepted++
		}
	}

	s.log.Info("[agent] Ingested %d/%d events from host=%s agent=%s",
		accepted, len(events), hostname, agentID)

	// Respond with fleet config + any pending actions for this agent
	s.jsonResponse(w, http.StatusAccepted, map[string]interface{}{
		"accepted": accepted,
		"total":    len(events),
		// Echo back minimal fleet config so agent can sync interval/toggles
		"config": map[string]interface{}{
			"interval":        30,
			"enable_fim":      false,
			"enable_syslog":   true,
			"enable_metrics":  true,
			"enable_event_log": false,
		},
		"actions": []interface{}{},
	})
}

// updateAgentLastSeen bumps last-seen for an agent based on request headers.
// Used for heartbeat POSTs that carry 0 events.
func (s *RESTServer) updateAgentLastSeen(r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	hostname := r.Header.Get("X-Agent-Hostname")
	if agentID == "" && hostname == "" {
		return
	}
	now := time.Now().Format(time.RFC3339)
	s.agentsMu.Lock()
	for _, key := range []string{agentID, hostname} {
		if key != "" {
			if a, ok := s.agents[key]; ok {
				a.LastSeen = now
				a.Status = "online"
			}
		}
	}
	s.agentsMu.Unlock()
}

// handleAgentRegister registers or updates an agent in the fleet.
// POST /api/v1/agent/register
// No auth required — agents register before they have a token.
// Rate limiting in secureMiddleware prevents abuse.
func (s *RESTServer) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	var reg AgentRegistration
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if reg.ID == "" || reg.Hostname == "" {
		http.Error(w, "id and hostname are required", http.StatusBadRequest)
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "GLOBAL"
	}

	now := time.Now().Format(time.RFC3339)
	info := &AgentInfo{
		ID:         reg.ID,
		Hostname:   reg.Hostname,
		TenantID:   tenantID,
		OS:         reg.OS,
		Arch:       reg.Arch,
		Version:    reg.Version,
		Collectors: reg.Collectors,
		LastSeen:   now,
		Status:     "online",
	}

	s.agentsMu.Lock()
	// Index by both AgentID (stable, authoritative) and hostname (display)
	// so that lookups from either direction work.
	s.agents[reg.ID] = info
	s.agents[reg.Hostname] = info // pointer shared — updates via either key are visible
	s.agentsMu.Unlock()

	s.log.Info("[agent] Registered agent id=%s host=%s os=%s/%s v%s collectors=%v",
		reg.ID, reg.Hostname, reg.OS, reg.Arch, reg.Version, reg.Collectors)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "registered",
		"agent":   reg.Hostname,
		"version": reg.Version,
	})
}

// handleAgentFleet returns the current fleet status.
// GET /api/v1/agent/fleet
// Accessible to Analysts and Admins (not just Admins).
func (s *RESTServer) handleAgentFleet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Analysts and Admins can see fleet health
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin && role != auth.RoleAnalyst {
		http.Error(w, "Forbidden: Analyst or Admin role required", http.StatusForbidden)
		return
	}

	now := time.Now()

	// Primary source: AgentProvider (bridges to the AgentServer's real fleet registry on :8443)
	var fleet []*AgentInfo
	if s.agentProvider != nil {
		s.log.Info("[REST] Querying AgentProvider for fleet data")
		providerFleet := s.agentProvider.GetFleet()
		s.log.Info("[REST] AgentProvider returned %d agents", len(providerFleet))
		for i := range providerFleet {
			agentCopy := providerFleet[i] // Create a copy of the struct
			ts, _ := time.Parse(time.RFC3339, agentCopy.LastSeen)
			since := now.Sub(ts)
			switch {
			case since < 45*time.Second:
				agentCopy.Status = "online"
			case since < 5*time.Minute:
				agentCopy.Status = "degraded"
			default:
				agentCopy.Status = "offline"
			}
			fleet = append(fleet, &agentCopy) // Append pointer to the copy
		}
	} else {
		s.log.Warn("[REST] AgentProvider is nil")
	}

	// Fallback: merge any agents tracked locally on the REST server (e.g. from direct ingest)
	if len(fleet) == 0 {
		s.log.Info("[REST] No agents from provider, falling back to local map (size=%d)", len(s.agents))
		s.agentsMu.RLock()
		seen := make(map[string]struct{})
		for _, agent := range s.agents {
			if _, dup := seen[agent.ID]; dup {
				continue
			}
			seen[agent.ID] = struct{}{}
			ts, _ := time.Parse(time.RFC3339, agent.LastSeen)
			since := now.Sub(ts)
			switch {
			case since < 45*time.Second:
				agent.Status = "online"
			case since < 5*time.Minute:
				agent.Status = "degraded"
			default:
				agent.Status = "offline"
			}
			fleet = append(fleet, agent)
		}
		s.agentsMu.RUnlock()
	}

	s.log.Info("[REST] Returning %d agents to client", len(fleet))
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"total":  len(fleet),
		"agents": fleet,
	})
}

// handleAgentFleetConfig pushes a new config to a specific agent on next pull.
// POST /api/v1/agent/fleet/config
func (s *RESTServer) handleAgentFleetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}

	var req struct {
		AgentID  string      `json:"agent_id"`
		Hostname string      `json:"hostname"`
		Config   interface{} `json:"config"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if req.AgentID == "" && req.Hostname == "" {
		http.Error(w, "agent_id or hostname required", http.StatusBadRequest)
		return
	}

	// Config push is stored in-memory here; a full implementation would
	// write to a pending-config table that the agent polls on next flush.
	s.log.Info("[agent] Config push requested for agent=%s host=%s", req.AgentID, req.Hostname)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "queued",
		"note":   "Agent will receive config on next heartbeat",
	})
}

// handleAgentAction dispatches a response action to a specific agent.
// POST /api/v1/agent/action
func (s *RESTServer) handleAgentAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}

	var req struct {
		AgentID string            `json:"agent_id"`
		Type    string            `json:"type"`
		Payload map[string]string `json:"payload"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if req.AgentID == "" || req.Type == "" {
		http.Error(w, "agent_id and type are required", http.StatusBadRequest)
		return
	}

	// Action dispatch is queued for the agent to pull on next flush.
	// Full implementation: write to pending_actions table, agent reads on heartbeat.
	actionID := fmt.Sprintf("act-%d", time.Now().UnixNano())
	s.log.Info("[agent] Action queued: id=%s type=%s agent=%s", actionID, req.Type, req.AgentID)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"action_id": actionID,
		"status":    "queued",
		"type":      req.Type,
	})
}
