package transpiler

import (
	"strings"
)

// SPLToOQL converts basic Splunk SPL to OQL.
func SPLToOQL(spl string) string {
	// Simple rule-based transpilation
	oql := spl
	// Replace Splunk-specific commands or syntax if needed
	// For now, SPL and OQL are intentionally similar (pipe-based)
	oql = strings.ReplaceAll(oql, "index=", "source=")
	oql = strings.ReplaceAll(oql, "sourcetype=", "type=")
	return oql
}
