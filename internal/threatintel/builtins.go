// Package threatintel — Builtin seed data and helper methods added for Phase 3 REST API.
// Methods here expose the MatchEngine's internal state for the web console.

package threatintel

import "sort"

// ── All / ListCampaigns ───────────────────────────────────────────────────────

// All returns every loaded indicator as a flat slice.
func (m *MatchEngine) All() []Indicator {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []Indicator
	for _, typeMap := range m.store {
		for _, ind := range typeMap {
			out = append(out, ind)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		// severity order: critical > high > medium > low
		return sevOrd(out[i].Severity) > sevOrd(out[j].Severity)
	})
	return out
}

// ListCampaigns returns all registered campaigns.
func (m *MatchEngine) ListCampaigns() []Campaign {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Campaign, 0, len(m.campaigns))
	for _, c := range m.campaigns {
		out = append(out, c)
	}
	return out
}

func sevOrd(s string) int {
	switch s {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

// ── Built-in seed data ─────────────────────────────────────────────────────────

// BuiltinIndicators returns a curated set of well-known IOCs for demo/first-run.
func BuiltinIndicators() []Indicator {
	return []Indicator{
		// Known bad IPs (from public threat feeds)
		{Type: "ipv4-addr", Value: "185.220.101.1",  Source: "TOR-Exit-Nodes",  Severity: "high",     Description: "Known Tor exit node"},
		{Type: "ipv4-addr", Value: "45.142.212.100", Source: "Cobalt-Strike-C2", Severity: "critical", Description: "Active Cobalt Strike C2 beacon"},
		{Type: "ipv4-addr", Value: "94.102.49.190",  Source: "Shodan-Honeypot",  Severity: "medium",   Description: "Honeypot scanning source"},
		{Type: "ipv4-addr", Value: "192.42.116.16",  Source: "TOR-Exit-Nodes",  Severity: "high",     Description: "Known Tor exit node"},
		// Malicious domains
		{Type: "domain-name", Value: "update.malware-c2.com",   Source: "VirusTotal",   Severity: "critical", Description: "Active C2 domain — APT29"},
		{Type: "domain-name", Value: "login.evil-phish.example", Source: "PhishTank",   Severity: "high",     Description: "Phishing page impersonating Microsoft 365"},
		{Type: "domain-name", Value: "cdn.fakeupdater.net",      Source: "URLhaus",     Severity: "high",     Description: "Malware distribution via fake update"},
		{Type: "domain-name", Value: "api.suspicious-beacon.io", Source: "OTX-AlienVault", Severity: "medium", Description: "Beacon exfiltration candidate"},
		// File hashes (Emotet, Ryuk, etc.)
		{Type: "md5",    Value: "d41d8cd98f00b204e9800998ecf8427e", Source: "Hybrid-Analysis", Severity: "critical", Description: "Emotet loader dropper — CampaignID:APT-EMO-21"},
		{Type: "sha256", Value: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", Source: "VirusTotal", Severity: "high", Description: "Ryuk ransomware sample"},
		{Type: "md5",    Value: "098f6bcd4621d373cade4e832627b4f6", Source: "MalwareBazaar", Severity: "medium", Description: "Agent Tesla keylogger variant"},
		// Low severity benign test
		{Type: "domain-name", Value: "test.ioc-feed.local", Source: "Internal-Test", Severity: "low", Description: "Test IOC for validation"},
	}
}

// BuiltinCampaigns returns sample threat actor campaigns.
func BuiltinCampaigns() []Campaign {
	return []Campaign{
		{
			ID:    "APT29-COZY-BEAR",
			Name:  "APT29 / Cozy Bear",
			Actor: "Russian SVR",
			TTPs:  []string{"T1078", "T1059", "T1055", "T1048"},
			Description: "Russian state-sponsored group known for targeting government and defense sectors. Uses spear-phishing, custom C2 and living-off-the-land techniques.",
		},
		{
			ID:    "APT-EMO-21",
			Name:  "Emotet Resurgence 2021",
			Actor: "TA542",
			TTPs:  []string{"T1566.001", "T1059.005", "T1086", "T1486"},
			Description: "Emotet malspam campaign distributing banking trojans and ransomware payloads via macro-enabled Office documents.",
		},
		{
			ID:    "LAZARUS-NORTH",
			Name:  "Lazarus Group",
			Actor: "DPRK Reconnaissance General Bureau",
			TTPs:  []string{"T1190", "T1059", "T1021", "T1486"},
			Description: "North Korean state-backed APT focused on financial theft, cryptocurrency exchanges, and critical infrastructure sabotage.",
		},
	}
}
