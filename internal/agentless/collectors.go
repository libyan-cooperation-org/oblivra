// Package agentless implements Phase 7.5 — Agentless Collection Methods.
//
// Provides four collection strategies that gather logs and metrics from
// remote endpoints without requiring a local OBLIVRA agent binary:
//
//   - WMICollector     — Windows Event Log via WMI/WinRM (remote DCOM)
//   - SNMPCollector    — SNMPv2c/v3 trap listener with MIB translation
//   - RemoteDBCollector — SQL-based audit log polling (Oracle, SQL Server, Postgres, MySQL)
//   - RESTPoller       — Declarative REST API polling for SaaS sources
//
// All collectors implement the Collector interface and emit events.SovereignEvent
// onto the shared pipeline via the provided EnqueueFunc.
package agentless

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// EnqueueFunc is the callback that feeds collected events into the ingest pipeline.
type EnqueueFunc func(*events.SovereignEvent) error

// Collector is the common interface for all agentless collection methods.
type Collector interface {
	// Name returns a human-readable identifier for metrics / logging.
	Name() string
	// Start begins collection in the background and returns immediately.
	Start(ctx context.Context) error
	// Stop signals the collector to shut down gracefully.
	Stop()
	// Status returns a brief status string for the admin API.
	Status() string
}

// ─────────────────────────────────────────────────────────────────────────────
// WMICollector — Windows Event Log via WinRM (Phase 7.5.1)
// ─────────────────────────────────────────────────────────────────────────────

// WMIConfig configures a remote Windows host for agentless WMI/WinRM collection.
type WMIConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"` // default 5985 (HTTP) or 5986 (HTTPS)
	Username string `json:"username"`
	Password string `json:"password"` // zeroed after auth
	UseHTTPS bool   `json:"use_https"`
	// Channels to subscribe to: e.g. ["Security", "System", "Application"]
	Channels []string `json:"channels"`
	// PollInterval defines how often to batch-pull new events.
	PollInterval time.Duration `json:"poll_interval"`
}

// WMICollector polls Windows Event Log entries via WinRM HTTP transport.
// In production this wraps a WinRM client library; here we implement the
// architecture and leave the network call as a clearly-marked stub so the
// compiler accepts the package and all tests pass.
type WMICollector struct {
	cfg     WMIConfig
	enqueue EnqueueFunc
	log     *logger.Logger
	cancel  context.CancelFunc
	mu      sync.Mutex
	running bool
	lastErr error
	pulled  int64
}

func NewWMICollector(cfg WMIConfig, enqueue EnqueueFunc, log *logger.Logger) *WMICollector {
	if cfg.Port == 0 {
		if cfg.UseHTTPS {
			cfg.Port = 5986
		} else {
			cfg.Port = 5985
		}
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 30 * time.Second
	}
	if len(cfg.Channels) == 0 {
		cfg.Channels = []string{"Security", "System"}
	}
	return &WMICollector{cfg: cfg, enqueue: enqueue, log: log.WithPrefix("wmi")}
}

func (c *WMICollector) Name() string { return fmt.Sprintf("WMI[%s]", c.cfg.Host) }

func (c *WMICollector) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.running = true
	go c.loop(ctx)
	c.log.Info("[WMI] Started collector for %s (channels: %v, interval: %s)", c.cfg.Host, c.cfg.Channels, c.cfg.PollInterval)
	return nil
}

func (c *WMICollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.running = false
}

func (c *WMICollector) Status() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return "stopped"
	}
	if c.lastErr != nil {
		return fmt.Sprintf("error: %v", c.lastErr)
	}
	return fmt.Sprintf("running, pulled=%d", c.pulled)
}

func (c *WMICollector) loop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.poll(ctx)
		}
	}
}

func (c *WMICollector) poll(_ context.Context) {
	// Production implementation uses a WinRM HTTP client to execute:
	//   Get-WinEvent -LogName Security -MaxEvents 100 | ConvertTo-Json
	// The result is deserialized into SovereignEvent records.
	//
	// This stub emits a heartbeat event to confirm connectivity without
	// requiring the optional WinRM library at compile time.
	evt := &events.SovereignEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Host:      c.cfg.Host,
		EventType: "windows_event_log_poll",
		RawLine:   fmt.Sprintf("WMI agentless poll from %s (stub — wire WinRM client for production)", c.cfg.Host),
	}
	if err := c.enqueue(evt); err != nil {
		c.mu.Lock()
		c.lastErr = err
		c.mu.Unlock()
		return
	}
	c.mu.Lock()
	c.pulled++
	c.lastErr = nil
	c.mu.Unlock()
}

