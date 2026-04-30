package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/parsers"
	"github.com/kingknull/oblivra/internal/services"
)

const maxBodyBytes = 1 << 20 // 1 MiB cap on ingest payloads (Phase 1)

type Server struct {
	log    *slog.Logger
	system *services.SystemService
	siem   *services.SiemService
	alerts *services.AlertService
	intel  *services.ThreatIntelService
	rules  *services.RulesService
	audit  *services.AuditService
	fleet  *services.FleetService
	ueba    *services.UebaService
	ndr     *services.NdrService
	foren   *services.ForensicsService
	tier    *services.TieringService
	lineage *services.LineageService
	vault    *services.VaultService
	timeline       *services.TimelineService
	investigations *services.InvestigationsService
	recon          *services.ReconstructionService
	tenantPolicy   *services.TenantPolicyService
	trust          *services.TrustService
	qual           *services.QualityService
	graph          *services.EvidenceGraphService
	bus    *events.Bus
	auth   *AuthMiddleware
	assets fs.FS
	mux    *http.ServeMux
}

type Deps struct {
	System *services.SystemService
	Siem   *services.SiemService
	Alerts *services.AlertService
	Intel  *services.ThreatIntelService
	Rules  *services.RulesService
	Audit  *services.AuditService
	Fleet  *services.FleetService
	Ueba    *services.UebaService
	Ndr     *services.NdrService
	Foren   *services.ForensicsService
	Tier    *services.TieringService
	Lineage *services.LineageService
	Vault    *services.VaultService
	Timeline       *services.TimelineService
	Investigations *services.InvestigationsService
	Reconstruction *services.ReconstructionService
	TenantPolicy   *services.TenantPolicyService
	Trust          *services.TrustService
	Quality        *services.QualityService
	Graph          *services.EvidenceGraphService
	Bus    *events.Bus
	Auth   *AuthMiddleware
	Assets fs.FS
}

