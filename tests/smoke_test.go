package architecture_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/app"
)

func TestIntegrationSmoke(t *testing.T) {
	// 1. Setup Temp Environment to avoid polluting production data
	tempDir := t.TempDir()
	os.Setenv("APPDATA", tempDir)
	os.Setenv("LOCALAPPDATA", tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Add a test marker to the context to skip Wails UI emissions and other non-headless side effects
	ctx = context.WithValue(ctx, "test", "true")

	// 2. Initialize App Container
	application := app.New()
	application.Startup(ctx)
	defer application.Shutdown(ctx)

	// 3. Vault Integrity Check
	t.Run("Vault_Integrity", func(t *testing.T) {
		password := "sovereign-test-pass"

		// Initial setup for a fresh temp directory
		err := application.VaultService.Setup(password, "")
		if err != nil {
			t.Fatalf("Failed to setup vault: %v", err)
		}

		err = application.VaultService.Unlock(password, nil, false)
		if err != nil {
			t.Fatalf("Failed to unlock vault: %v", err)
		}

		// Verify we can add and list a credential
		label := "SmokeTestSecret"
		credType := "password"
		rawData := "integration-test-value"

		id, err := application.VaultService.AddCredential(label, credType, rawData)
		if err != nil {
			t.Fatalf("Failed to add credential: %v", err)
		}

		if id == "" {
			t.Fatal("Expected non-empty credential ID")
		}

		decrypted, err := application.VaultService.GetDecryptedCredential(id)
		if err != nil {
			t.Fatalf("Failed to decrypt: %v", err)
		}

		if decrypted != rawData {
			t.Errorf("Decrypted data mismatch. Expected %s, got %s", rawData, decrypted)
		}

		// Cleanup: Lock vault
		application.VaultService.Lock()
	})

	// 4. SIEM Pipeline Check (Ingestion -> Search)
	t.Run("SIEM_Pipeline", func(t *testing.T) {
		// Ensure vault is unlocked for DB access
		password := "sovereign-test-pass"
		application.VaultService.Unlock(password, nil, false)
		defer application.VaultService.Lock()

		// Start Syslog Server
		err := application.IngestService.StartSyslogServer()
		if err != nil {
			t.Fatalf("Failed to start syslog server: %v", err)
		}
		defer application.IngestService.StopSyslogServer()

		// Send mock RFC5424 syslog message over UDP
		conn, err := net.Dial("udp", "127.0.0.1:1514")
		if err != nil {
			t.Fatalf("Failed to dial syslog: %v", err)
		}
		// A classic failed login log
		msg := "<34>1 2023-10-11T22:14:15.003Z smoke-host oblivra - ID47 - [meta sequence=\"1\"] 'su root' failed for root on /dev/pts/0"
		_, err = conn.Write([]byte(msg))
		if err != nil {
			t.Fatalf("Failed to send log: %v", err)
		}
		conn.Close()

		// Wait for processing through the pipeline (BadgerDB -> Bleve)
		time.Sleep(3 * time.Second)

		// Search for the event using "root" as keyword
		events, err := application.SIEMService.SearchHostEvents("root", 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		found := false
		for _, e := range events {
			if e.RawLog != "" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected at least one SIEM event indexed, but search returned zero results")
		}
	})

	// 5. Compliance Service Check
	t.Run("Compliance_Evaluation", func(t *testing.T) {
		packs, err := application.ComplianceService.ListCompliancePacks()
		if err != nil {
			t.Fatalf("Failed to list compliance packs: %v", err)
		}

		if len(packs) == 0 {
			t.Error("Expected compliance packs to be loaded, found 0")
		}

		// Evaluate first available pack (usually NIST or GDPR)
		if len(packs) > 0 {
			result, err := application.ComplianceService.EvaluatePack(packs[0].ID)
			if err != nil {
				t.Fatalf("Pack evaluation failed for %s: %v", packs[0].ID, err)
			}
			if result == nil {
				t.Fatal("Evaluation result is nil")
			}
		}
	})
}
