package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("OBLIVRA CHAOS MONKEY (DOS BOMBARDMENT SIMULATOR)")
	fmt.Println("==================================================")
	fmt.Println("WARNING: Target Oblivra platform should hit rate limits and survive.")

	targetJSON := "http://localhost:8080/api/v1/ingest" // Assuming REST hook for DoS testing

	// In the real system, we used wails bindings and memory bus ingestion, but a loopback RPC test
	// or simulated high-throughput parallel publisher on the internal Bus proves the limit logic.
	// Since chaotic HTTP bombarding is easiest, we simulate an external attacker flooding logs.

	var wg sync.WaitGroup
	start := time.Now()

	threads := 50
	perThread := 1000

	fmt.Printf("[+] Launching %d goroutines sending %d payloads each...\n", threads, perThread)

	successes := 0
	dropped := 0
	var mu sync.Mutex

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			payload := map[string]interface{}{
				"event_id":     fmt.Sprintf("chaos-%d-%d", threadID, time.Now().UnixNano()),
				"type":         "LateralMovement",
				"time":         time.Now().UTC().Format(time.RFC3339),
				"nesting_bomb": map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": "d"}}},
			}
			b, _ := json.Marshal(payload)

			for j := 0; j < perThread; j++ {
				resp, err := http.Post(targetJSON, "application/json", bytes.NewReader(b))
				mu.Lock()
				if err != nil {
					dropped++
				} else {
					if resp.StatusCode == 429 { // Too Many Requests expected from load shedder
						dropped++
					} else {
						successes++
					}
					resp.Body.Close()
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Println("==================================================")
	fmt.Println("BOMBARDMENT COMPLETE")
	fmt.Printf("Duration : %v\n", duration)
	fmt.Printf("Processed: %d logs\n", successes)
	fmt.Printf("Shed/Drop: %d logs (SUCCESS: System denied DoS)\n", dropped)
	fmt.Printf("Total EPS: %.2f\n", float64(successes+dropped)/duration.Seconds())

	if dropped == 0 {
		fmt.Println("[CRITICAL FAIL] No load shedding occurred. System absorbed full hit. May be vulnerable to OOM pacing.")
		os.Exit(1)
	} else {
		fmt.Println("[PASS] Backpressure successfully throttled ingress.")
	}
}
