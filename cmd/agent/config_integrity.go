package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Tamper-evident config — defense against silent edits to agent.yml.
//
// At first start, the agent computes a SHA-256 of the resolved config
// (after secrets-from-file expansion) and writes it alongside the
// position store as `agent.config.fingerprint.json`. On every
// subsequent start, the agent re-computes the fingerprint and compares.
//
// Fingerprint mismatch → the agent refuses to start unless either:
//   1. OBLIVRA_AGENT_ACKNOWLEDGE_CONFIG_CHANGE=1 is set, OR
//   2. `oblivra-agent run --acknowledge-config-change` is passed.
//
// Either path also rewrites the fingerprint file with the new value.
//
// The fingerprint file is mode 0600 and lives in the state dir. It's
// not signed by the platform — the threat model is "someone with
// disk access edits the YAML"; an attacker with disk access can also
// rewrite the fingerprint, so this is a tripwire, not a lock. The
// real lock is OBLIVRA_AGENT_PASSPHRASE on the encrypted .enc config
// (cmd/agent/config_enc.go).

type configFingerprint struct {
	SHA256    string    `json:"sha256"`
	WrittenAt time.Time `json:"writtenAt"`
	Source    string    `json:"source"` // path to the config file
}

// computeFingerprint hashes the resolved Config struct as JSON. We
// hash the struct (post defaults + post tokenFile expansion), not the
// raw file bytes, so a no-op whitespace edit doesn't trip the alarm
// but a real value change does.
//
// Sensitive fields (Server.Token, SpillSecret) are zeroed out before
// hashing so the fingerprint file doesn't leak credential material.
func computeFingerprint(c *Config) string {
	clone := *c
	// Don't include secret values in the hash — comparing the masked
	// shape gives us "did anything structural change" without writing
	// secrets to a file readable by anyone with $stateDir access.
	clone.Server.Token = "<redacted>"
	clone.Server.TokenFile = "<redacted>"
	clone.SpillSecret = "<redacted>"
	clone.SpillSecretFile = "<redacted>"
	body, _ := json.Marshal(&clone)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// fingerprintPath returns the on-disk location for the fingerprint.
func fingerprintPath(stateDir string) string {
	return filepath.Join(stateDir, "agent.config.fingerprint.json")
}

// VerifyConfigIntegrity is called from the run path before any tailers
// start. Returns nil if either the fingerprint matches or the operator
// has acknowledged the change; the side-effect on a fresh run or a
// successful acknowledge is to write the new fingerprint file.
//
// `acknowledged` is true when --acknowledge-config-change was passed
// or OBLIVRA_AGENT_ACKNOWLEDGE_CONFIG_CHANGE=1 in the environment.
func VerifyConfigIntegrity(c *Config, acknowledged bool) error {
	if err := os.MkdirAll(c.StateDir, 0o700); err != nil {
		return fmt.Errorf("config-integrity: mkdir state-dir: %w", err)
	}
	now := computeFingerprint(c)
	path := fingerprintPath(c.StateDir)

	body, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("config-integrity: read fingerprint: %w", err)
		}
		// First run — record the baseline.
		return writeFingerprint(path, configFingerprint{
			SHA256:    now,
			WrittenAt: time.Now().UTC(),
			Source:    "first-run",
		})
	}

	var prev configFingerprint
	if err := json.Unmarshal(body, &prev); err != nil {
		return fmt.Errorf("config-integrity: parse fingerprint: %w", err)
	}
	if prev.SHA256 == now {
		return nil // no change
	}

	if !acknowledged {
		return fmt.Errorf(
			"config-integrity: agent.yml has changed since last run\n"+
				"  previous SHA256: %s (recorded %s)\n"+
				"  current  SHA256: %s\n"+
				"If you intended this change, restart with one of:\n"+
				"  - flag:    oblivra-agent run --acknowledge-config-change\n"+
				"  - env:     OBLIVRA_AGENT_ACKNOWLEDGE_CONFIG_CHANGE=1 oblivra-agent run\n"+
				"This is a tamper tripwire — review the config before acknowledging.",
			prev.SHA256, prev.WrittenAt.Format(time.RFC3339), now,
		)
	}

	return writeFingerprint(path, configFingerprint{
		SHA256:    now,
		WrittenAt: time.Now().UTC(),
		Source:    "acknowledged",
	})
}

func writeFingerprint(path string, fp configFingerprint) error {
	body, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o600)
}
