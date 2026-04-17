package ingest

import (
	"compress/zlib"
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/kingknull/oblivrashell/internal/agent"
	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/time/rate"
)

// AgentServer listens for incoming telemetry from deployed Oblivra agents
// It handles mTLS, payload decompression (zlib), and JSON decoding into the ingest pipeline
type AgentServer struct {
	pipeline  IngestionPipeline
	port      int
	server    *http.Server
	log       *logger.Logger
	mu        sync.RWMutex
	validator *validator.Validate
	limiters  sync.Map

	// TLS Config paths
	certFile string
	keyFile  string
	caFile   string // Optional: for requiring client certs (mTLS)

	// Agent Tracking
	activeAgents   map[string]AgentInfo
	desiredConfig  FleetConfig
	pendingActions map[string][]PendingAction // agentID -> actions
}

// FleetConfig is the subset of agent configuration that can be pushed from the server.
type FleetConfig struct {
	Interval       time.Duration `json:"interval"`
	EnableFIM      bool          `json:"enable_fim"`
	EnableSyslog   bool          `json:"enable_syslog"`
	EnableMetrics  bool          `json:"enable_metrics"`
	EnableEventLog bool          `json:"enable_event_log"`
	Quarantine     bool          `json:"quarantine"` // Isolate agent from all non-C2 network traffic
}

// ActionType defines forensic or response operations the agent can perform.
type ActionType string

const (
	ActionKillProcess      ActionType = "kill_process"
	ActionProcessSnapshot  ActionType = "process_snapshot"
	ActionProcessInventory ActionType = "process_inventory"
	ActionIsolateNetwork   ActionType = "isolate_network"
	ActionRestoreNetwork   ActionType = "restore_network"
)

// PendingAction represents a command waiting for an agent to pull.
type PendingAction struct {
	ID      string            `json:"id"`
	Type    ActionType        `json:"type"`
	Payload map[string]string `json:"payload"`
}

// AgentInfo stores metadata about connected agents
type AgentInfo struct {
	ID            string    `json:"id"`
	Hostname      string    `json:"hostname"`
	TenantID      string    `json:"tenant_id"` // Added for isolation
	Version       string    `json:"version"`
	LastSeen      string    `json:"last_seen"`
	RemoteAddress string    `json:"remote_address"`
	OS            string    `json:"os"`
	Arch          string          `json:"arch"`
	Collectors    []string        `json:"collectors"`
	PublicKey     []byte          `json:"public_key"`
	TrustLevel    string          `json:"trust_level"`     // "unverified", "verified", "compromised"
	WatchdogActive bool           `json:"watchdog_active"`
}



// NewAgentServer creates a new agent ingestion server
func NewAgentServer(pipeline IngestionPipeline, port int, certFile, keyFile, caFile string, log *logger.Logger) *AgentServer {
	return &AgentServer{
		pipeline:       pipeline,
		port:           port,
		log:            log.WithPrefix("agent_server"),
		validator:      validator.New(),
		certFile:       certFile,
		keyFile:        keyFile,
		caFile:         caFile,
		activeAgents:   make(map[string]AgentInfo),
		pendingActions: make(map[string][]PendingAction),
		desiredConfig: FleetConfig{
			Interval:       10 * time.Second,
			EnableSyslog:   true,
			EnableMetrics:  true,
			EnableEventLog: true,
		},
	}
}

// Start spins up the HTTP listener
func (s *AgentServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/agent/ingest", s.handleIngest)
	mux.HandleFunc("/api/v1/agent/register", s.handleRegister)
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	s.log.Debug("Configured TLS Cert: %q, Key: %q", s.certFile, s.keyFile)

	// Enable mTLS if CA is provided
	if s.caFile != "" {
		caCert, err := os.ReadFile(s.caFile)
		if err != nil {
			return fmt.Errorf("read CA cert: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	s.server = &http.Server{
		Addr:      addr,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	s.log.Info("Starting Agent Ingest Server on %s (mTLS: %v)", addr, s.caFile != "")

	go func() {
		err := s.server.ListenAndServeTLS(s.certFile, s.keyFile)
		// If certs aren't available yet or user wants HTTP for dev, fallback
		if err != nil && err != http.ErrServerClosed {
			s.log.Warn("TLS setup failed (%v), falling back to unencrypted HTTP on %s", err, addr)
			s.server.TLSConfig = nil
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.log.Error("Agent Server listener failed: %v", err)
			}
		}
	}()

	return nil
}

// Stop halts the HTTP server gracefully
func (s *AgentServer) Stop() {
	s.log.Info("Stopping Agent Ingest Server...")
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}
}

// GetActiveAgents returns a snapshot of recently seen agents
func (s *AgentServer) GetActiveAgents() []AgentInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.log.Info("[AGENT_SERVER] GetActiveAgents called. Total tracked: %d", len(s.activeAgents))

	// Filter out agents not seen in last 5 minutes
	cutoff := time.Now().Add(-5 * time.Minute)
	var active []AgentInfo

	for id, info := range s.activeAgents {
		ts := parseTime(info.LastSeen)
		isAfter := ts.After(cutoff)
		s.log.Debug("[AGENT_SERVER] Checking agent %s: LastSeen=%s, Cutoff=%s, After=%v", id, info.LastSeen, cutoff.Format(time.RFC3339), isAfter)
		if isAfter {
			active = append(active, info)
		}
	}
	s.log.Info("[AGENT_SERVER] Returning %d active agents", len(active))
	return active
}

