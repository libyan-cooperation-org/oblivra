package monitoring

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ProbeType defines the protocol for a synthetic check
type ProbeType string

const (
	ProbeHTTP  ProbeType = "http"
	ProbeTCP   ProbeType = "tcp"
	ProbeICMP  ProbeType = "icmp"
)

// ProbeStatus defines the outcome of a check
type ProbeStatus string

const (
	StatusUp   ProbeStatus = "up"
	StatusDown ProbeStatus = "down"
)

// SyntheticProbe defines a target to be monitored
type SyntheticProbe struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        ProbeType         `json:"type"`
	Target      string            `json:"target"` // URL or IP:Port
	Interval    time.Duration     `json:"interval"`
	Timeout     time.Duration     `json:"timeout"`
	Expected    string            `json:"expected,omitempty"` // e.g. "200" for HTTP
	Labels      map[string]string `json:"labels,omitempty"`
}

// ProbeResult captures the outcome of an execution
type ProbeResult struct {
	ProbeID   string        `json:"probe_id"`
	Status    ProbeStatus   `json:"status"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// SyntheticManager manages the lifecycle of probes
type SyntheticManager struct {
	mu         sync.RWMutex
	probes     map[string]*SyntheticProbe
	results    map[string]ProbeResult
	log        *logger.Logger
	httpClient *http.Client
	cancelFunc context.CancelFunc
}

func NewSyntheticManager(log *logger.Logger) *SyntheticManager {
	return &SyntheticManager{
		probes:  make(map[string]*SyntheticProbe),
		results: make(map[string]ProbeResult),
		log:     log.WithPrefix("synthetic"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start begins periodic execution of probes
func (m *SyntheticManager) Start(ctx context.Context) {
	runCtx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel

	m.log.Info("Synthetic monitoring started")

	m.mu.RLock()
	currentProbes := make([]*SyntheticProbe, 0, len(m.probes))
	for _, p := range m.probes {
		currentProbes = append(currentProbes, p)
	}
	m.mu.RUnlock()

	for _, p := range currentProbes {
		go m.worker(runCtx, p)
	}
}

func (m *SyntheticManager) Stop() {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
}

func (m *SyntheticManager) AddProbe(p *SyntheticProbe) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.probes[p.ID] = p
}

func (m *SyntheticManager) worker(ctx context.Context, p *SyntheticProbe) {
	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	// Initial run
	m.runProbe(ctx, p)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.runProbe(ctx, p)
		}
	}
}

func (m *SyntheticManager) runProbe(ctx context.Context, p *SyntheticProbe) {
	start := time.Now()
	var err error

	switch p.Type {
	case ProbeHTTP:
		err = m.checkHTTP(ctx, p)
	case ProbeTCP:
		err = m.checkTCP(ctx, p)
	default:
		err = fmt.Errorf("unsupported probe type: %s", p.Type)
	}

	result := ProbeResult{
		ProbeID:   p.ID,
		Status:    StatusUp,
		Latency:   time.Since(start),
		Timestamp: time.Now(),
	}

	if err != nil {
		result.Status = StatusDown
		result.Error = err.Error()
		m.log.Warn("Probe %s (%s) DOWN: %v", p.Name, p.Target, err)
	}

	m.mu.Lock()
	m.results[p.ID] = result
	m.mu.Unlock()
}

func (m *SyntheticManager) checkHTTP(ctx context.Context, p *SyntheticProbe) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.Target, nil)
	if err != nil {
		return err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if p.Expected != "" && fmt.Sprintf("%d", resp.StatusCode) != p.Expected {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (m *SyntheticManager) checkTCP(ctx context.Context, p *SyntheticProbe) error {
	d := net.Dialer{Timeout: p.Timeout}
	conn, err := d.DialContext(ctx, "tcp", p.Target)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func (m *SyntheticManager) GetResults() []ProbeResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]ProbeResult, 0, len(m.results))
	for _, r := range m.results {
		res = append(res, r)
	}
	return res
}
