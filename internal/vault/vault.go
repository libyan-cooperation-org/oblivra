// Package vault implements a passphrase-encrypted secrets store.
//
// The on-disk format is a single JSON file with the layout:
//
//	{
//	  "version": 1,
//	  "kdf":     "argon2id",
//	  "kdfParams": {"time":3,"memory":65536,"threads":4,"keyLen":32},
//	  "salt":    "<hex>",
//	  "nonce":   "<hex>",
//	  "ciphertext": "<hex>"
//	}
//
// The plaintext is `{"items":{"name":"value", ...}}`. Argon2id derives a 256-bit
// key from the passphrase + salt; AES-256-GCM seals the JSON with the key.
//
// Hardware-backed key (TPM/FIDO2/OS keychain) is deferred. This package
// deliberately stays narrow: it's a tested, reviewable cryptographic primitive
// the rest of the platform can build on.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/crypto/argon2"
)

const (
	currentVersion = 1
	saltSize       = 16
	keySize        = 32 // AES-256
)

// KDFParams holds Argon2id tunables. Defaults are conservative for desktop
// hardware; raise time/memory in headless deployments.
type KDFParams struct {
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"`
	Threads uint8  `json:"threads"`
	KeyLen  uint32 `json:"keyLen"`
}

func DefaultKDFParams() KDFParams {
	return KDFParams{Time: 3, Memory: 64 * 1024, Threads: 4, KeyLen: keySize}
}

type fileFormat struct {
	Version    int       `json:"version"`
	KDF        string    `json:"kdf"`
	KDFParams  KDFParams `json:"kdfParams"`
	Salt       string    `json:"salt"`
	Nonce      string    `json:"nonce"`
	Ciphertext string    `json:"ciphertext"`
}

type plaintext struct {
	Items map[string]string `json:"items"`
}

// Vault is the in-memory unlocked view. Lock zeroes the key.
//
// Two mutexes intentionally:
//   - mu protects the in-memory items + key + bookkeeping
//   - saveMu serialises file writes so concurrent Set/Delete callers don't
//     race on the atomic-rename `.tmp` file (which Windows enforces, but
//     it's the right correctness invariant on every OS)
type Vault struct {
	path   string
	mu     sync.RWMutex
	saveMu sync.Mutex
	key    []byte
	params KDFParams
	salt   []byte
	items  map[string]string
	dirty  bool
}

// Create initialises a brand-new vault file at path with the given passphrase.
// Returns an error if the file already exists.
func Create(path, passphrase string) (*Vault, error) {
	if _, err := os.Stat(path); err == nil {
		return nil, errors.New("vault: refusing to overwrite existing file")
	}
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	params := DefaultKDFParams()
	key := deriveKey(passphrase, salt, params)
	v := &Vault{
		path: path, key: key, params: params, salt: salt,
		items: map[string]string{},
	}
	if err := v.save(); err != nil {
		return nil, err
	}
	return v, nil
}

// Open unlocks an existing vault file with passphrase. Returns ErrInvalidKey
// if the passphrase is wrong (GCM auth failure).
func Open(path, passphrase string) (*Vault, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f fileFormat
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("vault: bad file: %w", err)
	}
	if f.Version != currentVersion {
		return nil, fmt.Errorf("vault: unsupported version %d", f.Version)
	}
	if f.KDF != "argon2id" {
		return nil, fmt.Errorf("vault: unsupported KDF %s", f.KDF)
	}
	salt, err := hex.DecodeString(f.Salt)
	if err != nil {
		return nil, fmt.Errorf("vault: salt: %w", err)
	}
	nonce, err := hex.DecodeString(f.Nonce)
	if err != nil {
		return nil, fmt.Errorf("vault: nonce: %w", err)
	}
	ct, err := hex.DecodeString(f.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("vault: ciphertext: %w", err)
	}

	key := deriveKey(passphrase, salt, f.KDFParams)
	pt, err := decrypt(key, nonce, ct)
	if err != nil {
		zero(key)
		return nil, ErrInvalidKey
	}
	var p plaintext
	if err := json.Unmarshal(pt, &p); err != nil {
		zero(key)
		return nil, fmt.Errorf("vault: bad plaintext: %w", err)
	}
	if p.Items == nil {
		p.Items = map[string]string{}
	}
	return &Vault{path: path, key: key, params: f.KDFParams, salt: salt, items: p.Items}, nil
}

// ErrInvalidKey is returned when the passphrase fails to authenticate the GCM tag.
var ErrInvalidKey = errors.New("vault: invalid key")

// Get returns a stored secret.
func (v *Vault) Get(name string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	val, ok := v.items[name]
	return val, ok
}

// Set stores or replaces a secret. Persists immediately.
func (v *Vault) Set(name, value string) error {
	v.mu.Lock()
	v.items[name] = value
	v.dirty = true
	v.mu.Unlock()
	return v.save()
}

// Delete removes a secret. Persists immediately.
func (v *Vault) Delete(name string) error {
	v.mu.Lock()
	delete(v.items, name)
	v.dirty = true
	v.mu.Unlock()
	return v.save()
}

// Names lists every stored secret name (no values).
func (v *Vault) Names() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make([]string, 0, len(v.items))
	for k := range v.items {
		out = append(out, k)
	}
	return out
}

// Lock zeroes the in-memory key. Subsequent calls will panic; callers should
// drop the reference.
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	zero(v.key)
	v.key = nil
	v.items = nil
}

func (v *Vault) save() error {
	// Serialise the entire snapshot+write+rename so two concurrent Set()
	// calls don't both try to claim the .tmp file. The data snapshot itself
	// also holds mu.RLock so save() always reflects a consistent state.
	v.saveMu.Lock()
	defer v.saveMu.Unlock()

	v.mu.RLock()
	pt, err := json.Marshal(plaintext{Items: v.items})
	v.mu.RUnlock()
	if err != nil {
		return err
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	ct, err := encrypt(v.key, nonce, pt)
	if err != nil {
		return err
	}
	out := fileFormat{
		Version:    currentVersion,
		KDF:        "argon2id",
		KDFParams:  v.params,
		Salt:       hex.EncodeToString(v.salt),
		Nonce:      hex.EncodeToString(nonce),
		Ciphertext: hex.EncodeToString(ct),
	}
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: write to <path>.tmp then rename.
	tmp := v.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, v.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ---- crypto primitives ----

func deriveKey(passphrase string, salt []byte, p KDFParams) []byte {
	if p.KeyLen == 0 {
		p.KeyLen = keySize
	}
	return argon2.IDKey([]byte(passphrase), salt, p.Time, p.Memory, p.Threads, p.KeyLen)
}

func encrypt(key, nonce, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nil
}

func decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// helper for stable hex parity in tests
var _ = io.Discard
