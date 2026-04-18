package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestValidateTOTP_Replay(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP" // Valid Base32 secret
	
	// 1. Generate a valid code for 'now'
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// 2. First validation should pass
	if !ValidateTOTP(secret, code) {
		t.Error("First validation failed, should have passed")
	}

	// 3. Second validation with same code should fail (Replay)
	if ValidateTOTP(secret, code) {
		t.Error("Second validation passed, should have failed due to replay protection")
	}

	// 4. Different code should still work (if time permits)
	// We'll just wait a tiny bit or mock time if needed, 
	// but here we just want to ensure DIFFERENT secrets don't collide
	secret2 := "JBSWY3DPEHPK3PYY"
	code2, _ := totp.GenerateCode(secret2, time.Now())
	if !ValidateTOTP(secret2, code2) {
		t.Error("Validation for different secret failed")
	}
}
