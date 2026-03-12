//go:build !windows

package services

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// startPTY creates a real PTY-backed session using creack/pty.
// This gives full terminal emulation: cursor movement, tab completion, colours,
// applications like htop/vim, and proper window resize support.
func startPTY(cmd *exec.Cmd, cols, rows int) (*ptySession, error) {
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		fmt.Sprintf("COLUMNS=%d", cols),
		fmt.Sprintf("LINES=%d", rows),
	)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
	if err != nil {
		return nil, fmt.Errorf("start pty: %w", err)
	}

	return &ptySession{
		cmd:    cmd,
		stdin:  ptmx, // PTY master fd is both read and write
		stdout: ptmx, // reads from the same fd
		resize: func(c, r int) error {
			return pty.Setsize(ptmx, &pty.Winsize{
				Cols: uint16(c),
				Rows: uint16(r),
			})
		},
		closer: func() error {
			return ptmx.Close()
		},
	}, nil
}
