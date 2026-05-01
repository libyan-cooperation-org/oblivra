package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/argon2"
)

// Phase 40 — encrypted local config.
//
// For paranoia-grade offline forwarders the agent.yml may sit on a shared
// box where filesystem ACLs aren't enough. `agent.yml.enc` is an
// AES-256-GCM ciphertext over the plain YAML, with an Argon2id-derived
// key from a passphrase the operator provides via:
//
//   1. OBLIVRA_AGENT_PASSPHRASE env var, or
//   2. file referenced by OBLIVRA_AGENT_PASSPHRASE_FILE.
//
// Both env paths are deliberately less convenient than plain YAML — this
// is meant for environments that need it, not as a default. UF doesn't
// have an equivalent.
//
// File format (binary, no JSON wrapper to keep it small):
//   [4 bytes] magic "OEC1"
//   [16 bytes] salt
//   [12 bytes] nonce
//   [4 bytes] big-endian ciphertext length N
//   [N bytes] AES-256-GCM(plaintext, key=Argon2id(pass, salt))

const encMagic = "OEC1"

func loadEncryptedConfig(path string, passphrase []byte) ([]byte, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(body) < len(encMagic)+16+12+4 {
		return nil, errors.New("config.enc: file too short")
	}
	if !bytes.HasPrefix(body, []byte(encMagic)) {
		return nil, errors.New("config.enc: bad magic")
	}
	off := len(encMagic)
	salt := body[off : off+16]
	off += 16
	nonce := body[off : off+12]
	off += 12
	n := binary.BigEndian.Uint32(body[off : off+4])
	off += 4
	if int(n) != len(body)-off {
		return nil, fmt.Errorf("config.enc: declared length %d != actual %d", n, len(body)-off)
	}
	ciphertext := body[off:]

	key := argon2.IDKey(passphrase, salt, 3, 64*1024, 4, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("config.enc: decrypt failed (wrong passphrase or tampered): %w", err)
	}
	return plain, nil
}

// SaveEncryptedConfig is the helper an operator runs once to seal a
// config — wired into `oblivra-agent encrypt-config <plaintext> <out>`.
func SaveEncryptedConfig(plaintext, passphrase []byte, outPath string) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	key := argon2.IDKey(passphrase, salt, 3, 64*1024, 4, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	var buf bytes.Buffer
	buf.WriteString(encMagic)
	buf.Write(salt)
	buf.Write(nonce)
	_ = binary.Write(&buf, binary.BigEndian, uint32(len(ciphertext)))
	buf.Write(ciphertext)
	return os.WriteFile(outPath, buf.Bytes(), 0o600)
}

// readPassphrase pulls the passphrase from env or a 0600 file. Returns
// nil if neither is set — caller should fall back to the plain YAML.
func readPassphrase() ([]byte, error) {
	if v := os.Getenv("OBLIVRA_AGENT_PASSPHRASE"); v != "" {
		return []byte(v), nil
	}
	if path := os.Getenv("OBLIVRA_AGENT_PASSPHRASE_FILE"); path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("passphrase file: %w", err)
		}
		// Strip trailing newline; the rest is part of the passphrase.
		body = bytes.TrimRight(body, "\r\n")
		return body, nil
	}
	return nil, nil
}
