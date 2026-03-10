//go:build sqlcipher
// +build sqlcipher

package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mutecomm/go-sqlcipher/v4"
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

	// Open initializes the SQLite connection without the key in DSN
	dsn := fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000",
		dbPath,
	)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	if len(encryptionKey) > 0 {
		// Securely set the key via PRAGMA
		hexKey := fmt.Sprintf("%x", encryptionKey)
		if _, err := db.Exec(fmt.Sprintf("PRAGMA key = \"x'%s'\"", hexKey)); err != nil {
			db.Close()
			return fmt.Errorf("set encryption key: %w", err)
		}
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("database encryption key rejected (or file corrupt): %w", err)
	}

	d.db = db
	return nil
}