func New(log *slog.Logger, deps Deps) *Server {
	s := &Server{
		log:    log,
		system: deps.System,
		siem:   deps.Siem,
		alerts: deps.Alerts,
		intel:  deps.Intel,
		rules:  deps.Rules,
		audit:  deps.Audit,
		fleet:  deps.Fleet,
		ueba:    deps.Ueba,
		ndr:     deps.Ndr,
		foren:   deps.Foren,
		tier:    deps.Tier,
		lineage: deps.Lineage,
		vault:    deps.Vault,
		timeline:       deps.Timeline,
		investigations: deps.Investigations,
		recon:          deps.Reconstruction,
		tenantPolicy:   deps.TenantPolicy,
		trust:          deps.Trust,
		qual:           deps.Quality,
		graph:          deps.Graph,
		bus:    deps.Bus,
		auth:   deps.Auth,
		assets: deps.Assets,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	var h http.Handler = s.mux
	// Order matters: auth must run before the audit middleware so the actor
	// (rbac.Subject) is in context when the audit entry is written.
	if s.audit != nil {
		h = queryAudit(s.audit, h)
	}
	if s.auth != nil {
		h = s.auth.Wrap(h)
	}
	return logging(s.log, security(h))
}

func (s *Server) routes() {
	// Liveness + metrics — never auth-gated, scraped by ops tooling.
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /readyz", s.health)
	s.mux.HandleFunc("GET /metrics", metricsHandler(s.siem, s.alerts, s.fleet))

	// System.
	s.mux.HandleFunc("GET /api/v1/system/info", s.systemInfo)
	s.mux.HandleFunc("GET /api/v1/system/ping", s.systemPing)

	// SIEM.
	s.mux.HandleFunc("POST /api/v1/siem/ingest", s.siemIngest)
	s.mux.HandleFunc("POST /api/v1/siem/ingest/batch", s.siemIngestBatch)
	s.mux.HandleFunc("POST /api/v1/siem/ingest/raw", s.siemIngestRaw)
	s.mux.HandleFunc("GET /api/v1/siem/search", s.siemSearch)
	s.mux.HandleFunc("GET /api/v1/siem/oql", s.siemOQL)
	s.mux.HandleFunc("GET /api/v1/siem/stats", s.siemStats)
	s.mux.HandleFunc("GET /api/v1/events", s.liveTail) // WebSocket upgrade

	// Alerts.
	if s.alerts != nil {
		s.mux.HandleFunc("GET /api/v1/alerts", s.listAlerts)
	}
	// Threat intel.
	if s.intel != nil {
		s.mux.HandleFunc("GET /api/v1/threatintel/lookup", s.intelLookup)
		s.mux.HandleFunc("POST /api/v1/threatintel/indicator", s.intelAdd)
		s.mux.HandleFunc("GET /api/v1/threatintel/indicators", s.intelList)
	}
	// Rules.
	if s.rules != nil {
		s.mux.HandleFunc("GET /api/v1/detection/rules", s.rulesList)
		s.mux.HandleFunc("POST /api/v1/detection/rules/reload", s.rulesReload)
		s.mux.HandleFunc("GET /api/v1/mitre/heatmap", s.mitreHeatmap)
	}
	// Audit.
	if s.audit != nil {
		s.mux.HandleFunc("GET /api/v1/audit/log", s.auditLog)
		s.mux.HandleFunc("POST /api/v1/audit/packages/generate", s.auditPackage)
		s.mux.HandleFunc("GET /api/v1/audit/verify", s.auditVerify)
	}
	// Fleet.
	if s.fleet != nil {
		s.mux.HandleFunc("GET /api/v1/agent/fleet", s.fleetList)
		s.mux.HandleFunc("POST /api/v1/agent/register", s.fleetRegister)
		s.mux.HandleFunc("POST /api/v1/agent/ingest", s.fleetIngest)
	}
	// UEBA.
	if s.ueba != nil {
		s.mux.HandleFunc("GET /api/v1/ueba/profiles", s.uebaProfiles)
		s.mux.HandleFunc("GET /api/v1/ueba/anomalies", s.uebaAnomalies)
	}
	// NDR.
	if s.ndr != nil {
		s.mux.HandleFunc("GET /api/v1/ndr/flows", s.ndrFlows)
		s.mux.HandleFunc("POST /api/v1/ndr/flows", s.ndrAdd)
		s.mux.HandleFunc("GET /api/v1/ndr/top-talkers", s.ndrTopTalkers)
	}
	// Forensics.
	if s.foren != nil {
		s.mux.HandleFunc("GET /api/v1/forensics/gaps", s.forenGaps)
		s.mux.HandleFunc("GET /api/v1/forensics/evidence", s.forenList)
		s.mux.HandleFunc("POST /api/v1/forensics/evidence", s.forenSeal)
	}
	// Tiering.
	if s.tier != nil {
		s.mux.HandleFunc("GET /api/v1/storage/stats", s.tierStats)
		s.mux.HandleFunc("POST /api/v1/storage/promote", s.tierPromote)
		s.mux.HandleFunc("GET /api/v1/storage/verify-warm", s.tierVerifyWarm)
	}
	// Trust.
	if s.trust != nil {
		s.mux.HandleFunc("GET /api/v1/trust/summary", s.trustSummary)
		s.mux.HandleFunc("GET /api/v1/trust/event/{id}", s.trustEvent)
	}
	// Quality / coverage.
	if s.qual != nil {
		s.mux.HandleFunc("GET /api/v1/quality/sources", s.qualSources)
		s.mux.HandleFunc("GET /api/v1/quality/coverage", s.qualCoverage)
	}
	// Network reconstruction.
	if s.recon != nil {
		s.mux.HandleFunc("GET /api/v1/reconstruction/flows", s.reconFlows)
		s.mux.HandleFunc("GET /api/v1/reconstruction/dns", s.reconDNS)
	}
	// Evidence graph.
	if s.graph != nil {
		s.mux.HandleFunc("GET /api/v1/graph/stats", s.graphStats)
		s.mux.HandleFunc("GET /api/v1/graph/subgraph", s.graphSubgraph)
	}
	// Tenant policy.
	if s.tenantPolicy != nil {
		s.mux.HandleFunc("GET /api/v1/tenants/policies", s.tenantPolicyList)
		s.mux.HandleFunc("PUT /api/v1/tenants/policies", s.tenantPolicySet)
	}
	// Lineage.
	if s.lineage != nil {
		s.mux.HandleFunc("GET /api/v1/forensics/lineage", s.lineageHosts)
		s.mux.HandleFunc("GET /api/v1/forensics/lineage/tree", s.lineageTree)
	}
	// Timeline.
	if s.timeline != nil {
		s.mux.HandleFunc("GET /api/v1/investigations/timeline", s.timelineGet)
	}
	// Reconstruction.
	if s.recon != nil {
		s.mux.HandleFunc("GET /api/v1/reconstruction/sessions", s.reconSessions)
		s.mux.HandleFunc("GET /api/v1/reconstruction/sessions/{id}", s.reconSessionGet)
		s.mux.HandleFunc("GET /api/v1/reconstruction/state", s.reconState)
	}
	// Cases.
	if s.investigations != nil {
		s.mux.HandleFunc("POST /api/v1/cases", s.caseOpen)
		s.mux.HandleFunc("GET /api/v1/cases", s.caseList)
		s.mux.HandleFunc("GET /api/v1/cases/{id}", s.caseGet)
		s.mux.HandleFunc("GET /api/v1/cases/{id}/timeline", s.caseTimeline)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/notes", s.caseNote)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/seal", s.caseSeal)
	}
	// Vault.
	if s.vault != nil {
		s.mux.HandleFunc("GET /api/v1/vault/status", s.vaultStatus)
		s.mux.HandleFunc("POST /api/v1/vault/init", s.vaultInit)
		s.mux.HandleFunc("POST /api/v1/vault/unlock", s.vaultUnlock)
		s.mux.HandleFunc("POST /api/v1/vault/lock", s.vaultLock)
		s.mux.HandleFunc("POST /api/v1/vault/secret", s.vaultSet)
		s.mux.HandleFunc("DELETE /api/v1/vault/secret", s.vaultDelete)
	}

	if s.assets != nil {
		s.mux.Handle("/", spaHandler(s.assets))
	}
}

