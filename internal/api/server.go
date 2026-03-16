package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/ssh"
	"github.com/kingknull/oblivrashell/internal/vault"
)

// Server configures and runs the local REST API for external control
type Server struct {
	mu     sync.Mutex
	srv    *http.Server
	cfg    Config
	port   int
	token  string // API Token for authentication
	db     database.DatabaseStore
	vault  vault.Provider
	log    *logger.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds the server config
type Config struct {
	Port       int
	Token      string
	// StrictTLS causes the server to refuse starting without valid TLS certs.
	// Set to false only during development/testing.
	StrictTLS  bool
}

func NewServer(cfg Config, db database.DatabaseStore, v vault.Provider, log *logger.Logger) *Server {
	return &Server{
		cfg:   cfg,
		port:  cfg.Port,
		token: cfg.Token,
		db:    db,
		vault: v,
		log:   log.WithPrefix("api"),
	}
}

// Start launches the API server in the background
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv != nil {
		return fmt.Errorf("server already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()

	// Middleware
	handler := s.authMiddleware(s.logMiddleware(mux))

	// Routes
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/hosts", s.handleHosts)
	mux.HandleFunc("/api/v1/hosts/", s.handleHostAction)
	mux.HandleFunc("/api/v1/exec", s.handleExec)

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	s.srv = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// ── TLS Configuration ─────────────────────────────────────────────────
	home, _ := os.UserHomeDir()
	certPath := filepath.Join(home, ".oblivrashell", "cert.pem")
	keyPath := filepath.Join(home, ".oblivrashell", "key.pem")

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)
	hasCerts := certErr == nil && keyErr == nil

	if !hasCerts {
		if s.cfg.StrictTLS {
			// In strict mode the server MUST have TLS — refuse to start plaintext.
			// This prevents a misconfigured production deployment from silently
			// exposing sensitive API endpoints over an unencrypted channel.
			s.log.Error("FATAL: StrictTLS is enabled but no TLS certificates found at %s — refusing to start insecure API. Generate certs or disable StrictTLS for development.", certPath)
			s.cancel()
			return fmt.Errorf("strict TLS required but certificates not found at %s", certPath)
		}
		s.log.Warn("⚠️  TLS certificates not found at %s — API running in PLAINTEXT mode (development only). Set StrictTLS=true for production.", certPath)
	}

	if hasCerts {
		// Validate the certificate pair is loadable before committing to TLS
		if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
			if s.cfg.StrictTLS {
				s.cancel()
				return fmt.Errorf("strict TLS: certificate validation failed: %w", err)
			}
			s.log.Warn("⚠️  TLS cert/key pair invalid (%v) — falling back to plaintext.", err)
			hasCerts = false
		}
	}

	serveFunc := func() error {
		if hasCerts {
			return s.srv.ListenAndServeTLS(certPath, keyPath)
		}
		return s.srv.ListenAndServe()
	}

	if hasCerts {
		s.log.Info("Starting REST API (TLS) on https://%s", addr)
	} else {
		s.log.Info("Starting REST API (plaintext) on http://%s", addr)
	}

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				if err := serveFunc(); err != nil && err != http.ErrServerClosed {
					s.log.Error("API Server error: %v. Retrying in 5s...", err)
					time.Sleep(5 * time.Second)
				} else {
					return
				}
			}
		}
	}()

	return nil
}

// Stop stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv == nil {
		return nil
	}

	s.log.Info("Stopping REST API (Waiting up to 10s for connections to drain)...")
	s.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := s.srv.Shutdown(ctx)
	s.srv = nil
	return err
}

// ---- Middleware ----

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token == "" {
			// No token configured, allow localhost only
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != s.token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.log.Debug("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// ---- Handlers ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enforce RBAC: Non-admin/analyst tokens cannot dump the core inventory
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin && role != auth.RoleAnalyst {
		http.Error(w, "Forbidden: Action requires Admin or Analyst privileges", http.StatusForbidden)
		return
	}

	// Use tenant-scoped context for all DB operations
	ctx := database.WithTenantID(r.Context(), database.DefaultTenantID)
	query := `SELECT id, label, hostname, port, username, auth_method FROM hosts WHERE tenant_id = ?`
	rows, err := s.db.DB().QueryContext(ctx, query, database.DefaultTenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var hosts []map[string]interface{}
	for rows.Next() {
		var h database.Host
		if err := rows.Scan(&h.ID, &h.Label, &h.Hostname, &h.Port, &h.Username, &h.AuthMethod); err != nil {
			continue
		}
		hosts = append(hosts, map[string]interface{}{
			"id":       h.ID,
			"label":    h.Label,
			"hostname": h.Hostname,
			"username": h.Username,
			"port":     h.Port,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hosts)
}

func (s *Server) handleHostAction(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/hosts/")
	if id == "" {
		http.Error(w, "Host ID required", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enforce RBAC: Non-admin/analyst tokens cannot dump core inventory
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin && role != auth.RoleAnalyst {
		http.Error(w, "Forbidden: Action requires Admin or Analyst privileges", http.StatusForbidden)
		return
	}

	ctx := database.WithTenantID(r.Context(), database.DefaultTenantID)
	query := `SELECT id, label, hostname, port, username, auth_method FROM hosts WHERE id = ? AND tenant_id = ?`
	row := s.db.DB().QueryRowContext(ctx, query, id, database.DefaultTenantID)

	var h database.Host
	err := row.Scan(&h.ID, &h.Label, &h.Hostname, &h.Port, &h.Username, &h.AuthMethod)
	if err != nil {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       h.ID,
		"label":    h.Label,
		"hostname": h.Hostname,
		"username": h.Username,
		"port":     h.Port,
	})
}

func (s *Server) handleExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enforce RBAC: Only Admins can execute remote commands
	if auth.GetRole(r.Context()) != auth.RoleAdmin {
		http.Error(w, "Forbidden: Action requires Admin privileges", http.StatusForbidden)
		return
	}

	var req struct {
		HostID  string `json:"host_id"`
		Command string `json:"command"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.HostID == "" || req.Command == "" {
		http.Error(w, "host_id and command required", http.StatusBadRequest)
		return
	}

	execCtx := database.WithTenantID(r.Context(), database.DefaultTenantID)
	query := `SELECT id, label, hostname, port, username, auth_method FROM hosts WHERE id = ? AND tenant_id = ?`
	row := s.db.DB().QueryRowContext(execCtx, query, req.HostID, database.DefaultTenantID)

	var h database.Host
	err := row.Scan(&h.ID, &h.Label, &h.Hostname, &h.Port, &h.Username, &h.AuthMethod)
	if err != nil {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	cfg := ssh.DefaultConfig()
	cfg.Host = h.Hostname
	cfg.Port = h.Port
	cfg.Username = h.Username
	cfg.AuthMethod = ssh.AuthMethod(h.AuthMethod)

	client := ssh.NewClient(cfg)
	if client == nil {
		http.Error(w, "Failed to create SSH client", http.StatusInternalServerError)
		return
	}

	if err := client.Connect(); err != nil {
		http.Error(w, "Connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	output, err := client.ExecuteCommand(req.Command)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":  err.Error(),
			"output": string(output),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"output": string(output),
	})
}
