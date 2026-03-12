package logger_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestLoggerJSON(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	l, err := logger.New(logger.Config{
		Level:      logger.InfoLevel,
		OutputPath: logPath,
		JSON:       true,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	l.Info("Test message with %s", "param")
	l.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	err = json.Unmarshal(data, &entry)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON log: %v. Raw: %s", err, string(data))
	}

	if entry["message"] != "Test message with param" {
		t.Errorf("Expected message 'Test message with param', got '%v'", entry["message"])
	}
	if entry["level"] != "info" {
		t.Errorf("Expected level 'info', got '%v'", entry["level"])
	}
	if entry["timestamp"] == "" {
		t.Error("Expected timestamp to be present")
	}
}

func TestLoggerWithPrefixJSON(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test_prefix.log")

	l, _ := logger.New(logger.Config{
		Level:      logger.InfoLevel,
		OutputPath: logPath,
		JSON:       true,
	})

	pl := l.WithPrefix("test-svc")
	pl.Info("Hello")
	l.Close()

	data, _ := os.ReadFile(logPath)
	var entry map[string]interface{}
	json.Unmarshal(data, &entry)

	if entry["prefix"] != "TEST-SVC" {
		t.Errorf("Expected prefix 'TEST-SVC', got '%v'", entry["prefix"])
	}
	if entry["message"] != "Hello" {
		t.Errorf("Expected message 'Hello', got '%v'", entry["message"])
	}
}
