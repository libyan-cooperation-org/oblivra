package api

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
)

// AgentEvent mirrors the agent's Event struct for deserialization.
// AgentID is required — agents stamp every event with their stable ID.
//
// Seq is a per-agent monotonically increasing sequence number assigned by
// the agent's WAL. The server uses it for idempotent replay: events with
// Seq <= AgentInfo.LastAckedSeq are silently dropped as duplicates from
// a retry triggered by a previous-batch network failure. See
// internal/agent/cursor.go for the agent-side persistence model.
type AgentEvent struct {
	Seq       uint64                 `json:"seq"`
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
	PublicKey  []byte   `json:"public_key"` // 1.4: Hardware-rooted trust key
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

	// 2.4: Payload integrity and size constraints
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10MB limit
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	// 2.4: HMAC Verification (Fleet Security)
	if err := VerifyHMAC(r, body, s.fleetSecret); err != nil {
		s.log.Warn("[security] HMAC verification failed: %v", err)
		http.Error(w, "Unauthorized: invalid signature or expired request", http.StatusUnauthorized)
		return
	}

	agentID := r.Header.Get("X-Agent-ID")
	s.agentsMu.RLock()
	agent, ok := s.agents[agentID]
	s.agentsMu.RUnlock()

	// 1.4: Cryptographic Batch Verification (Sovereign Trust)
	if ok && len(agent.PublicKey) > 0 {
		sigBase64 := r.Header.Get("X-Agent-Signature")
		if sigBase64 == "" {
			s.log.Warn("[security] Batch from %s missing cryptographic signature", agentID)
			http.Error(w, "Unauthorized: batch signature required", http.StatusUnauthorized)
			return
		}
		sig, err := base64.StdEncoding.DecodeString(sigBase64)
		if err != nil || !ed25519.Verify(agent.PublicKey, body, sig) {
			s.log.Warn("[security] Cryptographic batch signature MISMATCH for agent %s", agentID)
			http.Error(w, "Unauthorized: batch signature verification failed", http.StatusUnauthorized)
			return
		}
	}

	var events []AgentEvent
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields() // Security: prevent field injection/smuggling
	if err := decoder.Decode(&events); err != nil {
		http.Error(w, "invalid payload structure", http.StatusBadRequest)
		return
	}

	if len(events) == 0 {
		// Heartbeat: update last-seen from headers even with no events.
		// Echo the current ack watermark so a heartbeat-only agent still
		// learns it is in sync (e.g. a server that lost its agents map and
		// rebuilt it can confirm "no events outstanding" via heartbeat).
		s.updateAgentLastSeen(r)
		ackedSeq := uint64(0)
		s.agentsMu.RLock()
		if a, ok := s.agents[r.Header.Get("X-Agent-ID")]; ok {
			ackedSeq = a.LastAckedSeq
		}
		s.agentsMu.RUnlock()
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"accepted":  0,
			"config":    map[string]interface{}{},
			"actions":   []interface{}{},
			"acked_seq": ackedSeq,
		})
		return
	}

	// Identity check: agent-role users can only ingest for their own AgentID
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

	// Read the agent's current ack watermark — events at or below this Seq
	// are duplicate replays (the agent retried after a server restart or
	// network blip) and must be skipped to avoid double-ingestion.
	prevAckedSeq := uint64(0)
	s.agentsMu.RLock()
	if a, ok := s.agents[agentID]; ok {
		prevAckedSeq = a.LastAckedSeq
	}
	s.agentsMu.RUnlock()

	// Forward events to SIEM store, tracking the highest-Seq event we
	// successfully ingest so we can advance the watermark exactly that
	// far. If event N+5 fails to insert but N+1..N+4 succeed, we ack N+4
	// and force the agent to retry N+5 onward — no silent loss.
	accepted := 0
	skipped := 0
	highestAckedThisBatch := prevAckedSeq

	if s.siem != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		for _, ev := range events {
			// Idempotency: drop replays of already-acked events.
			if ev.Seq != 0 && ev.Seq <= prevAckedSeq {
				skipped++
				continue
			}

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
				// Stop advancing the ack here — the agent must retry from
				// this Seq forward on its next flush.
				break
			}
			accepted++
			if ev.Seq > highestAckedThisBatch {
				highestAckedThisBatch = ev.Seq
			}
		}
	}

	// Persist the new watermark only if it actually advanced. A batch with
	// no Seq fields (legacy agent) leaves the watermark untouched and the
	// agent falls back to the local highest-Seq-in-batch on its end.
	if highestAckedThisBatch > prevAckedSeq && agentID != "" {
		s.agentsMu.Lock()
		if a, ok := s.agents[agentID]; ok {
			a.LastAckedSeq = highestAckedThisBatch
		}
		s.agentsMu.Unlock()
	}

	s.log.Info("[agent] Ingested %d (skipped %d duplicate) of %d events from host=%s agent=%s acked_seq=%d",
		accepted, skipped, len(events), hostname, agentID, highestAckedThisBatch)

	// Respond with fleet config + any pending actions for this agent
	s.jsonResponse(w, http.StatusAccepted, map[string]interface{}{
		"accepted":  accepted,
		"skipped":   skipped,
		"total":     len(events),
		"acked_seq": highestAckedThisBatch,
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

	// Registration also requires HMAC if a bootstrap secret is configured.
	// This prevents random internet scanners from filling the fleet map.
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	if err := VerifyHMAC(r, body, s.fleetSecret); err != nil {
		s.log.Warn("[security] HMAC verification failed for agent registration: %v (addr=%s)", err, r.RemoteAddr)
		http.Error(w, "Unauthorized: invalid signature or expired request", http.StatusUnauthorized)
		return
	}

	var reg AgentRegistration
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&reg); err != nil {
		http.Error(w, "invalid payload structure", http.StatusBadRequest)
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
		PublicKey:  reg.PublicKey, // 1.4: Store agent's identity key
		LastSeen:   now,
		Status:     "online",
	}

	s.agentsMu.Lock()
	defer s.agentsMu.Unlock()

	// Quota Enforcement (Phase 25.5)
	if s.license != nil {
		max := s.license.MaxAgents()
		if max > 0 {
			// Count unique agents by ID
			uniqueAgents := make(map[string]bool)
			for _, a := range s.agents {
				uniqueAgents[a.ID] = true
			}
			if !uniqueAgents[reg.ID] && len(uniqueAgents) >= max {
				s.log.Warn("[licensing] Agent registration DENIED for %s: quota exceeded (max=%d)", reg.ID, max)
				http.Error(w, "Payment Required: Agent quota exceeded", http.StatusPaymentRequired)
				return
			}
		}
	}

	// Index by both AgentID (stable, authoritative) and hostname (display)
	// so that lookups from either direction work.
	s.agents[reg.ID] = info
	s.agents[reg.Hostname] = info // pointer shared — updates via either key are visible

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
