// Package trust assigns a trust classification to every event flowing
// through the platform. The grades match the Beta-1 task tracker:
//
//	Verified — agent-signed (mTLS fingerprint present) AND content hash valid
//	Consistent — same event seen via two or more independent ingest paths /
//	             sources (corroborated)
//	Suspicious — a clock-skew or sequence anomaly is attached
//	Untrusted — single anonymous source / no provenance / unverifiable hash
//
// We deliberately do NOT make the grading immutable in the Event itself —
// instead we attach a Grade record keyed by event ID. That keeps event hashes
// stable while still letting cross-source validation upgrade or downgrade
// trust as new corroborating events arrive.
package trust

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// pickSeq extracts a monotonic sequence number from common log shapes.
// Returns 0 when no usable sequence is present.
func pickSeq(ev events.Event) int64 {
	for _, k := range []string{"seq", "RecordNumber", "EventRecordID", "msgId", "serial"} {
		if v, ok := ev.Fields[k]; ok && v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

type Grade string

const (
	GradeVerified   Grade = "verified"
	GradeConsistent Grade = "consistent"
	GradeSuspicious Grade = "suspicious"
	GradeUntrusted  Grade = "untrusted"
)

// Anomaly describes why an event was downgraded.
type Anomaly struct {
	Kind   string `json:"kind"` // "future-timestamp" | "stale-timestamp" | "non-monotonic" | "sequence-gap"
	Detail string `json:"detail,omitempty"`
}

// Record is one event's trust assessment.
type Record struct {
	EventID    string     `json:"eventId"`
	Grade      Grade      `json:"grade"`
	Anomalies  []Anomaly  `json:"anomalies,omitempty"`
	CorrobBy   []string   `json:"corroboratedBy,omitempty"` // other event IDs that confirm this one
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// Engine grades events as they flow through the bus. It maintains:
//   - a fingerprint → event-IDs map (for corroboration)
//   - a per-source last-seen-seq + last-seen-ts (for sequence/clock anomalies)
type Engine struct {
	mu           sync.RWMutex
	records      map[string]*Record       // eventID → record
	fingerprints map[string][]string      // fingerprint → event IDs
	srcWatermark map[string]time.Time     // host|source → last-seen timestamp
	seqWatermark map[string]int64         // host|source → last-seen sequence number
	skewLimit    time.Duration
}

func New() *Engine {
	return &Engine{
		records:      map[string]*Record{},
		fingerprints: map[string][]string{},
		srcWatermark: map[string]time.Time{},
		seqWatermark: map[string]int64{},
		skewLimit:    5 * time.Minute,
	}
}

// Observe is called per-event from the bus fan-out and updates the trust
// record for the event (and any others that share its fingerprint).
func (e *Engine) Observe(ev events.Event) {
	rec := &Record{EventID: ev.ID, UpdatedAt: time.Now().UTC()}

	// Hash check is the foundation.
	if !ev.VerifyHash() {
		rec.Anomalies = append(rec.Anomalies, Anomaly{Kind: "hash-broken", Detail: "VerifyHash returned false"})
		rec.Grade = GradeUntrusted
		e.store(rec, "")
		return
	}

	// Timestamp anomalies.
	now := time.Now().UTC()
	if ev.Timestamp.After(now.Add(e.skewLimit)) {
		rec.Anomalies = append(rec.Anomalies, Anomaly{
			Kind: "future-timestamp", Detail: "event ts is more than skewLimit in the future",
		})
	}
	if ev.Timestamp.Before(now.Add(-30 * 24 * time.Hour)) {
		rec.Anomalies = append(rec.Anomalies, Anomaly{Kind: "stale-timestamp"})
	}

	// Per-source monotonicity check.
	srcKey := ev.HostID + "|" + string(ev.Source)
	e.mu.Lock()
	last, seen := e.srcWatermark[srcKey]
	if seen && ev.Timestamp.Before(last.Add(-e.skewLimit)) {
		rec.Anomalies = append(rec.Anomalies, Anomaly{
			Kind: "non-monotonic", Detail: "event timestamp is well behind source's previous high watermark",
		})
	}
	if ev.Timestamp.After(last) {
		e.srcWatermark[srcKey] = ev.Timestamp
	}

	// Sequence-number break detection. Windows EventID (`recordId`), syslog
	// `seq` field, and auditd-like serial numbers all expose a monotonic
	// sequence we can check for missing IDs.
	if seq := pickSeq(ev); seq > 0 {
		seqKey := srcKey
		if prev, ok := e.seqWatermark[seqKey]; ok {
			if seq > prev+1 {
				rec.Anomalies = append(rec.Anomalies, Anomaly{
					Kind:   "sequence-gap",
					Detail: fmt.Sprintf("missing %d sequence numbers between %d and %d", seq-prev-1, prev, seq),
				})
			} else if seq < prev {
				rec.Anomalies = append(rec.Anomalies, Anomaly{
					Kind:   "sequence-rewound",
					Detail: fmt.Sprintf("seq %d arrived after %d (clock or rotation issue?)", seq, prev),
				})
			}
		}
		if seq > e.seqWatermark[seqKey] {
			e.seqWatermark[seqKey] = seq
		}
	}
	e.mu.Unlock()

	// Fingerprint = host + eventType + message (deduped at minute precision).
	// Same event from a different ingest path corroborates.
	fp := fingerprint(ev)
	rec.Grade = baselineGrade(ev, len(rec.Anomalies) > 0)
	e.store(rec, fp)

	// Promote earlier records that share this fingerprint to "consistent".
	//
	// Capped at maxCorrob corroborators per record so a high-cardinality
	// fingerprint (e.g. an alert pattern that fires hundreds of times)
	// doesn't turn this into O(n²) per call. Two corroborators is the
	// meaningful claim ("seen via 2+ paths"); more is informational
	// noise, not stronger evidence.
	const maxCorrob = 5
	e.mu.Lock()
	defer e.mu.Unlock()
	others := e.fingerprints[fp]
	if len(others) >= 2 {
		// Cap the inner promotion to maxCorrob most-recent peers — the
		// outer loop also short-circuits once a record is already at the cap.
		recent := others
		if len(recent) > maxCorrob+1 {
			recent = recent[len(recent)-(maxCorrob+1):]
		}
		for _, id := range recent {
			r, ok := e.records[id]
			if !ok {
				continue
			}
			if r.Grade == GradeUntrusted {
				r.Grade = GradeConsistent
			}
			if len(r.CorrobBy) >= maxCorrob {
				r.UpdatedAt = time.Now().UTC()
				continue
			}
			for _, otherID := range recent {
				if otherID == id {
					continue
				}
				if !contains(r.CorrobBy, otherID) {
					r.CorrobBy = append(r.CorrobBy, otherID)
					if len(r.CorrobBy) >= maxCorrob {
						break
					}
				}
			}
			r.UpdatedAt = time.Now().UTC()
		}
	}
}

func baselineGrade(ev events.Event, hasAnomaly bool) Grade {
	if hasAnomaly {
		return GradeSuspicious
	}
	if ev.Provenance.TLSFingerprint != "" {
		return GradeVerified
	}
	switch ev.Provenance.IngestPath {
	case "agent":
		return GradeVerified
	case "syslog-udp":
		return GradeUntrusted
	case "rest", "rest-batch":
		return GradeUntrusted
	case "raw", "import":
		return GradeUntrusted
	}
	return GradeUntrusted
}

func fingerprint(ev events.Event) string {
	min := ev.Timestamp.UTC().Truncate(time.Minute).Format("20060102T1504")
	return ev.HostID + "|" + ev.EventType + "|" + min + "|" + ev.Message
}

func (e *Engine) store(rec *Record, fp string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.records[rec.EventID] = rec
	if fp != "" {
		e.fingerprints[fp] = append(e.fingerprints[fp], rec.EventID)
	}
}

// Of returns the trust record for an event ID.
func (e *Engine) Of(id string) (*Record, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.records[id]
	if !ok {
		return nil, false
	}
	cc := *r
	return &cc, true
}

// Summary returns counts per grade — used by the dashboard.
type Summary struct {
	Verified   int `json:"verified"`
	Consistent int `json:"consistent"`
	Suspicious int `json:"suspicious"`
	Untrusted  int `json:"untrusted"`
}

func (e *Engine) Summary() Summary {
	e.mu.RLock()
	defer e.mu.RUnlock()
	s := Summary{}
	for _, r := range e.records {
		switch r.Grade {
		case GradeVerified:
			s.Verified++
		case GradeConsistent:
			s.Consistent++
		case GradeSuspicious:
			s.Suspicious++
		case GradeUntrusted:
			s.Untrusted++
		}
	}
	return s
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}
