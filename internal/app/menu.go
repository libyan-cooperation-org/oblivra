//go:build !server

package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// BuildApplicationMenu is a method on App so callers can reach it via the
// existing `oblivraApp` wiring in main_gui.go. The implementation is
// stateless — it constructs the menu from scratch each time, which is
// fine because Wails only calls it once at startup.
func (a *App) BuildApplicationMenu() *application.Menu {
	return buildApplicationMenu()
}

// buildApplicationMenu returns the OBLIVRA application menu used by the
// Wails desktop window. Items that should drive frontend behaviour
// (navigate to a route, pop out a panel, toggle a UI surface) emit a
// `menu:<action>` event via app.Event.Emit; the App.svelte onMount
// subscribes via the Wails runtime and dispatches to the right handler.
//
// Native roles (Reload, Cut/Copy/Paste, Minimise, etc.) are handled
// entirely by the OS — no event round-trip needed.
func buildApplicationMenu() *application.Menu {
	menu := application.NewMenu()

	// macOS App menu (About/Hide/Quit) is auto-injected by Wails on Darwin
	// when a *Menu is bound; we don't need to manually splice NewAppMenu()
	// in here. Earlier code did and it was a no-op — removed in audit.

	// ── File ──────────────────────────────────────────────────────────────
	file := menu.AddSubmenu("File")
	file.Add("New Local Terminal").
		SetAccelerator("CmdOrCtrl+T").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:new-terminal", nil)
		})
	file.Add("Pop Out Current Page").
		SetAccelerator("CmdOrCtrl+Shift+O").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:popout-current", nil)
		})
	file.AddSeparator()
	file.Add("Settings").
		SetAccelerator("CmdOrCtrl+,").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:goto", "/settings")
		})
	file.AddSeparator()
	file.AddRole(application.Quit)

	// ── Edit ──────────────────────────────────────────────────────────────
	// Standard cut/copy/paste with platform-correct accelerators.
	edit := menu.AddSubmenu("Edit")
	edit.AddRole(application.Undo)
	edit.AddRole(application.Redo)
	edit.AddSeparator()
	edit.AddRole(application.Cut)
	edit.AddRole(application.Copy)
	edit.AddRole(application.Paste)
	edit.AddRole(application.SelectAll)

	// ── View ──────────────────────────────────────────────────────────────
	view := menu.AddSubmenu("View")
	view.Add("Toggle Sidebar").
		SetAccelerator("CmdOrCtrl+B").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:toggle-sidebar", nil)
		})
	view.Add("Command Palette").
		SetAccelerator("CmdOrCtrl+K").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:command-palette", nil)
		})
	view.AddSeparator()
	view.AddRole(application.ToggleFullscreen)
	view.AddRole(application.ResetZoom)
	view.AddRole(application.ZoomIn)
	view.AddRole(application.ZoomOut)
	view.AddSeparator()
	view.AddRole(application.Reload)

	// ── Navigate ──────────────────────────────────────────────────────────
	// Quick-jump to high-traffic routes. Mirrors the G+letter keymap from
	// App.svelte but exposes it as a discoverable menu surface.
	nav := menu.AddSubmenu("Navigate")
	nav.Add("Dashboard").
		SetAccelerator("CmdOrCtrl+1").
		OnClick(func(_ *application.Context) { emitMenuEvent("menu:goto", "/") })
	nav.Add("SIEM Search").
		SetAccelerator("CmdOrCtrl+2").
		OnClick(func(_ *application.Context) { emitMenuEvent("menu:goto", "/siem-search") })
	nav.Add("Alerts").
		SetAccelerator("CmdOrCtrl+3").
		OnClick(func(_ *application.Context) { emitMenuEvent("menu:goto", "/alerts") })
	nav.Add("Fleet").
		SetAccelerator("CmdOrCtrl+4").
		OnClick(func(_ *application.Context) { emitMenuEvent("menu:goto", "/fleet") })
	nav.Add("Terminal").
		SetAccelerator("CmdOrCtrl+5").
		OnClick(func(_ *application.Context) { emitMenuEvent("menu:goto", "/terminal") })
	nav.AddSeparator()
	nav.Add("Operator Mode").
		OnClick(func(_ *application.Context) { emitMenuEvent("menu:goto", "/operator") })
	nav.Add("Threat Hunter").
		OnClick(func(_ *application.Context) { emitMenuEvent("menu:goto", "/threat-hunter") })
	nav.Add("Investigation").
		OnClick(func(_ *application.Context) { emitMenuEvent("menu:goto", "/investigation") })
	nav.Add("Evidence Vault").
		OnClick(func(_ *application.Context) { emitMenuEvent("menu:goto", "/evidence") })

	// ── Window ────────────────────────────────────────────────────────────
	wnd := menu.AddSubmenu("Window")
	wnd.AddRole(application.Minimise)
	wnd.AddRole(application.Zoom)
	wnd.AddSeparator()
	wnd.Add("Close All Pop-Outs").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:close-popouts", nil)
		})
	wnd.Add("Save Workspace").
		SetAccelerator("CmdOrCtrl+Shift+S").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:save-workspace", nil)
		})
	wnd.Add("Restore Workspace").
		SetAccelerator("CmdOrCtrl+Shift+R").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:restore-workspace", nil)
		})

	// ── Help ──────────────────────────────────────────────────────────────
	help := menu.AddSubmenu("Help")
	help.Add("Keyboard Shortcuts").
		SetAccelerator("CmdOrCtrl+/").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:goto", "/shortcuts")
		})
	help.Add("Documentation").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:open-url", "https://github.com/libyan-cooperation-org/oblivra")
		})
	help.Add("Diagnostics").
		OnClick(func(_ *application.Context) {
			emitMenuEvent("menu:diagnostics", nil)
		})
	help.AddSeparator()
	help.Add("About OBLIVRA").
		OnClick(func(_ *application.Context) {
			if a := application.Get(); a != nil {
				a.Menu.ShowAbout()
			}
		})

	return menu
}

// emitMenuEvent fires a menu-driven action over the Wails event bus so
// App.svelte's onMount listeners can react. Centralising this lets us
// keep the menu definition pure-data and put guard logic (nil-check on
// the application) in one place.
func emitMenuEvent(name string, payload any) {
	a := application.Get()
	if a == nil {
		return
	}
	if payload == nil {
		a.Event.Emit(name)
	} else {
		a.Event.Emit(name, payload)
	}
}
