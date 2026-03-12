package app

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/core"
)

func TestFullFlow(t *testing.T) {
	// 1. Setup Isolated Environment
	tmpDir := t.TempDir()

	// Override environment variables to point to temp directory
	os.Setenv("APPDATA", tmpDir)
	os.Setenv("LOCALAPPDATA", tmpDir)

	l, err := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: filepath.Join(tmpDir, "test.log"),
	})
	if err != nil {
		t.Fatalf("Failed to init logger: %v", err)
	}

	// 2. Initialize Container
	baseCtx := context.Background()
	ctx := context.WithValue(baseCtx, "test", "true")
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	container := core.NewContainer(l, "test-version")
	if err := container.Init(ctx); err != nil {
		t.Fatalf("Container Init failed: %v", err)
	}

	// Start all services via Kernel
	kernel, err := platform.NewKernel(container.Registry)
	if err != nil {
		t.Fatalf("Failed to initialize platform kernel: %v", err)
	}
	container.Kernel = kernel

	if err := container.Kernel.Start(); err != nil {
		t.Fatalf("Kernel Start failed: %v", err)
	}
	defer func() {
		container.Kernel.Stop()
		container.Close()
	}()

	// Wait for services to stabilize
	time.Sleep(500 * time.Millisecond)

	// --- MANDATORY: Unlock Vault Early ---
	// AlertingService needs the database to be unlocked to record incidents.
	masterKey := "test-master-key"
	if err := container.Product.VaultService.Setup(masterKey, ""); err != nil {
		t.Fatalf("Early Vault Setup failed: %v", err)
	}
	if err := container.Product.VaultService.Unlock(masterKey, nil, false); err != nil {
		t.Fatalf("Early Vault Unlock failed: %v", err)
	}

	// 3. Test Detection Layer (SIEM Ingestion)
	t.Run("SIEM_Ingestion_and_Search", func(t *testing.T) {
		evt := &ingest.SovereignEvent{
			Host:      "test-host-1",
			Timestamp: time.Now().Format(time.RFC3339),
			EventType: "failed_login",
			SourceIp:  "10.0.0.1",
			User:      "admin",
			RawLine:   "Failed password for admin from 10.0.0.1 port 22 ssh2 (INTEGRATION_TEST)",
		}

		container.SIEM.IngestService.QueueEvent(evt)

		// Bleve indexing is async, wait a bit
		var results []database.HostEvent
		var searchErr error
		for i := 0; i < 20; i++ {
			results, searchErr = container.SIEM.SIEMService.SearchHostEvents("INTEGRATION_TEST", 10)
			if searchErr == nil && len(results) > 0 {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		if searchErr != nil {
			t.Errorf("SIEM Search failed: %v", searchErr)
		}
		if len(results) == 0 {
			t.Errorf("Expected 1 SIEM result, got 0")
		} else if results[0].User != "admin" {
			t.Errorf("Expected user 'admin', got '%s'", results[0].User)
		}
	})

	// 4. Test Alerting Layer
	t.Run("Alerting_Trigger", func(t *testing.T) {
		alertFired := make(chan bool, 1)
		container.Infra.Bus.Subscribe("security.alert", func(e eventbus.Event) {
			t.Logf("Alert received: %v", e.Data)
			alertFired <- true
		})

		// Track ingestion completion
		eventsProcessed := make(chan bool, 1)
		count := 0
		target := 60 // 5 IPs * 10 attempts + 10 root attempts
		container.Infra.Bus.Subscribe("siem.event_indexed", func(e eventbus.Event) {
			evt, ok := e.Data.(database.HostEvent)
			if ok && evt.HostID == "test-host-alert" {
				count++
				if count >= target {
					select {
					case eventsProcessed <- true:
					default:
					}
				}
			}
		})

		// Trigger multiple failed logins to cause a heuristic alert (Score >= 70)
		ips := []string{"192.168.1.101", "192.168.1.102", "192.168.1.103", "192.168.1.104", "192.168.1.105"}
		for _, ip := range ips {
			for i := 0; i < 10; i++ {
				evt := &ingest.SovereignEvent{
					Host:      "test-host-alert",
					Timestamp: time.Now().Format(time.RFC3339),
					EventType: "failed_login",
					SourceIp:  ip,
					User:      "user" + strconv.Itoa(i),
					RawLine:   "Brute force attempt from " + ip,
				}
				container.SIEM.IngestService.QueueEvent(evt)
				time.Sleep(2 * time.Millisecond) // Ensure unique timestamps
			}
		}

		// Add root attempts
		for i := 0; i < 10; i++ {
			evt := &ingest.SovereignEvent{
				Host:      "test-host-alert",
				Timestamp: time.Now().Format(time.RFC3339),
				EventType: "failed_login",
				SourceIp:  "192.168.1.106",
				User:      "root",
				RawLine:   "Root target attempt",
			}
			container.SIEM.IngestService.QueueEvent(evt)
			time.Sleep(2 * time.Millisecond)
		}

		// Wait for ingestion to finish
		select {
		case <-eventsProcessed:
			t.Log("All events processed by pipeline")
		case <-time.After(15 * time.Second):
			t.Fatalf("Ingestion too slow, processed %d/%d", count, target)
		}

		// Pulse the audit completion to trigger calculation
		container.Infra.Bus.Publish("siem.audit_completed", map[string]string{
			"host_id": "test-host-alert",
		})

		select {
		case <-alertFired:
			t.Log("Alert successfully triggered")
		case <-time.After(10 * time.Second):
			t.Error("Timed out waiting for security alert")
		}
	})

	// 5. Test Vault Interaction
	t.Run("Vault_Operations", func(t *testing.T) {
		// Vault is already setup and unlocked at this point
		if !container.Product.VaultService.IsUnlocked() {
			t.Error("Vault should be unlocked")
		}

		_, err = container.Product.VaultService.AddCredential("Test Cred", "ssh_password", "supersecretpassword")
		if err != nil {
			t.Errorf("Failed to store credential: %v", err)
		}

		creds, err := container.Product.VaultService.ListCredentials("")
		if err != nil || len(creds) == 0 {
			t.Errorf("Failed to retrieve credentials: %v", err)
		}
	})

	// 6. Test Governance (Compliance Reporting)
	t.Run("Compliance_Report", func(t *testing.T) {
		start := time.Now().Add(-1 * time.Hour).Unix()
		end := time.Now().Add(1 * time.Hour).Unix()
		report, err := container.Product.ComplianceService.GenerateReport("security_audit", start, end)
		if err != nil {
			t.Errorf("Report generation failed: %v", err)
		}
		if report == nil {
			t.Error("Generated report is nil")
		}
	})

	// 7. Cleanup Grace Period
	time.Sleep(2 * time.Second)
}
