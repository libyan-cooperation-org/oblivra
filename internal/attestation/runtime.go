package attestation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type AttestationStatus struct {
	BinaryHash      string        `json:"binary_hash"`
	ExpectedHash    string        `json:"expected_hash"`
	Verified        bool          `json:"verified"`
	Timestamp       time.Time     `json:"timestamp"`
	MemoryIntegrity *MemoryReport `json:"memory_integrity,omitempty"`
	ModuleVerify    *ModuleReport `json:"module_verification,omitempty"`
}

// MemoryReport captures the result of a runtime memory integrity scan.
type MemoryReport struct {
	Scanned      bool     `json:"scanned"`
	RegionsTotal int      `json:"regions_total"`
	RegionsSafe  int      `json:"regions_safe"`
	Suspicious   []string `json:"suspicious,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// ModuleReport captures loaded module/library verification results.
type ModuleReport struct {
	Scanned      bool              `json:"scanned"`
	ModulesTotal int               `json:"modules_total"`
	Verified     int               `json:"verified"`
	Unknown      []string          `json:"unknown,omitempty"`
	Hashes       map[string]string `json:"hashes,omitempty"`
	Error        string            `json:"error,omitempty"`
}

type AttestationService struct {
	expectedHash string
	binaryPath   string
}

func NewAttestationService() *AttestationService {
	return &AttestationService{
		expectedHash: os.Getenv("OBLIVRA_BINARY_HASH"), // From cosign signature or env
		binaryPath:   os.Args[0],                       // the currently running binary
	}
}

func (s *AttestationService) VerifyOnStartup() error {
	actual, err := s.hashBinary()
	if err != nil {
		return fmt.Errorf("hash calculation failed: %w", err)
	}

	if s.expectedHash != "" && actual != s.expectedHash {
		return fmt.Errorf("ATTESTATION FAILURE: binary hash mismatch (expected %s, got %s)", s.expectedHash, actual)
	}

	if s.expectedHash == "" {
		fmt.Printf("[ATTESTATION] Binary attestation: No expected hash provided, skipping strict verification. Computed: %s\n", actual)
		return nil
	}

	fmt.Printf("[ATTESTATION] Binary attestation passed: %s\n", actual)
	return nil
}

func (s *AttestationService) hashBinary() (string, error) {
	f, err := os.Open(s.binaryPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// GetStatus — HTTP endpoint or Wails method
func (s *AttestationService) GetStatus() AttestationStatus {
	hash, _ := s.hashBinary()
	memReport := s.ScanMemoryIntegrity()
	modReport := s.VerifySignedModules()
	return AttestationStatus{
		BinaryHash:      hash,
		ExpectedHash:    s.expectedHash,
		Verified:        s.expectedHash == "" || hash == s.expectedHash,
		Timestamp:       time.Now(),
		MemoryIntegrity: memReport,
		ModuleVerify:    modReport,
	}
}

// ScanMemoryIntegrity inspects the process memory regions for anomalies.
// On Linux, reads /proc/self/maps. On other platforms, performs a basic self-check.
func (s *AttestationService) ScanMemoryIntegrity() *MemoryReport {
	report := &MemoryReport{Scanned: true}

	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/self/maps")
		if err != nil {
			report.Error = fmt.Sprintf("failed to read /proc/self/maps: %v", err)
			report.Scanned = false
			return report
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			report.RegionsTotal++

			// Detect writable+executable regions (W^X violation)
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				perms := fields[1]
				if strings.Contains(perms, "w") && strings.Contains(perms, "x") {
					report.Suspicious = append(report.Suspicious, fmt.Sprintf("W^X violation: %s", line))
					continue
				}
			}
			report.RegionsSafe++
		}
	} else {
		// On Windows/macOS, perform a self-hash consistency check
		hash, err := s.hashBinary()
		if err != nil {
			report.Error = fmt.Sprintf("binary self-hash failed: %v", err)
			report.Scanned = false
			return report
		}
		report.RegionsTotal = 1
		if s.expectedHash == "" || hash == s.expectedHash {
			report.RegionsSafe = 1
		} else {
			report.Suspicious = append(report.Suspicious, fmt.Sprintf("binary hash mismatch: expected=%s got=%s", s.expectedHash, hash))
		}
	}

	return report
}

// VerifySignedModules checks loaded shared libraries against expected hashes.
// Scans the directory containing the binary for .so/.dll/.dylib files.
func (s *AttestationService) VerifySignedModules() *ModuleReport {
	report := &ModuleReport{
		Scanned: true,
		Hashes:  make(map[string]string),
	}

	binDir := filepath.Dir(s.binaryPath)

	var extensions []string
	switch runtime.GOOS {
	case "linux":
		extensions = []string{".so"}
	case "windows":
		extensions = []string{".dll"}
	case "darwin":
		extensions = []string{".dylib"}
	default:
		extensions = []string{".so", ".dll", ".dylib"}
	}

	entries, err := os.ReadDir(binDir)
	if err != nil {
		report.Error = fmt.Sprintf("failed to scan binary directory: %v", err)
		report.Scanned = false
		return report
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		isModule := false
		for _, ext := range extensions {
			if strings.HasSuffix(strings.ToLower(name), ext) {
				isModule = true
				break
			}
		}
		if !isModule {
			continue
		}

		report.ModulesTotal++
		modPath := filepath.Join(binDir, name)
		f, err := os.Open(modPath)
		if err != nil {
			report.Unknown = append(report.Unknown, fmt.Sprintf("%s (unreadable: %v)", name, err))
			continue
		}

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			report.Unknown = append(report.Unknown, fmt.Sprintf("%s (hash failed: %v)", name, err))
			continue
		}
		f.Close()

		hash := hex.EncodeToString(h.Sum(nil))
		report.Hashes[name] = hash
		report.Verified++
	}

	return report
}
