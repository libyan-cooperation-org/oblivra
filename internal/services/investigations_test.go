package services

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/storage/hot"
	"github.com/kingknull/oblivra/internal/storage/search"
	"github.com/kingknull/oblivra/internal/wal"
)

// memHarness builds an in-memory hot store + search index + pipeline so we
// can exercise the investigations service end-to-end without disk I/O for
// the data plane (the audit + cases journals still need a tempdir).
type memHarness struct {
	hot      *hot.Store
	search   *search.Index
	wal      *wal.WAL
	pipeline *ingest.Pipeline
}

func newHarness(t *testing.T) *memHarness {
	t.Helper()
	walDir := t.TempDir()
	w, err := wal.Open(wal.Options{Dir: walDir})
	if err != nil {
		t.Fatal(err)
	}
	store, err := hot.Open(hot.Options{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	idx, err := search.Open(search.Options{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	pipe := ingest.New(slog.New(slog.NewTextHandler(testWriter{}, nil)), w, store, idx, nil)
	t.Cleanup(func() {
		_ = idx.Close()
		_ = store.Close()
		_ = w.Close()
	})
	return &memHarness{hot: store, search: idx, wal: w, pipeline: pipe}
}

func ingestAt(t *testing.T, p *ingest.Pipeline, host, msg string, when time.Time) {
	t.Helper()
	ev := &events.Event{
		Source:    events.SourceREST,
		HostID:    host,
		Message:   msg,
		Timestamp: when,
		ReceivedAt: when,
	}
	if err := p.Submit(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
}

func TestCaseSnapshotFreezesScope(t *testing.T) {
	h := newHarness(t)
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	audit, err := NewDurable(logger, dir, []byte("k"))
	if err != nil {
		t.Fatal(err)
	}
	defer audit.Close()
	alerts := NewAlertService(logger)
	foren := NewForensicsService(h.hot, audit)

	now := time.Now().UTC().Truncate(time.Second)

	// Pre-case events.
	ingestAt(t, h.pipeline, "web-01", "old-event-1", now.Add(-2*time.Hour))
	ingestAt(t, h.pipeline, "web-01", "old-event-2", now.Add(-1*time.Hour))
	ingestAt(t, h.pipeline, "web-02", "different-host", now.Add(-30*time.Minute))

	inv, err := NewInvestigationsService(logger, dir, h.hot, alerts, foren, audit)
	if err != nil {
		t.Fatal(err)
	}
	defer inv.Close()

	c, err := inv.Open(context.Background(), OpenCaseRequest{
		Title:    "test",
		HostID:   "web-01",
		OpenedBy: "alice",
		From:     now.Add(-3 * time.Hour),
		To:       now.Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Sleep a tick so future events land strictly after the cutoff.
	time.Sleep(20 * time.Millisecond)

	// Inject a NEW event after the case was opened.
	ingestAt(t, h.pipeline, "web-01", "post-case-event", time.Now().UTC())

	// The case timeline should NOT see the post-case event.
	tl, err := inv.Timeline(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range tl {
		if e.Detail == "post-case-event" {
			t.Fatalf("snapshot leaked post-case event: %+v", e)
		}
	}
	// And it should see the two pre-case events but not the wrong-host one.
	seen := map[string]bool{}
	for _, e := range tl {
		seen[e.Detail] = true
	}
	if !seen["old-event-1"] || !seen["old-event-2"] {
		t.Errorf("missing pre-case events: %v", seen)
	}
	if seen["different-host"] {
		t.Errorf("scope leak: web-02 event in web-01 case")
	}
}

func TestCasePersistAcrossRestart(t *testing.T) {
	h := newHarness(t)
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	audit, err := NewDurable(logger, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	alerts := NewAlertService(logger)
	foren := NewForensicsService(h.hot, audit)

	inv, err := NewInvestigationsService(logger, dir, h.hot, alerts, foren, audit)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := inv.Open(context.Background(), OpenCaseRequest{Title: "before-restart", OpenedBy: "alice"})
	_, _ = inv.AddNote(context.Background(), c.ID, "alice", "first note")
	_, _ = inv.Seal(context.Background(), c.ID, "alice")
	_ = inv.Close()
	_ = audit.Close()

	audit2, err := NewDurable(logger, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer audit2.Close()
	inv2, err := NewInvestigationsService(logger, dir, h.hot, alerts, foren, audit2)
	if err != nil {
		t.Fatal(err)
	}
	defer inv2.Close()

	got, ok := inv2.Get(c.ID)
	if !ok {
		t.Fatal("case lost after restart")
	}
	if got.State != CaseStateSealed {
		t.Errorf("state = %s after restart", got.State)
	}
	if len(got.Notes) != 1 || got.Notes[0].Body != "first note" {
		t.Errorf("notes lost: %+v", got.Notes)
	}
}

func TestSealedCaseRejectsNotes(t *testing.T) {
	h := newHarness(t)
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	audit, _ := NewDurable(logger, dir, nil)
	defer audit.Close()
	alerts := NewAlertService(logger)
	foren := NewForensicsService(h.hot, audit)
	inv, _ := NewInvestigationsService(logger, dir, h.hot, alerts, foren, audit)
	defer inv.Close()

	c, _ := inv.Open(context.Background(), OpenCaseRequest{Title: "x", OpenedBy: "alice"})
	if _, err := inv.Seal(context.Background(), c.ID, "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := inv.AddNote(context.Background(), c.ID, "alice", "after seal"); err == nil {
		t.Fatal("AddNote should fail on sealed case")
	}
}
