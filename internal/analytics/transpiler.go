package analytics

import (
	"fmt"
	"regexp"
	"strings"
)

// Transpiler converts SovereignQL to SQLite SQL
type Transpiler struct{}

func NewTranspiler() *Transpiler {
	return &Transpiler{}
}

// ConvertLogQLToSQL turns `{host="web"}` |= "error" into SELECT * FROM terminal_logs WHERE...
func (t *Transpiler) ConvertLogQLToSQL(query string, limit, offset int) (string, []interface{}, error) {
	// 1. Base Query
	sql := "SELECT * FROM terminal_logs WHERE 1=1"
	var args []interface{}

	// 2. Parse Stream Selector: {key="value", ...}
	streamRegex := regexp.MustCompile(`\{([^}]+)\}`)
	match := streamRegex.FindStringSubmatch(query)

	if len(match) > 1 {
		selectors := strings.Split(match[1], ",")
		for _, s := range selectors {
			parts := strings.Split(s, "=")
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

				// Map LogQL keys to DB columns
				col := ""
				switch key {
				case "host":
					col = "host"
				case "session":
					col = "session_id"
				default:
					continue
				}

				if col != "" {
					sql += fmt.Sprintf(" AND %s = ?", col)
					args = append(args, val)
				}
			}
		}
	}

	// 3. Parse Filter Operators
	filterPart := query
	if len(match) > 0 {
		filterPart = strings.Replace(query, match[0], "", 1)
	}

	pipeParts := strings.Split(filterPart, "|")
	for _, p := range pipeParts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		if strings.HasPrefix(p, "=") {
			// Contains: |= "error"
			term := strings.Trim(strings.TrimPrefix(p, "="), ` "`)
			sql += " AND output LIKE ?"
			args = append(args, "%"+term+"%")
		} else if strings.HasPrefix(p, "!") {
			// Not Contains: != "info"
			term := strings.Trim(strings.TrimPrefix(p, "!"), ` "`)
			sql += " AND output NOT LIKE ?"
			args = append(args, "%"+term+"%")
		} else if strings.HasPrefix(p, "~") {
			// Pattern match: |~ "error" → LIKE '%error%'
			term := strings.Trim(strings.TrimPrefix(p, "~"), ` "`)
			sql += " AND output LIKE ?"
			args = append(args, "%"+term+"%")
		}
	}

	// 4. Sort and Limit
	sql += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT %d OFFSET %d", limit, offset)

	return sql, args, nil
}

// ConvertLuceneToSQL turns "host:web AND error" into SQL
func (t *Transpiler) ConvertLuceneToSQL(query string, limit, offset int) (string, []interface{}, error) {
	// Simple approximation to DuckDB equivalent with LIKE
	sql := "SELECT * FROM terminal_logs WHERE 1=1"
	var args []interface{}

	parts := strings.Split(query, " AND ")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Contains(p, ":") {
			kv := strings.SplitN(p, ":", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				val := strings.TrimSpace(kv[1])

				if key == "host" || key == "session_id" || key == "output" {
					sql += fmt.Sprintf(" AND %s LIKE ?", key)
					args = append(args, "%"+strings.ReplaceAll(val, "*", "")+"%")
				}
			}
		} else {
			sql += " AND output LIKE ?"
			args = append(args, "%"+strings.ReplaceAll(p, "*", "")+"%")
		}
	}

	sql += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT %d OFFSET %d", limit, offset)
	return sql, args, nil
}
