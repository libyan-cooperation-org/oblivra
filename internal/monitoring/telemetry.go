package monitoring

import (
	"sync"
	"time"
)

// HostTelemetry represents system resources for a remote host
type HostTelemetry struct {
	HostID      string    `json:"host_id"`
	CPUUsage    float64   `json:"cpu_usage"` // Percentage 0-100
	MemUsedMB   float64   `json:"mem_used_mb"`
	MemTotalMB  float64   `json:"mem_total_mb"`
	DiskUsedGB  float64   `json:"disk_used_gb"`
	DiskTotalGB float64   `json:"disk_total_gb"`
	LoadAvg     float64   `json:"load_avg"` // 1-minute load average
	UpdatedAt   time.Time `json:"updated_at"`
}

// TelemetryManager orchestrates background polling of telemetry data
type TelemetryManager struct {
	mu           sync.RWMutex
	data         map[string]*HostTelemetry // maps host ID to latest telemetry
	onUpdate     func(string, HostTelemetry)
	pollInterval time.Duration
}

// NewTelemetryManager creates a new manager with a default 10s interval
func NewTelemetryManager() *TelemetryManager {
	return &TelemetryManager{
		data:         make(map[string]*HostTelemetry),
		pollInterval: 10 * time.Second,
	}
}

// SetUpdateCallback registers a listener for new telemetry data
func (tm *TelemetryManager) SetUpdateCallback(cb func(string, HostTelemetry)) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.onUpdate = cb
}

// UpdateHost records new telemetry for a host and triggers callbacks
func (tm *TelemetryManager) UpdateHost(hostID string, t HostTelemetry) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	copyData := t
	copyData.UpdatedAt = time.Now()
	tm.data[hostID] = &copyData

	if tm.onUpdate != nil {
		go tm.onUpdate(hostID, copyData)
	}
}

// GetTelemetry returns the latest data for a specific host
func (tm *TelemetryManager) GetTelemetry(hostID string) (HostTelemetry, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	t, exists := tm.data[hostID]
	if !exists {
		return HostTelemetry{}, false
	}
	return *t, true
}

// GetAll returns telemetry for all tracked hosts
func (tm *TelemetryManager) GetAll() []HostTelemetry {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var result []HostTelemetry
	for _, t := range tm.data {
		result = append(result, *t)
	}
	return result
}
