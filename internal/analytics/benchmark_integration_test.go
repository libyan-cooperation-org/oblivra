package analytics

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestBenchmark_InferenceAccuracy(t *testing.T) {
	datasetPath := filepath.Join("..", "..", "test", "datasets", "benchmark_1.json")
	
	runner := NewBenchmarkRunner()
	result, err := runner.RunBenchmark(datasetPath)
	if err != nil {
		t.Fatalf("Benchmark execution failed: %v", err)
	}

	fmt.Printf("\n--- OBLIVRA DETECTION BENCHMARK RESULTS ---\n")
	fmt.Printf("Dataset: %s\n", datasetPath)
	fmt.Printf("Total Events:     %d\n", result.TotalEvents)
	fmt.Printf("Expected Alerts:  %d\n", result.ExpectedAlerts)
	fmt.Printf("Total Detections: %d\n", result.TotalAlerts)
	fmt.Printf("True Positives:   %d\n", result.TruePositives)
	fmt.Printf("False Positives:  %d\n", result.FalsePositives)
	fmt.Printf("Precision:        %.2f%%\n", result.Precision*100)
	fmt.Printf("Recall:           %.2f%%\n", result.Recall*100)
	fmt.Printf("------------------------------------------\n\n")

	// We expect 100% accuracy on our own builtin rules for this sample
	if result.Precision < 1.0 || result.Recall < 1.0 {
		t.Errorf("expected 100%% precision/recall on sample dataset, got P=%.2f R=%.2f", result.Precision, result.Recall)
	}
}
