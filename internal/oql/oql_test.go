package oql

import "testing"

func TestParseEmpty(t *testing.T) {
	p, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if p.Expr != "" || len(p.Filters) != 0 {
		t.Errorf("expected empty plan, got %+v", p)
	}
}

func TestParseSimpleExpr(t *testing.T) {
	p, err := Parse("severity:warning | limit 25")
	if err != nil {
		t.Fatal(err)
	}
	if p.Expr != "severity:warning" {
		t.Errorf("expr = %q", p.Expr)
	}
	if p.Limit != 25 {
		t.Errorf("limit = %d", p.Limit)
	}
}

func TestParseWildcardExpr(t *testing.T) {
	p, err := Parse("* | where severity:critical")
	if err != nil {
		t.Fatal(err)
	}
	if p.Expr != "" {
		t.Errorf("expected wildcard expr to collapse to empty, got %q", p.Expr)
	}
	if len(p.Filters) != 1 || p.Filters[0].Field != "severity" || p.Filters[0].Value != "critical" {
		t.Errorf("filters = %+v", p.Filters)
	}
}

func TestParseSortDesc(t *testing.T) {
	p, err := Parse("hostId:web-01 | sort -timestamp | head 10")
	if err != nil {
		t.Fatal(err)
	}
	if !p.SortDesc {
		t.Error("expected SortDesc")
	}
	if p.SortField != "timestamp" {
		t.Errorf("sortField = %q", p.SortField)
	}
	if p.Limit != 10 {
		t.Errorf("limit = %d", p.Limit)
	}
}

func TestParseUnknownStage(t *testing.T) {
	if _, err := Parse("x | bogus 1"); err == nil {
		t.Error("expected unknown-stage error")
	}
}

func TestParseBadWhere(t *testing.T) {
	if _, err := Parse("a | where notvalid"); err == nil {
		t.Error("expected error on where without colon")
	}
}

func TestParseQuotedPipe(t *testing.T) {
	// pipe inside quotes must not split stages
	p, err := Parse(`message:"a | b" | limit 5`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Expr != `message:"a | b"` {
		t.Errorf("expr = %q", p.Expr)
	}
	if p.Limit != 5 {
		t.Errorf("limit = %d", p.Limit)
	}
}
