package oql

import "fmt"

type TypeChecker struct {
	Resolver *SCIMResolver
	errors   []TypeError
}

func (tc *TypeChecker) Check(q *Query) []TypeError {
	tc.errors = nil
	if q.Search != nil {
		tc.checkSearch(q.Search)
	}
	for _, cmd := range q.Commands {
		tc.checkCommand(cmd)
	}
	return tc.errors
}

func (tc *TypeChecker) checkSearch(expr SearchExpr) {
	switch e := expr.(type) {
	case *AndExpr:
		tc.checkSearch(e.Left)
		tc.checkSearch(e.Right)
	case *OrExpr:
		tc.checkSearch(e.Left)
		tc.checkSearch(e.Right)
	case *NotExpr:
		tc.checkSearch(e.Expr)
	}
}

func (tc *TypeChecker) checkCommand(cmd Command) {
	switch c := cmd.(type) {
	case *EvalCommand:
		for _, a := range c.Assignments {
			tc.inferEvalType(a.Expr)
		}
	case *WhereCommand:
		rt := tc.inferSearchType(c.Expr)
		if rt.Narrow() != FieldBoolean && rt.Narrow() != FieldAny {
			tc.errors = append(tc.errors, TypeError{
				Message: fmt.Sprintf("where condition must be boolean, got %s", rt.String()),
				Hint:    "Use comparison: where field=value",
			})
		}
	case *StatsCommand:
		for _, a := range c.Aggregations {
			tc.checkAgg(a)
		}
	}
}

func (tc *TypeChecker) inferSearchType(expr SearchExpr) TypeInfo {
	switch e := expr.(type) {
	case *CompareExpr:
		return TypeInfo{Type: FieldBoolean}
	case *AndExpr:
		lt := tc.inferSearchType(e.Left)
		rt := tc.inferSearchType(e.Right)
		tc.requireBool(lt, "AND left operand")
		tc.requireBool(rt, "AND right operand")
		return TypeInfo{Type: FieldBoolean}
	case *OrExpr:
		lt := tc.inferSearchType(e.Left)
		rt := tc.inferSearchType(e.Right)
		tc.requireBool(lt, "OR left operand")
		tc.requireBool(rt, "OR right operand")
		return TypeInfo{Type: FieldBoolean}
	case *NotExpr:
		inner := tc.inferSearchType(e.Expr)
		tc.requireBool(inner, "NOT operand")
		return TypeInfo{Type: FieldBoolean}
	case *FreeTextExpr:
		return TypeInfo{Type: FieldBoolean}
	case *FieldExistsExpr:
		return TypeInfo{Type: FieldBoolean}
	}
	return TypeInfo{Type: FieldAny}
}

func (tc *TypeChecker) requireBool(t TypeInfo, ctx string) {
	if t.Narrow() != FieldBoolean && t.Narrow() != FieldAny {
		tc.errors = append(tc.errors, TypeError{
			Message: fmt.Sprintf("%s must be boolean, got %s", ctx, t.String()),
		})
	}
}

func (tc *TypeChecker) inferEvalType(expr EvalExpr) TypeInfo {
	switch e := expr.(type) {
	case *EvalBinaryOp:
		lt := tc.inferEvalType(e.Left)
		rt := tc.inferEvalType(e.Right)
		res, err := ResolveBinaryOp(e.Op, lt.Narrow(), rt.Narrow())
		if err != nil {
			tc.errors = append(tc.errors, TypeError{Message: err.Error()})
			return TypeInfo{Type: FieldAny}
		}
		return TypeInfo{Type: res, Nullable: lt.Nullable || rt.Nullable}
	case *EvalFuncCall:
		fd, ok := BuiltinFuncs[e.Name]
		if !ok {
			tc.errors = append(tc.errors, TypeError{Message: fmt.Sprintf("unknown function '%s'", e.Name)})
			return TypeInfo{Type: FieldAny}
		}
		if len(e.Args) < fd.MinArgs || len(e.Args) > fd.MaxArgs {
			tc.errors = append(tc.errors, TypeError{
				Message: fmt.Sprintf("'%s' expects %d-%d args, got %d", e.Name, fd.MinArgs, fd.MaxArgs, len(e.Args)),
			})
		}
		for i, arg := range e.Args {
			at := tc.inferEvalType(arg)
			if i < len(fd.ArgTypes) && fd.ArgTypes[i] != FieldAny {
				ok, auto := CanCoerce(at.Narrow(), fd.ArgTypes[i])
				if !ok {
					tc.errors = append(tc.errors, TypeError{
						Message: fmt.Sprintf("'%s' arg %d: expected %s, got %s", e.Name, i+1, TypeName(fd.ArgTypes[i]), at.String()),
					})
				} else if !auto {
					tc.errors = append(tc.errors, TypeError{
						Message: fmt.Sprintf("'%s' arg %d: implicit %s→%s not allowed", e.Name, i+1, at.String(), TypeName(fd.ArgTypes[i])),
						Hint:    "Use tonumber() or tostring()",
					})
				}
			}
		}
		return TypeInfo{Type: fd.ReturnType}
	case *EvalLiteral:
		switch e.Value.(type) {
		case string:
			return TypeInfo{Type: FieldString}
		case float64:
			return TypeInfo{Type: FieldNumber}
		case bool:
			return TypeInfo{Type: FieldBoolean}
		case nil:
			return TypeInfo{Type: FieldAny, Nullable: true}
		}
	case *EvalFieldRef:
		if tc.Resolver != nil {
			if m, ok := tc.Resolver.FieldMeta[e.Field.Canonical()]; ok {
				return TypeInfo{Type: m.Type, Nullable: true}
			}
		}
		return TypeInfo{Type: FieldAny, Nullable: true}
	case *EvalTernary:
		ct := tc.inferEvalType(e.Cond)
		if ct.Narrow() != FieldBoolean && ct.Narrow() != FieldAny {
			tc.errors = append(tc.errors, TypeError{Message: "ternary condition must be boolean"})
		}
		tt := tc.inferEvalType(e.TrueVal)
		ft := tc.inferEvalType(e.FalseVal)
		return MakeUnion(tt, ft)
	case *EvalUnaryOp:
		inner := tc.inferEvalType(e.Expr)
		if e.Op == "NOT" {
			return TypeInfo{Type: FieldBoolean}
		}
		if e.Op == "-" {
			if inner.Narrow() != FieldNumber && inner.Narrow() != FieldAny {
				tc.errors = append(tc.errors, TypeError{
					Message: fmt.Sprintf("unary '-' requires number, got %s", inner.String()),
				})
			}
			return TypeInfo{Type: FieldNumber}
		}
		return inner
	}
	return TypeInfo{Type: FieldAny}
}

func (tc *TypeChecker) checkAgg(agg AggExpr) {
	switch agg.Func {
	case "sum", "avg", "stdev", "var", "median", "percentile", "range":
		if agg.Field != nil && tc.Resolver != nil {
			if m, ok := tc.Resolver.FieldMeta[agg.Field.Canonical()]; ok {
				if m.Type != FieldNumber && m.Type != FieldTimestamp {
					tc.errors = append(tc.errors, TypeError{
						Message: fmt.Sprintf("'%s' requires numeric field, '%s' is %s", agg.Func, agg.Field.Raw, TypeName(m.Type)),
						Hint:    "Use tonumber() first",
					})
				}
			}
		}
	}
}
