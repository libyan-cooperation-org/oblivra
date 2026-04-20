// Package licensing implements OBLIVRA's feature flag and license enforcement system.
//
// Architecture:
//   - License tokens are signed with Ed25519; public key embedded at build time via ldflags.
//   - Tier hierarchy:  Community < Professional < Enterprise < Sovereign
//   - Offline-first: all verification is local — no network call ever required.
//   - Thread-safe: Manager uses sync.RWMutex throughout.
//
// Typical usage:
//
//	lm := licensing.New(pubKeyHex)         // pubKeyHex injected at build time
//	if err := lm.Activate(licenseKey); err != nil { ... }
//	if err := lm.RequireFeature(licensing.FeatureUEBA); err != nil {
//	    return err   // surfaced to frontend as an upgrade prompt
//	}
package licensing

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Provider defines the interface for license management and feature gating.
// Other packages should depend on this interface rather than the concrete Manager.
type Provider interface {
	IsFeatureEnabled(f Feature) bool
	RequireFeature(f Feature) error
	CurrentTier() Tier
}

// ─── Tiers ───────────────────────────────────────────────────────────────────

// Tier represents a subscription level.
type Tier int

const (
	TierCommunity    Tier = iota // Free — core SSH + local terminal only
	TierProfessional             // Paid individual — SIEM, Vault, Agents
	TierEnterprise               // Team — NDR, UEBA, Compliance, SOAR
	TierSovereign                // Full sovereign — every feature, unlimited agents
)

func (t Tier) String() string {
	switch t {
	case TierCommunity:
		return "Community"
	case TierProfessional:
		return "Professional"
	case TierEnterprise:
		return "Enterprise"
	case TierSovereign:
		return "Sovereign"
	default:
		return "Unknown"
	}
}

// ─── Feature Flags ───────────────────────────────────────────────────────────

// Feature is a named capability gate.
type Feature string

const (
	// TierCommunity — always available
	FeatureSSH      Feature = "ssh"
	FeatureTerminal Feature = "terminal"
	FeatureVault    Feature = "vault_basic"
	FeatureSnippets Feature = "snippets"
	FeatureNotes    Feature = "notes"

	// TierProfessional — paid individual
	FeatureSIEM        Feature = "siem"
	FeatureAlerts      Feature = "alerts"
	FeatureAgents      Feature = "agents"
	FeatureVaultFull   Feature = "vault_full"
	FeatureRecordings  Feature = "recordings"
	FeatureTransfers   Feature = "transfers"
	FeatureTunnels     Feature = "tunnels"
	FeatureMultiExec   Feature = "multi_exec"
	FeatureAIAssistant Feature = "ai_assistant"
	FeaturePlugins     Feature = "plugins"
	FeatureDashboard   Feature = "dashboard"
	FeatureHealth      Feature = "health"
	FeatureMetrics     Feature = "metrics"
	FeatureTopology    Feature = "topology"

	// TierEnterprise — team / org
	FeatureUEBA       Feature = "ueba"
	FeatureNDR        Feature = "ndr"
	FeatureSOAR       Feature = "soar"
	FeaturePurpleTeam Feature = "purple_team"
	FeatureCompliance Feature = "compliance"
	FeatureForensics  Feature = "forensics"
	FeatureIdentity   Feature = "identity"
	FeatureGraph      Feature = "graph"
	FeatureTeam       Feature = "team"
	FeatureSync       Feature = "sync"
	FeatureThreatHunt Feature = "threat_hunt"
	FeatureSOC        Feature = "soc"
	FeatureRisk       Feature = "risk"
	FeatureGovernance Feature = "governance"
	FeatureExecutive  Feature = "executive"
	FeatureCluster    Feature = "cluster"

	// TierSovereign — full platform
	FeatureWarMode       Feature = "war_mode"
	FeatureRansomware    Feature = "ransomware_defense"
	FeatureSimulation    Feature = "simulation"
	FeatureTemporal      Feature = "temporal_integrity"
	FeatureLineage       Feature = "data_lineage"
	FeatureDecisions     Feature = "decision_log"
	FeatureLedger        Feature = "evidence_ledger"
	FeatureReplay        Feature = "response_replay"
	FeatureCounterfact   Feature = "counterfactual"
	FeatureDeterministic Feature = "deterministic_exec"
	FeatureMemorySec     Feature = "memory_security"
	FeatureDisaster      Feature = "disaster_recovery"
	FeatureOfflineUpdate Feature = "offline_updates"
)

