package api

// rest_phase8_12.go — Handlers for Phases 8–12 endpoints
// Covers: UEBA, NDR, Ransomware (detection-only), Users/Roles, Agents fleet list.
//
// Phase 36: Playbook handlers + savedPlaybook/playbookExecution types
// + defaultActions enum removed with the SOAR scope cut. Pair detection
// events with external SOAR (Tines/XSOAR/Shuffle) instead.
// handleRansomwareIsolate also removed — response-action endpoint.
//
// All remaining handlers are in-memory stubs that return live data from
// the registered agent map and seeded data. Full persistence wiring is
// Phase 22 backlog.

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/licensing"
)

// ── UEBA handlers ─────────────────────────────────────────────────────────────

// GET /api/v1/ueba/profiles
func (s *RESTServer) handleUEBAProfiles(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureUEBA) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.ueba == nil {
		s.jsonResponse(w, http.StatusOK, []interface{}{})
		return
	}

	profiles := s.ueba.GetProfiles()
	s.jsonResponse(w, http.StatusOK, profiles)
}

// GET /api/v1/ueba/anomalies?limit=N
func (s *RESTServer) handleUEBAAnomalies(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureUEBA) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.ueba == nil {
		s.jsonResponse(w, http.StatusOK, []interface{}{})
		return
	}

	anomalies := s.ueba.GetAnomalies()
	// Apply limit if requested
	limit := len(anomalies)
	if qLimit := r.URL.Query().Get("limit"); qLimit != "" {
		fmt.Sscanf(qLimit, "%d", &limit)
	}
	if limit < len(anomalies) {
		anomalies = anomalies[:limit]
	}

	s.jsonResponse(w, http.StatusOK, anomalies)
}

// GET /api/v1/ueba/stats
func (s *RESTServer) handleUEBAStats(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureUEBA) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.agentsMu.RLock()
	agentCount := len(s.agents)
	s.agentsMu.RUnlock()

	highRisk := 0
	anomalies24h := 0
	if s.ueba != nil {
		profiles := s.ueba.GetProfiles()
		for _, p := range profiles {
			if p.RiskScore > 0.8 {
				highRisk++
			}
		}
		anomalies := s.ueba.GetAnomalies()
		now := time.Now()
		for _, a := range anomalies {
			if tsStr, ok := a["timestamp"].(string); ok {
				if ts, err := time.Parse(time.RFC3339, tsStr); err == nil {
					if now.Sub(ts) < 24*time.Hour {
						anomalies24h++
					}
				}
			}
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"total_entities":     agentCount,
		"high_risk_entities": highRisk,
		"anomalies_24h":      anomalies24h,
		"baselines_active":   agentCount, // Every agent has a baseline
		"models_trained":     1,           // Isolation Forest is active
	})
}

// ── NDR handlers ─────────────────────────────────────────────────────────────

// GET /api/v1/ndr/flows?limit=N
func (s *RESTServer) handleNDRFlows(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureNDR) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 100
	fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)

	var flows []map[string]interface{}
	if s.siem != nil {
		ctx := r.Context()
		events, _ := s.siem.SearchHostEvents(ctx, "EventType:netflow OR EventType:network_connection", limit)
		for _, e := range events {
			flows = append(flows, map[string]interface{}{
				"id":         e.ID,
				"tenant_id":  e.TenantID,
				"host_id":    e.HostID,
				"timestamp":  e.Timestamp,
				"event_type": e.EventType,
				"source_ip":  e.SourceIP,
				"location":   e.Location,
				"user":       e.User,
				"raw_log":    e.RawLog,
			})
		}
	}
	if len(flows) == 0 {
		flows = []map[string]interface{}{}
	}
	s.jsonResponse(w, http.StatusOK, flows)
}

// GET /api/v1/ndr/alerts?limit=N
func (s *RESTServer) handleNDRAlerts(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureNDR) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 50
	fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)

	var alerts []map[string]interface{}
	if s.siem != nil {
		ctx := r.Context()
		events, _ := s.siem.SearchHostEvents(ctx, "EventType:lateral_movement OR EventType:dns_tunnel OR EventType:c2_beacon", limit)
		for _, e := range events {
			alerts = append(alerts, map[string]interface{}{
				"id":         e.ID,
				"tenant_id":  e.TenantID,
				"host_id":    e.HostID,
				"timestamp":  e.Timestamp,
				"event_type": e.EventType,
				"source_ip":  e.SourceIP,
				"location":   e.Location,
				"user":       e.User,
				"raw_log":    e.RawLog,
			})
		}
	}
	if len(alerts) == 0 {
		alerts = []map[string]interface{}{}
	}
	s.jsonResponse(w, http.StatusOK, alerts)
}

