package oql

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ── Core Types ────────────────────────────────────────────────────────────────

// Row is the fundamental data unit in OQL — a map of field names to values.
type Row map[string]interface{}

// QueryResult is returned by Execute.
type QueryResult struct {
	Rows     []Row
	Meta     QueryMeta
	Profile  *QueryProfile
	Warnings []string
}

// QueryMeta carries execution metadata.
type QueryMeta struct {
	QueryID    string
	ParseTime  time.Duration
	ExecTime   time.Duration
	TotalRows  int64
	OutputRows int64
	Truncated  bool
	Warnings   []string
	Budget     *BudgetSnapshot
}

// QueryCost is an estimated cost for scheduling.
type QueryCost struct {
	EstimatedBytes  int64
	EstimatedEvents int64
	Complexity      float64
}

// ── Executor ──────────────────────────────────────────────────────────────────

// Executor is the main OQL execution engine.
type Executor struct {
	Resolver *SCIMResolver
	Checker  *TypeChecker
	Optim    *Optimizer
	History  *QueryHistoryDB
	Pool     *WorkerPool
	Source   DataSource
}

// NewExecutor creates a fully wired OQL executor.
func NewExecutor() *Executor {
	r := NewSCIMResolver()
	return &Executor{
		Resolver: r,
		Checker:  &TypeChecker{Resolver: r},
		Optim:    &Optimizer{Resolver: r},
		History:  NewQueryHistoryDB(),
		Pool:     NewWorkerPool(8, 2),
		Source:   &InMemSource{}, // Default to empty in-memory source
	}
}

func (ex *Executor) SetSource(s DataSource) {
	ex.Source = s
}

// defaultOQLTimeout is the maximum wall-clock time a single OQL query may run.
// Can be overridden at startup; 30 s is a safe production default.
const defaultOQLTimeout = 30 * time.Second

// Execute parses, type-checks, optimizes, and executes an OQL query.
// A 30-second deadline is applied if the caller's context has no deadline yet.
func (ex *Executor) Execute(ctx context.Context, input string, data []Row, ectx *EvalContext) (*QueryResult, error) {
	// AUD-04: Enforce a hard query timeout to prevent runaway executions.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultOQLTimeout)
		defer cancel()
	}

	start := time.Now()
	prof := NewQueryProfile()

	// Parse
	parseProf := prof.BeginStage("PARSE", 0, input)
	q, err := Parse(input, nil)
	if err != nil {
		parseProf.Finish()
		return nil, err
	}
	if ectx != nil && !ectx.TimeRange.Earliest.IsZero() {
		q.TimeRange = ectx.TimeRange
	}
	parseProf.Finish()
	parseTime := time.Since(start)

	// Type check
	tcProf := prof.BeginStage("TYPECHECK", 1, "")
	typeErrors := ex.Checker.Check(q)
	warnings := typeErrorsToStrings(typeErrors)
	tcProf.SetDetail("warnings", len(warnings))
	tcProf.Finish()

	// Optimize
	optProf := prof.BeginStage("OPTIMIZE", 2, "")
	optimized := ex.Optim.Optimize(q)
	optProf.SetDetail("commands_before", len(q.Commands))
	optProf.SetDetail("commands_after", len(optimized.Commands))
	optProf.Finish()

	// Execute
	meta := QueryMeta{ParseTime: parseTime, Warnings: warnings}
	if ectx != nil {
		meta.QueryID = ectx.QueryID
	}
	// Fetch data if not provided
	if data == nil && ex.Source != nil {
		fetchProf := prof.BeginStage("FETCH", 1, "")
		var err error
		data, err = ex.Source.Fetch(ctx, optimized.Search, optimized.TimeRange)
		if err != nil {
			fetchProf.Finish()
			return nil, fmt.Errorf("failed to fetch data: %w", err)
		}
		fetchProf.SetDetail("rows_fetched", len(data))
		fetchProf.Finish()
	}

	rows, err := ex.executePipeline(ctx, optimized, data, prof, &meta)
	if err != nil {
		return nil, err
	}
	meta.ExecTime = time.Since(start)
	meta.TotalRows = int64(len(data))
	meta.OutputRows = int64(len(rows))

	ex.History.RecordProfile(prof)

	return &QueryResult{Rows: rows, Meta: meta, Profile: prof, Warnings: warnings}, nil
}

