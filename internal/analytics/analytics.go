package analytics

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/search"
)

// LogEntry represents a single terminal log record
type LogEntry struct {
	Timestamp string
	TenantID  string
	SessionID string
	Host      string
	Output    string
}

// TtyFrame represents an Asciinema v2 frame
type TtyFrame struct {
	RecordingID string
	Timestamp   float64
	Type        string
	Data        string // Compressed base64
}

// AnalyticsEngine provides persistent log storage and querying via SQLite
type AnalyticsEngine struct {
	db            *sql.DB
	ingestCh      chan LogEntry
	frameIngestCh chan TtyFrame
	done          chan struct{}
	mu            sync.RWMutex
	transpiler    *Transpiler
	archiver      *Archiver
	searchEngine  *search.SearchEngine
	log           *logger.Logger
	opened        bool
	cancelWorkers context.CancelFunc
	workerWg      sync.WaitGroup
}

const (
	MaxLogRows    = 500000
	RetentionDays = 7
	MaxAlertRows  = 5000
)

// backgroundWriter batches incoming log entries and writes them in bulk
func (e *AnalyticsEngine) backgroundWriter(ctx context.Context) {
	defer e.workerWg.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var batch []LogEntry

	flush := func() {
		if len(batch) == 0 {
			return
		}

		tx, err := e.db.Begin()
		if err != nil {
			return
		}

		stmt, err := tx.Prepare("INSERT INTO terminal_logs (timestamp, tenant_id, session_id, host, output) VALUES (?, ?, ?, ?, ?)")
		if err != nil {
			tx.Rollback()
			return
		}

		searchBatch := make(map[string]interface{})

		for _, entry := range batch {
			stmt.Exec(entry.Timestamp, entry.TenantID, entry.SessionID, entry.Host, entry.Output)

			// Buffer for Bleve batching
			if e.searchEngine != nil {
				docID := uuid.New().String()
				searchBatch[docID] = map[string]interface{}{
					"timestamp":  entry.Timestamp,
					"session_id": entry.SessionID,
					"host":       entry.Host,
					"output":     entry.Output,
				}
			}
		}
		stmt.Close()
		tx.Commit()

		// High-performance search indexing
		if len(searchBatch) > 0 {
			if err := e.searchEngine.BatchIndex("global", searchBatch, "terminal_log"); err != nil {
				e.log.Error("[ANALYTICS] Bleve batch index failed: %v", err)
			}
		}

		batch = batch[:0]
	}

	for {
		select {
		case entry := <-e.ingestCh:
			batch = append(batch, entry)
			if len(batch) >= 1000 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			// Final flush if possible
			flush()
			return
		case <-e.done: // Global shutdown
			return
		}
	}
}

// backgroundFrameWriter batches TTY frames and writes them in bulk
func (e *AnalyticsEngine) backgroundFrameWriter(ctx context.Context) {
	defer e.workerWg.Done()
	ticker := time.NewTicker(500 * time.Millisecond) // Faster ticker for frames
	defer ticker.Stop()

	var batch []TtyFrame

	flush := func() {
		if len(batch) == 0 {
			return
		}

		tx, err := e.db.Begin()
		if err != nil {
			return
		}

		stmt, err := tx.Prepare("INSERT INTO recording_frames (recording_id, timestamp, type, data) VALUES (?, ?, ?, ?)")
		if err != nil {
			tx.Rollback()
			return
		}

		ftsStmt, err := tx.Prepare("INSERT INTO recording_frames_fts (recording_id, data) VALUES (?, ?)")
		if err != nil {
			stmt.Close()
			tx.Rollback()
			return
		}

		for _, frame := range batch {
			// COMPRESSION OFFLOAD: Compress in the background worker instead of caller goroutine
			compressedData := compressString(frame.Data)

			stmt.Exec(frame.RecordingID, frame.Timestamp, frame.Type, compressedData)
			// Only index output and input for FTS, and strip ANSI
			if frame.Type == "o" || frame.Type == "i" {
				ftsStmt.Exec(frame.RecordingID, stripAnsi(frame.Data))
			}
		}
		stmt.Close()
		ftsStmt.Close()
		tx.Commit()
		batch = batch[:0]
	}

	for {
		select {
		case frame := <-e.frameIngestCh:
			batch = append(batch, frame)
			if len(batch) >= 500 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		case <-e.done:
			return
		}
	}
}

