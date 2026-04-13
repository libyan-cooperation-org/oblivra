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
type AgentEvent struct {
	Timestamp string                 `json:"timestamp"`
	Source    string                 `json:"source"`
	Type      string                 `json:"type"`
	Host      string                 `json:"host"`
	Data      map[string]interface{} `json:"data"`
}

// AgentRegistration is the payload agents send to register.
type AgentRegistration struct {
	ID         string   `json:"id"`
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

	// 1. Authenticate and Authorize
	identityUser := auth.UserFromContext(r.Context())
	if identityUser == nil || (identityUser.RoleName != string(auth.RoleAgent) && identityUser.RoleName != string(auth.RoleAdmin)) {
		http.Error(w, "Forbidden: Only agents or admins can ingest", http.StatusForbidden)
		return
	}

	// Enforce maximum body size of 1MB to prevent JSON decoding OOM DoS
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	var events []AgentEvent
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		http.Error(w, fmt.Sprintf("Invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	if len(events) == 0 {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"accepted": 0})
		return
	}

	// 2. Host Ownership Verification
	// If the user is an agent, it can ONLY ingest for its designated host (or AgentID)
	providedHost := events[0].Host
	// We check against the AgentID field if present. In the current stub mapping, ID is used.
	if identityUser.RoleName == string(auth.RoleAgent) && identityUser.ID != "" && identityUser.ID != providedHost {
		s.log.Warn("[Agent Security] SPOOFING ATTEMPT: %s tried to ingest for %s", identityUser.Email, providedHost)
		http.Error(w, "Forbidden: Host mismatch", http.StatusForbidden)
		return
	}

	// Update agent last-seen from the host field
	if providedHost != "" {
		s.agentsMu.Lock()
		if agent, ok := s.agents[providedHost]; ok {
			agent.LastSeen = time.Now().Format(time.RFC3339)
			agent.Status = "online"
		}
		s.agentsMu.Unlock()
	}

	// Forward events to SIEM store
	accepted := 0
	if s.siem != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
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
				SourceIP:  ev.Source,
				RawLog:    rawLog,
				Timestamp: ev.Timestamp,
			}
			if err := s.siem.InsertHostEvent(ctx, hostEvent); err != nil {
				s.log.Warn("[Agent] Failed to insert event from %s: %v", ev.Host, err)
				continue
			}
			accepted++
		}
	}

	s.log.Info("[Agent] Ingested %d/%d events from %s", accepted, len(events), events[0].Host)
	s.jsonResponse(w, http.StatusAccepted, map[string]interface{}{
		"accepted": accepted,
		"total":    len(events),
	})
}

// handleAgentRegister registers or updates an agent in the fleet.
// POST /api/v1/agent/register
func (s *RESTServer) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enforce maximum body size of 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	var reg AgentRegistration
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	if reg.ID == "" || reg.Hostname == "" {
		http.Error(w, "id and hostname are required", http.StatusBadRequest)
		return
	}

	s.agentsMu.Lock()
	s.agents[reg.Hostname] = &AgentInfo{
		ID:         reg.ID,
		Hostname:   reg.Hostname,
		OS:         reg.OS,
		Arch:       reg.Arch,
		Version:    reg.Version,
		Collectors: reg.Collectors,
		LastSeen:   time.Now().Format(time.RFC3339),
		Status:     "online",
	}
	s.agentsMu.Unlock()

	s.log.Info("[Agent] Registered agent %s (%s/%s) v%s with collectors: %v",
		reg.Hostname, reg.OS, reg.Arch, reg.Version, reg.Collectors)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "registered",
		"agent":   reg.Hostname,
		"version": reg.Version,
	})
}

// handleAgentFleet returns the current fleet status.
// GET /api/v1/agent/fleet
func (s *RESTServer) handleAgentFleet(w http.ResponseWriter, r *http.Request) {
	// 1. RBAC Check: Require Admin
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admins only", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Priority: Use the agent provider (which bridges to the Ingest/AgentServer)
	if s.agentProvider != nil {
		fleet := s.agentProvider.GetFleet()
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"total":  len(fleet),
			"agents": fleet,
		})
		return
	}

	// Fallback/Legacy: Use locally registered agents
	s.agentsMu.Lock()
	now := time.Now()
	fleet := make([]*AgentInfo, 0, len(s.agents))
	for _, agent := range s.agents {
		ts, _ := time.Parse(time.RFC3339, agent.LastSeen)
		since := now.Sub(ts)
		switch {
		case since < 30*time.Second:
			agent.Status = "online"
		case since < 5*time.Minute:
			agent.Status = "degraded"
		default:
			agent.Status = "offline"
		}
		fleet = append(fleet, agent)
	}
	s.agentsMu.Unlock()

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"total":  len(fleet),
		"agents": fleet,
	})
}
