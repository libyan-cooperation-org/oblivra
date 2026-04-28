// Package tamper implements the agent-side half of Tamper Path 1:
// oplog shipping (Layer 1), hash chain integrity (Layer 2), and
// heartbeat scheduling (Layer 3). The server-side handlers live in
// internal/api/rest_tamper.go.
//
// Three independent goroutines, all sharing the same agent identity
// + HMAC fleet secret + server URL:
//
//   • OplogShipper — reads the agent's own log file (or a buffered
//     channel if the caller sets up logger fan-out), batches every 5s
//     or 256 lines, signs, POSTs to /api/v1/agent/oplog. Tracks a
//     monotonic batch_seq per agent so the server detects gaps.
//
//   • HashChain — accumulates a rolling SHA256 over committed lines.
//     Each batch carries the prev_hash for verification. An attacker
//     who modifies the local log file in place cannot reproduce the
//     chain → server detects the break.
//
//   • Heartbeat — every 30s POSTs uptime + log file inode/size + last
//     hash + wall_clock to /api/v1/agent/heartbeat. Server-side rules
//     fire on log_truncated / heartbeat_missed / time_skew.
//
// Construct one Subsystem per agent process via NewSubsystem; call
// Start to launch the goroutines and Stop on shutdown.
package tamper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// Config carries everything needed to ship oplog + heartbeats.
type Config struct {
	AgentID     string
	ServerURL   string // e.g. "https://oblivra.internal:8443"
	FleetSecret []byte
	LogPath     string // path to the agent's own log file
	VerifyTLS   bool   // true = strict; false = skip-verify (dev)

	// Tunables — leave zero for sane defaults.
	OplogBatchSize    int           // default 256
	OplogBatchTimeout time.Duration // default 5s
	HeartbeatPeriod   time.Duration // default 30s
}

// Subsystem owns the three goroutines + their shared state. Safe for
// the caller to construct once per process.
type Subsystem struct {
	cfg    Config
	log    *logger.Logger
	chain  *HashChain
	uptime time.Time

	batchSeq atomic.Int64

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewSubsystem constructs the tamper subsystem and validates required
// config. Returns an error when AgentID / ServerURL / FleetSecret /
// LogPath are missing — without any one, the platform can't honour
// the tamper-evidence claim.
func NewSubsystem(cfg Config, log *logger.Logger) (*Subsystem, error) {
	if cfg.AgentID == "" {
		return nil, errMissingField("agent_id")
	}
	if cfg.ServerURL == "" {
		return nil, errMissingField("server_url")
	}
	if len(cfg.FleetSecret) == 0 {
		return nil, errMissingField("fleet_secret")
	}
	if cfg.LogPath == "" {
		return nil, errMissingField("log_path")
	}
	if cfg.OplogBatchSize <= 0 {
		cfg.OplogBatchSize = 256
	}
	if cfg.OplogBatchTimeout <= 0 {
		cfg.OplogBatchTimeout = 5 * time.Second
	}
	if cfg.HeartbeatPeriod <= 0 {
		cfg.HeartbeatPeriod = 30 * time.Second
	}
	return &Subsystem{
		cfg:    cfg,
		log:    log.WithPrefix("tamper"),
		chain:  NewHashChain(),
		uptime: time.Now(),
	}, nil
}

// Start launches the three goroutines. Returns immediately. Use Stop
// or cancel the parent context to shut down.
func (s *Subsystem) Start(ctx context.Context) error {
	subCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(1)
	go s.runOplogShipper(subCtx)

	s.wg.Add(1)
	go s.runHeartbeat(subCtx)

	s.log.Info("subsystem started: agent=%s server=%s logpath=%s",
		s.cfg.AgentID, s.cfg.ServerURL, s.cfg.LogPath)
	return nil
}

// Stop cancels every goroutine and waits up to 5s for them to drain.
func (s *Subsystem) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.log.Warn("stop timed out — goroutines still running")
	}
}

// nextBatchSeq returns a monotonic per-process sequence number. The
// server uses this to detect gaps (batch_seq jumped from 42 → 50 = 7
// batches lost or suppressed).
func (s *Subsystem) nextBatchSeq() int64 {
	return s.batchSeq.Add(1)
}

// ── HashChain (Layer 2) ────────────────────────────────────────

// HashChain accumulates a rolling SHA256: H(n) = SHA256(H(n-1) || raw)
// where raw is the bytes of one committed log line. Agent ships the
// current head with each batch; server stores it and verifies chain
// continuity on the next batch.
//
// Limitations (worth being honest about):
//   • An attacker who pwns the agent process can rewrite H(n) on the
//     fly. The chain is forensic for past events, not real-time
//     prevention. Still useful: most real-world tampering is
//     "after-the-fact rm + truncate", which the chain catches.
//   • The chain key is in process memory; we don't HMAC the chain
//     because the fleet secret already authenticates the transport.
//     A future hardening: derive a per-agent chain key from the
//     fleet secret at registration so the head is unforgeable even
//     if process memory leaks.
type HashChain struct {
	mu   sync.Mutex
	head [32]byte // running SHA256
	zero bool     // true if no commits yet
}

func NewHashChain() *HashChain {
	return &HashChain{zero: true}
}

// Commit advances the chain by one entry. Returns the new head as hex.
func (c *HashChain) Commit(raw []byte) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := sha256.New()
	if !c.zero {
		h.Write(c.head[:])
	}
	h.Write(raw)
	sum := h.Sum(nil)
	copy(c.head[:], sum)
	c.zero = false
	return hex.EncodeToString(c.head[:])
}

// Head returns the current head as hex (or empty string if no commits).
func (c *HashChain) Head() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.zero {
		return ""
	}
	return hex.EncodeToString(c.head[:])
}

// errMissingField is a tiny error for the common "required field
// blank" pattern used in NewSubsystem.
type missingFieldError struct{ field string }

func (e missingFieldError) Error() string {
	return "tamper: required config field missing: " + e.field
}
func errMissingField(f string) error { return missingFieldError{field: f} }
