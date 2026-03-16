package enrich

import (
	"net"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/oschwald/maxminddb-golang"
)

// MaxMindRecord outlines the expected schema from the GeoLite2 format
type MaxMindRecord struct {
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
	Traits struct {
		AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
		AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
	} `maxminddb:"traits"`
}

// GeoIPEnricher annotates events with geographic and ASN data.
// All lookups go through EnrichmentCache first — only cache misses hit the mmdb files.
// At 50k EPS with typical IP diversity this reduces mmdb reads by ~95%.
type GeoIPEnricher struct {
	cityDB *maxminddb.Reader
	asnDB  *maxminddb.Reader
	cache  *EnrichmentCache
}

func NewGeoIPEnricher(cityDbPath, asnDbPath string) (*GeoIPEnricher, error) {
	var city, asn *maxminddb.Reader
	var err error

	if cityDbPath != "" {
		city, err = maxminddb.Open(cityDbPath)
		if err != nil {
			return nil, err
		}
	}

	if asnDbPath != "" {
		asn, err = maxminddb.Open(asnDbPath)
		if err != nil {
			if city != nil {
				city.Close()
			}
			return nil, err
		}
	}

	return &GeoIPEnricher{
		cityDB: city,
		asnDB:  asn,
		// 50k IP cache, 10-minute TTL — covers a typical enterprise IP population
		// and prevents repeated disk lookups for the same threat actor IPs.
		cache: NewEnrichmentCache(50_000, 10*time.Minute),
	}, nil
}

func (g *GeoIPEnricher) Name() string { return "GeoIP" }

func (g *GeoIPEnricher) Enrich(event *database.HostEvent) error {
	if event.SourceIP == "" || event.SourceIP == "127.0.0.1" ||
		event.SourceIP == "::1" || event.SourceIP == "localhost" {
		return nil
	}

	ip := net.ParseIP(event.SourceIP)
	if ip == nil {
		return nil
	}

	if ip.IsPrivate() {
		event.Location = "Internal Network"
		return nil
	}

	// ── Cache lookup ──────────────────────────────────────────────────────────
	if location, asnOrg, _, ok := g.cache.Get(event.SourceIP); ok {
		event.Location = location
		if asnOrg != "" {
			event.User = asnOrg // ASN org mapped to user field per legacy schema
		}
		return nil
	}

	// ── Cache miss: query mmdb files ──────────────────────────────────────────
	var location, asnOrg, country string

	if g.cityDB != nil {
		var record MaxMindRecord
		if err := g.cityDB.Lookup(ip, &record); err == nil && record.Country.IsoCode != "" {
			country = record.Country.IsoCode
			location = country
			if cityName := record.City.Names["en"]; cityName != "" {
				location = cityName + ", " + country
			}
		}
	}

	if g.asnDB != nil {
		var record MaxMindRecord
		if err := g.asnDB.Lookup(ip, &record); err == nil {
			asnOrg = record.Traits.AutonomousSystemOrganization
		}
	}

	// Store in cache regardless of whether we got results —
	// avoids hammering mmdb for IPs not in the database.
	g.cache.Set(event.SourceIP, location, asnOrg, country)

	event.Location = location
	if asnOrg != "" {
		event.User = asnOrg
	}
	return nil
}

// CacheStats exposes hit/miss counters for the diagnostics panel.
func (g *GeoIPEnricher) CacheStats() (hits, misses uint64, size int) {
	return g.cache.Stats()
}

// Close cleans up mmdb readers
func (g *GeoIPEnricher) Close() {
	if g.cityDB != nil {
		g.cityDB.Close()
	}
	if g.asnDB != nil {
		g.asnDB.Close()
	}
}
