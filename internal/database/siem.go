package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type SIEMRepository struct {
	db DatabaseStore
}

func NewSIEMRepository(db DatabaseStore) *SIEMRepository {
	return &SIEMRepository{db: db}
}

// InsertHostEvent records a new security anomaly with cryptographic integrity hashing
func (r *SIEMRepository) InsertHostEvent(ctx context.Context, event *HostEvent) error {
	tenantID := MustTenantFromContext(ctx)
	if event.TenantID != "" {
		tenantID = event.TenantID
	}

	// 1.3: Retrieve the cryptographic hash of the most recent event to extend the chain
	var lastHash string
	lastHashRow := r.db.QueryRowContext(ctx, 
		"SELECT event_hash FROM host_events WHERE host_id = ? AND tenant_id = ? ORDER BY id DESC LIMIT 1", 
		event.HostID, tenantID)
	_ = lastHashRow.Scan(&lastHash) // If no previous event, lastHash remains empty string

	event.PrevHash = lastHash

	// 1.3: Calculate the hash for the current event — SHA256(PrevHash + EventData)
	hasher := sha256.New()
	hasher.Write([]byte(event.PrevHash))
	hasher.Write([]byte(event.HostID))
	hasher.Write([]byte(event.EventType))
	hasher.Write([]byte(event.RawLog))
	hasher.Write([]byte(event.Timestamp))
	event.EventHash = hex.EncodeToString(hasher.Sum(nil))

	query := `
		INSERT INTO host_events (tenant_id, host_id, event_type, source_ip, user, raw_log, event_hash, prev_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ReplicatedExecContext(ctx, query,
		tenantID,
		event.HostID,
		event.EventType,
		event.SourceIP,
		event.User,
		event.RawLog,
		event.EventHash,
		event.PrevHash,
	)
	if err != nil {
		return fmt.Errorf("insert host event: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		event.ID = id
	}
	return nil
}

// GetHostEvents returns the latest security events for a host, up to a limit
func (r *SIEMRepository) GetHostEvents(ctx context.Context, hostID string, limit int) ([]HostEvent, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, host_id, timestamp, event_type, source_ip, user, raw_log, event_hash, prev_hash 
		FROM host_events
		WHERE host_id = ? AND tenant_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`
	tenantID := MustTenantFromContext(ctx)
	rows, err := conn.QueryContext(ctx, query, hostID, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("query host events: %w", err)
	}
	defer rows.Close()

	var events []HostEvent
	for rows.Next() {
		var e HostEvent
		if err := rows.Scan(&e.ID, &e.HostID, &e.Timestamp, &e.EventType, &e.SourceIP, &e.User, &e.RawLog, &e.EventHash, &e.PrevHash); err != nil {
			return nil, fmt.Errorf("scan host event: %w", err)
		}
		e.TenantID = tenantID
		events = append(events, e)
	}
	return events, rows.Err()
}

// SearchHostEvents performs a flexible search across security anomalies
func (r *SIEMRepository) SearchHostEvents(ctx context.Context, query string, limit int) ([]HostEvent, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)
	likeQuery := "%" + query + "%"

	whereClause := "tenant_id = ?"
	args := []interface{}{likeQuery, likeQuery, likeQuery, tenantID, limit}
	if tenantID == "" {
		whereClause = "1=1"
		args = []interface{}{likeQuery, likeQuery, likeQuery, limit}
	}

	sqlQuery := fmt.Sprintf(`
		SELECT id, host_id, timestamp, event_type, source_ip, user, raw_log, event_hash, prev_hash 
		FROM host_events
		WHERE (raw_log LIKE ? OR source_ip LIKE ? OR user LIKE ?) AND %s
		ORDER BY timestamp DESC
		LIMIT ?
	`, whereClause)

	rows, err := conn.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search host events: %w", err)
	}
	defer rows.Close()

	var events []HostEvent
	for rows.Next() {
		var e HostEvent
		if err := rows.Scan(&e.ID, &e.HostID, &e.Timestamp, &e.EventType, &e.SourceIP, &e.User, &e.RawLog, &e.EventHash, &e.PrevHash); err != nil {
			return nil, fmt.Errorf("scan search event: %w", err)
		}
		e.TenantID = tenantID
		events = append(events, e)
	}
	return events, rows.Err()
}// GetTimelineEvents retrieves events for a principal within a specific time window for reconstruction.
func (r *SIEMRepository) GetTimelineEvents(ctx context.Context, principalID string, principalType string, startTime string, endTime string) ([]HostEvent, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)
	
	var filter string
	switch principalType {
	case "host":
		filter = "host_id = ?"
	case "user":
		filter = "user = ?"
	case "ip":
		filter = "source_ip = ?"
	default:
		return nil, fmt.Errorf("invalid principal type for timeline: %s", principalType)
	}

	query := fmt.Sprintf(`
		SELECT id, host_id, timestamp, event_type, source_ip, user, raw_log, event_hash, prev_hash 
		FROM host_events
		WHERE %s AND tenant_id = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp ASC
	`, filter)

	rows, err := conn.QueryContext(ctx, query, principalID, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query timeline events: %w", err)
	}
	defer rows.Close()

	var events []HostEvent
	for rows.Next() {
		var e HostEvent
		if err := rows.Scan(&e.ID, &e.HostID, &e.Timestamp, &e.EventType, &e.SourceIP, &e.User, &e.RawLog, &e.EventHash, &e.PrevHash); err != nil {
			return nil, fmt.Errorf("scan timeline event: %w", err)
		}
		e.TenantID = tenantID
		events = append(events, e)
	}
	return events, rows.Err()
}

