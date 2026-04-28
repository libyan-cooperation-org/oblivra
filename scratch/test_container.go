//go:build ignore
// +build ignore

// Scratch script — manual run only:
//   go run scratch/test_container.go
// Excluded from `go build ./...` so the package's three main files
// don't fight each other for the symbol `main`.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kingknull/oblivrashell/internal/core"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func MainTestContainer() {
	os.Setenv("OBLIVRA_ISOLATED_VAULT", "true")
	log := logger.NewStdoutLogger()
	
	// We need a dummy registry/config etc for Container
	// But let's just use the Init function if we can.
	// Actually, I'll just use the code from container.go directly to test.
	
	c := core.NewContainer(log, "dev")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("[TEST] Initializing Container...")
	if err := c.Init(ctx); err != nil {
		fmt.Printf("[FAIL] Init failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[PASS] Container initialized. Vault should be running.")
	
	// Wait a bit to ensure heartbeat starts
	time.Sleep(2 * time.Second)
	
	// Verify vault is running
	if _, err := os.Stat("/tmp/oblivra-vault.sock"); err != nil {
		fmt.Printf("[FAIL] Socket not found: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[PASS] Vault socket found.")

	fmt.Println("[TEST] Killing vault daemon...")
	os.Remove("/tmp/oblivra-vault.sock") // Simplified "kill" for test
	
	fmt.Println("[TEST] Waiting for heartbeat recovery (may take up to 40s)...")
	// The heartbeat is 30s.
	start := time.Now()
	recovered := false
	for time.Since(start) < 45*time.Second {
		if _, err := os.Stat("/tmp/oblivra-vault.sock"); err == nil {
			fmt.Printf("[PASS] Vault recovered in %v!\n", time.Since(start))
			recovered = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !recovered {
		fmt.Println("[FAIL] Vault did not recover in time.")
		os.Exit(1)
	}

	fmt.Println("[SUCCESS] All resilience tests passed.")
}
