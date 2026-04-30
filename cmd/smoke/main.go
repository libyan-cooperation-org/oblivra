// oblivra-smoke — end-to-end smoke test of every documented REST surface.
//
// Hits every endpoint, asserts response shape (status code + key field
// presence), and exits non-zero on the first surprise. Designed for CI:
// run a fresh server, point this at it, fail the build if anything drifts.
//
// Usage:
//
//	oblivra-smoke --server http://localhost:8080 --token $OBLIVRA_TOKEN
//
// Exits 0 if every check passed, 1 if any check failed.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	server := flag.String("server", "http://localhost:8080", "server URL")
	token := flag.String("token", os.Getenv("OBLIVRA_TOKEN"), "API key")
	flag.Parse()

	c := &client{base: strings.TrimRight(*server, "/"), token: *token, http: &http.Client{Timeout: 10 * time.Second}}

	var failures int

	// 1. Liveness.
	c.must(&failures, "GET /healthz", "GET", "/healthz", nil, 200, []string{"status"})
	c.must(&failures, "GET /readyz", "GET", "/readyz", nil, 200, nil)
	c.must(&failures, "GET /metrics", "GET", "/metrics", nil, 200, nil)

	// 2. System.
	c.must(&failures, "GET /api/v1/system/info", "GET", "/api/v1/system/info", nil, 200, []string{"version", "goVersion"})
	c.must(&failures, "GET /api/v1/system/ping", "GET", "/api/v1/system/ping", nil, 200, []string{"status"})

	// 3. Ingest a probe event.
	probe := map[string]any{
		"source": "rest", "hostId": "smoke-host", "severity": "info",
		"eventType": "smoke.probe", "message": "smoke test event",
	}
	c.must(&failures, "POST /api/v1/siem/ingest", "POST", "/api/v1/siem/ingest", probe, 202, []string{"id", "hash"})

	// 4. Search & stats.
	c.must(&failures, "GET /api/v1/siem/stats", "GET", "/api/v1/siem/stats", nil, 200, []string{"total", "latency"})
	c.must(&failures, "GET /api/v1/siem/search", "GET", "/api/v1/siem/search?limit=10", nil, 200, []string{"events", "mode"})
	c.must(&failures, "GET /api/v1/siem/oql", "GET", "/api/v1/siem/oql?q=*+%7C+limit+5", nil, 200, []string{"events"})

	// 5. Detection.
	c.must(&failures, "GET /api/v1/detection/rules", "GET", "/api/v1/detection/rules", nil, 200, nil)
	c.must(&failures, "GET /api/v1/mitre/heatmap", "GET", "/api/v1/mitre/heatmap", nil, 200, nil)
	c.must(&failures, "GET /api/v1/alerts", "GET", "/api/v1/alerts?limit=10", nil, 200, nil)

	// 6. Threat intel.
	c.must(&failures, "GET /api/v1/threatintel/indicators", "GET", "/api/v1/threatintel/indicators", nil, 200, nil)
	c.must(&failures, "GET /api/v1/threatintel/lookup", "GET", "/api/v1/threatintel/lookup?value=198.51.100.7", nil, 200, []string{"match"})

	// 7. Audit.
	c.must(&failures, "GET /api/v1/audit/verify", "GET", "/api/v1/audit/verify", nil, 200, []string{"ok", "rootHash"})
	c.must(&failures, "GET /api/v1/audit/log", "GET", "/api/v1/audit/log?limit=5", nil, 200, nil)

	// 8. Trust + quality.
	c.must(&failures, "GET /api/v1/trust/summary", "GET", "/api/v1/trust/summary", nil, 200, []string{"verified"})
	c.must(&failures, "GET /api/v1/quality/sources", "GET", "/api/v1/quality/sources", nil, 200, nil)
	c.must(&failures, "GET /api/v1/quality/coverage", "GET", "/api/v1/quality/coverage", nil, 200, nil)

	// 9. Reconstruction surface.
	c.must(&failures, "GET /api/v1/reconstruction/sessions", "GET", "/api/v1/reconstruction/sessions", nil, 200, nil)
	c.must(&failures, "GET /api/v1/reconstruction/state", "GET", "/api/v1/reconstruction/state?host=smoke-host", nil, 200, []string{"hostId"})
	c.must(&failures, "GET /api/v1/reconstruction/cmdline/suspicious", "GET", "/api/v1/reconstruction/cmdline/suspicious", nil, 200, nil)
	c.must(&failures, "GET /api/v1/reconstruction/auth/multi-protocol", "GET", "/api/v1/reconstruction/auth/multi-protocol", nil, 200, nil)
	c.must(&failures, "GET /api/v1/reconstruction/entities", "GET", "/api/v1/reconstruction/entities?kind=host", nil, 200, nil)

	// 10. Forensics.
	c.must(&failures, "GET /api/v1/forensics/gaps", "GET", "/api/v1/forensics/gaps", nil, 200, nil)
	c.must(&failures, "GET /api/v1/forensics/lineage", "GET", "/api/v1/forensics/lineage", nil, 200, nil)
	c.must(&failures, "GET /api/v1/forensics/tamper", "GET", "/api/v1/forensics/tamper", nil, 200, nil)
	c.must(&failures, "GET /api/v1/forensics/evidence", "GET", "/api/v1/forensics/evidence", nil, 200, nil)

	// 11. Storage + tenants.
	c.must(&failures, "GET /api/v1/storage/stats", "GET", "/api/v1/storage/stats", nil, 200, nil)
	c.must(&failures, "GET /api/v1/storage/verify-warm", "GET", "/api/v1/storage/verify-warm", nil, 200, []string{"ok"})
	c.must(&failures, "GET /api/v1/tenants/policies", "GET", "/api/v1/tenants/policies", nil, 200, nil)

	// 12. Vault.
	c.must(&failures, "GET /api/v1/vault/status", "GET", "/api/v1/vault/status", nil, 200, []string{"exists"})

	// 13. Cases — open then list, then run case-scoped surface.
	caseResp := c.expect(&failures, "POST /api/v1/cases", "POST", "/api/v1/cases",
		map[string]any{"title": "smoke", "hostId": "smoke-host"}, 201, []string{"id", "scope"})
	if caseResp != nil {
		id, _ := caseResp["id"].(string)
		if id != "" {
			c.must(&failures, "GET /api/v1/cases/{id}", "GET", "/api/v1/cases/"+id, nil, 200, nil)
			c.must(&failures, "GET /api/v1/cases/{id}/timeline", "GET", "/api/v1/cases/"+id+"/timeline", nil, 200, nil)
			c.must(&failures, "GET /api/v1/cases/{id}/confidence", "GET", "/api/v1/cases/"+id+"/confidence", nil, 200, []string{"score"})
			c.must(&failures, "POST /api/v1/cases/{id}/notes", "POST", "/api/v1/cases/"+id+"/notes",
				map[string]any{"body": "smoke note"}, 200, nil)
			c.must(&failures, "POST /api/v1/cases/{id}/seal", "POST", "/api/v1/cases/"+id+"/seal", nil, 200, []string{"state"})
			c.mustNon(&failures, "GET /api/v1/cases/{id}/report.html", "GET", "/api/v1/cases/"+id+"/report.html", nil, 200)
		}
	}

	// 14. Investigations + pivot.
	c.must(&failures, "GET /api/v1/investigations/timeline", "GET", "/api/v1/investigations/timeline", nil, 200, nil)
	c.must(&failures, "GET /api/v1/investigations/pivot", "GET", "/api/v1/investigations/pivot?host=smoke-host", nil, 200, nil)

	// 15. Graph.
	c.must(&failures, "GET /api/v1/graph/stats", "GET", "/api/v1/graph/stats", nil, 200, []string{"edges"})

	// 16. Fleet.
	c.must(&failures, "GET /api/v1/agent/fleet", "GET", "/api/v1/agent/fleet", nil, 200, nil)

	if failures > 0 {
		fmt.Fprintf(os.Stderr, "\n%d smoke check(s) failed\n", failures)
		os.Exit(1)
	}
	fmt.Println("\nall smoke checks passed")
}

