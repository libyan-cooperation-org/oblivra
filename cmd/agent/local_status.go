package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// Local status endpoint — a tiny HTTP server bound to 127.0.0.1 only.
// Lets `oblivra-agent status` (and ad-hoc `curl`) query the running
// agent's queue depth, spill bytes, signing-key fingerprint, etc.
// without ssh-then-tail-the-logs.
//
// Bind is hardcoded to loopback. The agent never opens a remotely-
// reachable port — all telemetry leaves via the platform's heartbeat,
// not via any port on this host.
//
// Optional: disabled when cfg.LocalStatusAddr is empty (the YAML knob
// is `localStatusAddr`). Default 127.0.0.1:18021 when not set.

type localStatusServer struct {
	cfg      *Config
	signer   *Signer
	queue    chan string
	hiQueue  chan string
	startedAt time.Time

	srv *http.Server
}

func startLocalStatus(ctx context.Context, cfg *Config, signer *Signer, queue, hiQueue chan string) (*localStatusServer, error) {
	addr := "127.0.0.1:18021"
	if cfg.LocalStatusAddr != "" {
		addr = cfg.LocalStatusAddr
	}
	if addr == "off" {
		return nil, nil // explicitly disabled
	}

	// Hard bind: only loopback addresses are allowed. Reject anything
	// else so an operator who put `0.0.0.0:18021` in the config
	// doesn't accidentally publish queue depth on the public NIC.
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("local-status: invalid addr %q: %w", addr, err)
	}
	if !isLoopbackHost(host) {
		return nil, fmt.Errorf("local-status: addr must be loopback (got %q)", host)
	}

	s := &localStatusServer{
		cfg:       cfg,
		signer:    signer,
		queue:     queue,
		hiQueue:   hiQueue,
		startedAt: time.Now().UTC(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	s.srv = &http.Server{Addr: addr, Handler: mux}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("local-status: listen %s: %w", addr, err)
	}
	go func() {
		_ = s.srv.Serve(ln)
	}()
	go func() {
		<-ctx.Done()
		_ = s.srv.Shutdown(context.Background())
	}()
	return s, nil
}

// requireLocalAuth gates a status request when an admin password is
// configured. The header `X-Admin-Password: <plaintext>` is the
// canonical channel; the request is loopback-only by socket bind so
// the credential never traverses an untrusted hop.
func (s *localStatusServer) requireLocalAuth(w http.ResponseWriter, r *http.Request) bool {
	if !HasAdminPassword(s.cfg.StateDir) {
		return true
	}
	pw := r.Header.Get("X-Admin-Password")
	if pw == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, `{"error":"X-Admin-Password header required"}`)
		return false
	}
	if err := VerifyAdminPassword(s.cfg.StateDir, pw); err != nil {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, `{"error":"invalid admin password"}`)
		return false
	}
	return true
}

func (s *localStatusServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !s.requireLocalAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	files, bytesOnDisk := scanSpillDir(s.cfg.Buffer.Dir)
	resp := map[string]any{
		"hostname":          s.cfg.Hostname,
		"tenant":            s.cfg.Tenant,
		"version":           version,
		"goVersion":         runtime.Version(),
		"os":                runtime.GOOS,
		"arch":              runtime.GOARCH,
		"startedAt":         s.startedAt.Format(time.RFC3339),
		"uptime":            time.Since(s.startedAt).Round(time.Second).String(),
		"queueDepth":        len(s.queue),
		"hiQueueDepth":      len(s.hiQueue),
		"droppedEvents":     droppedEvents.Load(),
		"spillFiles":        files,
		"spillBytes":        bytesOnDisk,
		"inputs":            len(s.cfg.Inputs),
		"signEvents":        s.cfg.SignEvents,
		"redact":            s.cfg.Redact,
		"localRules":        s.cfg.LocalRules,
		"adaptiveBatch":     s.cfg.AdaptiveBatch,
		"localStatusAddr":   s.srv.Addr,
	}
	if s.signer != nil {
		resp["pubkeyFingerprint"] = s.signer.FingerprintShort()
		resp["pubkeyB64"] = s.signer.PublicKeyB64()
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func isLoopbackHost(h string) bool {
	if h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// ---- last-gasp signal handler ----

// lastGaspMarker is a global atomic so the signal-handler goroutine
// can flip it from "running normally" to "shutdown initiated" and the
// forwarder loop can fast-path a final ed25519-signed event before
// the process exits.
var lastGaspMarker atomic.Bool

// buildLastGaspEvent assembles the JSON for the final shutdown event
// pushed at SIGTERM/SIGINT. Same shape as the regular agent ingest
// JSON so the platform processes it through the standard path.
//
// We sign the event when a signer is available — that's the whole
// point of last-gasp: an attacker who silently kills the agent (e.g.
// systemctl stop oblivra-agent followed by tampering) leaves an
// ed25519-signed exit marker in the chain, and the absence of further
// heartbeats is itself an alert under Phase 44's missing-anchor logic.
func buildLastGaspEvent(cfg *Config, signer *Signer) string {
	doc := map[string]any{
		"source":    "agent",
		"tenantId":  cfg.Tenant,
		"hostId":    cfg.Hostname,
		"eventType": "agent.shutdown",
		"severity":  "warning",
		"message":   "oblivra-agent received termination signal — last-gasp event before exit",
		"fields": map[string]string{
			"agentSource": "self",
			"agentInput":  "last-gasp",
			"signal":      "received",
			"shutdownAt":  time.Now().UTC().Format(time.RFC3339Nano),
		},
	}
	body, _ := json.Marshal(doc)
	out := string(body)
	if signer != nil {
		if signed, err := signer.SignEvent(out); err == nil {
			out = signed
		}
	}
	return out
}
