package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/plugin"
)

type PluginService struct {
	BaseService
	ctx      context.Context
	registry *plugin.Registry
	bus      *eventbus.Bus
	log      *logger.Logger
}

func (s *PluginService) Name() string { return "PluginService" }

func NewPluginService(bus *eventbus.Bus, log *logger.Logger) *PluginService {
	// Store in user configs dir or similar. For now inside the project or relative
	home, _ := os.UserHomeDir()
	pluginsDir := filepath.Join(home, ".oblivrashell", "plugins")

	pLog := log.WithPrefix("plugins")
	return &PluginService{
		registry: plugin.NewRegistry(pluginsDir, pLog, bus),
		bus:      bus,
		log:      pLog,
	}
}

func (s *PluginService) Startup(ctx context.Context) {
	s.ctx = ctx

	// Initial discovery
	if err := s.registry.Discover(); err != nil {
		s.log.Error("Failed to discover plugins: %v", err)
	}
}

func (s *PluginService) GetPlugins() []*plugin.Plugin {
	return s.registry.GetAll()
}

func (s *PluginService) Refresh() error {
	s.log.Info("Refreshing plugin registry")
	if err := s.registry.Discover(); err != nil {
		return err
	}
	s.bus.Publish("plugins.refreshed", nil)
	return nil
}

func (s *PluginService) Activate(id string) error {
	if err := s.registry.Activate(id); err != nil {
		s.log.Error("Failed to activate plugin %s: %v", id, err)
		return err
	}
	s.bus.Publish("plugin.activated", id)
	return nil
}

func (s *PluginService) Deactivate(id string) error {
	if err := s.registry.Deactivate(id); err != nil {
		s.log.Error("Failed to deactivate plugin %s: %v", id, err)
		return err
	}
	s.bus.Publish("plugin.deactivated", id)
	return nil
}
