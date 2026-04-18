package plugin

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WasmSandbox implements Sandbox for WebAssembly modules using wazero
type WasmSandbox struct {
	mu        sync.Mutex
	manifest  *Manifest
	log       *logger.Logger
	bus       *eventbus.Bus
	active    bool
	startedAt time.Time

	runtime wazero.Runtime
	mod     api.Module
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewWasmSandbox prepares a WebAssembly environment
func NewWasmSandbox(manifest *Manifest, bus *eventbus.Bus, log *logger.Logger) *WasmSandbox {
	return &WasmSandbox{
		manifest: manifest,
		bus:      bus,
		log:      log.WithPrefix(fmt.Sprintf("wasm:%s", manifest.ID)),
	}
}

// Start loads the wasm byte code and instantiates the module
func (s *WasmSandbox) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return fmt.Errorf("sandbox already active")
	}

	s.log.Info("Starting WASM sandbox for %s v%s", s.manifest.Name, s.manifest.Version)

	wasmBytes, err := os.ReadFile(s.manifest.Main)
	if err != nil {
		return fmt.Errorf("failed to read wasm file: %w", err)
	}

	// Create request-scoped context with timeout limits
	s.ctx, s.cancel = context.WithTimeout(context.Background(), time.Duration(s.manifest.TimeoutSec)*time.Second)

	s.runtime = wazero.NewRuntime(s.ctx)

	// Setup WASI preview 1 (needed for Go wasip1 target)
	wasi_snapshot_preview1.MustInstantiate(s.ctx, s.runtime)

	// Register Oblivra Host Functions
	env := s.runtime.NewHostModuleBuilder("env")
	env.NewFunctionBuilder().
		WithFunc(s.hostPrint).
		WithName("oblivra_print").
		Export("oblivra_print")

	env.NewFunctionBuilder().
		WithFunc(s.hostHasPermission).
		WithName("oblivra_has_permission").
		Export("oblivra_has_permission")

	_, err = env.Instantiate(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to instantiate host module: %w", err)
	}

	config := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr) // Allow module panic debugging

	s.mod, err = s.runtime.InstantiateWithConfig(s.ctx, wasmBytes, config)
	if err != nil {
		// If the module's "main" returns, it might be an exit error which isn't always fatal in WASI
		s.log.Warn("WASM instantiation finished: %v", err)
	}

	s.startedAt = time.Now()
	s.active = true
	return nil
}

// Stop cleanly terminates the runtime
func (s *WasmSandbox) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return nil
	}

	s.log.Info("Stopping WASM sandbox for %s", s.manifest.Name)
	if s.cancel != nil {
		s.cancel()
	}
	if s.mod != nil {
		s.mod.Close(context.Background())
	}
	if s.runtime != nil {
		s.runtime.Close(context.Background())
	}

	s.active = false
	return nil
}

// Call interacts with an exported WASM function
func (s *WasmSandbox) Call(function string, args ...interface{}) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active || s.mod == nil {
		return nil, fmt.Errorf("wasm sandbox not active")
	}

	fn := s.mod.ExportedFunction(function)
	if fn == nil {
		return nil, fmt.Errorf("function '%s' not exported by wasm module", function)
	}

	// wazero function calls expect specific uint64 encodings depending on parameter types.
	// We'll perform very basic mappings for hooks.
	var wasmArgs []uint64
	// (String passing into WASM requires memory management, allocating pointers manually,
	// so for simplicity in this example we will just pass lengths or integers if needed,
	// but advanced string passing requires writing to mod.Memory() and passing the offset).
	for _, arg := range args {
		wArg, ok := arg.(uint64)
		if ok {
			wasmArgs = append(wasmArgs, wArg)
		}
	}

	res, err := fn.Call(s.ctx, wasmArgs...)
	if err != nil {
		return nil, err
	}

	if len(res) > 0 {
		return res[0], nil
	}

	return nil, nil
}

func (s *WasmSandbox) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

func (s *WasmSandbox) Uptime() time.Duration {
	if !s.active {
		return 0
	}
	return time.Since(s.startedAt)
}

func (s *WasmSandbox) CheckPermission(perm Permission) error {
	if !s.manifest.HasPermission(perm) {
		return fmt.Errorf("plugin %s does not have permission %s", s.manifest.ID, perm)
	}
	return nil
}

// ----------------------------------------------------------------------------
// Host Function Implementations (called from inside WASM)
// ----------------------------------------------------------------------------

func (s *WasmSandbox) hostPrint(ctx context.Context, m api.Module, ptr uint32, len uint32) {
	if bytes, ok := m.Memory().Read(ptr, len); ok {
		s.log.Info("[WASM] %s", string(bytes))
	}
}

func (s *WasmSandbox) hostHasPermission(ctx context.Context, m api.Module, ptr uint32, len uint32) uint32 {
	if bytes, ok := m.Memory().Read(ptr, len); ok {
		perm := Permission(string(bytes))
		if s.manifest.HasPermission(perm) {
			return 1
		}
	}
	return 0
}
