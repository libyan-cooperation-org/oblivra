// Watchdog-driven self-restart for the OBLIVRA agent.
//
// Closes the "Watchdog auto-restart" gap from the agent feature audit:
// the existing `watchdog.go` detects tamper attempts (writes to the
// agent's own executable, ptrace, etc.) and synthesises CRITICAL
// alerts, but it does NOT actually restart the process when its
// integrity is compromised. This file plugs that gap.
//
// Design choice: the agent does NOT fork-and-exec itself. That's
// fragile (zombie processes, leaked file descriptors, race with WAL
// flushing). Instead we cleanly exit with a well-known status code,
// and rely on the OS service manager to bring us back:
//
//   Linux:    systemd unit with `Restart=always RestartSec=5s`
//   macOS:    launchd plist with `KeepAlive=true`
//   Windows:  Windows Service Control Manager with `recovery` policy
//
// Each of these treats exit code 75 (EX_TEMPFAIL on BSD) as "expected
// failure, please restart me." Code 0 means "I'm done, don't restart"
// and code 1 means "I crashed, please tell the operator."
//
// Frontend operator-control flow (Phase 30.5c "Restart Agent" button):
//   1. UI calls REST `POST /api/v1/agents/:id/restart`
//   2. Server queues a `restart` PendingAction
//   3. Agent polls, sees the action, calls `RequestRestart("ui-request")`
//   4. Agent flushes WAL, closes collectors, calls os.Exit(75)
//   5. systemd/launchd/SCM auto-respawns
//
// Tamper-detection flow:
//   1. Watchdog.Inspect produces a CRITICAL alert
//   2. Caller checks the alert severity and, if it's a self-tamper,
//      calls `RequestRestart("watchdog-tamper")`
//   3. Same shutdown sequence as above
//
// Both flows go through the SAME RequestRestart entry point so the
// shutdown drain is identical regardless of trigger.

package agent

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// RestartExitCode is the well-known status the agent exits with when
// it wants its supervisor to bring it back. Matches BSD EX_TEMPFAIL,
// which systemd, launchd, and the Windows Service Control Manager
// all recognise as "transient failure, restart me."
const RestartExitCode = 75

// RestartReason is a free-form tag that ends up in the structured
// log line just before exit. Used for forensic correlation: an
// operator looking at "agent disappeared at 13:42" should be able to
// grep `restart: reason=` in journald and see exactly why.
type RestartReason string

const (
	RestartReasonWatchdog       RestartReason = "watchdog-tamper"
	RestartReasonUIRequest      RestartReason = "ui-request"
	RestartReasonConfigChange   RestartReason = "config-change"
	RestartReasonOOMRecovery    RestartReason = "oom-recovery"
	RestartReasonHealthFailure  RestartReason = "health-check-failure"
)

// RestartManager coordinates a clean shutdown + supervisor-driven
// respawn. One per Agent. Safe to call RequestRestart from any
// goroutine — only the first call wins.
type RestartManager struct {
	mu       sync.Mutex
	once     sync.Once
	log      *logger.Logger
	shutdown func(ctx context.Context) error
	exitFn   func(int) // injectable for tests (default: os.Exit)
}

// NewRestartManager constructs a manager that calls `shutdown` to
// drain the agent (close collectors, flush WAL, etc.) before
// terminating the process.
//
// Pass `nil` shutdown for a "die immediately" mode used in tests.
func NewRestartManager(log *logger.Logger, shutdown func(ctx context.Context) error) *RestartManager {
	return &RestartManager{
		log:      log.WithPrefix("restart"),
		shutdown: shutdown,
		exitFn:   os.Exit,
	}
}

// RequestRestart fires the shutdown sequence and exits with
// RestartExitCode. Idempotent — second and later calls are
// silently ignored so concurrent triggers don't double-shut.
//
// `timeout` bounds how long the drain may take before we exit
// regardless. 10s is the default if you pass 0; the WAL flush is
// the only expected slow step and it's typically <1s.
func (rm *RestartManager) RequestRestart(reason RestartReason, timeout time.Duration) {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	rm.once.Do(func() {
		rm.log.Warn("restart requested: reason=%s timeout=%s", reason, timeout)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if rm.shutdown != nil {
			if err := rm.shutdown(ctx); err != nil {
				rm.log.Error("restart: shutdown drain failed: %v (proceeding with exit anyway)", err)
			}
		}
		rm.log.Warn("restart: exiting with code %d, supervisor will respawn", RestartExitCode)
		rm.exitFn(RestartExitCode)
	})
}

// SetExitFn replaces os.Exit for testing. Production callers should
// never use this.
func (rm *RestartManager) SetExitFn(fn func(int)) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.exitFn = fn
}
