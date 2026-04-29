package architecture_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/app"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/platform"
	"go.uber.org/goleak"
)

// ── shared setup ──────────────────────────────────────────────────────────────

func setupTestApp(t *testing.T) (*app.App, context.CancelFunc) {
	t.Helper()
	tmpDir := t.TempDir()
	os.Setenv("APPDATA", tmpDir)
	os.Setenv("LOCALAPPDATA", tmpDir)
	os.Setenv("HOME", tmpDir)

	// Seed rules directory so alerting service has something to load
	// We must call platform.DataDir() AFTER setting HOME
	dataDir := platform.DataDir()
	sigmaDir := filepath.Join(dataDir, "sigma")
	os.MkdirAll(sigmaDir, 0700)
	
	// Copy a few core rules for the smoke test
	rulesSrc := filepath.Join("..", "internal", "detection", "rules")
	files, _ := os.ReadDir(rulesSrc)
	if len(files) == 0 {
		// Fallback for different test execution depths
		rulesSrc = filepath.Join("internal", "detection", "rules")
		files, _ = os.ReadDir(rulesSrc)
	}
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".yaml" {
			data, _ := os.ReadFile(filepath.Join(rulesSrc, f.Name()))
			os.WriteFile(filepath.Join(sigmaDir, f.Name()), data, 0600)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	ctx = context.WithValue(ctx, "test", "true")

	application := app.New()
	application.Startup(ctx)

	const pw = "sovereign-test-pass-2026"
	if err := application.VaultService.Setup(pw, ""); err != nil {
		t.Logf("Vault Setup: %v", err)
	}
	if err := application.VaultService.Unlock(pw, nil, false); err != nil {
		t.Fatalf("Vault Unlock: %v", err)
	}

	return application, func() {
		application.Shutdown(ctx)
		cancel()
	}
}

// ── smoke tests ───────────────────────────────────────────────────────────────

func TestIntegrationSmoke(t *testing.T) {
	application, cleanup := setupTestApp(t)
	defer func() {
		cleanup()
		time.Sleep(2 * time.Second)
		goleak.VerifyNone(t, 
			goleak.IgnoreTopFunction("github.com/golang/glog.(*loggingT).flushDaemon"),
			goleak.IgnoreTopFunction("github.com/hashicorp/golang-lru/v2/expirable.NewLRU[...].func1"),
			goleak.IgnoreTopFunction("github.com/blevesearch/bleve_index_api.AnalysisWorker"),
			goleak.IgnoreTopFunction("database/sql.(*DB).connectionOpener"),
		)
	}()

	adminCtx := auth.ContextWithUser(context.Background(), &auth.IdentityUser{
		ID:          "superuser",
		TenantID:    database.DefaultTenantID,
		Email:       "admin@oblivra.local",
		RoleName:    "admin",
		Permissions: []string{"*"},
	})
	adminCtx = database.WithGlobalSearch(adminCtx)

	t.Run("Vault_Integrity", func(t *testing.T) {
		id, err := application.VaultService.AddCredential(adminCtx, "SmokeTestSecret", "password", "integration-test-value")
		if err != nil {
			t.Fatalf("AddCredential: %v", err)
		}
		decrypted, err := application.VaultService.GetDecryptedCredential(adminCtx, id)
		if err != nil {
			t.Fatalf("GetDecryptedCredential: %v", err)
		}
		if decrypted != "integration-test-value" {
			t.Errorf("got %q, want integration-test-value", decrypted)
		}
		application.VaultService.Lock()
		_, err = application.VaultService.GetDecryptedCredential(adminCtx, id)
		if err == nil {
			t.Error("expected error getting credential while locked")
		}
		// Re-unlock for subsequent subtests
		application.VaultService.Unlock("sovereign-test-pass-2026", nil, false) //nolint:errcheck
	})

	t.Run("SIEM_Pipeline", func(t *testing.T) {
		if err := application.IngestService.StartSyslogServer(); err != nil {
			t.Fatalf("StartSyslogServer: %v", err)
		}
		defer application.IngestService.StopSyslogServer() 
		
		// Inject a sample syslog message via TCP (more reliable for tests)
		t.Log("Connecting to syslog server via TCP...")
		conn, err := net.DialTimeout("tcp", "127.0.0.1:1514", 2*time.Second)
		if err != nil {
			t.Fatalf("failed to connect to syslog server: %v", err)
		}
		t.Log("Connected. Sending message...")
		msg := `<34>1 2026-01-01T00:00:00Z oblivra oblivra - ID47 - Failed login for root`
		fmt.Fprint(conn, msg+"\n")
		conn.Close()
		t.Log("Message sent. Waiting for processing...")

		time.Sleep(3 * time.Second)
		
		// Search for anything to see if we have events at all
		allEvents, _ := application.SIEMService.SearchHostEvents(adminCtx, "", 10)
		t.Logf("Total events found: %d", len(allEvents))
		for i, e := range allEvents {
			t.Logf("Event[%d]: Host=%s, User=%s, Type=%s, Raw=%s", i, e.HostID, e.User, e.EventType, e.RawLog)
		}

		events, err := application.SIEMService.SearchHostEvents(adminCtx, "root", 10)
		if err != nil {
			t.Fatalf("SearchHostEvents: %v", err)
		}
		found := false
		for _, e := range events {
			if e.RawLog != "" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected at least one indexed SIEM event after syslog injection")
		}
	})

	// Compliance_Evaluation subtest removed Phase 36.x — compliance
	// packs deleted with the broad scope cut.

	t.Run("Alerting_Service_Loads_Rules", func(t *testing.T) {
		rules := application.AlertingService.GetDetectionRules()
		if len(rules) < 50 {
			t.Errorf("expected ≥50 detection rules loaded, got %d", len(rules))
		}
	})

	t.Run("Sigma_HotReload_Trigger_NoOp", func(t *testing.T) {
		// ReloadSigmaRules on a non-existent dir should return 0 without crashing
		count := application.AlertingService.ReloadSigmaRules()
		t.Logf("ReloadSigmaRules returned %d rules", count)
		// No assertion — we only ensure it doesn't panic
	})

	t.Run("Diagnostics_Snapshot", func(t *testing.T) {
		snap := application.DiagnosticsService.GetSnapshot()
		if snap.HealthGrade == "" {
			t.Error("expected non-empty health grade in diagnostics snapshot")
		}
		if snap.Runtime.NumCPU == 0 {
			t.Error("expected NumCPU > 0 in diagnostics snapshot")
		}
	})

	t.Run("Observability_Status", func(t *testing.T) {
		status := application.ObservabilityService.GetObservabilityStatus()
		if status == nil {
			t.Fatal("GetObservabilityStatus returned nil")
		}
		if _, ok := status["goroutines"]; !ok {
			t.Error("missing 'goroutines' key in observability status")
		}
	})

	t.Run("IngestService_Metrics_NonNil", func(t *testing.T) {
		m := application.IngestService.GetMetrics()
		if m == nil {
			t.Fatal("GetMetrics returned nil")
		}
	})

	t.Run("Sigma_SigmaDirectory_Missing_NoError", func(t *testing.T) {
		// Calling LoadSigmaDirectory on a non-existent path should NOT crash
		nonExistent := filepath.Join(os.TempDir(), "does-not-exist-sigma-rules")
		err := application.AlertingService.GetEvaluator().LoadSigmaDirectory(nonExistent)
		// May return error (directory missing) — that's fine; must not panic
		if err != nil {
			t.Logf("LoadSigmaDirectory on missing dir: %v (expected)", err)
		}
	})
}


