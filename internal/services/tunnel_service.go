package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	sshpkg "github.com/kingknull/oblivrashell/internal/ssh"
)

// TunnelInfo is the frontend-friendly tunnel representation
type TunnelInfo struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	Type        string `json:"type"` // "local", "remote", "dynamic"
	LocalHost   string `json:"local_host"`
	LocalPort   int    `json:"local_port"`
	RemoteHost  string `json:"remote_host"`
	RemotePort  int    `json:"remote_port"`
	State       string `json:"state"`
	Connections int64  `json:"connections"`
}

type TunnelService struct {
	BaseService
	bus     *eventbus.Bus
	log     *logger.Logger
	tunnels map[string]*sshpkg.Tunnel
	meta    map[string]TunnelInfo // metadata
	mu      *sync.RWMutex
}

func (s *TunnelService) Name() string { return "tunnel-service" }

// Dependencies returns service dependencies
func (s *TunnelService) Dependencies() []string {
	return []string{"eventbus"}
}

func (s *TunnelService) Start(ctx context.Context) error {
	return nil
}

func (s *TunnelService) Stop(ctx context.Context) error {
	s.CloseAll()
	return nil
}

func NewTunnelService(bus *eventbus.Bus, log *logger.Logger) *TunnelService {
	return &TunnelService{
		bus:     bus,
		log:     log.WithPrefix("tunnels"),
		tunnels: make(map[string]*sshpkg.Tunnel),
		meta:    make(map[string]TunnelInfo),
		mu:      &sync.RWMutex{},
	}
}

// CreateTunnel creates and starts a port forwarding tunnel
func (s *TunnelService) CreateTunnel(
	client *sshpkg.Client,
	sessionID string,
	tunnelType string,
	localHost string,
	localPort int,
	remoteHost string,
	remotePort int,
) (*TunnelInfo, error) {
	cfg := sshpkg.TunnelConfig{
		Type:       sshpkg.TunnelType(tunnelType),
		LocalHost:  localHost,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
	}

	tunnel := sshpkg.NewTunnel(client, cfg)

	if err := tunnel.Start(); err != nil {
		return nil, fmt.Errorf("start tunnel: %w", err)
	}

	info := TunnelInfo{
		ID:         tunnel.ID,
		SessionID:  sessionID,
		Type:       tunnelType,
		LocalHost:  localHost,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		State:      "active",
	}

	s.mu.Lock()
	s.tunnels[tunnel.ID] = tunnel
	s.meta[tunnel.ID] = info
	s.mu.Unlock()

	s.log.Info("Created %s tunnel: %s:%d -> %s:%d (ID: %s)",
		tunnelType, localHost, localPort, remoteHost, remotePort, tunnel.ID)

	s.bus.Publish("tunnel.created", info)

	return &info, nil
}

// StopTunnel stops a tunnel
func (s *TunnelService) StopTunnel(tunnelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tunnel, ok := s.tunnels[tunnelID]
	if !ok {
		return fmt.Errorf("tunnel %s not found", tunnelID)
	}

	if err := tunnel.Stop(); err != nil {
		return err
	}

	if info, ok := s.meta[tunnelID]; ok {
		info.State = "closed"
		s.meta[tunnelID] = info
	}

	delete(s.tunnels, tunnelID)
	s.bus.Publish("tunnel.closed", tunnelID)
	s.log.Info("Stopped tunnel: %s", tunnelID)

	return nil
}

// GetAll returns all tunnels
func (s *TunnelService) GetAll() []TunnelInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infos := make([]TunnelInfo, 0, len(s.meta))
	for id, info := range s.meta {
		if tunnel, ok := s.tunnels[id]; ok {
			info.Connections = tunnel.ConnectionCount()
			info.State = string(tunnel.State)
		}
		infos = append(infos, info)
	}
	return infos
}

// GetBySession returns tunnels for a specific session
func (s *TunnelService) GetBySession(sessionID string) []TunnelInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var infos []TunnelInfo
	for id, info := range s.meta {
		if info.SessionID == sessionID {
			if tunnel, ok := s.tunnels[id]; ok {
				info.Connections = tunnel.ConnectionCount()
				info.State = string(tunnel.State)
			}
			infos = append(infos, info)
		}
	}
	return infos
}

// CloseAllForSession closes all tunnels for a session
func (s *TunnelService) CloseAllForSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, info := range s.meta {
		if info.SessionID == sessionID {
			if tunnel, ok := s.tunnels[id]; ok {
				tunnel.Stop()
				delete(s.tunnels, id)
			}
			info.State = "closed"
			s.meta[id] = info
		}
	}
}

// CloseAll stops all tunnels
func (s *TunnelService) CloseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, tunnel := range s.tunnels {
		tunnel.Stop()
		if info, ok := s.meta[id]; ok {
			info.State = "closed"
			s.meta[id] = info
		}
	}
	s.tunnels = make(map[string]*sshpkg.Tunnel)
}
