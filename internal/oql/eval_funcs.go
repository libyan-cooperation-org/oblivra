package oql

import (
	"fmt"
	"math"
	"net"
	"strings"
	"time"
)

type FuncDef struct {
	Name       string
	MinArgs    int
	MaxArgs    int
	ReturnType FieldType
	ArgTypes   []FieldType
	Pure       bool
	Cost       FuncCost
	Eval       func(ctx *EvalContext, args []interface{}) (interface{}, error)
}

var BuiltinFuncs = map[string]FuncDef{
	"now": {
		Name: "now", MaxArgs: 0, ReturnType: FieldTimestamp, Cost: CostTrivial,
		Eval: func(c *EvalContext, _ []interface{}) (interface{}, error) { return c.NowTime(), nil },
	},
	"random": {
		Name: "random", MaxArgs: 0, ReturnType: FieldNumber, Cost: CostTrivial,
		Eval: func(c *EvalContext, _ []interface{}) (interface{}, error) { return c.Random(), nil },
	},
	"if": {
		Name: "if", MinArgs: 3, MaxArgs: 3, ReturnType: FieldAny, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			c, _ := ToBool(a[0])
			if c {
				return a[1], nil
			}
			return a[2], nil
		},
	},
	"case": {
		Name: "case", MinArgs: 2, MaxArgs: 100, ReturnType: FieldAny, Pure: true, Cost: CostCheap,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			for i := 0; i+1 < len(a); i += 2 {
				c, _ := ToBool(a[i])
				if c {
					return a[i+1], nil
				}
			}
			if len(a)%2 == 1 {
				return a[len(a)-1], nil
			}
			return nil, nil
		},
	},
	"coalesce": {
		Name: "coalesce", MinArgs: 1, MaxArgs: 100, ReturnType: FieldAny, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			for _, v := range a {
				if v != nil {
					return v, nil
				}
			}
			return nil, nil
		},
	},
	"isnull": {
		Name: "isnull", MinArgs: 1, MaxArgs: 1, ReturnType: FieldBoolean, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) { return a[0] == nil, nil },
	},
	"isnotnull": {
		Name: "isnotnull", MinArgs: 1, MaxArgs: 1, ReturnType: FieldBoolean, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) { return a[0] != nil, nil },
	},
	"len": {
		Name: "len", MinArgs: 1, MaxArgs: 1, ReturnType: FieldNumber, ArgTypes: []FieldType{FieldString},
		Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			s, _ := ToString(a[0])
			return float64(len(s)), nil
		},
	},
	"lower": {
		Name: "lower", MinArgs: 1, MaxArgs: 1, ReturnType: FieldString, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			s, _ := ToString(a[0])
			return strings.ToLower(s), nil
		},
	},
	"upper": {
		Name: "upper", MinArgs: 1, MaxArgs: 1, ReturnType: FieldString, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			s, _ := ToString(a[0])
			return strings.ToUpper(s), nil
		},
	},
	"trim": {
		Name: "trim", MinArgs: 1, MaxArgs: 1, ReturnType: FieldString, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			s, _ := ToString(a[0])
			return strings.TrimSpace(s), nil
		},
	},
	"substr": {
		Name: "substr", MinArgs: 2, MaxArgs: 3, ReturnType: FieldString, Pure: true, Cost: CostMedium,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			s, _ := ToString(a[0])
			start, _ := ToNumber(a[1])
			i := int(start)
			if i < 0 {
				i = 0
			}
			if i > len(s) {
				return "", nil
			}
			if len(a) == 3 {
				l, _ := ToNumber(a[2])
				e := i + int(l)
				if e > len(s) {
					e = len(s)
				}
				return s[i:e], nil
			}
			return s[i:], nil
		},
	},
	"replace": {
		Name: "replace", MinArgs: 3, MaxArgs: 3, ReturnType: FieldString, Pure: true, Cost: CostMedium,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			s, _ := ToString(a[0])
			o, _ := ToString(a[1])
			n, _ := ToString(a[2])
			return strings.ReplaceAll(s, o, n), nil
		},
	},
	"split": {
		Name: "split", MinArgs: 2, MaxArgs: 2, ReturnType: FieldList, Pure: true, Cost: CostMedium,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			s, _ := ToString(a[0])
			d, _ := ToString(a[1])
			parts := strings.Split(s, d)
			r := make([]interface{}, len(parts))
			for i, p := range parts {
				r[i] = p
			}
			return r, nil
		},
	},
	"tonumber": {
		Name: "tonumber", MinArgs: 1, MaxArgs: 1, ReturnType: FieldNumber, Pure: true, Cost: CostCheap,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			n, ok := ToNumber(a[0])
			if !ok {
				return nil, nil
			}
			return n, nil
		},
	},
	"tostring": {
		Name: "tostring", MinArgs: 1, MaxArgs: 1, ReturnType: FieldString, Pure: true, Cost: CostCheap,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			s, _ := ToString(a[0])
			return s, nil
		},
	},
	"cidrmatch": {
		Name: "cidrmatch", MinArgs: 2, MaxArgs: 2, ReturnType: FieldBoolean, Pure: true, Cost: CostHeavy,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			c, _ := ToString(a[0])
			ip, _ := ToString(a[1])
			_, n, err := net.ParseCIDR(c)
			if err != nil {
				return false, nil
			}
			p := net.ParseIP(ip)
			if p == nil {
				return false, nil
			}
			return n.Contains(p), nil
		},
	},
	"typeof": {
		Name: "typeof", MinArgs: 1, MaxArgs: 1, ReturnType: FieldString, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			switch a[0].(type) {
			case nil:
				return "null", nil
			case string:
				return "string", nil
			case float64:
				return "number", nil
			case bool:
				return "boolean", nil
			case time.Time:
				return "timestamp", nil
			case []interface{}:
				return "list", nil
			default:
				return "unknown", nil
			}
		},
	},
	"round": {
		Name: "round", MinArgs: 1, MaxArgs: 2, ReturnType: FieldNumber, Pure: true, Cost: CostCheap,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			n, _ := ToNumber(a[0])
			p := 0
			if len(a) > 1 {
				pf, _ := ToNumber(a[1])
				p = int(pf)
			}
			s := math.Pow(10, float64(p))
			return math.Round(n*s) / s, nil
		},
	},
	"abs": {
		Name: "abs", MinArgs: 1, MaxArgs: 1, ReturnType: FieldNumber, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			n, _ := ToNumber(a[0])
			return math.Abs(n), nil
		},
	},
	"ceil": {
		Name: "ceil", MinArgs: 1, MaxArgs: 1, ReturnType: FieldNumber, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			n, _ := ToNumber(a[0])
			return math.Ceil(n), nil
		},
	},
	"floor": {
		Name: "floor", MinArgs: 1, MaxArgs: 1, ReturnType: FieldNumber, Pure: true, Cost: CostTrivial,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			n, _ := ToNumber(a[0])
			return math.Floor(n), nil
		},
	},
	"sqrt": {
		Name: "sqrt", MinArgs: 1, MaxArgs: 1, ReturnType: FieldNumber, Pure: true, Cost: CostCheap,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			n, _ := ToNumber(a[0])
			return math.Sqrt(n), nil
		},
	},
	"pow": {
		Name: "pow", MinArgs: 2, MaxArgs: 2, ReturnType: FieldNumber, Pure: true, Cost: CostCheap,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			b, _ := ToNumber(a[0])
			e, _ := ToNumber(a[1])
			return math.Pow(b, e), nil
		},
	},
	"log": {
		Name: "log", MinArgs: 1, MaxArgs: 2, ReturnType: FieldNumber, Pure: true, Cost: CostCheap,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			n, _ := ToNumber(a[0])
			if len(a) > 1 {
				b, _ := ToNumber(a[1])
				return math.Log(n) / math.Log(b), nil
			}
			return math.Log10(n), nil
		},
	},
	"strftime": {
		Name: "strftime", MinArgs: 2, MaxArgs: 2, ReturnType: FieldString, Pure: true, Cost: CostMedium,
		Eval: func(_ *EvalContext, a []interface{}) (interface{}, error) {
			t, ok := a[0].(time.Time)
			if !ok {
				return nil, nil
			}
			f, _ := ToString(a[1])
			r := strings.NewReplacer(
				"%Y", "2006", "%m", "01", "%d", "02", "%H", "15",
				"%M", "04", "%S", "05", "%z", "-0700", "%Z", "MST",
				"%A", "Monday", "%a", "Mon", "%B", "January", "%b", "Jan",
				"%p", "PM", "%I", "03",
			)
			return t.Format(r.Replace(f)), nil
		},
	},
}

