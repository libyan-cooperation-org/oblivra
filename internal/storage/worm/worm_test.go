package worm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockUnlockRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "warm.parquet")
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Lock(path); err != nil {
		t.Fatal(err)
	}
	locked, err := IsLocked(path)
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Error("expected locked")
	}
	// Writing should fail.
	if err := os.WriteFile(path, []byte("mutated"), 0o644); err == nil {
		t.Error("write to locked file should fail")
	}
	// Unlock and confirm we can write again.
	if err := Unlock(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("rewritten"), 0o644); err != nil {
		t.Fatalf("write after unlock failed: %v", err)
	}
}
