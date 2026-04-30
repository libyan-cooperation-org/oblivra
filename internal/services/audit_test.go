package services

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditChainVerifies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	a := NewAuditService(logger, []byte("secret"))

	ctx := context.Background()
	a.Append(ctx, "alice", "siem.search", "default", map[string]string{"q": "sshd"})
	a.Append(ctx, "alice", "alert.ack", "default", map[string]string{"id": "abc"})
	a.Append(ctx, "system", "evidence.seal", "default", map[string]string{"id": "ev1"})

	res := a.Verify()
	if !res.OK {
		t.Fatalf("verify failed: brokenAt=%d", res.BrokenAt)
	}
	if res.Entries != 3 {
		t.Errorf("entries = %d", res.Entries)
	}
	if res.RootHash == "" {
		t.Errorf("root hash empty")
	}
}

func TestAuditTamperDetected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	a := NewAuditService(logger, nil)

	ctx := context.Background()
	a.Append(ctx, "alice", "first", "default", nil)
	a.Append(ctx, "alice", "second", "default", nil)

	// Tamper directly with the in-memory slice — Verify must catch it.
	a.entries[0].Action = "TAMPERED"

	res := a.Verify()
	if res.OK {
		t.Fatalf("verify should fail after tamper, got %+v", res)
	}
	if res.BrokenAt != 1 {
		t.Errorf("brokenAt = %d (want 1)", res.BrokenAt)
	}
}

func TestEvidencePackageRefusesBrokenChain(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	a := NewAuditService(logger, []byte("k"))
	ctx := context.Background()
	a.Append(ctx, "x", "y", "default", nil)
	a.entries[0].Hash = "00"
	if _, err := a.GeneratePackage(ctx); err == nil {
		t.Fatal("expected refusal on broken chain")
	}
}

func TestDurableJournalSurvivesRestart(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	key := []byte("journal-key")

	// First lifetime: write three entries and close.
	a, err := NewDurable(logger, dir, key)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	a.Append(ctx, "alice", "first", "default", map[string]string{"k": "v1"})
	a.Append(ctx, "alice", "second", "default", map[string]string{"k": "v2"})
	a.Append(ctx, "system", "third", "default", nil)
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}

	// Second lifetime: re-open, replay, verify.
	b, err := NewDurable(logger, dir, key)
	if err != nil {
		t.Fatalf("re-open failed: %v", err)
	}
	if r := b.Verify(); !r.OK || r.Entries != 3 {
		t.Fatalf("verify after restart: %+v", r)
	}
	// Recent should still return the 3 entries newest-first.
	got := b.Recent(10)
	if len(got) != 3 || got[0].Action != "third" {
		t.Errorf("recent ordering wrong: got %d entries, head=%q", len(got), got[0].Action)
	}
	_ = b.Close()
}

func TestDurableJournalDetectsTampering(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()

	a, err := NewDurable(logger, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.Append(context.Background(), "x", "first", "default", nil)
	a.Append(context.Background(), "x", "second", "default", nil)
	_ = a.Close()

	// Mutate the on-disk journal.
	path := filepath.Join(dir, "audit.log")
	body, _ := os.ReadFile(path)
	idx := -1
	for i := range body {
		if body[i] == '\n' {
			idx = i
			break
		}
	}
	if idx > 0 {
		// Flip a byte inside the first entry's action field.
		mutated := make([]byte, len(body))
		copy(mutated, body)
		// Find "first" and turn it into "FIRST".
		for i := 0; i < idx-5; i++ {
			if string(mutated[i:i+5]) == "first" {
				copy(mutated[i:], "FIRST")
				break
			}
		}
		_ = os.WriteFile(path, mutated, 0o600)
	}

	if _, err := NewDurable(logger, dir, nil); err == nil {
		t.Fatal("expected NewDurable to refuse a tampered journal")
	}
}

func TestDurableJournalEmptyDir(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	a, err := NewDurable(logger, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r := a.Verify(); !r.OK || r.Entries != 0 {
		t.Errorf("fresh journal verify: %+v", r)
	}
	_ = a.Close()
}
