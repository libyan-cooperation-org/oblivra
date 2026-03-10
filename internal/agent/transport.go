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
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// Transport handles secure communication with the OBLIVRA server.
// It supports mTLS, retry with backoff, and WAL-based offline buffering.
type Transport struct {
	cfg    Config
	client *http.Client
	log    *logger.Logger
}

// NewTransport creates a new transport with optional mTLS.
func NewTransport(cfg Config, log *logger.Logger) (*Transport, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	// Load client certificate for mTLS
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate for server verification
	if cfg.TLSCA != "" {
		caCert, err := os.ReadFile(cfg.TLSCA)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("invalid CA cert")
		}
		tlsConfig.RootCAs = pool
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &Transport{
		cfg:    cfg,
		client: client,
		log:    log,
	}, nil
}

// Send transmits a batch of events to the server.
// It returns a pointer to a FleetConfig if the server provides a configuration update.
func (t *Transport) Send(events []Event) (*FleetConfig, error) {
	if len(events) == 0 {
		return nil, nil
	}

	data, err := json.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
	}

	// Zstd compression
	var body []byte
	contentEncoding := "identity"
	compressed := zstdCompress(data)
	if compressed != nil && len(compressed) < len(data) {
		body = compressed
		contentEncoding = "zstd"
	} else {
		body = data
	}

	url := fmt.Sprintf("https://%s/api/v1/agent/ingest", t.cfg.ServerAddr)
	req, err := http.NewRequest(http.MethodPost, url, bytesReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", contentEncoding)
	req.Header.Set("X-Agent-Version", t.cfg.Version)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	// Parse configuration from response
	var cfg FleetConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		// If decoding fails, it might be an empty body, which is fine
		return nil, nil
	}

	return &cfg, nil
}

// FlushLoop continuously drains the WAL and sends events to the server.
func (t *Transport) FlushLoop(ctx context.Context, wal *WAL, onConfig func(FleetConfig)) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	backoff := time.Second
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			// Final flush attempt
			t.flushOnce(wal, onConfig)
			return
		case <-ticker.C:
			if err := t.flushOnce(wal, onConfig); err != nil {
				t.log.Error("Flush error (retry in %s): %v", backoff, err)
				backoff = min(backoff*2, maxBackoff)
				ticker.Reset(backoff)
			} else {
				backoff = time.Second
				ticker.Reset(5 * time.Second)
			}
		}
	}
}

func (t *Transport) flushOnce(wal *WAL, onConfig func(FleetConfig)) error {
	events, err := wal.ReadAll()
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	cfg, err := t.Send(events)
	if err != nil {
		return err
	}

	// Update agent config if provided
	if cfg != nil && onConfig != nil {
		onConfig(*cfg)
	}

	return wal.Truncate()
}

// Close closes the transport.
func (t *Transport) Close() {
	t.client.CloseIdleConnections()
}

// ─── helpers ──────────────────────────────────────────

func bytesReader(data []byte) io.Reader {
	return bytes.NewReader(data)
}

// zstdCompress uses zlib compression as a pure-Go cross-platform fallback.
// For production, swap with github.com/klauspost/compress/zstd for real Zstd.
func zstdCompress(data []byte) []byte {
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
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
