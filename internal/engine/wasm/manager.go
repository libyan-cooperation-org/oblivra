package wasm

import (
	"context"
	"fmt"
	"sync"

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
	// Create a new runtime with a focus on non-preemptive isolation.
	rt := wazero.NewRuntime(ctx)

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
func (pm *PluginManager) ExecutePlugin(ctx context.Context, name string) error {
	pm.mu.RLock()
	compiled, ok := pm.plugins[name]
	pm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Instantiate the module (this creates a clean memory space)
	mod, err := pm.runtime.InstantiateModule(ctx, compiled, pm.config)
	if err != nil {
		return fmt.Errorf("failed to instantiate plugin %s: %w", name, err)
	}
	defer mod.Close(ctx)

	// In the future, we will call specific ABI functions here.
	return nil
}
