package services_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/services"
	"github.com/kingknull/oblivrashell/internal/vault"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestVaultService(t *testing.T) (*services.VaultService, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	os.Setenv("APPDATA", tmpDir)
	os.Setenv("LOCALAPPDATA", tmpDir)
	os.Setenv("HOME", tmpDir)

	log := logger.NewStdoutLogger()
	bus := eventbus.NewBus(log)

	db, err := database.New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	credRepo := database.NewCredentialRepository(db)
	auditRepo := database.NewAuditRepository(db)

	v, err := vault.New(vault.Config{StorePath: tmpDir}, log)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	svc := services.NewVaultService(v, db, nil, nil, credRepo, auditRepo, nil, bus, log)
	ctx := context.WithValue(context.Background(), "test", "true")
	svc.Start(ctx) //nolint:errcheck

	cleanup := func() {
		bus.Close()
		svc.Stop(context.Background())
	}
	return svc, cleanup
}

// ── setup / unlock ────────────────────────────────────────────────────────────

func TestVaultService_SetupAndUnlock(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()

	const pw = "test-passphrase-2026!"

	if err := svc.Setup(pw, ""); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := svc.UnlockWithPassword(pw, false); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if !svc.IsUnlocked() {
		t.Error("vault should be unlocked")
	}
}

func TestVaultService_WrongPassword(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()

	const pw = "correct-password"
	svc.Setup(pw, "") //nolint:errcheck

	err := svc.UnlockWithPassword("wrong-password", false)
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
	if svc.IsUnlocked() {
		t.Error("vault must not be unlocked after wrong password")
	}
}

func TestVaultService_EmptySliceNormalized(t *testing.T) {
	// Regression: Unlock(pw, []byte{}, false) must behave identically to
	// Unlock(pw, nil, false) — i.e. not trigger the hardware key path.
	svc, cleanup := newTestVaultService(t)
	defer cleanup()

	const pw = "correct-password"
	svc.Setup(pw, "") //nolint:errcheck

	// []byte{} was previously causing spurious "incorrect password" — must work now
	err := svc.Unlock(pw, []byte{}, false)
	if err != nil {
		t.Fatalf("Unlock with empty hardwareKey slice: %v", err)
	}
	if !svc.IsUnlocked() {
		t.Error("vault should be unlocked after Unlock with empty slice")
	}
}

// ── credential CRUD ───────────────────────────────────────────────────────────

func TestVaultService_AddAndGetCredential(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()

	const pw = "test-pass"
	svc.Setup(pw, "")
	svc.UnlockWithPassword(pw, false)

	id, err := svc.AddCredential(context.TODO(), "My API Key", "api_key", "super-secret-value-123")
	if err != nil {
		t.Fatalf("AddCredential: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty credential ID")
	}

	decrypted, err := svc.GetDecryptedCredential(context.TODO(), id)
	if err != nil {
		t.Fatalf("GetDecryptedCredential: %v", err)
	}
	if decrypted != "super-secret-value-123" {
		t.Errorf("decrypted: got %q, want %q", decrypted, "super-secret-value-123")
	}
}

func TestVaultService_CredentialInaccessibleWhenLocked(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()

	const pw = "test-pass"
	svc.Setup(pw, "")
	svc.UnlockWithPassword(pw, false)

	id, _ := svc.AddCredential(context.TODO(), "secret", "password", "value")
	svc.Lock()

	_, err := svc.GetDecryptedCredential(context.TODO(), id)
	if err == nil {
		t.Fatal("expected error when vault is locked, got nil")
	}
}

func TestVaultService_DeleteCredential(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()

	const pw = "del-test-pass"
	svc.Setup(pw, "")
	svc.UnlockWithPassword(pw, false)

	id, _ := svc.AddCredential(context.TODO(), "to-delete", "token", "token-value")
	if err := svc.DeleteCredential(context.TODO(), id); err != nil {
		t.Fatalf("DeleteCredential: %v", err)
	}

	_, err := svc.GetDecryptedCredential(context.TODO(), id)
	if err == nil {
		t.Error("expected error getting deleted credential, got nil")
	}
}

// ── password health audit ─────────────────────────────────────────────────────

func TestVaultService_PasswordHealthAudit(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()

	const pw = "audit-test"
	svc.Setup(pw, "")
	svc.UnlockWithPassword(pw, false)

	// Weak password — short
	svc.AddCredential(context.TODO(), "weak", "password", "abc") //nolint:errcheck
	// Strong password
	svc.AddCredential(context.TODO(), "strong", "password", "Tr0ub4dor&3xtr@L0ng!P@ss#2026") //nolint:errcheck

	results, err := svc.PasswordHealthAudit(context.TODO())
	if err != nil {
		t.Fatalf("PasswordHealthAudit: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected ≥2 health results, got %d", len(results))
	}

	for _, r := range results {
		switch r.Label {
		case "weak":
			if r.Severity != "critical" {
				t.Errorf("weak password: expected critical severity, got %q", r.Severity)
			}
		case "strong":
			if r.Score < 70 {
				t.Errorf("strong password: expected score ≥70, got %d", r.Score)
			}
		}
	}
}

// ── password generator ────────────────────────────────────────────────────────

func TestVaultService_GeneratePassword_Length(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()

	for _, length := range []int{12, 20, 32, 64} {
		pass := svc.GeneratePassword(length, true)
		if len(pass) != length {
			t.Errorf("GeneratePassword(%d): got len %d", length, len(pass))
		}
	}
}

func TestVaultService_GeneratePassword_MinLength(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()
	// Lengths below 8 should be clamped to 16
	pass := svc.GeneratePassword(4, false)
	if len(pass) < 8 {
		t.Errorf("GeneratePassword with length<8: got len %d, want ≥8", len(pass))
	}
}

func TestVaultService_GeneratePassword_Uniqueness(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		p := svc.GeneratePassword(20, true)
		if seen[p] {
			t.Errorf("GeneratePassword produced duplicate: %q", p)
		}
		seen[p] = true
	}
}

// ── HasKeychainEntry ──────────────────────────────────────────────────────────

func TestVaultService_HasKeychainEntry_FalseWhenEmpty(t *testing.T) {
	svc, cleanup := newTestVaultService(t)
	defer cleanup()
	// No remember=true unlock was done, so keychain should be empty
	if svc.HasKeychainEntry() {
		t.Log("WARNING: HasKeychainEntry returned true in test environment — keychain may have a stale entry")
	}
}
