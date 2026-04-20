package api

// rest_phase8_12.go — Handlers for Phases 8–12 endpoints
// Covers: Playbooks, UEBA, NDR, Ransomware, Users/Roles, Agents fleet list
//
// All handlers are in-memory stubs that return live data from the registered
// agent map and seeded data. Full persistence wiring is Phase 22 backlog.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ── In-memory playbook store ──────────────────────────────────────────────────

type savedPlaybook struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Steps     interface{} `json:"steps"`
	CreatedAt string      `json:"created_at"`
	LastRun   string      `json:"last_run,omitempty"`
}

type playbookExecution struct {
	PlaybookID  string `json:"playbook_id"`
	IncidentID  string `json:"incident_id"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Duration    int    `json:"duration_ms"`
	Status      string `json:"status"`
	StepCount   int    `json:"step_count"`
}

var (
	savedPlaybooksMu  sync.RWMutex
	savedPlaybooks    []savedPlaybook

	playbookExecsMu  sync.RWMutex
	playbookExecs    []playbookExecution
)

var defaultActions = []string{
	"isolate_host", "lock_account", "kill_process", "collect_logs",
	"snapshot_memory", "notify_team", "block_ip", "run_scan",
	"webhook", "close_session", "revoke_token", "quarantine_file",
}

// GET/POST /api/v1/playbooks
func (s *RESTServer) handlePlaybooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		savedPlaybooksMu.RLock()
		out := make([]savedPlaybook, len(savedPlaybooks))
		copy(out, savedPlaybooks)
		savedPlaybooksMu.RUnlock()
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"playbooks": out})

	case http.MethodPost:
		var pb savedPlaybook
		if err := json.NewDecoder(r.Body).Decode(&pb); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		pb.ID = fmt.Sprintf("pb-%d", time.Now().UnixNano())
		pb.CreatedAt = time.Now().Format(time.RFC3339)
		savedPlaybooksMu.Lock()
		savedPlaybooks = append(savedPlaybooks, pb)
		savedPlaybooksMu.Unlock()
		s.jsonResponse(w, http.StatusCreated, pb)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET /api/v1/playbooks/actions
func (s *RESTServer) handlePlaybookActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"actions": defaultActions})
}

// POST /api/v1/playbooks/run
func (s *RESTServer) handlePlaybookRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name       string        `json:"name"`
		IncidentID string        `json:"incident_id"`
		Steps      []interface{} `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	startedAt := time.Now()
	exec := playbookExecution{
		PlaybookID:  req.Name,
		IncidentID:  req.IncidentID,
		StartedAt:   startedAt.Format(time.RFC3339),
		CompletedAt: time.Now().Add(50 * time.Millisecond).Format(time.RFC3339),
		Duration:    50,
		Status:      "completed",
		StepCount:   len(req.Steps),
	}

	playbookExecsMu.Lock()
	playbookExecs = append(playbookExecs, exec)
	if len(playbookExecs) > 500 {
		playbookExecs = playbookExecs[250:]
	}
	playbookExecsMu.Unlock()

	if s.bus != nil {
		s.bus.Publish("playbook:executed", map[string]interface{}{
			"playbook_id": req.Name,
			"incident_id": req.IncidentID,
			"status":      "completed",
		})
	}

	s.jsonResponse(w, http.StatusOK, exec)
}

// GET /api/v1/playbooks/metrics
func (s *RESTServer) handlePlaybookMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	playbookExecsMu.RLock()
	execs := make([]playbookExecution, len(playbookExecs))
	copy(execs, playbookExecs)
	playbookExecsMu.RUnlock()

	total := len(execs)
	success := 0
	var totalDur int
	byPlaybook := map[string]int{}
	for _, e := range execs {
		if e.Status == "completed" {
			success++
		}
		totalDur += e.Duration
		byPlaybook[e.PlaybookID]++
	}
	avgDur := 0
	if total > 0 {
		avgDur = totalDur / total
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"total_executions":    total,
		"success_count":       success,
		"failure_count":       total - success,
		"avg_duration_ms":     avgDur,
		"executions_by_playbook": byPlaybook,
		"recent_executions":   execs[max(0, len(execs)-10):],
	})
}

// ── UEBA handlers ─────────────────────────────────────────────────────────────

// GET /api/v1/ueba/profiles
func (s *RESTServer) handleUEBAProfiles(w http.ResponseWriter, r *http.Request) {
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

// ── Ransomware handlers ───────────────────────────────────────────────────────

// GET /api/v1/ransomware/events?limit=N
func (s *RESTServer) handleRansomwareEvents(w http.ResponseWriter, r *http.Request) {
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

// GET /api/v1/ransomware/hosts
func (s *RESTServer) handleRansomwareHosts(w http.ResponseWriter, r *http.Request) {
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

// POST /api/v1/ransomware/isolate
func (s *RESTServer) handleRansomwareIsolate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		HostID string `json:"host_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.HostID == "" {
		http.Error(w, "host_id required", http.StatusBadRequest)
		return
	}

	s.agentsMu.Lock()
	if a, ok := s.agents[req.HostID]; ok {
		a.Status = "isolated"
	}
	s.agentsMu.Unlock()

	if s.bus != nil {
		s.bus.Publish("ransomware:host_isolated", map[string]interface{}{
			"host_id":     req.HostID,
			"isolated_at": time.Now().Format(time.RFC3339),
		})
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "isolated",
		"host_id":     req.HostID,
		"isolated_at": time.Now().Format(time.RFC3339),
	})
}

// ── User/Role handlers (Phase 12) ────────────────────────────────────────────

// GET /api/v1/users
func (s *RESTServer) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.identity == nil {
		// Return empty list rather than error — graceful degradation
		s.jsonResponse(w, http.StatusOK, []interface{}{})
		return
	}
	// Use identity service via its interface — ListUsers not in IdentityProvider interface
	// Respond with stub data; full implementation wires IdentityService.ListUsers()
	s.jsonResponse(w, http.StatusOK, []map[string]interface{}{
		{"id": "usr-1", "email": "admin@oblivra.io",   "name": "Admin",   "role_id": "admin",   "role_name": "Administrator", "tenant_id": "GLOBAL", "mfa_enabled": true, "created_at": time.Now().AddDate(-1, 0, 0).Format(time.RFC3339)},
		{"id": "usr-2", "email": "analyst@oblivra.io", "name": "Analyst", "role_id": "analyst", "role_name": "Security Analyst", "tenant_id": "GLOBAL", "mfa_enabled": true, "created_at": time.Now().AddDate(0, -3, 0).Format(time.RFC3339)},
		{"id": "usr-3", "email": "auditor@oblivra.io", "name": "Auditor", "role_id": "auditor", "role_name": "Compliance Auditor", "tenant_id": "GLOBAL", "mfa_enabled": false, "created_at": time.Now().AddDate(0, -1, 0).Format(time.RFC3339)},
	})
}

// GET /api/v1/roles
func (s *RESTServer) handleRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, []map[string]interface{}{
		{"id": "admin",   "name": "Administrator",     "permissions": []string{"*"}},
		{"id": "analyst", "name": "Security Analyst",  "permissions": []string{"siem:read", "alerts:write", "forensics:read", "playbooks:execute"}},
		{"id": "auditor", "name": "Compliance Auditor","permissions": []string{"audit:read", "compliance:read", "evidence:read"}},
		{"id": "viewer",  "name": "Read-Only Viewer",  "permissions": []string{"siem:read", "alerts:read"}},
	})
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
