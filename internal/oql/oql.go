// Package oql is OBLIVRA's pipe-syntax query language.
//
// Grammar (intentionally small):
//
//	query  = expr ( "|" stage )*
//	expr   = bleve query string  (passed through to Bleve unchanged)
//	stage  = where | limit | sort | head | tail
//	where  = "where" field ":" value
//	limit  = "limit" N
//	sort   = "sort" ( "-" )? field         // - = descending
//	head   = "head" N                      // alias for limit
//	tail   = "tail" N                      // last N from the result set
//
// Examples:
//
//	severity:warning | limit 25
//	hostId:web-01 | where eventType:failed_login | sort -timestamp | head 10
//	*                | where severity:critical
//
// The parser produces a Plan that the SiemService can run by feeding the
// expr to the existing search path and applying stage filters in Go.
package oql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Plan is the compiled query.
type Plan struct {
	Expr      string  // raw Bleve query — empty/"" means match-all
	Filters   []Filter
	SortField string
	SortDesc  bool
	Limit     int
	Tail      int
}

type Filter struct {
	Field string
	Value string
}

// Parse compiles an OQL string. Empty input → match-all plan.
func Parse(input string) (Plan, error) {
	plan := Plan{}
	input = strings.TrimSpace(input)
	if input == "" {
		return plan, nil
	}

	parts := splitPipe(input)
	plan.Expr = strings.TrimSpace(parts[0])
	if plan.Expr == "*" {
		plan.Expr = ""
	}

	for _, raw := range parts[1:] {
		stage := strings.TrimSpace(raw)
		if stage == "" {
			continue
		}
		head, rest := head1(stage)
		switch head {
		case "where":
			f, err := parseWhere(rest)
			if err != nil {
				return Plan{}, err
			}
			plan.Filters = append(plan.Filters, f)

		case "limit", "head":
			n, err := strconv.Atoi(strings.TrimSpace(rest))
			if err != nil || n <= 0 {
				return Plan{}, fmt.Errorf("oql: %s wants positive int, got %q", head, rest)
			}
			plan.Limit = n

		case "tail":
			n, err := strconv.Atoi(strings.TrimSpace(rest))
			if err != nil || n <= 0 {
				return Plan{}, fmt.Errorf("oql: tail wants positive int, got %q", rest)
			}
			plan.Tail = n

		case "sort":
			field := strings.TrimSpace(rest)
			if strings.HasPrefix(field, "-") {
				plan.SortDesc = true
				field = strings.TrimPrefix(field, "-")
				field = strings.TrimSpace(field)
			}
			if field == "" {
				return Plan{}, errors.New("oql: sort requires a field")
			}
			plan.SortField = field

		default:
			return Plan{}, fmt.Errorf("oql: unknown stage %q", head)
		}
	}
	return plan, nil
}

func parseWhere(rest string) (Filter, error) {
	rest = strings.TrimSpace(rest)
	idx := strings.IndexByte(rest, ':')
	if idx <= 0 {
		return Filter{}, fmt.Errorf("oql: where wants 'field:value', got %q", rest)
	}
	field := strings.TrimSpace(rest[:idx])
	value := strings.Trim(strings.TrimSpace(rest[idx+1:]), `"`)
	if field == "" || value == "" {
		return Filter{}, errors.New("oql: where field/value cannot be empty")
	}
	return Filter{Field: field, Value: value}, nil
}

// splitPipe respects "..." quoted segments so a pipe inside quotes isn't split.
func splitPipe(s string) []string {
	var out []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuote = !inQuote
		}
		if c == '|' && !inQuote {
			out = append(out, b.String())
			b.Reset()
			continue
		}
		b.WriteByte(c)
	}
	out = append(out, b.String())
	return out
}

// head1 splits "word rest" → ("word", "rest").
func head1(s string) (string, string) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i > 0 {
		return strings.ToLower(s[:i]), s[i+1:]
	}
	return strings.ToLower(s), ""
}
