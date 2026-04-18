package oql

import (
	"fmt"
	"strings"
)

type ParseError struct {
	Position  int
	Line, Col int
	Message   string
	Hint      string
}

func (e *ParseError) Error() string {
	msg := fmt.Sprintf("parse error at line %d col %d: %s", e.Line, e.Col, e.Message)
	if e.Hint != "" {
		msg += " (hint: " + e.Hint + ")"
	}
	return msg
}

type TypeError struct {
	Position int
	Message  string
	Hint     string
}

type QueryTypeError struct{ Errors []TypeError }

func (e *QueryTypeError) Error() string {
	msgs := make([]string, len(e.Errors))
	for i, te := range e.Errors {
		msgs[i] = te.Message
		if te.Hint != "" {
			msgs[i] += " (hint: " + te.Hint + ")"
		}
	}
	return "type errors:\n  " + strings.Join(msgs, "\n  ")
}

type BudgetViolation struct {
	Limit   string
	Current int64
	Maximum int64
	Message string
}

func (v *BudgetViolation) Error() string { return v.Message }
