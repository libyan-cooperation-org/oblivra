package analytics

import (
	"encoding/json"
	"os"
	"testing"
)

func TestBenchmarkRunner_RunBenchmark(t *testing.T) {
	// 1. Create a temporary dataset file
	events := []BenchmarkEvent{
		{Type: "log", Payload: "Accepted password for root", Host: "server-1", ExpectAlert: true},
		{Type: "log", Payload: "ls -la", Host: "server-1", ExpectAlert: false},
		{Type: "log", Payload: "Kernel panic", Host: "server-2", ExpectAlert: true},
		{Type: "log", Payload: "whoami", Host: "server-1", ExpectAlert: false},
	}
	
	tmpFile, err := os.CreateTemp("", "benchmark-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	data, _ := json.Marshal(events)
	os.WriteFile(tmpFile.Name(), data, 0644)

	// 2. Run the benchmark
	runner := NewBenchmarkRunner()
	result, err := runner.RunBenchmark(tmpFile.Name())
	if err != nil {
		t.Fatalf("Benchmark failed: %v", err)
	}

	// 3. Verify results
	if result.TotalEvents != 4 {
		t.Errorf("expected 4 events, got %d", result.TotalEvents)
	}
	if result.TruePositives != 2 {
		t.Errorf("expected 2 true positives, got %d", result.TruePositives)
	}
	if result.TotalAlerts != 2 {
		t.Errorf("expected 2 total alerts, got %d", result.TotalAlerts)
	}
	if result.Precision != 1.0 {
		t.Errorf("expected 1.0 precision, got %f", result.Precision)
	}
	if result.Recall != 1.0 {
		t.Errorf("expected 1.0 recall, got %f", result.Recall)
	}
}
