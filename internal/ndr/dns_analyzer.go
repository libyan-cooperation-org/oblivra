package ndr

import (
	"math"
	"strings"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// DNSAnalyzer inspects DNS queries for malicious patterns.
type DNSAnalyzer struct {
	bus *eventbus.Bus
	log *logger.Logger
}

func NewDNSAnalyzer(bus *eventbus.Bus, log *logger.Logger) *DNSAnalyzer {
	return &DNSAnalyzer{
		bus: bus,
		log: log.WithPrefix("ndr:dns"),
	}
}

// ProcessQuery evaluates a DNS query and its response.
func (a *DNSAnalyzer) ProcessQuery(query, answer string) {
	// 1. DGA Detection via Entropy
	if a.isHighEntropy(query) {
		a.log.Warn("⚠ DGA detected: %s", query)
		a.bus.Publish("siem.alert_fired", map[string]interface{}{
			"type":        "NDR_DGA_DETECTED",
			"severity":    "HIGH",
			"query":       query,
			"description": "Domain name exhibits high-entropy patterns characteristic of generation algorithms.",
		})
	}

	// 2. Tunneling Detection via Payload Size
	if len(query) > 128 || len(answer) > 256 {
		a.log.Warn("⚠ Potential DNS Tunneling: %s (len: %d)", query, len(query))
		a.bus.Publish("siem.alert_fired", map[string]interface{}{
			"type":        "NDR_DNS_TUNNELING",
			"severity":    "CRITICAL",
			"query":       query,
			"description": "Large DNS payload detected, potential exfiltration or C2 tunneling.",
		})
	}
}

// isHighEntropy calculates Shannon entropy. Malicious DGA domains usually score > 3.5.
func (a *DNSAnalyzer) isHighEntropy(s string) bool {
	// Remove common TLDs and dots to focus on the generated string
	s = strings.ToLower(s)
	idx := strings.LastIndex(s, ".")
	if idx != -1 {
		s = s[:idx]
	}

	if len(s) < 10 {
		return false // Short domains are rarely DGA
	}

	counts := make(map[rune]int)
	for _, r := range s {
		counts[r]++
	}

	var entropy float64
	for _, count := range counts {
		p := float64(count) / float64(len(s))
		entropy -= p * math.Log2(p)
	}

	return entropy > 3.8
}
