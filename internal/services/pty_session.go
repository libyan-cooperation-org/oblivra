package services

import (
	"fmt"
	"io"
	"os/exec"
)

// ptySession abstracts platform-specific PTY / pipe-based terminal I/O.
// On Linux/macOS this uses creack/pty for true PTY support.
// On Windows this falls back to stdin/stdout/stderr pipes.
type ptySession struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.Reader
	// resize is non-nil on platforms with PTY support (unix).
	// It accepts (cols, rows) and returns an error.
	resize func(cols, rows int) error
	// closer is called to clean up PTY-specific resources (e.g. close pty fd).
	closer func() error
}

// startPTYPipeMode is the legacy fallback for cases where PTY allocation fails.
func startPTYPipeMode(cmd *exec.Cmd) (*ptySession, error) {
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("pipe: stdin: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("pipe: stdout: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("pipe: stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("pipe: start: %w", err)
	}
	return &ptySession{
		cmd:    cmd,
		stdin:  stdinPipe,
		stdout: io.MultiReader(stdoutPipe, stderrPipe),
		resize: nil,
		closer: func() error { return stdinPipe.Close() },
	}, nil
}
