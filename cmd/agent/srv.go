package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Phase 40 — DNS SRV server discovery.
//
// An operator who can't bake the platform hostname into every agent
// rollout can put it in DNS instead. Configuring `server.url:
// "srv://_oblivra._tcp.example.com"` makes the agent resolve the SRV
// record, sort by RFC-2782 priority/weight, and try each target until
// one passes a /healthz probe. The first reachable target becomes the
// effective server URL for the rest of the agent's lifetime.
//
// This complements (does not replace) the explicit URL form. UF
// historically expects you to bake the indexer list into outputs.conf;
// SRV makes a fleet rollout one CNAME-edit-per-region instead of N
// agent-config rewrites.
//
// Format:
//   srv://[scheme@]_service._proto.domain
//
//   scheme      "https" (default if omitted) | "http"
//   _service    e.g. "_oblivra"
//   _proto      e.g. "_tcp"
//
// Examples:
//   srv://_oblivra._tcp.internal.example.com
//   srv://http@_oblivra._tcp.dev.example.com

const srvPrefix = "srv://"

// resolveSRVURL inspects url and returns either the original string (if
// it isn't an srv:// form) or the first reachable target's http(s) URL.
// Probes /healthz on each target with a short per-attempt timeout.
func resolveSRVURL(ctx context.Context, url string) (string, error) {
	if !strings.HasPrefix(url, srvPrefix) {
		return url, nil
	}
	tail := strings.TrimPrefix(url, srvPrefix)

	scheme := "https"
	if at := strings.Index(tail, "@"); at >= 0 {
		scheme = tail[:at]
		tail = tail[at+1:]
	}
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("srv: unsupported scheme %q (use http or https)", scheme)
	}

	parts := strings.SplitN(tail, ".", 3)
	if len(parts) < 3 || !strings.HasPrefix(parts[0], "_") || !strings.HasPrefix(parts[1], "_") {
		return "", errors.New("srv: expected `_service._proto.domain`")
	}
	service := strings.TrimPrefix(parts[0], "_")
	proto := strings.TrimPrefix(parts[1], "_")
	name := parts[2]

	resolver := net.DefaultResolver
	rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, addrs, err := resolver.LookupSRV(rctx, service, proto, name)
	if err != nil {
		return "", fmt.Errorf("srv lookup %s: %w", url, err)
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("srv: no records for %s", url)
	}

	// LookupSRV already returns records ordered by priority then weight
	// per RFC 2782, so we walk in order.
	probe := &http.Client{Timeout: 2 * time.Second}
	for _, a := range addrs {
		host := strings.TrimSuffix(a.Target, ".")
		candidate := fmt.Sprintf("%s://%s:%d", scheme, host, a.Port)
		if probeHealth(probe, candidate) {
			log.Printf("srv: %s → %s", url, candidate)
			return candidate, nil
		}
	}
	return "", fmt.Errorf("srv: no reachable target among %d records for %s", len(addrs), url)
}

func probeHealth(c *http.Client, base string) bool {
	resp, err := c.Get(base + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}
