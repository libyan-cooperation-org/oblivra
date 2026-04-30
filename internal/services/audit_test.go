package services

import (
	"context"
	"log/slog"
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
