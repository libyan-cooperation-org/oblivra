//go:build !server

package app

import (
	_ "embed"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// trayIcon is the OBLIVRA brand mark — same source as the window icon, baked
// into the binary so the tray works in air-gap deployments without external
// asset paths.
//
//go:embed appicon.png
var trayIcon []byte

// SetupSystemTray attaches the OBLIVRA tray icon to the OS taskbar and
// wires its menu. The tray menu mirrors the most common SOC operator
// actions (show window / open SIEM / open alerts / quit) so the operator
// can drive the platform without ever bringing the main window forward —
// critical for ops-room ambient awareness on a shared monitor.
//
// Click on the tray icon: toggle the main window (default Wails behaviour).
// Right-click: show the menu.
//
// Each menu item that drives frontend behaviour emits a `tray:<action>`
// event. App.svelte listens and dispatches to the right handler.
func (a *App) SetupSystemTray() {
	w := application.Get()
	if w == nil {
		return
	}

	tray := w.SystemTray.New()
	tray.SetTooltip("OBLIVRA — Sovereign Security Platform")
	if len(trayIcon) > 0 {
		tray.SetIcon(trayIcon)
	}

	menu := application.NewMenu()

	menu.Add("Show OBLIVRA").OnClick(func(_ *application.Context) {
		tray.ShowWindow()
		emitMenuEvent("tray:show", nil)
	})
	menu.AddSeparator()

	menu.Add("Open SIEM Search").OnClick(func(_ *application.Context) {
		tray.ShowWindow()
		emitMenuEvent("menu:goto", "/siem-search")
	})
	menu.Add("Open Alerts").OnClick(func(_ *application.Context) {
		tray.ShowWindow()
		emitMenuEvent("menu:goto", "/alerts")
	})
	menu.Add("Open Fleet").OnClick(func(_ *application.Context) {
		tray.ShowWindow()
		emitMenuEvent("menu:goto", "/fleet")
	})
	menu.Add("Open Terminal").OnClick(func(_ *application.Context) {
		tray.ShowWindow()
		emitMenuEvent("menu:goto", "/terminal")
	})

	menu.AddSeparator()

	menu.Add("New Pop-Out → SIEM").OnClick(func(_ *application.Context) {
		emitMenuEvent("tray:popout", "/siem-search")
	})
	menu.Add("New Pop-Out → Alerts").OnClick(func(_ *application.Context) {
		emitMenuEvent("tray:popout", "/alerts")
	})
	menu.Add("Close All Pop-Outs").OnClick(func(_ *application.Context) {
		emitMenuEvent("menu:close-popouts", nil)
	})

	menu.AddSeparator()
	menu.AddRole(application.Quit)

	tray.SetMenu(menu)
	tray.Run()
}
