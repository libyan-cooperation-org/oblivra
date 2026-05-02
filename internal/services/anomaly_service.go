package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// AnomalyService catches "I've never seen this error string before"
// — the canonical observability anomaly. Mechanism:
//
//   1. Tokenise every event's message into a stable template by
//      replacing volatile tokens (numbers, IPs, UUIDs, paths,
//      quoted strings) with placeholders.
//   2. Hash the template + (sourceType, severity) → fingerprint.
//   3. If we've never seen that fingerprint before, raise an
//      anomaly:new-pattern alert.
//
// To avoid pager-storming on a fresh deployment, the first
// `warmupDuration` after startup is silent — we accumulate baseline
// templates without alerting. After warmup, anything genuinely new
// surfaces.
//
// Fingerprint storage is an in-memory map; eventually backed by a
// persistent store when the platform's warm tier supports per-service
// state. Capped at 100k entries (eviction by oldest) to bound memory.
type AnomalyService struct {
	log    *slog.Logger
	alerts *AlertService

	mu           sync.RWMutex
	fingerprints map[string]time.Time // hex fp → first-seen
	cap          int

	startedAt      time.Time
	warmupDuration time.Duration

	// Severity gate — only flag templates from events at or above this
	// severity. Avoids "every new debug line is an anomaly" noise.
	minSeverity events.Severity

	// Per-(sourceType, severity) sample threshold — wait until we've
	// seen at least N events from a source before treating new
	// templates as anomalies, so a sourceType showing up for the first
	// time doesn't fire on its first event.
	minSourceVolume int
	sourceVolume    map[string]int
}

func NewAnomalyService(log *slog.Logger, alerts *AlertService) *AnomalyService {
	return &AnomalyService{
		log: log, alerts: alerts,
		fingerprints:    map[string]time.Time{},
		cap:             100_000,
		startedAt:       time.Now().UTC(),
		warmupDuration:  30 * time.Minute,
		minSeverity:     events.SeverityWarn,
		minSourceVolume: 50,
		sourceVolume:    map[string]int{},
	}
}

func (s *AnomalyService) ServiceName() string { return "AnomalyService" }

// Observe is the platform processor hook. Cheap: one regex per event
// + a map lookup. Returns true if an anomaly was raised (mostly used
// for tests).
func (s *AnomalyService) Observe(ctx context.Context, ev events.Event) bool {
	if !severityAtLeast(ev.Severity, s.minSeverity) {
		return false
	}
	if ev.Message == "" {
		return false
	}
	tpl := template(ev.Message)
	if tpl == "" {
		return false
	}
	source := categorize(ev)
	fp := fingerprint(source, string(ev.Severity), tpl)

	s.mu.Lock()
	s.sourceVolume[source]++
	vol := s.sourceVolume[source]
	if _, seen := s.fingerprints[fp]; seen {
		s.mu.Unlock()
		return false
	}
	now := time.Now().UTC()
	s.fingerprints[fp] = now
	if len(s.fingerprints) > s.cap {
		s.evictOldest()
	}

	// Decide whether to raise an alert. Three gates:
	//   1. Past warmup? Otherwise we'd alert on every distinct message
	//      a fresh deployment sees.
	//   2. Source has crossed minSourceVolume? Otherwise a brand-new
	//      sourceType fires on its first event.
	//   3. Severity gate already checked above.
	inWarmup := now.Sub(s.startedAt) < s.warmupDuration
	belowVolume := vol < s.minSourceVolume
	s.mu.Unlock()

	if inWarmup || belowVolume {
		return false
	}
	if s.alerts != nil {
		s.alerts.Raise(ctx, Alert{
			TenantID: ev.TenantID,
			RuleID:   "anomaly:new-pattern",
			RuleName: "New log pattern observed",
			Severity: AlertSeverityMedium,
			HostID:   ev.HostID,
			Message:  "first sighting of message template (" + source + "): " + truncate(ev.Message, 200),
			MITRE:    nil,
			EventIDs: []string{ev.ID},
		})
	}
	return true
}

