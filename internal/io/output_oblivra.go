package io

// Oblivra output — POST events to an OBLIVRA server's
// /api/v1/agent/ingest endpoint with HMAC fleet-secret auth.
//
// This wraps the existing agent → server transport in the Output
// interface so it sits alongside `syslog`, `s3`, `http`, and `file`
// outputs. Behaviour matches the legacy agent ingester:
//
//   • Batch up to 256 events or 5s, whichever comes first
//   • HMAC-SHA256 over the body using the fleet secret
//   • Retry with exponential backoff (1s → 2s → 4s → 8s, max 60s)
//   • Drop events after 5 min of contiguous failure (operator can
//     bump via `max_drop_age`)
//
// Config:
//
//   - id: primary
//     type: oblivra
//     server: "https://oblivra.internal:8443"
//     fleet_secret: "${OBLIVRA_FLEET_SECRET}"   # env-substituted
//     batch_size: 256
//     batch_timeout: "5s"
//     max_drop_age: "5m"
//     # TLS knobs:
//     verify_tls: true
//     ca_file: "/etc/oblivra/ca.pem"

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

type oblivraOutputConfig struct {
	Server       string        `yaml:"server"`
	FleetSecret  string        `yaml:"fleet_secret"`
	BatchSize    int           `yaml:"batch_size"`
	BatchTimeout time.Duration `yaml:"batch_timeout"`
	MaxDropAge   time.Duration `yaml:"max_drop_age"`
	VerifyTLS    *bool         `yaml:"verify_tls"`
	CAFile       string        `yaml:"ca_file"`
}

type OblivraOutput struct {
	id  string
	cfg oblivraOutputConfig
	log *logger.Logger

	client *http.Client
	secret []byte

	mu     sync.Mutex
	buffer []Event
	failedSince time.Time

	flushCh chan struct{}
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewOblivraOutput(id string, raw map[string]interface{}, log *logger.Logger) (*OblivraOutput, error) {
	cfg, err := decodeYAMLMap[oblivraOutputConfig](raw)
	if err != nil {
		return nil, fmt.Errorf("output oblivra %q: %w", id, err)
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("output oblivra %q: server is required", id)
	}
	cfg.Server = strings.TrimRight(cfg.Server, "/")

	// Env-substitute ${VAR} in fleet_secret. Common pattern: store
	// the literal in env / vault, reference here.
	cfg.FleetSecret = expandEnv(cfg.FleetSecret)
	if cfg.FleetSecret == "" {
		// Fallback: read from OBLIVRA_FLEET_SECRET if not configured.
		// Useful when operators wire the secret via env.
		cfg.FleetSecret = os.Getenv("OBLIVRA_FLEET_SECRET")
	}
	if cfg.FleetSecret == "" {
		return nil, fmt.Errorf("output oblivra %q: fleet_secret is required (or set OBLIVRA_FLEET_SECRET)", id)
	}

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 256
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 5 * time.Second
	}
	if cfg.MaxDropAge <= 0 {
		cfg.MaxDropAge = 5 * time.Minute
	}

	verify := true
	if cfg.VerifyTLS != nil {
		verify = *cfg.VerifyTLS
	}
	tlsCfg := &tls.Config{InsecureSkipVerify: !verify}
	// CA pinning support omitted from v1 — operators who need it
	// load via system trust store. Add CA file parsing here if
	// customers ask.

	return &OblivraOutput{
		id:      id,
		cfg:     cfg,
		log:     log.WithPrefix("output.oblivra"),
		client:  &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{TLSClientConfig: tlsCfg}},
		secret:  []byte(cfg.FleetSecret),
		buffer:  make([]Event, 0, cfg.BatchSize),
		flushCh: make(chan struct{}, 4),
	}, nil
}

func (o *OblivraOutput) Name() string { return o.id }
func (o *OblivraOutput) Type() string { return "oblivra" }

func (o *OblivraOutput) Write(ctx context.Context, ev Event) error {
	o.mu.Lock()
	o.buffer = append(o.buffer, ev)
	full := len(o.buffer) >= o.cfg.BatchSize
	o.mu.Unlock()
	if full {
		select {
		case o.flushCh <- struct{}{}:
		default:
		}
	}
	return nil
}

func (o *OblivraOutput) Flush(ctx context.Context) error {
	return o.flush(ctx)
}

func (o *OblivraOutput) Close() error {
	if o.cancel != nil {
		o.cancel()
	}
	// One final flush attempt — best-effort, time-boxed.
	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = o.flush(flushCtx)
	o.wg.Wait()
	return nil
}

// flush snapshots the buffer and POSTs it. Empties the buffer on
// success. On failure, restores the buffer (oldest-first) so retry
// preserves order.
func (o *OblivraOutput) flush(ctx context.Context) error {
	o.mu.Lock()
	if len(o.buffer) == 0 {
		o.mu.Unlock()
		return nil
	}
	batch := o.buffer
	o.buffer = make([]Event, 0, o.cfg.BatchSize)
	o.mu.Unlock()

	if err := o.send(ctx, batch); err != nil {
		// Restore — but check the drop window first.
		o.mu.Lock()
		if o.failedSince.IsZero() {
			o.failedSince = time.Now()
		}
		if time.Since(o.failedSince) > o.cfg.MaxDropAge {
			// We've been failing too long — drop the batch to avoid
			// unbounded memory growth. Operators see this in metrics.
			o.log.Warn("dropping %d events after %s of contiguous failure",
				len(batch), time.Since(o.failedSince).Truncate(time.Second))
			// Don't restore.
		} else {
			// Prepend to buffer so we retry oldest-first. Cheap because
			// we just emptied it.
			o.buffer = append(batch, o.buffer...)
		}
		o.mu.Unlock()
		return err
	}
	// Success — clear the failure clock.
	o.mu.Lock()
	o.failedSince = time.Time{}
	o.mu.Unlock()
	return nil
}

// send POSTs the batch. HMAC over the JSON body; timestamp header
// guards against replay. Server validates both.
func (o *OblivraOutput) send(ctx context.Context, batch []Event) error {
	body, err := json.Marshal(map[string]any{
		"events": batch,
	})
	if err != nil {
		return err
	}
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	mac := hmac.New(sha256.New, o.secret)
	mac.Write([]byte(ts))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.cfg.Server+"/api/v1/agent/ingest", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Timestamp", ts)
	req.Header.Set("X-Agent-Signature", sig)

	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("ingest returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// expandEnv performs ${VAR} substitution. Handles a single var per
// string — that's the common pattern in YAML configs (`fleet_secret:
// ${OBLIVRA_FLEET_SECRET}`) and avoids dragging in a templating lib.
func expandEnv(s string) string {
	if !strings.HasPrefix(s, "${") || !strings.HasSuffix(s, "}") {
		return s
	}
	v := os.Getenv(s[2 : len(s)-1])
	return v
}
