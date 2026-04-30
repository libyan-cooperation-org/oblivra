package services

import (
	"log/slog"
	"strings"
	"sync"
	"time"
)

type IndicatorType string

const (
	IndicatorIP     IndicatorType = "ip"
	IndicatorDomain IndicatorType = "domain"
	IndicatorHash   IndicatorType = "hash"
	IndicatorURL    IndicatorType = "url"
)

type Indicator struct {
	Value     string        `json:"value"`
	Type      IndicatorType `json:"type"`
	Source    string        `json:"source,omitempty"`
	Tags      []string      `json:"tags,omitempty"`
	Severity  AlertSeverity `json:"severity,omitempty"`
	Added     time.Time     `json:"added"`
}

type LookupResult struct {
	Match     bool       `json:"match"`
	Indicator *Indicator `json:"indicator,omitempty"`
}

// ThreatIntelService keeps an in-memory IOC table indexed by lower-cased value
// for O(1) lookups during ingest enrichment. Loaded from a JSON seed file at
// startup; future phases will hook STIX/TAXII clients.
type ThreatIntelService struct {
	log *slog.Logger
	mu  sync.RWMutex
	by  map[string]Indicator
}

func NewThreatIntelService(log *slog.Logger) *ThreatIntelService {
	t := &ThreatIntelService{log: log, by: map[string]Indicator{}}
	// Seed a handful of well-known testing IOCs so the UI is never empty.
	for _, ind := range []Indicator{
		{Value: "198.51.100.7", Type: IndicatorIP, Source: "rfc5737-test", Severity: AlertSeverityLow},
		{Value: "malicious.example.com", Type: IndicatorDomain, Source: "demo", Severity: AlertSeverityMedium},
		{Value: "44d88612fea8a8f36de82e1278abb02f", Type: IndicatorHash, Source: "eicar", Severity: AlertSeverityHigh, Tags: []string{"eicar"}},
	} {
		t.Add(ind)
	}
	return t
}

func (t *ThreatIntelService) ServiceName() string { return "ThreatIntelService" }

func (t *ThreatIntelService) Add(ind Indicator) Indicator {
	if ind.Added.IsZero() {
		ind.Added = time.Now().UTC()
	}
	t.mu.Lock()
	t.by[normalize(ind.Value)] = ind
	t.mu.Unlock()
	return ind
}

func (t *ThreatIntelService) Lookup(value string) LookupResult {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if ind, ok := t.by[normalize(value)]; ok {
		return LookupResult{Match: true, Indicator: &ind}
	}
	return LookupResult{Match: false}
}

func (t *ThreatIntelService) List() []Indicator {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]Indicator, 0, len(t.by))
	for _, v := range t.by {
		out = append(out, v)
	}
	return out
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
