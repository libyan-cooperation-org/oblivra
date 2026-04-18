package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DiscoveredHost represents a host found automatically
type DiscoveredHost struct {
	ID       string            `json:"id"`
	Source   string            `json:"source"` // "ssh_config", "terraform", "ansible", "known_hosts"
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Username string            `json:"username"`
	KeyPath  string            `json:"key_path,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// DiscoverySource defines an interface for finding hosts
type DiscoverySource interface {
	Name() string
	Discover(ctx context.Context) ([]DiscoveredHost, error)
}

// DiscoveryManager orchestrates host discovery
type DiscoveryManager struct {
	mu      sync.RWMutex
	sources []DiscoverySource
}

func NewDiscoveryManager() *DiscoveryManager {
	dm := &DiscoveryManager{}

	// Register default sources
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		dm.sources = append(dm.sources, &SSHConfigSource{
			configPath: filepath.Join(homeDir, ".ssh", "config"),
		})
		dm.sources = append(dm.sources, &KnownHostsSource{
			knownHostsPath: filepath.Join(homeDir, ".ssh", "known_hosts"),
		})
	}

	dm.sources = append(dm.sources, &TerraformSource{})
	dm.sources = append(dm.sources, &AnsibleSource{})

	return dm
}

// DiscoverAll runs all discovery sources and aggregates results
func (dm *DiscoveryManager) DiscoverAll(ctx context.Context) ([]DiscoveredHost, error) {
	dm.mu.RLock()
	sources := dm.sources
	dm.mu.RUnlock()

	var allHosts []DiscoveredHost
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, source := range sources {
		wg.Add(1)
		go func(src DiscoverySource) {
			defer wg.Done()

			// Give each source up to 5 seconds
			srcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			hosts, err := src.Discover(srcCtx)
			if err == nil && len(hosts) > 0 {
				mu.Lock()
				allHosts = append(allHosts, hosts...)
				mu.Unlock()
			}
		}(source)
	}

	wg.Wait()

	// Dedupe by address string
	return dedupeHosts(allHosts), nil
}

func dedupeHosts(hosts []DiscoveredHost) []DiscoveredHost {
	seen := make(map[string]bool)
	var deduped []DiscoveredHost

	for _, h := range hosts {
		if !seen[h.Address] && h.Address != "localhost" {
			seen[h.Address] = true
			deduped = append(deduped, h)
		}
	}
	return deduped
}

// SSHConfigSource parses ~/.ssh/config
type SSHConfigSource struct {
	configPath string
}

func (s *SSHConfigSource) Name() string { return "ssh_config" }

func (s *SSHConfigSource) Discover(ctx context.Context) ([]DiscoveredHost, error) {
	file, err := os.Open(s.configPath)
	if err != nil {
		return nil, nil // Not an error if file doesn't exist
	}
	defer file.Close()

	var hosts []DiscoveredHost
	var currentHost *DiscoveredHost

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := parts[1]

		if key == "host" && !strings.Contains(value, "*") && !strings.Contains(value, "?") {
			if currentHost != nil && currentHost.Address != "" {
				hosts = append(hosts, *currentHost)
			}
			currentHost = &DiscoveredHost{
				ID:     fmt.Sprintf("ssh-cfg-%d", time.Now().UnixNano()),
				Source: s.Name(),
				Name:   value,
				Port:   22, // default
			}
		} else if currentHost != nil {
			switch key {
			case "hostname":
				currentHost.Address = value
			case "user":
				currentHost.Username = value
			case "port":
				fmt.Sscanf(value, "%d", &currentHost.Port)
			case "identityfile":
				currentHost.KeyPath = strings.ReplaceAll(value, "~", os.Getenv("HOME"))
			}
		}
	}

	if currentHost != nil && currentHost.Address != "" {
		hosts = append(hosts, *currentHost)
	}

	return hosts, nil
}

// KnownHostsSource parses ~/.ssh/known_hosts
type KnownHostsSource struct {
	knownHostsPath string
}

func (s *KnownHostsSource) Name() string { return "known_hosts" }

func (s *KnownHostsSource) Discover(ctx context.Context) ([]DiscoveredHost, error) {
	file, err := os.Open(s.knownHostsPath)
	if err != nil {
		return nil, nil
	}
	defer file.Close()

	var hosts []DiscoveredHost
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// First part is usually hostnames/IPs
		hostPart := strings.Split(parts[0], ",")[0]

		// Remove bracketed ports e.g. [192.168.1.1]:2222
		host := hostPart
		if strings.HasPrefix(host, "[") {
			endIdx := strings.Index(host, "]")
			if endIdx != -1 {
				host = host[1:endIdx]
			}
		}

		// Skip hashed hosts
		if strings.HasPrefix(host, "|1|") {
			continue
		}

		hosts = append(hosts, DiscoveredHost{
			ID:      fmt.Sprintf("known-%d", time.Now().UnixNano()),
			Source:  s.Name(),
			Name:    host,
			Address: host,
			Port:    22,
		})
	}

	return hosts, nil
}

// TerraformSource finds IP addresses in local terraform state
type TerraformSource struct{}

func (s *TerraformSource) Name() string { return "terraform" }

func (s *TerraformSource) Discover(ctx context.Context) ([]DiscoveredHost, error) {
	// Look for terraform.tfstate in current or adjacent dirs
	// This is simplified; a real implementation might use `terraform show -json`
	cmd := exec.CommandContext(ctx, "terraform", "show", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil // Terraform not installed or no state
	}

	var state struct {
		Values struct {
			RootModule struct {
				Resources []struct {
					Type   string                 `json:"type"`
					Name   string                 `json:"name"`
					Values map[string]interface{} `json:"values"`
				} `json:"resources"`
			} `json:"root_module"`
		} `json:"values"`
	}

	if err := json.Unmarshal(output, &state); err != nil {
		return nil, nil
	}

	var hosts []DiscoveredHost
	for _, res := range state.Values.RootModule.Resources {
		// Look for common compute instances
		if strings.Contains(res.Type, "instance") || strings.Contains(res.Type, "droplet") {
			ip := extractIP(res.Values)
			if ip != "" {
				hosts = append(hosts, DiscoveredHost{
					ID:      fmt.Sprintf("tf-%d", time.Now().UnixNano()),
					Source:  s.Name(),
					Name:    res.Name,
					Address: ip,
					Port:    22,
					Tags:    map[string]string{"type": res.Type},
				})
			}
		}
	}

	return hosts, nil
}

func extractIP(values map[string]interface{}) string {
	keys := []string{"public_ip", "ipv4_address", "network_interface.0.nat_ip_address"}
	for _, k := range keys {
		if val, ok := values[k].(string); ok && val != "" {
			return val
		}
	}
	return ""
}

// AnsibleSource parses ansible inventories (simplified)
type AnsibleSource struct{}

func (s *AnsibleSource) Name() string { return "ansible" }

func (s *AnsibleSource) Discover(ctx context.Context) ([]DiscoveredHost, error) {
	cmd := exec.CommandContext(ctx, "ansible-inventory", "--list", "--export")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var inventory map[string]interface{}
	if err := json.Unmarshal(output, &inventory); err != nil {
		return nil, nil
	}

	var hosts []DiscoveredHost

	// Complex parsing needed for actual ansible output, simplified here
	if meta, ok := inventory["_meta"].(map[string]interface{}); ok {
		if hostvars, ok := meta["hostvars"].(map[string]interface{}); ok {
			for hostname, varsRaw := range hostvars {
				vars, ok := varsRaw.(map[string]interface{})
				if !ok {
					continue
				}

				addr := hostname
				if ip, ok := vars["ansible_host"].(string); ok {
					addr = ip
				}

				user := ""
				if u, ok := vars["ansible_user"].(string); ok {
					user = u
				}

				hosts = append(hosts, DiscoveredHost{
					ID:       fmt.Sprintf("ansible-%d", time.Now().UnixNano()),
					Source:   s.Name(),
					Name:     hostname,
					Address:  addr,
					Username: user,
					Port:     22,
				})
			}
		}
	}

	return hosts, nil
}