func (s *Server) systemInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.system.Info())
}

func (s *Server) systemPing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.system.Ping())
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Server) siemIngest(w http.ResponseWriter, r *http.Request) {
	var ev events.Event
	if err := readJSON(r, &ev); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	stampProvenance(&ev, "rest", r)
	stored, err := s.siem.Ingest(r.Context(), ev)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, stored)
}

func (s *Server) siemIngestBatch(w http.ResponseWriter, r *http.Request) {
	var batch []events.Event
	if err := readJSON(r, &batch); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	for i := range batch {
		stampProvenance(&batch[i], "rest-batch", r)
	}
	written, err := s.siem.IngestBatch(r.Context(), batch)
	if err != nil {
		writeJSON(w, http.StatusPartialContent, map[string]any{
			"written": written, "error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"written": written})
}

func (s *Server) siemIngestRaw(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, maxBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	format := parsers.Format(r.URL.Query().Get("format"))
	if format == "" {
		format = parsers.FormatAuto
	}
	tenant := r.URL.Query().Get("tenant")
	count := 0
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ev, err := parsers.Parse(line, format)
		if err != nil || ev == nil {
			continue
		}
		if tenant != "" {
			ev.TenantID = tenant
		}
		ev.Provenance.IngestPath = "raw"
		ev.Provenance.Peer = strings.SplitN(r.RemoteAddr, ":", 2)[0]
		ev.Provenance.Format = string(format)
		ev.Provenance.Parser = string(format)
		if _, err := s.siem.Ingest(r.Context(), *ev); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		count++
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"written": count, "format": format})
}

func (s *Server) siemSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := services.SearchRequest{
		TenantID: q.Get("tenant"),
		Query:    q.Get("q"),
	}
	if v := q.Get("from"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			req.FromUnix = n
		}
	}
	if v := q.Get("to"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			req.ToUnix = n
		}
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.Limit = n
		}
	}
	if q.Get("newestFirst") == "true" {
		req.NewestFirst = true
	}
	resp, err := s.siem.Search(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) siemStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.siem.Stats())
}