// Ingest queues a log entry for async batch writing
func (e *AnalyticsEngine) Ingest(ctx context.Context, sessionID, host, output string) {
	tenantID := database.MustTenantFromContext(ctx)
	e.mu.RLock()
	if !e.opened {
		e.mu.RUnlock()
		return
	}
	e.mu.RUnlock()

	select {
	case e.ingestCh <- LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		TenantID:  tenantID,
		SessionID: sessionID,
		Host:      host,
		Output:    output,
	}:
	default:
		// Channel full, drop entry to avoid blocking SSH output
		e.log.Warn("Analytics buffer full, dropped terminal log for: %s", sessionID)
	}
}

// Search executes a query against the analytics store.
// Uses Bleve for 'lucene' mode if available, otherwise falls back to SQL mapping.
func (e *AnalyticsEngine) Search(ctx context.Context, rawQuery string, mode string, limit, offset int) ([]map[string]interface{}, error) {
	tenantID := database.MustTenantFromContext(ctx)
	if e.db == nil || !e.opened {
		return nil, fmt.Errorf("analytics engine not opened")
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 1. Try Bleve fast-path for lucene queries
	if mode == "lucene" && e.searchEngine != nil {
		bleveResults, err := e.searchEngine.Search(tenantID, rawQuery, limit, offset)
		if err == nil {
			var formatResults []map[string]interface{}
			for _, hit := range bleveResults {
				formatResults = append(formatResults, map[string]interface{}{
					"id":         hit.ID,
					"timestamp":  hit.Data["timestamp"],
					"session_id": hit.Data["session_id"],
					"host":       hit.Data["host"],
					"output":     hit.Data["output"],
					"_score":     hit.Score, // Expose search score for frontend relevance
				})
			}
			return formatResults, nil
		}
		e.log.Warn("Bleve search failed, falling back to SQLite full-text scan: %v", err)
	}

	// 2. Fallback to SQL mapping (for logql, sql, or if Bleve fails)
	var sqlQuery string
	var args []interface{}
	var err error

	switch mode {
	case "logql":
		sqlQuery, args, err = e.transpiler.ConvertLogQLToSQL(rawQuery, limit, offset)
	case "lucene":
		sqlQuery, args, err = e.transpiler.ConvertLuceneToSQL(rawQuery, limit, offset)
	case "sql":
		sqlQuery = rawQuery
	default:
		sqlQuery, args, err = e.transpiler.ConvertLogQLToSQL(rawQuery, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("transpile query: %w", err)
	}

	// Structural Tenant Isolation: Append WHERE clause if not present, or inject into args.
	// For raw SQL, we must ensure tenant_id matches.
	if mode == "sql" {
		sqlQuery = fmt.Sprintf("SELECT * FROM (%s) WHERE tenant_id = ?", sqlQuery)
		args = append(args, tenantID)
	}

	rows, err := e.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}

// SetSearchEngine injects the Bleve full-text search engine for dual-writing
func (e *AnalyticsEngine) SetSearchEngine(se *search.SearchEngine) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.searchEngine = se
}

// Close flushes pending writes and closes the database
func (e *AnalyticsEngine) Close() error {
	close(e.done)
	if e.cancelWorkers != nil {
		e.cancelWorkers()
		e.workerWg.Wait()
	}
	if e.archiver != nil {
		e.archiver.Stop()
	}
	if e.db != nil {
		return e.db.Close()
	}
	return nil
}

// retentionLoop prunes old logs periodically
func (e *AnalyticsEngine) retentionLoop(ctx context.Context) {
	defer e.workerWg.Done()
	// Run once on startup
	e.runInitialRetention()

	ticker := time.NewTicker(time.Hour * 24)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.done:
			return
		case <-ticker.C:
			e.mu.Lock()
			// Delete logs older than 7 days
			res, err := e.db.Exec(`DELETE FROM terminal_logs WHERE timestamp < datetime('now', '-7 days')`)
			if err == nil {
				rows, _ := res.RowsAffected()
				if rows > 0 {
					e.db.Exec(`VACUUM`)
				}
			}
			e.mu.Unlock()
		}
	}
}

// runInitialRetention executes the retention pruning once asynchronously
func (e *AnalyticsEngine) runInitialRetention() {
	e.mu.Lock()
	defer e.mu.Unlock()
	res, err := e.db.Exec(`DELETE FROM terminal_logs WHERE timestamp < datetime('now', '-7 days')`)
	if err == nil {
		rows, _ := res.RowsAffected()
		if rows > 0 {
			e.db.Exec(`VACUUM`)
		}
	}
}

