package intelligence

import (
	"fmt"
)

// Evidence represents a single piece of metadata that explains an anomaly.
type Evidence struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Threshold   interface{} `json:"threshold,omitempty"`
	Description string      `json:"description"`
}

// ExplainAnomaly formats a list of evidence into a concise human-readable string.
func ExplainAnomaly(evidence []Evidence) string {
	if len(evidence) == 0 {
		return "No specific evidence provided."
	}

	summary := ""
	for i, e := range evidence {
		if i > 0 {
			summary += "; "
		}
		summary += fmt.Sprintf("%s: %v", e.Key, e.Value)
		if e.Threshold != nil {
			summary += fmt.Sprintf(" (threshold: %v)", e.Threshold)
		}
	}
	return summary
}
