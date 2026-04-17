package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// Watchdog provides self-protection for the OBLIVRA agent by monitoring
// eBPF telemetry for tampering attempts against its own process.
type Watchdog struct {
	log *logger.Logger
	pid int
}

// NewWatchdog creates a new self-protection service.
func NewWatchdog(log *logger.Logger) *Watchdog {
	return &Watchdog{
		log: log.WithPrefix("watchdog"),
		pid: os.Getpid(),
	}
}

// Inspect evaluates an event for anti-tampering violations.
// If a violation is found, it returns a synthetic 'security_alert' event.
func (w *Watchdog) Inspect(evt Event) *Event {
	// 5. Sovereign-Grade: Kernel Anti-Tamper Monitoring
	
	// 1. Unauthorized Ptrace (Process Injection / Debugging)
	if evt.Type == "ptrace_call" {
		targetPID := fmt.Sprintf("%v", evt.Data["target_pid"])
		myPID := fmt.Sprintf("%d", w.pid)
		
		if targetPID == myPID {
			w.log.Warn("[SECURITY] Unauthorized ptrace attempt detected on agent process (PID %d) from source PID %v!", w.pid, evt.Data["pid"])
			return &Event{
				Timestamp: evt.Timestamp,
				Source:    "sovereign_watchdog",
				Type:      "security_alert",
				Host:      evt.Host,
				Data: map[string]interface{}{
					"alert":            "Active Process Tampering Attempt",
					"mitre_technique":  "T1055.008",
					"severity":         "CRITICAL",
					"action_requested": "isolate_network", // Suggested remediation
					"details":          fmt.Sprintf("External process (PID %v, Comm: %v) attempted to attach to agent via ptrace", evt.Data["pid"], evt.Data["comm"]),
				},
			}
		}
	}

	// 2. Sensitive Configuration Tampering
	if evt.Type == "file_access" {
		path := fmt.Sprintf("%v", evt.Data["path"])
		// Monitor write/modify attempts on identity keys and vault
		if (strings.Contains(path, "identity.key") || strings.Contains(path, "vault.json")) && w.isWriteAccess(evt.Data["flags"]) {
			// Ignore if it's our own process (e.g. during rotation)
			if fmt.Sprintf("%v", evt.Data["pid"]) == fmt.Sprintf("%d", w.pid) {
				return nil
			}
			
			w.log.Warn("[SECURITY] Unauthorized modification attempt on sensitive file: %s", path)
			return &Event{
				Timestamp: evt.Timestamp,
				Source:    "sovereign_watchdog",
				Type:      "security_alert",
				Host:      evt.Host,
				Data: map[string]interface{}{
					"alert":           "Integrity Violation Attempt",
					"mitre_technique": "T1565.001",
					"severity":        "CRITICAL",
					"details":         fmt.Sprintf("Process (PID %v, Comm: %v) attempted unauthorized write to %s", evt.Data["pid"], evt.Data["comm"], path),
				},
			}
		}
	}

	// 3. Executable Memory Modification (Mitre T1055)
	if evt.Type == "memory_protect" || evt.Type == "memory_map" {
		prot := uint32(0)
		// Handle various possible types from JSON/Event data
		if p, ok := evt.Data["prot"].(uint32); ok {
			prot = p
		} else if p, ok := evt.Data["prot"].(float64); ok {
			prot = uint32(p)
		}

		// 0x4 is PROT_EXEC. Setting memory to executable is a high-confidence indicator of shellcode injection.
		if (prot & 0x4) != 0 {
			if fmt.Sprintf("%v", evt.Data["pid"]) == fmt.Sprintf("%d", w.pid) {
				w.log.Warn("[SECURITY] Self-modifying code or injection attempt detected in agent process!")
				return &Event{
					Timestamp: evt.Timestamp,
					Source:    "sovereign_watchdog",
					Type:      "security_alert",
					Host:      evt.Host,
					Data: map[string]interface{}{
						"alert":           "Unauthorized Executable Memory Mapping",
						"mitre_technique": "T1055",
						"severity":        "CRITICAL",
						"details":         fmt.Sprintf("Agent process (PID %d) attempted to create or modify executable memory (PROT_EXEC)", w.pid),
					},
				}
			}
		}
	}

	return nil
}

func (w *Watchdog) isWriteAccess(flags interface{}) bool {
	f, ok := flags.(uint32)
	if !ok {
		// Try int conversion if float (json unmarshal artifact)
		if f64, ok := flags.(float64); ok {
			f = uint32(f64)
		} else {
			return false
		}
	}
	// O_WRONLY (1) or O_RDWR (2) or O_CREAT (64)
	return (f & 0x1) != 0 || (f & 0x2) != 0 || (f & 0x40) != 0
}
