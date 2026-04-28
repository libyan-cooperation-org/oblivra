//go:build ignore
// +build ignore

// Scratch script — manual run only:
//   go run scratch/test_stability.go
// Excluded from `go build ./...` so the package's three main files
// don't fight each other for the symbol `main`.

package main

import (
	"fmt"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func MainTestStability() {
	log := logger.NewStdoutLogger()
	
	// Create an EXTREMELY expensive rule
	poisonRule := detection.Rule{
		ID:   "poison-rule",
		Name: "Expensive Regex Rule",
		Conditions: map[string]interface{}{
			"output_contains": "regex:(a+)+$", 
		},
		WindowSec: 3600,
	}
	
	for i := 0; i < 100; i++ {
		poisonRule.GroupBy = append(poisonRule.GroupBy, fmt.Sprintf("field%d", i))
	}

	fmt.Printf("[TEST] Rule Cost: %d (Max: %d)\n", poisonRule.ExecutionCost(), detection.MaxRuleCost)

	ev, _ := detection.NewEvaluator("/tmp/oblivra-test-rules", log)
	ev.UpsertRule(poisonRule)
	ev.RebuildRouteIndex() // IMPORTANT: Index must be rebuilt to pick up UpsertRule changes

	evt := detection.Event{
		EventType: "test",
		RawLog:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa!",
	}

	fmt.Println("[TEST] Processing event with expensive rule...")
	matches := ev.ProcessEvent(evt)
	
	if len(matches) == 0 {
		fmt.Println("[PASS] Expensive rule was throttled (skipped).")
	} else {
		fmt.Println("[FAIL] Expensive rule was NOT throttled!")
	}
}
