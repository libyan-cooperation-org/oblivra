package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// SpillEncryption derives an AES-256-GCM key from the operator-supplied
// secret and encrypts every disk-spill file. The plaintext never sits on
// disk after a TLS-failure spill — secrets that brushed past the agent
// stay encrypted until they reach the platform.
//
// Splunk Universal Forwarder spills plaintext. We don't.
type SpillEncryption struct {
	enabled bool
	key     []byte
}

// NewSpillEncryption derives a 32-byte key from the secret using Argon2id.
// `secret` may be empty — in that case spill is plaintext (compatible with
// older deployments that don't care). The salt is fixed-per-host so spill
// files written today are still decryptable tomorrow without a sidecar.
func NewSpillEncryption(secret, hostname string) (*SpillEncryption, error) {
	if secret == "" {
		return &SpillEncryption{enabled: false}, nil
	}
	salt := []byte("oblivra-agent-spill-v1:" + hostname)
	key := argon2.IDKey([]byte(secret), salt, 1, 64*1024, 4, 32)
	return &SpillEncryption{enabled: true, key: key}, nil
}

// WriteSpill writes one batch to a file. If encryption is enabled the
// payload is AES-256-GCM-sealed; otherwise it's a plain JSONL.
func (s *SpillEncryption) WriteSpill(dir string, items []string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	body := []byte(strings.Join(items, "\n") + "\n")
	prefix := "spill-"
	if s.enabled {
		prefix = "spill.enc-"
		ct, err := s.seal(body)
		if err != nil {
			return "", err
		}
		body = ct
	}
	name := filepath.Join(dir, fmt.Sprintf("%s%d.jsonl", prefix, time.Now().UnixNano()))
	f, err := os.Create(name)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(body); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return name, nil
}

// ReadSpill returns the JSONL lines from a spill file (decrypting if
// applicable). The function picks decryption-mode by filename prefix.
func (s *SpillEncryption) ReadSpill(path string) ([]string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.Contains(filepath.Base(path), "spill.enc-") {
		if !s.enabled {
			return nil, errors.New("encrypted spill found but no spill secret configured")
		}
		body, err = s.open(body)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", filepath.Base(path), err)
		}
	}
	out := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if len(out) == 1 && out[0] == "" {
		return nil, nil
	}
	return out, nil
}

// SpillFingerprint exposes the configured key fingerprint so the operator
// can confirm two agents are using the same spill secret without exposing it.
func (s *SpillEncryption) SpillFingerprint() string {
	if !s.enabled {
		return "(plaintext)"
	}
	return base64.RawURLEncoding.EncodeToString(s.key[:6])
}

func (s *SpillEncryption) seal(plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

func (s *SpillEncryption) open(ct []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ct) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, body := ct[:gcm.NonceSize()], ct[gcm.NonceSize():]
	return gcm.Open(nil, nonce, body, nil)
}
