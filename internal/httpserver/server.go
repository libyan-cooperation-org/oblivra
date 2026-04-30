package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/services"
)

const maxBodyBytes = 1 << 20 // 1 MiB cap on ingest payloads (Phase 1)

type Server struct {
	log    *slog.Logger
	system *services.SystemService
	siem   *services.SiemService
	assets fs.FS
	mux    *http.ServeMux
}

type Deps struct {
	System *services.SystemService
	Siem   *services.SiemService
	Assets fs.FS
}

func New(log *slog.Logger, deps Deps) *Server {
	s := &Server{
		log:    log,
		system: deps.System,
		siem:   deps.Siem,
		assets: deps.Assets,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return logging(s.log, security(s.mux))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /readyz", s.health)

	s.mux.HandleFunc("GET /api/v1/system/info", s.systemInfo)
	s.mux.HandleFunc("GET /api/v1/system/ping", s.systemPing)

	s.mux.HandleFunc("POST /api/v1/siem/ingest", s.siemIngest)
	s.mux.HandleFunc("POST /api/v1/siem/ingest/batch", s.siemIngestBatch)
	s.mux.HandleFunc("GET /api/v1/siem/search", s.siemSearch)
	s.mux.HandleFunc("GET /api/v1/siem/stats", s.siemStats)

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

// spaHandler serves static files from fs and falls back to /index.html for
// unknown routes so the Svelte router can take over.
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
