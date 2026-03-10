package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

func main() {
	target := flag.String("target", "127.0.0.1:1514", "Syslog server target UDP network address")
	eps := flag.Int("eps", 5000, "Events per second to generate")
	duration := flag.Duration("duration", 30*time.Second, "Duration of the soak test")
	workers := flag.Int("workers", 4, "Number of concurrent generation workers")
	flag.Parse()

	fmt.Printf("=> Starting SIEM Sovereign Soak Test\n")
	fmt.Printf("   Target: %s\n", *target)
	fmt.Printf("   EPS: %d | Duration: %s | Workers: %d\n", *eps, *duration, *workers)

	conn, err := net.Dial("udp", *target)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", *target, err)
	}
	defer conn.Close()

	var wg sync.WaitGroup
	sentChan := make(chan int, 1000)
	
	// EPS per worker
	workerEps := *eps / *workers
	interval := time.Second / time.Duration(workerEps)

	start := time.Now()

	// 1. Send START marker
	startMarker := fmt.Sprintf("<34>1 %s soak_machine su: SOAK_TEST_START_MARKER\n", time.Now().Format(time.RFC3339))
	conn.Write([]byte(startMarker))

	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			
			timer := time.NewTimer(*duration)
			defer timer.Stop()

			workerSent := 0
			for {
				select {
				case <-timer.C:
					sentChan <- workerSent
					return
				case t := <-ticker.C:
					msg := fmt.Sprintf("<34>1 %s load_gen_%d su: 'su root' failed for user_%d on /dev/pts/%d\n", 
						t.Format(time.RFC3339), id, workerSent%100, id)
					conn.Write([]byte(msg))
					workerSent++
				}
			}
		}(w)
	}

	// Metrics collector
	go func() {
		wg.Wait()
		close(sentChan)
	}()

	totalSent := 0
	for count := range sentChan {
		totalSent += count
	}

	elapsed := time.Since(start)
	actualEps := float64(totalSent) / elapsed.Seconds()

	// 2. Send END marker
	endMarker := fmt.Sprintf("<34>1 %s soak_machine su: SOAK_TEST_END_MARKER\n", time.Now().Format(time.RFC3339))
	conn.Write([]byte(endMarker))

	fmt.Printf("\n=> Sovereign Soak Test Complete!\n")
	fmt.Printf("   Total Sent: %d events in %.2fs\n", totalSent, elapsed.Seconds())
	fmt.Printf("   Throughput: %.0f EPS (%.1f%% of target)\n", actualEps, (actualEps/float64(*eps))*100)
}
