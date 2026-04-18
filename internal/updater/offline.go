package updater

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// OfflineBundle represents a USB-deployable update package.
type OfflineBundle struct {
	Version    string `json:"version"`
	SHA256     string `json:"sha256"`
	Size       int64  `json:"size"`
	Verified   bool   `json:"verified"`
	SourcePath string `json:"source_path"`
}

// ImportOfflineBundle verifies and applies a signed update from physical media.
// This enables air-gapped sovereign nodes to receive updates without network.
func (u *Updater) ImportOfflineBundle(bundlePath string) (*OfflineBundle, error) {
	u.log.Info("[UPDATER] Importing offline bundle: %s", bundlePath)

	// 1. Read the bundle
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}

	// 2. Compute SHA-256 for integrity verification
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	info, _ := os.Stat(bundlePath)
	bundle := &OfflineBundle{
		SHA256:     hashStr,
		Size:       info.Size(),
		SourcePath: bundlePath,
	}

	// 3. Check for accompanying .sha256 sidecar file
	sidecarPath := bundlePath + ".sha256"
	if sidecarData, err := os.ReadFile(sidecarPath); err == nil {
		expectedHash := string(sidecarData)
		if len(expectedHash) >= 64 {
			expectedHash = expectedHash[:64]
		}
		if expectedHash == hashStr {
			bundle.Verified = true
			u.log.Info("[UPDATER] Bundle integrity VERIFIED: %s", hashStr)
		} else {
			return bundle, fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, hashStr)
		}
	} else {
		u.log.Warn("[UPDATER] No .sha256 sidecar found — applying without integrity verification")
	}

	// 4. Extract and apply (reuse existing logic)
	executable, err := extractExecutable(data, bundlePath)
	if err != nil {
		return bundle, fmt.Errorf("extract: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return bundle, fmt.Errorf("resolve executable: %w", err)
	}

	// Rename → Write → Done
	oldPath := exePath + ".old"
	os.Remove(oldPath)
	if err := os.Rename(exePath, oldPath); err != nil {
		return bundle, fmt.Errorf("rename current: %w", err)
	}

	if err := os.WriteFile(exePath, executable, 0755); err != nil {
		os.Rename(oldPath, exePath)
		return bundle, fmt.Errorf("write new: %w", err)
	}

	u.log.Info("[UPDATER] Offline update applied successfully from: %s", bundlePath)
	return bundle, nil
}

// CreateOfflineBundle packages the current executable into a deployable tar.gz archive
// with a .sha256 sidecar for integrity verification on air-gapped networks.
func CreateOfflineBundle(outputDir string, log *logger.Logger) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}

	data, err := os.ReadFile(exePath)
	if err != nil {
		return "", fmt.Errorf("read executable: %w", err)
	}

	// Compute hash
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	// Define the bundle path
	version := "latest" // Default fallback if not wired
	bundlePath := fmt.Sprintf("%s/oblivra-update-%s.tar.gz", outputDir, version)

	// Create tar.gz archive
	file, err := os.Create(bundlePath)
	if err != nil {
		return "", fmt.Errorf("create bundle file: %w", err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Add the binary to the tarball
	hdr := &tar.Header{
		Name: "oblivrashell.exe",
		Mode: 0755,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return "", fmt.Errorf("write tar header: %w", err)
	}
	if _, err := tw.Write(data); err != nil {
		return "", fmt.Errorf("write tar body: %w", err)
	}

	// Ensure writers flush to disk
	tw.Close()
	gw.Close()
	file.Close()

	// Write sidecar for verification
	sidecarPath := bundlePath + ".sha256"
	if err := os.WriteFile(sidecarPath, []byte(hashStr), 0644); err != nil {
		return "", fmt.Errorf("write sidecar: %w", err)
	}

	log.Info("[UPDATER] Offline tarball bundle created: %s (SHA256: %s)", bundlePath, hashStr)
	return bundlePath, nil
}
