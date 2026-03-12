package services

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/vault"
)

type HostService struct {
	BaseService
	ctx   context.Context
	db    database.DatabaseStore
	vault vault.Provider
	hosts database.HostStore
	bus   *eventbus.Bus
	log   *logger.Logger
}

func (s *HostService) Name() string { return "host-service" }

// Dependencies returns service dependencies
func (s *HostService) Dependencies() []string {
	return []string{"vault", "eventbus"}
}

func NewHostService(db database.DatabaseStore, v vault.Provider, repo database.HostStore, bus *eventbus.Bus, log *logger.Logger) *HostService {
	return &HostService{
		db:    db,
		vault: v,
		hosts: repo,
		bus:   bus,
		log:   log.WithPrefix("hosts"),
	}
}

func (s *HostService) ListHosts() ([]database.Host, error) {
	s.log.Debug("Fetching all hosts")
	return s.hosts.GetAll(s.ctx)
}

func (s *HostService) Start(ctx context.Context) error {
	s.ctx = ctx
	// Don't poll until the database is actually open (after vault unlock)
	s.bus.Subscribe(eventbus.EventVaultUnlocked, func(e eventbus.Event) {
		go s.startPolling()
	})
	return nil
}

func (s *HostService) Stop(ctx context.Context) error {
	return nil
}

func (s *HostService) startPolling() {
	// Initial poll
	s.pollHosts()

	// Periodic poll
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.pollHosts()
		}
	}
}

func (s *HostService) pollHosts() {
	hosts, err := s.ListHosts()
	if err != nil {
		s.log.Error("Failed to fetch hosts for polling: %v", err)
		return
	}

	if len(hosts) == 0 {
		return
	}

	statusMap := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Set up worker pool
	numWorkers := 20
	if len(hosts) < numWorkers {
		numWorkers = len(hosts)
	}

	hostChan := make(chan database.Host, len(hosts))

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for h := range hostChan {
				address := net.JoinHostPort(h.Hostname, strconv.Itoa(h.Port))
				conn, err := net.DialTimeout("tcp", address, 2*time.Second)
				status := false
				if err == nil {
					conn.Close()
					status = true
				}
				mu.Lock()
				statusMap[h.ID] = status
				mu.Unlock()
			}
		}()
	}

	for _, host := range hosts {
		hostChan <- host
	}
	close(hostChan)

	wg.Wait()

	if s.ctx != nil {
		EmitEvent(s.ctx, "host-health-update", statusMap)
	}
}

func (s *HostService) Create(host database.Host) (*database.Host, error) {
	if host.ID == "" {
		host.ID = uuid.New().String()
	}
	s.log.Info("Creating host: %s (%s)", host.Label, host.Hostname)

	if err := s.hosts.Create(s.ctx, &host); err != nil {
		return nil, err
	}

	s.bus.Publish(eventbus.EventHostCreated, host)
	return &host, nil
}

func (s *HostService) Update(host database.Host) (*database.Host, error) {
	s.log.Info("Updating host: %s (hostname: %s, tenant from ctx: %s)",
		host.ID, host.Hostname, database.TenantFromContext(s.ctx))

	if err := s.hosts.Update(s.ctx, &host); err != nil {
		s.log.Error("Failed to update host %s: %v", host.ID, err)
		return nil, err
	}

	s.bus.Publish(eventbus.EventHostUpdated, host)
	return &host, nil
}

func (s *HostService) Delete(id string) error {
	s.log.Info("Deleting host: %s (tenant from ctx: %s)",
		id, database.TenantFromContext(s.ctx))

	if err := s.hosts.Delete(s.ctx, id); err != nil {
		s.log.Error("Failed to delete host %s: %v", id, err)
		return err
	}

	s.bus.Publish(eventbus.EventHostDeleted, id)
	return nil
}

func (s *HostService) ToggleFavorite(id string) error {
	s.log.Debug("Toggling favorite: %s", id)
	_, err := s.hosts.ToggleFavorite(s.ctx, id)
	return err
}

// WakeHost sends a WOL Magic Packet to the host if a MAC tag exists
func (s *HostService) WakeHost(id string) error {
	host, err := s.hosts.GetByID(s.ctx, id)
	if err != nil {
		return err
	}

	var mac string
	for _, tag := range host.Tags {
		if strings.HasPrefix(tag, "mac:") {
			mac = strings.TrimPrefix(tag, "mac:")
			break
		}
	}

	if mac == "" {
		return fmt.Errorf("no MAC address tag found for host (use format mac:XX:XX:XX:XX:XX:XX)")
	}

	s.log.Info("Sending WOL packet to %s (%s)", host.Label, mac)
	return monitoring.SendMagicPacket(mac)
}

// ImportSSHConfig reads the user's local ~/.ssh/config file and bulk-imports the host declarations into the database
func (s *HostService) ImportSSHConfig() (int, error) {
	s.log.Info("Starting bulk import from local ~/.ssh/config")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0, fmt.Errorf("cannot find user home directory: %v", err)
	}

	configPath := filepath.Join(homeDir, ".ssh", "config")
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("no ~/.ssh/config file found")
		}
		return 0, fmt.Errorf("failed to open config: %v", err)
	}
	defer file.Close()

	var hosts []database.Host
	var currentHost *database.Host

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")

		if key == "host" {
			if value == "*" {
				continue // Skip wildcards
			}
			if currentHost != nil && currentHost.Hostname != "" {
				hosts = append(hosts, *currentHost)
			}
			currentHost = &database.Host{
				ID:         uuid.New().String(),
				Label:      value,
				Hostname:   value, // Default hostname to label until HostName is parsed
				Port:       22,
				AuthMethod: "password",
				Category:   "Imported",
				Tags:       []string{"ssh-config"},
			}
		} else if currentHost != nil {
			switch key {
			case "hostname":
				currentHost.Hostname = value
			case "user":
				currentHost.Username = value
			case "port":
				if p, err := strconv.Atoi(value); err == nil {
					currentHost.Port = p
				}
			case "identityfile":
				// We cannot securely import absolute hard drive paths safely across OSes in Wails
				// The user will attach their Vault Keys manually inside the app
				continue
			}
		}
	}

	if currentHost != nil && currentHost.Hostname != "" {
		hosts = append(hosts, *currentHost)
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading config: %v", err)
	}

	count := 0
	for _, h := range hosts {
		if _, err := s.Create(h); err == nil {
			count++
		}
	}

	s.log.Info("Successfully imported %d hosts from config", count)
	return count, nil
}

