package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/kingknull/oblivra/internal/platform"
	"github.com/kingknull/oblivra/webassets"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	stack, err := platform.New(platform.Options{
		Logger:         logger,
		SyslogAddr:     "", // desktop doesn't bind to network listeners by default
		StartListeners: false,
	})
	if err != nil {
		log.Fatalf("oblivra: bootstrap: %v", err)
	}
	defer stack.Close()

	app := application.New(application.Options{
		Name:        "OBLIVRA",
		Description: "Sovereign Log-Driven Security Platform",
		LogLevel:    slog.LevelDebug,
		Services: []application.Service{
			application.NewService(stack.System),
			application.NewService(stack.Siem),
			application.NewService(stack.Alerts),
			application.NewService(stack.Intel),
			application.NewService(stack.Rules),
			application.NewService(stack.Audit),
			application.NewService(stack.Fleet),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(webassets.Raw()),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "OBLIVRA",
		Width:            1440,
		Height:           900,
		MinWidth:         1100,
		MinHeight:        700,
		BackgroundColour: application.NewRGB(11, 13, 18),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatalf("oblivra: %v", err)
	}
}
