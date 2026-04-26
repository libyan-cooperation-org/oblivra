package oql

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Parse parses an OQL query string into an AST.
func Parse(input string, macros map[string]MacroDef) (*Query, error) {
	expanded := expandMacros(input, macros)
	tokens, err := Tokenize(expanded)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens, macros: macros}
	q, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	if p.peek().Type != TokEOF {
		t := p.peek()
		return nil, &ParseError{Position: t.Pos, Line: t.Line, Col: t.Col,
			Message: fmt.Sprintf("unexpected token %s", t.String())}
	}
	return q, nil
}

type parser struct {
	tokens []Token
	pos    int
	macros map[string]MacroDef
}

func (p *parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() Token {
	t := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return t
}

func (p *parser) expect(t TokenType) (Token, error) {
	tok := p.advance()
	if tok.Type != t {
		return tok, &ParseError{Position: tok.Pos, Line: tok.Line, Col: tok.Col,
			Message: fmt.Sprintf("expected %s, got %s", tokenName(t), tok.String())}
	}
	return tok, nil
}

func (p *parser) matchIdent(name string) bool {
	if p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, name) {
		p.advance()
		return true
	}
	return false
}

func (p *parser) parseQuery() (*Query, error) {
	q := &Query{Version: CurrentOQLVersion, Macros: p.macros}
	// Parse optional search expression (everything before first pipe)
	if p.peek().Type != TokPipe && p.peek().Type != TokEOF {
		search, err := p.parseSearchExpr()
		if err != nil {
			return nil, err
		}
		q.Search = search
	}
	// Parse piped commands
	for p.peek().Type == TokPipe {
		p.advance() // consume pipe
		cmd, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		q.Commands = append(q.Commands, cmd)
	}
	return q, nil
}

func (p *parser) parseSearchExpr() (SearchExpr, error) {
	return p.parseOr()
}

func (p *parser) parseOr() (SearchExpr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "OR") {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &OrExpr{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (SearchExpr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		if p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "AND") {
			p.advance()
		} else if p.peek().Type == TokEOF || p.peek().Type == TokPipe {
			break
		} else if p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "OR") {
			break
		} else if p.peek().Type == TokRParen {
			break
		} else {
			// implicit AND
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &AndExpr{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (SearchExpr, error) {
	if p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "NOT") {
		p.advance()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &NotExpr{Expr: expr}, nil
	}
	if p.peek().Type == TokBang {
		p.advance()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &NotExpr{Expr: expr}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (SearchExpr, error) {
	if p.peek().Type == TokLParen {
		p.advance()
		expr, err := p.parseSearchExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, err
		}
		return expr, nil
	}
	if p.peek().Type == TokString {
		t := p.advance()
		return &FreeTextExpr{Text: t.Val}, nil
	}
	if p.peek().Type == TokIdent {
		// Look ahead for comparison operator
		if p.pos+1 < len(p.tokens) && isCmpTok(p.tokens[p.pos+1].Type) {
			field := p.advance()
			op := p.parseCmpOp()
			val, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			return &CompareExpr{Field: NewFieldRef(field.Val), Op: op, Value: val}, nil
		}
		// Check for IN / NOT IN
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokIdent {
			nextVal := strings.ToUpper(p.tokens[p.pos+1].Val)
			if nextVal == "IN" {
				field := p.advance()
				p.advance() // consume IN
				val, err := p.parseListValue()
				if err != nil {
					return nil, err
				}
				return &CompareExpr{Field: NewFieldRef(field.Val), Op: OpIn, Value: val}, nil
			}
		}
		// Free text search
		t := p.advance()
		return &FreeTextExpr{Text: t.Val}, nil
	}
	if (p.peek().Type == TokLBracket) {
		p.advance()
		if p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "search") {
			p.advance()
		}
		sub, err := p.parseQuery()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRBracket); err != nil {
			return nil, err
		}
		return &SubqueryExpr{Query: sub}, nil
	}
	t := p.peek()
	return nil, &ParseError{Position: t.Pos, Line: t.Line, Col: t.Col,
		Message: fmt.Sprintf("unexpected token in search: %s", t.String())}
}

func (p *parser) parseCmpOp() CompareOp {
	t := p.advance()
	switch t.Type {
	case TokEq:
		return OpEq
	case TokNeq:
		return OpNeq
	case TokGt:
		return OpGt
	case TokGte:
		return OpGte
	case TokLt:
		return OpLt
	case TokLte:
		return OpLte
	}
	return OpEq
}

func (p *parser) parseValue() (Value, error) {
	t := p.peek()
	switch t.Type {
	case TokString:
		p.advance()
		if strings.ContainsAny(t.Val, "*?") {
			return WildcardValue{Pattern: t.Val}, nil
		}
		return StringValue{V: t.Val}, nil
	case TokNumber:
		p.advance()
		n, _ := strconv.ParseFloat(t.Val, 64)
		return NumberValue{V: n}, nil
	case TokIdent:
		lower := strings.ToLower(t.Val)
		switch lower {
		case "true":
			p.advance()
			return BoolValue{V: true}, nil
		case "false":
			p.advance()
			return BoolValue{V: false}, nil
		case "null":
			p.advance()
			return NullValue{}, nil
		default:
			p.advance()
			if strings.ContainsAny(t.Val, "*?") {
				return WildcardValue{Pattern: t.Val}, nil
			}
			return StringValue{V: t.Val}, nil
		}
	case TokLBracket:
		p.advance()
		// Check if it's a subsearch: [ search ... ]
		if p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "search") {
			p.advance()
			sub, err := p.parseQuery()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokRBracket); err != nil {
				return nil, err
			}
			return SubqueryValue{Query: sub}, nil
		}
		p.pos-- // backtrack to LBracket for parseListValue
		return p.parseListValue()
	}
	return nil, &ParseError{Position: t.Pos, Line: t.Line, Col: t.Col,
		Message: fmt.Sprintf("expected value, got %s", t.String())}
}

