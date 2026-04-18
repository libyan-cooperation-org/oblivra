package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func main() {
	rulesDir := flag.String("rules", "internal/detection/rules", "Directory containing YAML detection rules")
	datasetDir := flag.String("datasets", "test/datasets", "Directory containing JSON benchmark datasets")
	flag.Parse()

	// 1. Setup Logger
	cfg := logger.Config{
		Level:      logger.InfoLevel,
		OutputPath: "bench_research.log",
	}
	l, err := logger.New(cfg)
	if err != nil {
		log.Fatalf("Init Logger failed: %v", err)
	}

	// 2. Init Benchmark Runner
	runner, err := detection.NewBenchmarkRunner(*rulesDir, l)
	if err != nil {
		log.Fatalf("Init BenchmarkRunner failed: %v", err)
	}

	datasets := []string{
		"cic_ids_2017.json",
		"zeek_traces.json",
		"benchmark_1.json",
	}

	fmt.Printf("\n--- OBLIVRA ELITE RESEARCH BENCHMARK SUITE ---\n\n")

	for _, ds := range datasets {
		path := filepath.Join(*datasetDir, ds)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("⚠️ Skip: Dataset %s not found\n", ds)
			continue
		}

		result, err := runner.RunBenchmark(path)
		if err != nil {
			fmt.Printf("❌ Error: %s failed: %v\n", ds, err)
			continue
		}

		fmt.Printf("Dataset: %s\n", ds)
		fmt.Printf("  Total Events:     %d\n", result.TotalEvents)
		fmt.Printf("  Expected Alerts:  %d\n", result.ExpectedAlerts)
		fmt.Printf("  Total Detections: %d\n", result.TotalAlerts)
		fmt.Printf("  True Positives:   %d\n", result.TruePositives)
		fmt.Printf("  False Positives:  %d\n", result.FalsePositives)
		fmt.Printf("  Precision:        %.2f%%\n", result.Precision*100)
		fmt.Printf("  Recall:           %.2f%%\n", result.Recall*100)
		fmt.Printf("  Duration:         %s\n", result.Duration)
		fmt.Printf("------------------------------------------\n")
	}

	fmt.Println("\n=> Research Benchmark Suite Finished.")
}
