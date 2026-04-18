//go:build windows

package services

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows kernel32 ConPTY procedures
var (
	kernel32                           = windows.NewLazySystemDLL("kernel32.dll")
	procCreatePseudoConsole            = kernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole            = kernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole             = kernel32.NewProc("ClosePseudoConsole")
	procInitializeProcThreadAttribList = kernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttrib         = kernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttribList     = kernel32.NewProc("DeleteProcThreadAttributeList")
)

const (
	// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE attribute number
	_PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE uintptr = 0x00020016
)

// coord is the Windows COORD struct (2 × int16)
type coord struct{ X, Y int16 }

// siEx is a STARTUPINFOEXW — STARTUPINFOW followed by lpAttributeList pointer.
// The total size must be passed in StartupInfo.Cb.
type siEx struct {
	windows.StartupInfo
	lpAttributeList uintptr
}

// startPTY is the top-level entry called by LocalService.
// It uses ConPTY on Windows 10 1809+ and falls back to pipe mode.
func startPTY(cmd *exec.Cmd, cols, rows int) (*ptySession, error) {
	if err := procCreatePseudoConsole.Find(); err != nil {
		// ConPTY not available – old Windows build
		return startPTYPipeMode(cmd)
	}
	// On Windows, ConPTY is better for terminal apps.
	return startPTYConPTY(cmd, cols, rows)
}

