package transpiler

import (
	"strings"
)

// KQLToOQL converts basic Microsoft KQL to OQL.
func KQLToOQL(kql string) string {
	// KQL uses | filter but OQL uses | where (standardized on SPL-style pipes)
	oql := kql
	oql = strings.ReplaceAll(oql, "| where", "| where") // same
	oql = strings.ReplaceAll(oql, "| summarize", "| stats")
	oql = strings.ReplaceAll(oql, "| project", "| table")
	oql = strings.ReplaceAll(oql, "| extend", "| eval")
	oql = strings.ReplaceAll(oql, "| limit", "| head")
	return oql
}
