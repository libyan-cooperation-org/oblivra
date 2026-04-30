package reconstruction

import (
	"context"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/storage/hot"
)

func mkProc(t *testing.T, store *hot.Store, host, msg, etype string, ts time.Time) {
	t.Helper()
	ev := &events.Event{
		Source:    events.SourceAgent,
		HostID:    host,
		Message:   msg,
		EventType: etype,
		Timestamp: ts,
	}
	if err := ev.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ev); err != nil {
		t.Fatal(err)
	}
}

func TestStateAtTimeT(t *testing.T) {
	store, err := hot.Open(hot.Options{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	t0 := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	// process_creation events
	mkProc(t, store, "win-01", "NewProcessId: 0x4d2 ParentProcessId: 0x32c Image: cmd.exe", "process_creation", t0)
	mkProc(t, store, "win-01", "NewProcessId: 0x999 ParentProcessId: 0x4d2 Image: powershell.exe", "process_creation", t0.Add(time.Second))
	// cmd.exe exits later
	mkProc(t, store, "win-01", "pid=1234 exit", "process_exit", t0.Add(2*time.Second))

	svc := NewStateService(store)

	// At t0+0.5s — both not yet exited; only cmd.exe is created.
	snap, err := svc.Reconstruct(context.Background(), "default", "win-01", t0.Add(500*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Running) != 1 || snap.Running[0].PID != 0x4d2 {
		t.Errorf("at t+0.5s expected only cmd.exe running, got %+v", snap.Running)
	}

	// At t0+1.5s — both running.
	snap, _ = svc.Reconstruct(context.Background(), "default", "win-01", t0.Add(1500*time.Millisecond))
	if len(snap.Running) != 2 {
		t.Errorf("at t+1.5s expected 2 running, got %+v", snap.Running)
	}

	// At t0+3s — cmd.exe (pid 1234) has exited; only powershell remains.
	snap, _ = svc.Reconstruct(context.Background(), "default", "win-01", t0.Add(3*time.Second))
	if len(snap.Running) != 1 || snap.Running[0].PID != 0x999 {
		t.Errorf("at t+3s expected only powershell, got %+v", snap.Running)
	}
	if len(snap.Exited) != 1 {
		t.Errorf("expected 1 exited entry, got %+v", snap.Exited)
	}
}
