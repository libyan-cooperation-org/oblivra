package services

import (
	"context"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/logsources"
)

func setupTestService(_ *testing.T) *LogSourceService {
	l, _ := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: "test.log",
		Sanitize:   true,
	})
	bus := eventbus.NewBus(l)
	ae := analytics.NewAnalyticsEngine(nil)
	_ = ae.Open(":memory:", []byte(""))
	manager := logsources.NewSourceManager(l)

	service := NewLogSourceService(manager, ae, bus, l)
	service.ctx = context.Background()
	return service
}

func TestGetSourcesObfuscation(t *testing.T) {
	service := setupTestService(t)

	src := logsources.LogSource{
		ID:       "test-src-1",
		Name:     "Test Source",
		Type:     logsources.SourceElasticsearch,
		Enabled:  true,
		APIKey:   "secret-api-key",
		Password: "super-secret-password",
	}

	service.AddSource(src)

	sources := service.GetSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	got := sources[0]
	if got.APIKey != "********" {
		t.Errorf("expected APIKey to be obfuscated, got: %s", got.APIKey)
	}
	if got.Password != "********" {
		t.Errorf("expected Password to be obfuscated, got: %s", got.Password)
	}
}

func TestTags(t *testing.T) {
	service := setupTestService(t)

	service.AddSource(logsources.LogSource{
		ID:   "src-1",
		Tags: []string{"prod", "db"},
	})

	service.AddSource(logsources.LogSource{
		ID:   "src-2",
		Tags: []string{"prod", "web"},
	})

	tags := service.GetAllTags()

	// Should be deduplicated
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got: %d (%v)", len(tags), tags)
	}

	// Test fetching by tag
	prodSources := service.GetSourcesByTag("prod")
	if len(prodSources) != 2 {
		t.Errorf("expected 2 prod sources, got: %d", len(prodSources))
	}

	dbSources := service.GetSourcesByTag("db")
	if len(dbSources) != 1 {
		t.Errorf("expected 1 db source, got: %d", len(dbSources))
	}
}

func TestRateLimiter(t *testing.T) {
	service := setupTestService(t)

	sourceID := "rate-test-1"

	service.AddSource(logsources.LogSource{
		ID:      sourceID,
		Name:    "Rate Test",
		Enabled: true,
	})

	// First query should pass rate limit check natively inside canQuery
	if !service.canQuery(sourceID) {
		t.Error("expected first query to pass rate limiter")
	}

	// Second query immediately should fail
	if service.canQuery(sourceID) {
		t.Error("expected immediate second query to fail rate limiter")
	}

	// Wait 500ms
	time.Sleep(510 * time.Millisecond)

	// Third query should pass
	if !service.canQuery(sourceID) {
		t.Error("expected query to pass after 500ms wait")
	}
}
