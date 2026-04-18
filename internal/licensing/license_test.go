package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

func makeKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	return pub, priv
}

func makeToken(t *testing.T, priv ed25519.PrivateKey, c Claims) string {
	t.Helper()
	claimsJSON, _ := json.Marshal(c)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"Ed25519"}`))
	payload := []byte(header + "." + claimsB64)
	sig := ed25519.Sign(priv, payload)
	return fmt.Sprintf("%s.%s.%s", header, claimsB64, hex.EncodeToString(sig))
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestCommunityDefaultFeatures(t *testing.T) {
	m := New("")
	communityFeatures := []Feature{FeatureSSH, FeatureTerminal, FeatureVault, FeatureSnippets, FeatureNotes}
	for _, f := range communityFeatures {
		if !m.IsFeatureEnabled(f) {
			t.Errorf("Community should enable %q by default", f)
		}
	}
	if m.IsFeatureEnabled(FeatureSIEM) {
		t.Error("SIEM must not be available on Community tier")
	}
	if m.IsFeatureEnabled(FeatureUEBA) {
		t.Error("UEBA must not be available on Community tier")
	}
}

func TestActivateValidLicense(t *testing.T) {
	pub, priv := makeKeyPair(t)
	m := New(hex.EncodeToString(pub))

	token := makeToken(t, priv, Claims{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour).Unix(),
		Tier:      TierEnterprise,
		Licensee:  "Acme Corp",
		LicenseID: "lic-001",
		IssuedBy:  "OBLIVRA Licensing Authority",
		MaxSeats:  10,
	})

	if err := m.Activate(token); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if m.CurrentTier() != TierEnterprise {
		t.Errorf("expected Enterprise, got %v", m.CurrentTier())
	}
	if !m.IsFeatureEnabled(FeatureSIEM) {
		t.Error("Enterprise should include SIEM (Professional feature)")
	}
	if !m.IsFeatureEnabled(FeatureUEBA) {
		t.Error("Enterprise should include UEBA")
	}
	if m.IsFeatureEnabled(FeatureWarMode) {
		t.Error("War Mode should require Sovereign tier")
	}
}

func TestActivateTamperedSignature(t *testing.T) {
	pub, _ := makeKeyPair(t)
	_, otherPriv := makeKeyPair(t)
	m := New(hex.EncodeToString(pub))

	token := makeToken(t, otherPriv, Claims{
		IssuedAt: time.Now().Unix(),
		Tier:     TierSovereign,
	})
	if err := m.Activate(token); err == nil {
		t.Error("expected signature error, got nil")
	}
}

func TestActivateExpiredLicense(t *testing.T) {
	pub, priv := makeKeyPair(t)
	m := New(hex.EncodeToString(pub))

	token := makeToken(t, priv, Claims{
		IssuedAt:     time.Now().Add(-48 * time.Hour).Unix(),
		ExpiresAt:    time.Now().Add(-47 * time.Hour).Unix(),
		Tier:         TierProfessional,
		OfflineGrace: 3600, // 1h grace — still expired after 47h
	})
	if err := m.Activate(token); err == nil {
		t.Error("expected expiry error, got nil")
	}
}

func TestActivatePerpetualLicense(t *testing.T) {
	pub, priv := makeKeyPair(t)
	m := New(hex.EncodeToString(pub))

	token := makeToken(t, priv, Claims{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: 0, // perpetual
		Tier:      TierSovereign,
	})
	if err := m.Activate(token); err != nil {
		t.Fatalf("Activate perpetual: %v", err)
	}
	if m.IsExpired() {
		t.Error("perpetual license must never be expired")
	}
}

func TestFeatureOverrideAdd(t *testing.T) {
	pub, priv := makeKeyPair(t)
	m := New(hex.EncodeToString(pub))

	// Community tier with SIEM explicitly granted via override
	token := makeToken(t, priv, Claims{
		IssuedAt: time.Now().Unix(),
		Tier:     TierCommunity,
		Features: []string{"siem"},
	})
	if err := m.Activate(token); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if !m.IsFeatureEnabled(FeatureSIEM) {
		t.Error("SIEM should be enabled via explicit override")
	}
}

func TestFeatureOverrideRemove(t *testing.T) {
	pub, priv := makeKeyPair(t)
	m := New(hex.EncodeToString(pub))

	// Sovereign tier with War Mode explicitly revoked
	token := makeToken(t, priv, Claims{
		IssuedAt: time.Now().Unix(),
		Tier:     TierSovereign,
		Features: []string{"-war_mode"},
	})
	if err := m.Activate(token); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if m.IsFeatureEnabled(FeatureWarMode) {
		t.Error("War Mode should be disabled via explicit override")
	}
	// Other Sovereign features should still work
	if !m.IsFeatureEnabled(FeatureRansomware) {
		t.Error("Ransomware defense should still be enabled")
	}
}

func TestRequireFeatureError(t *testing.T) {
	m := New("")
	err := m.RequireFeature(FeatureUEBA)
	if err == nil {
		t.Fatal("expected FeatureGateError for UEBA on Community tier")
	}
	if !IsFeatureGateError(err) {
		t.Errorf("expected *FeatureGateError, got %T", err)
	}
	fge := err.(*FeatureGateError)
	if fge.RequiredTier != TierEnterprise {
		t.Errorf("UEBA should require Enterprise tier, got %v", fge.RequiredTier)
	}
}

func TestDeactivate(t *testing.T) {
	pub, priv := makeKeyPair(t)
	m := New(hex.EncodeToString(pub))

	token := makeToken(t, priv, Claims{
		IssuedAt: time.Now().Unix(),
		Tier:     TierSovereign,
	})
	if err := m.Activate(token); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	m.Deactivate()

	if m.CurrentTier() != TierCommunity {
		t.Error("tier should reset to Community after Deactivate")
	}
	if m.IsFeatureEnabled(FeatureWarMode) {
		t.Error("War Mode should be disabled after Deactivate")
	}
	if m.RawLicense() != "" {
		t.Error("RawLicense should be empty after Deactivate")
	}
}

func TestNoPublicKey_DevMode(t *testing.T) {
	// Without a compiled-in public key, any syntactically valid token is accepted
	m := New("")

	c := Claims{IssuedAt: time.Now().Unix(), Tier: TierSovereign}
	claimsJSON, _ := json.Marshal(c)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	token := fmt.Sprintf("%s.%s.deadbeef", header, claimsB64)

	if err := m.Activate(token); err != nil {
		t.Fatalf("dev mode should accept any syntactically valid token: %v", err)
	}
	if m.CurrentTier() != TierSovereign {
		t.Error("dev mode should grant the claimed tier")
	}
}

func TestMalformedToken(t *testing.T) {
	m := New("")
	cases := []string{
		"",
		"noparts",
		"only.two",
		"a.b.c.d.extra",
		"a.b.ZZZnotvalidhex",
	}
	for _, tc := range cases {
		if err := m.Activate(tc); err == nil {
			t.Errorf("malformed token %q should return error", tc)
		}
	}
}
