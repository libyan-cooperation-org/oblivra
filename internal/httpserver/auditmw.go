package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivra/internal/rbac"
	"github.com/kingknull/oblivra/internal/services"
)

// auditedRoutes lists routes whose calls land in the tamper-evident query
// log. We deliberately don't audit liveness, metrics, or static assets — they
// would drown the chain in noise.
//
// `exact` routes only match an identical path; everything else matches by
// prefix. List children before parents when both are audited so the longer
// match wins. We deliberately set the parent (e.g. /api/v1/cases) to `exact`
// so child paths like /api/v1/cases/{id}/notes get their own classification.
var auditedRoutes = []struct {
	prefix string
	method string // "" matches any
	action string
	exact  bool
}{
	// SIEM / search
	{prefix: "/api/v1/siem/search", method: "GET", action: "siem.search"},
	{prefix: "/api/v1/siem/oql", method: "GET", action: "siem.oql"},
	{prefix: "/api/v1/siem/ingest/raw", method: "POST", action: "siem.ingest.raw"},

	// Audit
	{prefix: "/api/v1/audit/log", method: "GET", action: "audit.read"},
	{prefix: "/api/v1/audit/verify", method: "GET", action: "audit.verify"},
	{prefix: "/api/v1/audit/packages/generate", method: "POST", action: "audit.export"},

	// Evidence / forensics
	{prefix: "/api/v1/forensics/evidence", method: "POST", action: "evidence.seal"},

	// Storage / rules / intel
	{prefix: "/api/v1/storage/promote", method: "POST", action: "storage.promote"},
	{prefix: "/api/v1/detection/rules/reload", method: "POST", action: "rules.reload"},
	{prefix: "/api/v1/threatintel/indicator", method: "POST", action: "intel.add"},

	// Vault
	{prefix: "/api/v1/vault/init", method: "POST", action: "vault.init"},
	{prefix: "/api/v1/vault/unlock", method: "POST", action: "vault.unlock"},
	{prefix: "/api/v1/vault/lock", method: "POST", action: "vault.lock"},
	{prefix: "/api/v1/vault/secret", method: "POST", action: "vault.secret.set"},
	{prefix: "/api/v1/vault/secret", method: "DELETE", action: "vault.secret.delete"},

	// Fleet
	{prefix: "/api/v1/agent/register", method: "POST", action: "fleet.register"},

	// Cases — children first, parent exact. Service-level Append() in
	// InvestigationsService records the *semantic* action; the middleware
	// just records the HTTP-shaped call.
	{prefix: "/api/v1/cases/", method: "POST", action: "investigation.mutate"},
	{prefix: "/api/v1/cases/", method: "GET", action: "investigation.read"},
	{prefix: "/api/v1/cases", method: "POST", action: "investigation.open", exact: true},
	{prefix: "/api/v1/cases", method: "GET", action: "investigation.list", exact: true},
}

// queryAudit wraps an http.Handler so that every audited request lands a
// signed entry in the durable audit chain *after* the response is written.
// We capture the response status + body length via a tiny wrapper so the
// audit entry records what the client actually saw.
func queryAudit(audit *services.AuditService, next http.Handler) http.Handler {
	if audit == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action, ok := matchAudited(r.Method, r.URL.Path)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		started := time.Now()
		rec := &recorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)

		actor, tenant := actorOf(r.Context())
		detail := map[string]string{
			"method":   r.Method,
			"path":     r.URL.Path,
			"status":   strconv.Itoa(rec.status),
			"bytes":    strconv.FormatInt(rec.bytes.Load(), 10),
			"duration": time.Since(started).Round(time.Microsecond).String(),
			"remote":   strings.SplitN(r.RemoteAddr, ":", 2)[0],
		}
		if q := r.URL.RawQuery; q != "" {
			detail["query"] = q
		}
		if ua := r.Header.Get("User-Agent"); ua != "" {
			// Hash the UA so we don't bloat the chain with verbose strings.
			sum := sha256.Sum256([]byte(ua))
			detail["uaHash"] = hex.EncodeToString(sum[:8])
		}
		audit.Append(r.Context(), actor, action, tenant, detail)
	})
}

func matchAudited(method, path string) (string, bool) {
	for _, r := range auditedRoutes {
		if r.method != "" && r.method != method {
			continue
		}
		if r.exact {
			if path == r.prefix {
				return r.action, true
			}
			continue
		}
		if strings.HasPrefix(path, r.prefix) {
			return r.action, true
		}
	}
	return "", false
}

func actorOf(ctx context.Context) (actor, tenant string) {
	if s, ok := rbac.FromContext(ctx); ok {
		actor = string(s.Role) + ":" + s.ID
		tenant = s.Tenant
		if tenant == "" {
			tenant = "default"
		}
		return actor, tenant
	}
	return "anonymous", "default"
}

// recorder captures status + bytes written without copying the body.
type recorder struct {
	http.ResponseWriter
	status int
	bytes  atomic.Int64
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes.Add(int64(n))
	return n, err
}
