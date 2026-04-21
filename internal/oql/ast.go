package oql

import (
	"fmt"
	"strings"
	"time"
)

type Query struct {
	Version   int
	Search    SearchExpr
	Commands  []Command
	TimeRange TimeRange
	Macros    map[string]MacroDef
}

func (q *Query) String() string {
	var b strings.Builder
	if q.Search != nil {
		b.WriteString(q.Search.String())
	}
	for _, cmd := range q.Commands {
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(cmd.CommandName())
		// This is a simplified String(), but enough for now. 
		// Real implementation should serialize command arguments.
	}
	return b.String()
}

type TimeRange struct {
	Earliest time.Time
	Latest   time.Time
	Span     time.Duration
}

type MacroDef struct {
	Name string
	Args []string
	Body string
}

type SearchExpr interface {
	searchExpr()
	String() string
}

type AndExpr struct{ Left, Right SearchExpr }
type OrExpr struct{ Left, Right SearchExpr }
type NotExpr struct{ Expr SearchExpr }
type CompareExpr struct {
	Field FieldRef
	Op    CompareOp
	Value Value
}
type FreeTextExpr struct{ Text string }
type FieldExistsExpr struct {
	Field  FieldRef
	Exists bool
}
type SubqueryExpr struct{ Query *Query }
type ConstantSearchExpr struct{ Val bool }

func (*AndExpr) searchExpr()         {}
func (*OrExpr) searchExpr()          {}
func (*NotExpr) searchExpr()         {}
func (*CompareExpr) searchExpr()     {}
func (*FreeTextExpr) searchExpr()    {}
func (*FieldExistsExpr) searchExpr() {}
func (*SubqueryExpr) searchExpr() {}
func (*ConstantSearchExpr) searchExpr() {}

func (e *AndExpr) String() string         { return fmt.Sprintf("(%s AND %s)", e.Left, e.Right) }
func (e *OrExpr) String() string          { return fmt.Sprintf("(%s OR %s)", e.Left, e.Right) }
func (e *NotExpr) String() string         { return fmt.Sprintf("NOT %s", e.Expr) }
func (e *CompareExpr) String() string     { return fmt.Sprintf("%s %s %s", e.Field, e.Op, e.Value) }
func (e *FreeTextExpr) String() string    { return fmt.Sprintf("%q", e.Text) }
func (e *FieldExistsExpr) String() string { return fmt.Sprintf("%s EXISTS=%v", e.Field, e.Exists) }
func (e *SubqueryExpr) String() string    { return fmt.Sprintf("[ %s ]", e.Query) }
func (e *ConstantSearchExpr) String() string { return fmt.Sprint(e.Val) }

type FieldRef struct {
	Parts []string
	Raw   string
}

func NewFieldRef(raw string) FieldRef { return FieldRef{Parts: strings.Split(raw, "."), Raw: raw} }
func (f FieldRef) Canonical() string {
	if len(f.Parts) > 0 {
		return strings.Join(f.Parts, ".")
	}
	return f.Raw
}
func (f FieldRef) String() string { return f.Canonical() }

type CompareOp int

const (
	OpEq CompareOp = iota
	OpNeq
	OpGt
	OpGte
	OpLt
	OpLte
	OpIn
	OpNotIn
	OpLike
	OpMatches
	OpContains
	OpStartsWith
	OpEndsWith
)

var opNames = [...]string{"=", "!=", ">", ">=", "<", "<=", "IN", "NOT IN", "LIKE", "MATCHES", "CONTAINS", "STARTS_WITH", "ENDS_WITH"}

func (op CompareOp) String() string {
	if int(op) < len(opNames) {
		return opNames[op]
	}
	return "?"
}

type Value interface {
	value()
	String() string
}
type StringValue struct{ V string }
type NumberValue struct{ V float64 }
type BoolValue struct{ V bool }
type NullValue struct{}
type WildcardValue struct{ Pattern string }
type CIDRValue struct{ Network string }
type ListValue struct{ Items []Value }
type SubqueryValue struct{ Query *Query }

func (StringValue) value()   {}
func (NullValue) value()     {}
func (WildcardValue) value() {}
func (CIDRValue) value()     {}
func (NumberValue) value()   {}
func (BoolValue) value()     {}
func (ListValue) value()     {}
func (SubqueryValue) value() {}

func (v StringValue) String() string   { return fmt.Sprintf("%q", v.V) }
func (v NumberValue) String() string   { return fmt.Sprintf("%g", v.V) }
func (v BoolValue) String() string     { return fmt.Sprint(v.V) }
func (NullValue) String() string       { return "null" }
func (v WildcardValue) String() string { return v.Pattern }
func (v CIDRValue) String() string     { return v.Network }
func (v ListValue) String() string     { return fmt.Sprint(v.Items) }
func (v SubqueryValue) String() string { return fmt.Sprintf("[ %s ]", v.Query) }

// ── Commands ──────────────────────────────────────────────────────────────────

type Command interface {
	command()
	CommandName() string
}

