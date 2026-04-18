package threatintel

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/platform"
)

// Indicator represents a normalized IOC generated from TAXII, CSV, or Custom lists
type Indicator struct {
	Type        string    `json:"type"`  // ipv4-addr, ipv6-addr, domain-name, file:hashes.md5
	Value       string    `json:"value"` // 192.168.1.1, badguy.com, a1b2c3d4...
	Source      string    `json:"source"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	Description string    `json:"description"`
	CampaignID  string    `json:"campaign_id,omitempty"`
	ExpiresAt   string    `json:"expires_at"`
}

// Campaign represents a cluster of activity attributed to a common actor or goal.
type Campaign struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Actor       string   `json:"actor,omitempty"`
	TTPs        []string `json:"ttps,omitempty"`
	Description string   `json:"description,omitempty"`
}

// ParseOfflineCSV imports a simple flat-file CSV: type,value,severity,description
func ParseOfflineCSV(path string, source string) ([]Indicator, error) {
	safePath, err := platform.ValidateSafePath(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(safePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	var indicators []Indicator
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) < 2 {
			continue // Skip invalid rows
		}

		indType := strings.TrimSpace(record[0])
		indVal := strings.TrimSpace(record[1])
		severity := "medium"
		desc := "Offline Import"

		if len(record) >= 3 {
			severity = strings.ToLower(strings.TrimSpace(record[2]))
		}
		if len(record) >= 4 {
			desc = strings.TrimSpace(record[3])
		}

		indicators = append(indicators, Indicator{
			Type:        indType,
			Value:       indVal,
			Source:      source,
			Severity:    severity,
			Description: desc,
			ExpiresAt:   time.Now().AddDate(1, 0, 0).Format(time.RFC3339), // Expire in 1 year default
		})
	}

	return indicators, nil
}

// ConvertSTIX parses STIX Pattern syntax into direct Indicators
// NOTE: For Phase 3.1 we parse simple patterns like `[ipv4-addr:value = '198.51.100.1']`
func ConvertSTIX(bundle *Bundle, source string) ([]Indicator, error) {
	var results []Indicator

	for _, obj := range bundle.Objects {
		if obj.Type != "indicator" || obj.Pattern == "" {
			continue
		}

		// Barebones STIX 2.1 Pattern Parsing for MVP
		// e.g. [ipv4-addr:value = '10.0.0.1']
		pattern := strings.TrimSpace(obj.Pattern)
		if !strings.HasPrefix(pattern, "[") || !strings.HasSuffix(pattern, "]") {
			continue
		}

		inner := strings.Trim(pattern, "[]")
		parts := strings.SplitN(inner, "=", 2)
		if len(parts) != 2 {
			continue
		}

		lhs := strings.TrimSpace(parts[0]) // e.g. ipv4-addr:value
		rhs := strings.TrimSpace(parts[1]) // e.g. '10.0.0.1'

		// Strip quotes
		val := strings.Trim(rhs, "'\"")

		// Map type
		var iocType string
		if strings.Contains(lhs, "ipv4-addr") {
			iocType = "ipv4-addr"
		} else if strings.Contains(lhs, "ipv6-addr") {
			iocType = "ipv6-addr"
		} else if strings.Contains(lhs, "domain-name") {
			iocType = "domain-name"
		} else if strings.Contains(lhs, "file:hashes") {
			if strings.Contains(lhs, "md5") {
				iocType = "md5"
			} else if strings.Contains(lhs, "sha256") {
				iocType = "sha256"
			}
		}

		if iocType == "" {
			continue // Unsupported STIX type for MVP
		}

		results = append(results, Indicator{
			Type:        iocType,
			Value:       val,
			Source:      source,
			Severity:    "high", // Could map from obj.Labels but default high for now
			Description: obj.Name,
			ExpiresAt:   parseTime(obj.ValidFrom).AddDate(1, 0, 0).Format(time.RFC3339),
		})
	}

	return results, nil
}

func ParseOfflineSTIXFile(path string) ([]Indicator, error) {
	safePath, err := platform.ValidateSafePath(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(safePath)
	if err != nil {
		return nil, err
	}

	var bundle Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, err
	}

	return ConvertSTIX(&bundle, filepath.Base(path))
}

func parseTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return t
}
