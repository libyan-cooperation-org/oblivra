package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	lua "github.com/yuin/gopher-lua"
)

type PluginState string

const (
	StateInstalled PluginState = "installed"
	StateActive    PluginState = "active"
	StateDisabled  PluginState = "disabled"
	StateError     PluginState = "error"
)

type Plugin struct {
	Manifest *Manifest   `json:"manifest"`
	State    PluginState `json:"state"`
	Path     string      `json:"path"`
	Error    string      `json:"error,omitempty"`

	sandbox Sandbox // Internal reference to the active sandbox interface
}

type Registry struct {
	mu         sync.RWMutex
	pluginsDir string
	plugins    map[string]*Plugin
	log        *logger.Logger
	bus        *eventbus.Bus
}

func NewRegistry(pluginsDir string, log *logger.Logger, bus *eventbus.Bus) *Registry {
	r := &Registry{
		pluginsDir: pluginsDir,
		plugins:    make(map[string]*Plugin),
		log:        log,
		bus:        bus,
	}
	r.subscribeEvents()
	return r
}

func (r *Registry) subscribeEvents() {
	if r.bus == nil {
		return
	}

	r.bus.Subscribe(eventbus.EventConnectionOpened, func(event eventbus.Event) {
		sessionID, ok := event.Data.(string)
		if ok {
			r.dispatchHook("on_connect", lua.LString(sessionID))
		}
	})

	r.bus.Subscribe(eventbus.EventConnectionClosed, func(event eventbus.Event) {
		sessionID, ok := event.Data.(string)
		if ok {
			r.dispatchHook("on_disconnect", lua.LString(sessionID))
		}
	})
}

func (r *Registry) dispatchHook(funcName string, args ...lua.LValue) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if p.State == StateActive && p.sandbox != nil {
			if (p.Manifest.Hooks.OnConnect && funcName == "on_connect") ||
				(p.Manifest.Hooks.OnDisconnect && funcName == "on_disconnect") {

				var genericArgs []interface{}
				for _, arg := range args {
					if str, ok := arg.(lua.LString); ok {
						genericArgs = append(genericArgs, string(str))
					} else {
						genericArgs = append(genericArgs, arg.String())
					}
				}

				_, err := p.sandbox.Call(funcName, genericArgs...)
				if err != nil {
					r.log.Error("Plugin %s failed hook %s: %v", p.Manifest.ID, funcName, err)
				}
			}
		}
	}
}

// Discover scans the plugins directory for installed plugins.
// For each .wasm plugin, it verifies the Ed25519 signature before registering.
// Plugins that fail signature verification are marked StateError and never activated.
func (r *Registry) Discover() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := os.MkdirAll(r.pluginsDir, 0755)
	if err != nil {
		return fmt.Errorf("create plugins directory: %w", err)
	}

	entries, err := os.ReadDir(r.pluginsDir)
	if err != nil {
		return fmt.Errorf("read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(r.pluginsDir, entry.Name())
		manifest, err := LoadManifest(filepath.Join(pluginPath, "manifest.json"))
		if err != nil {
			r.plugins[entry.Name()] = &Plugin{
				State: StateError,
				Path:  pluginPath,
				Error: err.Error(),
			}
			continue
		}

		manifest.Main = filepath.Join(pluginPath, manifest.Main)

		// ── Ed25519 Signature Verification ────────────────────────────────────
		// Only enforce for WASM plugins when at least one trusted key is registered.
		// Lua scripts are sandboxed differently but should also be signed in production.
		if filepath.Ext(manifest.Main) == ".wasm" && IsTrustEnforced() {
			if err := VerifyManifestPlugin(manifest); err != nil {
				r.log.Error("[PLUGIN] SIGNATURE VERIFICATION FAILED for %s: %v — plugin will NOT be loaded",
					manifest.ID, err)
				r.plugins[manifest.ID] = &Plugin{
					Manifest: manifest,
					State:    StateError,
					Path:     pluginPath,
					Error:    fmt.Sprintf("signature verification failed: %v", err),
				}
				continue
			}
			r.log.Info("[PLUGIN] Signature verified for %s v%s", manifest.ID, manifest.Version)
		} else if filepath.Ext(manifest.Main) == ".wasm" {
			// No trusted keys registered — allow in dev mode, warn loudly in production
			r.log.Warn("[PLUGIN] WARNING: No trusted keys registered. Plugin %s loaded WITHOUT signature check. "+
				"Call plugin.AddTrustedKey() at startup to enforce trust.", manifest.ID)
		}

		r.plugins[manifest.ID] = &Plugin{
			Manifest: manifest,
			State:    StateInstalled,
			Path:     pluginPath,
		}
	}

	return nil
}

func (r *Registry) GetAll() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		list = append(list, p)
	}
	return list
}

func (r *Registry) Get(id string) (*Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[id]
	return p, ok
}

// Activate instantiates the sandbox for a plugin and starts it.
// Refuses to activate any plugin in StateError (failed signature, bad manifest, etc.)
func (r *Registry) Activate(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return fmt.Errorf("plugin not found: %s", id)
	}

	if p.State == StateError {
		return fmt.Errorf("plugin %s is in error state and cannot be activated: %s", id, p.Error)
	}

	if p.State == StateActive {
		return nil
	}

	var sandbox Sandbox
	if filepath.Ext(p.Manifest.Main) == ".wasm" {
		sandbox = NewWasmSandbox(p.Manifest, r.bus, r.log)
	} else {
		sandbox = NewLuaSandbox(p.Manifest, r.bus, r.log)
	}

	if err := sandbox.Start(); err != nil {
		p.State = StateError
		p.Error = fmt.Sprintf("failed to start sandbox: %v", err)
		return err
	}

	p.sandbox = sandbox
	p.State = StateActive
	p.Error = ""
	return nil
}

// Deactivate destroys the sandbox for a plugin.
func (r *Registry) Deactivate(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return fmt.Errorf("plugin not found: %s", id)
	}

	if p.sandbox != nil {
		_ = p.sandbox.Stop()
		p.sandbox = nil
	}

	p.State = StateDisabled
	return nil
}
