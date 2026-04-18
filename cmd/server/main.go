package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/kingknull/oblivrashell/internal/app"
)

func main() {
	// ── CLI flags ────────────────────────────────────────────────────────────
	webPort  := flag.Int("web-port", 8090, "Port to serve the web UI (static files + API proxy)")
	noTLS   := flag.Bool("no-tls", false, "Disable TLS for the REST API (development only)")
	webDir  := flag.String("web-dir", "", "Path to frontend/dist directory (defaults to ./frontend/dist)")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║          OBLIVRA — Headless Server Mode              ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")

	// ── Resolve frontend dist dir ────────────────────────────────────────────
	distDir := *webDir
	if distDir == "" {
		// Try relative to executable first, then working directory
		exe, _ := os.Executable()
		exeDir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(exeDir, "frontend", "dist"),
			filepath.Join(exeDir, "..", "frontend", "dist"),
			"frontend/dist",
			"../frontend/dist",
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				distDir = c
				break
			}
		}
	}

	serveWeb := distDir != ""
	if serveWeb {
		fmt.Printf("=> Web UI:      http://localhost:%d  (from %s)\n", *webPort, distDir)
	} else {
		fmt.Println("=> Web UI:      NOT FOUND — run 'bun run build' in frontend/ first")
		fmt.Println("                Or use --web-dir=path/to/frontend/dist")
		fmt.Println("                For dev mode: cd frontend && bun run dev")
	}

	// ── Initialize App & Container ───────────────────────────────────────────
	application := app.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application.Startup(ctx)

	// ── Start syslog ingestion ───────────────────────────────────────────────
	if application.IngestService != nil {
		if err := application.IngestService.StartSyslogServer(); err != nil {
			fmt.Printf("=> Syslog server: FAILED (%v)\n", err)
		} else {
			fmt.Println("=> Syslog:      :1514 (UDP/TCP)")
		}
	}

	// ── Web server: serve frontend + proxy API ───────────────────────────────
	if serveWeb || *webPort > 0 {
		go serveWebUI(*webPort, distDir, *noTLS)
	}

	fmt.Printf("\n=> REST API:    http://localhost:8080/api/v1/\n")
	fmt.Printf("=> Metrics:     http://localhost:8080/metrics\n")
	fmt.Printf("=> Health:      http://localhost:8080/healthz\n")
	fmt.Println("\n=> Press Ctrl+C to stop")

	// ── Hardening telemetry ──────────────────────────────────────────────────
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
				eps, drops := int64(0), int64(0)
				if application.IngestService != nil {
					metrics := application.IngestService.GetMetrics()
					if v, ok := metrics["events_per_second"].(int64); ok { eps = v }
					if v, ok := metrics["dropped_events"].(int64); ok { drops = v }
				}
				fmt.Printf("[TELEMETRY] %s | RSS: %d MB | Goroutines: %d | EPS: %d | Drops: %d\n",
					time.Now().Format("15:04:05"),
					m.Alloc/1024/1024,
					runtime.NumGoroutine(),
					eps, drops,
				)
			}
		}
	}()

	// ── Wait for signal ──────────────────────────────────────────────────────
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n=> Shutdown signal received. Draining...")
	application.Shutdown(ctx)
	fmt.Println("=> Done.")
}

// serveWebUI serves the frontend dist directory on the given port.
// All paths that don't match a static file fall through to index.html
// so SolidJS's hash router works correctly.
func serveWebUI(port int, distDir string, noTLS bool) {
	mux := http.NewServeMux()

	if distDir != "" {
		fs := http.FileServer(http.Dir(distDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Set permissive CORS so the API on :8080 can be called from the browser
			w.Header().Set("Access-Control-Allow-Origin", "*")
			// Try to serve the static file; if it doesn't exist, serve index.html
			// so the hash router can handle it.
			path := filepath.Join(distDir, filepath.Clean("/"+r.URL.Path))
			_, err := os.Stat(path)
			if os.IsNotExist(err) || r.URL.Path == "/" {
				http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, r)
		})
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Frontend not built. Run: cd frontend && bun run build", http.StatusServiceUnavailable)
		})
	}

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	fmt.Printf("=> Web server:  http://localhost%s\n", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("=> Web server error: %v\n", err)
	}
}
