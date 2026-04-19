//go:build !windows

package agent

import (
	"syscall"
	"time"
)

func (w *Watchdog) TriggerSelfTest() *Event {
	w.log.Info("[SIM] Triggering self-tamper test (PROT_EXEC)...")
	
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
			"details":         "Self-test triggered: simulated memory protection violation",
		},
	}
	w.log.Warn("[SIM] Synthetic security alert generated for UI verification.")

	// 2. Real Tamper Attempt (Will only be caught if eBPF is active)
	page, err := syscall.Mmap(-1, 0, 4096, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err == nil {
		defer syscall.Munmap(page)
		// This should be caught by the eBPF sys_enter_mprotect probe
		syscall.Mprotect(page, syscall.PROT_READ|syscall.PROT_EXEC)
	}

	return syntheticEvent
}
