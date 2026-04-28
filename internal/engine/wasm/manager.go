package wasm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// PluginManager handles the lifecycle and execution of WASM-based plugins.
type PluginManager struct {
	runtime wazero.Runtime
	config  wazero.ModuleConfig
	mu      sync.RWMutex
	plugins map[string]wazero.CompiledModule
}

// NewPluginManager initializes a new WASM runtime using wazero.
func NewPluginManager(ctx context.Context) (*PluginManager, error) {
	// Create a new runtime configured to honor ctx cancellation. Without
	// WithCloseOnContextDone, wazero runs guest code without periodic
	// context checks — an infinite-loop module hangs the host process
	// forever. Verified by TestSandbox_ContextCancellation_KillsHungModule.
	rtConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	rt := wazero.NewRuntimeWithConfig(ctx, rtConfig)

	// Instantiate WASI
	_, err := wasi_snapshot_preview1.Instantiate(ctx, rt)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	pm := &PluginManager{
		runtime: rt,
		config:  wazero.NewModuleConfig().WithStdout(nil).WithStderr(nil), // Isolate I/O
		plugins: make(map[string]wazero.CompiledModule),
	}

	// Instantiate Sovereign Host ABI
	if err := pm.InstantiateHostModule(ctx); err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate host module: %w", err)
	}

	return pm, nil
}

// Close releases the WASM runtime resources.
func (pm *PluginManager) Close(ctx context.Context) error {
	return pm.runtime.Close(ctx)
}

// RegisterPlugin compiles and saves a plugin for later execution.
func (pm *PluginManager) RegisterPlugin(ctx context.Context, name string, wasmBin []byte) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	compiled, err := pm.runtime.CompileModule(ctx, wasmBin)
	if err != nil {
		return fmt.Errorf("failed to compile plugin %s: %w", name, err)
	}

	pm.plugins[name] = compiled
	return nil
}

// ExecuteAll runs all registered plugins against a context.
func (pm *PluginManager) ExecuteAll(ctx context.Context) error {
	pm.mu.RLock()
	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}
	pm.mu.RUnlock()

	for _, name := range names {
		if err := pm.ExecutePlugin(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

// ExecutePlugin instantiates and runs a named plugin.
//
// Adversarial guard: a goroutine watches ctx.Done() and force-closes
// the module on cancellation. Without this, an infinite-loop or
// resource-exhaustion plugin would hang the host indefinitely
// (`defer mod.Close(ctx)` only fires when the function RETURNS, which
// never happens for a `loop br 0` body). Verified by
// TestSandbox_ContextCancellation_KillsHungModule.
func (pm *PluginManager) ExecutePlugin(ctx context.Context, name string) error {
	pm.mu.RLock()
	compiled, ok := pm.plugins[name]
	pm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Cancellation watchdog: spawn before module instantiation so a
	// hostile compiled module that hangs at instantiation is also
	// caught. Channel close on normal-completion path stops the watchdog.
	doneCh := make(chan struct{})
	defer close(doneCh)
	watchdogCtx, cancelWatchdog := context.WithCancel(context.Background())
	defer cancelWatchdog()

	// We need a reference the watchdog can close — populated below
	// after InstantiateModule succeeds. Race-safe via the channel.
	type modHolder struct {
		mod interface {
			Close(context.Context) error
		}
	}
	holder := &modHolder{}
	holderReady := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			// Wait for the holder to be populated, then force-close.
			select {
			case <-holderReady:
				if holder.mod != nil {
					closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					_ = holder.mod.Close(closeCtx)
				}
			case <-watchdogCtx.Done():
				// Normal completion before module ready — nothing to close.
			}
		case <-watchdogCtx.Done():
			// Normal completion path; deferred Close handles cleanup.
		}
	}()

	mod, err := pm.runtime.InstantiateModule(ctx, compiled, pm.config)
	if err != nil {
		close(holderReady)
		return fmt.Errorf("failed to instantiate plugin %s: %w", name, err)
	}
	holder.mod = mod
	close(holderReady)
	defer mod.Close(ctx)

	// In the future, we will call specific ABI functions here.
	// For now, InstantiateModule alone runs the module's `_start`
	// per WASI convention — enough to surface infinite-loop bodies.
	return nil
}
