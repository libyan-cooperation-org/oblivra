package enrich

import (
	"net"

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

type GeoIPEnricher struct {
	cityDB *maxminddb.Reader
	asnDB  *maxminddb.Reader
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
	}, nil
}

func (g *GeoIPEnricher) Name() string {
	return "GeoIP"
}

func (g *GeoIPEnricher) Enrich(event *database.HostEvent) error {
	if event.SourceIP == "" || event.SourceIP == "127.0.0.1" || event.SourceIP == "::1" || event.SourceIP == "localhost" {
		return nil // Skip localnets implicitly
	}

	ip := net.ParseIP(event.SourceIP)
	if ip == nil {
		return nil
	}

	// Local private IPs won't resolve, but we'll try anyway safely
	if ip.IsPrivate() {
		event.Location = "Internal Network"
		return nil
	}

	if g.cityDB != nil {
		var record MaxMindRecord
		if err := g.cityDB.Lookup(ip, &record); err == nil && record.Country.IsoCode != "" {
			event.Location = record.Country.IsoCode
			if cityName := record.City.Names["en"]; cityName != "" {
				event.Location = cityName + ", " + event.Location
			}
		}
	}

	if g.asnDB != nil {
		var record MaxMindRecord
		if err := g.asnDB.Lookup(ip, &record); err == nil && record.Traits.AutonomousSystemOrganization != "" {
			// e.g. "AS15169 Google LLC"
			event.User = record.Traits.AutonomousSystemOrganization // Co-opt user field or we could add an ASN field to DB
			// NOTE: In a true SIEM schema we'd map this to `source.as.organization.name` ECS fields
		}
	}

	return nil
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
