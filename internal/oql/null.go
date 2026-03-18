package oql

import "fmt"

// OQL Null Semantics — SQL-style three-valued logic.
//
//   null = anything    → null
//   null != anything   → null
//   null > anything    → null
//   null AND true      → null
//   null AND false     → false
//   null OR true       → true
//   null OR false      → null
//   NOT null           → null
//   isnull(null)       → true
//   coalesce(null, x)  → x
//
// WHERE context: null treated as false (row excluded).
// EVAL context: null propagates.
// STATS context: null values excluded from aggregations.

func IsNull(v interface{}) bool { return v == nil }

func NullSafeEqual(a, b interface{}) interface{} {
	if a == nil || b == nil {
		return nil
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func NullSafeCompare(a, b interface{}) interface{} {
	if a == nil || b == nil {
		return nil
	}
	an, aok := ToNumber(a)
	bn, bok := ToNumber(b)
	if aok && bok {
		if an < bn {
			return -1
		}
		if an > bn {
			return 1
		}
		return 0
	}
	as, _ := ToString(a)
	bs, _ := ToString(b)
	if as < bs {
		return -1
	}
	if as > bs {
		return 1
	}
	return 0
}

func NullSafeAnd(a, b interface{}) interface{} {
	ab, aok := ToBool(a)
	bb, bok := ToBool(b)
	if aok && !ab {
		return false
	}
	if bok && !bb {
		return false
	}
	if a == nil || b == nil {
		return nil
	}
	return ab && bb
}

func NullSafeOr(a, b interface{}) interface{} {
	ab, aok := ToBool(a)
	bb, bok := ToBool(b)
	if aok && ab {
		return true
	}
	if bok && bb {
		return true
	}
	if a == nil || b == nil {
		return nil
	}
	return ab || bb
}

func NullSafeNot(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	b, ok := ToBool(v)
	if !ok {
		return nil
	}
	return !b
}

func WhereFilter(result interface{}) bool {
	if result == nil {
		return false
	}
	b, ok := result.(bool)
	return ok && b
}

func ValueToInterface(v Value) interface{} {
	switch val := v.(type) {
	case StringValue:
		return val.V
	case NumberValue:
		return val.V
	case BoolValue:
		return val.V
	case NullValue:
		return nil
	default:
		return fmt.Sprint(v)
	}
}
