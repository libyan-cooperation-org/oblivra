package plugin_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/plugin"
)

func TestWasmSandbox_EndToEnd(t *testing.T) {
	// 1. Setup deterministic plugins directory
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("pwd: %v", err)
	}

	// We are currently running from internal/plugin, so root is ../..
	projectRoot := filepath.Join(pwd, "..", "..")
	sourcePluginDir := filepath.Join(projectRoot, "plugins", "example_wasm")

	// Ensure example_wasm built wasm file is present
	wasmPath := filepath.Join(sourcePluginDir, "main.wasm")
	if _, err := os.Stat(wasmPath); os.IsNotExist(err) {
		t.Skip("example_wasm plugin not compiled, skipping end-to-end test. Run 'GOOS=wasip1 GOARCH=wasm go build -o main.wasm main.go' in the plugin dir first.")
	}

	// 2. Mock discovering it globally
	log, _ := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: os.DevNull,
	})
	bus := eventbus.NewBus(log)

	registry := plugin.NewRegistry(filepath.Join(projectRoot, "plugins"), log, bus)

	if err := registry.Discover(); err != nil {
		t.Fatalf("Registry discovery failed: %v", err)
	}

	p, ok := registry.Get("oblivra.example.wasm")
	if !ok {
		// Log just the failing directory error explicitly
		data, _ := json.MarshalIndent(registry.GetAll(), "", "  ")
		t.Fatalf("Failed to load plugin! State of example_wasm dir: %s", string(data))
	}

	// 4. Activate it
	if err := registry.Activate(p.Manifest.ID); err != nil {
		t.Fatalf("Activation failed: %v", err)
	}

	pActive, ok := registry.Get("oblivra.example.wasm")
	if pActive.State != plugin.StateActive {
		t.Fatalf("Expected active state, got %v", pActive.State)
	}

	// Wait a moment for startup logs
	time.Sleep(100 * time.Millisecond)

	// 5. Trigger the custom WASM event handler (on_connect)
	bus.Publish(eventbus.EventConnectionOpened, "session-12345")

	// Wait a moment for the hook to dispatch and execute async inside Wazero
	time.Sleep(200 * time.Millisecond)

	// 6. Deactivate
	if err := registry.Deactivate(p.Manifest.ID); err != nil {
		t.Fatalf("Deactivation failed: %v", err)
	}
}
