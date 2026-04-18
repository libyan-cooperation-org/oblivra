package detection

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestRegexAndCIDRConditions(t *testing.T) {
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: filepath.Join(tmpDir, "test.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	ruleYAML := `
id: "test_regex_cidr"
name: "Regex and CIDR Test"
description: "Tests regex and CIDR logic"
severity: "high"
type: "threshold"
threshold: 1
window_sec: 60
conditions:
  EventType: "regex:^failed_.*"
  source_ip: "cidr:192.168.1.0/24"
  output_contains: "regex:.*auth error.*"
`
	err = os.WriteFile(filepath.Join(tmpDir, "regex.yaml"), []byte(ruleYAML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	evaluator, err := NewEvaluator(tmpDir, log)
	if err != nil {
		t.Fatal(err)
	}

	evt := Event{
		EventType: "failed_login",
		SourceIP:  "192.168.1.150",
		RawLog:    "sshd: auth error occurred",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	matches := evaluator.ProcessEvent(evt)
	if len(matches) == 0 {
		t.Fatal("Expected match for valid regex and CIDR, got none")
	}

	// Test Failing Event
	evtFail := Event{
		EventType: "failed_login",
		SourceIP:  "10.0.0.5", // Fails CIDR
		RawLog:    "sshd: auth error occurred",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if len(evaluator.ProcessEvent(evtFail)) > 0 {
		t.Fatal("Expected no match for invalid CIDR")
	}
}

func TestSequenceRule(t *testing.T) {
	tmpDir := t.TempDir()
	log, err := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: filepath.Join(tmpDir, "test.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	ruleYAML := `
id: "brute_force_success"
name: "Brute Force Followed by Success"
description: "Multiple failed logins followed by a success from the same IP"
severity: "critical"
type: "sequence"
window_sec: 300
group_by: ["source_ip"]
sequence:
  - step_id: "fail1"
    conditions:
      EventType: "failed_login"
  - step_id: "fail2"
    conditions:
      EventType: "failed_login"
  - step_id: "success"
    conditions:
      EventType: "successful_login"
`
	err = os.WriteFile(filepath.Join(tmpDir, "seq.yaml"), []byte(ruleYAML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	evaluator, err := NewEvaluator(tmpDir, log)
	if err != nil {
		t.Fatal(err)
	}

	ip := "1.2.3.4"

	// Step 1: Fail
	matches := evaluator.ProcessEvent(Event{
		EventType: "failed_login",
		SourceIP:  ip,
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if len(matches) > 0 {
		t.Fatal("Alerted prematurely on step 1")
	}

	// Step 2: Fail
	matches = evaluator.ProcessEvent(Event{
		EventType: "failed_login",
		SourceIP:  ip,
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if len(matches) > 0 {
		t.Fatal("Alerted prematurely on step 2")
	}

	// Step 3: Success
	matches = evaluator.ProcessEvent(Event{
		EventType: "successful_login",
		SourceIP:  ip,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	if len(matches) == 0 {
		t.Fatal("Failed to alert on completed sequence")
	}

	if matches[0].Context["count"] != "3" {
		t.Errorf("Expected 3 events in sequence match, got %s", matches[0].Context["count"])
	}
}
