//go:build !sqlcipher
// +build !sqlcipher

package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	_ "modernc.org/sqlite"
)

// NewAnalyticsEngine initializes the analytics engine structure.
// Actual database opening happens via the Open method.
func NewAnalyticsEngine(log *logger.Logger) *AnalyticsEngine {
	return &AnalyticsEngine{
		ingestCh:      make(chan LogEntry, 100000),
		frameIngestCh: make(chan TtyFrame, 50000),
		done:          make(chan struct{}),
		transpiler:    NewTranspiler(),
		log:           log,
	}
}

// Open initializes the underlying SQLite database.
func (e *AnalyticsEngine) Open(dbPath string, encryptionKey []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.opened && e.db != nil {
		if e.cancelWorkers != nil {
			e.cancelWorkers()
			e.workerWg.Wait()
		}
		if e.archiver != nil {
			e.archiver.Stop()
		}
		e.db.Close()
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return fmt.Errorf("create analytics dir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("open analytics db: %w", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	// Create tables
	if err := e.bootstrap(db); err != nil {
		db.Close()
		return err
	}

	e.db = db
	e.opened = true

	// Setup worker context
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelWorkers = cancel

	e.archiver = NewArchiver(e.db, filepath.Dir(dbPath), 30*24*time.Hour, e.log)
	go e.archiver.Start()

	e.workerWg.Add(3)
	go e.backgroundWriter(ctx)
	go e.backgroundFrameWriter(ctx)
	go e.retentionLoop(ctx)

	return nil
}

func (e *AnalyticsEngine) bootstrap(db *sql.DB) error {
	// Create the logs table if it doesn't exist
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS terminal_logs (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp  DATETIME DEFAULT CURRENT_TIMESTAMP,
		session_id TEXT NOT NULL,
		host       TEXT NOT NULL,
		output     TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create terminal_logs table: %w", err)
	}

	// Create indexes for common query patterns
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_host ON terminal_logs(host)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_session ON terminal_logs(session_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON terminal_logs(timestamp DESC)`)

	// Create app_config table for persistent settings
	db.Exec(`CREATE TABLE IF NOT EXISTS app_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Create alert_history table
	db.Exec(`CREATE TABLE IF NOT EXISTS alert_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		trigger_id TEXT,
		name TEXT,
		severity TEXT,
		host TEXT,
		session_id TEXT,
		log_line TEXT,
		sent INTEGER DEFAULT 0
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_alert_hist_ts ON alert_history(timestamp DESC)`)

	// Create session_recordings
	db.Exec(`CREATE TABLE IF NOT EXISTS session_recordings (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		host_label TEXT NOT NULL,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		duration REAL DEFAULT 0,
		event_count INTEGER DEFAULT 0,
		cols INTEGER NOT NULL,
		rows INTEGER NOT NULL,
		status TEXT DEFAULT 'in_progress'
	)`)

	// Create recording_frames
	db.Exec(`CREATE TABLE IF NOT EXISTS recording_frames (
		recording_id TEXT NOT NULL,
		timestamp REAL NOT NULL,
		type TEXT NOT NULL,
		data TEXT NOT NULL,
		FOREIGN KEY(recording_id) REFERENCES session_recordings(id) ON DELETE CASCADE
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_rec_frames_id ON recording_frames(recording_id, timestamp)`)

	// Create FTS5 virtual table for forensic search
	db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS recording_frames_fts USING fts5(
		data,
		recording_id UNINDEXED,
		tokenize='unicode61 remove_diacritics 1'
	)`)

	return nil
}
