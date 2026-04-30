package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kingknull/oblivra/internal/httpserver"
	"github.com/kingknull/oblivra/internal/platform"
	"github.com/kingknull/oblivra/webassets"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	addr := os.Getenv("OBLIVRA_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	stack, err := platform.New(platform.Options{Logger: logger})
	if err != nil {
		logger.Error("bootstrap failed", "err", err)
		os.Exit(1)
	}
	defer stack.Close()

	sub, err := webassets.FS()
	if err != nil {
		logger.Warn("static assets unavailable; serving API only", "err", err)
		sub = nil
	}

	srv := httpserver.New(logger, httpserver.Deps{
		System: stack.System,
		Siem:   stack.Siem,
		Assets: sub,
	})

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("oblivra-server listening", "addr", addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}