// SetFleetConfig updates the desired configuration for all agents
func (s *AgentServer) SetFleetConfig(cfg FleetConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.desiredConfig = cfg
	s.log.Info("Fleet configuration updated: interval=%s, fim=%v, syslog=%v, metrics=%v",
		cfg.Interval, cfg.EnableFIM, cfg.EnableSyslog, cfg.EnableMetrics)
}

func (s *AgentServer) getVisitor(ip string) *rate.Limiter {
	limiter, exists := s.limiters.Load(ip)
	if !exists {
		// Allow 50 requests/sec with a burst size of 100
		newLimiter := rate.NewLimiter(rate.Limit(50), 100)
		s.limiters.Store(ip, newLimiter)
		return newLimiter
	}
	return limiter.(*rate.Limiter)
}

func (s *AgentServer) handleIngest(w http.ResponseWriter, r *http.Request) {
	s.log.Info("Received ingestion request from %s", r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Rate Limiting
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}
	limiter := s.getVisitor(ip)
	if !limiter.Allow() {
		s.log.Warn("Rate limit exceeded for agent IP: %s", ip)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	agentVersion := r.Header.Get("X-Agent-Version")
	if agentVersion == "" {
		agentVersion = "unknown"
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "GLOBAL"
	}

	var reader io.Reader = r.Body
	defer r.Body.Close()

	// Extract OpenTelemetry TraceContext if present
	propagator := otel.GetTextMapPropagator()
	if propagator == nil {
		propagator = propagation.TraceContext{}
	}
	ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// Handle zstd/zlib compression
	encoding := r.Header.Get("Content-Encoding")
	if encoding == "zstd" || encoding == "deflate" {
		// Agent uses zlib writer, despite "zstd" terminology in transport.go fallback
		zr, err := zlib.NewReader(r.Body)
		if err != nil {
			s.log.Error("Failed to create zlib reader: %v", err)
			http.Error(w, "Invalid compression", http.StatusBadRequest)
			return
		}
		defer zr.Close()
		reader = zr
	}

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		s.log.Error("Failed to read body: %v", err)
		http.Error(w, "Read error", http.StatusInternalServerError)
		return
	}

	agentID := r.Header.Get("X-Agent-ID")
	signature := r.Header.Get("X-Agent-Signature")

	// 2. Cryptographic Origin Verification (Sovereign Trust)
	if signature != "" && agentID != "" {
		s.mu.RLock()
		info, exists := s.activeAgents[agentID]
		s.mu.RUnlock()

		if exists && len(info.PublicKey) > 0 {
			sigBytes, _ := base64.StdEncoding.DecodeString(signature)
			if !ed25519.Verify(info.PublicKey, bodyBytes, sigBytes) {
				s.log.Warn("[SECURITY] Invalid batch signature from agent %s! Potential spoofing or tampering detected.", agentID)
				
				s.mu.Lock()
				info.TrustLevel = "compromised"
				s.activeAgents[agentID] = info
				s.mu.Unlock()

				http.Error(w, "Cryptographic verification failed", http.StatusUnauthorized)
				return
			}
			
			s.mu.Lock()
			info.TrustLevel = "verified"
			// Check if EBPF collector is active for watchdog
			for _, c := range info.Collectors {
				if c == "ebpf" {
					info.WatchdogActive = true
					break
				}
			}
			s.activeAgents[agentID] = info
			s.mu.Unlock()

			s.log.Debug("[INGEST] Verified signature for agent %s", agentID)
		}
	}

	var incomingEvents []agent.Event
	if err := json.Unmarshal(bodyBytes, &incomingEvents); err != nil {
		s.log.Error("Failed to decode json: %v", err)
		http.Error(w, "Decode error", http.StatusBadRequest)
		return
	}

	// Update agent tracking
	hostname := r.Header.Get("X-Agent-Hostname")

	// If headers are missing, try to infer from events
	if agentID == "" && len(incomingEvents) > 0 {
		agentID = incomingEvents[0].AgentID
	}
	if hostname == "" && len(incomingEvents) > 0 {
		hostname = incomingEvents[0].Host
	}

	if agentID != "" {
		s.mu.Lock()
		_, exists := s.activeAgents[agentID]
		s.activeAgents[agentID] = AgentInfo{
			ID:            agentID,
			Hostname:      hostname,
			TenantID:      tenantID,
			Version:       agentVersion,
			LastSeen:      time.Now().Format(time.RFC3339),
			RemoteAddress: r.RemoteAddr,
		}
		s.mu.Unlock()

		if s.pipeline != nil && s.pipeline.Bus() != nil {
			if !exists {
				// First time we've seen this agent — fire registration event
				s.pipeline.Bus().Publish("agent.registered", map[string]interface{}{
					"host_id":        agentID,
					"hostname":       hostname,
					"remote_address": r.RemoteAddr,
					"version":        agentVersion,
				})
			} else {
				// Subsequent contact — fire heartbeat event
				s.pipeline.Bus().Publish("agent.heartbeat", map[string]interface{}{
					"host_id":  agentID,
					"last_seen": time.Now().Format(time.RFC3339),
				})
			}
		}
	}

	// Route events to the pipeline
	ingested := 0
	for _, ev := range incomingEvents {
		// Schema Validation
		if err := s.validator.Struct(ev); err != nil {
			s.log.Warn("Dropping event from %s due to malformed schema: %v", r.RemoteAddr, err)
			continue // Drop invalid event, but continue processing others in batch
		}

		// Convert agent.Event to ingest.ParsedEvent mapping structure
		rawBytes, _ := json.Marshal(ev.Data)

		user := ""
		if u, ok := ev.Data["user"].(string); ok {
			user = u
		}

		version := ev.Version
		if version == "" {
			version = "v1"
		}

		ingestEv := &events.SovereignEvent{
			Timestamp: ev.Timestamp,
			TenantID:  tenantID,
			Host:      ev.Host,
			SourceIp:  r.RemoteAddr,
			EventType: ev.Type,
			User:      user,
			SessionId: "agent-" + ev.Source,
			RawLine:   string(rawBytes),
			Version:   1, // SovereignEvent uses int32 version
			Ctx:       ctx,
		}

		if err := s.pipeline.QueueEvent(ingestEv); err != nil {
			s.log.Warn("Pipeline backpressure: dropping event from %s: %v", r.RemoteAddr, err)
			continue
		}
		ingested++
	}

	s.log.Info("[INGEST] Successfully ingested %d events for agent %s", ingested, agentID)

	// Return the desired configuration and pending actions in the response
	s.mu.Lock()
	config := s.desiredConfig
	actions := s.pendingActions[agentID]
	delete(s.pendingActions, agentID) // Hand off to agent
	s.mu.Unlock()

	response := struct {
		Config  FleetConfig     `json:"config"`
		Actions []PendingAction `json:"actions"`
	}{
		Config:  config,
		Actions: actions,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// AddAction queues a command for a specific agent
func (s *AgentServer) AddAction(agentID string, action PendingAction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingActions[agentID] = append(s.pendingActions[agentID], action)
	s.log.Info("Queued action %s (%s) for agent %s", action.ID, action.Type, agentID)
}

// handleRegister accepts heartbeat check-ins from the agent and populates the fleet registry
func (s *AgentServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reg struct {
		ID         string   `json:"id"`
		Hostname   string   `json:"hostname"`
		Version    string   `json:"version"`
		OS         string   `json:"os"`
		Arch       string   `json:"arch"`
		Collectors []string `json:"collectors"`
		PublicKey  []byte   `json:"public_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		s.log.Error("Failed to decode registration: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if reg.ID == "" {
		http.Error(w, "Missing Agent ID", http.StatusBadRequest)
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "GLOBAL"
	}

	s.mu.Lock()
	s.activeAgents[reg.ID] = AgentInfo{
		ID:            reg.ID,
		Hostname:      reg.Hostname,
		TenantID:      tenantID,
		Version:       reg.Version,
		OS:            reg.OS,
		Arch:          reg.Arch,
		Collectors:    reg.Collectors,
		PublicKey:     reg.PublicKey,
		TrustLevel:    "unverified",
		WatchdogActive: false,
		LastSeen:      time.Now().Format(time.RFC3339),
		RemoteAddress: r.RemoteAddr,
	}
	s.mu.Unlock()

	s.log.Info("Agent registered successfully: %s (%s)", reg.ID, reg.Hostname)
	
	if s.pipeline != nil && s.pipeline.Bus() != nil {
		s.pipeline.Bus().Publish("agent.registered", map[string]interface{}{
			"host_id":        reg.ID,
			"hostname":       reg.Hostname,
			"remote_address": r.RemoteAddr,
			"version":        reg.Version,
			"os":             reg.OS,
			"arch":           reg.Arch,
		})
	}

	w.WriteHeader(http.StatusOK)
}
