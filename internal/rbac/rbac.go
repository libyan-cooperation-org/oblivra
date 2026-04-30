// Package rbac defines OBLIVRA's role-based access control: a small fixed set
// of roles, the permissions each role implies, and helpers for guarding HTTP
// handlers / Wails methods.
package rbac

import (
	"context"
	"net/http"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleAnalyst  Role = "analyst"
	RoleReadOnly Role = "readonly"
	RoleAgent    Role = "agent"
)

// Permission constants. Keep this list flat — scoping is by role.
const (
	PermSiemRead    = "siem.read"
	PermSiemIngest  = "siem.ingest"
	PermAlertsRead  = "alerts.read"
	PermAlertsAck   = "alerts.ack"
	PermRulesRead   = "rules.read"
	PermRulesWrite  = "rules.write"
	PermIntelRead   = "intel.read"
	PermIntelWrite  = "intel.write"
	PermAuditRead   = "audit.read"
	PermAuditExport = "audit.export"
	PermFleetRead   = "fleet.read"
	PermFleetWrite  = "fleet.write"
	PermAdminAll    = "admin.all"
)

var rolePerms = map[Role]map[string]struct{}{
	RoleAdmin: setOf(
		PermSiemRead, PermSiemIngest, PermAlertsRead, PermAlertsAck,
		PermRulesRead, PermRulesWrite, PermIntelRead, PermIntelWrite,
		PermAuditRead, PermAuditExport, PermFleetRead, PermFleetWrite, PermAdminAll,
	),
	RoleAnalyst: setOf(
		PermSiemRead, PermSiemIngest, PermAlertsRead, PermAlertsAck,
		PermRulesRead, PermIntelRead, PermAuditRead, PermFleetRead,
	),
	RoleReadOnly: setOf(
		PermSiemRead, PermAlertsRead, PermRulesRead, PermIntelRead, PermAuditRead, PermFleetRead,
	),
	RoleAgent: setOf(PermSiemIngest, PermFleetRead),
}

func setOf(items ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, it := range items {
		out[it] = struct{}{}
	}
	return out
}

// HasPermission reports whether the role grants the named permission.
func (r Role) HasPermission(p string) bool {
	perms, ok := rolePerms[r]
	if !ok {
		return false
	}
	if _, has := perms[PermAdminAll]; has {
		return true
	}
	_, has := perms[p]
	return has
}

// Subject is the authenticated principal threaded through context.
type Subject struct {
	ID     string
	Role   Role
	Tenant string
}

type ctxKey struct{}

func WithSubject(ctx context.Context, s Subject) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

func FromContext(ctx context.Context) (Subject, bool) {
	v, ok := ctx.Value(ctxKey{}).(Subject)
	return v, ok
}

// WithSubjectRequest threads the Subject into the request context.
func WithSubjectRequest(r *http.Request, s Subject) *http.Request {
	return r.WithContext(WithSubject(r.Context(), s))
}