// GetFailedLoginsByHost aggregates invalid login counts per source IP
func (r *SIEMRepository) GetFailedLoginsByHost(ctx context.Context, hostID string) ([]map[string]interface{}, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, fmt.Errorf("query failed logins: %w", err)
	}

	tenantID := MustTenantFromContext(ctx)

	query := `
		SELECT source_ip, user, COUNT(id) as attempts, MAX(timestamp) as last_attempt
		FROM host_events
		WHERE host_id = ? AND event_type = 'failed_login' AND tenant_id = ?
		GROUP BY source_ip, user
		ORDER BY attempts DESC
	`
	rows, err := conn.QueryContext(ctx, query, hostID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query failed logins: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var ip, user, lastAttempt string
		var attempts int
		if err := rows.Scan(&ip, &user, &attempts, &lastAttempt); err != nil {
			return nil, fmt.Errorf("scan failed login diff: %w", err)
		}
		results = append(results, map[string]interface{}{
			"source_ip":    ip,
			"user":         user,
			"attempts":     attempts,
			"last_attempt": lastAttempt,
		})
	}
	return results, rows.Err()
}

// CalculateRiskScore computes a dynamic 0-100 severity score for a host based on failed login patterns
func (r *SIEMRepository) CalculateRiskScore(ctx context.Context, hostID string) (int, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return 0, fmt.Errorf("query risk stats: %w", err)
	}

	tenantID := MustTenantFromContext(ctx)

	// 1. Get failed login stats for the last 24h (or the available window)
	query := `
		SELECT source_ip, user, COUNT(id) as attempts
		FROM host_events
		WHERE host_id = ? AND event_type = 'failed_login' AND tenant_id = ?
		GROUP BY source_ip, user
	`
	rows, err := conn.QueryContext(ctx, query, hostID, tenantID)
	if err != nil {
		return 0, fmt.Errorf("query risk stats: %w", err)
	}
	defer rows.Close()

	score := 0
	uniqueIPs := make(map[string]bool)
	totalAttempts := 0
	targetedRoot := false

	for rows.Next() {
		var ip, user string
		var attempts int
		if err := rows.Scan(&ip, &user, &attempts); err != nil {
			return 0, fmt.Errorf("scan risk stats: %w", err)
		}

		uniqueIPs[ip] = true
		totalAttempts += attempts
		if user == "root" {
			targetedRoot = true
		}
	}

	// Algorithm:
	// - Base: 1 point per 5 attempts (capped at 40)
	score += (totalAttempts / 5)
	if score > 40 {
		score = 40
	}

	// - Root targeting penalty: +20 points
	if targetedRoot {
		score += 20
	}

	// - IP diversity penalty: +10 points per unique attacker IP (capped at 40)
	ipPenalty := len(uniqueIPs) * 10
	if ipPenalty > 40 {
		ipPenalty = 40
	}
	score += ipPenalty

	// Final clamp
	if score > 100 {
		score = 100
	}

	return score, nil
}

