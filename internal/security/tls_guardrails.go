// Package security — TLS-mode guardrails (Slice 3 of the I/O framework).
//
// The platform supports running with TLS off for legitimate use cases
// (lab, air-gap with identical-OS-image fleet, already-encrypted
// transport like Wireguard / VPC). The danger is that "off" is a
// single config knob that can silently make it into production. The
// five guardrails here make the cost loud:
//
//   1. Loud boot-time warning every 30s while TLS is off
//   2. UI plaintext banner via REST endpoint /api/v1/tls/state
//   3. Sovereignty score deducts 30 points (handled in
//      api/rest_sovereignty.go via `tls_off`)
//   4. Production lockout: OBLIVRA_PRODUCTION=1 + tls=off → fatal
//   5. Per-event audit hook: callers tag transport=plaintext when
//      ingesting over a non-TLS path
//
// Operators who want TLS off MUST set tls.mode = "off" in YAML AND
// either accept all five guardrails OR also unset OBLIVRA_PRODUCTION.
package security

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// TLSGuardrails owns the runtime state of the "TLS off" warning loop.
// One instance per process; constructed in main.go after config is
// parsed. Safe for concurrent reads via IsTLSOff().
type TLSGuardrails struct {
	off    atomic.Bool
	reason atomic.Value // string

	log    *logger.Logger
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewTLSGuardrails constructs the guardrail. `tlsMode` is the parsed
// config value ("on" or "off" or one of its aliases). Returns an
// error when the production lockout fires (caller MUST refuse to
// continue boot).
func NewTLSGuardrails(tlsMode string, log *logger.Logger) (*TLSGuardrails, error) {
	g := &TLSGuardrails{log: log.WithPrefix("tls")}
	off := isTLSOff(tlsMode)
	g.off.Store(off)

	if off {
		// Guardrail 4: production lockout.
		if isProductionEnv() {
			return nil, errors.New(
				"refusing to start: tls.mode=off is forbidden when OBLIVRA_PRODUCTION=1. " +
					"Either turn TLS on, or unset OBLIVRA_PRODUCTION (NOT recommended in real production)")
		}
		g.reason.Store("config tls.mode=off")
	} else {
		g.reason.Store("")
	}
	return g, nil
}

// Start launches the periodic warning goroutine. Called by the
// container after the rest of the boot succeeds. Idempotent — calling
// Start when TLS is on is a no-op.
func (g *TLSGuardrails) Start(ctx context.Context) {
	if !g.off.Load() {
		return
	}
	pluginCtx, cancel := context.WithCancel(ctx)
	g.cancel = cancel

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		// Loud first warning, immediately. Subsequent warnings every 30s.
		g.warn()
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-pluginCtx.Done():
				return
			case <-t.C:
				g.warn()
			}
		}
	}()
}

// Stop shuts down the warning loop. Safe to call when off=false.
func (g *TLSGuardrails) Stop() {
	if g.cancel != nil {
		g.cancel()
	}
	g.wg.Wait()
}

// IsTLSOff returns the live state. Callers in the audit / sovereignty
// paths read this on every event so a hot-reload that toggles TLS
// flips behaviour immediately.
func (g *TLSGuardrails) IsTLSOff() bool {
	return g.off.Load()
}

// SetMode is the hot-reload entry point. Called by the config-watcher
// when tls.mode changes. Returns true if the production lockout
// would fire — caller decides whether to keep the previous state.
func (g *TLSGuardrails) SetMode(mode string) (locked bool) {
	off := isTLSOff(mode)
	if off && isProductionEnv() {
		g.log.Error("tls.mode=off ignored: OBLIVRA_PRODUCTION=1. Keeping previous mode.")
		return true
	}
	prev := g.off.Swap(off)
	if prev != off {
		if off {
			g.log.Warn("TLS DISABLED at runtime via config reload")
			g.reason.Store("config tls.mode=off (hot-reloaded)")
		} else {
			g.log.Info("TLS RE-ENABLED at runtime via config reload")
			g.reason.Store("")
		}
	}
	return false
}

func (g *TLSGuardrails) warn() {
	g.log.Warn("════════════════════════════════════════════════════════════════")
	g.log.Warn("[SECURITY] TLS DISABLED. Fleet HMAC + payloads travel in plaintext.")
	g.log.Warn("[SECURITY] Set tls.mode: \"on\" before any production deploy.")
	g.log.Warn("[SECURITY] Reason: %v", g.reason.Load())
	g.log.Warn("════════════════════════════════════════════════════════════════")
}

func isTLSOff(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "off", "no", "false", "disabled", "0":
		return true
	}
	return false
}

// isProductionEnv reads OBLIVRA_PRODUCTION. Truthy values are 1 / true
// / yes / on (case-insensitive). Anything else (including unset) is
// non-production.
func isProductionEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OBLIVRA_PRODUCTION"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
