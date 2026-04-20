package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// FileService implements the FileManager interface by dispatching to local or remote providers.
type FileService struct {
	BaseService
	local    FileSystemProvider
	ssh      FileSystemProvider
	sessions database.SessionStore
	log      *logger.Logger
}

func NewFileService(local FileSystemProvider, ssh FileSystemProvider, sessions database.SessionStore, log *logger.Logger) *FileService {
	return &FileService{
		local:    local,
		ssh:      ssh,
		sessions: sessions,
		log:      log.WithPrefix("file_service"),
	}
}

func (s *FileService) Name() string { return "file-service" }

// Dependencies returns service dependencies
func (s *FileService) Dependencies() []string {
	return []string{}
}

func (s *FileService) Start(ctx context.Context) error {
	return nil
}

func (s *FileService) Stop(ctx context.Context) error {
	return nil
}

func (s *FileService) getProvider(ctxID string) (FileSystemProvider, error) {
	if ctxID == "local" {
		return s.local, nil
	}

	// CS-06: Explicit allowlist — only accept known session IDs from the store.
	// This prevents routing empty strings or typos to the SSH provider.
	if ctxID == "" {
		return nil, fmt.Errorf("invalid file context: empty ID")
	}

	// We use context.Background for the session check because getProvider
	// is a synchronous helper.
	if sess, err := s.sessions.GetByID(context.Background(), ctxID); err == nil && sess != nil {
		return s.ssh, nil
	}

	return nil, fmt.Errorf("unknown or inactive session context: %s", ctxID)
}

func (s *FileService) ListDirectory(ctxID string, path string) ([]FileInfo, error) {
	p, err := s.getProvider(ctxID)
	if err != nil {
		return nil, err
	}
	return p.ListDirectory(ctxID, path)
}

func (s *FileService) ReadFile(ctxID string, path string) (string, error) {
	p, err := s.getProvider(ctxID)
	if err != nil {
		return "", err
	}
	return p.ReadFile(ctxID, path)
}

func (s *FileService) WriteFile(ctxID string, path string, contentBase64 string) error {
	p, err := s.getProvider(ctxID)
	if err != nil {
		return err
	}
	return p.WriteFile(ctxID, path, contentBase64)
}

func (s *FileService) Mkdir(ctxID string, path string) error {
	p, err := s.getProvider(ctxID)
	if err != nil {
		return err
	}
	return p.Mkdir(ctxID, path)
}

func (s *FileService) Rename(ctxID string, oldPath, newPath string) error {
	p, err := s.getProvider(ctxID)
	if err != nil {
		return err
	}
	return p.Rename(ctxID, oldPath, newPath)
}

func (s *FileService) Remove(ctxID string, path string) error {
	p, err := s.getProvider(ctxID)
	if err != nil {
		return err
	}
	return p.Remove(ctxID, path)
}

func (s *FileService) Download(ctxID string, path string, destPath string, size int64) (string, error) {
	if ctxID == "local" {
		return "", fmt.Errorf("file is already local")
	}
	// For SSH, we need to access the transfer manager.
	// Since we are decoupling, we expect the provider to handle this if they implement a specific transfer interface.
	// For now, let's keep it simple and route it if the provider supports it.
	if p, ok := s.ssh.(TransferProvider); ok {
		return p.SftpDownloadAsync(ctxID, path, destPath, size)
	}
	return "", fmt.Errorf("provider does not support downloads")
}

func (s *FileService) Upload(ctxID string, localPath string, destPath string) (string, error) {
	if ctxID == "local" {
		return "", fmt.Errorf("upload to local not implemented (already local)")
	}
	if p, ok := s.ssh.(TransferProvider); ok {
		return p.SftpUploadAsync(ctxID, localPath, destPath)
	}
	return "", fmt.Errorf("provider does not support uploads")
}
