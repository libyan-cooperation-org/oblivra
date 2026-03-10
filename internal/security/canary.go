package security

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"strings"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// SftpProvider defines the interface for remote file operations.
type SftpProvider interface {
	WriteFile(ctxID string, path string, contentBase64 string) error
}

// CanaryService manages "honeypot" files on target hosts.
type CanaryService struct {
	log  *logger.Logger
	sftp SftpProvider
}

func NewCanaryService(log *logger.Logger, sftp SftpProvider) *CanaryService {
	return &CanaryService{
		log:  log.WithPrefix("canary"),
		sftp: sftp,
	}
}

func (s *CanaryService) Startup(ctx context.Context) {}
func (s *CanaryService) Shutdown()                   {}
func (s *CanaryService) Name() string                { return "CanaryService" }

// DeployCanary creates a dummy file that should never be modified.
// If FIM detects a write/delete on this file, it's a 100% signal of ransomware/unauthorized access.
func (s *CanaryService) DeployCanary(ctx context.Context, hostID string, path string) error {
	s.log.Info("Deploying canary file to %s at %s", hostID, path)

	// In a real implementation, this would use the SSH service to write the file.
	// For now, we simulate the logic.

	canaryContent := "OBLIVRA_SECURITY_CANARY_DO_NOT_TOUCH_OR_MODIFY_INTERNAL_SECURITY_TOKEN_" + hostID
	contentBase64 := base64.StdEncoding.EncodeToString([]byte(canaryContent))

	if s.sftp == nil {
		s.log.Warn("SFTP provider not available, skipping deployment to %s", hostID)
		return nil
	}

	return s.sftp.WriteFile(hostID, path, contentBase64)
}

// DeployCanaryFolder creates a hidden directory with multiple canary files.
// This is more effective at catching broad file-discovery scans used by ransomware.
func (s *CanaryService) DeployCanaryFolder(ctx context.Context, hostID string, rootPath string) error {
	s.log.Info("Deploying canary folder to %s at %s", hostID, rootPath)

	files := []string{".config", "secrets.txt", "passwords.db", "backups.zip"}
	for _, f := range files {
		path := filepath.Join(rootPath, f)
		if err := s.DeployCanary(ctx, hostID, path); err != nil {
			return err
		}
	}
	return nil
}

// IsCanaryFile returns true if the path matches a known canary location.
func (s *CanaryService) IsCanaryFile(path string) bool {
	base := filepath.Base(path)
	return base == ".oblivra_canary" || base == "README_SECURITY.txt" ||
		strings.Contains(path, ".config") || strings.EqualFold(base, "secrets.txt")
}
