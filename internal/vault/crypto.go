package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
	"github.com/kingknull/oblivrashell/internal/platform"
)

const (
	// Argon2 parameters
	argonTime    = 3
	argonThreads = 4
	argonKeyLen  = 32

	// OWASP minimum is 64 MB. We adapt based on available system RAM.
	argonMemoryMin  uint32 = 8 * 1024  // 8 MB  — lowest-end fallback
	argonMemoryIdeal uint32 = 64 * 1024 // 64 MB — OWASP recommended
	argonMemoryHigh  uint32 = 128 * 1024 // 128 MB — high-security mode

	// Salt sizes
	saltSize  = 32
	nonceSize = 12 // GCM standard nonce size
)

// argonMemory returns an appropriate Argon2 memory parameter based on
// total available system RAM to balance security vs. availability.
func argonMemory() uint32 {
	totalRAM := platform.Detect().TotalMemoryMB()
	switch {
	case totalRAM >= 8192: // 8 GB+
		return argonMemoryHigh // 128 MB
	case totalRAM >= 1024: // 1 GB+ (OWASP recommends 64MB)
		return argonMemoryIdeal // 64 MB
	case totalRAM >= 512: // 512 MB+
		return argonMemoryMin * 4 // 32 MB (risk of offline brute-force higher)
	default:
		return argonMemoryMin // 8 MB (extreme resource constraint)
	}
}

// DeriveKey derives an encryption key from a password using Argon2id.
// Memory usage scales with available system RAM to balance OWASP compliance
// against OOM risk on resource-constrained hosts.
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory(),
		argonThreads,
		argonKeyLen,
	)
}

// GenerateSalt generates a random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts data using AES-256-GCM
func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// nonce is prepended to ciphertext
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts AES-256-GCM encrypted data
func Decrypt(key []byte, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// DeriveSubKey derives a sub-key from a master key using HKDF
func DeriveSubKey(masterKey []byte, context string, keyLen int) ([]byte, error) {
	hkdfReader := hkdf.New(sha256.New, masterKey, nil, []byte(context))
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("derive sub-key: %w", err)
	}
	return key, nil
}

// EncryptJSON encrypts a struct as JSON
func EncryptJSON(key []byte, v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	encrypted, err := Encrypt(key, data)
	if err != nil {
		return nil, err
	}

	// Zero the plaintext JSON securely avoiding compiler dead-store elimination
	subtle.ConstantTimeCopy(1, data, make([]byte, len(data)))

	return encrypted, nil
}

// DecryptJSON decrypts and unmarshals JSON into a struct
func DecryptJSON(key []byte, ciphertext []byte, v interface{}) error {
	plaintext, err := Decrypt(key, ciphertext)
	if err != nil {
		return err
	}

	defer func() {
		subtle.ConstantTimeCopy(1, plaintext, make([]byte, len(plaintext)))
	}()

	return json.Unmarshal(plaintext, v)
}
