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

// auditedRoutes lists path prefixes whose calls land in the tamper-evident
// query log. We deliberately don't audit liveness, metrics, or static assets
// — they'd drown the chain in noise.
var auditedRoutes = []struct {
	prefix string
	method string // "" matches any
	action string
}{
	{"/api/v1/siem/search", "GET", "siem.search"},
	{"/api/v1/siem/oql", "GET", "siem.oql"},
	{"/api/v1/siem/ingest/raw", "POST", "siem.ingest.raw"},
	{"/api/v1/audit/log", "GET", "audit.read"},
	{"/api/v1/audit/verify", "GET", "audit.verify"},
	{"/api/v1/audit/packages/generate", "POST", "audit.export"},
	{"/api/v1/forensics/evidence", "POST", "evidence.seal"},
	{"/api/v1/storage/promote", "POST", "storage.promote"},
	{"/api/v1/detection/rules/reload", "POST", "rules.reload"},
	{"/api/v1/threatintel/indicator", "POST", "intel.add"},
	{"/api/v1/vault/init", "POST", "vault.init"},
	{"/api/v1/vault/unlock", "POST", "vault.unlock"},
	{"/api/v1/vault/lock", "POST", "vault.lock"},
	{"/api/v1/vault/secret", "POST", "vault.secret.set"},
	{"/api/v1/vault/secret", "DELETE", "vault.secret.delete"},
	{"/api/v1/agent/register", "POST", "fleet.register"},
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