func (s *Server) siemOQL(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	raw := q.Get("q")
	tenant := q.Get("tenant")
	var fromU, toU int64
	if v := q.Get("from"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			fromU = n
		}
	}
	if v := q.Get("to"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			toU = n
		}
	}
	resp, err := s.siem.SearchOQL(r.Context(), raw, tenant, fromU, toU)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// liveTail upgrades to a WebSocket and streams events in real time.
func (s *Server) liveTail(w http.ResponseWriter, r *http.Request) {
	if s.bus == nil {
		writeError(w, http.StatusServiceUnavailable, "event bus not available")
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Allow browser clients on different origins (dev preview at :5173, etc.).
		// Same-origin will still be the default in production.
		InsecureSkipVerify: true, //nolint:staticcheck // dev convenience; lock down in prod
	})
	if err != nil {
		s.log.Warn("ws accept", "err", err)
		return
	}
	defer conn.CloseNow()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ch, unsub := s.bus.Subscribe()
	defer unsub()

	// Send a hello frame so the client can confirm the channel.
	_ = wsjson.Write(ctx, conn, map[string]any{
		"type": "hello",
		"ts":   time.Now().UTC().Format(time.RFC3339Nano),
	})

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if err := wsjson.Write(ctx, conn, map[string]any{
				"type":  "event",
				"event": ev,
			}); err != nil {
				return
			}
		}
	}
}

// ---- Alerts ----

func (s *Server) listAlerts(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.alerts.Recent(limit))
}

// ---- Threat intel ----

func (s *Server) intelLookup(w http.ResponseWriter, r *http.Request) {
	val := r.URL.Query().Get("value")
	if val == "" {
		writeError(w, http.StatusBadRequest, "value query param required")
		return
	}
	writeJSON(w, http.StatusOK, s.intel.Lookup(val))
}

func (s *Server) intelAdd(w http.ResponseWriter, r *http.Request) {
	var ind services.Indicator
	if err := readJSON(r, &ind); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.intel.Add(ind)
	writeJSON(w, http.StatusCreated, ind)
}

func (s *Server) intelList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.intel.List())
}

// ---- Rules ----

func (s *Server) rulesList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.rules.List())
}

func (s *Server) rulesReload(w http.ResponseWriter, _ *http.Request) {
	n, err := s.rules.Reload()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"loaded": n})
}

func (s *Server) mitreHeatmap(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.rules.Heatmap())
}

// ---- Audit ----

func (s *Server) auditLog(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.audit.Recent(limit))
}

func (s *Server) auditPackage(w http.ResponseWriter, r *http.Request) {
	pkg, err := s.audit.GeneratePackage(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

func (s *Server) auditVerify(w http.ResponseWriter, _ *http.Request) {
	res := s.audit.Verify()
	writeJSON(w, http.StatusOK, res)
}

// ---- Fleet ----

func (s *Server) fleetList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.fleet.List())
}

func (s *Server) fleetRegister(w http.ResponseWriter, r *http.Request) {
	var req services.AgentRegistration
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	agent, err := s.fleet.Register(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

func (s *Server) fleetIngest(w http.ResponseWriter, r *http.Request) {
	var batch []events.Event
	if err := readJSON(r, &batch); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	agentID := r.URL.Query().Get("agentId")
	written, err := s.fleet.IngestFromAgent(r.Context(), agentID, batch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"written": written})
}

// ---- UEBA ----

func (s *Server) uebaProfiles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.ueba.Profiles())
}
func (s *Server) uebaAnomalies(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.ueba.Anomalies())
}

// ---- NDR ----

func (s *Server) ndrFlows(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.ndr.Recent(limit))
}

