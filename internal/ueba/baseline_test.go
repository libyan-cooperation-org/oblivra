package ueba

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/storage"
)

type mockKV struct {
	data map[string][]byte
}

func (m *mockKV) Put(key []byte, value []byte, ttl time.Duration) error {
	m.data[string(key)] = value
	return nil
}

func (m *mockKV) Get(key []byte) ([]byte, error) {
	return m.data[string(key)], nil
}

func (m *mockKV) IteratePrefix(prefix []byte, fn func(key, value []byte) error) error {
	for k, v := range m.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == string(prefix) {
			if err := fn([]byte(k), v); err != nil {
				return err
			}
		}
	}
	return nil
}

func TestBaselinePersistence(t *testing.T) {
	mock := &mockKV{data: make(map[string][]byte)}
	store := NewBaselineStore(mock)

	// 1. Create and update a profile
	p := store.GetOrCreateProfile("user1", "user")
	p.UpdateFeature("freq", 10.0)
	p.SetRiskScore(0.5)

	// 2. Save
	err := store.Save("user1")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 3. Create a new store and load
	store2 := NewBaselineStore(mock)
	err = store2.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	p2 := store2.GetProfile("user1")
	if p2 == nil {
		t.Fatal("Profile not recovered")
	}

	if p2.RiskScore != 0.5 {
		t.Fatalf("Risk score mismatch: expected 0.5, got %v", p2.RiskScore)
	}

	if p2.FeatureVectors["freq"] != 10.0 {
		t.Fatalf("Feature mismatch: expected 10.0, got %v", p2.FeatureVectors["freq"])
	}
}

func TestFeatureEMA(t *testing.T) {
	mock := &mockKV{data: make(map[string][]byte)}
	store := NewBaselineStore(mock)
	p := store.GetOrCreateProfile("user1", "user")

	// Update 1: Initial value
	p.UpdateFeature("freq", 100.0)
	if p.FeatureVectors["freq"] != 100.0 {
		t.Fatalf("Expected 100.0, got %v", p.FeatureVectors["freq"])
	}

	// Update 2: EMA logic: (0.2 * 50) + (0.8 * 100) = 10 + 80 = 90
	p.UpdateFeature("freq", 50.0)
	expected := (0.2 * 50.0) + (0.8 * 100.0)
	if p.FeatureVectors["freq"] != expected {
		t.Fatalf("Expected %v, got %v", expected, p.FeatureVectors["freq"])
	}
}

func TestUEBAService_Integration(t *testing.T) {
	// Real Badger integration test (short)
	dataDir := "test_ueba_badger"
	os.MkdirAll(dataDir, 0755)
	defer os.RemoveAll(dataDir)

	log, _ := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: filepath.Join(dataDir, "test.log"),
	})
	hotStore, err := storage.NewHotStore(dataDir, log)
	if err != nil {
		t.Fatalf("HotStore init failed: %v", err)
	}
	defer hotStore.Close()

	svc := NewUEBAService(nil, nil, hotStore, log)
	
	p := svc.baseline.GetOrCreateProfile("admin", "user")
	p.UpdateFeature("login_count", 1.0)
	
	// Manually save for test confirmation
	err = svc.baseline.Save("admin")
	if err != nil {
		t.Fatalf("Manual save failed: %v", err)
	}

	// Verify it exists in Badger
	val, err := hotStore.Get([]byte("ueba:profile:admin"))
	if err != nil || val == nil {
		t.Fatalf("Profile missing from Badger: %v", err)
	}
}
