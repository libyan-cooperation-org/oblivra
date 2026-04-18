package enrich

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
)

// DNSEnricher performs reverse DNS lookups (PTR records) on IP addresses
type DNSEnricher struct {
	resolver *net.Resolver
	timeout  time.Duration
}

func NewDNSEnricher() *DNSEnricher {
	return &DNSEnricher{
		// Use default system resolver
		resolver: net.DefaultResolver,
		timeout:  500 * time.Millisecond, // Strict timeout to prevent pipeline stalling
	}
}

func (d *DNSEnricher) Name() string {
	return "ReverseDNS"
}

func (d *DNSEnricher) Enrich(event *database.HostEvent) error {
	if event.SourceIP == "" || event.SourceIP == "127.0.0.1" || event.SourceIP == "::1" || event.SourceIP == "localhost" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	names, err := d.resolver.LookupAddr(ctx, event.SourceIP)
	if err == nil && len(names) > 0 {
		// Strip trailing dot from FQDN
		domain := strings.TrimSuffix(names[0], ".")

		// Map reverse DNS to the Hostname field if it's not already set
		// Or append it to location if Hostname is being used by the origin server name
		if event.Location == "" {
			event.Location = domain // Temporary mapping for UI visualization
		} else {
			event.Location = event.Location + " (" + domain + ")"
		}
	}

	return nil
}
