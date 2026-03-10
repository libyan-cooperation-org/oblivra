package detection

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// FuzzDetectionRuleEngine bombards the YAML compiler with malformed struct tags to
// verify the engine securely drops corrupted logic without poisoning memory or crashing.
func FuzzDetectionRuleEngine(f *testing.F) {
	// Valid rules seed
	f.Add(`
id: FZ-01
name: Fuzz Test Rule
description: Seed for fuzzing engine
severity: high
type: threshold
conditions:
  EventType: failed_login
threshold: 5
window_sec: 60
`)

	f.Fuzz(func(t *testing.T, payload string) {
		log, _ := logger.New(logger.Config{
			Level:      logger.DebugLevel,
			OutputPath: filepath.Join(t.TempDir(), "fuzz.log"),
		})
		defer log.Close()
		// Mock a temporary directory for the Engine to ingest from
		tempDir := t.TempDir()
		mockRulePath := filepath.Join(tempDir, "fuzz_rule.yaml")

		err := os.WriteFile(mockRulePath, []byte(payload), 0644)
		if err != nil {
			t.Fatalf("Failed to write mock rule: %v", err)
		}

		// Instantiate the engine and point it at the dirty payload
		// The fuzzer asserts that `yaml.Unmarshal(data, &rule)` inside `loadRuleFile`
		// handles garbage data securely and emits a ValidationResult rather than panicking.
		engine, err := NewRuleEngine(tempDir, log)
		
		// Expected behavior: The engine either cleanly loads it (if valid YAML),
		// or rejects it gracefully. It MUST NOT crash.
		if err != nil {
			return
		}
		
		_ = engine.GetVerificationResults()
	})
}
