package api

// Tests for the Phase 22.3 storage-tiering REST surface.
//
// We exercise:
//   1. GET /stats with no provider configured → 503
//   2. GET /stats with all three tiers configured → returns sizes
//   3. GET /stats correctly reports last cycle when migrator has run
//   4. GET /stats reports nil last_cycle when migrator hasn't run yet
//   5. GET /stats handles a tier whose EstimatedSize errors → reports -1
//   6. POST /promote with no provider → 503
//   7. POST /promote returns the cycle stats it just produced
//   8. Non-GET on /stats → 405; non-POST on /promote → 405
//   9. RoleReadOnly can read /stats but not promote
//
// We mock the providers rather than spinning up a real BadgerDB +
// Parquet stack — the handler logic is what we're testing.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// --- mock providers -------------------------------------------------

type mockTierStat struct {
	id     string
	size   int64
	sizeErr error
}

func (m *mockTierStat) ID() string { return m.id }
func (m *mockTierStat) EstimatedSize(_ context.Context) (int64, error) {
	return m.size, m.sizeErr
}

type mockTierMig struct {
	last      *TierMigrationStats
	runResult TierMigrationStats
}

func (m *mockTierMig) LastCycle() (TierMigrationStats, bool) {
	if m.last == nil {
		return TierMigrationStats{}, false
	}
	return *m.last, true
}
func (m *mockTierMig) RunOnce(_ context.Context) TierMigrationStats {
	return m.runResult
}

// --- helpers --------------------------------------------------------

// minimalServer constructs a RESTServer with just enough fields for
// the tiering handlers to run. The full constructor pulls in dozens
// of dependencies we don't need here.
func minimalServer() *RESTServer {
	log := logger.NewStdoutLogger()
	return &RESTServer{log: log}
}

func adminCtx() context.Context {
	return auth.ContextWithUser(context.Background(), &auth.IdentityUser{
		Email:       "admin@oblivra.org",
		Permissions: []string{"*"},
		RoleName:    string(auth.RoleAdmin),
	})
}
func analystCtx() context.Context {
	return auth.ContextWithUser(context.Background(), &auth.IdentityUser{
		Email:       "analyst@oblivra.org",
		Permissions: []string{"*:read"},
		RoleName:    string(auth.RoleAnalyst),
	})
}
func readOnlyCtx() context.Context {
	return auth.ContextWithUser(context.Background(), &auth.IdentityUser{
		Email:       "viewer@oblivra.org",
		Permissions: []string{"*:read"},
		RoleName:    string(auth.RoleReadOnly),
	})
}

// --- /stats ---------------------------------------------------------

func TestTieringStats_NotConfigured(t *testing.T) {
	s := minimalServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/tiering/stats", nil).
		WithContext(adminCtx())
	w := httptest.NewRecorder()
	s.handleTieringStats(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when tiering is not configured, got %d", w.Code)
	}
}

