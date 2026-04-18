package oql

import (
	"fmt"
	"strings"
)

type FieldType int

const (
	FieldString FieldType = iota
	FieldNumber
	FieldBoolean
	FieldTimestamp
	FieldIP
	FieldList
	FieldAny   FieldType = -1
	FieldUnion FieldType = -2
)

type TypeInfo struct {
	Type     FieldType
	Nullable bool
	List     bool
	Union    []FieldType
}

func (t TypeInfo) Narrow() FieldType {
	if t.Type != FieldUnion || len(t.Union) == 0 {
		return t.Type
	}
	if len(t.Union) == 1 {
		return t.Union[0]
	}
	first := t.Union[0]
	for _, u := range t.Union[1:] {
		if u != first {
			return FieldAny
		}
	}
	return first
}

func MakeUnion(a, b TypeInfo) TypeInfo {
	if a.Type == b.Type {
		return TypeInfo{Type: a.Type, Nullable: a.Nullable || b.Nullable}
	}
	var types []FieldType
	if a.Type == FieldUnion {
		types = append(types, a.Union...)
	} else {
		types = append(types, a.Type)
	}
	if b.Type == FieldUnion {
		types = append(types, b.Union...)
	} else {
		types = append(types, b.Type)
	}
	seen := map[FieldType]bool{}
	var unique []FieldType
	for _, t := range types {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}
	if len(unique) == 1 {
		return TypeInfo{Type: unique[0], Nullable: a.Nullable || b.Nullable}
	}
	return TypeInfo{Type: FieldUnion, Union: unique, Nullable: a.Nullable || b.Nullable}
}

func (t TypeInfo) String() string {
	if t.Type == FieldUnion && len(t.Union) > 0 {
		ns := make([]string, len(t.Union))
		for i, u := range t.Union {
			ns[i] = TypeName(u)
		}
		s := strings.Join(ns, "|")
		if t.Nullable {
			s += "?"
		}
		return s
	}
	s := TypeName(t.Type)
	if t.Nullable {
		s += "?"
	}
	return s
}

type FieldMeta struct {
	CanonicalName string
	Description   string
	DataModel     string
	Type          FieldType
	Indexed       bool
	Stored        bool
	Searchable    bool
}

func TypeName(t FieldType) string {
	names := map[FieldType]string{
		FieldString: "string", FieldNumber: "number", FieldBoolean: "boolean",
		FieldTimestamp: "timestamp", FieldIP: "ip", FieldList: "list",
		FieldAny: "any", FieldUnion: "union",
	}
	if n, ok := names[t]; ok {
		return n
	}
	return "unknown"
}

type CoercionRule struct {
	From, To FieldType
	Auto     bool
}

var coercionRules = []CoercionRule{
	{FieldNumber, FieldString, false}, {FieldString, FieldNumber, false},
	{FieldBoolean, FieldString, false}, {FieldBoolean, FieldNumber, true},
	{FieldNumber, FieldBoolean, true}, {FieldTimestamp, FieldNumber, true},
	{FieldTimestamp, FieldString, false}, {FieldString, FieldTimestamp, false},
	{FieldNumber, FieldTimestamp, true}, {FieldIP, FieldString, true},
	{FieldString, FieldIP, true},
}

func CanCoerce(from, to FieldType) (bool, bool) {
	if from == to || from == FieldAny || to == FieldAny {
		return true, true
	}
	for _, r := range coercionRules {
		if r.From == from && r.To == to {
			return true, r.Auto
		}
	}
	return false, false
}

type BinaryOpRule struct {
	Op            string
	Left, Right   FieldType
	Result        FieldType
}

var binaryOpRules = []BinaryOpRule{
	{"+", FieldNumber, FieldNumber, FieldNumber}, {"-", FieldNumber, FieldNumber, FieldNumber},
	{"*", FieldNumber, FieldNumber, FieldNumber}, {"/", FieldNumber, FieldNumber, FieldNumber},
	{"%", FieldNumber, FieldNumber, FieldNumber}, {".", FieldString, FieldString, FieldString},
	{"=", FieldString, FieldString, FieldBoolean}, {"=", FieldNumber, FieldNumber, FieldBoolean},
	{"=", FieldIP, FieldIP, FieldBoolean}, {"=", FieldTimestamp, FieldTimestamp, FieldBoolean},
	{"=", FieldBoolean, FieldBoolean, FieldBoolean}, {"!=", FieldString, FieldString, FieldBoolean},
	{"!=", FieldNumber, FieldNumber, FieldBoolean}, {">", FieldNumber, FieldNumber, FieldBoolean},
	{">", FieldTimestamp, FieldTimestamp, FieldBoolean}, {">", FieldString, FieldString, FieldBoolean},
	{">=", FieldNumber, FieldNumber, FieldBoolean}, {">=", FieldTimestamp, FieldTimestamp, FieldBoolean},
	{"<", FieldNumber, FieldNumber, FieldBoolean}, {"<", FieldTimestamp, FieldTimestamp, FieldBoolean},
	{"<=", FieldNumber, FieldNumber, FieldBoolean}, {"<=", FieldTimestamp, FieldTimestamp, FieldBoolean},
	{"AND", FieldBoolean, FieldBoolean, FieldBoolean}, {"OR", FieldBoolean, FieldBoolean, FieldBoolean},
}

func ResolveBinaryOp(op string, left, right FieldType) (FieldType, error) {
	for _, r := range binaryOpRules {
		if r.Op == op && r.Left == left && r.Right == right {
			return r.Result, nil
		}
	}
	for _, r := range binaryOpRules {
		if r.Op == op {
			lok, la := CanCoerce(left, r.Left)
			rok, ra := CanCoerce(right, r.Right)
			if lok && rok && la && ra {
				return r.Result, nil
			}
		}
	}
	return 0, fmt.Errorf("operator '%s' not defined for %s and %s", op, TypeName(left), TypeName(right))
}

type FuncCost int

const (
	CostTrivial FuncCost = 1
	CostCheap   FuncCost = 2
	CostMedium  FuncCost = 5
	CostHeavy   FuncCost = 10
	CostIO      FuncCost = 50
)
