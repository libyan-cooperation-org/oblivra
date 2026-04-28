package api

// Sovereignty score (marketing + operational signal).
//
// GET /api/v1/sovereignty/score
//
// Combines five signals into a single 0–100 number (each pillar
// weighted at 20):
//
//   1. On-prem deployment (this binary is running locally)
//   2. TPM-rooted custody (forensic signer is hardware)
//   3. Air-gap readiness (no required outbound to cloud feeds)
//   4. Key material residency (vault keys local)
//   5. Encryption-in-transit (TLS on)
//
// The aggregate is rendered in the chrome as a small badge ("🔒 92")
// and on the executive dash as a full tile with all five sub-scores.
// CISOs reviewing this tile can answer "where does my data go?" in
// one glance.
//
// The endpoint is anon-readable — it doesn't leak anything about the
// fleet, only the deployment posture itself.

import (
	"net/http"
	"os"
	"strings"
)

// SovereigntyComponent is one of the four sub-scores. The UI renders
// each as a row with an OK/WARN bullet so ops can see exactly which
// pillar is weak.
type SovereigntyComponent struct {
	Name        string `json:"name"`
	OK          bool   `json:"ok"`
	Reason      string `json:"reason"`
	Weight      int    `json:"weight"` // 0–25
	Earned      int    `json:"earned"` // 0–25
}

// SovereigntyScore is the response body.
type SovereigntyScore struct {
	Score      int                    `json:"score"`      // 0–100
	Tier       string                 `json:"tier"`       // gold | silver | bronze | unverified
	Components []SovereigntyComponent `json:"components"`
}

func (s *RESTServer) handleSovereigntyScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	comps := []SovereigntyComponent{
		s.componentOnPrem(),
		s.componentTPM(),
		s.componentAirGap(),
		s.componentKeyResidency(),
		s.componentTLS(),
	}

	earned := 0
	for _, c := range comps {
		earned += c.Earned
	}

	tier := "unverified"
	switch {
	case earned >= 90:
		tier = "gold"
	case earned >= 70:
		tier = "silver"
	case earned >= 40:
		tier = "bronze"
	}

	s.jsonResponse(w, http.StatusOK, SovereigntyScore{
		Score:      earned,
		Tier:       tier,
		Components: comps,
	})
}

// On-prem detection: we always award the on-prem points because the
// binary is running on the operator's own host. (Cloud-hosted variants
// would set OBLIVRA_HOSTED=1 to flip this.)
func (s *RESTServer) componentOnPrem() SovereigntyComponent {
	hosted := strings.EqualFold(os.Getenv("OBLIVRA_HOSTED"), "1")
	if hosted {
		return SovereigntyComponent{
			Name:   "On-premises deployment",
			OK:     false,
			Reason: "OBLIVRA_HOSTED=1 — running on shared infrastructure",
			Weight: 20,
			Earned: 0,
		}
	}
	return SovereigntyComponent{
		Name:   "On-premises deployment",
		OK:     true,
		Reason: "Single-binary on operator-controlled host",
		Weight: 20,
		Earned: 20,
	}
}

// TPM-rooted custody: presence of a hardware forensic signer.
// The signer is wired in main.go; we check the provider interface.
func (s *RESTServer) componentTPM() SovereigntyComponent {
	if s.forensicsProvider == nil {
		return SovereigntyComponent{
			Name:   "TPM-rooted chain-of-custody",
			OK:     false,
			Reason: "Forensics provider not initialised",
			Weight: 20,
			Earned: 0,
		}
	}
	// Best-effort: env var indicates whether boot detected a hardware
	// TPM. Set by services/forensics_service.go on startup.
	tpm := strings.EqualFold(os.Getenv("OBLIVRA_TPM_ACTIVE"), "1") ||
		strings.EqualFold(os.Getenv("OBLIVRA_TPM_PRESENT"), "1")
	if !tpm {
		return SovereigntyComponent{
			Name:   "TPM-rooted chain-of-custody",
			OK:     false,
			Reason: "Hardware TPM not detected — running with software signer",
			Weight: 20,
			Earned: 8, // partial credit for software signing
		}
	}
	return SovereigntyComponent{
		Name:   "TPM-rooted chain-of-custody",
		OK:     true,
		Reason: "Hardware TPM 2.0 active — evidence is hardware-signed",
		Weight: 20,
		Earned: 20,
	}
}

// Air-gap readiness: no mandatory outbound. We award full points unless
// the operator has explicitly enabled the cloud sync / threat-intel
// pull (which both require OBLIVRA_AIRGAP=0).
func (s *RESTServer) componentAirGap() SovereigntyComponent {
	airgap := !strings.EqualFold(os.Getenv("OBLIVRA_AIRGAP"), "0")
	if !airgap {
		return SovereigntyComponent{
			Name:   "Air-gap readiness",
			OK:     false,
			Reason: "OBLIVRA_AIRGAP=0 — outbound feeds enabled",
			Weight: 20,
			Earned: 8,
		}
	}
	return SovereigntyComponent{
		Name:   "Air-gap readiness",
		OK:     true,
		Reason: "No required outbound calls; can run fully disconnected",
		Weight: 20,
		Earned: 20,
	}
}

// Key residency: vault keys local. If the deployment uses a remote
// KMS we award 0; if it uses TPM-backed local keys we award 25.
func (s *RESTServer) componentKeyResidency() SovereigntyComponent {
	remoteKMS := os.Getenv("OBLIVRA_KMS_ENDPOINT") != ""
	if remoteKMS {
		return SovereigntyComponent{
			Name:   "Key material residency",
			OK:     false,
			Reason: "OBLIVRA_KMS_ENDPOINT set — keys live in remote KMS",
			Weight: 20,
			Earned: 0,
		}
	}
	return SovereigntyComponent{
		Name:   "Key material residency",
		OK:     true,
		Reason: "Vault keys stored locally; no third-party KMS dependency",
		Weight: 20,
		Earned: 20,
	}
}

// TLS pillar (Slice 3 guardrail #3): TLS-off deducts 20 of 20.
// Reads from the same TLSStateProvider that drives /api/v1/tls/state
// so the badge in the chrome and the score stay coherent.
func (s *RESTServer) componentTLS() SovereigntyComponent {
	if s.tlsState != nil && s.tlsState.IsTLSOff() {
		return SovereigntyComponent{
			Name:   "Encryption-in-transit (TLS)",
			OK:     false,
			Reason: "tls.mode=off — fleet HMAC + payloads in plaintext. Compliance frameworks require TLS.",
			Weight: 20,
			Earned: 0,
		}
	}
	return SovereigntyComponent{
		Name:   "Encryption-in-transit (TLS)",
		OK:     true,
		Reason: "TLS enabled — agent traffic encrypted end-to-end",
		Weight: 20,
		Earned: 20,
	}
}
