//go:build windows

package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows ConPTY constants
var (
	modKernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procCreatePseudoConsole        = modKernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole        = modKernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole         = modKernel32.NewProc("ClosePseudoConsole")
	procInitializeProcThreadAttribList = modKernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttrib     = modKernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttribList = modKernel32.NewProc("DeleteProcThreadAttributeList")
)

const (
	PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE uintptr = 0x00020016
)

type coord struct {
	X, Y int16
}

type conPTY struct {
	handle windows.Handle
	inRead  *os.File
	inWrite *os.File
	outRead *os.File
	outWrite *os.File
}

func createConPTY(cols, rows int) (*conPTY, error) {
	// Create pipes: pty reads from inRead, we write to inWrite
	//               pty writes to outWrite, we read from outRead
	inRead, inWrite, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create input pipe: %w", err)
	}
	outRead, outWrite, err := os.Pipe()
	if err != nil {
		inRead.Close()
		inWrite.Close()
		return nil, fmt.Errorf("create output pipe: %w", err)
	}

	size := coord{X: int16(cols), Y: int16(rows)}

	var hPC windows.Handle
	ret, _, err := procCreatePseudoConsole.Call(
		uintptr(unsafe.Pointer(&size)),
		uintptr(inRead.Fd()),
		uintptr(outWrite.Fd()),
		0,
		uintptr(unsafe.Pointer(&hPC)),
	)
	if ret != 0 {
		inRead.Close()
		inWrite.Close()
		outRead.Close()
		outWrite.Close()
		return nil, fmt.Errorf("CreatePseudoConsole failed: 0x%x: %w", ret, err)
	}

	// The PTY now owns these ends; close our copies so the PTY is sole owner
	inRead.Close()
	outWrite.Close()

	return &conPTY{
		handle:   hPC,
		inWrite:  inWrite,
		outRead:  outRead,
	}, nil
}

func (p *conPTY) resize(cols, rows int) error {
	size := coord{X: int16(cols), Y: int16(rows)}
	ret, _, err := procResizePseudoConsole.Call(
		uintptr(p.handle),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret != 0 {
		return fmt.Errorf("ResizePseudoConsole: 0x%x: %w", ret, err)
	}
	return nil
}

func (p *conPTY) close() {
	procClosePseudoConsole.Call(uintptr(p.handle))
	p.inWrite.Close()
	p.outRead.Close()
}

// startPTY on Windows uses ConPTY (Windows Pseudo Console) for full
// ANSI/VT100 terminal emulation. Falls back to pipe mode if ConPTY
// is unavailable (Windows builds before 1809).
func startPTY(cmd *exec.Cmd, cols, rows int) (*ptySession, error) {
	pty, err := createConPTY(cols, rows)
	if err != nil {
		// ConPTY not available — fall back to pipe mode
		return startPTYPipeMode(cmd, cols, rows)
	}

	// Build STARTUPINFOEX with PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE
	si, attrList, err := buildStartupInfoEx(pty.handle)
	if err != nil {
		pty.close()
		return nil, fmt.Errorf("build startup info: %w", err)
	}

	// Launch PowerShell attached to the ConPTY
	shell := "powershell.exe"
	argv := windows.StringToUTF16Ptr(shell + " -NoLogo -NoProfile -NoExit")

	procInfo := new(windows.ProcessInformation)
	err = windows.CreateProcess(
		nil,
		argv,
		nil,
		nil,
		false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		nil,
		nil,
		&si.StartupInfo,
		procInfo,
	)
	// Free attribute list
	procDeleteProcThreadAttribList.Call(uintptr(unsafe.Pointer(attrList)))

	if err != nil {
		pty.close()
		return nil, fmt.Errorf("CreateProcess: %w", err)
	}

	// Wrap process handle in exec.Cmd so waitExit works
	process, err := os.FindProcess(int(procInfo.ProcessId))
	if err != nil {
		pty.close()
		return nil, fmt.Errorf("find process: %w", err)
	}
	cmd.Process = process

	// Use a wrapper so cmd.Wait() works correctly
	go func() {
		process.Wait()
	}()

	// We need cmd.Wait() to not panic — attach a no-op wait
	cmd.Process = process

	return &ptySession{
		cmd:    cmd,
		stdin:  pty.inWrite,
		stdout: pty.outRead,
		resize: func(c, r int) error { return pty.resize(c, r) },
		closer: func() error {
			pty.close()
			if procInfo.Process != 0 {
				windows.CloseHandle(procInfo.Process)
			}
			if procInfo.Thread != 0 {
				windows.CloseHandle(procInfo.Thread)
			}
			return nil
		},
	}, nil
}

// startupInfoEx wraps STARTUPINFOEX with the attribute list pointer
type startupInfoEx struct {
	windows.StartupInfo
	lpAttributeList uintptr
}

func buildStartupInfoEx(hPC windows.Handle) (*startupInfoEx, unsafe.Pointer, error) {
	// First call: get required size
	var attrListSize uintptr
	procInitializeProcThreadAttribList.Call(0, 1, 0, uintptr(unsafe.Pointer(&attrListSize)))

	attrList := make([]byte, attrListSize)
	ret, _, err := procInitializeProcThreadAttribList.Call(
		uintptr(unsafe.Pointer(&attrList[0])),
		1,
		0,
		uintptr(unsafe.Pointer(&attrListSize)),
	)
	if ret == 0 {
		return nil, nil, fmt.Errorf("InitializeProcThreadAttributeList: %w", err)
	}

	ret, _, err = procUpdateProcThreadAttrib.Call(
		uintptr(unsafe.Pointer(&attrList[0])),
		0,
		PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		uintptr(hPC),
		unsafe.Sizeof(hPC),
		0,
		0,
	)
	if ret == 0 {
		procDeleteProcThreadAttribList.Call(uintptr(unsafe.Pointer(&attrList[0])))
		return nil, nil, fmt.Errorf("UpdateProcThreadAttribute: %w", err)
	}

	si := &startupInfoEx{}
	si.StartupInfo.Cb = uint32(unsafe.Sizeof(*si))
	si.lpAttributeList = uintptr(unsafe.Pointer(&attrList[0]))

	return si, unsafe.Pointer(&attrList[0]), nil
}

// startPTYPipeMode is the legacy pipe-based fallback for old Windows builds.
func startPTYPipeMode(cmd *exec.Cmd, cols, rows int) (*ptySession, error) {
	cmd.SysProcAttr = nil // clear any prior attrs

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start shell: %w", err)
	}

	return &ptySession{
		cmd:    cmd,
		stdin:  stdinPipe,
		stdout: io.MultiReader(stdoutPipe, stderrPipe),
		resize: nil,
		closer: func() error {
			return stdinPipe.Close()
		},
	}, nil
}
