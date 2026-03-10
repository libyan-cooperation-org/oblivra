package ssh

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SSHConfigEntry represents a parsed SSH config host entry
type SSHConfigEntry struct {
	Alias          string   `json:"alias"`
	Hostname       string   `json:"hostname"`
	Port           int      `json:"port"`
	User           string   `json:"user"`
	IdentityFile   string   `json:"identity_file"`
	ProxyJump      string   `json:"proxy_jump"`
	ProxyCommand   string   `json:"proxy_command"`
	ForwardAgent   bool     `json:"forward_agent"`
	LocalForwards  []string `json:"local_forwards"`
	RemoteForwards []string `json:"remote_forwards"`
}

// ParseSSHConfig reads and parses ~/.ssh/config
func ParseSSHConfig() ([]SSHConfigEntry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	return ParseSSHConfigFile(filepath.Join(home, ".ssh", "config"))
}

// ParseSSHConfigFile parses an SSH config file at the given path
func ParseSSHConfigFile(path string) ([]SSHConfigEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SSHConfigEntry{}, nil
		}
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var entries []SSHConfigEntry
	var current *SSHConfigEntry

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on space or =
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			parts = strings.SplitN(line, "=", 2)
		}
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(strings.Trim(parts[1], `"'`))

		switch key {
		case "host":
			// Skip wildcard patterns
			if strings.ContainsAny(value, "*?") {
				current = nil
				continue
			}
			entry := SSHConfigEntry{Alias: value, Port: 22}
			current = &entry
			entries = append(entries, entry)

		case "hostname":
			if current != nil {
				entries[len(entries)-1].Hostname = value
			}

		case "port":
			if current != nil {
				if p, err := strconv.Atoi(value); err == nil {
					entries[len(entries)-1].Port = p
				}
			}

		case "user":
			if current != nil {
				entries[len(entries)-1].User = value
			}

		case "identityfile":
			if current != nil {
				if strings.HasPrefix(value, "~/") {
					home, _ := os.UserHomeDir()
					value = filepath.Join(home, value[2:])
				}
				entries[len(entries)-1].IdentityFile = value
			}

		case "proxyjump":
			if current != nil {
				entries[len(entries)-1].ProxyJump = value
			}

		case "proxycommand":
			if current != nil {
				entries[len(entries)-1].ProxyCommand = value
			}

		case "forwardagent":
			if current != nil {
				entries[len(entries)-1].ForwardAgent = strings.ToLower(value) == "yes"
			}

		case "localforward":
			if current != nil {
				entries[len(entries)-1].LocalForwards = append(
					entries[len(entries)-1].LocalForwards, value,
				)
			}

		case "remoteforward":
			if current != nil {
				entries[len(entries)-1].RemoteForwards = append(
					entries[len(entries)-1].RemoteForwards, value,
				)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan config: %w", err)
	}

	return entries, nil
}
