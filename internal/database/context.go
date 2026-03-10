package database

import "context"

type contextKey string

const (
	tenantConfigKey contextKey = "tenant_id"
	DefaultTenantID string     = "default_tenant"
)

// WithTenantID returns a new context with the given tenant ID.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantConfigKey, tenantID)
}

// TenantFromContext extracts the tenant ID from the context.
// Returns "default_tenant" if no tenant is found (for backward compatibility).
func TenantFromContext(ctx context.Context) string {
	if ctx == nil {
		return DefaultTenantID
	}
	if tenantID, ok := ctx.Value(tenantConfigKey).(string); ok && tenantID != "" {
		return tenantID
	}
	return DefaultTenantID
}