func (ex *Executor) executePipeline(ctx context.Context, q *Query, data []Row, prof *QueryProfile, meta *QueryMeta) ([]Row, error) {
	// Apply search filter
	rows := data
	if q.Search != nil {
		sp := prof.BeginStage("SEARCH", 0, "")
		filtered := make([]Row, 0, len(rows)/2)
		for i, row := range rows {
			if cancelled(ctx, i) {
				sp.Finish()
				return nil, ctx.Err()
			}
			sp.TrackRowIn()
			if matchSearch(row, q.Search) {
				filtered = append(filtered, row)
				sp.TrackRowOut()
			}
		}
		rows = filtered
		sp.Finish()
	}

	for cmdIdx, cmd := range q.Commands {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		cp := prof.BeginStage(cmd.CommandName(), cmdIdx+10, "")
		var err error
		rows, err = ex.executeCommand(ctx, cmd, rows, cp, meta)
		cp.Finish()
		if err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (ex *Executor) executeCommand(ctx context.Context, cmd Command, rows []Row, prof *StageProfiler, _ *QueryMeta) ([]Row, error) {
	switch c := cmd.(type) {
	case *WhereCommand:
		return ex.execWhere(ctx, c, rows, prof)
	case *StatsCommand:
		return ex.execStats(ctx, c, rows, prof)
	case *EvalCommand:
		return ex.execEval(ctx, c, rows, prof)
	case *TableCommand:
		return ex.execTable(ctx, c, rows, prof)
	case *SortCommand:
		return ex.execSort(ctx, c, rows, prof)
	case *HeadCommand:
		return ex.execHead(c, rows, prof)
	case *TailCommand:
		return ex.execTail(c, rows, prof)
	case *DedupCommand:
		return ex.execDedup(ctx, c, rows, prof)
	case *RenameCommand:
		return ex.execRename(c, rows, prof)
	case *FieldsCommand:
		return ex.execFields(c, rows, prof)
	case *FillNullCommand:
		return ex.execFillNull(c, rows, prof)
	case *TopCommand:
		return ex.execTop(c, rows, prof)
	case *RareCommand:
		return ex.execRare(c, rows, prof)
	case *RexCommand:
		return ex.execRex(c, rows, prof)
	case *LookupCommand:
		return ex.execLookup(ctx, c, rows, prof)
	case *JoinCommand:
		return ex.execJoin(ctx, c, rows, prof)
	case *AppendCommand:
		return ex.execAppend(ctx, c, rows, prof)
	case *TimeChartCommand:
		return ex.execTimechart(ctx, c, rows, prof)
	case *ChartCommand:
		return ex.execChart(ctx, c, rows, prof)
	case *MvExpandCommand:
		return ex.execMvExpand(ctx, c, rows, prof)
	case *PredictCommand:
		return ex.execPredict(ctx, c, rows, prof)
	case *AnomalyDetectionCommand:
		return ex.execAnomalyDetection(ctx, c, rows, prof)
	default:
		return nil, fmt.Errorf("unsupported command: %s", cmd.CommandName())
	}
}

// ── Search Matching ───────────────────────────────────────────────────────────

func matchSearch(row Row, expr SearchExpr) bool {
	switch e := expr.(type) {
	case *AndExpr:
		return matchSearch(row, e.Left) && matchSearch(row, e.Right)
	case *OrExpr:
		return matchSearch(row, e.Left) || matchSearch(row, e.Right)
	case *NotExpr:
		return !matchSearch(row, e.Expr)
	case *CompareExpr:
		return matchCompare(row, e)
	case *FreeTextExpr:
		return matchFreeText(row, e.Text)
	case *FieldExistsExpr:
		_, ok := row[e.Field.Canonical()]
		if e.Exists {
			return ok
		}
		return !ok
	}
	return false
}

func matchCompare(row Row, e *CompareExpr) bool {
	val, ok := row[e.Field.Canonical()]
	if !ok {
		return e.Op == OpNeq || e.Op == OpNotIn
	}
	switch e.Op {
	case OpEq:
		return compareValues(val, e.Value) == 0
	case OpNeq:
		return compareValues(val, e.Value) != 0
	case OpGt:
		return compareValues(val, e.Value) > 0
	case OpGte:
		return compareValues(val, e.Value) >= 0
	case OpLt:
		return compareValues(val, e.Value) < 0
	case OpLte:
		return compareValues(val, e.Value) <= 0
	case OpIn:
		if lv, ok := e.Value.(ListValue); ok {
			for _, item := range lv.Items {
				if compareValues(val, item) == 0 {
					return true
				}
			}
		}
		return false
	case OpContains:
		return strings.Contains(strings.ToLower(fmt.Sprint(val)), strings.ToLower(valueStr(e.Value)))
	case OpStartsWith:
		return strings.HasPrefix(fmt.Sprint(val), valueStr(e.Value))
	case OpEndsWith:
		return strings.HasSuffix(fmt.Sprint(val), valueStr(e.Value))
	case OpLike:
		return matchLike(fmt.Sprint(val), valueStr(e.Value))
	case OpMatches:
		re, err := regexp.Compile(valueStr(e.Value))
		if err != nil {
			return false
		}
		return re.MatchString(fmt.Sprint(val))
	}
	return false
}

func matchFreeText(row Row, text string) bool {
	lower := strings.ToLower(text)
	for _, v := range row {
		if strings.Contains(strings.ToLower(fmt.Sprint(v)), lower) {
			return true
		}
	}
	return false
}

func matchLike(s, pattern string) bool {
	p := strings.ReplaceAll(pattern, "%", ".*")
	p = strings.ReplaceAll(p, "_", ".")
	re, err := regexp.Compile("^" + p + "$")
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

func compareValues(val interface{}, v Value) int {
	switch cv := v.(type) {
	case StringValue:
		return strings.Compare(fmt.Sprint(val), cv.V)
	case NumberValue:
		n, ok := ToNumber(val)
		if !ok {
			return strings.Compare(fmt.Sprint(val), fmt.Sprint(cv.V))
		}
		if n < cv.V {
			return -1
		}
		if n > cv.V {
			return 1
		}
		return 0
	case BoolValue:
		b, ok := val.(bool)
		if !ok {
			return -1
		}
		if b == cv.V {
			return 0
		}
		return 1
	case NullValue:
		if val == nil {
			return 0
		}
		return 1
	case WildcardValue:
		if matchWildcard(fmt.Sprint(val), cv.Pattern) {
			return 0
		}
		return 1
	}
	return strings.Compare(fmt.Sprint(val), fmt.Sprint(v))
}

func matchWildcard(s, pattern string) bool {
	p := strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*")
	p = strings.ReplaceAll(p, `\?`, ".")
	re, err := regexp.Compile("^(?i)" + p + "$")
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

func valueStr(v Value) string {
	switch cv := v.(type) {
	case StringValue:
		return cv.V
	case NumberValue:
		return fmt.Sprint(cv.V)
	case BoolValue:
		return fmt.Sprint(cv.V)
	case WildcardValue:
		return cv.Pattern
	}
	return fmt.Sprint(v)
}

// ── Command Executors ─────────────────────────────────────────────────────────

func (ex *Executor) execWhere(ctx context.Context, c *WhereCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	out := make([]Row, 0, len(rows)/2)
	for i, row := range rows {
		if cancelled(ctx, i) {
			return nil, ctx.Err()
		}
		prof.TrackRowIn()
		if matchSearch(row, c.Expr) {
			out = append(out, row)
			prof.TrackRowOut()
		}
	}
	return out, nil
}

func (ex *Executor) execStats(_ context.Context, c *StatsCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	groups := make(map[string]map[string]AggState)
	groupByFields := c.GroupBy
	for _, row := range rows {
		prof.TrackRowIn()
		key := buildGroupKey(row, groupByFields)
		states, exists := groups[key]
		if !exists {
			states = make(map[string]AggState, len(c.Aggregations))
			for _, a := range c.Aggregations {
				states[a.Alias] = NewAggState(a.Func)
			}
			groups[key] = states
		}
		for _, a := range c.Aggregations {
			states[a.Alias].Update(row, a.Field)
		}
	}
	out := make([]Row, 0, len(groups))
	for key, states := range groups {
		row := make(Row)
		for alias, state := range states {
			row[alias] = state.Finalize()
		}
		addGroupByFields(row, key, groupByFields)
		out = append(out, row)
		prof.TrackRowOut()
	}
	return out, nil
}

func (ex *Executor) execEval(ctx context.Context, c *EvalCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	for i, row := range rows {
		if cancelled(ctx, i) {
			return nil, ctx.Err()
		}
		prof.TrackRowIn()
		for _, assign := range c.Assignments {
			val := evaluateExpr(row, assign.Expr)
			row[assign.Field.Canonical()] = val
		}
		prof.TrackRowOut()
	}
	return rows, nil
}

func (ex *Executor) execTable(_ context.Context, c *TableCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	out := make([]Row, 0, len(rows))
	for _, row := range rows {
		prof.TrackRowIn()
		nr := make(Row, len(c.Fields))
		for _, f := range c.Fields {
			if v, ok := row[f.Canonical()]; ok {
				nr[f.Canonical()] = v
			}
		}
		out = append(out, nr)
		prof.TrackRowOut()
	}
	return out, nil
}

func (ex *Executor) execSort(_ context.Context, c *SortCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	for range rows {
		prof.TrackRowIn()
	}
	sortRows(rows, c.Specs)
	for range rows {
		prof.TrackRowOut()
	}
	return rows, nil
}

func (ex *Executor) execHead(c *HeadCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	n := c.Count
	if n > len(rows) {
		n = len(rows)
	}
	for i := 0; i < n; i++ {
		prof.TrackRowIn()
		prof.TrackRowOut()
	}
	return rows[:n], nil
}

func (ex *Executor) execTail(c *TailCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	n := c.Count
	if n > len(rows) {
		n = len(rows)
	}
	start := len(rows) - n
	for i := start; i < len(rows); i++ {
		prof.TrackRowIn()
		prof.TrackRowOut()
	}
	return rows[start:], nil
}

func (ex *Executor) execDedup(ctx context.Context, c *DedupCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	seen := make(map[string]int)
	out := make([]Row, 0, len(rows))
	for i, row := range rows {
		if cancelled(ctx, i) {
			return nil, ctx.Err()
		}
		prof.TrackRowIn()
		key := buildDedupKey(row, c.Fields)
		if seen[key] < c.Count {
			seen[key]++
			out = append(out, row)
			prof.TrackRowOut()
		}
	}
	return out, nil
}

func (ex *Executor) execRename(c *RenameCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	for _, row := range rows {
		prof.TrackRowIn()
		for _, r := range c.Renames {
			from := r.From.Canonical()
			if v, ok := row[from]; ok {
				row[r.To] = v
				delete(row, from)
			}
		}
		prof.TrackRowOut()
	}
	return rows, nil
}

func (ex *Executor) execFields(c *FieldsCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	fieldSet := make(map[string]bool, len(c.Fields))
	for _, f := range c.Fields {
		fieldSet[f.Canonical()] = true
	}
	if c.Mode == "exclude" {
		for _, row := range rows {
			prof.TrackRowIn()
			for k := range row {
				if fieldSet[k] {
					delete(row, k)
				}
			}
			prof.TrackRowOut()
		}
	} else {
		out := make([]Row, 0, len(rows))
		for _, row := range rows {
			prof.TrackRowIn()
			nr := make(Row)
			for k, v := range row {
				if fieldSet[k] {
					nr[k] = v
				}
			}
			out = append(out, nr)
			prof.TrackRowOut()
		}
		return out, nil
	}
	return rows, nil
}

func (ex *Executor) execFillNull(c *FillNullCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	fill := ""
	if c.Value != nil {
		fill = *c.Value
	}
	for _, row := range rows {
		prof.TrackRowIn()
		if len(c.Fields) > 0 {
			for _, f := range c.Fields {
				if _, ok := row[f.Canonical()]; !ok {
					row[f.Canonical()] = fill
				}
			}
		}
		prof.TrackRowOut()
	}
	return rows, nil
}

func (ex *Executor) execTop(c *TopCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	counts := make(map[string]int64)
	keyRows := make(map[string]Row)
	for _, row := range rows {
		prof.TrackRowIn()
		key := buildDedupKey(row, c.Fields)
		counts[key]++
		if _, exists := keyRows[key]; !exists {
			keyRows[key] = row
		}
	}
	type kv struct {
		key   string
		count int64
	}
	var items []kv
	for k, v := range counts {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].count > items[j].count })
	n := c.Count
	if n > len(items) {
		n = len(items)
	}
	out := make([]Row, 0, n)
	for i := 0; i < n; i++ {
		row := make(Row)
		for k, v := range keyRows[items[i].key] {
			row[k] = v
		}
		row["count"] = items[i].count
		out = append(out, row)
		prof.TrackRowOut()
	}
	return out, nil
}

func (ex *Executor) execRare(c *RareCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	counts := make(map[string]int64)
	keyRows := make(map[string]Row)
	for _, row := range rows {
		prof.TrackRowIn()
		key := buildDedupKey(row, c.Fields)
		counts[key]++
		if _, exists := keyRows[key]; !exists {
			keyRows[key] = row
		}
	}
	type kv struct {
		key   string
		count int64
	}
	var items []kv
	for k, v := range counts {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].count < items[j].count })
	n := c.Count
	if n > len(items) {
		n = len(items)
	}
	out := make([]Row, 0, n)
	for i := 0; i < n; i++ {
		row := make(Row)
		for k, v := range keyRows[items[i].key] {
			row[k] = v
		}
		row["count"] = items[i].count
		out = append(out, row)
		prof.TrackRowOut()
	}
	return out, nil
}

