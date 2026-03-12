package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// FileService implements the FileManager interface by dispatching to local or remote providers.
type FileService struct {
	BaseService
	local FileSystemProvider
	ssh   FileSystemProvider
	log   *logger.Logger
}

func NewFileService(local FileSystemProvider, ssh FileSystemProvider, log *logger.Logger) *FileService {
	return &FileService{
		local: local,
		ssh:   ssh,
		log:   log.WithPrefix("file_service"),
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

func (s *FileService) getProvider(ctxID string) FileSystemProvider {
	if ctxID == "local" {
		return s.local
	}
	return s.ssh
}

func (s *FileService) ListDirectory(ctxID string, path string) ([]FileInfo, error) {
	return s.getProvider(ctxID).ListDirectory(ctxID, path)
}

func (s *FileService) ReadFile(ctxID string, path string) (string, error) {
	return s.getProvider(ctxID).ReadFile(ctxID, path)
}

func (s *FileService) WriteFile(ctxID string, path string, contentBase64 string) error {
	return s.getProvider(ctxID).WriteFile(ctxID, path, contentBase64)
}

func (s *FileService) Mkdir(ctxID string, path string) error {
	return s.getProvider(ctxID).Mkdir(ctxID, path)
}

func (s *FileService) Rename(ctxID string, oldPath, newPath string) error {
	return s.getProvider(ctxID).Rename(ctxID, oldPath, newPath)
}

func (s *FileService) Remove(ctxID string, path string) error {
	return s.getProvider(ctxID).Remove(ctxID, path)
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
