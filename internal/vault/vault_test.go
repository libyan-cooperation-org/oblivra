package vault

import (
	"os"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("hello world, this is a secret message")

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted mismatch: %s != %s", decrypted, plaintext)
	}

	// Wrong key should fail
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = byte(i + 1)
	}

	_, err = Decrypt(wrongKey, encrypted)
	if err == nil {
		t.Error("expected decrypt to fail with wrong key")
	}
}

func TestVaultLifecycle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vault-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		StorePath: tmpDir,
		Platform:  platform.Detect(),
	}
	l, _ := logger.New(logger.Config{
		Level:      logger.ErrorLevel,
		OutputPath: os.DevNull,
	})
	v, err := New(cfg, l)
	if err != nil {
		t.Fatal(err)
	}

	if v.IsSetup() {
		t.Error("expected IsSetup to be false initially")
	}

	passphrase := "correct-horse-battery-staple"
	if err := v.Setup(passphrase, ""); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if !v.IsSetup() {
		t.Error("expected IsSetup to be true after setup")
	}

	// Test unlocking
	if err := v.Unlock("wrong-password", nil, false); err == nil {
		t.Error("expected Unlock to fail with wrong password")
	}

	if v.IsUnlocked() {
		t.Error("vault should be locked after failed unlock")
	}

	if err := v.Unlock(passphrase, nil, false); err != nil {
		t.Fatalf("Unlock failed with correct password: %v", err)
	}

	if !v.IsUnlocked() {
		t.Error("expected IsUnlocked to be true")
	}

	// Test data encryption
	plain := []byte("secret data")
	enc, err := v.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	dec, err := v.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(dec) != string(plain) {
		t.Errorf("Decrypted mismatch: %s != %s", dec, plain)
	}

	// Test locking
	v.Lock()
	if v.IsUnlocked() {
		t.Error("expected IsUnlocked to be false after Lock()")
	}

	_, err = v.Encrypt(plain)
	if err != ErrLocked {
		t.Errorf("expected ErrLocked, got %v", err)
	}
}

func TestPlatformKeychain(t *testing.T) {
	keychain := GetKeychainStore()
	if !keychain.Available() {
		t.Skip("Skipping platform keychain test: not available on this host")
	}

	testKey := "test-secret-key"
	testValue := []byte("top-secret-vault-blob")

	// 1. Set
	if err := keychain.Set(testKey, testValue); err != nil {
		t.Fatalf("Keychain.Set failed: %v", err)
	}
	defer keychain.Delete(testKey)

	// 2. Get
	got, err := keychain.Get(testKey)
	if err != nil {
		t.Fatalf("Keychain.Get failed: %v", err)
	}

	if string(got) != string(testValue) {
		t.Errorf("Keychain.Get mismatch: %s != %s", got, testValue)
	}

	// 3. Delete
	if err := keychain.Delete(testKey); err != nil {
		t.Fatalf("Keychain.Delete failed: %v", err)
	}

	// 4. Get after delete should fail
	_, err = keychain.Get(testKey)
	if err == nil {
		t.Error("expected Keychain.Get to fail after Delete")
	}
}
