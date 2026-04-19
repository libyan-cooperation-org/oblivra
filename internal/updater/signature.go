package updater

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// sovereignPublicKey is the ed25519 public key used to verify release signatures.
// In production this is embedded at build time via ldflags:
//
//	-X github.com/kingknull/oblivrashell/internal/updater.sovereignPublicKey=<hex>
var sovereignPublicKey = ""

// SignatureVerifier validates update bundles and release artifacts using ed25519.
type SignatureVerifier struct {
	pubKey ed25519.PublicKey
	log    *logger.Logger
}

// NewSignatureVerifier creates a verifier using the embedded sovereign public key.
func NewSignatureVerifier(log *logger.Logger) (*SignatureVerifier, error) {
	if sovereignPublicKey == "" {
		log.Warn("[UPDATER] No sovereign public key configured — signature verification disabled")
		return &SignatureVerifier{log: log}, nil
	}

	keyBytes, err := hex.DecodeString(sovereignPublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode sovereign public key: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("sovereign public key must be %d bytes, got %d",
			ed25519.PublicKeySize, len(keyBytes))
	}

	log.Info("[UPDATER] Signature verifier initialised with sovereign key: %s...%s",
		sovereignPublicKey[:8], sovereignPublicKey[len(sovereignPublicKey)-8:])

	return &SignatureVerifier{
		pubKey: ed25519.PublicKey(keyBytes),
		log:    log,
	}, nil
}

// VerifyBundle checks the ed25519 signature of an update bundle.
// The signature file must be at <bundlePath>.sig and contain a hex-encoded signature.
func (v *SignatureVerifier) VerifyBundle(bundlePath string, data []byte) error {
	if v.pubKey == nil {
		if os.Getenv("OBLIVRA_ENV") == "production" {
			return fmt.Errorf("CRITICAL: no sovereign public key configured — cannot verify bundle signature in production")
		}
		v.log.Warn("[UPDATER] Skipping signature verification — no public key configured (non-production)")
		return nil
	}

	sigPath := bundlePath + ".sig"
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		return fmt.Errorf("signature file not found at %s: %w — bundle is unsigned and cannot be applied", sigPath, err)
	}

	sigHex := strings.TrimSpace(string(sigData))
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature size: expected %d bytes, got %d",
			ed25519.SignatureSize, len(sigBytes))
	}

	if !ed25519.Verify(v.pubKey, data, sigBytes) {
		return fmt.Errorf("SIGNATURE VERIFICATION FAILED: bundle %s has been tampered or was not signed by the sovereign key", filepath.Base(bundlePath))
	}

	v.log.Info("[UPDATER] Bundle signature verified ✓ (%s)", filepath.Base(bundlePath))
	return nil
}

// DowngradeProtector enforces monotonic version progression to prevent rollback attacks.
type DowngradeProtector struct {
	stateFile string
	log       *logger.Logger
}

// NewDowngradeProtector creates a protector that persists the highest-seen version.
func NewDowngradeProtector(dataDir string, log *logger.Logger) *DowngradeProtector {
	return &DowngradeProtector{
		stateFile: filepath.Join(dataDir, "update_version.lock"),
		log:       log,
	}
}

// RecordVersion writes the new version to persistent state after a successful update.
func (d *DowngradeProtector) RecordVersion(version string) error {
	v := normalizeVersion(version)
	if err := os.WriteFile(d.stateFile, []byte(v), 0600); err != nil {
		return fmt.Errorf("record version: %w", err)
	}
	d.log.Info("[UPDATER] Version lock updated to %s", v)
	return nil
}

// CheckAllowed returns an error if the proposed version is older than the recorded minimum.
func (d *DowngradeProtector) CheckAllowed(proposedVersion string) error {
	data, err := os.ReadFile(d.stateFile)
	if err != nil {
		// No lock file yet — first install, allow any version
		return nil
	}

	recorded := strings.TrimSpace(string(data))
	proposed := normalizeVersion(proposedVersion)

	cmp, err := compareVersions(proposed, recorded)
	if err != nil {
		return fmt.Errorf("version comparison error: %w", err)
	}

	if cmp < 0 {
		return fmt.Errorf("DOWNGRADE BLOCKED: proposed version %s is older than installed minimum %s — rollback attacks are not permitted",
			proposed, recorded)
	}

	if cmp == 0 {
		d.log.Info("[UPDATER] Re-installing same version %s (allowed)", proposed)
	}

	return nil
}

