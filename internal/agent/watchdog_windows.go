//go:build windows

package agent

import (
	"time"
)

func (w *Watchdog) TriggerSelfTest() *Event {
	w.log.Info("[SIM] Triggering self-tamper test (Simulated for Windows)...")
	
	// 1. Synthetic Alert (Direct verification of the alerting pipeline)
	syntheticEvent := &Event{
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    "sovereign_watchdog",
		Type:      "security_alert",
		Host:      "localhost",
		Data: map[string]interface{}{
			"alert":           "SIMULATED: Unauthorized Executable Memory Mapping",
			"mitre_technique": "T1055",
			"severity":        "CRITICAL",
			"details":         "Self-test triggered (Windows Stub): simulated memory protection violation",
		},
	}
	w.log.Warn("[SIM] Synthetic security alert generated for UI verification (Windows).")

	// Note: Memory protection syscalls like Mmap/Mprotect are not directly available via 'syscall' package on Windows 
	// without using specific Windows API calls (VirtualAlloc, VirtualProtect).
	// For the watchdog self-test on Windows, we'll stick to the synthetic alert for now.

	return syntheticEvent
}
