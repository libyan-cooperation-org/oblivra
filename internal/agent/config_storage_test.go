package agent

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return priv
}

// TestEncryptDecrypt_RoundTrip verifies the AEAD wrap/unwrap is reversible.
func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	priv := newTestKey(t)
	plain := []byte(`{"server":"https://siem.example.com","tenant":"prod"}`)

	blob, err := EncryptConfigBlob(priv, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if !bytes.HasPrefix(blob, configMagic) {
		t.Errorf("output missing OBC1 magic: % x", blob[:4])
	}
	got, err := DecryptConfigBlob(priv, blob)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Errorf("round-trip mismatch:\n got:  %q\n want: %q", got, plain)
	}
}

// TestDecrypt_LegacyPlaintext: a file without the OBC1 magic is
// returned as-is so existing deployments don't break on first upgrade.
func TestDecrypt_LegacyPlaintext(t *testing.T) {
	priv := newTestKey(t)
	legacy := []byte(`{"server":"old"}`)
	got, err := DecryptConfigBlob(priv, legacy)
	if err != nil {
		t.Fatalf("expected legacy passthrough, got err: %v", err)
	}
	if !bytes.Equal(got, legacy) {
		t.Errorf("legacy bytes mutated:\n got:  %q\n want: %q", got, legacy)
	}
}

// TestDecrypt_DifferentKey: a config encrypted with one key must NOT
// decrypt with another. Catches the "stolen disk + brute-forced
// key derivation" attack scenario.
func TestDecrypt_DifferentKey(t *testing.T) {
	a := newTestKey(t)
	b := newTestKey(t)
	blob, err := EncryptConfigBlob(a, []byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	_, err = DecryptConfigBlob(b, blob)
	if !errors.Is(err, ErrConfigCorrupt) {
		t.Errorf("expected ErrConfigCorrupt with wrong key, got: %v", err)
	}
}

// TestDecrypt_TamperedCiphertext: flipping any byte in the ciphertext
// must fail authentication.
func TestDecrypt_TamperedCiphertext(t *testing.T) {
	priv := newTestKey(t)
	blob, err := EncryptConfigBlob(priv, []byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	// Flip a byte in the ciphertext region.
	blob[len(blob)-1] ^= 0x01
	_, err = DecryptConfigBlob(priv, blob)
	if !errors.Is(err, ErrConfigCorrupt) {
		t.Errorf("expected ErrConfigCorrupt on tamper, got: %v", err)
	}
}

// TestWriteRead_Atomic: end-to-end file write + read using the
// helpers callers actually use.
func TestWriteRead_Atomic(t *testing.T) {
	priv := newTestKey(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.cfg")
	plain := []byte("hello-world")

	if err := WriteEncryptedConfigFile(priv, path, plain); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Verify on-disk bytes start with OBC1 (i.e. not plaintext).
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read raw: %v", err)
	}
	if !bytes.HasPrefix(onDisk, configMagic) {
		t.Errorf("on-disk file missing OBC1 magic — config not encrypted")
	}
	if bytes.Contains(onDisk, plain) {
		t.Errorf("plaintext leaked to disk: file contains %q", plain)
	}
	// Read back via helper.
	got, err := ReadEncryptedConfigFile(priv, path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Errorf("round-trip via file failed:\n got:  %q\n want: %q", got, plain)
	}
}

// TestRead_MissingFile returns (nil, nil), not an error — so callers
// can distinguish "no config yet" from "corrupt."
func TestRead_MissingFile(t *testing.T) {
	priv := newTestKey(t)
	got, err := ReadEncryptedConfigFile(priv, filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Errorf("missing file should be (nil, nil), got err: %v", err)
	}
	if got != nil {
		t.Errorf("missing file should be (nil, nil), got bytes: %q", got)
	}
}
