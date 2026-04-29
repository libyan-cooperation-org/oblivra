package core

// Adapters between the concrete `internal/storage/tiering` types and
// the REST API's `TierStatProvider` / `TierMigrationProvider`
// interfaces. The api package can't import tiering directly (would
// create a cycle through services), so this file lives in core which
// already imports both.

import (
	"context"
	"sync"

	"github.com/kingknull/oblivrashell/internal/api"
	"github.com/kingknull/oblivrashell/internal/services"
	"github.com/kingknull/oblivrashell/internal/storage/tiering"
)

// tierStatAdapter wraps a single concrete `tiering.Tier` and exposes
// the `api.TierStatProvider` surface. Only the small subset of methods
// the REST stats endpoint needs is forwarded — Range / Write / Delete
// stay private to the migrator.
type tierStatAdapter struct {
	t tiering.Tier
}

func (a *tierStatAdapter) ID() string { return string(a.t.ID()) }

func (a *tierStatAdapter) EstimatedSize(ctx context.Context) (int64, error) {
	return a.t.EstimatedSize(ctx)
}

// tierMigrationAdapter wraps `*tiering.Migrator` and:
//   1. Exposes a thread-safe `LastCycle()` accessor (the migrator
//      itself doesn't keep a "most recent cycle stats" field; we
//      cache it on every RunOnce that flows through this adapter).
//   2. Translates `tiering.MigrationStats` to `api.TierMigrationStats`
//      (same shape, but living in the api package so the REST
//      handler doesn't import tiering).
type tierMigrationAdapter struct {
	m *tiering.Migrator

	mu   sync.RWMutex
	last *api.TierMigrationStats // nil until first RunOnce completes
}

func newTierMigrationAdapter(m *tiering.Migrator) *tierMigrationAdapter {
	return &tierMigrationAdapter{m: m}
}

func (a *tierMigrationAdapter) LastCycle() (api.TierMigrationStats, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.last == nil {
		return api.TierMigrationStats{}, false
	}
	return *a.last, true
}

func (a *tierMigrationAdapter) RunOnce(ctx context.Context) api.TierMigrationStats {
	stats := a.m.RunOnce(ctx)
	out := api.TierMigrationStats{
		StartedAt:  stats.StartedAt,
		FinishedAt: stats.FinishedAt,
		HotToWarm:  stats.HotToWarm,
		WarmToCold: stats.WarmToCold,
		Errors:     stats.Errors,
	}
	a.mu.Lock()
	a.last = &out
	a.mu.Unlock()
	return out
}

// WireTieringIntoAPI plugs the migrator + tier list into the REST
// server via APIService (which owns the *api.RESTServer instance).
// Called from container.initPlatform after APIService is constructed.
// Safe to call with nil migrator — endpoints return 503 cleanly.
//
// The tier-migration adapter caches the most recent cycle stats from
// the manual-promote path. Background scheduled cycles aren't yet
// observed (the migrator has no OnCycleComplete callback today);
// dashboards poll the stats endpoint to pick up the latest data.
// Adding a callback hook is a small follow-up.
func WireTieringIntoAPI(svc *services.APIService, infra *InfrastructureCluster) {
	if svc == nil || infra == nil {
		return
	}
	if infra.TierMigrator == nil {
		return
	}
	tiers := []api.TierStatProvider{}
	if infra.HotTier != nil {
		tiers = append(tiers, &tierStatAdapter{t: infra.HotTier})
	}
	if infra.WarmTier != nil {
		tiers = append(tiers, &tierStatAdapter{t: infra.WarmTier})
	}
	if infra.ColdTier != nil {
		tiers = append(tiers, &tierStatAdapter{t: infra.ColdTier})
	}
	mig := newTierMigrationAdapter(infra.TierMigrator)
	svc.SetTieringProvider(tiers, mig)
}
