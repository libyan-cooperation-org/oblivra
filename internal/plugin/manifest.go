package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// PluginType defines the kind of plugin
type PluginType string

const (
	PluginTypeTerminal   PluginType = "terminal"
	PluginTypeAutomation PluginType = "automation"
	PluginTypeUI         PluginType = "ui"
	PluginTypeSecurity   PluginType = "security"
)

// Permission defines what a plugin can access
type Permission string

const (
	PermSSHRead       Permission = "ssh.read"
	PermSSHWrite      Permission = "ssh.write"
	PermSSHConnect    Permission = "ssh.connect"
	PermVaultRead     Permission = "vault.read"
	PermFilesystem    Permission = "filesystem"
	PermNetwork       Permission = "network"
	PermUIRender      Permission = "ui.render"
	PermEvents        Permission = "events"
	PermNotifications Permission = "notifications"
)

// Manifest describes a plugin's metadata and requirements
type Manifest struct {
	// Identity
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	License     string `json:"license"`
	Homepage    string `json:"homepage,omitempty"`
	Repository  string `json:"repository,omitempty"`

	// Type and capabilities
	Type        PluginType   `json:"type"`
	Permissions []Permission `json:"permissions"`

	// Entry points
	Main    string `json:"main"`               // WASM file path
	UIEntry string `json:"ui_entry,omitempty"` // Frontend script

	// Resources
	MaxMemoryMB int `json:"max_memory_mb"`
	MaxCPUPct   int `json:"max_cpu_pct"`
	TimeoutSec  int `json:"timeout_sec"`

	// Hooks
	Hooks PluginHooks `json:"hooks,omitempty"`

	// Dependencies
	MinAppVersion string   `json:"min_app_version"`
	Dependencies  []string `json:"dependencies,omitempty"`
}

type PluginHooks struct {
	OnConnect    bool `json:"on_connect"`
	OnDisconnect bool `json:"on_disconnect"`
	OnCommand    bool `json:"on_command"`
	OnOutput     bool `json:"on_output"`
	OnStartup    bool `json:"on_startup"`
	OnShutdown   bool `json:"on_shutdown"`
}

// LoadManifest reads a plugin manifest file
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("validate manifest: %w", err)
	}

	return &manifest, nil
}

// validPluginID matches safe plugin identifiers: alphanumeric, hyphens, underscores, dots.
// Rejects path separators, spaces, and all characters that could enable directory traversal (SA-04).
var validPluginID = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,64}$`)

// Validate checks the manifest for required fields
func (m *Manifest) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("missing plugin ID")
	}
	// SA-04: reject IDs containing path traversal sequences or invalid characters.
	if !validPluginID.MatchString(m.ID) {
		return fmt.Errorf("invalid plugin ID %q: must match [a-zA-Z0-9._-]{1,64}", m.ID)
	}
	// Belt-and-suspenders: filepath.Clean + filepath.Base must not alter the ID.
	if cleaned := filepath.Base(filepath.Clean(m.ID)); cleaned != m.ID {
		return fmt.Errorf("invalid plugin ID %q: must not contain path separators", m.ID)
	}
	if m.Name == "" {
		return fmt.Errorf("missing plugin name")
	}
	if m.Version == "" {
		return fmt.Errorf("missing version")
	}
	if m.Main == "" {
		return fmt.Errorf("missing main entry point")
	}
	if m.Type == "" {
		return fmt.Errorf("missing plugin type")
	}

	// Validate resource limits
	if m.MaxMemoryMB <= 0 {
		m.MaxMemoryMB = 64 // default 64MB
	}
	if m.MaxMemoryMB > 512 {
		return fmt.Errorf("max memory cannot exceed 512MB")
	}
	if m.MaxCPUPct <= 0 {
		m.MaxCPUPct = 25
	}
	if m.TimeoutSec <= 0 {
		m.TimeoutSec = 30
	}

	return nil
}

// HasPermission checks if the plugin has a specific permission
func (m *Manifest) HasPermission(perm Permission) bool {
	for _, p := range m.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}