// SaveConfig stores a JSON blob under a key in app_config
func (e *AnalyticsEngine) SaveConfig(ctx context.Context, key string, value string) error {
	tenantID := database.MustTenantFromContext(ctx)
	if e.db == nil || !e.opened {
		return fmt.Errorf("analytics engine not opened")
	}
	_, err := e.db.Exec(
		"INSERT INTO app_config (key, tenant_id, value, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP) ON CONFLICT(key, tenant_id) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at",
		key, tenantID, value)
	return err
}

// LoadConfig retrieves a JSON blob by key from app_config
func (e *AnalyticsEngine) LoadConfig(ctx context.Context, key string) (string, error) {
	tenantID := database.MustTenantFromContext(ctx)
	if e.db == nil || !e.opened {
		return "", fmt.Errorf("analytics engine not opened")
	}
	var value string
	err := e.db.QueryRow("SELECT value FROM app_config WHERE key = ? AND tenant_id = ?", key, tenantID).Scan(&value)
	return value, err
}

// SaveAlertEvent writes an alert event to the alert_history table
func (e *AnalyticsEngine) SaveAlertEvent(ctx context.Context, triggerID, name, severity, host, sessionID, logLine string, sent bool) {
	tenantID := database.MustTenantFromContext(ctx)
	if e.db == nil || !e.opened {
		return
	}
	sentInt := 0
	if sent {
		sentInt = 1
	}
	e.db.Exec(
		"INSERT INTO alert_history (tenant_id, trigger_id, name, severity, host, session_id, log_line, sent) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		tenantID, triggerID, name, severity, host, sessionID, logLine, sentInt)
}

// GetAlertHistory returns the last N alert events from the database
func (e *AnalyticsEngine) GetAlertHistory(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	tenantID := database.MustTenantFromContext(ctx)
	if e.db == nil || !e.opened {
		return nil, fmt.Errorf("analytics engine not opened")
	}
	rows, err := e.db.Query(
		"SELECT timestamp, trigger_id, name, severity, host, session_id, log_line, sent FROM alert_history WHERE tenant_id = ? ORDER BY timestamp DESC LIMIT ?",
		tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var ts, triggerID, name, severity, host, sessionID, logLine string
		var sent int
		if err := rows.Scan(&ts, &triggerID, &name, &severity, &host, &sessionID, &logLine, &sent); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"timestamp":  ts,
			"trigger_id": triggerID,
			"name":       name,
			"severity":   severity,
			"host":       host,
			"session_id": sessionID,
			"log_line":   logLine,
			"sent":       sent == 1,
		})
	}
	return results, nil
}

// IngestFrame queues a TTY frame for async batch writing
func (e *AnalyticsEngine) IngestFrame(ctx context.Context, recordingID string, timestamp float64, frameType string, data string) {
	e.mu.RLock()
	if !e.opened {
		e.mu.RUnlock()
		return
	}
	e.mu.RUnlock()

	select {
	case e.frameIngestCh <- TtyFrame{
		RecordingID: recordingID,
		Timestamp:   timestamp,
		Type:        frameType,
		Data:        data, // We send raw data and compress in worker
	}:
	default:
		// Channel full, drop frame to avoid blocking session
		e.log.Warn("Analytics buffer full, dropped TTY frame for: %s", recordingID)
	}
}

// SaveRecording stores recording metadata
func (e *AnalyticsEngine) SaveRecording(ctx context.Context, id, sessionID, hostLabel string, cols, rows int, duration float64, eventCount int, status string) error {
	tenantID := database.MustTenantFromContext(ctx)
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.opened {
		return fmt.Errorf("analytics engine not opened")
	}

	_, err := e.db.Exec(`
		INSERT INTO session_recordings (id, tenant_id, session_id, host_label, duration, event_count, cols, rows, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET 
			duration = excluded.duration, 
			event_count = excluded.event_count,
			status = excluded.status
	`, id, tenantID, sessionID, hostLabel, duration, eventCount, cols, rows, status)
	return err
}

// GetRecordingMeta retrieves metadata for a specific recording
func (e *AnalyticsEngine) GetRecordingMeta(ctx context.Context, id string) (map[string]interface{}, error) {
	tenantID := database.MustTenantFromContext(ctx)
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.opened {
		return nil, fmt.Errorf("analytics engine not opened")
	}

	row := e.db.QueryRow(`SELECT id, session_id, host_label, started_at, duration, event_count, cols, rows, status FROM session_recordings WHERE id = ? AND tenant_id = ?`, id, tenantID)
	var rid, sid, host, started, status string
	var dur float64
	var count, cols, rows int
	if err := row.Scan(&rid, &sid, &host, &started, &dur, &count, &cols, &rows, &status); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":          rid,
		"session_id":  sid,
		"host_label":  host,
		"started_at":  started,
		"duration":    dur,
		"event_count": count,
		"cols":        cols,
		"rows":        rows,
		"status":      status,
	}, nil
}

