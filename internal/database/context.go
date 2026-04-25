package database

import (
	"context"
	"fmt"
)

type contextKey string

const (
	tenantConfigKey contextKey = "tenant_id"
	DefaultTenantID string     = "default_tenant"
	GlobalSearchKey contextKey = "global_search"
)

// EnforceStrictIsolation controls whether repository calls without an explicit
// tenant ID in the context should panic (true) or fallback to DefaultTenantID (false).
// It defaults to false for the desktop/sovereign terminal experience.
var EnforceStrictIsolation bool = false

// WithTenant returns a new context with the given tenant ID.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	if tenantID == "" {
		panic("SECURITY: WithTenant called with empty string. Use WithGlobalSearch for unrestricted access.")
	}
	return context.WithValue(ctx, tenantConfigKey, tenantID)
}

// TenantFromContext extracts the tenant ID from the context.
// Returns an error if no tenant is found and EnforceStrictIsolation is true.
func TenantFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		if !EnforceStrictIsolation {
			return DefaultTenantID, nil
		}
		return "", fmt.Errorf("missing context")
	}
	if tenantID, ok := ctx.Value(tenantConfigKey).(string); ok && tenantID != "" {
		return tenantID, nil
	}
	
	// Check if this is an explicitly allowed global search
	if isGlobal, _ := ctx.Value(GlobalSearchKey).(bool); isGlobal {
		return "", nil // empty string is allowed ONLY if GlobalSearchKey is true
	}

	if !EnforceStrictIsolation {
		return DefaultTenantID, nil
	}

	return "", fmt.Errorf("missing tenant context (unscoped access denied)")
}

// MustTenantFromContext extracts the tenant ID or panics if missing (and strict mode is on).
func MustTenantFromContext(ctx context.Context) string {
	tid, err := TenantFromContext(ctx)
	if err != nil {
		panic(fmt.Sprintf("SECURITY: %v", err))
	}
	return tid
}

// WithGlobalSearch returns a context that signals repositories to skip tenant filtering.
func WithGlobalSearch(ctx context.Context) context.Context {
	return context.WithValue(ctx, GlobalSearchKey, true)
}