// tierFeatures maps each tier to the features it adds (Manager applies cumulatively).
var tierFeatures = map[Tier][]Feature{
	TierCommunity: {
		FeatureSSH, FeatureTerminal, FeatureVault, FeatureSnippets, FeatureNotes,
	},
	TierProfessional: {
		FeatureSIEM, FeatureAlerts, FeatureAgents, FeatureVaultFull, FeatureRecordings,
		FeatureTransfers, FeatureTunnels, FeatureMultiExec, FeatureAIAssistant,
		FeaturePlugins, FeatureDashboard, FeatureHealth, FeatureMetrics, FeatureTopology,
	},
	TierEnterprise: {
		FeatureUEBA, FeatureNDR, FeatureSOAR, FeaturePurpleTeam, FeatureCompliance,
		FeatureForensics, FeatureIdentity, FeatureGraph, FeatureTeam, FeatureSync,
		FeatureThreatHunt, FeatureSOC, FeatureRisk, FeatureGovernance, FeatureExecutive,
		FeatureCluster,
	},
	TierSovereign: {
		FeatureWarMode, FeatureRansomware, FeatureSimulation, FeatureTemporal,
		FeatureLineage, FeatureDecisions, FeatureLedger, FeatureReplay,
		FeatureCounterfact, FeatureDeterministic, FeatureMemorySec, FeatureDisaster,
		FeatureOfflineUpdate,
	},
}

// ─── Claims ───────────────────────────────────────────────────────────────────

// Claims is the JSON payload carried inside a license token.
type Claims struct {
	// Validity window (Unix seconds; ExpiresAt == 0 means perpetual)
	IssuedAt  int64 `json:"iat"`
	ExpiresAt int64 `json:"exp"`
	NotBefore int64 `json:"nbf"`

	// Identity
	LicenseID string `json:"lid"`
	Licensee  string `json:"sub"` // organisation or email
	IssuedBy  string `json:"iss"` // "OBLIVRA Licensing Authority"

	// Entitlements
	Tier      Tier     `json:"tier"`
	MaxSeats  int      `json:"seats"`    // 0 = unlimited
	MaxAgents int      `json:"agents"`   // 0 = unlimited
	Features  []string `json:"features"` // overrides: "feature_name" adds, "-feature_name" removes

	// Clock-skew tolerance in seconds (default 86400 if 0)
	OfflineGrace int64 `json:"grace"`
}

// ─── Manager ──────────────────────────────────────────────────────────────────

// Manager is the central license authority. Create with New(); fully thread-safe.
type Manager struct {
	mu       sync.RWMutex
	pubKey   ed25519.PublicKey
	claims   *Claims
	features map[Feature]bool
	raw      string // original token string, for persistence to settings DB
}

// New creates a Manager with the provided Ed25519 public key (hex-encoded).
// An empty or invalid pubKeyHex places the manager in Community-only mode —
// safe default when no commercial key is compiled in (e.g. dev builds).
func New(pubKeyHex string) *Manager {
	m := &Manager{}
	if pubKeyHex != "" {
		raw, err := hex.DecodeString(strings.TrimSpace(pubKeyHex))
		if err == nil && len(raw) == ed25519.PublicKeySize {
			m.pubKey = ed25519.PublicKey(raw)
		}
	}
	m.buildFeatureMap(TierCommunity, nil)
	return m
}

// ─── Activation ───────────────────────────────────────────────────────────────

