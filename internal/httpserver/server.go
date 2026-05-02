package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
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
	imp            *services.ImportService
	report         *services.ReportService
	tamper         *services.TamperService
	webhooks       *services.WebhookService
	categories     *services.CategoriesService
	notifications  *services.NotificationService
	savedSearches  *services.SavedSearchService
	oidc           *OIDCHandler
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
	Import         *services.ImportService
	Report         *services.ReportService
	Tamper         *services.TamperService
	Webhooks       *services.WebhookService
	Categories     *services.CategoriesService
	Notifications  *services.NotificationService
	SavedSearches  *services.SavedSearchService
	OIDC           *OIDCHandler
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
		imp:            deps.Import,
		report:         deps.Report,
		tamper:         deps.Tamper,
		webhooks:       deps.Webhooks,
		categories:     deps.Categories,
		notifications:  deps.Notifications,
		savedSearches:  deps.SavedSearches,
		oidc:           deps.OIDC,
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
		s.mux.HandleFunc("GET /api/v1/alerts/{id}", s.getAlert)
		s.mux.HandleFunc("POST /api/v1/alerts/{id}/ack", s.alertAck)
		s.mux.HandleFunc("POST /api/v1/alerts/{id}/assign", s.alertAssign)
		s.mux.HandleFunc("POST /api/v1/alerts/{id}/resolve", s.alertResolve)
		s.mux.HandleFunc("POST /api/v1/alerts/{id}/reopen", s.alertReopen)
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
		s.mux.HandleFunc("GET /api/v1/detection/rules/effectiveness", s.rulesEffectiveness)
		s.mux.HandleFunc("POST /api/v1/detection/rules/{id}/feedback", s.rulesFeedback)
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
		s.mux.HandleFunc("GET /api/v1/agent/fleet/{id}", s.fleetGet)
		s.mux.HandleFunc("POST /api/v1/agent/register", s.fleetRegister)
		s.mux.HandleFunc("POST /api/v1/agent/heartbeat", s.fleetHeartbeat)
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
	// Import / backfill.
	if s.imp != nil {
		s.mux.HandleFunc("POST /api/v1/import", s.runImport)
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
		s.mux.HandleFunc("GET /api/v1/forensics/lineage/cross-host", s.lineageCrossHost)
	}
	// Tamper findings.
	if s.tamper != nil {
		s.mux.HandleFunc("GET /api/v1/forensics/tamper", s.tamperFindings)
	}
	// Webhooks.
	if s.webhooks != nil {
		s.mux.HandleFunc("GET /api/v1/webhooks", s.webhookList)
		s.mux.HandleFunc("POST /api/v1/webhooks", s.webhookRegister)
		s.mux.HandleFunc("DELETE /api/v1/webhooks/{id}", s.webhookDelete)
		s.mux.HandleFunc("GET /api/v1/webhooks/deliveries", s.webhookDeliveries)
	}
	// OIDC SSO (optional — only mounted when configured).
	if s.oidc != nil && s.oidc.Configured() {
		s.mux.HandleFunc("GET /api/v1/auth/oidc/login", s.oidc.Login)
		s.mux.HandleFunc("GET /api/v1/auth/oidc/callback", s.oidc.Callback)
	}
	// Timeline.
	if s.timeline != nil {
		s.mux.HandleFunc("GET /api/v1/investigations/timeline", s.timelineGet)
		s.mux.HandleFunc("GET /api/v1/investigations/pivot", s.timelinePivot)
	}
	// Reconstruction.
	if s.recon != nil {
		s.mux.HandleFunc("GET /api/v1/reconstruction/sessions", s.reconSessions)
		s.mux.HandleFunc("GET /api/v1/reconstruction/sessions/{id}", s.reconSessionGet)
		s.mux.HandleFunc("GET /api/v1/reconstruction/state", s.reconState)
		s.mux.HandleFunc("GET /api/v1/reconstruction/entities", s.reconEntities)
		s.mux.HandleFunc("GET /api/v1/reconstruction/entities/{kind}/{id}", s.reconEntityProfile)
		s.mux.HandleFunc("GET /api/v1/reconstruction/cmdline", s.reconCmdLines)
		s.mux.HandleFunc("GET /api/v1/reconstruction/cmdline/suspicious", s.reconCmdLineSus)
		s.mux.HandleFunc("GET /api/v1/reconstruction/auth", s.reconAuthByUser)
		s.mux.HandleFunc("GET /api/v1/reconstruction/auth/multi-protocol", s.reconAuthMulti)
	}
	// Cases.
	if s.investigations != nil {
		s.mux.HandleFunc("POST /api/v1/cases", s.caseOpen)
		s.mux.HandleFunc("GET /api/v1/cases", s.caseList)
		s.mux.HandleFunc("GET /api/v1/cases/{id}", s.caseGet)
		s.mux.HandleFunc("GET /api/v1/cases/{id}/timeline", s.caseTimeline)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/notes", s.caseNote)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/seal", s.caseSeal)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/hypotheses", s.caseHypothesisAdd)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/hypotheses/{hid}", s.caseHypothesisStatus)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/annotate", s.caseAnnotate)
		s.mux.HandleFunc("GET /api/v1/cases/{id}/confidence", s.caseConfidence)
		s.mux.HandleFunc("GET /api/v1/cases/{id}/report.html", s.caseReportHTML)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/legal/submit", s.caseLegalSubmit)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/legal/approve", s.caseLegalApprove)
		s.mux.HandleFunc("POST /api/v1/cases/{id}/legal/reject", s.caseLegalReject)
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

	// Universal forwarder compatibility (Phase 41).
	s.mux.HandleFunc("POST /services/collector/event", s.hecHandler())
	s.mux.HandleFunc("POST /services/collector", s.hecHandler())
	s.mux.HandleFunc("POST /v1/logs", s.otlpLogsHandler())

	// Phase 46 — Compliance attestation feed (read-only).
	if s.audit != nil {
		s.mux.HandleFunc("GET /api/v1/compliance/feed/{framework}", s.complianceFeedHandler())
	}

	// Categories — sourceType breakdown for the operator UI.
	if s.categories != nil {
		s.mux.HandleFunc("GET /api/v1/categories", s.categoriesList)
	}

	// Notifications — email + webhook channels with throttling.
	if s.notifications != nil {
		s.mux.HandleFunc("GET /api/v1/notifications", s.notificationsList)
		s.mux.HandleFunc("POST /api/v1/notifications", s.notificationsAdd)
		s.mux.HandleFunc("POST /api/v1/notifications/{id}/test", s.notificationsTest)
		s.mux.HandleFunc("DELETE /api/v1/notifications/{id}", s.notificationsDelete)
	}

	// Saved searches — operator-managed, optionally scheduled queries.
	if s.savedSearches != nil {
		s.mux.HandleFunc("GET /api/v1/saved-searches", s.savedSearchesList)
		s.mux.HandleFunc("POST /api/v1/saved-searches", s.savedSearchesSave)
		s.mux.HandleFunc("POST /api/v1/saved-searches/{id}/run", s.savedSearchesRun)
		s.mux.HandleFunc("DELETE /api/v1/saved-searches/{id}", s.savedSearchesDelete)
	}

	// Event detail — drill-through from search results.
	s.mux.HandleFunc("GET /api/v1/siem/events/{id}", s.eventDetail)

	// Phase 47 — pprof, behind the standard auth middleware.
	if s.auth != nil && s.auth.Required() {
		registerPprof(s.mux)
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
	switch q.Get("format") {
	case "csv":
		writeEventsCSV(w, resp.Events)
	case "ndjson":
		writeEventsNDJSON(w, resp.Events)
	default:
		writeJSON(w, http.StatusOK, resp)
	}
}

