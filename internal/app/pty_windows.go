//go:build windows

package app

import (
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

// startPTY on Windows falls back to pipe-based I/O since creack/pty
// is not supported. The CREATE_NO_WINDOW flag ensures the console host
// is allocated invisibly for the Wails (windowsgui) subsystem.
func startPTY(cmd *exec.Cmd, cols, rows int) (*ptySession, error) { //nolint:unparam
	_, _ = cols, rows // PTY resize not supported on Windows pipe mode
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

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
		resize: nil, // resize is not supported in pipe mode
		closer: func() error {
			stdinPipe.Close()
			return nil
		},
	}, nil
}
