package app

import (
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
