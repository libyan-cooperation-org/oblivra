package architecture_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/app"
)

func TestHighThroughputIngestion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// 1. Setup Temp Environment
	tempDir := t.TempDir()
	os.Setenv("APPDATA", tempDir)
	os.Setenv("LOCALAPPDATA", tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ctx = context.WithValue(ctx, "test", "true")

	// 2. Initialize App
	application := app.New()
	application.Startup(ctx)
	defer application.Shutdown(ctx)

	// Unlock vault for DB access
	password := "stress-test-pass"
	application.VaultService.Setup(password, "")
	application.VaultService.Unlock(password, nil, false)
	defer application.VaultService.Lock()

	// Start Syslog Server
	err := application.IngestService.StartSyslogServer()
	if err != nil {
		t.Fatalf("Failed to start syslog server: %v", err)
	}

	// 3. Generate high-throughput burst (5,000 EPS for 10 seconds = 50,000 events)
	targetEPS := 5000
	duration := 10 * time.Second
	totalExpected := targetEPS * 10

	t.Logf("Firing burst: %d EPS for %s (%d events total)...", targetEPS, duration, totalExpected)

	conn, err := net.Dial("udp", "127.0.0.1:1514")
	if err != nil {
		t.Fatalf("Failed to dial syslog: %v", err)
	}
	defer conn.Close()

	ticker := time.NewTicker(time.Second / time.Duration(targetEPS))
	defer ticker.Stop()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	sent := 0
	burstStart := time.Now()

	for {
		select {
		case <-timer.C:
			goto BurstDone
		case ts := <-ticker.C:
			msg := fmt.Sprintf("<34>1 %s stress-generator su: 'su root' failed for nobody on /dev/pts/8\n", ts.Format(time.RFC3339))
			conn.Write([]byte(msg))
			sent++
		}
	}

BurstDone:
	elapsed := time.Since(burstStart)
	t.Logf("Burst complete. Sent %d events in %.2fs (%.1f EPS). Waiting for pipeline to drain...", sent, elapsed.Seconds(), float64(sent)/elapsed.Seconds())

	// 4. Wait for pipeline to drain (using metrics)
	maxWait := 30 * time.Second
	waitStart := time.Now()
	for {
		metrics := application.IngestService.GetMetrics()
		processed := metrics["total_processed"].(int64)

		if processed >= int64(sent) {
			t.Logf("Pipeline drained. Total processed: %d", processed)
			break
		}

		if time.Since(waitStart) > maxWait {
			t.Fatalf("Timeout waiting for pipeline to drain. Processed: %d/%d", processed, sent)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 5. Integrity Check: Query SIEM store for count
	// Note: We need to use a query that matches our generated logs.
	// "nobody" is a good keyword from the generation loop.
	events, err := application.SIEMService.SearchHostEvents("nobody", 10)
	if err != nil {
		t.Fatalf("Failed to query SIEM store: %v", err)
	}

	// Since SearchHostEvents has a limit, we should ideally have a "Count" method.
	// But we can check the total_processed metric which is tied to the successful DB insert in the pipeline.

	finalMetrics := application.IngestService.GetMetrics()
	dropped := finalMetrics["dropped_events"].(int64)

	if dropped > 0 {
		t.Errorf("Data Loss Detected! Dropped events: %d", dropped)
	}

	if len(events) == 0 {
		t.Error("Zero events found in SIEM search after high-throughput burst")
	}

	t.Logf("Stress test passed. Dropped: %d, Samples Found: %d", dropped, len(events))
}