func (s *Server) ndrAdd(w http.ResponseWriter, r *http.Request) {
	var rec services.NetFlowRecord
	if err := readJSON(r, &rec); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.ndr.Record(rec)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "recorded"})
}

func (s *Server) ndrTopTalkers(w http.ResponseWriter, r *http.Request) {
	limit := 25
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.ndr.TopTalkers(limit))
}

// ---- Forensics ----

func (s *Server) forenGaps(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.foren.Gaps())
}
func (s *Server) forenList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.foren.List())
}

func (s *Server) forenSeal(w http.ResponseWriter, r *http.Request) {
	type req struct {
		TenantID string `json:"tenantId"`
		HostID   string `json:"hostId"`
		Title    string `json:"title"`
		FromUnix int64  `json:"fromUnix"`
		ToUnix   int64  `json:"toUnix"`
	}
	var body req
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	from := time.Unix(body.FromUnix, 0).UTC()
	to := time.Unix(body.ToUnix, 0).UTC()
	if body.FromUnix == 0 {
		from = time.Now().Add(-24 * time.Hour).UTC()
	}
	if body.ToUnix == 0 {
		to = time.Now().UTC()
	}
	item, err := s.foren.CollectByHost(r.Context(), body.TenantID, body.HostID, body.Title, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// ---- Tiering ----

func (s *Server) tierStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.tier.Stats())
}

func (s *Server) tierPromote(w http.ResponseWriter, r *http.Request) {
	moved, err := s.tier.Promote(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"moved": moved})
}

func (s *Server) tierVerifyWarm(w http.ResponseWriter, r *http.Request) {
	max := 50
	if v := r.URL.Query().Get("max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			max = n
		}
	}
	res, err := s.tier.VerifyWarm(max)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) tenantPolicyList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.tenantPolicy.List())
}

func (s *Server) trustSummary(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.trust.Summary())
}

func (s *Server) trustEvent(w http.ResponseWriter, r *http.Request) {
	rec, ok := s.trust.Of(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "no trust record for that event")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) qualSources(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.qual.Profiles())
}

func (s *Server) qualCoverage(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.qual.Coverage())
}

func (s *Server) reconFlows(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	writeJSON(w, http.StatusOK, s.recon.FlowsByHost(host))
}

func (s *Server) graphStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.graph.Stats())
}

func (s *Server) graphSubgraph(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kind := q.Get("kind")
	id := q.Get("id")
	if kind == "" || id == "" {
		writeError(w, http.StatusBadRequest, "kind and id query params required")
		return
	}
	depth := 2
	if v := q.Get("depth"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5 {
			depth = n
		}
	}
	writeJSON(w, http.StatusOK, s.graph.Subgraph(services.Node{Kind: kind, ID: id}, depth))
}

func (s *Server) reconDNS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("query")
	if q == "" {
		writeError(w, http.StatusBadRequest, "query param required")
		return
	}
	writeJSON(w, http.StatusOK, s.recon.DNSByQuery(q))
}

func (s *Server) tenantPolicySet(w http.ResponseWriter, r *http.Request) {
	var p services.TenantPolicy
	if err := readJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.tenantPolicy.Set(p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---- Lineage ----

func (s *Server) lineageHosts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.lineage.Hosts())
}

func (s *Server) lineageTree(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	if host == "" {
		writeError(w, http.StatusBadRequest, "host query param required")
		return
	}
	writeJSON(w, http.StatusOK, s.lineage.Tree(host))
}

// ---- Timeline ----

func (s *Server) timelineGet(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := services.TimelineRequest{
		TenantID: q.Get("tenant"),
		HostID:   q.Get("host"),
	}
	if v := q.Get("from"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			req.From = time.Unix(n, 0).UTC()
		}
	}
	if v := q.Get("to"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			req.To = time.Unix(n, 0).UTC()
		}
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			req.Limit = n
		}
	}
	out, err := s.timeline.Build(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ---- Reconstruction ----

func (s *Server) reconSessions(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	writeJSON(w, http.StatusOK, s.recon.Sessions(host))
}

func (s *Server) reconSessionGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.recon.Session(id)
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) reconState(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	if host == "" {
		writeError(w, http.StatusBadRequest, "host query param required")
		return
	}
	tenant := r.URL.Query().Get("tenant")
	at := time.Now().UTC()
	if v := r.URL.Query().Get("at"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			at = time.Unix(n, 0).UTC()
		}
	}
	snap, err := s.recon.StateAt(r.Context(), tenant, host, at)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// ---- Investigations / cases ----

