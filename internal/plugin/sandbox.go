package plugin

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	lua "github.com/yuin/gopher-lua"
)

// Sandbox interface provides an isolated execution environment for plugins
type Sandbox interface {
	Start() error
	Stop() error
	Call(function string, args ...interface{}) (interface{}, error)
	IsActive() bool
	Uptime() time.Duration
	CheckPermission(perm Permission) error
}

// LuaSandbox provides an isolated execution environment for Lua plugins
type LuaSandbox struct {
	mu        sync.Mutex
	manifest  *Manifest
	log       *logger.Logger
	state     *lua.LState
	bus       *eventbus.Bus
	active    bool
	startedAt time.Time
	cancelCtx context.CancelFunc // cancels the Lua execution context on Stop()
}

// NewLuaSandbox creates a new plugin sandbox
func NewLuaSandbox(manifest *Manifest, bus *eventbus.Bus, log *logger.Logger) *LuaSandbox {
	return &LuaSandbox{
		manifest: manifest,
		bus:      bus,
		log:      log.WithPrefix(fmt.Sprintf("plugin:%s", manifest.ID)),
	}
}

// Start initializes the Lua state and loads the main script
func (s *LuaSandbox) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return fmt.Errorf("sandbox already active")
	}

	s.log.Info("Starting Lua sandbox for %s v%s", s.manifest.Name, s.manifest.Version)

	s.state = lua.NewState()

	// 1. Load basic required libraries only
	s.loadRestrictedLibs()

	// 2. Register Oblivra global object (host functions)
	oblivraTable := s.state.NewTable()
	s.state.SetFuncs(oblivraTable, map[string]lua.LGFunction{
		"print": func(L *lua.LState) int {
			msg := L.ToString(1)
			s.log.Info("[Lua] %s", msg)
			return 0
		},
		"has_permission": func(L *lua.LState) int {
			perm := L.ToString(1)
			has := s.manifest.HasPermission(Permission(perm))
			L.Push(lua.LBool(has))
			return 1
		},
	})

	// Add UI API
	s.registerUIAPI(oblivraTable)

	// Make oblivra object read-only
	mt := s.state.NewTable()
	s.state.SetField(mt, "__newindex", s.state.NewFunction(func(L *lua.LState) int {
		L.RaiseError("oblivra object is read-only")
		return 0
	}))
	s.state.SetField(mt, "__metatable", lua.LBool(false)) // Prevent metatable manipulation
	s.state.SetMetatable(oblivraTable, mt)

	s.state.SetGlobal("oblivra", oblivraTable)

	// 3. Apply execution limits
	s.applyLimits()

	// Execute the main script
	if err := s.state.DoFile(s.manifest.Main); err != nil {
		s.state.Close()
		s.state = nil
		return fmt.Errorf("failed to load lua script %s: %w", s.manifest.Main, err)
	}

	s.startedAt = time.Now()
	s.active = true
	return nil
}

func (s *LuaSandbox) loadRestrictedLibs() {
	// Base libraries always allowed
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		s.state.Push(s.state.GetField(s.state.Get(lua.EnvironIndex), "require"))
		s.state.Push(lua.LString(lib.name))
		s.state.Call(1, 0)
	}

	// Remove dangerous globals from base
	s.state.SetGlobal("dofile", lua.LNil)
	s.state.SetGlobal("loadfile", lua.LNil)
	s.state.SetGlobal("module", lua.LNil)

	if s.manifest.HasPermission(PermFilesystem) {
		// Restricted IO would go here
	}
}

func (s *LuaSandbox) applyLimits() {
	// Store cancel so Stop() can release the context goroutine immediately
	// rather than waiting for the timeout to expire.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.manifest.TimeoutSec)*time.Second)
	s.cancelCtx = cancel
	s.state.SetContext(ctx)
}

func (s *LuaSandbox) registerStorageAPI(root *lua.LTable) {
	storage := s.state.NewTable()
	s.state.SetFuncs(storage, map[string]lua.LGFunction{
		"get_dir": func(L *lua.LState) int {
			// Scoped storage path: <app_data>/plugins/<id>/storage
			path := platform.DataDir() + "/plugins/" + s.manifest.ID + "/storage"
			_ = os.MkdirAll(path, 0755)
			L.Push(lua.LString(path))
			return 1
		},
	})
	s.state.SetField(root, "storage", storage)
}

func (s *LuaSandbox) registerUIAPI(root *lua.LTable) {
	uiTable := s.state.NewTable()
	s.state.SetFuncs(uiTable, map[string]lua.LGFunction{
		"register_panel": func(L *lua.LState) int {
			id := L.ToString(1)
			label := L.ToString(2)
			icon := L.ToString(3)

			s.log.Info("Plugin %s registering UI panel: %s", s.manifest.ID, label)
			s.bus.Publish("ui.register_panel", map[string]string{
				"plugin_id": s.manifest.ID,
				"panel_id":  id,
				"label":     label,
				"icon":      icon,
			})
			return 0
		},
		"add_status_icon": func(L *lua.LState) int {
			id := L.ToString(1)
			icon := L.ToString(2)
			tooltip := L.ToString(3)

			s.log.Info("Plugin %s adding status icon: %s", s.manifest.ID, id)
			s.bus.Publish("ui.add_status_icon", map[string]string{
				"plugin_id": s.manifest.ID,
				"icon_id":   id,
				"icon":      icon,
				"tooltip":   tooltip,
			})
			return 0
		},
	})
	s.state.SetField(root, "ui", uiTable)
}

// Stop terminates the plugin and releases all resources including the context goroutine.
func (s *LuaSandbox) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return nil
	}

	s.log.Info("Stopping sandbox for %s", s.manifest.Name)

	// Cancel the execution context to release the timeout goroutine
	if s.cancelCtx != nil {
		s.cancelCtx()
		s.cancelCtx = nil
	}

	if s.state != nil {
		s.state.Close()
		s.state = nil
	}
	s.active = false

	return nil
}

// Call invokes a Lua function from the script
func (s *LuaSandbox) Call(function string, args ...interface{}) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active || s.state == nil {
		return nil, fmt.Errorf("sandbox not active")
	}

	fn := s.state.GetGlobal(function)
	if fn.Type() != lua.LTFunction {
		return nil, fmt.Errorf("function %s not defined in the plugin", function)
	}

	var luaArgs []lua.LValue
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			luaArgs = append(luaArgs, lua.LString(v))
		case float64:
			luaArgs = append(luaArgs, lua.LNumber(v))
		case int:
			luaArgs = append(luaArgs, lua.LNumber(v))
		case bool:
			luaArgs = append(luaArgs, lua.LBool(v))
		default:
			luaArgs = append(luaArgs, lua.LNil)
		}
	}

	err := s.state.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, luaArgs...)

	if err != nil {
		s.log.Error("Lua execution error in %s: %v", function, err)
		return nil, err
	}

	// Return the result
	ret := s.state.Get(-1)
	s.state.Pop(1)

	return ret, nil
}

// IsActive returns whether the sandbox is running
func (s *LuaSandbox) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// Uptime returns how long the sandbox has been active
func (s *LuaSandbox) Uptime() time.Duration {
	if !s.active {
		return 0
	}
	return time.Since(s.startedAt)
}

// CheckPermission verifies the plugin has the required permission
func (s *LuaSandbox) CheckPermission(perm Permission) error {
	if !s.manifest.HasPermission(perm) {
		return fmt.Errorf("plugin %s does not have permission %s", s.manifest.ID, perm)
	}
	return nil
}
