// oblivra-soak — sustained-load ingest tester with credibility-grade reports.
//
// Drives a configurable EPS at a target server for a configurable duration,
// reporting throughput, p50/p95/p99 ingest latency, and HTTP error rates.
// Used to validate that the platform sustains its claimed ingest rate
// without dropping events.
//
// Beyond the human-readable summary, this tool can write:
//
//   --report-json FILE   Full result record for archival / regression analysis
//   --report-md   FILE   Markdown report (drop-in for docs/operator/)
//
// Pass/fail gates make this usable as a CI / pre-release credibility check:
//
//   --require-eps 5000        Fail unless sustained rate ≥ N events/sec
//   --max-error-rate 0.01     Fail if error fraction > 1%
//   --max-p99 250ms           Fail if p99 latency exceeds this
//
// Usage:
//
//	oblivra-soak --server http://localhost:8080 --eps 5000 --duration 5m
//	oblivra-soak --server http://localhost:8080 --eps 1000 \
//	             --report-md docs/operator/soak-results-2026-05-02.md
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
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	var (
		server         = flag.String("server", "http://localhost:8080", "server URL")
		token          = flag.String("token", os.Getenv("OBLIVRA_TOKEN"), "API key")
		eps            = flag.Int("eps", 1000, "target events per second")
		duration       = flag.Duration("duration", 30*time.Second, "soak duration")
		batch          = flag.Int("batch", 50, "events per HTTP request")
		workers        = flag.Int("workers", 8, "concurrent worker goroutines")
		hosts          = flag.Int("hosts", 50, "synthetic host pool size")
		warmup         = flag.Duration("warmup", 2*time.Second, "warmup before measuring")
		reportJSON     = flag.String("report-json", "", "write machine-readable JSON results to this file")
		reportMD       = flag.String("report-md", "", "write markdown report to this file")
		requireEPS     = flag.Float64("require-eps", 0, "fail if sustained EPS < N (0 disables gate)")
		maxErrorRate   = flag.Float64("max-error-rate", 0, "fail if error fraction > F (0 disables gate)")
		maxP99         = flag.Duration("max-p99", 0, "fail if p99 latency > D (0 disables gate)")
		labelHardware  = flag.String("label-hardware", os.Getenv("OBLIVRA_SOAK_HARDWARE"), "hardware label for the report header")
		labelComment   = flag.String("label-comment", "", "free-form comment to embed in the report")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	r := &runner{
		server:   *server,
		token:    *token,
		batch:    *batch,
		hostPool: *hosts,
	}

	log.Printf("oblivra-soak: %d EPS for %s via %d workers (batch=%d) → %s",
		*eps, *duration, *workers, *batch, *server)
	log.Printf("warmup %s …", *warmup)

	// Capture target system info BEFORE the run so the report has it
	// even if the server tips over partway through.
	sys := captureSystemInfo(ctx, *server, *token)

	// Compute per-worker tick interval to hit the target EPS.
	// Each tick sends `batch` events; with W workers each ticking at
	// the same rate, aggregate = (batch * W) / per events/sec. Solve
	// for per: per = time.Second * batch * W / target_eps.
	//
	// (Earlier versions of this file dropped the batch factor and
	// overshot by `batch×` — caught by the run-soak.sh smoke test.)
	per := time.Duration(float64(time.Second) * float64(*batch) * float64(*workers) / float64(*eps))
	if per < time.Microsecond {
		per = time.Microsecond
	}

	wg := sync.WaitGroup{}
	startTime := time.Now()
	deadline := startTime.Add(*warmup + *duration)
	measureFrom := startTime.Add(*warmup)

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r.worker(ctx, id, per, deadline, measureFrom)
		}(i)
	}
	wg.Wait()
	endTime := time.Now()

	report := r.buildReport(reportConfig{
		startTime: startTime,
		endTime:   endTime,
		duration:  *duration,
		warmup:    *warmup,
		targetEPS: *eps,
		batch:     *batch,
		workers:   *workers,
		server:    *server,
		hardware:  *labelHardware,
		comment:   *labelComment,
		sysInfo:   sys,
		gates: gates{
			requireEPS:   *requireEPS,
			maxErrorRate: *maxErrorRate,
			maxP99:       *maxP99,
		},
	})

	report.printHuman()

	if *reportJSON != "" {
		if err := report.writeJSON(*reportJSON); err != nil {
			log.Printf("write JSON report: %v", err)
		} else {
			log.Printf("wrote %s", *reportJSON)
		}
	}
	if *reportMD != "" {
		if err := report.writeMarkdown(*reportMD); err != nil {
			log.Printf("write markdown report: %v", err)
		} else {
			log.Printf("wrote %s", *reportMD)
		}
	}

	if !report.PassedGates {
		os.Exit(1)
	}
}

