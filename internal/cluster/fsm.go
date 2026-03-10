package cluster

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/raft"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// SQLWriteCommand represents a database write operation replicated through Raft
type SQLWriteCommand struct {
	Query string        `json:"query"`
	Args  []interface{} `json:"args"`
}

// FSM is a finite state machine that applies Raft log entries to the local SQLite database.
type FSM struct {
	db     *sql.DB
	dbPath string
	log    *logger.Logger
}

// FSMApplyResponse contains the result of a database execution
type FSMApplyResponse struct {
	LastInsertId int64
	RowsAffected int64
	Err          error
}

// NewFSM creates a new SQLite-backed FSM
func NewFSM(db *sql.DB, dbPath string, log *logger.Logger) *FSM {
	return &FSM{
		db:     db,
		dbPath: dbPath,
		log:    log,
	}
}

// Apply executes a Raft log entry on the local database
func (f *FSM) Apply(l *raft.Log) interface{} {
	var cmd SQLWriteCommand
	if err := json.Unmarshal(l.Data, &cmd); err != nil {
		f.log.Error("[RAFT-FSM] Failed to unmarshal log entry: %v", err)
		return FSMApplyResponse{Err: err}
	}

	result, err := f.db.Exec(cmd.Query, cmd.Args...)
	if err != nil {
		f.log.Error("[RAFT-FSM] Failed to execute replicated query: %v - Query: %s", err, cmd.Query)
		return FSMApplyResponse{Err: err}
	}

	lastId, _ := result.LastInsertId()
	rowsAff, _ := result.RowsAffected()

	return FSMApplyResponse{
		LastInsertId: lastId,
		RowsAffected: rowsAff,
		Err:          nil,
	}
}

// Snapshot creates an FSMSnapshot by using SQLite's online backup API via SQL
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.log.Info("[RAFT-FSM] Creating database snapshot...")

	// Create a temp file for the snapshot
	tmpFile, err := os.CreateTemp("", "raft-snapshot-*.db")
	if err != nil {
		return nil, fmt.Errorf("create snapshot temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Use SQLite VACUUM INTO to create a consistent copy of the database
	// This is atomic and doesn't require locking the source database.
	_, err = f.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", tmpPath))
	if err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("vacuum into snapshot: %w", err)
	}

	f.log.Info("[RAFT-FSM] Snapshot created at %s", tmpPath)
	return &FSMSnapshot{snapshotPath: tmpPath, log: f.log}, nil
}

// Restore applies a snapshot to the local FSM state by swapping the database file
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	f.log.Info("[RAFT-FSM] Restoring database from snapshot...")

	// Write the incoming snapshot to a temporary file
	tmpFile, err := os.CreateTemp("", "raft-restore-*.db")
	if err != nil {
		return fmt.Errorf("create restore temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, rc); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write snapshot to temp: %w", err)
	}
	tmpFile.Close()

	// Close the current database connection
	if err := f.db.Close(); err != nil {
		f.log.Warn("[RAFT-FSM] Error closing db before restore: %v", err)
	}

	// Replace the database file with the snapshot
	if err := os.Rename(tmpPath, f.dbPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("swap database file: %w", err)
	}

	// Reopen the database
	newDB, err := sql.Open("sqlite3", f.dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("reopen database after restore: %w", err)
	}
	f.db = newDB

	f.log.Info("[RAFT-FSM] Database restored successfully from snapshot")
	return nil
}

// FSMSnapshot implements raft.FSMSnapshot backed by a real database copy
type FSMSnapshot struct {
	snapshotPath string
	log          *logger.Logger
}

// Persist streams the snapshot database file to the Raft sink
func (s *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	s.log.Info("[RAFT-FSM] Persisting snapshot to Raft store...")

	file, err := os.Open(s.snapshotPath)
	if err != nil {
		sink.Cancel()
		return fmt.Errorf("open snapshot file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(sink, file); err != nil {
		sink.Cancel()
		return fmt.Errorf("stream snapshot to sink: %w", err)
	}

	return sink.Close()
}

// Release cleans up the temporary snapshot file
func (s *FSMSnapshot) Release() {
	if s.snapshotPath != "" {
		os.Remove(s.snapshotPath)
		s.log.Info("[RAFT-FSM] Snapshot temp file cleaned up")
	}
}
