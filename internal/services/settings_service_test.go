package services

import (
	"database/sql"
	"os"
	"testing"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/gdpr"
	"github.com/kingknull/oblivrashell/internal/logger"
	_ "modernc.org/sqlite"
)

func TestSettingsService_ClearDatabase(t *testing.T) {
	// Setup infra
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	bus := eventbus.NewBus(log)

	// Create a real in-memory SQLite DB for the test
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open sqlite: %v", err)
	}
	defer sqlDB.Close()

	// Create a minimal schema for testing
	_, err = sqlDB.Exec(`CREATE TABLE hosts (id TEXT, password TEXT, notes TEXT, hostname TEXT, username TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	destroyer := gdpr.NewDataDestructionService(sqlDB)

	// Mock DatabaseStore
	mockDB := &mockDatabaseStore{sqlDB: sqlDB}

	svc := NewSettingsService(mockDB, bus, log, destroyer)

	if svc == nil {
		t.Fatal("Failed to create SettingsService")
	}

	// Attempt clear
	err = svc.ClearDatabase()
	if err != nil {
		t.Fatalf("ClearDatabase failed: %v", err)
	}
}

type mockDatabaseStore struct {
	database.DatabaseStore
	sqlDB *sql.DB
}

func (m *mockDatabaseStore) DB() *sql.DB {
	return m.sqlDB
}
