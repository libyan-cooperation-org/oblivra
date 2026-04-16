package agent

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// Transport handles secure communication with the OBLIVRA server.
// Features:
//   - mTLS with pinned CA
//   - Exponential backoff on failure (1s → 5min)
//   - Zlib compression (Content-Encoding: deflate)
//   - Agent registration / heartbeat via dedicated endpoint
//   - Bounded batch size to prevent oversized payloads
type Transport struct {
	cfg      Config
	client   *http.Client
	hostname string
	log      *logger.Logger
}

// NewTransport creates a new transport with optional mTLS.
func NewTransport(cfg Config, log *logger.Logger) (*Transport, error) {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: cfg.InsecureTLS,
	}

	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if cfg.TLSCA != "" {
		caCert, err := os.ReadFile(cfg.TLSCA)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("invalid CA cert PEM")
		}
		tlsConfig.RootCAs = pool
	}

	hostname, _ := os.Hostname()

	return &Transport{
		cfg:      cfg,
		hostname: hostname,
		log:      log,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig:     tlsConfig,
				MaxIdleConns:        4,
				MaxIdleConnsPerHost: 4,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}, nil
}

// Register sends an agent registration heartbeat to the server.
// The server uses this to populate the fleet registry.
func (t *Transport) Register(ctx context.Context, collectors []string) error {
	payload := map[string]interface{}{
		"id":          t.cfg.AgentID,
		"hostname":    t.hostname,
		"version":     t.cfg.Version,
		"os":          goOS(),
		"arch":        goArch(),
		"collectors":  collectors,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	base := t.normalizeURL(t.cfg.ServerAddr)
	url := fmt.Sprintf("%s/api/v1/agent/register", base)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	t.setHeaders(req, "identity")
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("register: server returned %d", resp.StatusCode)
	}
	t.log.Info("[transport] Registered with server as %s", t.cfg.AgentID)
	return nil
}

// Send transmits a batch of events to the server.
// Returns updated FleetConfig and PendingActions if the server sends them.
func (t *Transport) Send(events []Event) (*FleetConfig, []PendingAction, error) {
	if events == nil {
		events = []Event{}
	}

	data, err := json.Marshal(events)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal events: %w", err)
	}

	// Zlib (deflate) compression — honest Content-Encoding header
	var body []byte
	contentEncoding := "identity"
	if compressed := zlibCompress(data); compressed != nil && len(compressed) < len(data) {
		body = compressed
		contentEncoding = "deflate"
	} else {
		body = data
	}

	base := t.normalizeURL(t.cfg.ServerAddr)
	url := fmt.Sprintf("%s/api/v1/agent/ingest", base)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	t.setHeaders(req, contentEncoding)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("send: %w", err)
	}
	defer resp.Body.Close()

	// Read up to 1 MB of response body (avoid surprises)
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var response struct {
		Config  FleetConfig     `json:"config"`
		Actions []PendingAction `json:"actions"`
	}
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &response)
	}

	return &response.Config, response.Actions, nil
}

// FlushLoop continuously drains the WAL and sends events to the server.
// maxBatch caps the number of events per HTTP POST.
func (t *Transport) FlushLoop(ctx context.Context, wal *WAL, maxBatch int, onConfig func(FleetConfig, []PendingAction)) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	backoff := time.Second
	const maxBackoff = 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			// One final flush — best effort
			_ = t.flushOnce(wal, maxBatch, onConfig)
			return
		case <-ticker.C:
			if err := t.flushOnce(wal, maxBatch, onConfig); err != nil {
				t.log.Warn("[transport] Flush error (retry in %s): %v", backoff, err)
				ticker.Reset(backoff)
				if backoff < maxBackoff {
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}
			} else {
				backoff = time.Second
				ticker.Reset(5 * time.Second)
			}
		}
	}
}

func (t *Transport) flushOnce(wal *WAL, maxBatch int, onConfig func(FleetConfig, []PendingAction)) error {
	events, err := wal.ReadAll()
	if err != nil {
		return fmt.Errorf("WAL read: %w", err)
	}

	// Always send at least a heartbeat (empty batch)
	if len(events) > maxBatch {
		events = events[:maxBatch]
	}

	cfg, actions, err := t.Send(events)
	if err != nil {
		return err
	}

	// Only truncate after confirmed server acknowledgement
	if err := wal.Truncate(); err != nil {
		t.log.Warn("[transport] WAL truncate failed: %v", err)
	}

	if cfg != nil && onConfig != nil {
		onConfig(*cfg, actions)
	}

	if len(events) > 0 {
		t.log.Info("[transport] Flushed %d events to server", len(events))
	}
	return nil
}

// Close cleans up idle connections.
func (t *Transport) Close() {
	t.client.CloseIdleConnections()
}

// setHeaders applies standard agent identification headers.
func (t *Transport) setHeaders(req *http.Request, contentEncoding string) {
	req.Header.Set("X-Agent-ID", t.cfg.AgentID)
	req.Header.Set("X-Agent-Version", t.cfg.Version)
	req.Header.Set("X-Agent-Hostname", t.hostname)
	req.Header.Set("X-Tenant-ID", t.cfg.TenantID)
	req.Header.Set("Content-Encoding", contentEncoding)
}

func (t *Transport) normalizeURL(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	// Use HTTPS when TLS credentials are configured OR InsecureTLS is requested.
	// We also default to HTTPS if the port is 8443 as a convenience for OBLIVRA.
	if t.cfg.TLSCert != "" || t.cfg.TLSCA != "" || t.cfg.InsecureTLS || strings.HasSuffix(addr, ":8443") {
		return "https://" + addr
	}
	return "http://" + addr
}

// zlibCompress compresses data using zlib (RFC 1950 / deflate).
// Returns nil if compression fails.
func zlibCompress(data []byte) []byte {
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.BestSpeed) // BestSpeed for low latency
	if err != nil {
		return nil
	}
	if _, err := w.Write(data); err != nil {
		return nil
	}
	if err := w.Close(); err != nil {
		return nil
	}
	return buf.Bytes()
}

// goOS / goArch are thin wrappers to avoid importing runtime in transport.
func goOS() string {
	// Determined at compile time via runtime.GOOS
	return runtimeGOOS()
}
func goArch() string {
	return runtimeGOARCH()
}
