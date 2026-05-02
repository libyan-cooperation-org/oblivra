package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// Splunk-UF-style admin password — gates sensitive local subcommands
// and the loopback status endpoint. Stored as an Argon2id hash, never
// in plaintext on disk.
//
// Why have one when the agent isn't a remote-reachable management
// interface? Two reasons:
//
//   1. The local status endpoint (127.0.0.1:18021) returns the
//      signing-key public hex + queue depth + spill bytes. Any user
//      on the host can read it without auth. Adding a password makes
//      the threshold "must know the operator's password", matching
//      what UF protects with `splunk edit user admin -password`.
//
//   2. `oblivra-agent setup` / `reload` / `encrypt-config` mutate
//      the agent's state. A coworker who can ssh to the box shouldn't
//      be able to silently re-run setup and overwrite the config.
//      Password-gating these is the same UF model.
//
// Storage: <stateDir>/agent.password.json
//
//   {
//     "version": 1,
//     "algorithm": "argon2id",
//     "memMiB": 64, "iters": 3, "parallel": 4, "saltLen": 16, "keyLen": 32,
//     "saltB64": "...", "hashB64": "...",
//     "createdAt": "..."
//   }
//
// The file is mode 0600 and lives in the operator-controlled state
// dir alongside the position store. Re-running `oblivra-agent setup`
// rotates the password (operator confirms).

const passwordFile = "agent.password.json"

type passwordRecord struct {
	Version   int       `json:"version"`
	Algorithm string    `json:"algorithm"`
	MemMiB    uint32    `json:"memMiB"`
	Iters     uint32    `json:"iters"`
	Parallel  uint8     `json:"parallel"`
	SaltLen   uint8     `json:"saltLen"`
	KeyLen    uint8     `json:"keyLen"`
	SaltB64   string    `json:"saltB64"`
	HashB64   string    `json:"hashB64"`
	CreatedAt time.Time `json:"createdAt"`
}

func passwordPath(stateDir string) string {
	return filepath.Join(stateDir, passwordFile)
}

// HasAdminPassword returns true when a password file exists. The
// agent's run loop is unconditional — but `setup` / `reload` /
// `encrypt-config` / the local status endpoint check this and demand
// the password when present.
func HasAdminPassword(stateDir string) bool {
	if _, err := os.Stat(passwordPath(stateDir)); err == nil {
		return true
	}
	return false
}

// SetAdminPassword writes a new password hash. Called by the setup
// wizard when the operator chooses to enable the lock, and by a
// dedicated `oblivra-agent password set` subcommand for rotations.
func SetAdminPassword(stateDir, plaintext string) error {
	if len(plaintext) < 8 {
		return errors.New("password too short (minimum 8 characters)")
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	const memMiB, iters, parallelism, keyLen = uint32(64), uint32(3), uint8(4), uint8(32)
	hash := argon2.IDKey([]byte(plaintext), salt,
		iters, memMiB*1024, parallelism, uint32(keyLen))
	rec := passwordRecord{
		Version:   1,
		Algorithm: "argon2id",
		MemMiB:    memMiB,
		Iters:     iters,
		Parallel:  parallelism,
		SaltLen:   uint8(len(salt)),
		KeyLen:    keyLen,
		SaltB64:   base64.StdEncoding.EncodeToString(salt),
		HashB64:   base64.StdEncoding.EncodeToString(hash),
		CreatedAt: time.Now().UTC(),
	}
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(passwordPath(stateDir), append(body, '\n'), 0o600)
}

// VerifyAdminPassword returns nil if `attempt` matches. Constant-time
// comparison so a brute-force can't time-channel the hash. Also
// non-existent file returns "no password set" — caller decides
// whether that's acceptable for the operation.
func VerifyAdminPassword(stateDir, attempt string) error {
	body, err := os.ReadFile(passwordPath(stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return errPasswordNotSet
		}
		return fmt.Errorf("read password record: %w", err)
	}
	var rec passwordRecord
	if err := json.Unmarshal(body, &rec); err != nil {
		return fmt.Errorf("parse password record: %w", err)
	}
	if rec.Algorithm != "argon2id" || rec.Version != 1 {
		return fmt.Errorf("password record uses unsupported version/algorithm")
	}
	salt, err := base64.StdEncoding.DecodeString(rec.SaltB64)
	if err != nil {
		return fmt.Errorf("salt decode: %w", err)
	}
	want, err := base64.StdEncoding.DecodeString(rec.HashB64)
	if err != nil {
		return fmt.Errorf("hash decode: %w", err)
	}
	got := argon2.IDKey([]byte(attempt), salt, rec.Iters, rec.MemMiB*1024, rec.Parallel, uint32(rec.KeyLen))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return errBadPassword
	}
	return nil
}

// ClearAdminPassword removes the file. Used by the password-rotate /
// password-clear flows.
func ClearAdminPassword(stateDir string) error {
	if err := os.Remove(passwordPath(stateDir)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Sentinel errors so callers can treat "no password set" differently
// from "wrong password".
var (
	errPasswordNotSet = errors.New("admin password not set")
	errBadPassword    = errors.New("invalid admin password")
)

// IsPasswordNotSet / IsBadPassword: classifier helpers for callers.
func IsPasswordNotSet(err error) bool { return errors.Is(err, errPasswordNotSet) }
func IsBadPassword(err error) bool    { return errors.Is(err, errBadPassword) }

// readPasswordFromEnv tries OBLIVRA_AGENT_ADMIN_PASSWORD or its file
// variant before falling back to interactive prompt. Used by
// non-interactive flows (systemd reload).
func readPasswordFromEnv() string {
	if v := os.Getenv("OBLIVRA_AGENT_ADMIN_PASSWORD"); v != "" {
		return v
	}
	if path := os.Getenv("OBLIVRA_AGENT_ADMIN_PASSWORD_FILE"); path != "" {
		body, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimRight(string(body), "\r\n")
		}
	}
	return ""
}
