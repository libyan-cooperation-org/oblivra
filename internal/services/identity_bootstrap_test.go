package services

import (
	"testing"
)

// TestValidatePassword_BootstrapPolicy locks down the password policy
// that BootstrapAdmin uses. The frontend SetupWizard.svelte enforces a
// 12-character minimum but the backend is the actual security boundary
// — anyone with raw API access can hit /api/v1/setup/initialize without
// going through the wizard.
//
// We don't test the full BootstrapAdmin end-to-end (that requires a
// migrated SQLite + UserRepository, scoped to a separate integration
// test). What this test guards is the *first gate* — short / weak
// passwords must be rejected before they ever reach the DB.
func TestValidatePassword_BootstrapPolicy(t *testing.T) {
	cases := []struct {
		name      string
		password  string
		wantError bool
	}{
		{"too-short", "Aa1bC", true},
		{"missing-upper", "lowercase123!", true},
		{"missing-lower", "UPPERCASE123!", true},
		{"missing-digit", "OnlyLetters", true},
		{"valid-minimum", "GoodPass1", false},
		{"valid-strong", "Sov3reignAdmin2026!", false},
		{"empty", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.password)
			if tc.wantError && err == nil {
				t.Errorf("expected validatePassword(%q) to fail, got nil", tc.password)
			}
			if !tc.wantError && err != nil {
				t.Errorf("expected validatePassword(%q) to pass, got %v", tc.password, err)
			}
		})
	}
}