func (ex *Executor) execRex(c *RexCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	if c.Pattern == "" {
		return rows, nil
	}
	re, err := regexp.Compile(c.Pattern)
	if err != nil {
		return nil, fmt.Errorf("rex: invalid regex: %w", err)
	}
	names := re.SubexpNames()
	fieldName := "_raw"
	if c.Field != nil {
		fieldName = c.Field.Canonical()
	}
	for _, row := range rows {
		prof.TrackRowIn()
		input, ok := row[fieldName]
		if !ok {
			prof.TrackRowOut()
			continue
		}
		matches := re.FindStringSubmatch(fmt.Sprint(input))
		if matches != nil {
			for i, name := range names {
				if i > 0 && name != "" && i < len(matches) {
					row[name] = matches[i]
				}
			}
		}
		prof.TrackRowOut()
	}
	return rows, nil
}

// ── Eval Expression Evaluator ─────────────────────────────────────────────────

func evaluateExpr(row Row, expr EvalExpr) interface{} {
	switch e := expr.(type) {
	case *EvalLiteral:
		return e.Value
	case *EvalFieldRef:
		return row[e.Field.Canonical()]
	case *EvalFuncCall:
		return evaluateFunc(row, e)
	case *EvalBinaryOp:
		return evaluateBinaryOp(row, e)
	case *EvalUnaryOp:
		return evaluateUnaryOp(row, e)
	case *EvalTernary:
		cond := evaluateExpr(row, e.Cond)
		if toBool(cond) {
			return evaluateExpr(row, e.TrueVal)
		}
		return evaluateExpr(row, e.FalseVal)
	}
	return nil
}

