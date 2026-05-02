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
	syslogAddr := os.Getenv("OBLIVRA_SYSLOG_ADDR")
	if syslogAddr == "" && os.Getenv("OBLIVRA_DISABLE_SYSLOG") == "" {
		syslogAddr = ":1514"
	}
	netflowAddr := os.Getenv("OBLIVRA_NETFLOW_ADDR")
	if netflowAddr == "" && os.Getenv("OBLIVRA_DISABLE_NETFLOW") == "" {
		netflowAddr = ":2055"
	}

	stack, err := platform.New(platform.Options{
		Logger:         logger,
		SyslogAddr:     syslogAddr,
		NetFlowAddr:    netflowAddr,
		StartListeners: true,
	})
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

	auth := httpserver.NewAuth(os.Getenv("OBLIVRA_API_KEYS"))
	if auth.Required() {
		logger.Info("API auth required")
	} else {
		logger.Warn("API auth disabled (set OBLIVRA_API_KEYS to enable)")
	}

	srv := httpserver.New(logger, httpserver.Deps{
		System: stack.System,
		Siem:   stack.Siem,
		Alerts: stack.Alerts,
		Intel:  stack.Intel,
		Rules:  stack.Rules,
		Audit:  stack.Audit,
		Fleet:  stack.Fleet,
		Ueba:   stack.Ueba,
		Ndr:    stack.Ndr,
		Foren:   stack.Foren,
		Tier:    stack.Tier,
		Lineage: stack.Lineage,
		Vault:          stack.Vault,
		Timeline:       stack.Timeline,
		Investigations: stack.Investigations,
		Reconstruction: stack.Reconstruction,
		TenantPolicy:   stack.TenantPolicy,
		Trust:          stack.Trust,
		Quality:        stack.Quality,
		Graph:          stack.Graph,
		Import:         stack.Import,
		Report:         stack.Report,
		Tamper:         stack.Tamper,
		Webhooks:       stack.Webhooks,
		Categories:     stack.Categories,
		Notifications:  stack.Notifications,
		Bus:            stack.Bus,
		Auth:   auth,
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
		logger.Info("oblivra-server listening",
			"addr", addr, "syslog", syslogAddr, "netflow", netflowAddr)
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
