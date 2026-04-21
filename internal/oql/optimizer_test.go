package oql

import (
	"testing"
)

func TestOptimizer_ConstantFolding(t *testing.T) {
	// true AND true should be folded
	expr := &AndExpr{
		Left:  &ConstantSearchExpr{Val: true},
		Right: &ConstantSearchExpr{Val: true},
	}
	
	opt := &Optimizer{}
	folded := opt.foldConstants(expr)
	
	res, ok := folded.(*ConstantSearchExpr)
	if !ok {
		t.Fatalf("Expected ConstantSearchExpr, got %T", folded)
	}
	if res.Val != true {
		t.Errorf("Expected true, got %v", res.Val)
	}

	// false AND something should be folded to false
	expr2 := &AndExpr{
		Left:  &ConstantSearchExpr{Val: false},
		Right: &FreeTextExpr{Text: "anything"},
	}
	folded2 := opt.foldConstants(expr2)
	res2, ok := folded2.(*ConstantSearchExpr)
	if !ok || res2.Val != false {
		t.Errorf("Expected false constant, got %v", folded2)
	}
}

func TestOptimizer_LimitPushdown(t *testing.T) {
	q := &Query{
		Commands: []Command{
			&HeadCommand{Count: 100},
			&HeadCommand{Count: 50},
		},
	}
	
	opt := &Optimizer{}
	optimized := opt.Optimize(q)
	
	if len(optimized.Commands) != 1 {
		t.Fatalf("Expected 1 command after limit pushdown, got %d", len(optimized.Commands))
	}
	
	head, ok := optimized.Commands[0].(*HeadCommand)
	if !ok {
		t.Fatalf("Expected HeadCommand, got %T", optimized.Commands[0])
	}
	
	if head.Count != 50 {
		t.Errorf("Expected count 50, got %d", head.Count)
	}
}
