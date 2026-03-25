package agent

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ResponseAction defines the interface for agent-side response actions
// dispatched by the Web layer (SOAR/Playbooks) via the gRPC transport.
type ResponseAction interface {
	Name() string
	Execute() error
}

// ProcessSnapshot captures metadata about a process before termination.
type ProcessSnapshot struct {
	PID        int               `json:"pid"`
	Name       string            `json:"name"`
	CmdLine    string            `json:"cmdline,omitempty"`
	CWD        string            `json:"cwd,omitempty"`
	User       string            `json:"user,omitempty"`
	Memory     uint64            `json:"memory_bytes"`
	OpenFiles  int               `json:"open_files"`
	CapturedAt string            `json:"captured_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ResponseActionExecutor handles agent-side response actions.
type ResponseActionExecutor struct {
	log *logger.Logger
}

// NewResponseActionExecutor creates a new response action executor for the agent.
func NewResponseActionExecutor(log *logger.Logger) *ResponseActionExecutor {
	return &ResponseActionExecutor{
		log: log.WithPrefix("response-actions"),
	}
}

// KillProcess terminates a process on the agent's host by PID.
// This is an AGENT-LAYER operation dispatched by the Web's SOAR engine.
func (r *ResponseActionExecutor) KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}
	if pid == os.Getpid() {
		return fmt.Errorf("refusing to kill self (PID %d)", pid)
	}
	// Do not allow killing PID 1 (init/systemd)
	if pid == 1 {
		return fmt.Errorf("refusing to kill PID 1 (init)")
	}

	r.log.Warn("[RESPONSE] 🔴 Killing process PID=%d on behalf of SOAR", pid)

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	if err := proc.Kill(); err != nil {
		return fmt.Errorf("kill process %d: %w", pid, err)
	}

	r.log.Info("[RESPONSE] ✅ Process PID=%d terminated by SOAR directive", pid)
	return nil
}

// CollectProcessSnapshot captures metadata about a process before termination.
// Useful for forensic analysis — capture state before killing malware.
func (r *ResponseActionExecutor) CollectProcessSnapshot(pid int) (*ProcessSnapshot, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid PID: %d", pid)
	}

	snapshot := &ProcessSnapshot{
		PID:        pid,
		CapturedAt: time.Now().Format(time.RFC3339),
		Metadata:   make(map[string]string),
	}

	switch runtime.GOOS {
	case "linux":
		r.collectLinuxSnapshot(pid, snapshot)
	case "windows":
		snapshot.Metadata["platform"] = "windows"
		// Windows process snapshot requires Win32 API
		// For now, we capture basic metadata
		snapshot.Name = fmt.Sprintf("pid-%d", pid)
	default:
		snapshot.Metadata["platform"] = runtime.GOOS
	}

	r.log.Info("[RESPONSE] Process snapshot captured: PID=%d name=%s", pid, snapshot.Name)
	return snapshot, nil
}

// KillAndCapture combines snapshot + kill in a single atomic operation.
// This is the recommended flow: capture evidence, then terminate.
func (r *ResponseActionExecutor) KillAndCapture(pid int) (*ProcessSnapshot, error) {
	// 1. Capture snapshot first (before the process is gone)
	snapshot, err := r.CollectProcessSnapshot(pid)
	if err != nil {
		r.log.Warn("[RESPONSE] Snapshot failed for PID=%d (proceeding with kill): %v", pid, err)
		// Continue with kill even if snapshot fails
	}

	// 2. Kill the process
	if err := r.KillProcess(pid); err != nil {
		return snapshot, err
	}

	return snapshot, nil
}

// ──────────────────────────────────────────────
// Platform-specific snapshot collection
// ──────────────────────────────────────────────

func (r *ResponseActionExecutor) collectLinuxSnapshot(pid int, snap *ProcessSnapshot) {
	base := fmt.Sprintf("/proc/%d", pid)

	// Process name
	if comm, err := os.ReadFile(base + "/comm"); err == nil {
		snap.Name = string(comm)
	}

	// Command line
	if cmdline, err := os.ReadFile(base + "/cmdline"); err == nil {
		snap.CmdLine = string(cmdline)
	}

	// Current working directory
	if cwd, err := os.Readlink(base + "/cwd"); err == nil {
		snap.CWD = cwd
	}

	// Count open file descriptors
	if fds, err := os.ReadDir(base + "/fd"); err == nil {
		snap.OpenFiles = len(fds)
	}

	// Memory from status
	if status, err := os.ReadFile(base + "/status"); err == nil {
		snap.Metadata["status"] = string(status)
	}
}