func (p *parser) parseListValue() (Value, error) {
	if p.peek().Type == TokLBracket {
		p.advance()
	} else if _, err := p.expect(TokLParen); err != nil {
		return nil, err
	}
	var items []Value
	for p.peek().Type != TokRBracket && p.peek().Type != TokRParen && p.peek().Type != TokEOF {
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		items = append(items, v)
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	if p.peek().Type == TokRBracket {
		p.advance()
	} else if p.peek().Type == TokRParen {
		p.advance()
	}
	return ListValue{Items: items}, nil
}

func (p *parser) parseCommand() (Command, error) {
	if p.peek().Type != TokIdent {
		t := p.peek()
		return nil, &ParseError{Position: t.Pos, Line: t.Line, Col: t.Col,
			Message: "expected command name"}
	}
	name := strings.ToLower(p.peek().Val)
	switch name {
	case "where":
		return p.parseWhere()
	case "stats":
		return p.parseStats()
	case "eval":
		return p.parseEval()
	case "table":
		return p.parseTable()
	case "sort":
		return p.parseSort()
	case "head":
		return p.parseHead()
	case "tail":
		return p.parseTail()
	case "dedup":
		return p.parseDedup()
	case "rename":
		return p.parseRename()
	case "fields":
		return p.parseFields()
	case "fillnull":
		return p.parseFillNull()
	case "top":
		return p.parseTopRare("top")
	case "rare":
		return p.parseTopRare("rare")
	case "rex":
		return p.parseRex()
	case "parse":
		return p.parseParse()
	case "lookup":
		return p.parseLookup()
	case "join":
		return p.parseJoin()
	case "append":
		return p.parseAppend()
	case "timechart":
		return p.parseTimeChart()
	case "chart":
		return p.parseChart()
	case "mvexpand":
		return p.parseMvExpand()
	case "predict":
		return p.parsePredict()
	case "anomalydetection":
		return p.parseAnomalyDetection()
	default:
		t := p.advance()
		return nil, &ParseError{Position: t.Pos, Line: t.Line, Col: t.Col,
			Message: fmt.Sprintf("unknown command '%s'", t.Val),
			Hint:    "Available: where, stats, eval, table, sort, head, tail, dedup, rename, fields, fillnull, top, rare, rex"}
	}
}

func (p *parser) parseWhere() (Command, error) {
	p.advance() // consume "where"
	expr, err := p.parseSearchExpr()
	if err != nil {
		return nil, err
	}
	return &WhereCommand{Expr: expr}, nil
}

func (p *parser) parseStats() (Command, error) {
	p.advance() // consume "stats"
	cmd := &StatsCommand{}
	for p.peek().Type != TokPipe && p.peek().Type != TokEOF {
		if p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "by") {
			p.advance()
			for p.peek().Type == TokIdent {
				cmd.GroupBy = append(cmd.GroupBy, NewFieldRef(p.advance().Val))
				if p.peek().Type == TokComma {
					p.advance()
				}
			}
			break
		}
		agg, err := p.parseAggExpr()
		if err != nil {
			return nil, err
		}
		cmd.Aggregations = append(cmd.Aggregations, agg)
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseAggExpr() (AggExpr, error) {
	// Check for alias: alias=func(field)
	var alias string
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokEq {
		alias = p.advance().Val
		p.advance() // consume =
	}
	funcName := p.advance().Val
	a := AggExpr{Func: strings.ToLower(funcName)}
	if p.peek().Type == TokLParen {
		p.advance()
		if p.peek().Type != TokRParen {
			f := NewFieldRef(p.advance().Val)
			a.Field = &f
		}
		if _, err := p.expect(TokRParen); err != nil {
			return a, err
		}
	}
	if alias != "" {
		a.Alias = alias
	} else if a.Field != nil {
		a.Alias = a.Func + "_" + a.Field.Canonical()
	} else {
		a.Alias = a.Func
	}
	return a, nil
}

func (p *parser) parseLookup() (Command, error) {
	p.advance() // consume "lookup"
	table := p.advance().Val
	cmd := &LookupCommand{Table: table}
	p.matchIdent("match")
	cmd.MatchField = NewFieldRef(p.advance().Val)
	if p.matchIdent("as") {
		f := NewFieldRef(p.advance().Val)
		cmd.AsField = &f
	}
	if p.matchIdent("output") {
		for p.peek().Type == TokIdent {
			cmd.OutputFields = append(cmd.OutputFields, NewFieldRef(p.advance().Val))
			if p.peek().Type == TokComma {
				p.advance()
			}
		}
	}
	return cmd, nil
}

func (p *parser) parseJoin() (Command, error) {
	p.advance() // consume "join"
	cmd := &JoinCommand{Type: "inner"}
	if p.peek().Type == TokIdent && (strings.EqualFold(p.peek().Val, "left") || strings.EqualFold(p.peek().Val, "outer")) {
		cmd.Type = strings.ToLower(p.advance().Val)
	}
	cmd.Field = NewFieldRef(p.advance().Val)
	if _, err := p.expect(TokLBracket); err != nil {
		return nil, err
	}
	p.matchIdent("search")
	sub, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	cmd.Subquery = sub
	if _, err := p.expect(TokRBracket); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (p *parser) parseAppend() (Command, error) {
	p.advance() // consume "append"
	if _, err := p.expect(TokLBracket); err != nil {
		return nil, err
	}
	p.matchIdent("search")
	sub, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokRBracket); err != nil {
		return nil, err
	}
	return &AppendCommand{Subquery: sub}, nil
}

func (p *parser) parseTimeChart() (Command, error) {
	p.advance() // consume "timechart"
	cmd := &TimeChartCommand{}
	for p.peek().Type != TokPipe && p.peek().Type != TokEOF {
		if p.matchIdent("span") {
			p.matchIdent("=")
			s := p.advance().Val
			dur, _ := time.ParseDuration(s)
			cmd.Span = &dur
			continue
		}
		if p.matchIdent("by") {
			f := NewFieldRef(p.advance().Val)
			cmd.SplitBy = &f
			break
		}
		agg, err := p.parseAggExpr()
		if err != nil {
			return nil, err
		}
		cmd.Aggregations = append(cmd.Aggregations, agg)
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseChart() (Command, error) {
	p.advance() // consume "chart"
	cmd := &ChartCommand{}
	for p.peek().Type != TokPipe && p.peek().Type != TokEOF {
		if p.matchIdent("over") {
			cmd.Over = NewFieldRef(p.advance().Val)
			continue
		}
		if p.matchIdent("by") {
			f := NewFieldRef(p.advance().Val)
			cmd.By = &f
			continue
		}
		agg, err := p.parseAggExpr()
		if err != nil {
			return nil, err
		}
		cmd.Aggregations = append(cmd.Aggregations, agg)
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseMvExpand() (Command, error) {
	p.advance() // consume "mvexpand"
	return &MvExpandCommand{Field: NewFieldRef(p.advance().Val)}, nil
}

func (p *parser) parseEval() (Command, error) {
	p.advance() // consume "eval"
	cmd := &EvalCommand{}
	for p.peek().Type != TokPipe && p.peek().Type != TokEOF {
		field := p.advance()
		if _, err := p.expect(TokEq); err != nil {
			return nil, err
		}
		expr, err := p.parseEvalExpr()
		if err != nil {
			return nil, err
		}
		cmd.Assignments = append(cmd.Assignments, EvalAssignment{Field: NewFieldRef(field.Val), Expr: expr})
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseEvalExpr() (EvalExpr, error) {
	return p.parseEvalTernary()
}

func (p *parser) parseEvalTernary() (EvalExpr, error) {
	expr, err := p.parseEvalOr()
	if err != nil {
		return nil, err
	}
	if p.peek().Type == TokQuestion {
		p.advance()
		tv, err := p.parseEvalOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokColon); err != nil {
			return nil, err
		}
		fv, err := p.parseEvalOr()
		if err != nil {
			return nil, err
		}
		return &EvalTernary{Cond: expr, TrueVal: tv, FalseVal: fv}, nil
	}
	return expr, nil
}

func (p *parser) parseEvalOr() (EvalExpr, error) {
	left, err := p.parseEvalAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == TokPipePipe || (p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "OR")) {
		p.advance()
		right, err := p.parseEvalAnd()
		if err != nil {
			return nil, err
		}
		left = &EvalBinaryOp{Op: "OR", Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseEvalAnd() (EvalExpr, error) {
	left, err := p.parseEvalComparison()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == TokAmpAmp || (p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "AND")) {
		p.advance()
		right, err := p.parseEvalComparison()
		if err != nil {
			return nil, err
		}
		left = &EvalBinaryOp{Op: "AND", Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseEvalComparison() (EvalExpr, error) {
	left, err := p.parseEvalAddSub()
	if err != nil {
		return nil, err
	}
	for isCmpTok(p.peek().Type) {
		op := p.advance()
		right, err := p.parseEvalAddSub()
		if err != nil {
			return nil, err
		}
		left = &EvalBinaryOp{Op: op.Val, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseEvalAddSub() (EvalExpr, error) {
	left, err := p.parseEvalMulDiv()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == TokPlus || p.peek().Type == TokMinus {
		op := p.advance()
		right, err := p.parseEvalMulDiv()
		if err != nil {
			return nil, err
		}
		left = &EvalBinaryOp{Op: op.Val, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseEvalMulDiv() (EvalExpr, error) {
	left, err := p.parseEvalUnary()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == TokStar || p.peek().Type == TokSlash || p.peek().Type == TokPercent {
		op := p.advance()
		right, err := p.parseEvalUnary()
		if err != nil {
			return nil, err
		}
		left = &EvalBinaryOp{Op: op.Val, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseEvalUnary() (EvalExpr, error) {
	if p.peek().Type == TokMinus {
		p.advance()
		expr, err := p.parseEvalPrimary()
		if err != nil {
			return nil, err
		}
		return &EvalUnaryOp{Op: "-", Expr: expr}, nil
	}
	if p.peek().Type == TokBang || (p.peek().Type == TokIdent && strings.EqualFold(p.peek().Val, "NOT")) {
		p.advance()
		expr, err := p.parseEvalPrimary()
		if err != nil {
			return nil, err
		}
		return &EvalUnaryOp{Op: "NOT", Expr: expr}, nil
	}
	return p.parseEvalPrimary()
}

func (p *parser) parseEvalPrimary() (EvalExpr, error) {
	t := p.peek()
	switch t.Type {
	case TokNumber:
		p.advance()
		n, _ := strconv.ParseFloat(t.Val, 64)
		return &EvalLiteral{Value: n}, nil
	case TokString:
		p.advance()
		return &EvalLiteral{Value: t.Val}, nil
	case TokLParen:
		p.advance()
		expr, err := p.parseEvalExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, err
		}
		return expr, nil
	case TokIdent:
		lower := strings.ToLower(t.Val)
		if lower == "true" {
			p.advance()
			return &EvalLiteral{Value: true}, nil
		}
		if lower == "false" {
			p.advance()
			return &EvalLiteral{Value: false}, nil
		}
		if lower == "null" {
			p.advance()
			return &EvalLiteral{Value: nil}, nil
		}
		if lower == "if" && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokLParen {
			p.advance() // consume "if"
			p.advance() // consume "("
			cond, err := p.parseEvalExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokComma); err != nil {
				return nil, err
			}
			tv, err := p.parseEvalExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokComma); err != nil {
				return nil, err
			}
			fv, err := p.parseEvalExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokRParen); err != nil {
				return nil, err
			}
			return &EvalTernary{Cond: cond, TrueVal: tv, FalseVal: fv}, nil
		}
		// Check for function call
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokLParen {
			name := p.advance().Val
			p.advance() // consume "("
			var args []EvalExpr
			for p.peek().Type != TokRParen && p.peek().Type != TokEOF {
				arg, err := p.parseEvalExpr()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				if p.peek().Type == TokComma {
					p.advance()
				}
			}
			if _, err := p.expect(TokRParen); err != nil {
				return nil, err
			}
			return &EvalFuncCall{Name: strings.ToLower(name), Args: args}, nil
		}
		// Field reference
		p.advance()
		return &EvalFieldRef{Field: NewFieldRef(t.Val)}, nil
	}
	return nil, &ParseError{Position: t.Pos, Line: t.Line, Col: t.Col,
		Message: fmt.Sprintf("unexpected token in eval expression: %s", t.String())}
}

func (p *parser) parseTable() (Command, error) {
	p.advance()
	cmd := &TableCommand{}
	for p.peek().Type == TokIdent {
		cmd.Fields = append(cmd.Fields, NewFieldRef(p.advance().Val))
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseSort() (Command, error) {
	p.advance()
	cmd := &SortCommand{}
	for p.peek().Type == TokIdent || p.peek().Type == TokMinus || p.peek().Type == TokPlus {
		desc := false
		if p.peek().Type == TokMinus {
			desc = true
			p.advance()
		} else if p.peek().Type == TokPlus {
			p.advance()
		}
		if p.peek().Type != TokIdent {
			break
		}
		cmd.Specs = append(cmd.Specs, SortSpec{Field: NewFieldRef(p.advance().Val), Descending: desc})
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseHead() (Command, error) {
	p.advance()
	count := 10
	if p.peek().Type == TokNumber {
		n, _ := strconv.Atoi(p.advance().Val)
		if n > 0 {
			count = n
		}
	}
	return &HeadCommand{Count: count}, nil
}

func (p *parser) parseTail() (Command, error) {
	p.advance()
	count := 10
	if p.peek().Type == TokNumber {
		n, _ := strconv.Atoi(p.advance().Val)
		if n > 0 {
			count = n
		}
	}
	return &TailCommand{Count: count}, nil
}

func (p *parser) parseDedup() (Command, error) {
	p.advance()
	cmd := &DedupCommand{Count: 1}
	if p.peek().Type == TokNumber {
		n, _ := strconv.Atoi(p.advance().Val)
		if n > 0 {
			cmd.Count = n
		}
	}
	for p.peek().Type == TokIdent && !strings.EqualFold(p.peek().Val, "sortby") {
		cmd.Fields = append(cmd.Fields, NewFieldRef(p.advance().Val))
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseRename() (Command, error) {
	p.advance()
	cmd := &RenameCommand{}
	for p.peek().Type == TokIdent {
		from := p.advance().Val
		if !p.matchIdent("as") {
			return nil, &ParseError{Message: "expected 'as' in rename"}
		}
		to := p.advance().Val
		cmd.Renames = append(cmd.Renames, RenameSpec{From: NewFieldRef(from), To: to})
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseFields() (Command, error) {
	p.advance()
	cmd := &FieldsCommand{Mode: "include"}
	if p.peek().Type == TokMinus {
		cmd.Mode = "exclude"
		p.advance()
	} else if p.peek().Type == TokPlus {
		p.advance()
	}
	for p.peek().Type == TokIdent {
		cmd.Fields = append(cmd.Fields, NewFieldRef(p.advance().Val))
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseFillNull() (Command, error) {
	p.advance()
	cmd := &FillNullCommand{}
	if p.matchIdent("value") {
		if p.peek().Type == TokEq {
			p.advance()
		}
		if p.peek().Type == TokString || p.peek().Type == TokNumber || p.peek().Type == TokIdent {
			v := p.advance().Val
			cmd.Value = &v
		}
	}
	for p.peek().Type == TokIdent {
		cmd.Fields = append(cmd.Fields, NewFieldRef(p.advance().Val))
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func (p *parser) parseTopRare(name string) (Command, error) {
	p.advance()
	count := 10
	if p.peek().Type == TokNumber {
		n, _ := strconv.Atoi(p.advance().Val)
		if n > 0 {
			count = n
		}
	}
	var fields []FieldRef
	var by *FieldRef
	for p.peek().Type == TokIdent {
		if strings.EqualFold(p.peek().Val, "by") {
			p.advance()
			if p.peek().Type == TokIdent {
				f := NewFieldRef(p.advance().Val)
				by = &f
			}
			break
		}
		fields = append(fields, NewFieldRef(p.advance().Val))
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	if name == "top" {
		return &TopCommand{Count: count, Fields: fields, By: by}, nil
	}
	return &RareCommand{Count: count, Fields: fields, By: by}, nil
}

func (p *parser) parseRex() (Command, error) {
	p.advance()
	cmd := &RexCommand{}
	if p.matchIdent("field") {
		if p.peek().Type == TokEq {
			p.advance()
		}
		if p.peek().Type == TokIdent {
			f := NewFieldRef(p.advance().Val)
			cmd.Field = &f
		}
	}
	if p.peek().Type == TokString {
		cmd.Pattern = p.advance().Val
	}
	return cmd, nil
}

// parseParse consumes the `parse` command body. Grammar:
//
//	parse <format> [<field>] [as <output>]
//
// where `<format>` is one of `json`, `xml`, `kv`. The source field
// defaults to `_raw` when omitted. The optional `as <output>` clause
// puts every extracted key under the given prefix.
//
// Examples accepted:
//
//	... | parse json
//	... | parse json _raw
//	... | parse json message as evt
//	... | parse xml body
//	... | parse kv message
func (p *parser) parseParse() (Command, error) {
	p.advance() // consume "parse"
	if p.peek().Type != TokIdent {
		t := p.peek()
		return nil, &ParseError{Position: t.Pos, Line: t.Line, Col: t.Col,
			Message: "parse: expected format (json|xml|kv) after `parse`"}
	}
	formatTok := p.advance()
	cmd := &ParseCommand{
		Field: NewFieldRef("_raw"),
	}
	switch strings.ToLower(formatTok.Val) {
	case "json":
		cmd.Format = ParseJSON
	case "xml":
		cmd.Format = ParseXML
	case "kv":
		cmd.Format = ParseKV
	default:
		return nil, &ParseError{Position: formatTok.Pos, Line: formatTok.Line, Col: formatTok.Col,
			Message: "parse: unknown format '" + formatTok.Val + "' (expected json|xml|kv)"}
	}
	// Optional source field — anything that isn't `as` and starts a
	// new identifier is treated as the source field name.
	if p.peek().Type == TokIdent && !p.matchIdent("as") {
		cmd.Field = NewFieldRef(p.advance().Val)
	}
	// Optional `as <output>` prefix.
	if p.matchIdent("as") {
		p.advance()
		if p.peek().Type != TokIdent {
			t := p.peek()
			return nil, &ParseError{Position: t.Pos, Line: t.Line, Col: t.Col,
				Message: "parse: expected output prefix after `as`"}
		}
		cmd.Output = p.advance().Val
	}
	return cmd, nil
}

func isCmpTok(t TokenType) bool {
	switch t {
	case TokEq, TokNeq, TokGt, TokGte, TokLt, TokLte:
		return true
	}
	return false
}

func (p *parser) parsePredict() (Command, error) {
	p.advance() // consume "predict"
	if p.peek().Type != TokIdent {
		return nil, &ParseError{Message: "expected field name for predict"}
	}
	field := p.advance()
	cmd := &PredictCommand{Field: NewFieldRef(field.Val), Future: 5}
	if p.matchIdent("future") {
		if p.peek().Type == TokEq {
			p.advance()
		}
		if p.peek().Type == TokNumber {
			f, _ := strconv.Atoi(p.advance().Val)
			cmd.Future = f
		}
	}
	return cmd, nil
}

func (p *parser) parseAnomalyDetection() (Command, error) {
	p.advance() // consume "anomalydetection"
	cmd := &AnomalyDetectionCommand{Method: "iqr"}
	for p.peek().Type == TokIdent {
		if strings.EqualFold(p.peek().Val, "method") {
			p.advance()
			if p.peek().Type == TokEq {
				p.advance()
			}
			if p.peek().Type == TokIdent {
				cmd.Method = p.advance().Val
			}
			continue
		}
		cmd.Fields = append(cmd.Fields, NewFieldRef(p.advance().Val))
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return cmd, nil
}

func expandMacros(input string, macros map[string]MacroDef) string {
	if macros == nil {
		return input
	}
	result := input
	for name, macro := range macros {
		placeholder := "`" + name + "`"
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, macro.Body)
		}
	}
	return result
}