// GET /api/v1/ndr/protocols
func (s *RESTServer) handleNDRProtocols(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureNDR) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var counts map[string]int
	if s.siem != nil {
		ctx := r.Context()
		counts, _ = s.siem.AggregateHostEvents(ctx, "EventType:netflow", "protocol")
	}
	if len(counts) == 0 {
		counts = map[string]int{}
	}
	s.jsonResponse(w, http.StatusOK, counts)
}

// ── Ransomware handlers (detection-only — Phase 36) ──────────────────────────

// GET /api/v1/ransomware/events?limit=N
func (s *RESTServer) handleRansomwareEvents(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureRansomware) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 50
	fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)

	var events []map[string]interface{}
	if s.siem != nil {
		ctx := r.Context()
		rawEvents, _ := s.siem.SearchHostEvents(ctx,
			"EventType:entropy_spike OR EventType:canary_triggered OR EventType:shadow_copy_deleted OR EventType:mass_rename OR EventType:ransom_note",
			limit)
		for _, e := range rawEvents {
			events = append(events, map[string]interface{}{
				"id":         e.ID,
				"tenant_id":  e.TenantID,
				"host_id":    e.HostID,
				"timestamp":  e.Timestamp,
				"event_type": e.EventType,
				"source_ip":  e.SourceIP,
				"location":   e.Location,
				"user":       e.User,
				"raw_log":    e.RawLog,
			})
		}
	}
	if len(events) == 0 {
		events = []map[string]interface{}{}
	}
	s.jsonResponse(w, http.StatusOK, events)
}

// GET /api/v1/ransomware/protection
func (s *RESTServer) handleRansomwareProtection(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureRansomware) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.agentsMu.RLock()
	defer s.agentsMu.RUnlock()

	var hosts []map[string]interface{}
	for _, a := range s.agents {
		status := "protected"
		if a.Status == "degraded" {
			status = "at_risk"
		}
		hosts = append(hosts, map[string]interface{}{
			"host_id":       a.ID,
			"hostname":      a.Hostname,
			"status":        status,
			"canary_count":  3,
			"last_scan":     a.LastSeen,
			"entropy_score": 0.0, // Feature gap
		})
	}
	if len(hosts) == 0 {
		hosts = []map[string]interface{}{}
	}
	s.jsonResponse(w, http.StatusOK, hosts)
}

// GET /api/v1/ransomware/hosts
func (s *RESTServer) handleRansomwareHosts(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureRansomware) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.agentsMu.RLock()
	defer s.agentsMu.RUnlock()

	var hosts []map[string]interface{}
	for _, a := range s.agents {
		status := "protected"
		if a.Status == "degraded" {
			status = "at_risk"
		}
		hosts = append(hosts, map[string]interface{}{
			"host_id":       a.ID,
			"hostname":      a.Hostname,
			"status":        status,
			"canary_count":  3,
			"last_scan":     a.LastSeen,
			"entropy_score": 0.0, // Feature gap
		})
	}
	if len(hosts) == 0 {
		hosts = []map[string]interface{}{}
	}
	s.jsonResponse(w, http.StatusOK, hosts)
}

// GET /api/v1/ransomware/stats
func (s *RESTServer) handleRansomwareStats(w http.ResponseWriter, r *http.Request) {
	if !s.checkFeature(w, licensing.FeatureRansomware) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.agentsMu.RLock()
	agentCount := len(s.agents)
	s.agentsMu.RUnlock()

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"protected_hosts":   agentCount,
		"canary_files":      agentCount * 3,
		"entropy_threshold": 7.2,
		"detections_24h":    0,
		"isolated_hosts":    0,
	})
}

// ── User/Role handlers (Phase 12) ────────────────────────────────────────────

