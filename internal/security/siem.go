package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// SIEMType defines the SIEM system type
type SIEMType string

const (
	SIEMSyslog  SIEMType = "syslog"
	SIEMWebhook SIEMType = "webhook"
	SIEMSplunk  SIEMType = "splunk"
	SIEMElastic SIEMType = "elasticsearch"
)

// SIEMConfig holds SIEM integration settings
type SIEMConfig struct {
	Type          SIEMType      `json:"type"`
	Endpoint      string        `json:"endpoint"` // URL or host:port
	Token         string        `json:"token"`    // API token/key
	Index         string        `json:"index"`    // Elasticsearch index / Splunk index
	Protocol      string        `json:"protocol"` // "tcp" or "udp" for syslog
	Format        string        `json:"format"`   // "json", "cef", "leef"
	TLS           bool          `json:"tls"`
	Enabled       bool          `json:"enabled"`
	BatchSize     int           `json:"batch_size"`
	FlushInterval time.Duration `json:"flush_interval"`
}

// SIEMEvent is a security event to forward
type SIEMEvent struct {
	Timestamp string                 `json:"timestamp"`
	EventType string                 `json:"event_type"`
	Severity  string                 `json:"severity"` // "info", "low", "medium", "high", "critical"
	Source    string                 `json:"source"`
	HostID    string                 `json:"host_id,omitempty"`
	HostLabel string                 `json:"host_label,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// SIEMForwarder forwards security events to SIEM systems
type SIEMForwarder struct {
	mu      sync.RWMutex
	config  SIEMConfig
	log     *logger.Logger
	queue   []SIEMEvent
	client  *http.Client
	conn    net.Conn // for syslog
	stopCh  chan struct{}
	started bool
}

func NewSIEMForwarder(config SIEMConfig, log *logger.Logger) *SIEMForwarder {
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 10 * time.Second
	}

	return &SIEMForwarder{
		config: config,
		log:    log.WithPrefix("siem"),
		queue:  make([]SIEMEvent, 0, config.BatchSize),
		client: &http.Client{Timeout: 10 * time.Second},
		stopCh: make(chan struct{}),
	}
}

// Start begins the SIEM forwarder
func (f *SIEMForwarder) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.started {
		return nil
	}

	if !f.config.Enabled {
		return nil
	}

	// For syslog, establish connection
	if f.config.Type == SIEMSyslog {
		protocol := f.config.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		conn, err := net.DialTimeout(protocol, f.config.Endpoint, 5*time.Second)
		if err != nil {
			return fmt.Errorf("connect to syslog: %w", err)
		}
		f.conn = conn
	}

	f.started = true
	go f.flushLoop()

	f.log.Info("SIEM forwarder started: type=%s endpoint=%s", f.config.Type, f.config.Endpoint)
	return nil
}

// Stop stops the SIEM forwarder
func (f *SIEMForwarder) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.started {
		return
	}

	close(f.stopCh)
	f.started = false

	// Flush remaining events
	f.flush()

	if f.conn != nil {
		f.conn.Close()
	}
}

// Send queues a security event for forwarding
func (f *SIEMForwarder) Send(event SIEMEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.config.Enabled {
		return
	}

	event.Timestamp = time.Now().Format(time.RFC3339)
	event.Source = "sovereign-terminal"
	f.queue = append(f.queue, event)

	if len(f.queue) >= f.config.BatchSize {
		go f.flush()
	}
}

func (f *SIEMForwarder) RecordEvent(eventType, severity, userID, hostID string, details map[string]interface{}) {
	f.Send(SIEMEvent{
		EventType: eventType,
		Severity:  severity,
		UserID:    userID,
		HostID:    hostID,
		Details:   details,
	})
}

func (f *SIEMForwarder) Configure(config SIEMConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.config = config
}

// SendImmediate sends an event immediately (for critical events)
func (f *SIEMForwarder) SendImmediate(event SIEMEvent) error {
	event.Timestamp = time.Now().Format(time.RFC3339)
	event.Source = "sovereign-terminal"

	switch f.config.Type {
	case SIEMWebhook:
		return f.sendWebhook([]SIEMEvent{event})
	case SIEMSyslog:
		return f.sendSyslog(event)
	case SIEMSplunk:
		return f.sendSplunk([]SIEMEvent{event})
	case SIEMElastic:
		return f.sendElasticsearch([]SIEMEvent{event})
	default:
		return fmt.Errorf("unsupported SIEM type: %s", f.config.Type)
	}
}

func (f *SIEMForwarder) flushLoop() {
	ticker := time.NewTicker(f.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.stopCh:
			return
		case <-ticker.C:
			f.flush()
		}
	}
}

func (f *SIEMForwarder) flush() {
	f.mu.Lock()
	if len(f.queue) == 0 {
		f.mu.Unlock()
		return
	}

	events := make([]SIEMEvent, len(f.queue))
	copy(events, f.queue)
	f.queue = f.queue[:0]
	f.mu.Unlock()

	var err error
	switch f.config.Type {
	case SIEMWebhook:
		err = f.sendWebhook(events)
	case SIEMSyslog:
		for _, e := range events {
			if err2 := f.sendSyslog(e); err2 != nil {
				err = err2
			}
		}
	case SIEMSplunk:
		err = f.sendSplunk(events)
	case SIEMElastic:
		err = f.sendElasticsearch(events)
	}

	if err != nil {
		f.log.Error("SIEM flush failed: %v", err)
		// Re-queue failed events (with limit)
		f.mu.Lock()
		if len(f.queue)+len(events) <= f.config.BatchSize*10 {
			f.queue = append(events, f.queue...)
		}
		f.mu.Unlock()
	} else {
		f.log.Debug("Forwarded %d events to SIEM", len(events))
	}
}

func (f *SIEMForwarder) sendWebhook(events []SIEMEvent) error {
	payload := struct {
		Events []SIEMEvent `json:"events"`
	}{Events: events}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", f.config.Endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if f.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+f.config.Token)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}

	return nil
}

func (f *SIEMForwarder) sendSyslog(event SIEMEvent) error {
	if f.conn == nil {
		return fmt.Errorf("syslog not connected")
	}

	// RFC 5424 format
	severityMap := map[string]int{
		"critical": 2,
		"high":     3,
		"medium":   4,
		"low":      5,
		"info":     6,
	}

	severity := severityMap[event.Severity]
	if severity == 0 {
		severity = 6
	}

	// Facility 4 (security/authorization) + severity
	priority := 4*8 + severity

	msg := fmt.Sprintf("<%d>1 %s %s %s - - - %s",
		priority,
		event.Timestamp,
		"sovereign-terminal",
		event.EventType,
		event.Message,
	)

	_, err := fmt.Fprintln(f.conn, msg)
	return err
}

func (f *SIEMForwarder) sendSplunk(events []SIEMEvent) error {
	var buf bytes.Buffer

	for _, event := range events {
		// Unparseable timestamps fall back to 0 (epoch) so Splunk still ingests the event;
		// the SIEM forwarder should not drop alerts because of a malformed timestamp string.
		eventTime, _ := parseTime(event.Timestamp)
		splunkEvent := map[string]interface{}{
			"time":       eventTime.Unix(),
			"host":       "sovereign-terminal",
			"source":     "sovereign-terminal",
			"sourcetype": "sovereign:security",
			"index":      f.config.Index,
			"event":      event,
		}

		data, err := json.Marshal(splunkEvent)
		if err != nil {
			continue
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	req, err := http.NewRequest("POST",
		f.config.Endpoint+"/services/collector/event",
		&buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Splunk "+f.config.Token)

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("splunk HEC returned %d", resp.StatusCode)
	}

	return nil
}

func (f *SIEMForwarder) sendElasticsearch(events []SIEMEvent) error {
	var buf bytes.Buffer

	index := f.config.Index
	if index == "" {
		index = "sovereign-terminal"
	}

	for _, event := range events {
		// Bulk API format
		action := fmt.Sprintf(`{"index":{"_index":"%s"}}`, index)
		buf.WriteString(action)
		buf.WriteByte('\n')

		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	req, err := http.NewRequest("POST", f.config.Endpoint+"/_bulk", &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-ndjson")
	if f.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+f.config.Token)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("elasticsearch returned %d", resp.StatusCode)
	}

	return nil
}

// FormatCEF converts an event to Common Event Format
func FormatCEF(event SIEMEvent) string {
	severityMap := map[string]string{
		"info":     "1",
		"low":      "3",
		"medium":   "5",
		"high":     "7",
		"critical": "10",
	}

	sev := severityMap[event.Severity]
	if sev == "" {
		sev = "1"
	}

	// CEF:Version|Device Vendor|Device Product|Device Version|Signature ID|Name|Severity|Extension
	return fmt.Sprintf("CEF:0|SovereignSecurity|SovereignTerminal|1.0|%s|%s|%s|src=%s dst=%s msg=%s",
		event.EventType,
		event.Message,
		sev,
		event.Source,
		event.HostLabel,
		strings.ReplaceAll(event.Message, "|", "\\|"),
	)
}
