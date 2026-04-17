package vault

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// EnsureDaemonRunning checks if the vault socket exists; if not, it attempts to spawn
// the oblivra-vault daemon process.
func EnsureDaemonRunning(socketPath string, log *logger.Logger) error {
	if _, err := os.Stat(socketPath); err == nil {
		return nil // Already running
	}

	// Determine binary name
	binName := "oblivra-vault"
	if runtime.GOOS == "windows" {
		binName = "oblivra-vault.exe"
	}

	// Try to find the binary in the same directory as the current process
	self, _ := os.Executable()
	binPath := binName
	if self != "" {
		binPath = fmt.Sprintf("%s/%s", (self), binName)
	}

	log.Info("[VAULT] Spawning isolated vault daemon: %s", binPath)
	
	cmd := exec.Command(binPath, "-socket", socketPath, "-ppid", fmt.Sprintf("%d", os.Getpid()))
	// We don't wait for it to finish; it's a daemon
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start vault daemon: %w", err)
	}

	// Give it a second to create the socket
	for i := 0; i < 10; i++ {
		time.Sleep(200 * time.Millisecond)
		if _, err := os.Stat(socketPath); err == nil {
			log.Info("[VAULT] Isolated daemon active on %s", socketPath)
			return nil
		}
	}

	return fmt.Errorf("vault daemon started but socket %s was not created in time", socketPath)
}
