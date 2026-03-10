package ndr

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// TLSMetadataExtractor simulates JA3/JA3S fingerprinting for TLS traffic.
type TLSMetadataExtractor struct {
	bus *eventbus.Bus
	log *logger.Logger
}

func NewTLSMetadataExtractor(bus *eventbus.Bus, log *logger.Logger) *TLSMetadataExtractor {
	return &TLSMetadataExtractor{
		bus: bus,
		log: log.WithPrefix("ndr:tls"),
	}
}

// ProcessHandshake simulates extraction of TLS handshake parameters.
func (e *TLSMetadataExtractor) ProcessHandshake(host string, port int, version uint16, ciphers []uint16) {
	// Simulate JA3 string: Version,Ciphers,Extensions,EllipticCurves,EllipticCurvePoints
	ja3Raw := fmt.Sprintf("%d,%v,ext,curves,points", version, ciphers)
	hash := md5.Sum([]byte(ja3Raw))
	ja3 := hex.EncodeToString(hash[:])

	e.log.Debug("[NDR] JA3 Fingerprint for %s: %s", host, ja3)

	// Detection: Known malicious JA3 fingerprints (Simulated list)
	maliciousJA3 := map[string]string{
		"771,4866-4865-4867-49195-49199-49196-49200-158-159-52393-52392-49171-49172-107-103-57-51-136-135-47-53-10": "Metasploit/Cobalt Strike",
		"e7f04128038827727192772719277271": "Dridex Downloader",
	}

	for malHash, name := range maliciousJA3 {
		if ja3 == malHash || strings.Contains(ja3Raw, malHash) {
			aName := name
			if aName == "" {
				aName = "Unknown Malware"
			}

			e.log.Warn("⚠ Malicious TLS Fingerprint (%s) detected for %s", aName, host)
			e.bus.Publish("siem.alert_fired", map[string]interface{}{
				"type":        "NDR_MALICIOUS_TLS_FINGERPRINT",
				"severity":    "CRITICAL",
				"host":        host,
				"fingerprint": ja3,
				"description": fmt.Sprintf("TLS handshake matches known malicious fingerprint associated with %s.", aName),
			})
		}
	}
}