// ListRecordings returns all recorded sessions
func (e *AnalyticsEngine) ListRecordings(ctx context.Context) ([]map[string]interface{}, error) {
	tenantID := database.MustTenantFromContext(ctx)
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.opened {
		return nil, fmt.Errorf("analytics engine not opened")
	}

	rows, err := e.db.Query(`SELECT id, host_label, started_at, duration, event_count, status FROM session_recordings WHERE tenant_id = ? ORDER BY started_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var id, host, started, status string
		var dur float64
		var count int
		if err := rows.Scan(&id, &host, &started, &dur, &count, &status); err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"id":          id,
			"host_label":  host,
			"started_at":  started,
			"duration":    dur,
			"event_count": count,
			"status":      status,
		})
	}
	return result, nil
}

// DeleteRecording removes a recording and all its frames
func (e *AnalyticsEngine) DeleteRecording(ctx context.Context, id string) error {
	tenantID := database.MustTenantFromContext(ctx)
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.opened {
		return fmt.Errorf("analytics engine not opened")
	}

	_, err := e.db.Exec(`DELETE FROM session_recordings WHERE id = ? AND tenant_id = ?`, id, tenantID)
	return err
}

// GetRecordingFrames returns all frames for a recording ordered by timestamp
func (e *AnalyticsEngine) GetRecordingFrames(ctx context.Context, recordingID string) ([]map[string]interface{}, error) {
	tenantID := database.MustTenantFromContext(ctx)
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.opened {
		return nil, fmt.Errorf("analytics engine not opened")
	}

	// Double check tenant owns recording
	var exists int
	err := e.db.QueryRow("SELECT COUNT(*) FROM session_recordings WHERE id = ? AND tenant_id = ?", recordingID, tenantID).Scan(&exists)
	if err != nil || exists == 0 {
		return nil, fmt.Errorf("access denied or recording not found")
	}

	rows, err := e.db.Query(`SELECT timestamp, type, data FROM recording_frames WHERE recording_id = ? ORDER BY timestamp ASC`, recordingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var ts float64
		var t, d string
		if err := rows.Scan(&ts, &t, &d); err != nil {
			continue
		}

		// Decompress data
		decompressed := decompressString(d)

		result = append(result, map[string]interface{}{
			"timestamp": ts,
			"type":      t,
			"data":      decompressed,
		})
	}
	return result, nil
}

// SearchRecordings executes a forensic search across all sessions using FTS5
func (e *AnalyticsEngine) SearchRecordings(ctx context.Context, query string) ([]map[string]interface{}, error) {
	tenantID := database.MustTenantFromContext(ctx)
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.opened {
		return nil, fmt.Errorf("analytics engine not opened")
	}

	// Search the FTS5 table which contains decompressed command snippets
	sqlQuery := `
		SELECT r.id, r.host_label, r.started_at, snippet(recording_frames_fts, 0, '<b>', '</b>', '...', 10) as highlight
		FROM recording_frames_fts f
		JOIN session_recordings r ON f.recording_id = r.id
		WHERE recording_frames_fts MATCH ? AND r.tenant_id = ?
		GROUP BY r.id
		ORDER BY r.started_at DESC
		LIMIT 100
	`
	rows, err := e.db.Query(sqlQuery, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, host, started, highlight string
		if err := rows.Scan(&id, &host, &started, &highlight); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id":         id,
			"host_label": host,
			"started_at": started,
			"highlight":  highlight,
		})
	}
	return results, nil
}

// Helpers for compression
func compressString(s string) string {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	zw.Write([]byte(s))
	zw.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func decompressString(s string) string {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s // Fallback to raw if not base64 or not compressed
	}

	zr, err := zlib.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return s // Fallback to raw
	}
	defer zr.Close()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		return s
	}
	return string(decompressed)
}

func stripAnsi(str string) string {
	// Basic ANSI escape code stripper
	var b bytes.Buffer
	inEscape := false
	for i := 0; i < len(str); i++ {
		if str[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (str[i] >= 'A' && str[i] <= 'Z') || (str[i] >= 'a' && str[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		b.WriteByte(str[i])
	}
	return b.String()
}
