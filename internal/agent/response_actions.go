package agent

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ResponseAction defines the interface for agent-side response actions.
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

// ResponseActionExecutor handles agent-side response actions dispatched by SOAR.
type ResponseActionExecutor struct {
	log *logger.Logger
}

func NewResponseActionExecutor(log *logger.Logger) *ResponseActionExecutor {
	return &ResponseActionExecutor{log: log.WithPrefix("response-actions")}
}

// protectedPIDs is the set of OS PIDs that must never be killed by SOAR actions.
// Killing these would crash or destabilise the host.
var protectedPIDs = map[int]string{
	0: "idle/kernel",
	1: "init/systemd",
	2: "kthreadd",
	4: "System (Windows)",
}

// KillProcess terminates a process by PID.
// Refuses to kill the agent itself, PID 0/1/2 (Linux kernel threads),
// or PID 4 (Windows System process).
func (r *ResponseActionExecutor) KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}
	if pid == os.Getpid() {
		return fmt.Errorf("refusing to kill self (PID %d)", pid)
	}
	if name, protected := protectedPIDs[pid]; protected {
		return fmt.Errorf("refusing to kill protected system process PID %d (%s)", pid, name)
	}

	r.log.Warn("[response] Killing PID=%d on behalf of SOAR", pid)

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := proc.Kill(); err != nil {
		// On Linux, ESRCH means the process already exited — not an error
		if strings.Contains(err.Error(), "no such process") {
			r.log.Info("[response] PID=%d already gone (process exited before kill)", pid)
			return nil
		}
		return fmt.Errorf("kill PID %d: %w", pid, err)
	}

	r.log.Info("[response] PID=%d terminated by SOAR directive", pid)
	return nil
}

// CollectProcessSnapshot captures metadata about a process for forensic analysis.
// Call this before KillProcess to preserve evidence.
func (r *ResponseActionExecutor) CollectProcessSnapshot(pid int) (*ProcessSnapshot, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid PID: %d", pid)
	}

	snap := &ProcessSnapshot{
		PID:        pid,
		CapturedAt: time.Now().Format(time.RFC3339),
		Metadata:   make(map[string]string),
	}

	switch runtime.GOOS {
	case "linux":
		r.collectLinuxSnapshot(pid, snap)
	case "windows":
		snap.Metadata["platform"] = "windows"
		snap.Name = fmt.Sprintf("pid-%d", pid)
		// Full Win32 snapshot (OpenProcess + QueryInformationProcess) is in
		// response_actions_windows.go — this stub provides the minimum.
	default:
		snap.Metadata["platform"] = runtime.GOOS
	}

	r.log.Info("[response] Process snapshot: PID=%d name=%q open_files=%d",
		pid, snap.Name, snap.OpenFiles)
	return snap, nil
}

// KillAndCapture atomically snapshots then kills a process.
// This is the correct SOAR flow: capture forensic evidence first, then terminate.
func (r *ResponseActionExecutor) KillAndCapture(pid int) (*ProcessSnapshot, error) {
	snap, err := r.CollectProcessSnapshot(pid)
	if err != nil {
		r.log.Warn("[response] Snapshot failed for PID=%d (proceeding with kill): %v", pid, err)
	}
	if err := r.KillProcess(pid); err != nil {
		return snap, err
	}
	return snap, nil
}

// ─── platform-specific snapshot helpers ──────────────────────────────────────

func (r *ResponseActionExecutor) collectLinuxSnapshot(pid int, snap *ProcessSnapshot) {
	base := fmt.Sprintf("/proc/%d", pid)

	if comm, err := os.ReadFile(base + "/comm"); err == nil {
		snap.Name = strings.TrimRight(string(comm), "\n\x00")
	}
	if cmdline, err := os.ReadFile(base + "/cmdline"); err == nil {
		snap.CmdLine = strings.ReplaceAll(string(cmdline), "\x00", " ")
	}
	if cwd, err := os.Readlink(base + "/cwd"); err == nil {
		snap.CWD = cwd
	}
	if fds, err := os.ReadDir(base + "/fd"); err == nil {
		snap.OpenFiles = len(fds)
	}
	if status, err := os.ReadFile(base + "/status"); err == nil {
		snap.Metadata["status"] = string(status)
	}
	// Parse VmRSS from /proc/<pid>/status for memory
	for _, line := range strings.Split(snap.Metadata["status"], "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			var kb uint64
			fmt.Sscanf(strings.TrimPrefix(line, "VmRSS:"), "%d", &kb)
			snap.Memory = kb * 1024
			break
		}
	}
}
