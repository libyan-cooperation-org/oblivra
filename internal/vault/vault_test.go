package vault

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateOpenRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.vault")

	v, err := Create(path, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Set("api.key", "secret-value-1"); err != nil {
		t.Fatal(err)
	}
	if err := v.Set("ssh.passphrase", "another-secret"); err != nil {
		t.Fatal(err)
	}

	v.Lock()

	v2, err := Open(path, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := v2.Get("api.key")
	if !ok || got != "secret-value-1" {
		t.Errorf("api.key = %q ok=%v", got, ok)
	}
	if got, ok := v2.Get("ssh.passphrase"); !ok || got != "another-secret" {
		t.Errorf("ssh.passphrase = %q ok=%v", got, ok)
	}
}

func TestOpenWrongPassphrase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.vault")
	if _, err := Create(path, "right"); err != nil {
		t.Fatal(err)
	}
	_, err := Open(path, "wrong")
	if !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey, got %v", err)
	}
}

func TestRefuseOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.vault")
	if _, err := Create(path, "p"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(path, "p"); err == nil {
		t.Error("expected refuse-overwrite error")
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.vault")
	v, _ := Create(path, "p")
	_ = v.Set("a", "1")
	_ = v.Set("b", "2")
	if err := v.Delete("a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := v.Get("a"); ok {
		t.Error("a should be gone")
	}
	if _, ok := v.Get("b"); !ok {
		t.Error("b should still be present")
	}
}

func TestNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.vault")
	v, _ := Create(path, "p")
	_ = v.Set("alpha", "x")
	_ = v.Set("beta", "y")
	names := v.Names()
	if len(names) != 2 {
		t.Errorf("len = %d", len(names))
	}
}

func TestUnreadableFile(t *testing.T) {
	if _, err := Open("does-not-exist", "p"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestCorruptCiphertext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.vault")
	v, _ := Create(path, "p")
	_ = v.Set("k", "v")

	body, _ := os.ReadFile(path)
	// flip the last byte of ciphertext (closing quote in hex string is the last char)
	if len(body) > 100 {
		body[len(body)-3] ^= 0x01
	}
	_ = os.WriteFile(path, body, 0o600)

	_, err := Open(path, "p")
	if err == nil {
		t.Fatal("expected open to fail on corrupt ciphertext")
	}
}
