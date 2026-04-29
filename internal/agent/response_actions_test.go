package agent

import (
	"os"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
)

func newTestExecutor(t *testing.T) *ResponseActionExecutor {
	t.Helper()
	log, err := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	return NewResponseActionExecutor(log)
}

// Phase 36.7: TestKillProcess* tests removed (KillProcess primitive deleted
// with response-action chain). Snapshot tests retained.

func TestCollectProcessSnapshotSelf(t *testing.T) {
	ex := newTestExecutor(t)
	snap, err := ex.CollectProcessSnapshot(os.Getpid())
	if err != nil {
		t.Fatalf("CollectProcessSnapshot(self) failed: %v", err)
	}
	if snap.PID != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), snap.PID)
	}
	if snap.CapturedAt == "" {
		t.Error("expected non-empty CapturedAt")
	}
	t.Logf("self snapshot: PID=%d name=%q open_files=%d mem_bytes=%d",
		snap.PID, snap.Name, snap.OpenFiles, snap.Memory)
}

func TestCollectProcessSnapshotRejectsInvalid(t *testing.T) {
	ex := newTestExecutor(t)
	for _, pid := range []int{0, -1} {
		if _, err := ex.CollectProcessSnapshot(pid); err == nil {
			t.Errorf("expected error for PID %d, got nil", pid)
		}
	}
}

func TestPIIRedactorDoesNotRedactIPs(t *testing.T) {
	r := NewPIIRedactor()
	// IPs must pass through — they are security signals
	input := "Connection from 192.168.1.42 port 22"
	got := r.RedactString(input)
	if got != input {
		t.Errorf("IP address was unexpectedly redacted: %q → %q", input, got)
	}
}

func TestPIIRedactorRedactsSecrets(t *testing.T) {
	r := NewPIIRedactor()
	cases := []struct {
		input    string
		mustDrop string
	}{
		{"my email is user@example.com in logs", "user@example.com"},
		{"token=sk-abcdef1234567890secret", "sk-abcdef1234567890secret"},
		{"AWS key AKIAIOSFODNN7EXAMPLE found", "AKIAIOSFODNN7EXAMPLE"},
	}
	for _, tc := range cases {
		out := r.RedactString(tc.input)
		if out == tc.input {
			t.Errorf("expected %q to be redacted in %q, got %q", tc.mustDrop, tc.input, out)
		}
	}
}

func TestStopOnceNoPanic(t *testing.T) {
	so := newStopOnce()
	// Calling stop multiple times must not panic
	so.stop()
	so.stop()
	so.stop()
}

func TestWALWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir+"/wal", 1000)
	if err != nil {
		t.Fatalf("NewWAL: %v", err)
	}
	defer wal.Close()

	evt := Event{
		Timestamp: "2026-01-01T00:00:00Z",
		Source:    "test",
		Type:      "unit_test",
		Host:      "testhost",
		AgentID:   "test-agent-id",
		Data:      map[string]interface{}{"key": "value"},
	}
	if err := wal.Write(evt); err != nil {
		t.Fatalf("WAL Write: %v", err)
	}

	events, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("WAL ReadAll: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "unit_test" {
		t.Errorf("expected type unit_test, got %s", events[0].Type)
	}
}

func TestWALCapEnforced(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir+"/wal", 3) // cap of 3
	if err != nil {
		t.Fatalf("NewWAL: %v", err)
	}
	defer wal.Close()

	evt := Event{Source: "test", Type: "t", Host: "h", AgentID: "a"}
	for i := 0; i < 3; i++ {
		if err := wal.Write(evt); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}
	// 4th write must be rejected
	if err := wal.Write(evt); err != ErrWALFull {
		t.Errorf("expected ErrWALFull, got %v", err)
	}
}

func TestWALTruncate(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir+"/wal", 1000)
	if err != nil {
		t.Fatalf("NewWAL: %v", err)
	}
	defer wal.Close()

	evt := Event{Source: "test", Type: "t", Host: "h", AgentID: "a"}
	for i := 0; i < 5; i++ {
		_ = wal.Write(evt)
	}
	if err := wal.Truncate(); err != nil {
		t.Fatalf("Truncate: %v", err)
	}
	events, _ := wal.ReadAll()
	if len(events) != 0 {
		t.Errorf("expected 0 events after truncate, got %d", len(events))
	}
}