type WhereCommand struct{ Expr SearchExpr }
type StatsCommand struct {
	Aggregations []AggExpr
	GroupBy      []FieldRef
}
type AggExpr struct {
	Alias string
	Func  string
	Field *FieldRef
	Args  []interface{}
}
type EvalCommand struct{ Assignments []EvalAssignment }
type EvalAssignment struct {
	Field FieldRef
	Expr  EvalExpr
}
type RexCommand struct {
	Field   *FieldRef
	Pattern string
}
type TableCommand struct{ Fields []FieldRef }
type SortCommand struct{ Specs []SortSpec }
type SortSpec struct {
	Field      FieldRef
	Descending bool
}
type HeadCommand struct{ Count int }
type TailCommand struct{ Count int }
type DedupCommand struct {
	Count  int
	Fields []FieldRef
	SortBy *SortSpec
}
type LookupCommand struct {
	Table        string
	MatchField   FieldRef
	AsField      *FieldRef
	OutputFields []FieldRef
}
type JoinCommand struct {
	Type     string
	Field    FieldRef
	Subquery *Query
}
type AppendCommand struct{ Subquery *Query }
type TimeChartCommand struct {
	Span         *time.Duration
	Aggregations []AggExpr
	SplitBy      *FieldRef
}
type ChartCommand struct {
	Aggregations []AggExpr
	Over         FieldRef
	By           *FieldRef
}
type TopCommand struct {
	Count  int
	Fields []FieldRef
	By     *FieldRef
}
type RareCommand struct {
	Count  int
	Fields []FieldRef
	By     *FieldRef
}
type TransactionCommand struct {
	Fields     []FieldRef
	MaxSpan    *time.Duration
	StartsWith SearchExpr
	EndsWith   SearchExpr
}
type RenameCommand struct{ Renames []RenameSpec }
type RenameSpec struct {
	From FieldRef
	To   string
}
type FieldsCommand struct {
	Mode   string
	Fields []FieldRef
}
type FillNullCommand struct {
	Value  *string
	Fields []FieldRef
}
type MvExpandCommand struct{ Field FieldRef }
type PredictCommand struct {
	Field  FieldRef
	Future int
}
type AnomalyDetectionCommand struct {
	Fields []FieldRef
	Method string
}

func (*WhereCommand) command()       {}
func (*StatsCommand) command()       {}
func (*EvalCommand) command()        {}
func (*RexCommand) command()         {}
func (*TableCommand) command()       {}
func (*SortCommand) command()        {}
func (*HeadCommand) command()        {}
func (*TailCommand) command()        {}
func (*DedupCommand) command()       {}
func (*LookupCommand) command()      {}
func (*JoinCommand) command()        {}
func (*AppendCommand) command()      {}
func (*TimeChartCommand) command()   {}
func (*ChartCommand) command()       {}
func (*TopCommand) command()         {}
func (*RareCommand) command()        {}
func (*TransactionCommand) command() {}
func (*RenameCommand) command()      {}
func (*FieldsCommand) command()      {}
func (*FillNullCommand) command()    {}
func (*MvExpandCommand) command()    {}
func (*PredictCommand) command()     {}
func (*AnomalyDetectionCommand) command() {}

func (*WhereCommand) CommandName() string       { return "where" }
func (*StatsCommand) CommandName() string       { return "stats" }
func (*EvalCommand) CommandName() string        { return "eval" }
func (*RexCommand) CommandName() string         { return "rex" }
func (*TableCommand) CommandName() string       { return "table" }
func (*SortCommand) CommandName() string        { return "sort" }
func (*HeadCommand) CommandName() string        { return "head" }
func (*TailCommand) CommandName() string        { return "tail" }
func (*DedupCommand) CommandName() string       { return "dedup" }
func (*LookupCommand) CommandName() string      { return "lookup" }
func (*JoinCommand) CommandName() string        { return "join" }
func (*AppendCommand) CommandName() string      { return "append" }
func (*TimeChartCommand) CommandName() string   { return "timechart" }
func (*ChartCommand) CommandName() string       { return "chart" }
func (*TopCommand) CommandName() string         { return "top" }
func (*RareCommand) CommandName() string        { return "rare" }
func (*TransactionCommand) CommandName() string { return "transaction" }
func (*RenameCommand) CommandName() string      { return "rename" }
func (*FieldsCommand) CommandName() string      { return "fields" }
func (*FillNullCommand) CommandName() string    { return "fillnull" }
func (*MvExpandCommand) CommandName() string    { return "mvexpand" }
func (*PredictCommand) CommandName() string     { return "predict" }
func (*AnomalyDetectionCommand) CommandName() string { return "anomalydetection" }

// ── Eval Expressions ──────────────────────────────────────────────────────────

type EvalExpr interface {
	evalExpr()
	String() string
}
type EvalBinaryOp struct {
	Op          string
	Left, Right EvalExpr
}
type EvalUnaryOp struct {
	Op   string
	Expr EvalExpr
}
type EvalFuncCall struct {
	Name string
	Args []EvalExpr
}
type EvalFieldRef struct{ Field FieldRef }
type EvalLiteral struct{ Value interface{} }
type EvalTernary struct{ Cond, TrueVal, FalseVal EvalExpr }

func (*EvalBinaryOp) evalExpr() {}
func (*EvalUnaryOp) evalExpr()  {}
func (*EvalFuncCall) evalExpr() {}
func (*EvalFieldRef) evalExpr() {}
func (*EvalLiteral) evalExpr()  {}
func (*EvalTernary) evalExpr()  {}

func (e *EvalBinaryOp) String() string { return fmt.Sprintf("(%s %s %s)", e.Left, e.Op, e.Right) }
func (e *EvalUnaryOp) String() string  { return fmt.Sprintf("%s%s", e.Op, e.Expr) }
func (e *EvalFuncCall) String() string { return fmt.Sprintf("%s(...)", e.Name) }
func (e *EvalFieldRef) String() string { return e.Field.String() }
func (e *EvalLiteral) String() string  { return fmt.Sprint(e.Value) }
func (e *EvalTernary) String() string {
	return fmt.Sprintf("if(%s,%s,%s)", e.Cond, e.TrueVal, e.FalseVal)
}