type client struct {
	base  string
	token string
	http  *http.Client
}

func (c *client) do(method, path string, body any) (*http.Response, []byte, error) {
	var bodyR io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bodyR = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, bodyR)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, out, nil
}

// must hits an endpoint and prints ✓ on success, ✗ + reason on failure.
func (c *client) must(failures *int, label, method, path string, body any, wantStatus int, wantFields []string) {
	c.expect(failures, label, method, path, body, wantStatus, wantFields)
}

// mustNon checks status only — used for non-JSON endpoints (HTML report).
func (c *client) mustNon(failures *int, label, method, path string, body any, wantStatus int) {
	resp, _, err := c.do(method, path, body)
	if err != nil {
		*failures++
		fmt.Printf("✗  %-40s  %v\n", label, err)
		return
	}
	if resp.StatusCode != wantStatus {
		*failures++
		fmt.Printf("✗  %-40s  status %d != %d\n", label, resp.StatusCode, wantStatus)
		return
	}
	fmt.Printf("✓  %-40s  status=%d\n", label, resp.StatusCode)
}

// expect returns the parsed JSON when shape passes (or nil on failure).
func (c *client) expect(failures *int, label, method, path string, body any, wantStatus int, wantFields []string) map[string]any {
	resp, out, err := c.do(method, path, body)
	if err != nil {
		*failures++
		fmt.Printf("✗  %-40s  %v\n", label, err)
		return nil
	}
	if resp.StatusCode != wantStatus {
		*failures++
		fmt.Printf("✗  %-40s  status %d != %d  body=%s\n", label, resp.StatusCode, wantStatus, truncate(string(out), 80))
		return nil
	}
	if len(wantFields) > 0 && len(out) > 0 && out[0] == '{' {
		var m map[string]any
		if err := json.Unmarshal(out, &m); err == nil {
			for _, f := range wantFields {
				if _, ok := m[f]; !ok {
					*failures++
					fmt.Printf("✗  %-40s  missing field %q\n", label, f)
					return m
				}
			}
		}
		fmt.Printf("✓  %-40s  status=%d\n", label, resp.StatusCode)
		return m
	}
	fmt.Printf("✓  %-40s  status=%d\n", label, resp.StatusCode)
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
