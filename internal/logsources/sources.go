package logsources

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// SourceType defines the log provider
type SourceType string

const (
	SourceElasticsearch SourceType = "elasticsearch"
	SourceLoki          SourceType = "loki"
	SourceSplunk        SourceType = "splunk"
)

// LogSource represents an external log provider configuration
type LogSource struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Type          SourceType `json:"type"`
	URL           string     `json:"url"`
	Enabled       bool       `json:"enabled"`
	APIKey        string     `json:"api_key,omitempty"`
	Username      string     `json:"username,omitempty"`
	Password      string     `json:"password,omitempty"`
	Index         string     `json:"index,omitempty"`  // ES index pattern, Splunk index
	OrgID         string     `json:"org_id,omitempty"` // Loki X-Scope-OrgID
	TLSSkipVerify bool       `json:"tls_skip_verify"`  // Allow self-signed certs
	Tags          []string   `json:"tags,omitempty"`   // Grouping tags (e.g. "production", "dc-1")
}

// LogResult is a generic log entry returned from any source
type LogResult struct {
	Timestamp string            `json:"timestamp"`
	Source    string            `json:"source"`
	Host      string            `json:"host"`
	Message   string            `json:"message"`
	Level     string            `json:"level"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// SourceManager manages external log source connections
type SourceManager struct {
	mu      sync.RWMutex
	sources map[string]LogSource
	clients map[string]*http.Client // Per-source clients (TLS config varies)
	log     *logger.Logger
}

// NewSourceManager creates a new log sources manager
func NewSourceManager(log *logger.Logger) *SourceManager {
	return &SourceManager{
		sources: make(map[string]LogSource),
		clients: make(map[string]*http.Client),
		log:     log,
	}
}

// clientFor returns an HTTP client configured for the source (TLS, pooling)
func (m *SourceManager) clientFor(src LogSource) *http.Client {
	if c, ok := m.clients[src.ID]; ok {
		return c
	}
	if src.TLSSkipVerify {
		m.log.Warn("⚠️ SECURITY: TLS certificate verification disabled for log source %s (%s)", src.Name, src.URL)
	}
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: src.TLSSkipVerify,
		},
	}
	c := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	m.clients[src.ID] = c
	return c
}

// AddSource adds or updates a log source
func (m *SourceManager) AddSource(src LogSource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Invalidate cached client on update (TLS config might change)
	delete(m.clients, src.ID)
	m.sources[src.ID] = src
	m.log.Info("Log source added: %s (%s @ %s)", src.Name, src.Type, src.URL)
}

// RemoveSource removes a log source by ID
func (m *SourceManager) RemoveSource(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sources, id)
	delete(m.clients, id)
	m.log.Info("Log source removed: %s", id)
}

// GetSources returns all configured sources
func (m *SourceManager) GetSources() []LogSource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]LogSource, 0, len(m.sources))
	for _, s := range m.sources {
		result = append(result, s)
	}
	return result
}

// SetSources replaces all sources (for loading from persistence)
func (m *SourceManager) SetSources(sources []LogSource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sources = make(map[string]LogSource, len(sources))
	m.clients = make(map[string]*http.Client)
	for _, s := range sources {
		m.sources[s.ID] = s
	}
}

// GetSource returns a single source by ID
func (m *SourceManager) GetSource(id string) (LogSource, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sources[id]
	return s, ok
}

// Query executes a search against a specific source
func (m *SourceManager) Query(sourceID string, query string, timeRange string, limit, offset int) ([]LogResult, error) {
	m.mu.RLock()
	src, ok := m.sources[sourceID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceID)
	}
	if !src.Enabled {
		return nil, fmt.Errorf("source disabled: %s", src.Name)
	}

	switch src.Type {
	case SourceElasticsearch:
		return m.queryElasticsearch(src, query, timeRange, limit, offset)
	case SourceLoki:
		return m.queryLoki(src, query, timeRange, limit, offset)
	case SourceSplunk:
		return m.querySplunk(src, query, timeRange, limit, offset)
	default:
		return nil, fmt.Errorf("unknown source type: %s", src.Type)
	}
}

// TestConnection verifies a source is reachable
func (m *SourceManager) TestConnection(sourceID string) (bool, string) {
	m.mu.RLock()
	src, ok := m.sources[sourceID]
	m.mu.RUnlock()

	if !ok {
		return false, "source not found"
	}

	switch src.Type {
	case SourceElasticsearch:
		return m.testElasticsearch(src)
	case SourceLoki:
		return m.testLoki(src)
	case SourceSplunk:
		return m.testSplunk(src)
	default:
		return false, "unknown source type"
	}
}

// ─── Elasticsearch ───

func (m *SourceManager) testElasticsearch(src LogSource) (bool, string) {
	req, _ := http.NewRequest("GET", src.URL, nil)
	m.setAuth(req, src)

	client := m.clientFor(src)
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var info map[string]interface{}
	json.Unmarshal(body, &info)

	version := "unknown"
	if v, ok := info["version"].(map[string]interface{}); ok {
		if num, ok := v["number"].(string); ok {
			version = num
		}
	}
	return true, fmt.Sprintf("Elasticsearch %s", version)
}

func (m *SourceManager) queryElasticsearch(src LogSource, query string, timeRange string, limit, offset int) ([]LogResult, error) {
	index := src.Index
	if index == "" {
		index = "*"
	}

	if timeRange == "" {
		timeRange = "24h" // default
	}

	esQuery := map[string]interface{}{
		"size": limit,
		"from": offset,
		"sort": []map[string]interface{}{
			{"@timestamp": map[string]string{"order": "desc"}},
		},
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"query_string": map[string]interface{}{
							"query": query,
						},
					},
				},
				"filter": []map[string]interface{}{
					{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{
								"gte": fmt.Sprintf("now-%s", timeRange),
							},
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(esQuery)
	reqURL := fmt.Sprintf("%s/%s/_search", strings.TrimRight(src.URL, "/"), index)

	req, _ := http.NewRequest("POST", reqURL, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	m.setAuth(req, src)

	client := m.clientFor(src)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elasticsearch error %d: %s", resp.StatusCode, string(respBody))
	}

	var esResp struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	results := make([]LogResult, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		lr := LogResult{
			Source: src.Name,
			Fields: make(map[string]string),
		}
		for k, v := range hit.Source {
			val := fmt.Sprintf("%v", v)
			switch k {
			case "@timestamp", "timestamp":
				lr.Timestamp = val
			case "host", "hostname", "host.name":
				lr.Host = val
			case "message", "msg", "log":
				lr.Message = val
			case "level", "severity", "log.level":
				lr.Level = val
			default:
				lr.Fields[k] = fmt.Sprintf("%v", v)
			}
		}
		results = append(results, lr)
	}

	// Apply offset slice
	if offset > 0 {
		if offset >= len(results) {
			return []LogResult{}, nil
		}
		results = results[offset:]
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// ─── Grafana Loki ───

func (m *SourceManager) testLoki(src LogSource) (bool, string) {
	reqURL := fmt.Sprintf("%s/ready", strings.TrimRight(src.URL, "/"))
	req, _ := http.NewRequest("GET", reqURL, nil)
	m.setAuth(req, src)
	if src.OrgID != "" {
		req.Header.Set("X-Scope-OrgID", src.OrgID)
	}

	client := m.clientFor(src)
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, "Loki is ready"
	}
	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

func (m *SourceManager) queryLoki(src LogSource, query string, timeRange string, limit, offset int) ([]LogResult, error) {
	// Parse timeRange into duration
	if timeRange == "" {
		timeRange = "24h"
	}
	dur, err := time.ParseDuration(timeRange)
	if err != nil {
		// handle 'd' suffix (e.g. 7d) since Go time.ParseDuration doesn't support days natively
		if strings.HasSuffix(timeRange, "d") {
			days, _ := strconv.Atoi(strings.TrimSuffix(timeRange, "d"))
			dur = time.Duration(days*24) * time.Hour
		} else {
			dur = 24 * time.Hour
		}
	}

	start := time.Now().Add(-dur).UnixNano()

	// Loki query_range doesn't support 'offset', but we can approximate it by loading more hits
	// and cutting off the top if needed. Loki has a max limit of 5000 usually.
	effectiveLimit := limit + offset
	if effectiveLimit > 5000 {
		effectiveLimit = 5000
	}

	reqURL := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&limit=%d&direction=backward&start=%d",
		strings.TrimRight(src.URL, "/"),
		url.QueryEscape(query),
		effectiveLimit,
		start,
	)

	req, _ := http.NewRequest("GET", reqURL, nil)
	m.setAuth(req, src)
	if src.OrgID != "" {
		req.Header.Set("X-Scope-OrgID", src.OrgID)
	}

	client := m.clientFor(src)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("loki query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki error %d: %s", resp.StatusCode, string(respBody))
	}

	var lokiResp struct {
		Data struct {
			Result []struct {
				Stream map[string]string `json:"stream"`
				Values [][]string        `json:"values"` // [timestamp_ns, line]
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		return nil, fmt.Errorf("failed to parse loki response: %w", err)
	}

	var results []LogResult
	for _, stream := range lokiResp.Data.Result {
		for _, val := range stream.Values {
			if len(val) < 2 {
				continue
			}

			// Convert Loki nanosecond timestamp to RFC3339
			ts := val[0]
			if nsec, err := strconv.ParseInt(ts, 10, 64); err == nil {
				ts = time.Unix(0, nsec).Format(time.RFC3339)
			}

			lr := LogResult{
				Timestamp: ts,
				Message:   val[1],
				Source:    src.Name,
				Host:      stream.Stream["host"],
				Level:     stream.Stream["level"],
				Fields:    make(map[string]string),
			}
			for k, v := range stream.Stream {
				if k != "host" && k != "level" {
					lr.Fields[k] = v
				}
			}
			results = append(results, lr)
		}
	}
	return results, nil
}

// ─── Splunk ───

func (m *SourceManager) testSplunk(src LogSource) (bool, string) {
	reqURL := fmt.Sprintf("%s/services/server/info?output_mode=json", strings.TrimRight(src.URL, "/"))
	req, _ := http.NewRequest("GET", reqURL, nil)
	m.setAuth(req, src)

	client := m.clientFor(src)
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, "Splunk connected"
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return false, "authentication failed — check credentials"
	}
	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

func (m *SourceManager) querySplunk(src LogSource, query string, timeRange string, limit, offset int) ([]LogResult, error) {
	searchURL := fmt.Sprintf("%s/services/search/jobs/export", strings.TrimRight(src.URL, "/"))

	if !strings.HasPrefix(strings.TrimSpace(query), "search") && !strings.HasPrefix(strings.TrimSpace(query), "|") {
		query = "search " + query
	}
	// Note: SPL handles offsets poorly through export API, so we fetch offset+limit and then slice it natively
	effectiveLimit := limit + offset
	query += fmt.Sprintf(" | head %d", effectiveLimit)

	if timeRange == "" {
		timeRange = "24h"
	}

	// Use proper url encoding
	formData := url.Values{}
	formData.Set("search", query)
	formData.Set("output_mode", "json")
	formData.Set("earliest_time", fmt.Sprintf("-%s", timeRange))

	req, _ := http.NewRequest("POST", searchURL, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	m.setAuth(req, src)

	client := m.clientFor(src)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("splunk query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("splunk error %d: %s", resp.StatusCode, string(respBody))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read splunk response: %w", err)
	}

	var results []LogResult
	for _, line := range strings.Split(string(bodyBytes), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event struct {
			Result map[string]interface{} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Result == nil {
			continue
		}

		lr := LogResult{
			Source: src.Name,
			Fields: make(map[string]string),
		}
		for k, v := range event.Result {
			val := fmt.Sprintf("%v", v)
			switch k {
			case "_time":
				lr.Timestamp = val
			case "host":
				lr.Host = val
			case "_raw":
				lr.Message = val
			case "severity", "level":
				lr.Level = val
			default:
				if !strings.HasPrefix(k, "_") {
					lr.Fields[k] = val
				}
			}
		}
		results = append(results, lr)
	}

	// Apply offset slice
	if offset > 0 {
		if offset >= len(results) {
			return []LogResult{}, nil
		}
		results = results[offset:]
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// ─── Helpers ───

func (m *SourceManager) setAuth(req *http.Request, src LogSource) {
	if src.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+src.APIKey)
	} else if src.Username != "" {
		req.SetBasicAuth(src.Username, src.Password)
	}
}

// GetSourcesByTag returns sources matching any of the given tags
func (m *SourceManager) GetSourcesByTag(tag string) []LogSource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []LogSource
	for _, s := range m.sources {
		for _, t := range s.Tags {
			if strings.EqualFold(t, tag) {
				result = append(result, s)
				break
			}
		}
	}
	return result
}

// GetAllTags returns a deduplicated list of all tags across sources
func (m *SourceManager) GetAllTags() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	seen := make(map[string]bool)
	var tags []string
	for _, s := range m.sources {
		for _, t := range s.Tags {
			lower := strings.ToLower(t)
			if !seen[lower] {
				seen[lower] = true
				tags = append(tags, t)
			}
		}
	}
	return tags
}

// StreamLoki establishes a WebSocket connection to Loki and emits logs via Wails events
func (m *SourceManager) StreamLoki(ctx context.Context, emitFunc func(logsources LogResult), sourceID string, query string) error {
	m.mu.RLock()
	src, ok := m.sources[sourceID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("source not found: %s", sourceID)
	}

	wsURL := strings.Replace(strings.TrimRight(src.URL, "/"), "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/loki/api/v1/tail?query=%s", wsURL, url.QueryEscape(query))

	header := http.Header{}
	if src.APIKey != "" {
		pass := src.Password
		if pass == "" {
			header.Set("Authorization", "Bearer "+src.APIKey)
		} else {
			header.Set("Authorization", "Basic "+src.APIKey+":"+src.Password) // Trusting existing format where APIKey holds username and Password holds password
		}
	} else if src.Username != "" && src.Password != "" {
		header.Set("Authorization", "Basic "+src.Username+":"+src.Password)
	}
	if src.OrgID != "" {
		header.Set("X-Scope-OrgID", src.OrgID)
	}

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: src.TLSSkipVerify},
	}

	c, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	m.log.Info("Started streaming Loki tail for source %s", sourceID)

	// Close websocket when context is cancelled
	go func() {
		<-ctx.Done()
		m.log.Info("Stopped streaming Loki tail for source %s", sourceID)
		c.Close()
	}()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				m.log.Error("Loki websocket closed unexpectedly: %v", err)
			}
			return err
		}

		var tailResp struct {
			Streams []struct {
				Stream map[string]string `json:"stream"`
				Values [][]string        `json:"values"`
			} `json:"streams"`
		}

		if err := json.Unmarshal(message, &tailResp); err != nil {
			continue // skip malformed chunks
		}

		// Fast format and emit to callback
		for _, stream := range tailResp.Streams {
			for _, val := range stream.Values {
				if len(val) < 2 {
					continue
				}

				ts := val[0]
				if nsec, err := strconv.ParseInt(ts, 10, 64); err == nil {
					ts = time.Unix(0, nsec).Format(time.RFC3339)
				}

				lr := LogResult{
					Timestamp: ts,
					Message:   val[1],
					Source:    src.Name,
					Host:      stream.Stream["host"],
					Level:     stream.Stream["level"],
					Fields:    make(map[string]string),
				}
				for k, v := range stream.Stream {
					if k != "host" && k != "level" {
						lr.Fields[k] = v
					}
				}

				emitFunc(lr)
			}
		}
	}
}

// TailLoki connects to Loki's /loki/api/v1/tail SSE endpoint and returns live log entries.
// It blocks until the context is cancelled, pushing results to the returned channel.
func (m *SourceManager) TailLoki(sourceID, query string, limit int) ([]LogResult, error) {
	m.mu.RLock()
	src, ok := m.sources[sourceID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceID)
	}
	if src.Type != SourceLoki {
		return nil, fmt.Errorf("tail is only supported for Loki sources")
	}

	// Use query_range with very recent window as a "tail" approximation
	// (real WebSocket tail requires persistent connection, not suitable for Wails RPC)
	now := time.Now()
	start := now.Add(-30 * time.Second) // Last 30 seconds

	reqURL := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&limit=%d&direction=backward&start=%d&end=%d",
		strings.TrimRight(src.URL, "/"),
		url.QueryEscape(query),
		limit,
		start.UnixNano(),
		now.UnixNano(),
	)

	req, _ := http.NewRequest("GET", reqURL, nil)
	m.setAuth(req, src)
	if src.OrgID != "" {
		req.Header.Set("X-Scope-OrgID", src.OrgID)
	}

	client := m.clientFor(src)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("loki tail failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki tail error %d: %s", resp.StatusCode, string(respBody))
	}

	var lokiResp struct {
		Data struct {
			Result []struct {
				Stream map[string]string `json:"stream"`
				Values [][]string        `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		return nil, fmt.Errorf("failed to parse loki tail response: %w", err)
	}

	var results []LogResult
	for _, stream := range lokiResp.Data.Result {
		for _, val := range stream.Values {
			if len(val) < 2 {
				continue
			}
			ts := val[0]
			if nsec, err := strconv.ParseInt(ts, 10, 64); err == nil {
				ts = time.Unix(0, nsec).Format(time.RFC3339)
			}
			lr := LogResult{
				Timestamp: ts,
				Message:   val[1],
				Source:    src.Name,
				Host:      stream.Stream["host"],
				Level:     stream.Stream["level"],
				Fields:    make(map[string]string),
			}
			for k, v := range stream.Stream {
				if k != "host" && k != "level" {
					lr.Fields[k] = v
				}
			}
			results = append(results, lr)
		}
	}
	return results, nil
}
