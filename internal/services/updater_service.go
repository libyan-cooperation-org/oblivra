package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/updater"
)

// UpdaterService exposes application update functionality to the frontend
type UpdaterService struct {
	BaseService
	ctx     context.Context
	log     *logger.Logger
	updater *updater.Updater
}

// Name returns the service name
func (s *UpdaterService) Name() string { return "updater-service" }

// Dependencies returns service dependencies
func (s *UpdaterService) Dependencies() []string {
	return []string{}
}

// NewUpdaterService creates a new UpdaterService
func NewUpdaterService(log *logger.Logger, u *updater.Updater) *UpdaterService {
	return &UpdaterService{
		log:     log.WithPrefix("updaterservice"),
		updater: u,
	}
}

// Startup is called at application startup
func (s *UpdaterService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("Updater Service started")
	return nil
}

func (s *UpdaterService) Stop(ctx context.Context) error {
	return nil
}

// CheckForUpdate checks if a newer version is available
func (s *UpdaterService) CheckForUpdate() (map[string]interface{}, error) {
	s.log.Info("Checking for updates...")
	rel, hasUpdate, err := s.updater.CheckUpdate()
	if err != nil {
		s.log.Error("Failed to check for updates: %v", err)
		return nil, err
	}

	var notes string
	var version string
	if rel != nil {
		notes = rel.Body
		version = rel.TagName
	}

	return map[string]interface{}{
		"has_update": hasUpdate,
		"version":    version,
		"notes":      notes,
	}, nil
}

// ApplyUpdate downloads and applies the latest update
func (s *UpdaterService) ApplyUpdate() error {
	s.log.Info("Applying update...")
	rel, hasUpdate, err := s.updater.CheckUpdate()
	if err != nil {
		return err
	}
	if !hasUpdate || rel == nil {
		s.log.Info("No update available to apply")
		return nil
	}

	return s.updater.DownloadAndApply(rel)
}

// ImportOfflineBundle applies a signed update from a local file path (USB/Physical media)
func (s *UpdaterService) ImportOfflineBundle(path string) error {
	s.log.Info("Importing offline update bundle from: %s", path)
	_, err := s.updater.ImportOfflineBundle(path)
	return err
}

// CreateOfflineBundle packages the current binary into a signed update for air-gapped nodes
func (s *UpdaterService) CreateOfflineBundle(outputDir string) (string, error) {
	s.log.Info("Creating offline update bundle in: %s", outputDir)
	return updater.CreateOfflineBundle(outputDir, s.log)
}
