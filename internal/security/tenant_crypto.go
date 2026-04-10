package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// TenantCryptoEngine handles per-tenant encryption key derivation
type TenantCryptoEngine struct {
	masterKey []byte
}

func NewTenantCryptoEngine(masterKey []byte) *TenantCryptoEngine {
	return &TenantCryptoEngine{masterKey: masterKey}
}

// DeriveTenantKey generates a 256-bit AES key for a specific tenant using HMAC-SHA256
// combining the master key and the tenant salt.
func (e *TenantCryptoEngine) DeriveTenantKey(tenantID, saltHex string) ([]byte, error) {
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, err
	}
	h := hmac.New(sha256.New, e.masterKey)
	h.Write([]byte(tenantID))
	h.Write(salt)
	return h.Sum(nil), nil
}

// Encrypt payload for a tenant
func (e *TenantCryptoEngine) Encrypt(tenantID, saltHex string, plaintext []byte) ([]byte, error) {
	key, err := e.DeriveTenantKey(tenantID, saltHex)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
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
	
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt payload for a tenant
func (e *TenantCryptoEngine) Decrypt(tenantID, saltHex string, ciphertext []byte) ([]byte, error) {
	key, err := e.DeriveTenantKey(tenantID, saltHex)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertextBytes := ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertextBytes, nil)
}

// GenerateTenantSalt creates a random salt for cryptographic wiping
func GenerateTenantSalt() string {
	salt := make([]byte, 32)
	rand.Read(salt)
	return hex.EncodeToString(salt)
}
