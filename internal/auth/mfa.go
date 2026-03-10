package auth

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"

	"github.com/pquerna/otp/totp"
)

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

// ValidateTOTP checks whether a 6-digit code is currently valid
func ValidateTOTP(secret string, code string) bool {
	return totp.Validate(code, secret)
}