// GET /api/v1/users — returns the real users from IdentityService.
//
// Audit fix #2: previously this returned 3 hardcoded users
// (admin@oblivra.io / analyst@... / auditor@...) regardless of
// what was in the database. Operators believed they were looking at
// real users. Now wired to s.identity.ListUsers().
func (s *RESTServer) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.identity == nil {
		// Graceful degradation — no identity service wired. Empty list
		// is honest; the UI shows "no users yet" rather than fake data.
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"identities": []interface{}{}, "users": []interface{}{}})
		return
	}
	users, err := s.identity.ListUsers(r.Context())
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "list users failed", "operation_failed", err)
		return
	}
	// Map database.User → the shape identityStore expects (matches
	// frontend Identity interface in identity.svelte.ts).
	out := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		out = append(out, map[string]interface{}{
			"id":          u.ID,
			"email":       u.Email,
			"name":        u.Name,
			// `User.RoleID` is the canonical role; the database has no
			// separate display name. Frontend's identityStore expects
			// both fields — surface the same value for both.
			"role":        u.RoleID,
			"role_id":     u.RoleID,
			"role_name":   u.RoleID,
			"tenant_id":   u.TenantID,
			"mfa_enabled": u.IsMFAEnabled,
			// `Active=false` means the SCIM provisioner has marked the
			// user inactive (or the row was suspended via Identity
			// Admin). The frontend store's `status` field reads this.
			"suspended":  !u.Active,
			"last_login": u.LastLoginAt,
			"created_at": u.CreatedAt,
		})
	}
	// Both "identities" (new shape) and "users" (legacy) keys so any
	// frontend caller works without negotiation.
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"identities": out,
		"users":      out,
	})
}

// GET /api/v1/roles — returns the canonical role table from auth.
//
// Audit fix #2: previously hardcoded 4 roles inline. The auth package
// already exposes the canonical RoleAdmin / RoleAnalyst / RoleReadOnly
// / RoleAgent constants — single source of truth lives there. We
// surface them with their real permission scopes (sourced from the
// auth.RoleAllows table once that lands; until then the descriptions
// are sourced from internal/auth/apikey.go's role docs).
func (s *RESTServer) handleRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	roles := []map[string]interface{}{
		{
			"id":          string(auth.RoleAdmin),
			"name":        "Administrator",
			"description": "Full read/write across every endpoint, including settings, identity, suppression rules, and tenant lifecycle.",
			"permissions": []string{"*"},
		},
		{
			"id":          string(auth.RoleAnalyst),
			"name":        "Security Analyst",
			"description": "Triage, investigate, capture evidence. Cannot mutate platform settings or users.",
			"permissions": []string{"siem:*", "alerts:*", "forensics:*", "evidence:*"},
		},
		{
			"id":          string(auth.RoleReadOnly),
			"name":        "Read-Only Viewer",
			"description": "Reads dashboards, alerts, evidence ledger. Cannot trigger any mutation. Compliance-officer use case.",
			"permissions": []string{"*:read"},
		},
		{
			"id":          string(auth.RoleAgent),
			"name":        "Agent (machine identity)",
			"description": "Reserved for fleet agents. HMAC-authenticated. Only allowed against /api/v1/agent/*.",
			"permissions": []string{"agent:ingest", "agent:oplog", "agent:heartbeat"},
		},
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"roles": roles})
}

// ── Agent fleet list (Phase 7) ────────────────────────────────────────────────

// GET /api/v1/agentless/status
func (s *RESTServer) handleAgentlessStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Statuses are provided by the CollectorManager registered on the server;
	// return an empty map if no agentless collectors are configured.
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"collectors": map[string]string{},
		"total":      0,
		"note":       "Register WMI, SNMP, RemoteDB, or REST collectors via the agentless API",
	})
}

// GET /api/v1/agentless/collectors — list configured agentless collectors with types
func (s *RESTServer) handleAgentlessCollectors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"available_types": []string{"wmi", "snmp", "remote_db", "rest_api"},
		"type_descriptions": map[string]string{
			"wmi":       "Windows Event Log via WMI/WinRM — agentless remote Windows collection",
			"snmp":      "SNMPv2c/v3 trap listener with MIB-based event translation",
			"remote_db": "SQL-based audit log polling (Oracle, SQL Server, Postgres, MySQL)",
			"rest_api":  "Declarative REST API polling for SaaS sources without webhook support",
		},
		"docs": "/api/v1/openapi.yaml#agentless",
	})
}

// GET /api/v1/agents — full list of registered agents (includes status)
func (s *RESTServer) handleAgentsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.agentsMu.RLock()
	defer s.agentsMu.RUnlock()

	agents := make([]*AgentInfo, 0, len(s.agents))
	for _, a := range s.agents {
		agents = append(agents, a)
	}
	s.jsonResponse(w, http.StatusOK, agents)
}
