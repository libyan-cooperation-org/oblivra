package database

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/kingknull/oblivrashell/internal/cluster"
)

var ErrLocked = errors.New("database is locked")

type Database struct {
	mu sync.RWMutex
	db *sql.DB
	cm cluster.Manager
}

func (d *Database) IsLocked() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db == nil
}

// Conn returns the underlying sql.DB connection.
// It returns ErrLocked if the vault hasn't been unlocked yet.
func (d *Database) Conn() (*sql.DB, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, ErrLocked
	}
	return d.db, nil
}

func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *Database) DB() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

func (d *Database) Lock()    { d.mu.Lock() }
func (d *Database) Unlock()  { d.mu.Unlock() }
func (d *Database) RLock()   { d.mu.RLock() }
func (d *Database) RUnlock() { d.mu.RUnlock() }

func (d *Database) SetClusterManager(cm cluster.Manager) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cm = cm
}

func (d *Database) ClusterManager() cluster.Manager {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.cm
}

type raftResult struct {
	lastInsertId int64
	rowsAffected int64
}

func (r *raftResult) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

func (r *raftResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// ReplicatedExecContext intercepts a write query and routes it through the Raft FSM if clustering is active.
// Otherwise, it falls back to a standard local database execution.
func (d *Database) ReplicatedExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	d.mu.RLock()
	db := d.db
	cm := d.cm
	d.mu.RUnlock()

	if db == nil {
		return nil, ErrLocked
	}

	if cm != nil {
		lastId, rowsAff, err := cm.ApplyWrite(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		return &raftResult{
			lastInsertId: lastId,
			rowsAffected: rowsAff,
		}, nil
	}

	// Standalone fallback
	return db.ExecContext(ctx, query, args...)
}
