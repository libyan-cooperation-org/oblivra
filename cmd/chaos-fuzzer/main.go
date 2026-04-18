package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// This standalone tool simulates an external physical human red team consultancy
// utilizing automated exploit fuzzing against the Domain 4 requirements.
func main() {
	fmt.Println("==================================================")
	fmt.Println("OBLIVRA PRE-AUDIT RED TEAM FUZZER (DOMAIN 4)")
	fmt.Println("==================================================")

	targetBase := "http://localhost:8080" // Assuming local dev daemon runs back on 8080 HTTP

	// Create an HTTP client that skips TLS verification for loopback fuzzing
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	var wg sync.WaitGroup

	// Test 1: API Fuzzing & Deserialization Panics
	fmt.Println("\n[+] TASK 1: API Malformed JSON & Null-Byte Fuzzing")
	payloads := []string{
		`{"id": "test", "value": "}`,                                     // Malformed JSON
		`{"id": "test", "value": "data\u0000hidden"}`,                    // Null-Byte injection
		strings.Repeat(`{"a": `, 100) + `"b"` + strings.Repeat(`}`, 100), // Deep nesting max-depth bomb
		`{"id": "sql", "value": "admin' OR '1'='1"}`,                     // Legacy SQLi fallback
	}

	for i, payload := range payloads {
		wg.Add(1)
		go func(id int, p string) {
			defer wg.Done()
			req, _ := http.NewRequest("POST", targetBase+"/api/v1/ingest", bytes.NewBufferString(p))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", "oblivra-dev-key") // Use valid auth to reach parser

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("[FAIL] Fuzz %d caused a connection crash: %v\n", id, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 500 {
				fmt.Printf("[FAIL] Fuzz %d triggered a 500 Internal Server Error panic!\n", id)
			} else {
				fmt.Printf("[PASS] Fuzz %d blocked gracefully. Code: %d\n", id, resp.StatusCode)
			}
		}(i, payload)
	}

	// Wait for API tests
	wg.Wait()

	// Test 2: ReDoS (Regular Expression Denial of Service)
	fmt.Println("\n[+] TASK 2: ReDoS Syslog Parsers")
	evilSyslog := `<165>1 2003-10-11T22:14:15.003Z server.example.com ` + strings.Repeat("evil ", 50000) + "!"

	start := time.Now()
	req, _ := http.NewRequest("POST", targetBase+"/api/v1/ingest?type=syslog", bytes.NewBufferString(evilSyslog))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-API-Key", "oblivra-dev-key")

	resp, err := client.Do(req)
	duration := time.Since(start)

	if err == nil {
		defer resp.Body.Close()
		fmt.Printf("[PASS] Syslog parser ingested evil string in %v. Status: %d\n", duration, resp.StatusCode)
		if duration > 2*time.Second {
			fmt.Printf("[WARN] Parsing took %v, indicating potential regex backtracking lag.\n", duration)
		}
	} else {
		fmt.Printf("[FAIL] Syslog parser crashed the connection: %v\n", err)
	}

	// Test 3: WAL Corruption Injection
	fmt.Println("\n[+] TASK 3: Write-Ahead Log (WAL) Bit-Flip Corruption")
	walPath := "data/ledger.wal"
	if _, err := os.Stat(walPath); err == nil {
		f, err := os.OpenFile(walPath, os.O_RDWR, 0644)
		if err == nil {
			f.Seek(100, 0)
			f.Write([]byte{0xFF, 0x00, 0xFF, 0x00}) // Inject junk bytes dead center
			f.Close()
			fmt.Println("[PASS] Injected WAL corruption. Daemon must recover gracefully on next reboot.")
		}
	} else {
		fmt.Println("[-] WAL file not found at data/ledger.wal, skipping physical corruption step.")
	}

	fmt.Println("\n==================================================")
	fmt.Println("FUZZING HARNESS COMPLETE")
	fmt.Println("==================================================")
}
