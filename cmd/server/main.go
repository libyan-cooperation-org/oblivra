package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/kingknull/oblivrashell/internal/app"
)

func main() {
	fmt.Println("Starting OBLIVRA Headless Hardening Server...")

	// 1. Initialize App & Container
	application := app.New()
	baseCtx := context.Background()
	
	// Mark as test context to skip Wails GUI event emission 
	ctx := context.WithValue(baseCtx, "test", "true")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 2. Manual startup (bypassing Wails)
	application.Startup(ctx)

	// 3. Explicitly start Ingestion Server for Headless Mode
	fmt.Println("=> Starting Ingestion Listeners (Syslog/UDP)...")
	if application.IngestService != nil {
		if err := application.IngestService.StartSyslogServer(); err != nil {
			fmt.Printf("FAILED to start syslog server: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("=> Services started. Ingestion listening on :1514 (Syslog) and :8443 (Agent).")
	fmt.Println("=> Hardening telemetry active. Waiting for indexing warmup...")
	
	// Wait for auto-unlock to finish
	time.Sleep(5 * time.Second)

	fmt.Println("=> READY FOR SCALE AUDIT.")
	
	// 4. Hardening Telemetry Loop
	fmt.Println("=> Initializing Hardening Telemetry Loop (1m interval)...")
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				
				eps := 0
				drops := int64(0)
				if application.IngestService != nil {
					metrics := application.IngestService.GetMetrics()
					if v, ok := metrics["events_per_second"].(int64); ok {
						eps = int(v)
					}
					if v, ok := metrics["dropped_events"].(int64); ok {
						drops = v
					}
				}

				fmt.Printf("[HARDENING-TELEMETRY] Time: %s | RSS: %d MB | Goroutines: %d | EPS: %d | Drops: %d\n",
					time.Now().Format("15:04:05"),
					m.Alloc/1024/1024,
					runtime.NumGoroutine(),
					eps,
					drops,
				)
			}
		}
	}()

	// 5. Wait for termination
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n=> Shutdown signal received. Draining services...")
	application.Shutdown(ctx)
	fmt.Println("=> Shutdown complete.")
}
