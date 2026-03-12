package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/ssh"
)

type TelemetryWorker struct {
	manager          *ssh.SessionManager
	telemetryManager *monitoring.TelemetryManager
	log              *TelemetryWorkerLogger
}

type TelemetryWorkerLogger interface {
	Debug(format string, v ...interface{})
	Error(format string, v ...interface{})
}

func (s *SSHService) startTelemetryPolling(ctx context.Context, sessionID string) {
	ticker := time.NewTicker(s.telemetryPollingInterval())
	defer ticker.Stop()

	// Safety timeout — goroutine exits after 1 hour max
	maxLife := time.NewTimer(1 * time.Hour)
	defer maxLife.Stop()

	s.log.Debug("Starting telemetry polling for session: %s", sessionID)

	for {
		select {
		case <-ctx.Done():
			s.log.Debug("Telemetry polling context cancelled for session: %s", sessionID)
			return
		case <-maxLife.C:
			s.log.Debug("Telemetry polling max lifetime reached for session: %s", sessionID)
			return
		case <-ticker.C:
			session, ok := s.manager.Get(sessionID)
			if !ok {
				return // Session closed
			}

			// Compact script for sidecar-less telemetry
			// 1. LoadAvg 2. CPU Usage 3. Mem Total/Used 4. Disk Total/Used
			script := `cat /proc/loadavg | awk '{print $1}'; top -bn1 | grep "Cpu(s)" | awk '{print $2 + $4}'; free -m | grep "Mem:" | awk '{print $2 " " $3}'; df -m / | tail -1 | awk '{print $2 " " $3}'`

			output, err := session.GetClient().ExecuteCommand(script)
			if err != nil {
				s.log.Error("Telemetry failed for %s: %v", sessionID, err)
				continue
			}

			telemetry := s.parseTelemetry(string(output))
			telemetry.HostID = session.HostID
			if s.telemetryManager != nil {
				s.telemetryManager.UpdateHost(session.HostID, telemetry)
			}
		}
	}
}

func (s *SSHService) telemetryPollingInterval() time.Duration {
	return 10 * time.Second
}

func (s *SSHService) parseTelemetry(output string) monitoring.HostTelemetry {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	t := monitoring.HostTelemetry{}

	if len(lines) < 4 {
		return t
	}

	// 1. Load Average
	fmt.Sscanf(lines[0], "%f", &t.LoadAvg)

	// 2. CPU Usage
	fmt.Sscanf(lines[1], "%f", &t.CPUUsage)

	// 3. Memory (MB)
	fmt.Sscanf(lines[2], "%f %f", &t.MemTotalMB, &t.MemUsedMB)

	// 4. Disk (MB converted to GB)
	var diskTotalMB, diskUsedMB float64
	fmt.Sscanf(lines[3], "%f %f", &diskTotalMB, &diskUsedMB)
	t.DiskTotalGB = diskTotalMB / 1024.0
	t.DiskUsedGB = diskUsedMB / 1024.0

	return t
}