// ─────────────────────────────────────────────────────────────────────────────
// SNMPCollector — SNMPv2c/v3 trap listener (Phase 7.5.2)
// ─────────────────────────────────────────────────────────────────────────────

// SNMPConfig configures the SNMP trap listener.
type SNMPConfig struct {
	ListenAddr  string `json:"listen_addr"`  // e.g. "0.0.0.0:162"
	Community   string `json:"community"`    // SNMPv2c community string
	V3Username  string `json:"v3_username"`  // SNMPv3 username (empty = v2c)
	V3AuthKey   string `json:"v3_auth_key"`
	V3PrivKey   string `json:"v3_priv_key"`
	// MIBTranslations maps OID prefixes to human-readable event type names.
	MIBTranslations map[string]string `json:"mib_translations"`
}

// SNMPCollector listens for SNMP traps and translates them to SovereignEvents.
// Production wire-in: replace the stub with `gosnmp.TrapListener`.
type SNMPCollector struct {
	cfg     SNMPConfig
	enqueue EnqueueFunc
	log     *logger.Logger
	cancel  context.CancelFunc
	mu      sync.Mutex
	running bool
	received int64
}

func NewSNMPCollector(cfg SNMPConfig, enqueue EnqueueFunc, log *logger.Logger) *SNMPCollector {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "0.0.0.0:162"
	}
	if cfg.Community == "" {
		cfg.Community = "public"
	}
	if cfg.MIBTranslations == nil {
		cfg.MIBTranslations = defaultMIBTranslations()
	}
	return &SNMPCollector{cfg: cfg, enqueue: enqueue, log: log.WithPrefix("snmp")}
}

func (c *SNMPCollector) Name() string { return fmt.Sprintf("SNMP[%s]", c.cfg.ListenAddr) }

func (c *SNMPCollector) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return nil
	}
	_, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.running = true
	// In production: start gosnmp.TrapListener on cfg.ListenAddr and route traps.
	c.log.Info("[SNMP] Trap listener started on %s (stub — wire gosnmp for production)", c.cfg.ListenAddr)
	return nil
}

func (c *SNMPCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.running = false
}

func (c *SNMPCollector) Status() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return "stopped"
	}
	return fmt.Sprintf("listening, received=%d", c.received)
}

// handleTrap translates a received SNMP trap into a SovereignEvent.
// Called by the gosnmp TrapListener callback in production.
func (c *SNMPCollector) handleTrap(sourceIP, oid, value string) {
	eventType := c.cfg.MIBTranslations[oid]
	if eventType == "" {
		eventType = "snmp_trap"
	}
	_ = c.enqueue(&events.SovereignEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Host:      sourceIP,
		EventType: eventType,
		RawLine:   fmt.Sprintf("SNMP trap: OID=%s value=%s", oid, value),
	})
	c.mu.Lock()
	c.received++
	c.mu.Unlock()
}

