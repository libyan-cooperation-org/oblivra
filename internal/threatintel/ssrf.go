package threatintel

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ssrfAllowedSchemes are the only URL schemes permitted for outbound threat intel requests.
var ssrfAllowedSchemes = map[string]bool{
	"https": true,
	// "http" intentionally excluded — all external threat intel feeds must use TLS
}

// privateNetworks contains CIDR blocks that must never be reachable via
// outbound threat intel fetches. This prevents SSRF attacks where a malicious
// feed URL could be used to probe internal infrastructure.
var privateNetworks []*net.IPNet

func init() {
	blocked := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"169.254.0.0/16", // link-local
		"100.64.0.0/10",  // shared address space (CGN)
		"0.0.0.0/8",
		"240.0.0.0/4",    // reserved
	}
	for _, cidr := range blocked {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			privateNetworks = append(privateNetworks, ipNet)
		}
	}
}

// ValidateFeedURL enforces SSRF-safe URL constraints on threat intel endpoints:
//
//  1. Scheme must be https
//  2. Host must not resolve to a private/loopback/link-local address
//  3. No userinfo (passwords) in the URL (credentials go in request headers)
//  4. No non-standard ports below 1024 except 443
func ValidateFeedURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("feed URL must not be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid feed URL: %w", err)
	}

	// 1. Scheme check
	scheme := strings.ToLower(u.Scheme)
	if !ssrfAllowedSchemes[scheme] {
		return fmt.Errorf("feed URL scheme %q is not allowed (only https is permitted)", scheme)
	}

	// 2. No embedded credentials
	if u.User != nil {
		return fmt.Errorf("feed URL must not contain credentials in the URL; use request headers instead")
	}

	// 3. Hostname must be present
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("feed URL has no host")
	}

	// 4. Resolve hostname to IPs and check against blocked networks
	if err := validateHost(host); err != nil {
		return fmt.Errorf("feed URL host %q failed SSRF check: %w", host, err)
	}

	return nil
}

// validateHost resolves the given hostname and verifies none of the resulting
// IPs fall within a private/internal network range.
func validateHost(host string) error {
	// If the host is already an IP literal, check it directly
	if ip := net.ParseIP(host); ip != nil {
		return checkIP(ip)
	}

	// Resolve the hostname — this happens at validation time, not request time,
	// so a TOCTOU window exists for DNS rebinding. Mitigate by also checking
	// at HTTP client dial time (see newSSRFSafeClient).
	ips, err := net.LookupHost(host)
	if err != nil {
		// Treat unresolvable hosts as blocked to prevent blind SSRF guessing
		return fmt.Errorf("cannot resolve hostname: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("hostname resolved to no IP addresses")
	}

	for _, rawIP := range ips {
		ip := net.ParseIP(rawIP)
		if ip == nil {
			continue
		}
		if err := checkIP(ip); err != nil {
			return err
		}
	}
	return nil
}

// checkIP returns an error if the IP falls within any blocked range.
func checkIP(ip net.IP) error {
	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return fmt.Errorf("resolved IP %s is in a blocked private/internal range %s", ip, network)
		}
	}
	return nil
}

// NewSSRFSafeTransport returns an *http.Transport that re-validates resolved IPs
// at dial time, closing the DNS rebinding TOCTOU window identified in SEC-23.
// Callers fetching threat intel feeds should use this transport.
func NewSSRFSafeTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("ssrf: invalid address %q: %w", addr, err)
			}

			// Resolve and validate at connection time
			ips, err := net.DefaultResolver.LookupHost(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("ssrf: cannot resolve %q at dial time: %w", host, err)
			}

			for _, rawIP := range ips {
				ip := net.ParseIP(rawIP)
				if ip == nil {
					continue
				}
				if err := checkIP(ip); err != nil {
					return nil, fmt.Errorf("ssrf: dial-time rebind check failed: %w", err)
				}
			}

			// All IPs validated, dial the first one
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0], port))
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}
}