// ---- runner -----------------------------------------------------------

type runner struct {
	server   string
	token    string
	batch    int
	hostPool int

	sent   atomic.Int64
	ok     atomic.Int64
	failed atomic.Int64

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

// ---- system info ------------------------------------------------------

type systemInfo struct {
	ServerVersion   string `json:"serverVersion,omitempty"`
	ServerGoVersion string `json:"serverGoVersion,omitempty"`
	ServerOS        string `json:"serverOS,omitempty"`
	ServerArch      string `json:"serverArch,omitempty"`
	ServerCPUs      int    `json:"serverCPUs,omitempty"`
	ServerStartedAt string `json:"serverStartedAt,omitempty"`
	ClientGoVersion string `json:"clientGoVersion"`
	ClientOS        string `json:"clientOS"`
	ClientArch      string `json:"clientArch"`
	ClientCPUs      int    `json:"clientCPUs"`
}

// captureSystemInfo polls the server's /api/v1/system/info so the
// report has both ends of the wire on record. Best-effort: if the
// server is down or auth-gated, we still report client-side facts.
func captureSystemInfo(ctx context.Context, server, token string) systemInfo {
	out := systemInfo{
		ClientGoVersion: runtime.Version(),
		ClientOS:        runtime.GOOS,
		ClientArch:      runtime.GOARCH,
		ClientCPUs:      runtime.NumCPU(),
	}
	rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(rctx, "GET", server+"/api/v1/system/info", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return out
	}
	var srv struct {
		Version    string `json:"version"`
		GoVersion  string `json:"goVersion"`
		OS         string `json:"os"`
		Arch       string `json:"arch"`
		NumCPU     int    `json:"numCpu"`
		StartedAt  string `json:"startedAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&srv); err != nil {
		return out
	}
	out.ServerVersion = srv.Version
	out.ServerGoVersion = srv.GoVersion
	out.ServerOS = srv.OS
	out.ServerArch = srv.Arch
	out.ServerCPUs = srv.NumCPU
	out.ServerStartedAt = srv.StartedAt
	return out
}

// ---- report -----------------------------------------------------------

type reportConfig struct {
	startTime time.Time
	endTime   time.Time
	duration  time.Duration
	warmup    time.Duration
	targetEPS int
	batch     int
	workers   int
	server    string
	hardware  string
	comment   string
	sysInfo   systemInfo
	gates     gates
}

type gates struct {
	requireEPS   float64
	maxErrorRate float64
	maxP99       time.Duration
}

type Report struct {
	GeneratedAt    time.Time         `json:"generatedAt"`
	Server         string            `json:"server"`
	Hardware       string            `json:"hardware,omitempty"`
	Comment        string            `json:"comment,omitempty"`
	StartedAt      time.Time         `json:"startedAt"`
	EndedAt        time.Time         `json:"endedAt"`
	WarmupSeconds  float64           `json:"warmupSeconds"`
	MeasuredSeconds float64          `json:"measuredSeconds"`
	TargetEPS      int               `json:"targetEPS"`
	BatchSize      int               `json:"batchSize"`
	Workers        int               `json:"workers"`

	Sent   int64 `json:"sent"`
	OK     int64 `json:"ok"`
	Failed int64 `json:"failed"`

	OKPct       float64 `json:"okPct"`
	FailedPct   float64 `json:"failedPct"`
	SustainedEPS float64 `json:"sustainedEPS"`

	LatencyP50 time.Duration `json:"latencyP50"`
	LatencyP95 time.Duration `json:"latencyP95"`
	LatencyP99 time.Duration `json:"latencyP99"`

	System systemInfo `json:"system"`

	Gates       gateOutcome `json:"gates"`
	PassedGates bool        `json:"passedGates"`
}

type gateOutcome struct {
	RequireEPS    float64 `json:"requireEPS,omitempty"`
	MaxErrorRate  float64 `json:"maxErrorRate,omitempty"`
	MaxP99Ms      int64   `json:"maxP99Ms,omitempty"`
	EPSPass       bool    `json:"epsPass"`
	ErrorRatePass bool    `json:"errorRatePass"`
	P99Pass       bool    `json:"p99Pass"`
}

func (r *runner) buildReport(cfg reportConfig) Report {
	r.muLatency.Lock()
	lats := append([]time.Duration(nil), r.latencies...)
	r.muLatency.Unlock()
	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })

	measured := cfg.duration.Seconds()
	if measured <= 0 {
		measured = cfg.endTime.Sub(cfg.startTime).Seconds() - cfg.warmup.Seconds()
	}
	if measured <= 0 {
		measured = 1
	}
	rep := Report{
		GeneratedAt:     time.Now().UTC(),
		Server:          cfg.server,
		Hardware:        cfg.hardware,
		Comment:         cfg.comment,
		StartedAt:       cfg.startTime,
		EndedAt:         cfg.endTime,
		WarmupSeconds:   cfg.warmup.Seconds(),
		MeasuredSeconds: measured,
		TargetEPS:       cfg.targetEPS,
		BatchSize:       cfg.batch,
		Workers:         cfg.workers,

		Sent:   r.sent.Load(),
		OK:     r.ok.Load(),
		Failed: r.failed.Load(),

		LatencyP50: percentile(lats, 50),
		LatencyP95: percentile(lats, 95),
		LatencyP99: percentile(lats, 99),

		System: cfg.sysInfo,
	}
	if rep.Sent > 0 {
		rep.OKPct = float64(rep.OK) / float64(rep.Sent) * 100
		rep.FailedPct = float64(rep.Failed) / float64(rep.Sent) * 100
	}
	rep.SustainedEPS = float64(rep.OK) / measured

	// Pass/fail gates.
	rep.Gates.RequireEPS = cfg.gates.requireEPS
	rep.Gates.MaxErrorRate = cfg.gates.maxErrorRate
	rep.Gates.MaxP99Ms = cfg.gates.maxP99.Milliseconds()
	rep.Gates.EPSPass = cfg.gates.requireEPS == 0 || rep.SustainedEPS >= cfg.gates.requireEPS
	errorRate := 0.0
	if rep.Sent > 0 {
		errorRate = float64(rep.Failed) / float64(rep.Sent)
	}
	rep.Gates.ErrorRatePass = cfg.gates.maxErrorRate == 0 || errorRate <= cfg.gates.maxErrorRate
	rep.Gates.P99Pass = cfg.gates.maxP99 == 0 || rep.LatencyP99 <= cfg.gates.maxP99
	rep.PassedGates = rep.Gates.EPSPass && rep.Gates.ErrorRatePass && rep.Gates.P99Pass

	return rep
}

func (r Report) printHuman() {
	fmt.Println("============================================================")
	fmt.Printf("  target:       %d events/sec for %.0fs (warmup %.0fs)\n", r.TargetEPS, r.MeasuredSeconds, r.WarmupSeconds)
	fmt.Printf("  sent:         %d events\n", r.Sent)
	fmt.Printf("  ok:           %d events (%.2f%%)\n", r.OK, r.OKPct)
	fmt.Printf("  failed:       %d events (%.2f%%)\n", r.Failed, r.FailedPct)
	fmt.Printf("  sustained:    %.0f events/sec\n", r.SustainedEPS)
	fmt.Printf("  latency p50:  %s\n", r.LatencyP50)
	fmt.Printf("  latency p95:  %s\n", r.LatencyP95)
	fmt.Printf("  latency p99:  %s\n", r.LatencyP99)
	if r.Gates.RequireEPS > 0 || r.Gates.MaxErrorRate > 0 || r.Gates.MaxP99Ms > 0 {
		fmt.Println("  gates:")
		if r.Gates.RequireEPS > 0 {
			fmt.Printf("    require-eps:    %.0f → %s\n", r.Gates.RequireEPS, passFail(r.Gates.EPSPass))
		}
		if r.Gates.MaxErrorRate > 0 {
			fmt.Printf("    max-error-rate: %.4f → %s\n", r.Gates.MaxErrorRate, passFail(r.Gates.ErrorRatePass))
		}
		if r.Gates.MaxP99Ms > 0 {
			fmt.Printf("    max-p99:        %dms → %s\n", r.Gates.MaxP99Ms, passFail(r.Gates.P99Pass))
		}
		fmt.Printf("  overall:      %s\n", passFail(r.PassedGates))
	}
	fmt.Println("============================================================")
}

func passFail(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func (r Report) writeJSON(path string) error {
	body, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}

func (r Report) writeMarkdown(path string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# OBLIVRA soak test — %s\n\n", r.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC"))
	fmt.Fprintf(&b, "**Verdict:** %s\n\n", passFail(r.PassedGates))

	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "| Metric | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| Target EPS              | %d |\n", r.TargetEPS)
	fmt.Fprintf(&b, "| Sustained EPS           | **%.0f** |\n", r.SustainedEPS)
	fmt.Fprintf(&b, "| Duration                | %.0fs (%.0fs warmup) |\n", r.MeasuredSeconds, r.WarmupSeconds)
	fmt.Fprintf(&b, "| Total sent              | %s events |\n", commaInt(r.Sent))
	fmt.Fprintf(&b, "| OK                      | %s (%.2f%%) |\n", commaInt(r.OK), r.OKPct)
	fmt.Fprintf(&b, "| Failed                  | %s (%.2f%%) |\n", commaInt(r.Failed), r.FailedPct)
	fmt.Fprintf(&b, "| Latency p50             | %s |\n", r.LatencyP50)
	fmt.Fprintf(&b, "| Latency p95             | %s |\n", r.LatencyP95)
	fmt.Fprintf(&b, "| Latency p99             | %s |\n", r.LatencyP99)
	fmt.Fprintf(&b, "| Workers / batch size    | %d / %d |\n", r.Workers, r.BatchSize)

	if r.Gates.RequireEPS > 0 || r.Gates.MaxErrorRate > 0 || r.Gates.MaxP99Ms > 0 {
		fmt.Fprintf(&b, "\n## Gates\n\n")
		fmt.Fprintf(&b, "| Gate | Threshold | Result |\n|---|---|---|\n")
		if r.Gates.RequireEPS > 0 {
			fmt.Fprintf(&b, "| Sustained EPS    | ≥ %.0f       | %s |\n", r.Gates.RequireEPS, passFail(r.Gates.EPSPass))
		}
		if r.Gates.MaxErrorRate > 0 {
			fmt.Fprintf(&b, "| Error rate       | ≤ %.4f      | %s |\n", r.Gates.MaxErrorRate, passFail(r.Gates.ErrorRatePass))
		}
		if r.Gates.MaxP99Ms > 0 {
			fmt.Fprintf(&b, "| Latency p99      | ≤ %dms       | %s |\n", r.Gates.MaxP99Ms, passFail(r.Gates.P99Pass))
		}
	}

	fmt.Fprintf(&b, "\n## Topology\n\n")
	fmt.Fprintf(&b, "- **Server**: `%s`\n", r.Server)
	if r.Hardware != "" {
		fmt.Fprintf(&b, "- **Hardware label**: %s\n", r.Hardware)
	}
	fmt.Fprintf(&b, "- **Started**: %s\n", r.StartedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "- **Ended**:   %s\n", r.EndedAt.UTC().Format(time.RFC3339))

	fmt.Fprintf(&b, "\n## System info\n\n")
	if r.System.ServerVersion != "" {
		fmt.Fprintf(&b, "Server: OBLIVRA %s · %s · %s/%s · %d CPUs · started %s\n\n",
			r.System.ServerVersion, r.System.ServerGoVersion, r.System.ServerOS, r.System.ServerArch,
			r.System.ServerCPUs, r.System.ServerStartedAt)
	} else {
		fmt.Fprintf(&b, "Server: (system info unavailable — server unreachable or auth-gated)\n\n")
	}
	fmt.Fprintf(&b, "Client: %s · %s/%s · %d CPUs\n",
		r.System.ClientGoVersion, r.System.ClientOS, r.System.ClientArch, r.System.ClientCPUs)

	if r.Comment != "" {
		fmt.Fprintf(&b, "\n## Notes\n\n%s\n", r.Comment)
	}

	fmt.Fprintf(&b, "\n## Reproduce\n\n```bash\noblivra-soak \\\n")
	fmt.Fprintf(&b, "  --server %s \\\n", r.Server)
	fmt.Fprintf(&b, "  --eps %d \\\n", r.TargetEPS)
	fmt.Fprintf(&b, "  --duration %.0fs \\\n", r.MeasuredSeconds)
	fmt.Fprintf(&b, "  --workers %d \\\n", r.Workers)
	fmt.Fprintf(&b, "  --batch %d\n", r.BatchSize)
	fmt.Fprintf(&b, "```\n")

	return os.WriteFile(path, []byte(b.String()), 0o644)
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

// commaInt formats an int with thousands-separator commas. Inline so
// we don't pull in golang.org/x/text just for the report.
func commaInt(n int64) string {
	if n < 0 {
		return "-" + commaInt(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", commaInt(n/1000), n%1000)
}
