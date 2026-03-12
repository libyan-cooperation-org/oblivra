package storage

import (
	"bytes"
	"context" // Added context import
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/search"
)

// BadgerSIEMRepository implements database.SIEMStore using the ultra-fast BadgerDB hot store.
// Replaces the SQLite-based SIEMRepository.
type BadgerSIEMRepository struct {
	store  *HotStore
	search **search.SearchEngine
	db     database.DatabaseStore
}

// NewBadgerSIEMRepository creates a new SIEM repository backed by BadgerDB.
func NewBadgerSIEMRepository(store *HotStore, searchEngine **search.SearchEngine, db database.DatabaseStore) *BadgerSIEMRepository {
	return &BadgerSIEMRepository{
		store:  store,
		search: searchEngine,
		db:     db,
	}
}

// Keys:
// Primary: event:{host_id}:{timestamp_ns}:{event_id} -> JSON(HostEvent)
// IP Index: idx:ip:{source_ip}:{timestamp_ns}:{event_id} -> JSON(HostEvent)

func formatEventKey(hostID string, ts time.Time, id int64) []byte {
	return []byte(fmt.Sprintf("event:%s:%020d:%d", hostID, ts.UnixNano(), id))
}

func formatIPKey(ip string, ts time.Time, id int64) []byte {
	return []byte(fmt.Sprintf("idx:ip:%s:%020d:%d", ip, ts.UnixNano(), id))
}

// InsertHostEvent records a new security anomaly.
// Writes to Badger with a 30-day TTL (since events are archived or age out anyway).
func (r *BadgerSIEMRepository) InsertHostEvent(ctx context.Context, event *database.HostEvent) error {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().Format(time.RFC3339)
	}

	// We use UnixNano as a pseudo-ID since Badger is KV.
	if event.ID == 0 {
		event.ID = time.Now().UnixNano()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	ttl := 30 * 24 * time.Hour

	// 1. Primary write
	ts, _ := time.Parse(time.RFC3339, event.Timestamp)
	primaryKey := formatEventKey(event.HostID, ts, event.ID)
	if err := r.store.Put(primaryKey, data, ttl); err != nil {
		return fmt.Errorf("failed to write primary event: %w", err)
	}

	// 2. Secondary index for IP lookups (useful for risk scoring)
	if event.SourceIP != "" && event.SourceIP != "-" {
	ts, _ := time.Parse(time.RFC3339, event.Timestamp)
		ipKey := formatIPKey(event.SourceIP, ts, event.ID)
		if err := r.store.Put(ipKey, data, ttl); err != nil {
			return fmt.Errorf("failed to write IP index: %w", err)
		}
	}

	// 3. Dual-write to Bleve for full-text search
	if r.search != nil && *r.search != nil {
		docID := fmt.Sprintf("event_%s_%d", event.HostID, event.ID)
		searchData := map[string]interface{}{
			"host":       event.HostID,
			"source_ip":  event.SourceIP,
			"event_type": event.EventType,
			"user":       event.User,
			"output":     event.RawLog, // Maps to Bleve's text analyzer
			"timestamp":  ts.UnixNano(),
		}
		// We ignore error here so ingestion doesn't fail if Bleve is temporarily locked/busy
		(*r.search).Index(docID, "siem_event", searchData)
	}

	return nil
}

// GetHostEvents returns the latest security events for a host, up to a limit
func (r *BadgerSIEMRepository) GetHostEvents(ctx context.Context, hostID string, limit int) ([]database.HostEvent, error) {
	prefix := []byte(fmt.Sprintf("event:%s:", hostID))
	var events []database.HostEvent

	err := r.store.ReverseIteratePrefix(prefix, limit, func(key, value []byte) error {
		var e database.HostEvent
		if err := json.Unmarshal(value, &e); err != nil {
			return err
		}
		events = append(events, e)
		return nil
	})

	return events, err
}

// SearchHostEvents performs a flexible search across security anomalies
func (r *BadgerSIEMRepository) SearchHostEvents(ctx context.Context, query string, limit int) ([]database.HostEvent, error) {
	if r.search != nil && *r.search != nil {
		// Attempt to hit Bleve first for actual full-text performance
		results, err := (*r.search).Search(query, limit, 0)
		if err == nil {
			var events []database.HostEvent
			for _, res := range results {
				// We reconstruct HostEvent directly from Bleve data to avoid N+1 DB lookups
				var e database.HostEvent
				if host, ok := res.Data["host"].(string); ok {
					e.HostID = host
				}
				if ip, ok := res.Data["source_ip"].(string); ok {
					e.SourceIP = ip
				}
				if u, ok := res.Data["user"].(string); ok {
					e.User = u
				}
				if out, ok := res.Data["output"].(string); ok {
					e.RawLog = out
				}
				if typ, ok := res.Data["event_type"].(string); ok {
					e.EventType = typ
				}

				// Timestamp is indexed as float64 by bleve
				if tsFloat, ok := res.Data["timestamp"].(float64); ok {
					e.Timestamp = time.Unix(0, int64(tsFloat)).Format(time.RFC3339)
				}

				events = append(events, e)
			}
			return events, nil
		}
	}

	// Fallback to slow BadgerDB prefix scan if Bleve is offline or locked
	prefix := []byte("event:")
	var events []database.HostEvent
	qLower := strings.ToLower(query)

	err := r.store.ReverseIteratePrefix(prefix, 0, func(key, value []byte) error {
		if limit > 0 && len(events) >= limit {
			return nil
		}

		if bytes.Contains(bytes.ToLower(value), []byte(qLower)) {
			var e database.HostEvent
			if err := json.Unmarshal(value, &e); err != nil {
				return nil
			}
			if strings.Contains(strings.ToLower(e.RawLog), qLower) ||
				strings.Contains(strings.ToLower(e.SourceIP), qLower) ||
				strings.Contains(strings.ToLower(e.User), qLower) {
				events = append(events, e)
			}
		}
		return nil
	})

	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}

	return events, err
}