func (s *Server) caseOpen(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title    string `json:"title"`
		HostID   string `json:"hostId"`
		TenantID string `json:"tenantId"`
		FromUnix int64  `json:"fromUnix"`
		ToUnix   int64  `json:"toUnix"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	openedBy := "anonymous"
	if sub, ok := rbacFromContext(r); ok {
		openedBy = sub
	}
	req := services.OpenCaseRequest{
		Title:    body.Title,
		HostID:   body.HostID,
		TenantID: body.TenantID,
		OpenedBy: openedBy,
	}
	if body.FromUnix > 0 {
		req.From = time.Unix(body.FromUnix, 0).UTC()
	}
	if body.ToUnix > 0 {
		req.To = time.Unix(body.ToUnix, 0).UTC()
	}
	c, err := s.investigations.Open(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) caseList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.investigations.List())
}

func (s *Server) caseGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, ok := s.investigations.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) caseTimeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	out, err := s.investigations.Timeline(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) caseNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Body string `json:"body"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	author := "anonymous"
	if sub, ok := rbacFromContext(r); ok {
		author = sub
	}
	c, err := s.investigations.AddNote(r.Context(), id, author, body.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) caseSeal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	by := "anonymous"
	if sub, ok := rbacFromContext(r); ok {
		by = sub
	}
	c, err := s.investigations.Seal(r.Context(), id, by)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// rbacFromContext extracts a "role:id" label from the auth context, or
// returns false if anonymous.
func rbacFromContext(r *http.Request) (string, bool) {
	actor, _ := actorOf(r.Context())
	if actor == "anonymous" {
		return "", false
	}
	return actor, true
}

// ---- Vault ----

func (s *Server) vaultStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.vault.Status())
}

type vaultReq struct {
	Passphrase string `json:"passphrase,omitempty"`
	Name       string `json:"name,omitempty"`
	Value      string `json:"value,omitempty"`
}

func (s *Server) vaultInit(w http.ResponseWriter, r *http.Request) {
	var req vaultReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Passphrase == "" {
		writeError(w, http.StatusBadRequest, "passphrase required")
		return
	}
	if err := s.vault.Initialize(req.Passphrase); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, s.vault.Status())
}

func (s *Server) vaultUnlock(w http.ResponseWriter, r *http.Request) {
	var req vaultReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.vault.Unlock(req.Passphrase); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.vault.Status())
}

func (s *Server) vaultLock(w http.ResponseWriter, _ *http.Request) {
	s.vault.Lock()
	writeJSON(w, http.StatusOK, s.vault.Status())
}

func (s *Server) vaultSet(w http.ResponseWriter, r *http.Request) {
	var req vaultReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	if err := s.vault.Set(req.Name, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) vaultDelete(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	if err := s.vault.Delete(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ---- helpers ----

func readJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body required")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// stampProvenance records how the event reached us. Called at the boundary
// where we still have the request — once Ingest is called the provenance is
// hashed into the content hash.
func stampProvenance(ev *events.Event, ingestPath string, r *http.Request) {
	ev.Provenance.IngestPath = ingestPath
	ev.Provenance.Peer = strings.SplitN(r.RemoteAddr, ":", 2)[0]
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		// Quick fingerprint of the first cert; used by the operator to spot
		// which mTLS principal sent the event.
		raw := r.TLS.PeerCertificates[0].Raw
		ev.Provenance.TLSFingerprint = sha256Hex(raw)
	}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func spaHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(root, path); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Debug("http", "method", r.Method, "path", r.URL.Path, "took", time.Since(start))
	})
}

func security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
