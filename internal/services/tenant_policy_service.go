package services

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TenantPolicy holds per-tenant overrides for retention + a few related
// knobs. Stored as a single JSON file so the analyst can inspect / edit it
// without a database round-trip.
type TenantPolicy struct {
	TenantID   string        `json:"tenantId"`
	HotMaxAge  time.Duration `json:"hotMaxAge"`  // events older than this migrate to warm
	WarmMaxAge time.Duration `json:"warmMaxAge"` // older than this go to cold (or evict)
	UpdatedAt  time.Time     `json:"updatedAt"`
}

type TenantPolicyService struct {
	log *slog.Logger
	dir string

	mu       sync.RWMutex
	policies map[string]TenantPolicy
}

const tenantPolicyFile = "tenant_policies.json"

func NewTenantPolicyService(log *slog.Logger, dir string) (*TenantPolicyService, error) {
	if dir == "" {
		return nil, errors.New("tenant policy: dir required")
	}
	t := &TenantPolicyService{log: log, dir: dir, policies: map[string]TenantPolicy{}}
	if err := t.load(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TenantPolicyService) ServiceName() string { return "TenantPolicyService" }

func (t *TenantPolicyService) path() string { return filepath.Join(t.dir, tenantPolicyFile) }

func (t *TenantPolicyService) load() error {
	body, err := os.ReadFile(t.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var slice []TenantPolicy
	if err := json.Unmarshal(body, &slice); err != nil {
		return err
	}
	for _, p := range slice {
		t.policies[p.TenantID] = p
	}
	return nil
}

func (t *TenantPolicyService) save() error {
	t.mu.RLock()
	out := make([]TenantPolicy, 0, len(t.policies))
	for _, p := range t.policies {
		out = append(out, p)
	}
	t.mu.RUnlock()
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	tmp := t.path() + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, t.path())
}

// Get returns the tenant's policy or a sensible default.
func (t *TenantPolicyService) Get(tenantID string) TenantPolicy {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if p, ok := t.policies[tenantID]; ok {
		return p
	}
	return TenantPolicy{
		TenantID:   tenantID,
		HotMaxAge:  30 * 24 * time.Hour,
		WarmMaxAge: 180 * 24 * time.Hour,
	}
}

func (t *TenantPolicyService) Set(p TenantPolicy) error {
	if p.TenantID == "" {
		return errors.New("tenantId required")
	}
	if p.HotMaxAge <= 0 {
		p.HotMaxAge = 30 * 24 * time.Hour
	}
	if p.WarmMaxAge <= 0 {
		p.WarmMaxAge = 180 * 24 * time.Hour
	}
	p.UpdatedAt = time.Now().UTC()
	t.mu.Lock()
	t.policies[p.TenantID] = p
	t.mu.Unlock()
	return t.save()
}

func (t *TenantPolicyService) List() []TenantPolicy {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]TenantPolicy, 0, len(t.policies))
	for _, p := range t.policies {
		out = append(out, p)
	}
	return out
}

// HotMaxAge resolves the effective hot-tier max age for a tenant.
func (t *TenantPolicyService) HotMaxAge(tenantID string) time.Duration {
	return t.Get(tenantID).HotMaxAge
}
