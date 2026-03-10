package ingest

import (
	"compress/zlib"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/agent"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// AgentServer listens for incoming telemetry from deployed Oblivra agents
// It handles mTLS, payload decompression (zlib), and JSON decoding into the ingest pipeline
type AgentServer struct {
	pipeline *Pipeline
	port     int
	server   *http.Server
	log      *logger.Logger
	mu       sync.RWMutex

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
	LastSeen      time.Time `json:"last_seen"`
	RemoteAddress string    `json:"remote_address"`
}

// NewAgentServer creates a new agent ingestion server
func NewAgentServer(pipeline *Pipeline, port int, certFile, keyFile, caFile string, log *logger.Logger) *AgentServer {
	return &AgentServer{
		pipeline:     pipeline,
		port:         port,
		log:          log.WithPrefix("agent_server"),
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
		if info.LastSeen.After(cutoff) {
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

func (s *AgentServer) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentVersion := r.Header.Get("X-Agent-Version")
	if agentVersion == "" {
		agentVersion = "unknown"
	}

	var reader io.Reader = r.Body
	defer r.Body.Close()

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
		s.activeAgents[host] = AgentInfo{
			ID:            host,
			Hostname:      host,
			Version:       agentVersion,
			LastSeen:      time.Now(),
			RemoteAddress: r.RemoteAddr,
		}
		s.mu.Unlock()
	}

	// Route events to the pipeline
	ingested := 0
	for _, ev := range events {
		// Convert agent.Event to ingest.ParsedEvent mapping structure
		rawBytes, _ := json.Marshal(ev.Data)

		user := ""
		if u, ok := ev.Data["user"].(string); ok {
			user = u
		}

		ingestEv := ParsedEvent{
			Timestamp: ev.Timestamp,
			Host:      ev.Host,
			SourceIP:  r.RemoteAddr,
			EventType: ev.Type,
			User:      user,
			SessionID: "agent-" + ev.Source,
			RawLine:   string(rawBytes),
		}

		s.pipeline.QueueEvent(ingestEv)
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