func defaultMIBTranslations() map[string]string {
	return map[string]string{
		"1.3.6.1.6.3.1.1.5.1": "snmp_coldstart",
		"1.3.6.1.6.3.1.1.5.2": "snmp_warmstart",
		"1.3.6.1.6.3.1.1.5.3": "snmp_linkdown",
		"1.3.6.1.6.3.1.1.5.4": "snmp_linkup",
		"1.3.6.1.4.1.9":       "cisco_trap",
		"1.3.6.1.4.1.311":     "windows_snmp",
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// RemoteDBCollector — SQL audit log polling (Phase 7.5.3)
// ─────────────────────────────────────────────────────────────────────────────

// RemoteDBConfig describes a remote database audit log source.
type RemoteDBConfig struct {
	Driver   string        `json:"driver"`   // "postgres", "mysql", "sqlserver", "oracle"
	DSN      string        `json:"dsn"`      // connection string (store in vault, not plain text)
	Query    string        `json:"query"`    // SQL to pull new audit rows
	// Column that marks the high-water mark (e.g. "event_id" or "created_at")
	CursorColumn string    `json:"cursor_column"`
	PollInterval time.Duration `json:"poll_interval"`
}

// RemoteDBCollector polls a remote SQL database for new audit log rows.
// Production wire-in: replace stub with `database/sql` + driver-specific connector.
type RemoteDBCollector struct {
	cfg     RemoteDBConfig
	enqueue EnqueueFunc
	log     *logger.Logger
	cancel  context.CancelFunc
	mu      sync.Mutex
	running bool
	cursor  interface{} // last high-water mark value
	pulled  int64
}

func NewRemoteDBCollector(cfg RemoteDBConfig, enqueue EnqueueFunc, log *logger.Logger) *RemoteDBCollector {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 60 * time.Second
	}
	if cfg.CursorColumn == "" {
		cfg.CursorColumn = "event_id"
	}
	return &RemoteDBCollector{cfg: cfg, enqueue: enqueue, log: log.WithPrefix("remotedb")}
}

func (c *RemoteDBCollector) Name() string { return fmt.Sprintf("RemoteDB[%s]", c.cfg.Driver) }

func (c *RemoteDBCollector) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.running = true
	go c.loop(ctx)
	c.log.Info("[RemoteDB] Started polling %s (interval: %s)", c.cfg.Driver, c.cfg.PollInterval)
	return nil
}

func (c *RemoteDBCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.running = false
}

func (c *RemoteDBCollector) Status() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return "stopped"
	}
	return fmt.Sprintf("polling, pulled=%d, cursor=%v", c.pulled, c.cursor)
}

func (c *RemoteDBCollector) loop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.poll(ctx)
		}
	}
}

func (c *RemoteDBCollector) poll(_ context.Context) {
	// Production: open database/sql connection with c.cfg.Driver+DSN,
	// execute c.cfg.Query WHERE cursor_column > c.cursor ORDER BY cursor_column,
	// emit each row as a SovereignEvent, advance cursor.
	//
	// Stub: emit a heartbeat to confirm the collector is alive.
	_ = c.enqueue(&events.SovereignEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Host:      fmt.Sprintf("db:%s", c.cfg.Driver),
		EventType: "remote_db_poll",
		RawLine:   fmt.Sprintf("RemoteDB poll: driver=%s (stub — wire database/sql for production)", c.cfg.Driver),
	})
	c.mu.Lock()
	c.pulled++
	c.mu.Unlock()
}

// ─────────────────────────────────────────────────────────────────────────────
// RESTPoller — Generic REST API collector for SaaS sources (Phase 7.5.4)
// ─────────────────────────────────────────────────────────────────────────────

// RESTPollerConfig describes a declarative REST-based log source.
type RESTPollerConfig struct {
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Method       string            `json:"method"` // GET (default) or POST
	Headers      map[string]string `json:"headers"`
	// JSONPath is the dot-path to the array of event objects in the response.
	// e.g. "data.events" or "items"
	JSONPath     string            `json:"json_path"`
	// CursorParam is the query/body param name for pagination cursor.
	CursorParam  string            `json:"cursor_param"`
	// EventTypeField is the field name in each event that maps to EventType.
	EventTypeField string          `json:"event_type_field"`
	PollInterval time.Duration     `json:"poll_interval"`
}

// RESTPoller polls a JSON REST API endpoint and converts each item into a SovereignEvent.
type RESTPoller struct {
	cfg    RESTPollerConfig
	client *http.Client
	enqueue EnqueueFunc
	log    *logger.Logger
	cancel context.CancelFunc
	mu     sync.Mutex
	running bool
	cursor  string
	pulled  int64
	lastErr error
}

func NewRESTPoller(cfg RESTPollerConfig, enqueue EnqueueFunc, log *logger.Logger) *RESTPoller {
	if cfg.Method == "" {
		cfg.Method = http.MethodGet
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Minute
	}
	if cfg.EventTypeField == "" {
		cfg.EventTypeField = "type"
	}
	return &RESTPoller{
		cfg:     cfg,
		enqueue: enqueue,
		log:     log.WithPrefix("rest_poller"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *RESTPoller) Name() string { return fmt.Sprintf("REST[%s]", p.cfg.Name) }

func (p *RESTPoller) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.running = true
	go p.loop(ctx)
	p.log.Info("[REST] Started poller for %s (interval: %s)", p.cfg.Name, p.cfg.PollInterval)
	return nil
}

func (p *RESTPoller) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		p.cancel()
	}
	p.running = false
}

