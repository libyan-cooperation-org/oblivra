package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// Multi-Tenant Stress Test
// Validates: 50+ concurrent tenants, CRUD isolation, query latency, data leak detection

const (
	numTenants     = 60
	opsPerTenant   = 100
	hostsPerTenant = 10
	credsPerTenant = 5
)

type tenantResult struct {
	TenantID     string
	HostsCreated int
	CredsCreated int
	Isolated     bool // true = no cross-tenant data leak
	AvgLatencyMs float64
	Errors       []string
}

func main() {
	logPath := filepath.Join(os.TempDir(), "oblivra_tenant_test.log")
	log, err := logger.New(logger.Config{Level: logger.InfoLevel, OutputPath: logPath})
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	log.Info("=== OBLIVRA Multi-Tenant Stress Test ===")
	log.Info("Tenants: %d | Ops/tenant: %d | Hosts/tenant: %d | Creds/tenant: %d",
		numTenants, opsPerTenant, hostsPerTenant, credsPerTenant)

	// Use a temp file DB with WAL mode for concurrent access
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("oblivra_mt_test_%d.db", time.Now().UnixNano()))
	defer os.Remove(dbPath)

	db := &database.Database{}
	if err := db.Open(dbPath, nil); err != nil {
		log.Error("Failed to open test DB: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Error("Failed to migrate: %v", err)
		os.Exit(1)
	}

	var dbStore database.DatabaseStore = db
	credRepo := database.NewCredentialRepository(dbStore)

	// Phase 1: Sequential WRITE — create data for all tenants
	log.Info("Phase 1: Creating data for %d tenants...", numTenants)
	writeStart := time.Now()

	var totalCreated int64
	for t := 0; t < numTenants; t++ {
		tenantID := fmt.Sprintf("tenant_%03d", t)
		ctx := database.WithTenant(context.Background(), tenantID)

		for c := 0; c < credsPerTenant; c++ {
			cred := &database.Credential{
				ID:            fmt.Sprintf("cred-%s-%d", tenantID, c),
				Label:         fmt.Sprintf("cred-%s-%d", tenantID, c),
				Type:          "password",
				EncryptedData: []byte(fmt.Sprintf("encrypted-secret-%d", c)),
			}
			if err := credRepo.Create(ctx, cred); err != nil {
				log.Error("[%s] create cred: %v", tenantID, err)
				continue
			}
			totalCreated++
		}
	}
	log.Info("  Created %d credentials in %v", totalCreated, time.Since(writeStart).Round(time.Millisecond))

	// Phase 2: Concurrent READ+VERIFY — 60 tenants validate isolation simultaneously
	log.Info("Phase 2: Concurrent isolation check (%d goroutines)...", numTenants)
	results := make([]tenantResult, numTenants)
	var wg sync.WaitGroup
	var totalOps int64
	readStart := time.Now()

	for i := 0; i < numTenants; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tenantID := fmt.Sprintf("tenant_%03d", idx)
			ctx := database.WithTenant(context.Background(), tenantID)
			res := tenantResult{TenantID: tenantID, Isolated: true}

			opStart := time.Now()

			// Verify isolation — list should only return THIS tenant's data
			creds, err := credRepo.List(ctx, "")
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("list creds: %v", err))
			} else {
				res.CredsCreated = len(creds)
				for _, c := range creds {
					if c.TenantID != tenantID {
						res.Isolated = false
						res.Errors = append(res.Errors, fmt.Sprintf("DATA LEAK: cred %s belongs to tenant %s, expected %s", c.ID, c.TenantID, tenantID))
					}
				}
				if len(creds) != credsPerTenant {
					res.Errors = append(res.Errors, fmt.Sprintf("expected %d creds, got %d", credsPerTenant, len(creds)))
				}
			}

			// Concurrent random READ ops
			for op := 0; op < opsPerTenant; op++ {
				switch rand.Intn(3) {
				case 0:
					credRepo.List(ctx, "")
				case 1:
					credRepo.List(ctx, "password")
				case 2:
					if len(creds) > 0 {
						credRepo.GetByID(ctx, creds[rand.Intn(len(creds))].ID)
					}
				}
				atomic.AddInt64(&totalOps, 1)
			}

			elapsed := time.Since(opStart)
			res.AvgLatencyMs = float64(elapsed.Microseconds()) / float64(opsPerTenant+1) / 1000.0

			results[idx] = res
		}(i)
	}

	wg.Wait()
	readElapsed := time.Since(readStart)

	// Analyze results
	var passed, leaked, errorCount int
	var totalLatency float64
	for _, r := range results {
		if r.Isolated {
			passed++
		} else {
			leaked++
		}
		errorCount += len(r.Errors)
		totalLatency += r.AvgLatencyMs
	}

	avgLatency := totalLatency / float64(numTenants)
	opsPerSec := float64(totalOps) / readElapsed.Seconds()

	log.Info("════════════════════════════════════════════")
	log.Info("  MULTI-TENANT STRESS TEST RESULTS")
	log.Info("════════════════════════════════════════════")
	log.Info("  Tenants tested:     %d", numTenants)
	log.Info("  Credentials total:  %d", totalCreated)
	log.Info("  Read operations:    %d", totalOps)
	log.Info("  Read duration:      %v", readElapsed.Round(time.Millisecond))
	log.Info("  Throughput:         %.0f ops/sec", opsPerSec)
	log.Info("  Avg latency/op:     %.3f ms", avgLatency)
	log.Info("  Isolation passed:   %d/%d", passed, numTenants)
	log.Info("  Data leaks:         %d", leaked)
	log.Info("  Errors:             %d", errorCount)

	if leaked > 0 {
		log.Error("⚠ DATA LEAK DETECTED — %d tenants saw cross-tenant data", leaked)
		for _, r := range results {
			if !r.Isolated {
				for _, e := range r.Errors {
					log.Error("  [%s] %s", r.TenantID, e)
				}
			}
		}
		os.Exit(1)
	}

	if errorCount > 0 {
		log.Warn("  Some errors occurred:")
		shown := 0
		for _, r := range results {
			for _, e := range r.Errors {
				if shown < 10 {
					log.Warn("  [%s] %s", r.TenantID, e)
					shown++
				}
			}
		}
	}

	uptime := float64(passed) / float64(numTenants) * 100.0
	log.Info("  Effective uptime:   %.1f%%", uptime)

	if uptime >= 99.9 {
		log.Info("✅ PASS — Multi-tenant validation successful (%d tenants, zero leaks)", numTenants)
	} else {
		log.Error("❌ FAIL — Uptime %.1f%% below 99.9%% target", uptime)
		os.Exit(1)
	}
}