// writeEventsCSV streams a search result set as CSV. Columns are the
// stable subset analysts ask for first; everything else lands in a
// `fields` JSON column so nothing is lost in the export.
func writeEventsCSV(w http.ResponseWriter, evs []events.Event) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="oblivra-events.csv"`)
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{"id", "timestamp", "tenantId", "hostId", "source", "eventType", "severity", "message", "fields"})
	for _, e := range evs {
		var fbuf string
		if len(e.Fields) > 0 {
			b, _ := json.Marshal(e.Fields)
			fbuf = string(b)
		}
		_ = cw.Write([]string{
			e.ID,
			e.Timestamp.UTC().Format(time.RFC3339Nano),
			e.TenantID,
			e.HostID,
			string(e.Source),
			e.EventType,
			string(e.Severity),
			e.Message,
			fbuf,
		})
	}
}

// writeEventsNDJSON streams events as newline-delimited JSON — the
// shape downstream tools (jq, ndjson loaders, log shippers) prefer.
func writeEventsNDJSON(w http.ResponseWriter, evs []events.Event) {
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="oblivra-events.ndjson"`)
	enc := json.NewEncoder(w)
	for _, e := range evs {
		_ = enc.Encode(e)
	}
}

// eventDetail returns one event's full record. Drives the per-event
// drill-down in the SIEM view. Searches the hot store via Search()
// with a hash-id filter — slower than a direct lookup but keeps the
// SiemService surface simple.
func (s *Server) eventDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}
	// Pull a wide-enough recent slice and filter by ID. The hot store
	// keeps O(seconds) of events; for older events the analyst would
	// need to query the warm tier (a roadmap item).
	res, err := s.siem.Search(r.Context(), services.SearchRequest{Limit: 5000, NewestFirst: true})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, e := range res.Events {
		if e.ID == id {
			// Surface related-events context: same host within ±60s.
			related := []events.Event{}
			lo := e.Timestamp.Add(-60 * time.Second)
			hi := e.Timestamp.Add(60 * time.Second)
			for _, other := range res.Events {
				if other.ID == id {
					continue
				}
				if other.HostID != e.HostID {
					continue
				}
				if other.Timestamp.Before(lo) || other.Timestamp.After(hi) {
					continue
				}
				related = append(related, other)
				if len(related) >= 50 {
					break
				}
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"event":   e,
				"related": related,
			})
			return
		}
	}
	writeError(w, http.StatusNotFound, "event not found in hot store; try the warm-tier verifier")
}

