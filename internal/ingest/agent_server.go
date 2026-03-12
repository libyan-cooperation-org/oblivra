package ingest

import (
	"compress/zlib"
	"context"
	"crypto/tls"
	"crypto/x509"
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
	"github.com/kingknull/oblivrashell/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/time/rate"
)

// AgentServer listens for incoming telemetry from deployed Oblivra agents
// It handles mTLS, payload decompression (zlib), and JSON decoding into the ingest pipeline
type AgentServer struct {
	pipeline  *Pipeline
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
	activeAgents  map[string]AgentInfo
	desiredConfig FleetConfig
}

// FleetConfig is the subset of agent configuration that can be pushed from the server.
type FleetConfig struct {
	Interval       time.Duration `json:"interval"`
	EnableFIM      bool          `json:"enable_fim"`
	EnableSyslog   bool          `json:"enable_syslog"`
	EnableMetrics  bool          `json:"enable_metrics"`
	EnableEventLog bool          `json:"enable_event_log"`
}

// AgentInfo stores metadata about connected agents
type AgentInfo struct {
	ID            string    `json:"id"`
	Hostname      string    `json:"hostname"`
	Version       string    `json:"version"`
	LastSeen      string    `json:"last_seen"`
	RemoteAddress string    `json:"remote_address"`
}

// NewAgentServer creates a new agent ingestion server
func NewAgentServer(pipeline *Pipeline, port int, certFile, keyFile, caFile string, log *logger.Logger) *AgentServer {
	return &AgentServer{
		pipeline:     pipeline,
		port:         port,
		log:          log.WithPrefix("agent_server"),
		validator:    validator.New(),
		certFile:     certFile,
		keyFile:      keyFile,
		caFile:       caFile,
		activeAgents: make(map[string]AgentInfo),
	}
}

// Start spins up the HTTP listener
func (s *AgentServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/agent/ingest", s.handleIngest)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

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

	// Filter out agents not seen in last 5 minutes
	cutoff := time.Now().Add(-5 * time.Minute)
	var active []AgentInfo

	for _, info := range s.activeAgents {
		if parseTime(info.LastSeen).After(cutoff) {
			active = append(active, info)
		}
	}
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
	if encoding == "zstd" {
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

	var events []agent.Event
	if err := json.Unmarshal(bodyBytes, &events); err != nil {
		s.log.Error("Failed to decode json: %v", err)
		http.Error(w, "Decode error", http.StatusBadRequest)
		return
	}

	if len(events) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Update agent tracking based on the first event's host
	host := events[0].Host
	if host != "" {
		s.mu.Lock()
		_, isNew := s.activeAgents[host]
		s.activeAgents[host] = AgentInfo{
			ID:            host,
			Hostname:      host,
			Version:       agentVersion,
			LastSeen:      time.Now().Format(time.RFC3339),
			RemoteAddress: r.RemoteAddr,
		}
		s.mu.Unlock()

		if s.pipeline != nil && s.pipeline.Bus() != nil {
			if !isNew {
				// First time we've seen this agent — fire registration event
				s.pipeline.Bus().Publish("agent.registered", map[string]interface{}{
					"host_id":        host,
					"remote_address": r.RemoteAddr,
					"version":        agentVersion,
				})
			} else {
				// Subsequent contact — fire heartbeat event
				s.pipeline.Bus().Publish("agent.heartbeat", map[string]interface{}{
					"host_id":  host,
					"last_seen": time.Now().Format(time.RFC3339),
				})
			}
		}
	}

	// Route events to the pipeline
	ingested := 0
	for _, ev := range events {
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

		ingestEv := ParsedEvent{
			Timestamp: ev.Timestamp,
			Host:      ev.Host,
			SourceIP:  r.RemoteAddr,
			EventType: ev.Type,
			User:      user,
			SessionID: "agent-" + ev.Source,
			RawLine:   string(rawBytes),
			Version:   version,
			Ctx:       ctx,
		}

		if err := s.pipeline.QueueEvent(ingestEv); err != nil {
			s.log.Warn("Pipeline backpressure: dropping event from %s: %v", r.RemoteAddr, err)
			continue
		}
		ingested++
	}

	s.log.Debug("Ingested %d events from agent %s", ingested, host)

	// Return the desired configuration in the response
	s.mu.RLock()
	config := s.desiredConfig
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}