func evaluateFunc(row Row, call *EvalFuncCall) interface{} {
	args := make([]interface{}, len(call.Args))
	for i, a := range call.Args {
		args[i] = evaluateExpr(row, a)
	}
	if fn, ok := BuiltinFuncs[call.Name]; ok {
		result, err := fn.Eval(nil, args)
		if err != nil {
			return nil
		}
		return result
	}
	return nil
}

func evaluateBinaryOp(row Row, e *EvalBinaryOp) interface{} {
	left := evaluateExpr(row, e.Left)
	right := evaluateExpr(row, e.Right)
	switch e.Op {
	case "+":
		ln, lok := ToNumber(left)
		rn, rok := ToNumber(right)
		if lok && rok {
			return ln + rn
		}
		return fmt.Sprint(left) + fmt.Sprint(right)
	case "-":
		ln, lok := ToNumber(left)
		rn, rok := ToNumber(right)
		if lok && rok {
			return ln - rn
		}
		return nil
	case "*":
		ln, lok := ToNumber(left)
		rn, rok := ToNumber(right)
		if lok && rok {
			return ln * rn
		}
		return nil
	case "/":
		ln, lok := ToNumber(left)
		rn, rok := ToNumber(right)
		if lok && rok && rn != 0 {
			return ln / rn
		}
		return nil
	case "%":
		ln, lok := ToNumber(left)
		rn, rok := ToNumber(right)
		if lok && rok && rn != 0 {
			return float64(int64(ln) % int64(rn))
		}
		return nil
	case "=", "==":
		return fmt.Sprint(left) == fmt.Sprint(right)
	case "!=":
		return fmt.Sprint(left) != fmt.Sprint(right)
	case ">":
		ln, lok := ToNumber(left)
		rn, rok := ToNumber(right)
		if lok && rok {
			return ln > rn
		}
		return fmt.Sprint(left) > fmt.Sprint(right)
	case ">=":
		ln, lok := ToNumber(left)
		rn, rok := ToNumber(right)
		if lok && rok {
			return ln >= rn
		}
		return fmt.Sprint(left) >= fmt.Sprint(right)
	case "<":
		ln, lok := ToNumber(left)
		rn, rok := ToNumber(right)
		if lok && rok {
			return ln < rn
		}
		return fmt.Sprint(left) < fmt.Sprint(right)
	case "<=":
		ln, lok := ToNumber(left)
		rn, rok := ToNumber(right)
		if lok && rok {
			return ln <= rn
		}
		return fmt.Sprint(left) <= fmt.Sprint(right)
	case "AND":
		return toBool(left) && toBool(right)
	case "OR":
		return toBool(left) || toBool(right)
	}
	return nil
}