// Activate verifies and applies a license token.
// Token format: base64url(header) + "." + base64url(claimsJSON) + "." + hex(ed25519sig)
func (m *Manager) Activate(token string) error {
	_, claimsB64, sig, payload, err := parseToken(token)
	if err != nil {
		return fmt.Errorf("malformed license: %w", err)
	}

	// Signature check — skip only in dev mode (no public key compiled in)
	if m.pubKey != nil {
		if !ed25519.Verify(m.pubKey, payload, sig) {
			return errors.New("license signature invalid — file may have been tampered with")
		}
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(claimsB64)
	if err != nil {
		return fmt.Errorf("claims base64 decode: %w", err)
	}
	var c Claims
	if err := json.Unmarshal(claimsJSON, &c); err != nil {
		return fmt.Errorf("claims JSON decode: %w", err)
	}

	// Temporal validity
	now := time.Now().Unix()
	grace := c.OfflineGrace
	if grace <= 0 {
		grace = 86400 // 24h default offline grace
	}
	if c.NotBefore > 0 && now < c.NotBefore-grace {
		return errors.New("license not yet valid (check system clock)")
	}
	if c.ExpiresAt > 0 && now > c.ExpiresAt+grace {
		return fmt.Errorf("license expired on %s", time.Unix(c.ExpiresAt, 0).Format("2006-01-02"))
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.claims = &c
	m.raw = token
	m.buildFeatureMapLocked(c.Tier, c.Features)
	return nil
}

// Deactivate resets the manager to Community tier.
func (m *Manager) Deactivate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.claims = nil
	m.raw = ""
	m.buildFeatureMapLocked(TierCommunity, nil)
}

// ─── Queries ──────────────────────────────────────────────────────────────────

// IsFeatureEnabled reports whether f is available under the current license.
func (m *Manager) IsFeatureEnabled(f Feature) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.features[f]
}

// RequireFeature returns a *FeatureGateError if f is not enabled.
// The error carries the upgrade path so the frontend can render the correct CTA.
func (m *Manager) RequireFeature(f Feature) error {
	if !m.IsFeatureEnabled(f) {
		return &FeatureGateError{
			Feature:      f,
			CurrentTier:  m.CurrentTier(),
			RequiredTier: featureMinTier(f),
		}
	}
	return nil
}

// CurrentTier returns the effective tier of the active license.
func (m *Manager) CurrentTier() Tier {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.claims == nil {
		return TierCommunity
	}
	return m.claims.Tier
}

// CurrentClaims returns a copy of the active claims, or nil for the Community default.
func (m *Manager) CurrentClaims() *Claims {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.claims == nil {
		return nil
	}
	cp := *m.claims
	return &cp
}

// IsExpired reports whether the license has passed its hard expiry (ignoring grace).
func (m *Manager) IsExpired() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.claims == nil || m.claims.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > m.claims.ExpiresAt
}

// RawLicense returns the raw token string suitable for persistence.
func (m *Manager) RawLicense() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.raw
}

// Summary returns a one-line human-readable status string.
func (m *Manager) Summary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.claims == nil {
		return "Community (no license activated)"
	}
	exp := "perpetual"
	if m.claims.ExpiresAt > 0 {
		exp = time.Unix(m.claims.ExpiresAt, 0).Format("2006-01-02")
	}
	return fmt.Sprintf("%s | %s | seats=%d | expires=%s",
		m.claims.Tier, m.claims.Licensee, m.claims.MaxSeats, exp)
}

// ─── Internal ─────────────────────────────────────────────────────────────────

func (m *Manager) buildFeatureMap(t Tier, overrides []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buildFeatureMapLocked(t, overrides)
}

// buildFeatureMapLocked must be called with m.mu write-locked.
func (m *Manager) buildFeatureMapLocked(t Tier, overrides []string) {
	fm := make(map[Feature]bool)
	for tier := TierCommunity; tier <= t; tier++ {
		for _, f := range tierFeatures[tier] {
			fm[f] = true
		}
	}
	for _, raw := range overrides {
		if strings.HasPrefix(raw, "-") {
			delete(fm, Feature(raw[1:]))
		} else {
			fm[Feature(raw)] = true
		}
	}
	m.features = fm
}

func featureMinTier(f Feature) Tier {
	for _, tier := range []Tier{TierCommunity, TierProfessional, TierEnterprise, TierSovereign} {
		for _, feat := range tierFeatures[tier] {
			if feat == f {
				return tier
			}
		}
	}
	return TierSovereign
}

// parseToken splits "header.claimsB64.sigHex" into its components.
// payload = the bytes that were signed = header + "." + claimsB64.
func parseToken(raw string) (header, claimsB64 string, sig, payload []byte, err error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 3 {
		return "", "", nil, nil, fmt.Errorf("expected 3 dot-separated parts, got %d", len(parts))
	}
	sig, err = hex.DecodeString(parts[2])
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("signature hex decode: %w", err)
	}
	payload = []byte(parts[0] + "." + parts[1])
	return parts[0], parts[1], sig, payload, nil
}
