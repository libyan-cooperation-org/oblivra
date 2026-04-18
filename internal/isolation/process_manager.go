package isolation

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ProcessInfo represents a running process on the local system.
// Used by the desktop operator console for direct host control.
type ProcessInfo struct {
	PID       int    `json:"pid"`
	Name      string `json:"name"`
	User      string `json:"user"`
	Memory    uint64 `json:"memory_bytes"`
	StartTime string `json:"start_time,omitempty"`
}

// LocalProcessController provides local process management for the Desktop operator.
// This is DESKTOP-ONLY — never exposed to the Web layer.
// (Named LocalProcessController to avoid collision with the existing ProcessManager
// in manager.go which handles worker subprocess orchestration.)
type LocalProcessController struct {
	log *logger.Logger
}

// NewLocalProcessController creates a new local process controller.
func NewLocalProcessController(log *logger.Logger) *LocalProcessController {
	return &LocalProcessController{
		log: log.WithPrefix("process-ctrl"),
	}
}

// ListProcesses returns a list of running processes on the local machine.
// Uses /proc on Linux and tasklist-style enumeration on Windows.
func (pc *LocalProcessController) ListProcesses() ([]ProcessInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return pc.listProcessesLinux()
	case "windows":
		return pc.listProcessesWindows()
	case "darwin":
		return pc.listProcessesLinux() // /proc-compatible fallback
	default:
		return nil, fmt.Errorf("process listing not supported on %s", runtime.GOOS)
	}
}

// KillProcess terminates a process by PID.
// [CAUTION] Desktop-only. Requires appropriate privileges for system processes.
func (pc *LocalProcessController) KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}
	if pid == os.Getpid() {
		return fmt.Errorf("cannot kill own process (PID %d)", pid)
	}

	pc.log.Warn("[PROCESS-CTRL] 🔴 Killing process PID=%d", pid)

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	if err := proc.Kill(); err != nil {
		return fmt.Errorf("kill process %d: %w", pid, err)
	}

	pc.log.Info("[PROCESS-CTRL] ✅ Process PID=%d terminated", pid)
	return nil
}

// KillProcessByName terminates all processes matching the given name.
// Returns the number of processes killed and any error from the last failure.
func (pc *LocalProcessController) KillProcessByName(name string) (int, error) {
	if name == "" {
		return 0, fmt.Errorf("empty process name")
	}

	processes, err := pc.ListProcesses()
	if err != nil {
		return 0, fmt.Errorf("list processes: %w", err)
	}

	killed := 0
	var lastErr error
	for _, p := range processes {
		if strings.EqualFold(p.Name, name) {
			if err := pc.KillProcess(p.PID); err != nil {
				lastErr = err
				pc.log.Warn("[PROCESS-CTRL] Failed to kill PID=%d (%s): %v", p.PID, name, err)
			} else {
				killed++
			}
		}
	}

	if killed == 0 && lastErr != nil {
		return 0, lastErr
	}
	return killed, nil
}

// ──────────────────────────────────────────────
// Platform-specific implementations
// ──────────────────────────────────────────────

func (pc *LocalProcessController) listProcessesLinux() ([]ProcessInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	var processes []ProcessInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Not a PID directory
		}

		info := ProcessInfo{PID: pid}

		// Read process name from /proc/PID/comm
		if comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
			info.Name = strings.TrimSpace(string(comm))
		}

		// Read process status for memory and user info
		if status, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid)); err == nil {
			for _, line := range strings.Split(string(status), "\n") {
				if strings.HasPrefix(line, "VmRSS:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
							info.Memory = kb * 1024 // Convert KB to bytes
						}
					}
				}
				if strings.HasPrefix(line, "Uid:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						info.User = fields[1] // Effective UID
					}
				}
			}
		}

		// Read start time
		if stat, err := entry.Info(); err == nil {
			info.StartTime = stat.ModTime().Format(time.RFC3339)
		}

		processes = append(processes, info)
	}

	return processes, nil
}

func (pc *LocalProcessController) listProcessesWindows() ([]ProcessInfo, error) {
	// On Windows, we use the /proc-like approach via the os package.
	// For a production implementation, this would use the Windows API
	// (CreateToolhelp32Snapshot / Process32First / Process32Next).
	// For now, we provide a minimal implementation using os.FindProcess probing.

	var processes []ProcessInfo

	// Probe common PID range (simplified; production would use Win32 API)
	for pid := 1; pid <= 65535; pid++ {
		proc, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		_ = proc
		processes = append(processes, ProcessInfo{
			PID:  pid,
			Name: fmt.Sprintf("pid-%d", pid),
		})

		// Limit to first 500 to avoid excessive probing
		if len(processes) >= 500 {
			break
		}
	}

	return processes, nil
}
