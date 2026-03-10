package database

import (
	"os"
	"testing"
)

func TestMigrationsFreshInstall(t *testing.T) {
	os.Remove("test_fresh.db")
	db, err := New("test_fresh.db")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	err = db.Open("test_fresh.db", nil)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	defer os.Remove("test_fresh.db")

	err = db.Migrate()
	if err != nil {
		t.Fatalf("Migration failed on fresh install: %v", err)
	}
}
