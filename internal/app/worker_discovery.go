package app

import (
	"context"
	"strings"
	"time"
)

type StackProbe struct {
	Tag          string
	Command      string
	MatchPattern string
}

// startStackDiscovery probes the remote host for common technologies and automatically tags the database entry
func (s *SSHService) startStackDiscovery(ctx context.Context, sessionID string, hostID string) {
	// Give the initial interactive shell a moment to breathe
	select {
	case <-ctx.Done():
		return
	case <-time.After(3 * time.Second):
	}

	session, ok := s.manager.Get(sessionID)
	if !ok {
		return
	}

	client := session.GetClient()
	if client == nil {
		return
	}

	probes := []StackProbe{
		// Infrastructure
		{"docker", "docker info", "Containers:"},
		{"kubernetes", "kubectl version --client", "client version"},

		// Web Servers
		{"nginx", "nginx -v", "nginx"},
		{"apache", "apache2 -v", "apache"},

		// Databases
		{"postgresql", "psql -V", "postgresql"},
		{"mysql", "mysql -V", "mysql"},
		{"redis", "redis-cli -v", "redis"},
		{"mongodb", "mongod --version", "mongodb"},

		// Languages & Runtimes
		{"node", "node -v", "v"},
		{"python", "python3 --version", "python"},
		{"java", "java -version", "java"},
		{"go", "go version", "go"},
		{"php", "php -v", "php"},
		{"ruby", "ruby -v", "ruby"},
	}

	var discoveredTags []string

	for _, probe := range probes {
		out, err := client.ExecuteCommand(probe.Command)
		if err == nil && strings.Contains(strings.ToLower(string(out)), strings.ToLower(probe.MatchPattern)) {
			discoveredTags = append(discoveredTags, probe.Tag)
		}
	}

	if len(discoveredTags) > 0 {
		s.log.Info("Stack Discovery found tags for host %s: %v", hostID, discoveredTags)

		host, err := s.hosts.GetByID(context.Background(), hostID)
		if err != nil {
			s.log.Error("Discovery failed to fetch host %s: %v", hostID, err)
			return
		}

		// Merge without duplicates
		existing := make(map[string]bool)
		for _, tag := range host.Tags {
			existing[strings.ToLower(tag)] = true
		}

		for _, newTag := range discoveredTags {
			if !existing[strings.ToLower(newTag)] {
				host.Tags = append(host.Tags, newTag)
			}
		}

		// Update DB and refresh UI
		if err := s.hosts.Update(context.Background(), host); err == nil {
			s.bus.Publish("hosts_updated", nil)
		} else {
			s.log.Error("Stack Discovery failed to update host %s tags: %v", hostID, err)
		}
	}
}
