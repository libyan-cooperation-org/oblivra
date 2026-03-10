package monitoring

import (
	"context"
	"net"
	"sync"
	"time"
)

// HostStatus represents the health of a remote host
type HostStatus string

const (
	StatusHealthy     HostStatus = "healthy"
	StatusDegraded    HostStatus = "degraded"
	StatusUnreachable HostStatus = "unreachable"
	StatusUnknown     HostStatus = "unknown"
)

// HostHealth encapsulates health metrics for a host
type HostHealth struct {
	HostID       string     `json:"host_id"`
	Address      string     `json:"address"`
	Status       HostStatus `json:"status"`
	Latency      int64      `json:"latency_ms"`           // average latency in milliseconds
	UptimeSec    int64      `json:"uptime_sec,omitempty"` // populated via SSH if possible
	LastCheck    time.Time  `json:"last_check"`
	SuccessRate  float64    `json:"success_rate"` // percentage 0-100
	ChecksTotal  int        `json:"checks_total"`
	ChecksFailed int        `json:"checks_failed"`
	LastError    string     `json:"last_error,omitempty"`
}

// HealthChecker periodically verifies connectivity to saved hosts
type HealthChecker struct {
	mu             sync.RWMutex
	hosts          map[string]*HostHealth // maps host ID to its health state
	checkCtx       context.Context
	checkCancel    context.CancelFunc
	interval       time.Duration
	tcpTimeout     time.Duration
	onStatusChange func(string, HostHealth)
}

func NewHealthChecker(interval time.Duration) *HealthChecker {
	if interval == 0 {
		interval = 60 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		hosts:       make(map[string]*HostHealth),
		checkCtx:    ctx,
		checkCancel: cancel,
		interval:    interval,
		tcpTimeout:  5 * time.Second,
	}
}

// SetCallback sets a callback triggered when a host's health status changes
func (hc *HealthChecker) SetCallback(cb func(string, HostHealth)) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.onStatusChange = cb
}

// RegisterHost adds a host to the health checking rotation
func (hc *HealthChecker) RegisterHost(hostID, address string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if _, exists := hc.hosts[hostID]; !exists {
		hc.hosts[hostID] = &HostHealth{
			HostID:  hostID,
			Address: address,
			Status:  StatusUnknown,
		}
	} else {
		// Update address just in case it changed
		hc.hosts[hostID].Address = address
	}
}

// UnregisterHost removes a host from checks
func (hc *HealthChecker) UnregisterHost(hostID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.hosts, hostID)
}

// Start begins the periodic health checks
func (hc *HealthChecker) Start() {
	go hc.runLoop()
}

// Stop halts health checking
func (hc *HealthChecker) Stop() {
	hc.checkCancel()
}

// GetHealth returns the current health of a specific host
func (hc *HealthChecker) GetHealth(hostID string) (HostHealth, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	health, exists := hc.hosts[hostID]
	if !exists {
		return HostHealth{}, false
	}
	return *health, true
}

// GetAllHealth returns all current health statuses
func (hc *HealthChecker) GetAllHealth() map[string]HostHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]HostHealth)
	for id, health := range hc.hosts {
		result[id] = *health
	}
	return result
}

func (hc *HealthChecker) runLoop() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// Initial check immediately
	hc.checkAll()

	for {
		select {
		case <-hc.checkCtx.Done():
			return
		case <-ticker.C:
			hc.checkAll()
		}
	}
}

func (hc *HealthChecker) checkAll() {
	// Snapshot hosts
	hc.mu.RLock()
	type target struct {
		id   string
		addr string
	}
	var targets []target
	for id, h := range hc.hosts {
		targets = append(targets, target{id: id, addr: h.Address})
	}
	hc.mu.RUnlock()

	// Need a dialer to limit concurrent connections or use a small waitgroup
	var wg sync.WaitGroup
	// Limit concurrency to 10
	sem := make(chan struct{}, 10)

	for _, t := range targets {
		wg.Add(1)
		sem <- struct{}{}

		go func(tgt target) {
			defer wg.Done()
			defer func() { <-sem }()

			hc.checkHost(tgt.id, tgt.addr)
		}(t)
	}

	wg.Wait()
}

func (hc *HealthChecker) checkHost(id, address string) {
	// address might just be IP, needs port for TCP dial. Default to 22.
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "22"
	}
	dialAddr := net.JoinHostPort(host, port)

	start := time.Now()
	conn, err := net.DialTimeout("tcp", dialAddr, hc.tcpTimeout)
	latency := time.Since(start).Milliseconds()

	success := err == nil
	var errMsg string
	if !success {
		errMsg = err.Error()
	} else {
		conn.Close()
	}

	hc.updateHostHealth(id, success, latency, errMsg)
}

func (hc *HealthChecker) updateHostHealth(id string, success bool, latency int64, errMsg string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	health, exists := hc.hosts[id]
	if !exists {
		return
	}

	prevStatus := health.Status

	health.ChecksTotal++
	health.LastCheck = time.Now()

	if success {
		// Moving average latency
		if health.Latency == 0 {
			health.Latency = latency
		} else {
			health.Latency = (health.Latency*4 + latency) / 5
		}
		health.LastError = ""

		if health.Latency > 500 {
			health.Status = StatusDegraded
		} else {
			health.Status = StatusHealthy
		}
	} else {
		health.ChecksFailed++
		health.LastError = errMsg
		health.Status = StatusUnreachable
	}

	// Recalculate success rate
	health.SuccessRate = float64(health.ChecksTotal-health.ChecksFailed) / float64(health.ChecksTotal) * 100.0

	if health.SuccessRate < 80.0 && health.Status != StatusUnreachable {
		health.Status = StatusDegraded
	}

	// Trigger callback on status change
	if prevStatus != health.Status && hc.onStatusChange != nil {
		// Make a copy to pass to callback without holding lock
		cbData := *health
		go hc.onStatusChange(id, cbData)
	}
}
