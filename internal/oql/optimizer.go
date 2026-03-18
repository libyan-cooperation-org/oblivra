package oql

import (
	"fmt"
	"sort"
	"strings"
)

type Optimizer struct {
	Resolver *SCIMResolver
	Stats    OptimizerStats
}

type OptimizerStats interface {
	IsIndexed(field string) bool
	FieldCardinality(field string) int64
	FieldCoverage(field string) float64
}

func (o *Optimizer) Optimize(q *Query) *Query {
	result := *q
	result.Search, result.Commands = o.pushDownPredicates(result.Search, result.Commands)
	result.Commands = o.reorderFilters(result.Commands)
	result.Commands = o.eliminateRedundantProjections(result.Commands)
	return &result
}

func (o *Optimizer) pushDownPredicates(search SearchExpr, cmds []Command) (SearchExpr, []Command) {
	var result []Command
	computed := map[string]bool{}
	for _, cmd := range cmds {
		switch c := cmd.(type) {
		case *EvalCommand:
			for _, a := range c.Assignments {
				computed[a.Field.Canonical()] = true
			}
			result = append(result, cmd)
		case *WhereCommand:
			if !referencesComputed(c.Expr, computed) {
				if search == nil {
					search = c.Expr
				} else {
					search = &AndExpr{Left: search, Right: c.Expr}
				}
			} else {
				result = append(result, cmd)
			}
		default:
			result = append(result, cmd)
		}
	}
	return search, result
}

func (o *Optimizer) reorderFilters(cmds []Command) []Command {
	result := make([]Command, 0, len(cmds))
	i := 0
	for i < len(cmds) {
		if _, ok := cmds[i].(*WhereCommand); !ok {
			result = append(result, cmds[i])
			i++
			continue
		}
		var run []scoredWhere
		for i < len(cmds) {
			wc, ok := cmds[i].(*WhereCommand)
			if !ok {
				break
			}
			sel := o.estimateSelectivity(wc.Expr)
			cost := o.estimateFilterCost(wc.Expr)
			run = append(run, scoredWhere{cmd: wc, score: sel * cost})
			i++
		}
		sort.Slice(run, func(a, b int) bool { return run[a].score < run[b].score })
		for _, sw := range run {
			result = append(result, sw.cmd)
		}
	}
	return result
}

type scoredWhere struct {
	cmd   *WhereCommand
	score float64
}

func (o *Optimizer) eliminateRedundantProjections(cmds []Command) []Command {
	if len(cmds) < 2 {
		return cmds
	}
	result := make([]Command, 0, len(cmds))
	for i := 0; i < len(cmds); i++ {
		isProj := false
		switch cmds[i].(type) {
		case *TableCommand, *FieldsCommand:
			isProj = true
		}
		if isProj && i+1 < len(cmds) {
			switch cmds[i+1].(type) {
			case *TableCommand, *FieldsCommand:
				continue
			}
		}
		result = append(result, cmds[i])
	}
	return result
}

func (o *Optimizer) estimateSelectivity(expr SearchExpr) float64 {
	switch e := expr.(type) {
	case *CompareExpr:
		if o.Stats != nil {
			card := o.Stats.FieldCardinality(e.Field.Canonical())
			if card > 0 {
				switch e.Op {
				case OpEq:
					return 1.0 / float64(card)
				case OpNeq:
					return 1.0 - 1.0/float64(card)
				}
			}
		}
		switch e.Op {
		case OpEq:
			return 0.01
		case OpNeq:
			return 0.99
		case OpGt, OpGte, OpLt, OpLte:
			return 0.33
		case OpIn:
			return 0.1
		case OpContains, OpLike, OpMatches:
			return 0.1
		case OpStartsWith:
			return 0.05
		}
	case *FreeTextExpr:
		return 0.01
	case *FieldExistsExpr:
		if o.Stats != nil {
			cov := o.Stats.FieldCoverage(e.Field.Canonical())
			if e.Exists {
				return cov
			}
			return 1.0 - cov
		}
		return 0.5
	case *AndExpr:
		return o.estimateSelectivity(e.Left) * o.estimateSelectivity(e.Right)
	case *OrExpr:
		l := o.estimateSelectivity(e.Left)
		r := o.estimateSelectivity(e.Right)
		return 1.0 - (1.0-l)*(1.0-r)
	case *NotExpr:
		return 1.0 - o.estimateSelectivity(e.Expr)
	}
	return 0.5
}

func (o *Optimizer) estimateFilterCost(expr SearchExpr) float64 {
	switch e := expr.(type) {
	case *CompareExpr:
		base := 1.0
		if o.Stats != nil && o.Stats.IsIndexed(e.Field.Canonical()) {
			base = 0.1
		}
		switch e.Op {
		case OpContains:
			return 5.0
		case OpLike:
			return 3.0
		case OpMatches:
			return 10.0
		default:
			return base
		}
	case *FreeTextExpr:
		return 0.1
	case *AndExpr:
		return o.estimateFilterCost(e.Left) + o.estimateFilterCost(e.Right)
	case *OrExpr:
		return o.estimateFilterCost(e.Left) + o.estimateFilterCost(e.Right)
	case *NotExpr:
		return o.estimateFilterCost(e.Expr)
	}
	return 1.0
}

func referencesComputed(expr SearchExpr, computed map[string]bool) bool {
	switch e := expr.(type) {
	case *CompareExpr:
		return computed[e.Field.Canonical()]
	case *FieldExistsExpr:
		return computed[e.Field.Canonical()]
	case *AndExpr:
		return referencesComputed(e.Left, computed) || referencesComputed(e.Right, computed)
	case *OrExpr:
		return referencesComputed(e.Left, computed) || referencesComputed(e.Right, computed)
	case *NotExpr:
		return referencesComputed(e.Expr, computed)
	}
	return false
}

func (o *Optimizer) ExplainPlan(original, optimized *Query) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Optimizer Report\n")
	fmt.Fprintf(&b, "  Original commands: %d\n", len(original.Commands))
	fmt.Fprintf(&b, "  Optimized commands: %d\n", len(optimized.Commands))
	if len(optimized.Commands) < len(original.Commands) {
		fmt.Fprintf(&b, "  Predicates pushed down: %d\n", len(original.Commands)-len(optimized.Commands))
	}
	return b.String()
}
