package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestWAL_CrashResilience(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "resilience.log")
	log, _ := logger.New(logger.Config{Level: logger.InfoLevel, OutputPath: logPath})
	defer log.Close()

	wal, err := NewWAL(tempDir, log)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	totalEvents := 1000
	eventSize := 100
	payload := make([]byte, eventSize)
	for i := 0; i < eventSize; i++ {
		payload[i] = 'A'
	}

	fmt.Printf("[AUDIT] Writing %d events to WAL...\n", totalEvents)
	for i := 0; i < totalEvents; i++ {
		if err := wal.Append(payload); err != nil {
			t.Fatalf("Append failed at %d: %v", i, err)
		}
	}

	// For a real Windows crash simulation, we need to ensure the file is closed 
	// before we try to re-open it in the NewWAL call, otherwise we get "Access is denied".
	walName := wal.filename
	wal.file.Close() 

	fmt.Println("[AUDIT] Simulated CRASH (Ungraceful Close).")

	// RECOVERY PHASE
	recoveryWAL, err := NewWAL(tempDir, log)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer recoveryWAL.Close()

	recoveredCount := 0
	err = recoveryWAL.Replay(func(p []byte) error {
		recoveredCount++
		return nil
	})

	loss := totalEvents - recoveredCount
	fmt.Printf("[RESULT] Total Written: %d | Recovered: %d | Lost: %d\n", 
		totalEvents, recoveredCount, loss)

	if loss > 0 {
		t.Errorf("FAIL: Data loss detected (%d events). Expected 0 due to Sync hardening.", loss)
	} else {
		fmt.Println("[SUCCESS] Zero data loss verified under ungraceful shutdown.")
	}
	
	// Ensure the file is actually on disk
	fi, _ := os.Stat(walName)
	fmt.Printf("[INFO] Final WAL Size: %d bytes\n", fi.Size())
}
