package agent

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
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
	privKey  ed25519.PrivateKey // 1.4: Sovereign identity key
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
		"public_key":  t.privKey.Public().(ed25519.PublicKey),
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
	t.setHeaders(req, "identity", data)
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

// SendResult is the outcome of one Send call.
type SendResult struct {
	Config    FleetConfig
	Actions   []PendingAction
	// AckedSeq is the highest event sequence number the server confirms it
	// has durably ingested. The agent uses this to drive WAL.TruncateUpTo
	// so events that arrived during the flush race are not lost.
	AckedSeq  uint64
}

// Send transmits a batch of events to the server.
// Returns updated FleetConfig, PendingActions, and the server's
// acked-sequence watermark for partial-truncate.
func (t *Transport) Send(events []Event) (*SendResult, error) {
	if events == nil {
		events = []Event{}
	}

	data, err := json.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
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
		return nil, fmt.Errorf("create request: %w", err)
	}
	t.setHeaders(req, contentEncoding, body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}
	defer resp.Body.Close()

	// Read up to 1 MB of response body (avoid surprises)
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var response struct {
		Config   FleetConfig     `json:"config"`
		Actions  []PendingAction `json:"actions"`
		AckedSeq uint64          `json:"acked_seq"`
	}
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &response)
	}

	// Backward-compatibility: if the server doesn't return an ack (older
	// builds), fall back to the highest seq we sent. This keeps fresh
	// agents working against legacy servers but provides no idempotency.
	acked := response.AckedSeq
	if acked == 0 && len(events) > 0 {
		for _, ev := range events {
			if ev.Seq > acked {
				acked = ev.Seq
			}
		}
	}

	return &SendResult{
		Config:   response.Config,
		Actions:  response.Actions,
		AckedSeq: acked,
	}, nil
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
	all, err := wal.ReadAll()
	if err != nil {
		return fmt.Errorf("WAL read: %w", err)
	}

	// Filter out events the server has already acknowledged on a previous
	// flush. Without this filter, a successful flush followed by a crash
	// before TruncateUpTo would resend the entire WAL on restart; the
	// server would dedupe it but the round-trip is wasted.
	_, lastAcked := wal.Cursor().Snapshot()
	var events []Event
	for _, ev := range all {
		if ev.Seq > lastAcked {
			events = append(events, ev)
		}
	}

	// Cap the batch size; the rest will be picked up on the next tick.
	if len(events) > maxBatch {
		events = events[:maxBatch]
	}

	result, err := t.Send(events)
	if err != nil {
		return err
	}

	// Persist the new ack watermark before touching disk. If we crash
	// between MarkAcked and TruncateUpTo, the next flush re-reads the WAL
	// and the cursor filter above skips the already-acked prefix.
	if result.AckedSeq > 0 {
		if err := wal.Cursor().MarkAcked(result.AckedSeq); err != nil {
			t.log.Warn("[transport] cursor ack persist failed: %v", err)
		}
		if err := wal.TruncateUpTo(result.AckedSeq); err != nil {
			t.log.Warn("[transport] WAL truncate-up-to failed: %v", err)
		}
	}

	if onConfig != nil {
		onConfig(result.Config, result.Actions)
	}

	if len(events) > 0 {
		t.log.Info("[transport] Flushed %d events to server (acked_seq=%d)", len(events), result.AckedSeq)
	}
	return nil
}

// Close cleans up idle connections.
func (t *Transport) Close() {
	t.client.CloseIdleConnections()
}
// SetIdentityKey sets the key used for batch signing.
func (t *Transport) SetIdentityKey(key ed25519.PrivateKey) {
	t.privKey = key
}

// setHeaders applies standard agent identification headers and cryptographic signatures.
//
// Two layers of authentication go on every request:
//
//   1. HMAC fleet auth — `X-Timestamp` + `X-Signature`, validated by
//      `internal/api/middleware.go:VerifyHMAC` against the shared
//      `fleetSecret`. Proves the request came from a registered
//      agent that knows the deployment-wide secret.
//
//   2. Ed25519 batch signature — `X-Agent-Signature`, validated in
//      `agent_handlers.go` against the agent's registered public key.
//      Proves the BATCH bytes haven't been tampered with in flight.
//
// Both must be present or the server rejects with 401. This was the
// root cause of the "server returned 401" failure log seen on first
// agent boot — the agent was sending only #2 and the server was
// looking for #1 too.
func (t *Transport) setHeaders(req *http.Request, contentEncoding string, body []byte) {
	req.Header.Set("X-Agent-ID", t.cfg.AgentID)
	req.Header.Set("X-Agent-Version", t.cfg.Version)
	req.Header.Set("X-Agent-Hostname", t.hostname)
	req.Header.Set("X-Tenant-ID", t.cfg.TenantID)
	req.Header.Set("Content-Encoding", contentEncoding)

	// 1. HMAC fleet auth — required by every agent endpoint.
	// `secret` defaults to the dev value when no secret is configured;
	// production deployments MUST override via `--fleet-secret` or
	// the env var to a per-deployment 32+ byte random value.
	secret := t.cfg.FleetSecret
	if len(secret) == 0 {
		secret = []byte("oblivra-fleet-secret-v1") // matches api_service.go default
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	mac.Write([]byte(ts))
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))

	// 2. Ed25519 sovereign batch signature — proves byte-for-byte
	// integrity of the payload independently of the HMAC envelope.
	if t.privKey != nil && len(body) > 0 {
		sig := ed25519.Sign(t.privKey, body)
		req.Header.Set("X-Agent-Signature", base64.StdEncoding.EncodeToString(sig))
	}
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