func evaluateUnaryOp(row Row, e *EvalUnaryOp) interface{} {
	val := evaluateExpr(row, e.Expr)
	switch e.Op {
	case "-":
		if n, ok := ToNumber(val); ok {
			return -n
		}
		return nil
	case "NOT":
		return !toBool(val)
	}
	return nil
}

func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case int64:
		return b != 0
	case int:
		return b != 0
	case string:
		return b != "" && b != "0" && strings.ToLower(b) != "false"
	}
	return true
}

// ── Utility Functions ─────────────────────────────────────────────────────────

func buildGroupKey(row Row, fields []FieldRef) string {
	if len(fields) == 0 {
		return "_all"
	}
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = fmt.Sprint(row[f.Canonical()])
	}
	return strings.Join(parts, "\x00")
}

func buildDedupKey(row Row, fields []FieldRef) string {
	return buildGroupKey(row, fields)
}

func addGroupByFields(row Row, key string, fields []FieldRef) {
	if len(fields) == 0 || key == "_all" {
		return
	}
	parts := strings.Split(key, "\x00")
	for i, f := range fields {
		if i < len(parts) {
			row[f.Canonical()] = parts[i]
		}
	}
}

// EstimateRowBytes returns an approximate memory size for a row.
func EstimateRowBytes(row Row) int64 {
	size := int64(64 + len(row)*56)
	for k, v := range row {
		size += int64(len(k))
		switch val := v.(type) {
		case string:
			size += int64(len(val))
		case float64:
			size += 8
		case int64:
			size += 8
		case bool:
			size += 1
		default:
			size += 16
		}
	}
	return size
}

// sortRows sorts a slice of rows by the given sort specs.
func sortRows(rows []Row, specs []SortSpec) {
	if len(specs) == 0 || len(rows) == 0 {
		return
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return shouldSwap(rows[i], rows[j], specs)
	})
}

// shouldSwap returns true if row a should sort before row b.
func shouldSwap(a, b Row, specs []SortSpec) bool {
	for _, s := range specs {
		field := s.Field.Canonical()
		va, oa := a[field]
		vb, ob := b[field]
		if !oa && !ob {
			continue
		}
		if !oa {
			return !s.Descending
		}
		if !ob {
			return s.Descending
		}
		na, naOK := ToNumber(va)
		nb, nbOK := ToNumber(vb)
		if naOK && nbOK {
			if na != nb {
				if s.Descending {
					return na > nb
				}
				return na < nb
			}
			continue
		}
		sa := fmt.Sprint(va)
		sb := fmt.Sprint(vb)
		cmp := strings.Compare(sa, sb)
		if cmp != 0 {
			if s.Descending {
				return cmp > 0
			}
			return cmp < 0
		}
	}
	return false
}