// GetFailedLoginsByHost aggregates invalid login counts per source IP
func (r *BadgerSIEMRepository) GetFailedLoginsByHost(ctx context.Context, hostID string) ([]map[string]interface{}, error) {
	prefix := []byte(fmt.Sprintf("event:%s:", hostID))

	// ip -> user -> {attempts, last_attempt}
	type stats struct {
		attempts    int
		lastAttempt string
	}

	agg := make(map[string]map[string]*stats)

	err := r.store.ReverseIteratePrefix(prefix, 10000, func(key, value []byte) error {
		// Only care about failed_logins. Quick byte check for speed.
		if !bytes.Contains(value, []byte(`"event_type":"failed_login"`)) {
			return nil
		}

		var e database.HostEvent
		if err := json.Unmarshal(value, &e); err != nil {
			return nil
		}

		if e.EventType == "failed_login" {
			if agg[e.SourceIP] == nil {
				agg[e.SourceIP] = make(map[string]*stats)
			}
			if agg[e.SourceIP][e.User] == nil {
				agg[e.SourceIP][e.User] = &stats{
					attempts:    0,
					lastAttempt: e.Timestamp,
				}
			}
			s := agg[e.SourceIP][e.User]
			s.attempts++
			// since we iterateReverse (newest first), the first time we see an IP/User combo
			// is the most recent attempt.
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for ip, users := range agg {
		for user, st := range users {
			results = append(results, map[string]interface{}{
				"source_ip":    ip,
				"user":         user,
				"attempts":     st.attempts,
				"last_attempt": st.lastAttempt,
			})
		}
	}

	return results, nil
}

// CalculateRiskScore heuristically calculates risk of a host based on event frequency over 24h
func (r *BadgerSIEMRepository) CalculateRiskScore(ctx context.Context, hostID string) (int, error) {
	prefix := []byte(fmt.Sprintf("event:%s:", hostID))

	score := 0
	uniqueIPs := make(map[string]bool)
	totalAttempts := 0
	targetedRoot := false

	// Scan last 5000 events for this host
	r.store.ReverseIteratePrefix(prefix, 5000, func(key, value []byte) error {
		if !bytes.Contains(value, []byte(`"event_type":"failed_login"`)) {
			return nil
		}

		var e database.HostEvent
		if err := json.Unmarshal(value, &e); err != nil {
			return nil
		}

		if e.EventType == "failed_login" {
			uniqueIPs[e.SourceIP] = true
			totalAttempts++
			if e.User == "root" {
				targetedRoot = true
			}
		}
		return nil
	})

	// Algorithm matches sqlite version
	score += (totalAttempts / 5)
	if score > 40 {
		score = 40
	}
	if targetedRoot {
		score += 20
	}
	ipPenalty := len(uniqueIPs) * 10
	if ipPenalty > 40 {
		ipPenalty = 40
	}
	score += ipPenalty

	if score > 100 {
		score = 100
	}

	return score, nil
}

// GetGlobalThreatStats returns total counts of security events for the dashboard
func (r *BadgerSIEMRepository) GetGlobalThreatStats(ctx context.Context) (map[string]interface{}, error) {
	return nil, errors.New("not implemented in badger cache")
}

// GetEventTrend aggregates event counts over the last given days
func (r *BadgerSIEMRepository) GetEventTrend(ctx context.Context, days int) ([]map[string]interface{}, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	trendMap := make(map[string]int)

	// Pre-fill days with 0
	for i := 0; i <= days; i++ {
		d := cutoff.AddDate(0, 0, i).Format("2006-01-02")
		trendMap[d] = 0
	}

	prefix := []byte("event:")
	r.store.ReverseIteratePrefix(prefix, 0, func(key, value []byte) error {
		// Key format: event:{host_id}:{timestamp_ns}:{id}
		parts := bytes.Split(key, []byte(":"))
		if len(parts) >= 4 {
			tsNano, err := strconv.ParseInt(string(parts[2]), 10, 64)
			if err == nil {
				ts := time.Unix(0, tsNano)
				if ts.Before(cutoff) {
					return fmt.Errorf("cutoff_reached") // Stop iteration
				}

				dayStr := ts.Format("2006-01-02")
				trendMap[dayStr]++
			}
		}
		return nil
	})

	var trend []map[string]interface{}
	// Ordered output
	for i := 0; i <= days; i++ {
		d := cutoff.AddDate(0, 0, i).Format("2006-01-02")
		trend = append(trend, map[string]interface{}{
			"date":  d,
			"count": trendMap[d],
		})
	}

	return trend, nil
}

// AggregateHostEvents is not supported in the badger cache
func (r *BadgerSIEMRepository) AggregateHostEvents(ctx context.Context, query string, facetField string) (map[string]int, error) {
	return nil, errors.New("not implemented in badger cache")
}

// CreateSavedSearch persists a new SIEM query pattern to the global SQL DB
func (r *BadgerSIEMRepository) CreateSavedSearch(ctx context.Context, search *database.SavedSearch) error {
	return errors.New("not implemented in badger cache")
}

// GetSavedSearches retrieves all persisted SIEM queries from the global SQL DB
func (r *BadgerSIEMRepository) GetSavedSearches(ctx context.Context) ([]database.SavedSearch, error) {
	return nil, errors.New("not implemented in badger cache")
}