// ---- Saved searches ----

func (s *Server) savedSearchesList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.savedSearches.List())
}

func (s *Server) savedSearchesSave(w http.ResponseWriter, r *http.Request) {
	var q services.SavedSearch
	if err := readJSON(r, &q); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	q.CreatedBy = alertActor(r)
	out, err := s.savedSearches.Save(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.audit != nil {
		s.audit.Append(r.Context(), alertActor(r), "saved-search.save", "default",
			map[string]string{"id": out.ID, "name": out.Name})
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) savedSearchesRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	hits, err := s.savedSearches.Run(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "hits": hits})
}

func (s *Server) savedSearchesDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.savedSearches.Delete(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	// Origin allow-list: same-origin always works; OBLIVRA_WS_ORIGINS is a
	// comma-separated list of additional patterns (e.g. "localhost:5173"
	// for the Vite dev server). Without that env, only same-origin
	// requests are accepted — closes a cross-origin WebSocket CSRF surface.
	opts := &websocket.AcceptOptions{}
	if extra := os.Getenv("OBLIVRA_WS_ORIGINS"); extra != "" {
		for _, p := range strings.Split(extra, ",") {
			if p = strings.TrimSpace(p); p != "" {
				opts.OriginPatterns = append(opts.OriginPatterns, p)
			}
		}
	}
	conn, err := websocket.Accept(w, r, opts)
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

// getAlert returns a single alert; drives the per-alert detail page.
func (s *Server) getAlert(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, ok := s.alerts.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "no such alert")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// alertActor pulls the authenticated subject out of the request
// context (set by the auth middleware). Falls back to "operator" so a
// no-auth dev server still records something useful.
func alertActor(r *http.Request) string {
	actor, _ := actorOf(r.Context())
	if actor == "" {
		return "operator"
	}
	return actor
}

func (s *Server) alertAck(w http.ResponseWriter, r *http.Request) {
	a, ok := s.alerts.Ack(r.PathValue("id"), alertActor(r))
	if !ok {
		writeError(w, http.StatusNotFound, "no such alert")
		return
	}
	if s.audit != nil {
		s.audit.Append(r.Context(), alertActor(r), "alert.ack", a.TenantID,
			map[string]string{"id": a.ID, "rule": a.RuleID})
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) alertAssign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Assignee string `json:"assignee"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Assignee == "" {
		writeError(w, http.StatusBadRequest, "assignee required")
		return
	}
	a, ok := s.alerts.Assign(r.PathValue("id"), alertActor(r), body.Assignee)
	if !ok {
		writeError(w, http.StatusNotFound, "no such alert")
		return
	}
	if s.audit != nil {
		s.audit.Append(r.Context(), alertActor(r), "alert.assign", a.TenantID,
			map[string]string{"id": a.ID, "assignee": body.Assignee})
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) alertResolve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Verdict string `json:"verdict"`
		Notes   string `json:"notes"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	a, ok := s.alerts.Resolve(r.PathValue("id"), alertActor(r), body.Verdict, body.Notes)
	if !ok {
		writeError(w, http.StatusNotFound, "no such alert")
		return
	}
	// Map verdict → rule effectiveness feedback so the rules.MarkAlert
	// scorecard tracks operator triage automatically.
	if s.rules != nil {
		switch body.Verdict {
		case "true-positive", "benign-true-positive":
			s.rules.MarkAlert(a.RuleID, "tp")
		case "false-positive":
			s.rules.MarkAlert(a.RuleID, "fp")
		}
	}
	if s.audit != nil {
		s.audit.Append(r.Context(), alertActor(r), "alert.resolve", a.TenantID,
			map[string]string{"id": a.ID, "verdict": body.Verdict})
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) alertReopen(w http.ResponseWriter, r *http.Request) {
	a, ok := s.alerts.Reopen(r.PathValue("id"), alertActor(r))
	if !ok {
		writeError(w, http.StatusNotFound, "no such alert")
		return
	}
	if s.audit != nil {
		s.audit.Append(r.Context(), alertActor(r), "alert.reopen", a.TenantID,
			map[string]string{"id": a.ID})
	}
	writeJSON(w, http.StatusOK, a)
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

// rulesEffectiveness returns the per-rule scorecard (Phase 48): cumulative
// fires, recent fires, analyst-marked TP/FP counts, and running FP rate.
func (s *Server) rulesEffectiveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.rules.Effectiveness())
}

// rulesFeedback marks an alert as TP or FP. The rule ID is in the path;
// the body is `{"label": "tp"|"fp"}`. Lands in the audit chain so a
// reviewer can see who tuned which rule.
func (s *Server) rulesFeedback(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing rule id")
		return
	}
	var body struct {
		Label string `json:"label"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Label != "tp" && body.Label != "fp" {
		writeError(w, http.StatusBadRequest, `label must be "tp" or "fp"`)
		return
	}
	s.rules.MarkAlert(id, body.Label)
	if s.audit != nil {
		s.audit.Append(r.Context(), "analyst", "rule.feedback", "default", map[string]string{
			"ruleId": id,
			"label":  body.Label,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ruleId": id, "label": body.Label})
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

// fleetGet returns the full record for a single agent — drives the
// fleet detail panel in the operator UI.
func (s *Server) fleetGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing agent id")
		return
	}
	a, ok := s.fleet.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "no such agent")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// fleetHeartbeat records a rich self-report from an agent (pubkey
// fingerprint, spill bytes, queue depth, dropped count, etc.). Cheap
// to call; agents send this every 30s by default. Updates LastSeen so
// the existing "healthy in last 5m" tile picks it up too.
func (s *Server) fleetHeartbeat(w http.ResponseWriter, r *http.Request) {
	var stats services.HeartbeatStats
	if err := readJSON(r, &stats); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.fleet.Heartbeat(stats); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agentId": stats.AgentID, "ackAt": time.Now().UTC().Format(time.RFC3339)})
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

func (s *Server) runImport(w http.ResponseWriter, r *http.Request) {
	tenant := r.URL.Query().Get("tenant")
	source := r.URL.Query().Get("source")
	format := r.URL.Query().Get("format")
	// Cap: 256 MiB per request — chunk larger imports through the CLI.
	r.Body = http.MaxBytesReader(nil, r.Body, 256<<20)
	res, err := s.imp.Run(r.Context(), r.Body, tenant, source, format)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, res)
}

func (s *Server) reconEntities(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "host"
	}
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.recon.EntityList(kind, limit))
}

func (s *Server) reconEntityProfile(w http.ResponseWriter, r *http.Request) {
	p := s.recon.EntityProfile(r.PathValue("kind"), r.PathValue("id"))
	if p == nil {
		writeError(w, http.StatusNotFound, "entity not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) reconCmdLines(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.recon.CmdLines(host, limit))
}

func (s *Server) reconCmdLineSus(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.recon.SuspiciousCmdLines(limit))
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

func (s *Server) lineageCrossHost(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name query param required")
		return
	}
	writeJSON(w, http.StatusOK, s.lineage.CrossHostByName(name))
}

func (s *Server) webhookList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.webhooks.List())
}

func (s *Server) webhookRegister(w http.ResponseWriter, r *http.Request) {
	var hook services.Webhook
	if err := readJSON(r, &hook); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.webhooks.Register(hook)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) webhookDelete(w http.ResponseWriter, r *http.Request) {
	s.webhooks.Delete(r.PathValue("id"))
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) webhookDeliveries(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.webhooks.Recent(limit))
}

func (s *Server) tamperFindings(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.tamper.Findings(host, limit))
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

func (s *Server) caseHypothesisAdd(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Statement string `json:"statement"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	author, _ := actorOf(r.Context())
	if author == "" {
		author = "anonymous"
	}
	c, err := s.investigations.AddHypothesis(r.Context(), id, author, body.Statement)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) caseHypothesisStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	hid := r.PathValue("hid")
	var body struct {
		Status      string   `json:"status"`
		EvidenceIDs []string `json:"evidenceIds"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	author, _ := actorOf(r.Context())
	if author == "" {
		author = "anonymous"
	}
	c, err := s.investigations.SetHypothesisStatus(r.Context(), id, hid, author, body.Status, body.EvidenceIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) caseAnnotate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		EventID string `json:"eventId"`
		Body    string `json:"body"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	author, _ := actorOf(r.Context())
	if author == "" {
		author = "anonymous"
	}
	c, err := s.investigations.Annotate(r.Context(), id, body.EventID, author, body.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) caseReportHTML(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, err := s.report.CaseHTML(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="oblivra-evidence-`+id+`.html"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Server) caseConfidence(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	conf, err := s.investigations.Confidence(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conf)
}

func (s *Server) caseLegalSubmit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	by, _ := actorOf(r.Context())
	if by == "" {
		by = "anonymous"
	}
	c, err := s.investigations.SubmitForLegalReview(r.Context(), id, by)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) caseLegalApprove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct{ Reason string `json:"reason"` }
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	by, _ := actorOf(r.Context())
	if by == "" {
		by = "anonymous"
	}
	c, err := s.investigations.LegalApprove(r.Context(), id, by, body.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) caseLegalReject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct{ Reason string `json:"reason"` }
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	by, _ := actorOf(r.Context())
	if by == "" {
		by = "anonymous"
	}
	c, err := s.investigations.LegalReject(r.Context(), id, by, body.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) timelinePivot(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	host := q.Get("host")
	tenant := q.Get("tenant")
	pivotUnix, _ := strconv.ParseInt(q.Get("at"), 10, 64)
	deltaSec, _ := strconv.Atoi(q.Get("delta"))
	if deltaSec <= 0 {
		deltaSec = 900 // ±15 minutes
	}
	out, err := s.timeline.PivotWindow(r.Context(), tenant, host, time.Unix(pivotUnix, 0).UTC(), time.Duration(deltaSec)*time.Second, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) reconAuthByUser(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		writeError(w, http.StatusBadRequest, "user query param required")
		return
	}
	writeJSON(w, http.StatusOK, s.recon.AuthChainsByUser(user))
}

func (s *Server) reconAuthMulti(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, s.recon.AuthMultiProtocol(limit))
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

// ---- Categories ---------------------------------------------------------

func (s *Server) categoriesList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.categories.List())
}

// ---- Notifications ------------------------------------------------------

func (s *Server) notificationsList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.notifications.List())
}

func (s *Server) notificationsAdd(w http.ResponseWriter, r *http.Request) {
	var c services.NotificationChannel
	if err := readJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.notifications.Register(r.Context(), c)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) notificationsTest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}
	if err := s.notifications.Test(r.Context(), id); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"delivered": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"delivered": true})
}

func (s *Server) notificationsDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}
	if err := s.notifications.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
