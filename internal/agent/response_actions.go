package agent

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
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

// ResponseActionExecutor (legacy name retained) now handles read-only forensic
// snapshot operations. Phase 36.7: KillProcess + KillAndCapture removed —
// active response is delegated to external SOAR per the broad scope cut.
type ResponseActionExecutor struct {
	log *logger.Logger
}

func NewResponseActionExecutor(log *logger.Logger) *ResponseActionExecutor {
	return &ResponseActionExecutor{log: log.WithPrefix("forensic-snapshot")}
}

// CollectProcessSnapshot captures metadata about a process for forensic analysis.
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

// Phase 36.7: KillAndCapture removed — kill primitive deleted with the broad
// scope cut. CollectProcessSnapshot remains as a forensic primitive.

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

// CollectProcessInventory captures a snapshot of all running processes on the system.
func (r *ResponseActionExecutor) CollectProcessInventory() []ProcessSnapshot {
	r.log.Info("[response] Capturing full process inventory...")
	var inventory []ProcessSnapshot

	if runtime.GOOS != "linux" {
		r.log.Warn("[response] Full process inventory only implemented for Linux")
		return inventory
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		r.log.Error("[response] Failed to read /proc: %v", err)
		return inventory
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		snap := ProcessSnapshot{
			PID:        pid,
			CapturedAt: time.Now().Format(time.RFC3339),
			Metadata:   make(map[string]string),
		}
		r.collectLinuxSnapshot(pid, &snap)
		inventory = append(inventory, snap)
	}

	r.log.Info("[response] Captured %d processes", len(inventory))
	return inventory
}