// RequiresBundleVersion parses the version embedded in an offline bundle filename.
// Expected format: oblivrashell_v1.2.3_<os>_<arch>.(zip|tar.gz)
func RequiresBundleVersion(bundlePath string) (string, error) {
	base := filepath.Base(bundlePath)
	// Remove extension(s)
	base = strings.TrimSuffix(base, ".tar.gz")
	base = strings.TrimSuffix(base, ".zip")

	// Format: oblivrashell_v1.2.3_linux_amd64
	parts := strings.Split(base, "_")
	if len(parts) < 2 {
		return "", fmt.Errorf("cannot parse version from bundle filename: %s", base)
	}
	version := parts[1]
	if !strings.HasPrefix(version, "v") {
		return "", fmt.Errorf("unexpected version format in bundle filename: %s", version)
	}
	return version, nil
}

// VerifiedUpdater wraps Updater with signature and downgrade checks.
type VerifiedUpdater struct {
	*Updater
	verifier   *SignatureVerifier
	downgrade  *DowngradeProtector
}

// NewVerifiedUpdater creates a production-hardened updater with all security checks.
func NewVerifiedUpdater(repoURL, currentVersion, dataDir string, log *logger.Logger) (*VerifiedUpdater, error) {
	base := NewUpdater(repoURL, currentVersion, log)
	verifier, err := NewSignatureVerifier(log)
	if err != nil {
		return nil, err
	}
	return &VerifiedUpdater{
		Updater:   base,
		verifier:  verifier,
		downgrade: NewDowngradeProtector(dataDir, log),
	}, nil
}

// ApplyVerifiedOfflineBundle runs the full security gauntlet before applying a bundle.
func (u *VerifiedUpdater) ApplyVerifiedOfflineBundle(bundlePath string) error {
	// 1. Parse version from filename
	version, err := RequiresBundleVersion(bundlePath)
	if err != nil {
		return fmt.Errorf("bundle validation: %w", err)
	}

	// 2. Downgrade protection
	if err := u.downgrade.CheckAllowed(version); err != nil {
		return err
	}

	// 3. Read bundle data
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}

	// 4. Signature verification
	if err := u.verifier.VerifyBundle(bundlePath, data); err != nil {
		return err
	}

	// 5. Apply via base updater (which handles SHA-256 sidecar + extraction)
	if err := u.Updater.ApplyOfflineUpdate(bundlePath); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}

	// 6. Record version after successful apply
	if err := u.downgrade.RecordVersion(version); err != nil {
		u.log.Warn("[UPDATER] Failed to record version lock: %v", err)
	}

	return nil
}

// GetExpectedPlatformAsset returns the canonical asset name for the current platform.
func GetExpectedPlatformAsset(version string) string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	if os == "darwin" {
		os = "macos"
	}
	ext := ".tar.gz"
	if os == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("oblivrashell_%s_%s_%s%s", version, os, arch, ext)
}

// normalizeVersion strips a leading 'v' for comparison.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// compareVersions compares two semver strings (major.minor.patch).
// Returns -1, 0, or 1.
func compareVersions(a, b string) (int, error) {
	aParts, err := parseVersion(a)
	if err != nil {
		return 0, fmt.Errorf("parse version %q: %w", a, err)
	}
	bParts, err := parseVersion(b)
	if err != nil {
		return 0, fmt.Errorf("parse version %q: %w", b, err)
	}

	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1, nil
		}
		if aParts[i] > bParts[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseVersion(v string) ([3]int, error) {
	v = normalizeVersion(v)
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		// Strip any pre-release suffix (e.g. "1-beta")
		p = strings.Split(p, "-")[0]
		n, err := strconv.Atoi(p)
		if err != nil {
			return result, fmt.Errorf("invalid version segment %q: %w", p, err)
		}
		result[i] = n
	}
	return result, nil
}
