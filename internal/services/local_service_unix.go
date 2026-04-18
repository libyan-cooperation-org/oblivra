//go:build !windows
package services

import (
	"os"
	"os/exec"
)

func (s *LocalService) getShellCommand() *exec.Cmd {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	return exec.Command(shell, "-l")
}