// EstimateQueryCost estimates the cost of a parsed query for scheduling.
func EstimateQueryCost(q *Query, dataSize int64) QueryCost {
	c := QueryCost{EstimatedBytes: dataSize, EstimatedEvents: dataSize / 256}
	if q.Search != nil {
		c.Complexity += 1.0
	}
	for _, cmd := range q.Commands {
		switch cmd.(type) {
		case *StatsCommand:
			c.Complexity += 3.0
		case *SortCommand:
			c.Complexity += 2.0
		case *JoinCommand:
			c.Complexity += 5.0
		case *WhereCommand:
			c.Complexity += 0.5
		case *EvalCommand:
			c.Complexity += 1.0
		default:
			c.Complexity += 0.5
		}
	}
	return c
}

// ── Convenience Exports ───────────────────────────────────────────────────────

// ExecuteString is a convenience that creates an executor, parses, and runs.
func ExecuteString(ctx context.Context, query string, data []Row) (*QueryResult, error) {
	ex := NewExecutor()
	return ex.Execute(ctx, query, data, &EvalContext{
		Now:     time.Now(),
		QueryID: fmt.Sprintf("q-%d", time.Now().UnixNano()),
	})
}

// ValidateQuery parses and type-checks without executing.
func ValidateQuery(input string) ([]string, error) {
	q, err := Parse(input, nil)
	if err != nil {
		return nil, err
	}
	r := NewSCIMResolver()
	tc := &TypeChecker{Resolver: r}
	typeErrors := tc.Check(q)
	return typeErrorsToStrings(typeErrors), nil
}

// Autocomplete returns field name completions for a partial query.
func Autocomplete(partial string) []string {
	r := NewSCIMResolver()
	words := strings.Fields(partial)
	if len(words) == 0 {
		return nil
	}
	last := words[len(words)-1]
	suggestions := r.Autocomplete(last)
	result := make([]string, len(suggestions))
	for i, s := range suggestions {
		result[i] = s.Name
	}
	return result
}

