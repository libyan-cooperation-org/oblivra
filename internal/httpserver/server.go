package httpserver

import (
	"context"
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
	if s.auth != nil {
		h = s.auth.Wrap(h)
	}
	return logging(s.log, security(h))
}

func (s *Server) routes() {
	// Liveness — never auth-gated.
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /readyz", s.health)

	// System.
	s.mux.HandleFunc("GET /api/v1/system/info", s.systemInfo)
	s.mux.HandleFunc("GET /api/v1/system/ping", s.systemPing)

	// SIEM.
	s.mux.HandleFunc("POST /api/v1/siem/ingest", s.siemIngest)
	s.mux.HandleFunc("POST /api/v1/siem/ingest/batch", s.siemIngestBatch)
	s.mux.HandleFunc("POST /api/v1/siem/ingest/raw", s.siemIngestRaw)
	s.mux.HandleFunc("GET /api/v1/siem/search", s.siemSearch)
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
