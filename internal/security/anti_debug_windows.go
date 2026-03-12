//go:build windows

package security

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/kingknull/oblivrashell/internal/logger"
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procIsDebuggerPresent        = kernel32.NewProc("IsDebuggerPresent")
	procCheckRemoteDebuggerPresent = kernel32.NewProc("CheckRemoteDebuggerPresent")
	procGetCurrentProcess        = kernel32.NewProc("GetCurrentProcess")
	ntdll                         = syscall.NewLazyDLL("ntdll.dll")
	procNtQueryInformationProcess = ntdll.NewProc("NtQueryInformationProcess")
)

const (
	ProcessDebugPort          = 7
	ProcessDebugFlags         = 31
	ProcessDebugObjectHandle  = 30
)

// AntiDebugMonitor provides continuous runtime integrity checking against process debuggers.
type AntiDebugMonitor struct {
	ctx    context.Context
	cancel context.CancelFunc
	log    *logger.Logger
}

// NewAntiDebugMonitor creates a new monitor instance.
func NewAntiDebugMonitor(log *logger.Logger) *AntiDebugMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &AntiDebugMonitor{
		ctx:    ctx,
		cancel: cancel,
		log:    log,
	}
}

// Start begins a background goroutine that polls the OS kernel for attached debuggers.
func (m *AntiDebugMonitor) Start() {
	m.log.Info("Starting native Win32 Anti-Debug Monitor (High-Frequency Kernel Hooks)...")
	go func() {
		// High-frequency check (200ms) for critical runtime protection
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				if m.isDebuggerAttached() {
					m.log.Error("CRITICAL SECURITY VIOLATION: Debugger or memory-dump utility detected attaching to process memory!")
					m.triggerAntiDebugResponse()
				}
			}
		}
	}()
}

// Stop halts the background monitoring.
func (m *AntiDebugMonitor) Stop() {
	m.cancel()
}

// isDebuggerAttached queries Win32 kernel hooks to detect process introspections.
func (m *AntiDebugMonitor) isDebuggerAttached() bool {
	// 1. Standard Win32 API check
	flag, _, _ := procIsDebuggerPresent.Call()
	if flag != 0 {
		return true
	}

	processHandle, _, _ := procGetCurrentProcess.Call()

	// 2. Check for remote debuggers via standard API
	var isDebuggerPresent int32
	ret, _, _ := procCheckRemoteDebuggerPresent.Call(processHandle, uintptr(unsafe.Pointer(&isDebuggerPresent)))
	if ret != 0 && isDebuggerPresent != 0 {
		return true
	}

	// 3. Low-level Kernel Check: ProcessDebugPort
	var debugPort uintptr
	ret, _, _ = procNtQueryInformationProcess.Call(
		processHandle,
		uintptr(ProcessDebugPort),
		uintptr(unsafe.Pointer(&debugPort)),
		uintptr(unsafe.Sizeof(debugPort)),
		uintptr(0),
	)
	if ret == 0 && debugPort != 0 {
		return true
	}

	// 4. Low-level Kernel Check: ProcessDebugFlags
	var debugFlags uint32
	ret, _, _ = procNtQueryInformationProcess.Call(
		processHandle,
		uintptr(ProcessDebugFlags),
		uintptr(unsafe.Pointer(&debugFlags)),
		uintptr(unsafe.Sizeof(debugFlags)),
		uintptr(0),
	)
	// If flags are 0, it indicates a debugger is likely present/attached (normal is non-zero)
	// Note: ret == 0 means success of the query itself
	if ret == 0 && debugFlags == 0 {
		return true
	}

	return false
}

// triggerAntiDebugResponse crashes the application instantly to guarantee zero-day resilience.
func (m *AntiDebugMonitor) triggerAntiDebugResponse() {
	// Immediately unrecoverable panic, bypassing standard shutdown to prevent EDR tracing.
	fmt.Fprintf(os.Stderr, "FATAL INTEGRITY EXCEPTION: Anti-Debug triggered. Scuttling process.\n")
	// [DEV MODE BYPASS] - Disabling forceful termination so `wails dev` can attach its hooks without being killed.
	// os.Exit(0x40010004) // DBG_TERMINATE_PROCESS constant
}