// FormatRows returns a text table of rows.
func FormatRows(rows []Row, fields []string) string {
	if len(rows) == 0 {
		return "(no results)"
	}
	if len(fields) == 0 {
		fm := make(map[string]bool)
		for _, r := range rows {
			for k := range r {
				fm[k] = true
			}
		}
		for k := range fm {
			fields = append(fields, k)
		}
		sort.Strings(fields)
	}
	widths := make([]int, len(fields))
	for i, f := range fields {
		widths[i] = len(f)
	}
	strs := make([][]string, len(rows))
	for i, row := range rows {
		strs[i] = make([]string, len(fields))
		for j, f := range fields {
			s := ""
			if v, ok := row[f]; ok {
				s = fmt.Sprint(v)
			}
			strs[i][j] = s
			if len(s) > widths[j] {
				widths[j] = len(s)
			}
		}
	}
	for i := range widths {
		if widths[i] > 60 {
			widths[i] = 60
		}
	}
	var b strings.Builder
	for i, f := range fields {
		if i > 0 {
			b.WriteString("  ")
		}
		fmt.Fprintf(&b, "%-*s", widths[i], f)
	}
	b.WriteByte('\n')
	for i := range fields {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(strings.Repeat("─", widths[i]))
	}
	b.WriteByte('\n')
	for _, row := range strs {
		for i, s := range row {
			if i > 0 {
				b.WriteString("  ")
			}
			if len(s) > widths[i] {
				s = s[:widths[i]-1] + "…"
			}
			fmt.Fprintf(&b, "%-*s", widths[i], s)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func (ex *Executor) execLookup(_ context.Context, c *LookupCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	for _, row := range rows {
		prof.TrackRowIn()
		// Mock lookup for now, Phase 20.1 will integrate real LookupDB
		if key, ok := row[c.MatchField.Canonical()]; ok {
			if c.AsField != nil {
				row[c.AsField.Canonical()] = key
			}
		}
		prof.TrackRowOut()
	}
	return rows, nil
}

func (ex *Executor) execJoin(ctx context.Context, c *JoinCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	// Execute subquery
	subRows, err := ex.Execute(ctx, c.Subquery.String(), nil, nil)
	if err != nil {
		return nil, err
	}
	
	subMap := make(map[string]Row)
	for _, sr := range subRows.Rows {
		if kv, ok := sr[c.Field.Canonical()]; ok {
			subMap[fmt.Sprint(kv)] = sr
		}
	}

	out := make([]Row, 0, len(rows))
	for _, row := range rows {
		prof.TrackRowIn()
		kv := fmt.Sprint(row[c.Field.Canonical()])
		if sr, ok := subMap[kv]; ok {
			// Merge
			newRow := make(Row)
			for k, v := range row { newRow[k] = v }
			for k, v := range sr { newRow[k] = v }
			out = append(out, newRow)
			prof.TrackRowOut()
		} else if c.Type == "left" || c.Type == "outer" {
			out = append(out, row)
			prof.TrackRowOut()
		}
	}
	return out, nil
}

func (ex *Executor) execAppend(ctx context.Context, c *AppendCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	subRows, err := ex.Execute(ctx, c.Subquery.String(), nil, nil)
	if err != nil {
		return nil, err
	}
	for _, r := range subRows.Rows {
		prof.TrackRowIn()
		rows = append(rows, r)
		prof.TrackRowOut()
	}
	return rows, nil
}

func (ex *Executor) execTimechart(_ context.Context, c *TimeChartCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	span := time.Minute * 10
	if c.Span != nil { span = *c.Span }
	
	buckets := make(map[int64]map[string]map[string]AggState)
	for _, row := range rows {
		prof.TrackRowIn()
		ts, ok := row["timestamp"].(string)
		if !ok { continue }
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil { continue }
		
		bt := t.Truncate(span).Unix()
		if buckets[bt] == nil { buckets[bt] = make(map[string]map[string]AggState) }
		
		splitKey := "all"
		if c.SplitBy != nil {
			splitKey = fmt.Sprint(row[c.SplitBy.Canonical()])
		}
		
		if buckets[bt][splitKey] == nil {
			buckets[bt][splitKey] = make(map[string]AggState)
			for _, a := range c.Aggregations {
				buckets[bt][splitKey][a.Alias] = NewAggState(a.Func)
			}
		}
		
		for _, a := range c.Aggregations {
			buckets[bt][splitKey][a.Alias].Update(row, a.Field)
		}
	}
	
	var times []int64
	for t := range buckets { times = append(times, t) }
	sort.Slice(times, func(a, b int) bool { return times[a] < times[b] })
	
	out := make([]Row, 0, len(times))
	for _, t := range times {
		row := Row{"_time": time.Unix(t, 0).Format(time.RFC3339)}
		for splitKey, aggs := range buckets[t] {
			for alias, state := range aggs {
				key := alias
				if splitKey != "all" { key = splitKey + ":" + alias }
				row[key] = state.Finalize()
			}
		}
		out = append(out, row)
		prof.TrackRowOut()
	}
	return out, nil
}

func (ex *Executor) execChart(_ context.Context, c *ChartCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	// Simple chart by over/by
	groups := make(map[string]map[string]map[string]AggState)
	for _, row := range rows {
		prof.TrackRowIn()
		overKey := fmt.Sprint(row[c.Over.Canonical()])
		byKey := "all"
		if c.By != nil { byKey = fmt.Sprint(row[c.By.Canonical()]) }
		
		if groups[overKey] == nil { groups[overKey] = make(map[string]map[string]AggState) }
		if groups[overKey][byKey] == nil {
			groups[overKey][byKey] = make(map[string]AggState)
			for _, a := range c.Aggregations {
				groups[overKey][byKey][a.Alias] = NewAggState(a.Func)
			}
		}
		
		for _, a := range c.Aggregations {
			groups[overKey][byKey][a.Alias].Update(row, a.Field)
		}
	}
	
	out := make([]Row, 0, len(groups))
	for overKey, byMap := range groups {
		row := Row{c.Over.Canonical(): overKey}
		for byKey, aggs := range byMap {
			for alias, state := range aggs {
				key := alias
				if byKey != "all" { key = byKey + ":" + alias }
				row[key] = state.Finalize()
			}
		}
		out = append(out, row)
		prof.TrackRowOut()
	}
	return out, nil
}

func (ex *Executor) execMvExpand(_ context.Context, c *MvExpandCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	out := make([]Row, 0, len(rows))
	for _, row := range rows {
		prof.TrackRowIn()
		val := row[c.Field.Canonical()]
		if list, ok := val.([]interface{}); ok {
			for _, item := range list {
				newRow := make(Row)
				for k, v := range row { newRow[k] = v }
				newRow[c.Field.Canonical()] = item
				out = append(out, newRow)
				prof.TrackRowOut()
			}
		} else {
			out = append(out, row)
			prof.TrackRowOut()
		}
	}
	return out, nil
}

func (ex *Executor) execPredict(ctx context.Context, c *PredictCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	if len(rows) < 2 {
		return rows, nil
	}

	for range rows { prof.TrackRowIn(); prof.TrackRowOut() }

	field := c.Field.Canonical()
	
	// Linear Regression: y = a + bx
	var sumX, sumY, sumXY, sumXX float64
	n := float64(len(rows))

	vals := make([]float64, len(rows))
	for i, row := range rows {
		val, _ := ToNumber(row[field])
		vals[i] = val
		x := float64(i)
		sumX += x
		sumY += val
		sumXY += x * val
		sumXX += x * x
	}

	b := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	a := (sumY - b*sumX) / n

	// Predict next N points
	future := 5
	if c.Future > 0 {
		future = c.Future
	}

	for i := 0; i < future; i++ {
		x := n + float64(i)
		predictY := a + b*x
		
		newRow := make(Row)
		for k, v := range rows[len(rows)-1] { newRow[k] = v }
		newRow[field] = predictY
		newRow["is_predicted"] = true
		rows = append(rows, newRow)
	}

	return rows, nil
}

// interpolatedQuantile returns the p-th quantile of a sorted slice using
// linear interpolation (NIST recommended method). p must be in [0, 1].
func interpolatedQuantile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	h := p * float64(n-1)
	lo := int(h)
	hi := lo + 1
	if hi >= n {
		return sorted[n-1]
	}
	return sorted[lo] + (h-float64(lo))*(sorted[hi]-sorted[lo])
}

// fieldAnomalyStats holds pre-computed statistics for a single field.
type fieldAnomalyStats struct {
	name  string
	mean  float64
	stdDev float64
	q1    float64
	q3    float64
	iqr   float64
	lower float64
	upper float64
}

// computeFieldStats derives IQR and Z-score parameters for a field.
func computeFieldStats(name string, vals []float64) fieldAnomalyStats {
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)

	q1 := interpolatedQuantile(sorted, 0.25)
	q3 := interpolatedQuantile(sorted, 0.75)
	iqr := q3 - q1

	var sum float64
	for _, v := range vals {
		sum += v
	}
	mean := sum / float64(len(vals))

	var variance float64
	for _, v := range vals {
		diff := v - mean
		variance += diff * diff
	}
	stdDev := 0.0
	if len(vals) > 1 {
		stdDev = variance / float64(len(vals)-1) // sample variance
		if stdDev > 0 {
			// Use math.Sqrt equivalent via Newton's method to avoid importing math
			x := stdDev
			for i := 0; i < 50; i++ {
				x = (x + stdDev/x) / 2
			}
			stdDev = x
		}
	}

	return fieldAnomalyStats{
		name:   name,
		mean:   mean,
		stdDev: stdDev,
		q1:     q1,
		q3:     q3,
		iqr:    iqr,
		lower:  q1 - 1.5*iqr,
		upper:  q3 + 1.5*iqr,
	}
}

// clamp restricts a float64 to [min, max].
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// execAnomalyDetection implements multi-field behavioral anomaly detection.
// AUD-02: Replaces the previous single-field approximation with:
//   - Interpolated Tukey quartiles (not index-based)
//   - Per-field is_anomaly_<field> tagging
//   - Normalized anomaly score clamped to [-1, +1]
//   - Z-score method support (method=zscore)
func (ex *Executor) execAnomalyDetection(_ context.Context, c *AnomalyDetectionCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	if len(rows) < 4 || len(c.Fields) == 0 {
		return rows, nil
	}

	useZScore := strings.EqualFold(c.Method, "zscore")

	// Phase 1: collect numeric values per field
	fieldVals := make(map[string][]float64, len(c.Fields))
	for _, f := range c.Fields {
		name := f.Canonical()
		fieldVals[name] = make([]float64, 0, len(rows))
		for _, row := range rows {
			if v, ok := ToNumber(row[name]); ok {
				fieldVals[name] = append(fieldVals[name], v)
			}
		}
	}

	// Phase 2: compute per-field statistics
	statsByField := make(map[string]fieldAnomalyStats, len(c.Fields))
	for _, f := range c.Fields {
		name := f.Canonical()
		vals := fieldVals[name]
		if len(vals) < 4 {
			continue
		}
		statsByField[name] = computeFieldStats(name, vals)
	}

	if len(statsByField) == 0 {
		return rows, nil
	}

	// Phase 3: annotate each row
	for _, row := range rows {
		prof.TrackRowIn()
		var totalScore float64
		var scoredFields int
		rowIsAnomaly := false

		for _, f := range c.Fields {
			name := f.Canonical()
			stats, hasStats := statsByField[name]
			if !hasStats {
				continue
			}
			v, ok := ToNumber(row[name])
			if !ok {
				// Missing field: tag as indeterminate
				row["is_anomaly_"+name] = false
				continue
			}

			var fieldAnomaly bool
			var score float64

			if useZScore {
				if stats.stdDev > 0 {
					z := (v - stats.mean) / stats.stdDev
					fieldAnomaly = z > 3.0 || z < -3.0 // 3-sigma rule
					score = clamp(z/3.0, -1.0, 1.0)
				}
			} else {
				// IQR Tukey method
				fieldAnomaly = v < stats.lower || v > stats.upper
				fence := stats.iqr + 0.00001
				rawScore := (v - stats.q1) / fence
				score = clamp(rawScore/3.0, -1.0, 1.0) // normalize to [-1,1]
			}

			row["is_anomaly_"+name] = fieldAnomaly
			row["anomaly_score_"+name] = score
			if fieldAnomaly {
				rowIsAnomaly = true
			}
			totalScore += score
			scoredFields++
		}

		// Composite tags
		row["is_anomaly"] = rowIsAnomaly
		if scoredFields > 0 {
			row["anomaly_score"] = clamp(totalScore/float64(scoredFields), -1.0, 1.0)
		} else {
			row["anomaly_score"] = 0.0
		}
		prof.TrackRowOut()
	}

	return rows, nil
}

func typeErrorsToStrings(errs []TypeError) []string {
	if len(errs) == 0 {
		return nil
	}
	out := make([]string, len(errs))
	for i, e := range errs {
		msg := e.Message
		if e.Hint != "" {
			msg += " (hint: " + e.Hint + ")"
		}
		out[i] = msg
	}
	return out
}

var _ = strconv.Itoa
