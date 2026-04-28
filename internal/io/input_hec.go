package io

// HEC (HTTP Event Collector) input — POST /event with JSON body.
//
// Splunk's HEC is the universal "anything can curl into the platform"
// endpoint. We mirror its core surface so existing scripts written
// against Splunk HEC work against OBLIVRA without modification.
//
// Endpoints:
//   POST /event       — single event JSON
//   POST /event/batch — array of events
//
// Auth: optional `token` config; when set, callers send
//   Authorization: Splunk <token>
// or
//   Authorization: Bearer <token>
//
// Both header forms are accepted; matches HEC + bearer convention.
//
// Config:
//
//   - id: hec
//     type: hec
//     listen: "0.0.0.0:8088"        # required
//     token: "${env.OBLIVRA_HEC_TOKEN}"   # optional but strongly recommended
//     sourcetype: "hec:default"     # default sourcetype if event doesn't supply one
//     tls_cert: "/etc/oblivra/cert.pem"  # optional; off → plaintext (guardrails apply)
//     tls_key:  "/etc/oblivra/key.pem"

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

type hecInputConfig struct {
	Listen     string `yaml:"listen"`
	Token      string `yaml:"token"`
	Sourcetype string `yaml:"sourcetype"`
	TLSCert    string `yaml:"tls_cert"`
	TLSKey     string `yaml:"tls_key"`
}

type HECInput struct {
	id  string
	cfg hecInputConfig
	log *logger.Logger

	srv *http.Server
	ln  net.Listener

	// Reference to the channel from Start so the http handler can
	// publish without re-injecting on every request.
	out    chan<- Event
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewHECInputReal(id string, raw map[string]interface{}, log *logger.Logger) (*HECInput, error) {
	cfg, err := decodeYAMLMap[hecInputConfig](raw)
	if err != nil {
		return nil, fmt.Errorf("input hec %q: %w", id, err)
	}
	if cfg.Listen == "" {
		return nil, fmt.Errorf("input hec %q: listen is required", id)
	}
	cfg.Token = expandEnv(cfg.Token)
	if cfg.Token == "" {
		// Fallback env var — same convention as oblivra output.
		cfg.Token = os.Getenv("OBLIVRA_HEC_TOKEN")
	}
	if cfg.Sourcetype == "" {
		cfg.Sourcetype = "hec:default"
	}
	return &HECInput{
		id:  id,
		cfg: cfg,
		log: log.WithPrefix("input.hec"),
	}, nil
}

func (h *HECInput) Name() string { return h.id }
func (h *HECInput) Type() string { return "hec" }

func (h *HECInput) Start(ctx context.Context, out chan<- Event) error {
	pluginCtx, cancel := context.WithCancel(ctx)
	h.cancel = cancel
	h.out = out

	mux := http.NewServeMux()
	mux.HandleFunc("/event", h.handleSingle)
	mux.HandleFunc("/event/batch", h.handleBatch)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	h.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", h.cfg.Listen)
	if err != nil {
		return fmt.Errorf("hec %q: listen %s: %w", h.id, h.cfg.Listen, err)
	}
	h.ln = ln

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		var serveErr error
		if h.cfg.TLSCert != "" && h.cfg.TLSKey != "" {
			serveErr = h.srv.ServeTLS(ln, h.cfg.TLSCert, h.cfg.TLSKey)
			h.log.Info("[%s] listening https %s", h.id, h.cfg.Listen)
		} else {
			h.log.Info("[%s] listening http %s (plaintext — TLS guardrails apply)", h.id, h.cfg.Listen)
			serveErr = h.srv.Serve(ln)
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			h.log.Warn("[%s] serve: %v", h.id, serveErr)
		}
	}()

	// Watch ctx for shutdown.
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		<-pluginCtx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = h.srv.Shutdown(shutCtx)
	}()
	return nil
}

func (h *HECInput) Stop() error {
	if h.cancel != nil {
		h.cancel()
	}
	done := make(chan struct{})
	go func() { h.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		h.log.Warn("[%s] stop timed out", h.id)
	}
	return nil
}

// authOK validates the Authorization header against the configured
// token. When no token is configured we accept any caller — operator
// has explicitly opted in to "open HEC endpoint" by leaving token unset.
func (h *HECInput) authOK(r *http.Request) bool {
	if h.cfg.Token == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	for _, prefix := range []string{"Splunk ", "Bearer "} {
		if strings.HasPrefix(auth, prefix) {
			tok := strings.TrimPrefix(auth, prefix)
			// Constant-time compare to avoid timing leaks.
			if subtle.ConstantTimeCompare([]byte(tok), []byte(h.cfg.Token)) == 1 {
				return true
			}
		}
	}
	return false
}

// hecPayload is the canonical Splunk-HEC event shape. Everything is
// optional except the event itself. Field names match Splunk's so
// existing scripts work unchanged.
type hecPayload struct {
	Event      json.RawMessage        `json:"event"`
	Time       any                    `json:"time"` // unix-secs (number or string)
	Host       string                 `json:"host"`
	Source     string                 `json:"source"`
	Sourcetype string                 `json:"sourcetype"`
	Index      string                 `json:"index"`
	Fields     map[string]interface{} `json:"fields"`
}

func (h *HECInput) handleSingle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.publishOne(r, body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _ = w.Write([]byte(`{"text":"Success","code":0}`))
}

func (h *HECInput) handleBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20) // 16 MiB
	dec := json.NewDecoder(r.Body)
	count := 0
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			http.Error(w, "decode: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.publishOne(r, raw); err != nil {
			h.log.Warn("[%s] batch item: %v", h.id, err)
			// Continue processing the rest of the batch — partial
			// success is better than rejecting 1000 events because
			// one is malformed.
		}
		count++
	}
	_, _ = fmt.Fprintf(w, `{"text":"Success","code":0,"count":%d}`, count)
}

func (h *HECInput) publishOne(r *http.Request, body []byte) error {
	var p hecPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return fmt.Errorf("invalid JSON")
	}

	// Resolve timestamp — Splunk's `time` can be unix secs (number)
	// or a string with sub-second precision. Default to now.
	ts := time.Now().UTC()
	if p.Time != nil {
		switch v := p.Time.(type) {
		case float64:
			ts = time.Unix(int64(v), int64((v-float64(int64(v)))*1e9)).UTC()
		case string:
			if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
				ts = t
			} else if f, err := time.Parse(time.RFC3339, v); err == nil {
				ts = f
			}
		}
	}

	host := p.Host
	if host == "" {
		host = r.RemoteAddr
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
	}
	st := p.Sourcetype
	if st == "" {
		st = h.cfg.Sourcetype
	}

	// Event payload can be a string OR a JSON object. We carry the
	// raw bytes verbatim in `raw` so neither shape is lossy.
	raw := string(p.Event)
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		// String — unwrap the JSON quotes for the raw line.
		var s string
		_ = json.Unmarshal(p.Event, &s)
		raw = s
	}

	ev := Event{
		Timestamp:  ts,
		Source:     "hec:" + p.Source,
		Sourcetype: st,
		Host:       host,
		Raw:        raw,
		Fields:     map[string]any{},
		InputID:    h.id,
	}
	for k, v := range p.Fields {
		ev.Fields[k] = v
	}
	if p.Index != "" {
		ev.Fields["index"] = p.Index
	}

	select {
	case h.out <- ev:
	default:
		// Pipeline channel is full — drop and log. This is back-
		// pressure for the slowest output; HEC clients are best
		// served by retrying.
		return fmt.Errorf("pipeline full")
	}
	return nil
}