// GetGlobalThreatStats aggregates security data across all hosts for the Dashboard KPIs
func (r *SIEMRepository) GetGlobalThreatStats(ctx context.Context) (map[string]interface{}, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	tenantID := MustTenantFromContext(ctx)

	// 1. Total failed logins
	var totalFailed int
	err = conn.QueryRow("SELECT COUNT(*) FROM host_events WHERE event_type = 'failed_login' AND tenant_id = ?", tenantID).Scan(&totalFailed)
	if err != nil {
		return nil, err
	}
	stats["total_failed_logins"] = totalFailed

	// 2. Total unique attacker IPs
	var uniqueIPs int
	err = conn.QueryRow("SELECT COUNT(DISTINCT source_ip) FROM host_events WHERE event_type = 'failed_login' AND tenant_id = ?", tenantID).Scan(&uniqueIPs)
	if err != nil {
		return nil, err
	}
	stats["unique_attacker_ips"] = uniqueIPs

	// 3. High risk hosts count
	// We count hosts with > 10 failed logins (simplified risk metric for global stat)
	var highRiskHosts int
	err = conn.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT host_id FROM host_events 
			WHERE event_type = 'failed_login' AND tenant_id = ?
			GROUP BY host_id 
			HAVING COUNT(id) > 10
		)
	`, tenantID).Scan(&highRiskHosts)
	if err != nil {
		return nil, err
	}
	stats["high_risk_hosts"] = highRiskHosts

	return stats, nil
}

// GetEventTrend returns a day-by-day count of security events for the last N days
func (r *SIEMRepository) GetEventTrend(ctx context.Context, days int) ([]map[string]interface{}, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)

	query := `
		SELECT date(timestamp) as day, COUNT(*) as count
		FROM host_events
		WHERE timestamp > date('now', ?) AND tenant_id = ?
		GROUP BY day
		ORDER BY day ASC
	`
	rows, err := conn.QueryContext(ctx, query, fmt.Sprintf("-%d days", days), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trend []map[string]interface{}
	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		trend = append(trend, map[string]interface{}{
			"date":  day,
			"count": count,
		})
	}
	return trend, rows.Err()
}

// AggregateHostEvents groups host events by a specific field (e.g., event_type, source_ip)
func (r *SIEMRepository) AggregateHostEvents(ctx context.Context, query string, facetField string) (map[string]int, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	validFields := map[string]string{
		"event_type": "event_type",
		"host_id":    "host_id",
		"source_ip":  "source_ip",
		"user":       "user",
	}

	sanitizedField, ok := validFields[facetField]
	if !ok {
		return nil, fmt.Errorf("invalid facet field: %s", facetField)
	}

	tenantID := MustTenantFromContext(ctx)
	whereParts := []string{"tenant_id = ?"}
	args := []interface{}{tenantID}

	if query != "" {
		whereParts = append(whereParts, "(raw_log LIKE ? OR source_ip LIKE ? OR user LIKE ?)")
		likeQuery := "%" + query + "%"
		args = append(args, likeQuery, likeQuery, likeQuery)
	}

	whereClause := "WHERE " + strings.Join(whereParts, " AND ")

	sqlQuery := fmt.Sprintf(`
		SELECT %s, COUNT(id) as count 
		FROM host_events 
		%s
		GROUP BY %s
		ORDER BY count DESC
		LIMIT 15
	`, sanitizedField, whereClause, sanitizedField)

	rows, err := conn.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregate host events: %w", err)
	}
	defer rows.Close()

	results := make(map[string]int)
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			continue // Skip nulls
		}
		if key != "" {
			results[key] = count
		}
	}
	return results, rows.Err()
}

// CreateSavedSearch stores a reusable SIEM query
func (r *SIEMRepository) CreateSavedSearch(ctx context.Context, search *SavedSearch) error {
	tenantID := MustTenantFromContext(ctx)
	query := `INSERT INTO saved_searches (tenant_id, name, query) VALUES (?, ?, ?)`
	result, err := r.db.ReplicatedExecContext(ctx, query, tenantID, search.Name, search.Query)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		search.ID = fmt.Sprintf("%d", id)
	}
	return nil
}

// GetSavedSearches retrieves all reusable SIEM queries
func (r *SIEMRepository) GetSavedSearches(ctx context.Context) ([]SavedSearch, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)
	query := `SELECT id, name, query, created_at FROM saved_searches WHERE tenant_id = ? ORDER BY name ASC`
	rows, err := conn.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var searches []SavedSearch
	for rows.Next() {
		var s SavedSearch
		if err := rows.Scan(&s.ID, &s.Name, &s.Query, &s.CreatedAt); err != nil {
			return nil, err
		}
		searches = append(searches, s)
	}
	return searches, rows.Err()
}
