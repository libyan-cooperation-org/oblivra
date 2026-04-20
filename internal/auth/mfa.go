package auth

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"sync"
	"time"

	"github.com/pquerna/otp/totp"
)

// usedCodes tracks recently used TOTP codes to prevent replay attacks.
// Keys are "secret:code", values are the expiration time.
var usedCodes sync.Map

// cleanupInterval is how often the usedCodes cache is purged of expired entries.
const cleanupInterval = 5 * time.Minute

// StartCleanup initiates the background goroutine to purge expired TOTP codes.
// Returns a function that stops the cleanup.
func StartCleanup() func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now().Unix()
				usedCodes.Range(func(key, value interface{}) bool {
					if expiry, ok := value.(int64); ok && expiry < now {
						usedCodes.Delete(key)
					}
					return true
				})
			case <-stop:
				return
			}
		}
	}()
	return func() { close(stop) }
}

// TOTPConfig holds options for TOTP generation
type TOTPConfig struct {
	Issuer      string
	AccountName string
}

// TOTPSetupResult contains the secret and QR code for enrollment
type TOTPSetupResult struct {
	Secret    string `json:"secret"`
	QRCodeB64 string `json:"qr_code"` // Base64-encoded PNG
	URL       string `json:"url"`
}

// GenerateTOTP creates a new TOTP secret and QR code for a user
func GenerateTOTP(cfg TOTPConfig) (*TOTPSetupResult, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      cfg.Issuer,
		AccountName: cfg.AccountName,
	})
	if err != nil {
		return nil, fmt.Errorf("generate TOTP key: %w", err)
	}

	// Generate QR code image
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("generate QR image: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode QR PNG: %w", err)
	}

	return &TOTPSetupResult{
		Secret:    key.Secret(),
		QRCodeB64: base64.StdEncoding.EncodeToString(buf.Bytes()),
		URL:       key.URL(),
	}, nil
}

// ValidateTOTP checks whether a 6-digit code is currently valid and not replayed.
func ValidateTOTP(secret string, code string) bool {
	// 1. Basic TOTP validation (includes drift window)
	if !totp.Validate(code, secret) {
		return false
	}

	// 2. Prevent replay by checking our cache
	key := fmt.Sprintf("%s:%s", secret, code)
	if _, loaded := usedCodes.LoadOrStore(key, time.Now().Add(2*time.Minute).Unix()); loaded {
		// Code was already used recently
		return false
	}

	return true
}
