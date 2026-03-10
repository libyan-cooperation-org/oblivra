package attestation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

// Build-time variables injected via -ldflags
var (
	Version   = "dev"
	CommitSHA = "unknown"
	BuildTime = "unknown"
	BuildHash = "" // SHA-256 of the compiled binary at build time
)

// Report contains the full attestation state of the running binary.
type Report struct {
	Version      string `json:"version"`
	CommitSHA    string `json:"commit_sha"`
	BuildTime    string `json:"build_time"`
	GoVersion    string `json:"go_version"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	ExpectedHash string `json:"expected_hash"`
	ActualHash   string `json:"actual_hash"`
	Verified     bool   `json:"verified"`
	Tampered     bool   `json:"tampered"`
}

// Attest performs a full integrity check of the running binary.
func Attest() (*Report, error) {
	report := &Report{
		Version:      Version,
		CommitSHA:    CommitSHA,
		BuildTime:    BuildTime,
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		ExpectedHash: BuildHash,
	}

	// Compute the SHA-256 of the currently running executable
	exePath, err := os.Executable()
	if err != nil {
		return report, fmt.Errorf("resolve executable: %w", err)
	}

	f, err := os.Open(exePath)
	if err != nil {
		return report, fmt.Errorf("open executable: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return report, fmt.Errorf("hash executable: %w", err)
	}

	report.ActualHash = hex.EncodeToString(h.Sum(nil))

	// If we have a build-time hash, verify it
	if BuildHash != "" {
		report.Verified = report.ActualHash == BuildHash
		report.Tampered = !report.Verified
	}

	return report, nil
}

// GetBuildInfo extracts VCS info from Go's embedded build metadata.
func GetBuildInfo() map[string]string {
	info := map[string]string{
		"version":    Version,
		"commit":     CommitSHA,
		"build_time": BuildTime,
		"go_version": runtime.Version(),
	}

	bi, ok := debug.ReadBuildInfo()
	if ok {
		info["module"] = bi.Main.Path
		info["module_version"] = bi.Main.Version
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if CommitSHA == "unknown" {
					info["commit"] = s.Value
				}
			case "vcs.time":
				if BuildTime == "unknown" {
					info["build_time"] = s.Value
				}
			case "vcs.modified":
				info["dirty"] = s.Value
			}
		}
	}

	return info
}

// StartupTime records when the process started for uptime tracking.
var StartupTime = time.Now()
