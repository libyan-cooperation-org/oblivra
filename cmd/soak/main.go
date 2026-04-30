// oblivra-soak — sustained-load ingest tester.
//
// Drives a configurable EPS at a target server for a configurable duration,
// reporting throughput, p50/p95/p99 ingest latency, and HTTP error rates.
// Used to validate that the platform sustains its claimed ingest rate
// without dropping events.
//
// Usage:
//
//	oblivra-soak --server http://localhost:8080 --eps 5000 --duration 5m
//	oblivra-soak --server http://localhost:8080 --eps 1000 --batch 100
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	var (
		server   = flag.String("server", "http://localhost:8080", "server URL")
		token    = flag.String("token", os.Getenv("OBLIVRA_TOKEN"), "API key")
		eps      = flag.Int("eps", 1000, "target events per second")
		duration = flag.Duration("duration", 30*time.Second, "soak duration")
		batch    = flag.Int("batch", 50, "events per HTTP request")
		workers  = flag.Int("workers", 8, "concurrent worker goroutines")
		hosts    = flag.Int("hosts", 50, "synthetic host pool size")
		warmup   = flag.Duration("warmup", 2*time.Second, "warmup before measuring")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	r := &runner{
		server:  *server,
		token:   *token,
		batch:   *batch,
		hostPool: *hosts,
	}

	log.Printf("oblivra-soak: %d EPS for %s via %d workers (batch=%d) → %s",
		*eps, *duration, *workers, *batch, *server)
	log.Printf("warmup %s …", *warmup)

	// Compute per-worker tick interval to hit the target EPS.
	per := time.Duration(float64(time.Second) / float64(*eps) * float64(*workers))
	if per < time.Microsecond {
		per = time.Microsecond
	}

	wg := sync.WaitGroup{}
	deadline := time.Now().Add(*warmup + *duration)
	measureFrom := time.Now().Add(*warmup)

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r.worker(ctx, id, per, deadline, measureFrom)
		}(i)
	}
	wg.Wait()
	r.report(*duration)
}

type runner struct {
	server   string
	token    string
	batch    int
	hostPool int

	sent     atomic.Int64
	ok       atomic.Int64
	failed   atomic.Int64

	muLatency sync.Mutex
	latencies []time.Duration
}

func (r *runner) worker(ctx context.Context, id int, per time.Duration, deadline, measureFrom time.Time) {
	rng := rand.New(rand.NewSource(int64(id)*1000 + time.Now().UnixNano()))
	t := time.NewTicker(per)
	defer t.Stop()
	client := &http.Client{Timeout: 10 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if time.Now().After(deadline) {
				return
			}
			r.flush(ctx, client, rng, id, time.Now().After(measureFrom))
		}
	}
}

func (r *runner) flush(ctx context.Context, client *http.Client, rng *rand.Rand, workerID int, measure bool) {
	batch := make([]map[string]any, r.batch)
	for i := 0; i < r.batch; i++ {
		host := fmt.Sprintf("host-%03d", rng.Intn(r.hostPool))
		batch[i] = map[string]any{
			"source":    "rest",
			"hostId":    host,
			"severity":  pickSeverity(rng),
			"eventType": "soak",
			"message":   fmt.Sprintf("soak worker=%d seq=%d host=%s rand=%d", workerID, r.sent.Load(), host, rng.Int63()),
		}
	}
	body, _ := json.Marshal(batch)
	req, err := http.NewRequestWithContext(ctx, "POST", r.server+"/api/v1/siem/ingest/batch", bytes.NewReader(body))
	if err != nil {
		r.failed.Add(int64(r.batch))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	start := time.Now()
	resp, err := client.Do(req)
	dur := time.Since(start)

	r.sent.Add(int64(r.batch))
	if err != nil {
		r.failed.Add(int64(r.batch))
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		r.failed.Add(int64(r.batch))
		return
	}
	r.ok.Add(int64(r.batch))
	if measure {
		r.muLatency.Lock()
		r.latencies = append(r.latencies, dur)
		r.muLatency.Unlock()
	}
}

func (r *runner) report(measured time.Duration) {
	sent := r.sent.Load()
	ok := r.ok.Load()
	failed := r.failed.Load()

	r.muLatency.Lock()
	lats := append([]time.Duration(nil), r.latencies...)
	r.muLatency.Unlock()
	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })

	p50 := percentile(lats, 50)
	p95 := percentile(lats, 95)
	p99 := percentile(lats, 99)

	rate := float64(ok) / measured.Seconds()
	fmt.Println("============================================================")
	fmt.Printf("  sent:        %d events\n", sent)
	fmt.Printf("  ok:          %d events (%.1f%%)\n", ok, pct(ok, sent))
	fmt.Printf("  failed:      %d events (%.1f%%)\n", failed, pct(failed, sent))
	fmt.Printf("  rate:        %.0f events/sec sustained\n", rate)
	fmt.Printf("  latency p50: %s\n", p50)
	fmt.Printf("  latency p95: %s\n", p95)
	fmt.Printf("  latency p99: %s\n", p99)
	fmt.Println("============================================================")
}

func pickSeverity(r *rand.Rand) string {
	switch r.Intn(20) {
	case 0:
		return "error"
	case 1, 2:
		return "warning"
	default:
		return "info"
	}
}

func percentile(s []time.Duration, p int) time.Duration {
	if len(s) == 0 {
		return 0
	}
	idx := (len(s) * p) / 100
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return s[idx]
}

func pct(n, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}
