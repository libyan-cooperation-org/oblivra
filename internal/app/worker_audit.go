package app

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
)

// startSecurityAudit silently tails the auth log looking for brute-force signatures
func (s *SSHService) startSecurityAudit(ctx context.Context, sessionID string, hostID string) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second): // Let discovery/telemetry pass first
	}

	session, ok := s.manager.Get(sessionID)
	if !ok {
		return
	}

	client := session.GetClient()
	if client == nil {
		return
	}

	// For standard debian/ubuntu we check /var/log/auth.log
	out, err := client.ExecuteCommand(`grep -i "failed password" /var/log/auth.log | tail -n 50`)
	if err != nil {
		// Possibly RHEL-based (secure) or no permission
		out, err = client.ExecuteCommand(`grep -i "failed password" /var/log/secure | tail -n 50`)
	}

	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return // No logs or insufficient permissions
	}

	lines := strings.Split(string(out), "\n")

	// Matcher expects (invalid )user (USERNAME) from (IP_ADDRESS)
	userIpRegex := regexp.MustCompile(`[Ff]ailed password for (?:invalid user )?(\S+) from (\S+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := userIpRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			user := matches[1]
			ip := matches[2]

			event := &database.HostEvent{
				HostID:    hostID,
				Timestamp: time.Now().Format(time.RFC3339),
				EventType: "failed_login",
				SourceIP:  ip,
				User:      user,
				RawLog:    line,
			}

			if s.siem != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				s.siem.InsertHostEvent(ctx, event)
				cancel()
			}
		}
	}
	s.log.Info("Security Audit completed for host %s", hostID)
	s.bus.Publish("siem.audit_completed", map[string]string{
		"host_id":    hostID,
		"session_id": sessionID,
	})
}
