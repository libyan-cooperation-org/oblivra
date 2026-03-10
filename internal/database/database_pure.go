//go:build !sqlcipher
// +build !sqlcipher

package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func New(dbPath string) (*Database, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	return &Database{}, nil
}

func (d *Database) Open(dbPath string, encryptionKey []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db != nil {
		d.db.Close()
	}

	var dsn string
	dsn = fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000",
		dbPath,
	)

	// Open initializes the SQLite connection (pure driver)
	// SQLite pure doesn't support encryption via this driver easily, but we'll ignore key for now
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("database ping failed: %w", err)
	}

	d.db = db
	return nil
}