func EstimateEvalCost(expr EvalExpr) int {
	switch e := expr.(type) {
	case *EvalLiteral:
		return 0
	case *EvalFieldRef:
		return 1
	case *EvalBinaryOp:
		return 1 + EstimateEvalCost(e.Left) + EstimateEvalCost(e.Right)
	case *EvalUnaryOp:
		return 1 + EstimateEvalCost(e.Expr)
	case *EvalFuncCall:
		cost := int(CostCheap)
		if fd, ok := BuiltinFuncs[e.Name]; ok {
			cost = int(fd.Cost)
		}
		for _, arg := range e.Args {
			cost += EstimateEvalCost(arg)
		}
		return cost
	case *EvalTernary:
		return 1 + EstimateEvalCost(e.Cond) + EstimateEvalCost(e.TrueVal) + EstimateEvalCost(e.FalseVal)
	}
	return 1
}

func ToBool(v interface{}) (bool, bool) {
	switch val := v.(type) {
	case bool:
		return val, true
	case float64:
		return val != 0, true
	case string:
		return val != "" && val != "false" && val != "0", true
	case nil:
		return false, true
	default:
		return false, false
	}
}

func ToString(v interface{}) (string, bool) {
	if v == nil {
		return "", false
	}
	return fmt.Sprint(v), true
}

func ToNumber(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case bool:
		if val {
			return 1, true
		}
		return 0, true
	case string:
		var n float64
		_, err := fmt.Sscan(val, &n)
		return n, err == nil
	default:
		return 0, false
	}
}
