package transpiler

import (
	"fmt"
	"strings"
)

// SigmaToOQL converts basic Sigma rule detection to OQL.
func SigmaToOQL(sigma map[string]interface{}) string {
	// Simple Sigma to OQL transformation
	var parts []string
	if selection, ok := sigma["selection"].(map[string]interface{}); ok {
		for k, v := range selection {
			parts = append(parts, fmt.Sprintf("%s=%q", k, v))
		}
	}
	return strings.Join(parts, " AND ")
}