func startPTYConPTY(cmd *exec.Cmd, cols, rows int) (*ptySession, error) {
	// ── Create pipe pair for PTY input (keystrokes: us → shell) ────────
	ptyInR, ptyInW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("conpty: input pipe: %w", err)
	}
	// ── Create pipe pair for PTY output (shell output: shell → us) ─────
	ptyOutR, ptyOutW, err := os.Pipe()
	if err != nil {
		ptyInR.Close()
		ptyInW.Close()
		return nil, fmt.Errorf("conpty: output pipe: %w", err)
	}

	// ── CreatePseudoConsole ────────────────────────────────────────────
	// ConPTY reads from ptyInR, writes to ptyOutW.
	// After CreatePseudoConsole succeeds, these fd's are owned by the PTY.
	sz := coord{X: int16(cols), Y: int16(rows)}
	var hPC windows.Handle

	hr, _, _ := procCreatePseudoConsole.Call(
		uintptr(unsafe.Pointer(&sz)),
		uintptr(ptyInR.Fd()),
		uintptr(ptyOutW.Fd()),
		0,
		uintptr(unsafe.Pointer(&hPC)),
	)
	// We hand off ptyInR and ptyOutW to the PTY — close our copies
	ptyInR.Close()
	ptyOutW.Close()

	if hr != 0 { // HRESULT S_OK = 0
		ptyInW.Close()
		ptyOutR.Close()
		return nil, fmt.Errorf("conpty: CreatePseudoConsole HRESULT=0x%08x", hr)
	}

	// ── Build PROC_THREAD_ATTRIBUTE_LIST ────────────────────────────────
	attrBuf, err := buildAttrList(hPC)
	if err != nil {
		procClosePseudoConsole.Call(uintptr(hPC))
		ptyInW.Close()
		ptyOutR.Close()
		return nil, err
	}

	// ── Build STARTUPINFOEX ─────────────────────────────────────────────
	si := siEx{}
	si.Cb = uint32(unsafe.Sizeof(si))
	si.lpAttributeList = uintptr(unsafe.Pointer(&attrBuf[0]))
	si.Flags = windows.STARTF_USESHOWWINDOW
	si.ShowWindow = windows.SW_HIDE

	// ── CreateProcess ───────────────────────────────────────────────────
	cmdStr := cmd.Path
	for i, arg := range cmd.Args {
		if i == 0 {
			continue
		}
		cmdStr += " " + arg
	}
	cmdLine, err := windows.UTF16PtrFromString(cmdStr)
	if err != nil {
		procDeleteProcThreadAttribList.Call(uintptr(unsafe.Pointer(&attrBuf[0])))
		procClosePseudoConsole.Call(uintptr(hPC))
		ptyInW.Close()
		ptyOutR.Close()
		return nil, fmt.Errorf("conpty: UTF16: %w", err)
	}

	var pi windows.ProcessInformation
	err = windows.CreateProcess(
		nil,
		cmdLine,
		nil,
		nil,
		false,
		// EXTENDED_STARTUPINFO_PRESENT tells CreateProcess to treat lpStartupInfo
		// as STARTUPINFOEX rather than STARTUPINFO.
		windows.EXTENDED_STARTUPINFO_PRESENT | 
		windows.CREATE_UNICODE_ENVIRONMENT | 
		windows.CREATE_NO_WINDOW,
		nil,
		nil,
		// We pass a pointer to StartupInfo (the first field of siEx).
		// Because EXTENDED_STARTUPINFO_PRESENT is set, Windows reads the full siEx.
		(*windows.StartupInfo)(unsafe.Pointer(&si)),
		&pi,
	)

	// Attribute list is no longer needed after CreateProcess
	procDeleteProcThreadAttribList.Call(uintptr(unsafe.Pointer(&attrBuf[0])))

	if err != nil {
		procClosePseudoConsole.Call(uintptr(hPC))
		ptyInW.Close()
		ptyOutR.Close()
		return nil, fmt.Errorf("conpty: CreateProcess: %w", err)
	}

	// Close thread handle — we only need the process handle
	windows.CloseHandle(pi.Thread)

	// Wrap in os.Process so Process.Wait() works
	proc, err := os.FindProcess(int(pi.ProcessId))
	if err != nil {
		procClosePseudoConsole.Call(uintptr(hPC))
		ptyInW.Close()
		ptyOutR.Close()
		windows.CloseHandle(pi.Process)
		return nil, fmt.Errorf("conpty: FindProcess: %w", err)
	}

	bareCmd := exec.Command(cmd.Path)
	bareCmd.Process = proc

	// Keep attrBuf alive until after CreateProcess (already called above).
	runtime.KeepAlive(attrBuf)

	return &ptySession{
		cmd:    bareCmd,
		stdin:  ptyInW,
		stdout: ptyOutR,
		resize: func(c, r int) error {
			newSz := coord{X: int16(c), Y: int16(r)}
			ret, _, _ := procResizePseudoConsole.Call(
				uintptr(hPC),
				uintptr(unsafe.Pointer(&newSz)),
			)
			if ret != 0 {
				return fmt.Errorf("conpty: ResizePseudoConsole 0x%08x", ret)
			}
			return nil
		},
		closer: func() error {
			procClosePseudoConsole.Call(uintptr(hPC))
			ptyInW.Close()
			ptyOutR.Close()
			windows.CloseHandle(pi.Process)
			return nil
		},
	}, nil
}

// buildAttrList allocates and initialises a PROC_THREAD_ATTRIBUTE_LIST
// with the PSEUDOCONSOLE attribute set to hPC.
// Caller must invoke procDeleteProcThreadAttribList on &buf[0] when done.
func buildAttrList(hPC windows.Handle) ([]byte, error) {
	// Pass nil to get the required buffer size
	var size uintptr
	procInitializeProcThreadAttribList.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return nil, fmt.Errorf("conpty: InitializeProcThreadAttributeList returned 0 size")
	}

	buf := make([]byte, size)
	ret, _, err := procInitializeProcThreadAttribList.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		1,
		0,
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("conpty: InitializeProcThreadAttributeList: %w", err)
	}

	// UpdateProcThreadAttribute expects a pointer to the handle value.
	hPCVal := hPC // local copy so we can take its address safely
	ret, _, err = procUpdateProcThreadAttrib.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		0,
		_PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		uintptr(unsafe.Pointer(&hPCVal)),
		unsafe.Sizeof(hPCVal),
		0,
		0,
	)
	if ret == 0 {
		procDeleteProcThreadAttribList.Call(uintptr(unsafe.Pointer(&buf[0])))
		return nil, fmt.Errorf("conpty: UpdateProcThreadAttribute: %w", err)
	}

	return buf, nil
}
