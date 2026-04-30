package services

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// TestAuditConcurrentAppend hammers the durable audit chain from many
// goroutines and confirms the resulting chain still verifies. The append path
// holds a mutex; this test is what catches the race a future refactor might
// introduce.
func TestAuditConcurrentAppend(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	a, err := NewDurable(logger, dir, []byte("k"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	const workers = 12
	const perW = 25
	var wg sync.WaitGroup
	wg.Add(workers)
	for g := 0; g < workers; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perW; i++ {
				a.Append(context.Background(), "alice", "siem.search",
					"default", map[string]string{"q": "x"})
			}
		}(g)
	}
	wg.Wait()

	r := a.Verify()
	if !r.OK {
		t.Fatalf("audit chain broken under concurrent append: brokenAt=%d", r.BrokenAt)
	}
	if r.Entries != workers*perW {
		t.Errorf("entries = %d, want %d", r.Entries, workers*perW)
	}
}

// TestCaseSnapshotLeakUnderConcurrentIngest exercises the §3 promise:
// opening a case freezes a snapshot, and events ingested *after* the
// snapshot must not appear through it — even if concurrent ingest is racing
// with the case open.
func TestCaseSnapshotLeakUnderConcurrentIngest(t *testing.T) {
	h := newHarness(t)
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	audit, err := NewDurable(logger, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer audit.Close()
	alerts := NewAlertService(logger)
	foren := NewForensicsService(h.hot, audit)

	// Pre-populate.
	for i := 0; i < 50; i++ {
		ingestAt(t, h.pipeline, "web-01", "pre-event", time.Now().Add(-time.Hour))
	}

	inv, err := NewInvestigationsService(logger, dir, h.hot, alerts, foren, audit)
	if err != nil {
		t.Fatal(err)
	}
	defer inv.Close()

	// Race: open a case while another goroutine is hammering ingest.
	var wg sync.WaitGroup
	wg.Add(1)
	stop := make(chan struct{})
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				ingestAt(t, h.pipeline, "web-01", "during-or-after",
					time.Now())
				i++
			}
		}
	}()

	// Brief delay so ingest is actively running when we open the case.
	time.Sleep(20 * time.Millisecond)

	c, err := inv.Open(context.Background(), OpenCaseRequest{
		Title: "leak-test", HostID: "web-01", OpenedBy: "alice",
		From: time.Now().Add(-2 * time.Hour),
		To:   time.Now().Add(2 * time.Hour),
	})
	if err != nil {
		close(stop)
		t.Fatal(err)
	}
	cutoff := c.Scope.ReceivedAtCutoff

	// Let post-cutoff ingest accumulate.
	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()

	tl, err := inv.Timeline(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range tl {
		if e.Timestamp.After(cutoff) {
			t.Fatalf("snapshot leak: event at %v is after cutoff %v",
				e.Timestamp, cutoff)
		}
	}
}

// TestConcurrentCaseLifecycle confirms that open / annotate / hypothesis /
// seal can all run from many goroutines without losing writes. We only
// assert "the case file deserialises and Verify is happy" — the audit chain
// is the source of truth for ordering.
func TestConcurrentCaseLifecycle(t *testing.T) {
	h := newHarness(t)
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	audit, _ := NewDurable(logger, dir, nil)
	defer audit.Close()
	alerts := NewAlertService(logger)
	foren := NewForensicsService(h.hot, audit)
	inv, err := NewInvestigationsService(logger, dir, h.hot, alerts, foren, audit)
	if err != nil {
		t.Fatal(err)
	}
	defer inv.Close()

	// Open a few cases concurrently.
	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	ids := make(chan string, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			c, err := inv.Open(context.Background(), OpenCaseRequest{Title: "race", OpenedBy: "alice"})
			if err == nil {
				ids <- c.ID
			}
		}()
	}
	wg.Wait()
	close(ids)

	// For every successfully opened case, do annotate + hypothesis + seal
	// concurrently.
	var wg2 sync.WaitGroup
	for id := range ids {
		wg2.Add(1)
		go func(id string) {
			defer wg2.Done()
			ctx := context.Background()
			_, _ = inv.AddNote(ctx, id, "alice", "concurrent note")
			_, _ = inv.AddHypothesis(ctx, id, "alice", "h1")
			_, _ = inv.Seal(ctx, id, "alice")
		}(id)
	}
	wg2.Wait()

	// Audit chain must still verify.
	if r := audit.Verify(); !r.OK {
		t.Fatalf("audit chain broken: brokenAt=%d entries=%d", r.BrokenAt, r.Entries)
	}

	// All cases must be retrievable and sealed.
	for _, c := range inv.List() {
		if c.State != CaseStateSealed {
			t.Errorf("case %s not sealed: state=%s", c.ID, c.State)
		}
	}
}

