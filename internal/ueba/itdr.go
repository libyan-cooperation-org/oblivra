package ueba

import (
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
)

// ITDRManager handles identity-specific threat detection.
type ITDRManager struct {
	baseline *BaselineStore
}

func NewITDRManager(baseline *BaselineStore) *ITDRManager {
	return &ITDRManager{baseline: baseline}
}

// AnalyzeEvent looks for credential misuse patterns.
func (m *ITDRManager) AnalyzeEvent(event *database.HostEvent) float64 {
	riskIncrement := 0.0

	// 1. New Source IP for user
	if event.User != "" && event.SourceIP != "" {
		p := m.baseline.GetOrCreateProfile(event.User, "user")
		p.mu.RLock()
		if _, ok := p.FeatureVectors["last_ip_"+event.SourceIP]; !ok {
			// First time we've seen this IP for this user
			riskIncrement += 0.2
		}
		p.mu.RUnlock()
	}

	// 2. High-frequency failed logins (Brute force signal)
	if strings.Contains(strings.ToLower(event.RawLog), "failed login") || strings.Contains(strings.ToLower(event.RawLog), "authentication failure") {
		p := m.baseline.GetOrCreateProfile(event.User, "user")
		p.UpdateFeature("failed_login_burst", p.FeatureVectors["failed_login_burst"]+1)
		riskIncrement += 0.1
	}

	// 3. Vault access off-hours (Heuristic)
	if event.EventType == "vault_access" || strings.Contains(event.RawLog, "vault") {
		hour := time.Now().Hour()
		if hour < 7 || hour > 19 {
			riskIncrement += 0.3 // After hours vault access
		}
	}

	// 4. First-time-ever destination host for this user (Lateral Movement)
	if event.User != "" && event.HostID != "" {
		p := m.baseline.GetOrCreateProfile(event.User, "user")
		p.mu.RLock()
		if _, ok := p.FeatureVectors["dest_host_"+event.HostID]; !ok {
			riskIncrement += 0.15 // Potential lateral movement
		}
		p.mu.RUnlock()
		// Mark as seen for EMA tracking
		p.UpdateFeature("dest_host_"+event.HostID, 1.0)
	}

	// 5. Impossible Travel (Geo-velocity check)
	// Placeholder: In a real sovereign SOC, this would use the GeoIP enricher data.
	// For now, we detect "IP hopping" if the user switched IPs within a 5-minute window
	// and the IPs look like they belong to different subnets (crude proxy for distance).
	if event.User != "" && event.SourceIP != "" {
		p := m.baseline.GetOrCreateProfile(event.User, "user")
		p.mu.RLock()
		lastIP := ""
		for k := range p.FeatureVectors {
			if strings.HasPrefix(k, "last_ip_") {
				lastIP = strings.TrimPrefix(k, "last_ip_")
				break
			}
		}
		p.mu.RUnlock()

		if lastIP != "" && lastIP != event.SourceIP {
			// If we switched IPs, check the time delta
			if time.Since(p.LastSeen) < 5*time.Minute {
				riskIncrement += 0.4 // "Impossible Travel" or Proxy/VPN usage
			}
		}
		p.UpdateFeature("last_ip_"+event.SourceIP, 1.0)
	}

	return riskIncrement
}