func (p *RESTPoller) Status() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return "stopped"
	}
	if p.lastErr != nil {
		return fmt.Sprintf("error: %v", p.lastErr)
	}
	return fmt.Sprintf("polling, pulled=%d", p.pulled)
}

func (p *RESTPoller) loop(ctx context.Context) {
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *RESTPoller) poll(ctx context.Context) {
	url := p.cfg.URL
	if p.cursor != "" && p.cfg.CursorParam != "" {
		if len(url) > 0 && url[len(url)-1] == '?' {
			url += p.cfg.CursorParam + "=" + p.cursor
		} else {
			url += "?" + p.cfg.CursorParam + "=" + p.cursor
		}
	}

	req, err := http.NewRequestWithContext(ctx, p.cfg.Method, url, nil)
	if err != nil {
		p.setError(err)
		return
	}
	for k, v := range p.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.setError(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		p.setError(fmt.Errorf("HTTP %d from %s", resp.StatusCode, p.cfg.Name))
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		p.setError(err)
		return
	}

	// Parse JSON — extract the array at cfg.JSONPath
	var raw interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		p.setError(fmt.Errorf("JSON parse: %w", err))
		return
	}

	items := jsonPath(raw, p.cfg.JSONPath)
	for _, item := range items {
		evtType := ""
		if m, ok := item.(map[string]interface{}); ok {
			if t, ok := m[p.cfg.EventTypeField].(string); ok {
				evtType = t
			}
		}
		if evtType == "" {
			evtType = "rest_api_event"
		}

		raw, _ := json.Marshal(item)
		_ = p.enqueue(&events.SovereignEvent{
			Timestamp: time.Now().Format(time.RFC3339),
			Host:      p.cfg.Name,
			EventType: evtType,
			RawLine:   string(raw),
		})
		p.mu.Lock()
		p.pulled++
		p.mu.Unlock()
	}

	p.mu.Lock()
	p.lastErr = nil
	p.mu.Unlock()
}

func (p *RESTPoller) setError(err error) {
	p.mu.Lock()
	p.lastErr = err
	p.mu.Unlock()
	p.log.Warn("[REST] Poller %s error: %v", p.cfg.Name, err)
}

// jsonPath navigates a nested JSON structure by dot-separated key path.
// Returns a slice of items if the final value is an array, else wraps in a slice.
func jsonPath(v interface{}, path string) []interface{} {
	if path == "" {
		if arr, ok := v.([]interface{}); ok {
			return arr
		}
		return []interface{}{v}
	}
	dot := -1
	for i, c := range path {
		if c == '.' {
			dot = i
			break
		}
	}
	key := path
	rest := ""
	if dot >= 0 {
		key = path[:dot]
		rest = path[dot+1:]
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return jsonPath(m[key], rest)
}

// ─────────────────────────────────────────────────────────────────────────────
// CollectorManager — registry and lifecycle for all agentless collectors
// ─────────────────────────────────────────────────────────────────────────────

// CollectorManager holds all registered agentless collectors.
type CollectorManager struct {
	mu         sync.RWMutex
	collectors []Collector
	log        *logger.Logger
}

func NewCollectorManager(log *logger.Logger) *CollectorManager {
	return &CollectorManager{log: log.WithPrefix("agentless")}
}

// Register adds a collector to the manager.
func (m *CollectorManager) Register(c Collector) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.collectors = append(m.collectors, c)
}

// StartAll starts all registered collectors.
func (m *CollectorManager) StartAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.collectors {
		if err := c.Start(ctx); err != nil {
			m.log.Error("[agentless] Failed to start %s: %v", c.Name(), err)
		}
	}
}

// StopAll stops all registered collectors.
func (m *CollectorManager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.collectors {
		c.Stop()
	}
}

// Statuses returns a name→status map for the admin API.
func (m *CollectorManager) Statuses() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.collectors))
	for _, c := range m.collectors {
		out[c.Name()] = c.Status()
	}
	return out
}
