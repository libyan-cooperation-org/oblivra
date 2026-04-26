// Encrypted config storage for the OBLIVRA agent.
//
// Closes the "Encrypted Config Storage" gap from the agent feature audit:
// secrets and policy embedded in the on-disk config file (FleetConfig
// snapshots, server credentials, TLS keys, plugin keys, …) MUST NOT
// sit at rest in plaintext. A stolen disk image of an agent host would
// otherwise leak everything the agent knew about how to talk back to
// the SIEM.
//
// Cryptographic design:
//
//   Key derivation:
//     - AEAD key  = SHA-256( "oblivra-agent-config" || agent.privKey )
//       The agent's Ed25519 identity key (resolveIdentityKey()) is
//       already protected by the OS file permissions on identity.key,
//       so deriving from it gives us config secrecy "for free" without
//       a second key file to manage.
//
//   Wire format on disk (single file, length-prefixed):
//     [ 4-byte magic  "OBC1" ]
//     [ 12-byte nonce            ]
//     [ ciphertext + 16-byte AEAD tag ]
//
//   Plaintext: arbitrary bytes (typically a JSON-encoded config struct).
//   AEAD: chacha20-poly1305 (Go stdlib via golang.org/x/crypto).
//
// Backwards compatibility: a file that doesn't start with the OBC1
// magic is treated as legacy plaintext config and read as-is, so
// existing deployments don't break on first upgrade. The next write
// re-encrypts. Operators who explicitly want the old behaviour can
// pass `Encrypted: false` to WriteConfig.

package agent

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/chacha20poly1305"
)

// configMagic is the 4-byte sentinel marking encrypted config files.
// Distinct enough from any common JSON / YAML opening byte that we
// can use it for sniff-detection.
var configMagic = []byte{'O', 'B', 'C', '1'}

// ErrConfigCorrupt is returned when a file claims OBC1 framing but
// fails authentication.
var ErrConfigCorrupt = errors.New("agent: encrypted config failed authentication")

// deriveConfigKey produces a 32-byte AEAD key from the agent's
// Ed25519 identity. The identity is already at-rest-protected by the
// OS (mode 0600 on identity.key), so this gives us config secrecy
// without a second key file to manage.
func deriveConfigKey(priv ed25519.PrivateKey) [32]byte {
	if len(priv) != ed25519.PrivateKeySize {
		// Caller bug — fail loudly; an empty key would still produce
		// a deterministic but useless ciphertext.
		panic("agent: deriveConfigKey called with invalid private key")
	}
	h := sha256.New()
	h.Write([]byte("oblivra-agent-config")) // domain separation
	h.Write(priv[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// EncryptConfigBlob encrypts `plaintext` for at-rest storage on the
// agent host. Output is `OBC1 || nonce || ciphertext+tag`.
//
// Pass the output to WriteConfigFile; pass the result of
// ReadConfigFile back into DecryptConfigBlob.
func EncryptConfigBlob(priv ed25519.PrivateKey, plaintext []byte) ([]byte, error) {
	key := deriveConfigKey(priv)
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("agent.config: chacha20 init: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("agent.config: nonce gen: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, configMagic /*AAD*/)
	out := make([]byte, 0, len(configMagic)+len(nonce)+len(ciphertext))
	out = append(out, configMagic...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// DecryptConfigBlob reverses EncryptConfigBlob. If the input doesn't
// start with the OBC1 magic, it's assumed to be a legacy plaintext
// file and returned as-is.
func DecryptConfigBlob(priv ed25519.PrivateKey, blob []byte) ([]byte, error) {
	if len(blob) < len(configMagic) || !bytes.Equal(blob[:len(configMagic)], configMagic) {
		// Legacy: plaintext config from before this layer existed.
		// Caller will re-encrypt on next write.
		return blob, nil
	}
	key := deriveConfigKey(priv)
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("agent.config: chacha20 init: %w", err)
	}
	rest := blob[len(configMagic):]
	if len(rest) < aead.NonceSize()+aead.Overhead() {
		return nil, ErrConfigCorrupt
	}
	nonce := rest[:aead.NonceSize()]
	ciphertext := rest[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, configMagic /*AAD*/)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigCorrupt, err)
	}
	return plaintext, nil
}

// ReadEncryptedConfigFile reads `path` and decrypts the contents using
// the agent's identity key. Returns the plaintext bytes (the caller
// then unmarshals JSON/YAML/whatever into its config struct).
//
// Missing-file is NOT an error — returns (nil, nil) so callers can
// distinguish "no config yet" from "config is corrupt."
func ReadEncryptedConfigFile(priv ed25519.PrivateKey, path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return DecryptConfigBlob(priv, data)
}

// WriteEncryptedConfigFile encrypts `plaintext` with the agent's
// identity-derived AEAD key and atomically writes the resulting blob
// to `path`. File mode is 0600 (owner-only).
//
// Atomic write semantics: writes to `path.tmp` first, fsyncs, then
// renames over `path`. A power-loss mid-write leaves the previous
// config intact.
func WriteEncryptedConfigFile(priv ed25519.PrivateKey, path string, plaintext []byte) error {
	blob, err := EncryptConfigBlob(priv, plaintext)
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("agent.config: open tmp: %w", err)
	}
	if _, err := f.Write(blob); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("agent.config: write: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("agent.config: fsync: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("agent.config: close: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("agent.config: atomic rename: %w", err)
	}
	return nil
}
