package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/search"
	"github.com/kingknull/oblivrashell/internal/storage"
)

func main() {
	countFlag := flag.Int("count", 100000, "Number of events to generate")
	flag.Parse()

	// 1. Setup Data Directory
	tmpDir, err := os.MkdirTemp("", "bench_siem")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("=> Initializing Storage (BadgerDB + Bleve) in %s\n", tmpDir)

	cfg := logger.Config{
		Level:      logger.InfoLevel,
		OutputPath: filepath.Join(tmpDir, "bench.log"),
	}
	l, err := logger.New(cfg)
	if err != nil {
		log.Fatalf("Init Logger failed: %v", err)
	}

	// Init BadgerDB
	hotStore, err := storage.NewHotStore(tmpDir, l)
	if err != nil {
		log.Fatalf("Init HotStore failed: %v", err)
	}
	defer hotStore.Close()

	// Init Bleve
	searchEngine, err := search.NewSearchEngine(tmpDir, l)
	if err != nil {
		log.Fatalf("Init SearchEngine failed: %v", err)
	}
	defer searchEngine.Close()

	// Init SIEM Repo
	// (Note: To pass double pointer we create a local reference)
	searchRef := searchEngine
	siemRepo := storage.NewBadgerSIEMRepository(hotStore, &searchRef, nil)

	// 2. Bulk Data Generation
	fmt.Printf("\n=> Inserting %d synthetic SIEM events...\n", *countFlag)

	startIngest := time.Now()

	// We'll insert in batches for speed but use the standard API
	for i := 0; i < *countFlag; i++ {
		hostID := fmt.Sprintf("host_%d", i%10)
		evtType := "failed_login"
		if i%3 == 0 {
			evtType = "sudo_exec"
		}

		evt := &database.HostEvent{
			HostID:    hostID,
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			EventType: evtType,
			SourceIP:  fmt.Sprintf("192.168.1.%d", i%255),
			User:      "root",
			RawLog:    fmt.Sprintf("Failed password for root from 192.168.1.%d port 22 ssh2", i%255),
		}

		// Inject an anomaly we can find later
		if i == *countFlag/2 {
			evt.RawLog = "CRITICAL_ANOMALY_PAYLOAD_DETECTED"
		}

		if err := siemRepo.InsertHostEvent(context.Background(), evt); err != nil {
			log.Fatalf("Insert failed at %d: %v", i, err)
		}

		if i > 0 && i%50000 == 0 {
			fmt.Printf("   ... %d inserted\n", i)
		}
	}

	ingestDuration := time.Since(startIngest)
	eps := float64(*countFlag) / ingestDuration.Seconds()
	fmt.Printf("=> Ingestion Complete: %s (%.0f events/sec)\n", ingestDuration, eps)

	// 3. Search Benchmarking
	fmt.Printf("\n=> Running Sub-5 Second Benchmark Tests...\n")

	queries := []string{
		"failed",
		"sudo",
		"CRITICAL_ANOMALY_PAYLOAD_DETECTED",
	}

	for _, q := range queries {
		startSearch := time.Now()
		results, err := siemRepo.SearchHostEvents(context.Background(), q, 100)
		searchDuration := time.Since(startSearch)

		if err != nil {
			log.Fatalf("Search for %q failed: %v", q, err)
		}

		if searchDuration.Seconds() > 5.0 {
			fmt.Printf("❌ FAIL: Search for %q took %s (exceeded 5s limit)\n", q, searchDuration)
		} else {
			fmt.Printf("✅ PASS: Search for %q took %s (Found %d hits)\n", q, searchDuration, len(results))
		}
	}

	fmt.Println("\n=> Benchmark Suite Finished.")
}
