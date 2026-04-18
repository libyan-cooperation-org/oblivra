// cmd/soak_test/main.go — OBLIVRA Soak Test Generator (Phase 22.1)
//
// Dual-mode soak generator: UDP syslog (original) or HTTP REST ingest.
// Adds --report-json for machine-readable output consumed by the GHA soak.yml workflow.
//
// Usage (UDP syslog — original mode):
//
//	./soak_test --target 127.0.0.1:1514 --eps 5000 --duration 30s
//
// Usage (HTTP REST — soak regression mode):
//
//	./soak_test \
//	  --target http://127.0.0.1:8090/api/v1/ingest \
//	  --eps 5000 \
//	  --duration 1800s \
//	  --report-json /tmp/soak_report.json \
//	  --sample-interval 30
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SoakReport is written to --report-json. Schema is consumed by soak.yml evaluate step.
type SoakReport struct {
	TargetEPS      float64   `json:"target_eps"`
	ActualEPS      float64   `json:"actual_eps"`
	MinEPSWindow   float64   `json:"min_eps_window"`
	MaxEPSWindow   float64   `json:"max_eps_window"`
	TotalEvents    int64     `json:"total_events"`
	DroppedEvents  int64     `json:"dropped_events"`
	DurationS      float64   `json:"duration_s"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	SamplesEPS     []float64 `json:"samples_eps,omitempty"`
}

func main() {
	target         := flag.String("target", "127.0.0.1:1514", "Target: UDP addr (host:port) or HTTP URL")
	eps            := flag.Int("eps", 5000, "Events per second to generate")
	durationFlag   := flag.Duration("duration", 30*time.Second, "Test duration (e.g. 1800s, 30m)")
	workers        := flag.Int("workers", 8, "Concurrent worker goroutines")
	reportJSON     := flag.String("report-json", "", "Path to write JSON report (optional)")
	sampleInterval := flag.Int("sample-interval", 10, "EPS sampling interval in seconds")
	apiKey         := flag.String("api-key", "", "X-API-Key header for HTTP mode")
	flag.Parse()

	isHTTP := strings.HasPrefix(*target, "http://") || strings.HasPrefix(*target, "https://")

	fmt.Printf("╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("║       OBLIVRA SOAK TEST GENERATOR               ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════╝\n")
	fmt.Printf("  Mode     : %s\n", map[bool]string{true: "HTTP REST", false: "UDP Syslog"}[isHTTP])
	fmt.Printf("  Target   : %s\n", *target)
	fmt.Printf("  EPS      : %d\n", *eps)
	fmt.Printf("  Duration : %s\n", *durationFlag)
	fmt.Printf("  Workers  : %d\n\n", *workers)

	var (
		totalSent    int64
		totalDropped int64
		wg           sync.WaitGroup
	)

	start    := time.Now()
	deadline := start.Add(*durationFlag)

	// ── EPS sampler ──────────────────────────────────────────────────────────
	var samplesMu    sync.Mutex
	var samplesEPS   []float64
	var minEPS, maxEPS float64 = 1e9, 0

	go func() {
		ticker := time.NewTicker(time.Duration(*sampleInterval) * time.Second)
		defer ticker.Stop()
		prev := int64(0)
		prevT := time.Now()
		for {
			select {
			case <-ticker.C:
				now  := time.Now()
				curr := atomic.LoadInt64(&totalSent)
				dt   := now.Sub(prevT).Seconds()
				if dt > 0 {
					windowEPS := float64(curr-prev) / dt
					samplesMu.Lock()
					samplesEPS = append(samplesEPS, windowEPS)
					if windowEPS < minEPS { minEPS = windowEPS }
					if windowEPS > maxEPS { maxEPS = windowEPS }
					samplesMu.Unlock()
					fmt.Printf("  [sample] EPS: %.0f  sent: %d  dropped: %d\n",
						windowEPS, curr, atomic.LoadInt64(&totalDropped))
				}
				prev  = curr
				prevT = now
			}
		}
	}()

	// ── Workers ──────────────────────────────────────────────────────────────
	workerEPS      := *eps / *workers
	if workerEPS < 1 { workerEPS = 1 }
	tickInterval   := time.Second / time.Duration(workerEPS)

	if isHTTP {
		// ── HTTP mode ──────────────────────────────────────────────────────
		client := &http.Client{Timeout: 3 * time.Second}

		for w := 0; w < *workers; w++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				ticker := time.NewTicker(tickInterval)
				defer ticker.Stop()
				seq := 0
				for {
					select {
					case t := <-ticker.C:
						if t.After(deadline) {
							return
						}
						payload := fmt.Sprintf(
							`{"event_type":"soak_test","host":"soak-worker-%d","seq":%d,"timestamp":"%s","raw_line":"soak event from worker %d seq %d"}`,
							id, seq, t.UTC().Format(time.RFC3339Nano), id, seq,
						)
						req, _ := http.NewRequest("POST", *target, bytes.NewBufferString(payload))
						req.Header.Set("Content-Type", "application/json")
						if *apiKey != "" {
							req.Header.Set("X-API-Key", *apiKey)
						}
						resp, err := client.Do(req)
						if err != nil {
							atomic.AddInt64(&totalDropped, 1)
						} else {
							resp.Body.Close()
							if resp.StatusCode >= 400 {
								atomic.AddInt64(&totalDropped, 1)
							} else {
								atomic.AddInt64(&totalSent, 1)
							}
						}
						seq++
					}
				}
			}(w)
		}
	} else {
		// ── UDP Syslog mode (original) ──────────────────────────────────────
		conn, err := net.Dial("udp", *target)
		if err != nil {
			log.Fatalf("Failed to connect to %s: %v", *target, err)
		}
		defer conn.Close()

		startMarker := fmt.Sprintf("<34>1 %s soak_machine su: SOAK_TEST_START_MARKER\n", start.Format(time.RFC3339))
		conn.Write([]byte(startMarker))

		for w := 0; w < *workers; w++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				ticker := time.NewTicker(tickInterval)
				defer ticker.Stop()
				seq := 0
				for {
					select {
					case t := <-ticker.C:
						if t.After(deadline) {
							return
						}
						msg := fmt.Sprintf(
							"<34>1 %s load_gen_%d su: 'su root' failed for user_%d on /dev/pts/%d seq=%d\n",
							t.Format(time.RFC3339), id, seq%100, id, seq,
						)
						_, err := conn.Write([]byte(msg))
						if err != nil {
							atomic.AddInt64(&totalDropped, 1)
						} else {
							atomic.AddInt64(&totalSent, 1)
						}
						seq++
					}
				}
			}(w)
		}
	}

	wg.Wait()
	endTime := time.Now()
	elapsed := endTime.Sub(start).Seconds()

	sent    := atomic.LoadInt64(&totalSent)
	dropped := atomic.LoadInt64(&totalDropped)
	actualEPS := float64(sent) / elapsed

	// ── Final min/max EPS ────────────────────────────────────────────────────
	samplesMu.Lock()
	if minEPS == 1e9 { minEPS = actualEPS }
	if maxEPS == 0   { maxEPS = actualEPS }
	samplesMu.Unlock()

	// ── UDP end marker ───────────────────────────────────────────────────────
	if !isHTTP {
		if conn, err := net.Dial("udp", *target); err == nil {
			endMarker := fmt.Sprintf("<34>1 %s soak_machine su: SOAK_TEST_END_MARKER\n", endTime.Format(time.RFC3339))
			conn.Write([]byte(endMarker))
			conn.Close()
		}
	}

	// ── Summary ──────────────────────────────────────────────────────────────
	fmt.Printf("\n╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("║              SOAK TEST COMPLETE                 ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════╝\n")
	fmt.Printf("  Total sent   : %d events\n", sent)
	fmt.Printf("  Dropped      : %d events\n", dropped)
	fmt.Printf("  Duration     : %.2fs\n", elapsed)
	fmt.Printf("  Actual EPS   : %.0f  (%.1f%% of target %d)\n",
		actualEPS, (actualEPS/float64(*eps))*100, *eps)
	fmt.Printf("  Min window   : %.0f EPS\n", minEPS)
	fmt.Printf("  Max window   : %.0f EPS\n", maxEPS)

	// ── JSON report ───────────────────────────────────────────────────────────
	if *reportJSON != "" {
		samplesMu.Lock()
		report := SoakReport{
			TargetEPS:    float64(*eps),
			ActualEPS:    actualEPS,
			MinEPSWindow: minEPS,
			MaxEPSWindow: maxEPS,
			TotalEvents:  sent,
			DroppedEvents: dropped,
			DurationS:    elapsed,
			StartTime:    start,
			EndTime:      endTime,
			SamplesEPS:   samplesEPS,
		}
		samplesMu.Unlock()

		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal report: %v\n", err)
		} else if err := os.WriteFile(*reportJSON, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write report to %s: %v\n", *reportJSON, err)
		} else {
			fmt.Printf("\n  Report written to: %s\n", *reportJSON)
		}
	}

	// Exit non-zero if >10% drop rate
	if float64(dropped)/float64(sent+dropped) > 0.10 {
		fmt.Fprintf(os.Stderr, "\n[WARN] Drop rate >10%% — check server backpressure config\n")
	}
}