func TestTieringStats_AllThreeTiers(t *testing.T) {
	s := minimalServer()
	s.SetTieringProvider([]TierStatProvider{
		&mockTierStat{id: "hot", size: 1024 * 1024 * 100}, // 100 MB
		&mockTierStat{id: "warm", size: 1024 * 1024 * 50}, //  50 MB
		&mockTierStat{id: "cold", size: 0},
	}, &mockTierMig{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/tiering/stats", nil).
		WithContext(adminCtx())
	w := httptest.NewRecorder()
	s.handleTieringStats(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		OK    bool                  `json:"ok"`
		Tiers []map[string]any      `json:"tiers"`
		Last  *TierMigrationStats   `json:"last_cycle"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.OK {
		t.Fatal("expected ok=true")
	}
	if len(body.Tiers) != 3 {
		t.Fatalf("expected 3 tier rows, got %d", len(body.Tiers))
	}
	// Tier order matches the slice we passed.
	if body.Tiers[0]["id"] != "hot" || body.Tiers[1]["id"] != "warm" || body.Tiers[2]["id"] != "cold" {
		t.Errorf("tier order: %+v", body.Tiers)
	}
	// last_cycle is null because the mock migrator's last is nil.
	if body.Last != nil {
		t.Errorf("expected nil last_cycle when migrator hasn't run, got %+v", body.Last)
	}
}

func TestTieringStats_LastCycleReported(t *testing.T) {
	s := minimalServer()
	now := time.Now().UTC()
	s.SetTieringProvider(nil, &mockTierMig{
		last: &TierMigrationStats{
			StartedAt:  now.Add(-2 * time.Minute),
			FinishedAt: now,
			HotToWarm:  150,
			WarmToCold: 75,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/tiering/stats", nil).
		WithContext(adminCtx())
	w := httptest.NewRecorder()
	s.handleTieringStats(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Last *TierMigrationStats `json:"last_cycle"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Last == nil {
		t.Fatal("expected last_cycle to be reported")
	}
	if body.Last.HotToWarm != 150 || body.Last.WarmToCold != 75 {
		t.Errorf("counts: %+v", body.Last)
	}
}

func TestTieringStats_TierSizeErrorReportsMinusOne(t *testing.T) {
	s := minimalServer()
	s.SetTieringProvider([]TierStatProvider{
		&mockTierStat{id: "hot", size: 100, sizeErr: errors.New("simulated badger panic")},
	}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/tiering/stats", nil).
		WithContext(adminCtx())
	w := httptest.NewRecorder()
	s.handleTieringStats(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even when a tier errors, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"size_bytes":-1`) {
		t.Errorf("expected size_bytes:-1 for failed tier, body=%s", w.Body.String())
	}
}

func TestTieringStats_RoleReadOnlyAllowed(t *testing.T) {
	s := minimalServer()
	s.SetTieringProvider([]TierStatProvider{
		&mockTierStat{id: "hot", size: 0},
	}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/tiering/stats", nil).
		WithContext(readOnlyCtx())
	w := httptest.NewRecorder()
	s.handleTieringStats(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("RoleReadOnly should see /stats; got %d", w.Code)
	}
}

func TestTieringStats_RejectsNonGET(t *testing.T) {
	s := minimalServer()
	s.SetTieringProvider([]TierStatProvider{
		&mockTierStat{id: "hot", size: 0},
	}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/tiering/stats", nil).
		WithContext(adminCtx())
	w := httptest.NewRecorder()
	s.handleTieringStats(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 on POST /stats, got %d", w.Code)
	}
}

// --- /promote -------------------------------------------------------

func TestTieringPromote_NotConfigured(t *testing.T) {
	s := minimalServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/tiering/promote", nil).
		WithContext(adminCtx())
	w := httptest.NewRecorder()
	s.handleTieringPromote(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestTieringPromote_RejectsNonAdmin(t *testing.T) {
	s := minimalServer()
	s.SetTieringProvider(nil, &mockTierMig{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/tiering/promote", nil).
		WithContext(analystCtx())
	w := httptest.NewRecorder()
	s.handleTieringPromote(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for analyst on promote, got %d", w.Code)
	}
}

func TestTieringPromote_ReturnsCycleStats(t *testing.T) {
	s := minimalServer()
	s.SetTieringProvider(nil, &mockTierMig{
		runResult: TierMigrationStats{
			StartedAt:  time.Now().Add(-time.Minute),
			FinishedAt: time.Now(),
			HotToWarm:  42,
			WarmToCold: 7,
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/tiering/promote", nil).
		WithContext(adminCtx())
	w := httptest.NewRecorder()
	s.handleTieringPromote(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		OK    bool               `json:"ok"`
		Cycle TierMigrationStats `json:"cycle"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if !body.OK || body.Cycle.HotToWarm != 42 || body.Cycle.WarmToCold != 7 {
		t.Errorf("response shape wrong: %+v", body)
	}
}

func TestTieringPromote_RejectsNonPOST(t *testing.T) {
	s := minimalServer()
	s.SetTieringProvider(nil, &mockTierMig{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/tiering/promote", nil).
		WithContext(adminCtx())
	w := httptest.NewRecorder()
	s.handleTieringPromote(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 on GET /promote, got %d", w.Code)
	}
}
