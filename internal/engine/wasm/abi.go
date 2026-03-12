package wasm

import (
	"context"
	"log"

	"github.com/tetratelabs/wazero/api"
)

// HostModule is the name of the module that provides host functions to the WASM plugin.
const HostModule = "env"

// Context keys for passing data between the host and the WASM plugin.
type ctxKey struct{}

var (
	eventKey     = ctxKey{}
	droppedKey   = ctxKey{}
)

// InstantiateHostModule registers the Sovereign Terminal's ABI functions into the runtime.
func (pm *PluginManager) InstantiateHostModule(ctx context.Context) error {
	_, err := pm.runtime.NewHostModuleBuilder(HostModule).
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, ptr, size uint32) {
			// Host implementation of 'log' - for debugging from WASM
			buf, ok := mod.Memory().Read(ptr, size)
			if !ok {
				log.Printf("[WASM] Failed to read log memory")
				return
			}
			log.Printf("[WASM PLUGIN] %s", string(buf))
		}).
		Export("host_log").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module) {
			// Host implementation of 'drop_event'
			if dropped, ok := ctx.Value(droppedKey).(*bool); ok {
				*dropped = true
			}
		}).
		Export("drop_event").
		Instantiate(ctx)

	return err
}

// WithEvent attaches event data to the context for use in host functions.
func WithEvent(ctx context.Context, eventRaw string, dropped *bool) context.Context {
	ctx = context.WithValue(ctx, eventKey, eventRaw)
	ctx = context.WithValue(ctx, droppedKey, dropped)
	return ctx
}
