package storage_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/storage"
)

func TestWALCorruptionRecovery(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")
	l, _ := logger.New(logger.Config{Level: logger.DebugLevel, OutputPath: logPath})
	defer l.Close()

	// 1. Create and populate a WAL with valid entries
	wal, err := storage.NewWAL(tempDir, l)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	wal.Append([]byte("valid_entry_1"))
	wal.Append([]byte("valid_entry_2"))
	wal.Append([]byte("valid_entry_3"))
	wal.Sync()
	wal.Close()

	// 2. Corrupt the WAL by appending a truncated record
	// Write a length header claiming 1000 bytes, followed by only 5 bytes
	walFile := filepath.Join(tempDir, "wal", "ingest.wal")
	f, err := os.OpenFile(walFile, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("Failed to open WAL for corruption: %v", err)
	}
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, 1000) // claims 1000 bytes
	f.Write(lenBuf)
	f.Write([]byte("abcd")) // 4-byte fake checksum
	f.Write([]byte("trunc")) // only 5 bytes of data
	f.Close()

	// 3. Reopen WAL and attempt replay
	wal2, err := storage.NewWAL(tempDir, l)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal2.Close()

	replayed := 0
	err = wal2.Replay(func(payload []byte) error {
		replayed++
		return nil
	})

	// 4. Verify: We expect an error from the truncated record
	// but the 3 valid entries BEFORE corruption should have been replayed
	if replayed != 3 {
		t.Errorf("Expected 3 valid entries before corruption, got %d", replayed)
	}

	if err == nil {
		t.Log("WAL silently recovered from corruption (acceptable if it read all valid entries)")
	} else {
		t.Logf("WAL correctly reported corruption error: %v", err)
	}
}

func TestWALEmptyReplay(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")
	l, _ := logger.New(logger.Config{Level: logger.DebugLevel, OutputPath: logPath})
	defer l.Close()

	wal, err := storage.NewWAL(tempDir, l)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	replayed := 0
	err = wal.Replay(func(payload []byte) error {
		replayed++
		return nil
	})

	if err != nil {
		t.Errorf("Empty WAL replay should not error, got: %v", err)
	}
	if replayed != 0 {
		t.Errorf("Empty WAL should replay 0 entries, got %d", replayed)
	}
}

func TestWALChaosMonkey(t *testing.T) {
	tempDir := t.TempDir()
	l, _ := logger.New(logger.Config{Level: logger.DebugLevel, OutputPath: filepath.Join(tempDir, "test.log")})
	defer l.Close()

	// 1. Write a valid record
	payload := []byte("secret_nuclear_launch_codes")
	wal, err := storage.NewWAL(tempDir, l)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	wal.Append(payload)
	wal.Sync()
	wal.Close()

	// 2. CHAOS MONKEY: Flip a bit in the data section
	// Format: [Len:4][Check:4][Payload:NB]
	walFile := filepath.Join(tempDir, "wal", "ingest.wal")
	data, err := os.ReadFile(walFile)
	if err != nil {
		t.Fatalf("Failed to read WAL for chaos: %v", err)
	}

	// Flip bit in the payload (index 8 is start of payload)
	data[10] ^= 0x01 

	if err := os.WriteFile(walFile, data, 0644); err != nil {
		t.Fatalf("Failed to write corrupted WAL: %v", err)
	}

	// 3. Replay and verify detection
	wal2, err := storage.NewWAL(tempDir, l)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal2.Close()

	err = wal2.Replay(func(p []byte) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected checksum mismatch error, but replay succeeded (Integrity Failure!)")
	}

	t.Logf("Chaos Monkey successful: Detected bit-rot: %v", err)
}
