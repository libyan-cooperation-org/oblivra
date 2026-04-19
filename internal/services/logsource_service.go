package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/logsources"
	
)

const (
	configKeyLogSources  = "log_sources"
	configKeySavedSearch = "saved_searches"
)

// ConnectionResult wraps test connection output for Wails
type ConnectionResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// LogSourceSavedSearch stores a reusable query per source
type LogSourceSavedSearch struct {
	ID       string `json:"id"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
	Query    string `json:"query"`
	Created  string `json:"created"`
}

// SourceHealth tracks live connectivity status
type SourceHealth struct {
	SourceID  string `json:"source_id"`
	Name      string `json:"name"`
	Healthy   bool   `json:"healthy"`
	Message   string `json:"message"`
	CheckedAt string `json:"checked_at"`
}

// CorrelationResult represents surrounding log context
type CorrelationResult struct {
	Before []logsources.LogResult `json:"before"`
	Target logsources.LogResult   `json:"target"`
	After  []logsources.LogResult `json:"after"`
}

// LogSourceService exposes external log source management to the frontend
type LogSourceService struct {
	BaseService
	ctx       context.Context
	manager   *logsources.SourceManager
	analytics analytics.Engine
	bus       *eventbus.Bus
	log       *logger.Logger

	// Health monitoring
	healthMu     *sync.RWMutex
	healthStatus map[string]SourceHealth
	healthDone   chan struct{}

	// Saved searches
	savedMu       *sync.RWMutex
	savedSearches []LogSourceSavedSearch

	// Rate limiting
	rateMu    *sync.Mutex
	lastQuery map[string]time.Time

	// Active Streams
	streamMu      *sync.Mutex
	activeStreams map[string]context.CancelFunc
}

func (s *LogSourceService) Name() string { return "log-source-service" }

// Dependencies returns service dependencies.
func (s *LogSourceService) Dependencies() []string {
	return []string{}
}

func NewLogSourceService(manager *logsources.SourceManager, ae analytics.Engine, bus *eventbus.Bus, log *logger.Logger) *LogSourceService {
	return &LogSourceService{
		manager:       manager,
		analytics:     ae,
		bus:           bus,
		log:           log,
		healthMu:      &sync.RWMutex{},
		healthStatus:  make(map[string]SourceHealth),
		healthDone:    make(chan struct{}),
		savedMu:       &sync.RWMutex{},
		savedSearches: make([]LogSourceSavedSearch, 0),
		rateMu:        &sync.Mutex{},
		lastQuery:     make(map[string]time.Time),
		streamMu:      &sync.Mutex{},
		activeStreams: make(map[string]context.CancelFunc),
	}
}

func (s *LogSourceService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.loadPersistedSources()
	s.loadPersistedSearches()
	go s.healthMonitorLoop()
	return nil
}

func (s *LogSourceService) Stop(ctx context.Context) error {
	close(s.healthDone)
	return nil
}

// ─── Persistence ───

func (s *LogSourceService) loadPersistedSources() {
	if s.analytics == nil {
		return
	}
	ctx := database.WithGlobalSearch(s.ctx)
	data, err := s.analytics.LoadConfig(ctx, configKeyLogSources)
	if err != nil {
		return
	}
	var sources []logsources.LogSource
	if json.Unmarshal([]byte(data), &sources) == nil {
		// Deobfuscate credentials
		for i := range sources {
			sources[i].APIKey = deobfuscate(sources[i].APIKey)
			sources[i].Password = deobfuscate(sources[i].Password)
		}
		s.manager.SetSources(sources)
		s.log.Info("Restored %d log sources from database", len(sources))
	}
}

func (s *LogSourceService) persistSources() {
	if s.analytics == nil {
		return
	}
	sources := s.manager.GetSources()
	// Obfuscate credentials before saving
	safe := make([]logsources.LogSource, len(sources))
	copy(safe, sources)
	for i := range safe {
		safe[i].APIKey = obfuscate(safe[i].APIKey)
		safe[i].Password = obfuscate(safe[i].Password)
	}
	if data, err := json.Marshal(safe); err == nil {
		ctx := database.WithGlobalSearch(s.ctx)
		s.analytics.SaveConfig(ctx, configKeyLogSources, string(data))
	}
}

func (s *LogSourceService) loadPersistedSearches() {
	if s.analytics == nil {
		return
	}
	ctx := database.WithGlobalSearch(s.ctx)
	data, err := s.analytics.LoadConfig(ctx, configKeySavedSearch)
	if err != nil {
		return
	}
	s.savedMu.Lock()
	defer s.savedMu.Unlock()
	json.Unmarshal([]byte(data), &s.savedSearches)
}

func (s *LogSourceService) persistSearches() {
	if s.analytics == nil {
		return
	}
	s.savedMu.RLock()
	defer s.savedMu.RUnlock()
	if data, err := json.Marshal(s.savedSearches); err == nil {
		ctx := database.WithGlobalSearch(s.ctx)
		s.analytics.SaveConfig(ctx, configKeySavedSearch, string(data))
	}
}

// ─── Source CRUD ───

func (s *LogSourceService) AddSource(src logsources.LogSource) {
	// Preserve existing credentials if UI sent the mask string
	if src.APIKey == "********" || src.Password == "********" {
		existing := s.manager.GetSources()
		for _, e := range existing {
			if e.ID == src.ID {
				if src.APIKey == "********" {
					src.APIKey = e.APIKey
				}
				if src.Password == "********" {
					src.Password = e.Password
				}
				break
			}
		}
	}
	s.manager.AddSource(src)
	s.persistSources()

	s.bus.Publish("config:logsource:changed", map[string]string{
		"source_id": src.ID,
		"old":       "config_update",
		"new":       "applied",
	})
}

func (s *LogSourceService) RemoveSource(id string) {
	s.manager.RemoveSource(id)
	s.persistSources()

	s.bus.Publish("config:logsource:changed", map[string]string{
		"source_id": id,
		"old":       "active",
		"new":       "removed",
	})
	// Remove associated saved searches
	s.savedMu.Lock()
	filtered := make([]LogSourceSavedSearch, 0)
	for _, ss := range s.savedSearches {
		if ss.SourceID != id {
			filtered = append(filtered, ss)
		}
	}
	s.savedSearches = filtered
	s.savedMu.Unlock()
	s.persistSearches()
}

func (s *LogSourceService) GetSources() []logsources.LogSource {
	sources := s.manager.GetSources()
	safe := make([]logsources.LogSource, len(sources))
	copy(safe, sources)
	for i := range safe {
		if safe[i].APIKey != "" {
			safe[i].APIKey = "********"
		}
		if safe[i].Password != "" {
			safe[i].Password = "********"
		}
	}
	return safe
}

func (s *LogSourceService) GetSourcesByTag(tag string) []logsources.LogSource {
	return s.manager.GetSourcesByTag(tag)
}

func (s *LogSourceService) GetAllTags() []string {
	return s.manager.GetAllTags()
}

func (s *LogSourceService) TestConnection(sourceID string) ConnectionResult {
	ok, msg := s.manager.TestConnection(sourceID)
	return ConnectionResult{OK: ok, Message: msg}
}

func (s *LogSourceService) canQuery(sourceID string) bool {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	last, ok := s.lastQuery[sourceID]
	now := time.Now()
	if ok && now.Sub(last) < 500*time.Millisecond {
		return false // Rate limited (max 2 req/sec per source)
	}
	s.lastQuery[sourceID] = now
	return true
}

func (s *LogSourceService) QuerySource(sourceID, query, timeRange string, limit, offset int) ([]logsources.LogResult, error) {
	if !s.canQuery(sourceID) {
		return nil, fmt.Errorf("rate limit exceeded for source %s (max 2 requests per second)", sourceID)
	}
	return s.manager.Query(sourceID, query, timeRange, limit, offset)
}

func (s *LogSourceService) TailLoki(sourceID, query string, limit int) ([]logsources.LogResult, error) {
	if !s.canQuery(sourceID) {
		return nil, fmt.Errorf("rate limit exceeded for loki tail")
	}
	return s.manager.TailLoki(sourceID, query, limit)
}

func (s *LogSourceService) StartLokiStream(sourceID, query string) error {
	s.streamMu.Lock()
	if cancel, ok := s.activeStreams[sourceID]; ok {
		cancel() // cancel existing stream for this source if one exists
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.activeStreams[sourceID] = cancel
	s.streamMu.Unlock()

	// Launch stream in background
	go func() {
		// Provide an emit callback to SourceManager
		err := s.manager.StreamLoki(ctx, func(res logsources.LogResult) {
			EmitEvent("loki-stream-"+sourceID, res)
		}, sourceID, query)

		if err != nil && err != context.Canceled {
			s.log.Error("Loki stream error for %s: %v", sourceID, err)
			EmitEvent("loki-stream-error-"+sourceID, err.Error())
		}

		s.streamMu.Lock()
		delete(s.activeStreams, sourceID)
		s.streamMu.Unlock()
	}()

	return nil
}

func (s *LogSourceService) StopLokiStream(sourceID string) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	if cancel, ok := s.activeStreams[sourceID]; ok {
		cancel()
		delete(s.activeStreams, sourceID)
	}
}

// ─── Multi-Source Unified Search ───

// SearchAllSources queries ALL enabled sources in parallel and merges results
func (s *LogSourceService) SearchAllSources(query, timeRange string, limit, offset int) ([]logsources.LogResult, error) {
	sources := s.GetSources()
	var allResults []logsources.LogResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	errCh := make(chan error, len(sources))

	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		wg.Add(1)
		go func(sourceID string) {
			defer wg.Done()
			if !s.canQuery(sourceID) {
				return
			}
			res, err := s.manager.Query(sourceID, query, timeRange, limit, offset)
			if err != nil {
				errCh <- fmt.Errorf("%s: %v", sourceID, err)
				return
			}
			mu.Lock()
			allResults = append(allResults, res...)
			mu.Unlock()
		}(src.ID)
	}

	wg.Wait()
	close(errCh)

	var multiErr error
	for err := range errCh {
		if multiErr == nil {
			multiErr = err
		} else {
			multiErr = fmt.Errorf("%w; %v", multiErr, err)
		}
	}

	return allResults, multiErr
}

// ─── Saved Searches ───

func (s *LogSourceService) SaveSearch(sourceID, name, query string) LogSourceSavedSearch {
	ss := LogSourceSavedSearch{
		ID:       fmt.Sprintf("ss-%d", time.Now().UnixNano()),
		SourceID: sourceID,
		Name:     name,
		Query:    query,
		Created:  time.Now().Format(time.RFC3339),
	}
	s.savedMu.Lock()
	s.savedSearches = append(s.savedSearches, ss)
	s.savedMu.Unlock()
	s.persistSearches()
	return ss
}

func (s *LogSourceService) GetSavedSearches() []LogSourceSavedSearch {
	s.savedMu.RLock()
	defer s.savedMu.RUnlock()
	cp := make([]LogSourceSavedSearch, len(s.savedSearches))
	copy(cp, s.savedSearches)
	return cp
}

func (s *LogSourceService) DeleteSavedSearch(id string) {
	s.savedMu.Lock()
	for i, ss := range s.savedSearches {
		if ss.ID == id {
			s.savedSearches = append(s.savedSearches[:i], s.savedSearches[i+1:]...)
			break
		}
	}
	s.savedMu.Unlock()
	s.persistSearches()
}

// ─── Source Health Monitoring ───

func (s *LogSourceService) healthMonitorLoop() {
	// Check on startup
	s.checkAllHealth()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.checkAllHealth()
		case <-s.healthDone:
			return
		}
	}
}

func (s *LogSourceService) checkAllHealth() {
	sources := s.manager.GetSources()
	var wg sync.WaitGroup

	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		wg.Add(1)
		go func(source logsources.LogSource) {
			defer wg.Done()
			ok, msg := s.manager.TestConnection(source.ID)
			h := SourceHealth{
				SourceID:  source.ID,
				Name:      source.Name,
				Healthy:   ok,
				Message:   msg,
				CheckedAt: time.Now().Format(time.RFC3339),
			}
			s.healthMu.Lock()
			s.healthStatus[source.ID] = h
			s.healthMu.Unlock()
		}(src)
	}
	wg.Wait()
}

// GetHealthStatus returns current health for all configured sources
func (s *LogSourceService) GetHealthStatus() []SourceHealth {
	s.healthMu.RLock()
	defer s.healthMu.RUnlock()
	result := make([]SourceHealth, 0, len(s.healthStatus))
	for _, h := range s.healthStatus {
		result = append(result, h)
	}
	return result
}

// ─── Export Results ───

// ExportCSV returns query results as a CSV string
func (s *LogSourceService) ExportCSV(sourceID, query, timeRange string, limit int) (string, error) {
	results, err := s.manager.Query(sourceID, query, timeRange, limit, 0)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("timestamp,host,level,source,message\n")
	for _, r := range results {
		// Escape commas and quotes in message
		msg := strings.ReplaceAll(r.Message, "\"", "\"\"")
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,\"%s\"\n",
			r.Timestamp, r.Host, r.Level, r.Source, msg))
	}
	return sb.String(), nil
}

// ExportJSON returns query results as a JSON string
func (s *LogSourceService) ExportJSON(sourceID, query, timeRange string, limit int) (string, error) {
	results, err := s.manager.Query(sourceID, query, timeRange, limit, 0)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ─── Credential Obfuscation ───

func obfuscate(s string) string {
	if s == "" {
		return ""
	}
	return "obf:" + base64.StdEncoding.EncodeToString([]byte(s))
}

func deobfuscate(s string) string {
	if !strings.HasPrefix(s, "obf:") {
		return s // Already plaintext (legacy)
	}
	data, err := base64.StdEncoding.DecodeString(s[4:])
	if err != nil {
		return s
	}
	return string(data)
}
