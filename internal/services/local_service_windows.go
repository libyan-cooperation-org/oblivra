//go:build windows
package services

import (
	"os/exec"
	"golang.org/x/sys/windows"
)

func (s *LocalService) getShellCommand() *exec.Cmd {
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NoExit")
	cmd.SysProcAttr = &windows.SysProcAttr{
		HideWindow: true,
	}
	return cmd
}
