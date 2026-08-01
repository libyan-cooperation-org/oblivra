// Package oql is OBLIVRA's pipe-syntax query language.
//
// Grammar (intentionally small):
//
//	query  = expr ( "|" stage )*
//	expr   = bleve query string  (passed through to Bleve unchanged)
//	stage  = where | limit | sort | head | tail | stats | top | timechart
//	where  = "where" field ":" value
//	limit  = "limit" N
//	sort   = "sort" ( "-" )? field         // - = descending
//	head   = "head" N                      // alias for limit
//	tail   = "tail" N                      // last N from the result set
//
//	// v2 aggregation stages — must be the last stage in the pipe:
//	stats     = "stats" fn ( "(" field ")" )? ( "by" field )?
//	fn        = "count" | "sum" | "avg" | "min" | "max" | "dc"
//	top       = "top" N field              // count by field, desc, first N
//	timechart = "timechart" "span=" dur "count" ( "by" field )?
//
// Examples:
//
//	severity:warning | limit 25
//	hostId:web-01 | where eventType:failed_login | sort -timestamp | head 10
//	*                | where severity:critical
//	eventType:failed_login | stats count by srcIP
//	*                | stats dc(user) by hostId
//	*                | top 10 srcIP
//	*                | timechart span=1h count by severity
//
// The parser produces a Plan that the SiemService can run by feeding the
// expr to the existing search path and applying stage filters in Go.
package oql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Plan is the compiled query.
type Plan struct {
	Expr      string // raw Bleve query — empty/"" means match-all
	Filters   []Filter
	SortField string
	SortDesc  bool
	Limit     int
	Tail      int

	// v2: at most one aggregation stage, always terminal.
	Agg *Agg
}

// AggFn enumerates the supported aggregation functions.
type AggFn string

const (
	AggCount AggFn = "count"
	AggSum   AggFn = "sum"
	AggAvg   AggFn = "avg"
	AggMin   AggFn = "min"
	AggMax   AggFn = "max"
	AggDC    AggFn = "dc" // distinct count
)

// Agg describes a terminal aggregation stage. Kind ∈ {stats, top, timechart}.
type Agg struct {
	Kind  string        // "stats" | "top" | "timechart"
	Fn    AggFn         // aggregation function (top/timechart imply count)
	Field string        // fn argument, e.g. sum(bytes) → "bytes"; empty for count
	By    string        // group-by field; empty = single-bucket
	TopN  int           // top only
	Span  time.Duration // timechart only
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
		if plan.Agg != nil {
			return Plan{}, fmt.Errorf("oql: %s must be the last stage", plan.Agg.Kind)
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

		case "stats":
			agg, err := parseStats(rest)
			if err != nil {
				return Plan{}, err
			}
			plan.Agg = agg

		case "top":
			agg, err := parseTop(rest)
			if err != nil {
				return Plan{}, err
			}
			plan.Agg = agg

		case "timechart":
			agg, err := parseTimechart(rest)
			if err != nil {
				return Plan{}, err
			}
			plan.Agg = agg

		default:
			return Plan{}, fmt.Errorf("oql: unknown stage %q", head)
		}
	}
	return plan, nil
}

// parseStats compiles "count by srcIP" / "sum(bytes) by hostId" / "dc(user)".
func parseStats(rest string) (*Agg, error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return nil, errors.New("oql: stats requires a function, e.g. 'stats count by hostId'")
	}
	fnPart := rest
	by := ""
	if i := indexWord(rest, "by"); i >= 0 {
		fnPart = strings.TrimSpace(rest[:i])
		by = strings.TrimSpace(rest[i+2:])
		if by == "" {
			return nil, errors.New("oql: stats 'by' requires a field")
		}
	}
	fn, field, err := parseFn(fnPart)
	if err != nil {
		return nil, err
	}
	return &Agg{Kind: "stats", Fn: fn, Field: field, By: by}, nil
}

// parseFn splits "sum(bytes)" → (sum, bytes); bare "count" → (count, "").
func parseFn(s string) (AggFn, string, error) {
	s = strings.TrimSpace(s)
	name, arg := s, ""
	if i := strings.IndexByte(s, '('); i >= 0 {
		if !strings.HasSuffix(s, ")") {
			return "", "", fmt.Errorf("oql: malformed function call %q", s)
		}
		name = strings.TrimSpace(s[:i])
		arg = strings.TrimSpace(s[i+1 : len(s)-1])
	}
	fn := AggFn(strings.ToLower(name))
	switch fn {
	case AggCount:
		// count() and count are both fine; arg optional.
		return AggCount, arg, nil
	case AggSum, AggAvg, AggMin, AggMax, AggDC:
		if arg == "" {
			return "", "", fmt.Errorf("oql: %s requires a field, e.g. %s(bytes)", fn, fn)
		}
		return fn, arg, nil
	default:
		return "", "", fmt.Errorf("oql: unknown stats function %q (want count/sum/avg/min/max/dc)", name)
	}
}

// parseTop compiles "10 srcIP" → count-by-srcIP, desc, first 10.
func parseTop(rest string) (*Agg, error) {
	fields := strings.Fields(rest)
	if len(fields) != 2 {
		return nil, fmt.Errorf("oql: top wants 'top N field', got %q", rest)
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil || n <= 0 {
		return nil, fmt.Errorf("oql: top wants positive int, got %q", fields[0])
	}
	return &Agg{Kind: "top", Fn: AggCount, By: fields[1], TopN: n}, nil
}

// parseTimechart compiles "span=1h count by severity".
func parseTimechart(rest string) (*Agg, error) {
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return nil, errors.New("oql: timechart wants 'span=<dur> count [by field]'")
	}
	agg := &Agg{Kind: "timechart", Fn: AggCount, Span: time.Hour}
	i := 0
	if strings.HasPrefix(fields[0], "span=") {
		d, err := time.ParseDuration(strings.TrimPrefix(fields[0], "span="))
		if err != nil || d <= 0 {
			return nil, fmt.Errorf("oql: bad timechart span %q", fields[0])
		}
		agg.Span = d
		i = 1
	}
	if i >= len(fields) || strings.ToLower(fields[i]) != "count" {
		return nil, errors.New("oql: timechart supports 'count' (optionally 'by field')")
	}
	i++
	if i < len(fields) {
		if strings.ToLower(fields[i]) != "by" || i+1 >= len(fields) {
			return nil, fmt.Errorf("oql: timechart trailing tokens %q (want 'by <field>')", strings.Join(fields[i:], " "))
		}
		agg.By = fields[i+1]
		if i+2 < len(fields) {
			return nil, fmt.Errorf("oql: timechart trailing tokens after by-field")
		}
	}
	return agg, nil
}

// indexWord finds standalone word w in s (space-delimited), -1 if absent.
func indexWord(s, w string) int {
	fields := strings.Fields(s)
	off := 0
	for _, f := range fields {
		i := strings.Index(s[off:], f)
		pos := off + i
		if strings.EqualFold(f, w) {
			return pos
		}
		off = pos + len(f)
	}
	return -1
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