// Stats — surfaced on /metrics and the Overview view eventually.
type AnomalyStats struct {
	FingerprintsTracked int    `json:"fingerprintsTracked"`
	WarmupRemaining     string `json:"warmupRemaining"`
	StartedAt           string `json:"startedAt"`
}

func (s *AnomalyService) Stats() AnomalyStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rem := s.warmupDuration - time.Since(s.startedAt)
	if rem < 0 {
		rem = 0
	}
	return AnomalyStats{
		FingerprintsTracked: len(s.fingerprints),
		WarmupRemaining:     rem.Round(time.Second).String(),
		StartedAt:           s.startedAt.Format(time.RFC3339),
	}
}

// evictOldest drops the oldest fingerprint when the cap is exceeded.
// Called from inside Observe under the write lock.
func (s *AnomalyService) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	for k, ts := range s.fingerprints {
		if oldestKey == "" || ts.Before(oldestTime) {
			oldestKey = k
			oldestTime = ts
		}
	}
	if oldestKey != "" {
		delete(s.fingerprints, oldestKey)
	}
}

// ---- template extraction --------------------------------------------------

var (
	rxIP        = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	rxIPv6      = regexp.MustCompile(`\b[0-9a-fA-F]{0,4}(?::[0-9a-fA-F]{0,4}){2,}\b`)
	rxUUID      = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
	rxHexLong   = regexp.MustCompile(`\b[0-9a-fA-F]{16,}\b`)
	rxHexMed    = regexp.MustCompile(`\b[0-9a-fA-F]{8,15}\b`)
	// No trailing \b — "12.5ms" should normalise to "<N>ms", not "<N>.5ms".
	// \b at start still anchors so "ssh2" doesn't get butchered into "ssh<N>".
	rxNumber    = regexp.MustCompile(`\b\d+(\.\d+)?`)
	rxTimestamp = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:[\.,]\d+)?(?:Z|[+\-]\d{2}:?\d{2})?\b`)
	rxPath      = regexp.MustCompile(`(?:[A-Za-z]:)?(?:/[^\s/]+){2,}/?`)
	rxQuoted    = regexp.MustCompile(`"[^"]*"|'[^']*'`)
)

// template normalises a log message into a stable shape that repeats
// across distinct events of the same kind. Order matters: the more
// specific patterns must run first so they don't get half-eaten by a
// generic number replacement.
func template(msg string) string {
	s := msg
	s = rxTimestamp.ReplaceAllString(s, "<TS>")
	s = rxUUID.ReplaceAllString(s, "<UUID>")
	s = rxIP.ReplaceAllString(s, "<IP>")
	s = rxIPv6.ReplaceAllString(s, "<IP6>")
	s = rxQuoted.ReplaceAllString(s, "<STR>")
	s = rxPath.ReplaceAllString(s, "<PATH>")
	s = rxHexLong.ReplaceAllString(s, "<HEX>")
	s = rxHexMed.ReplaceAllString(s, "<HEX>")
	s = rxNumber.ReplaceAllString(s, "<N>")
	// Collapse repeated whitespace.
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 512 {
		s = s[:512]
	}
	return s
}

func fingerprint(source, severity, tpl string) string {
	h := sha256.New()
	h.Write([]byte(source))
	h.Write([]byte("|"))
	h.Write([]byte(severity))
	h.Write([]byte("|"))
	h.Write([]byte(tpl))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func severityAtLeast(have, min events.Severity) bool {
	rank := map[events.Severity]int{
		events.SeverityDebug:    0,
		events.SeverityInfo:     1,
		events.SeverityNotice:   2,
		events.SeverityWarn:     3,
		events.SeverityError:    4,
		events.SeverityCritical: 5,
		events.SeverityAlert:    6,
	}
	return rank[have] >= rank[min]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
