package oql

import (
	"testing"
	"time"
)

func TestParseStatsCountBy(t *testing.T) {
	p, err := Parse("eventType:failed_login | stats count by srcIP")
	if err != nil {
		t.Fatal(err)
	}
	if p.Agg == nil {
		t.Fatal("expected Agg plan")
	}
	if p.Agg.Kind != "stats" || p.Agg.Fn != AggCount || p.Agg.By != "srcIP" || p.Agg.Field != "" {
		t.Errorf("agg = %+v", p.Agg)
	}
}

func TestParseStatsBareCount(t *testing.T) {
	p, err := Parse("* | stats count")
	if err != nil {
		t.Fatal(err)
	}
	if p.Agg == nil || p.Agg.Fn != AggCount || p.Agg.By != "" {
		t.Errorf("agg = %+v", p.Agg)
	}
}

func TestParseStatsFnField(t *testing.T) {
	for _, tc := range []struct {
		in    string
		fn    AggFn
		field string
		by    string
	}{
		{"* | stats sum(bytes) by hostId", AggSum, "bytes", "hostId"},
		{"* | stats avg(duration)", AggAvg, "duration", ""},
		{"* | stats min(latency) by host", AggMin, "latency", "host"},
		{"* | stats max(latency)", AggMax, "latency", ""},
		{"* | stats dc(user) by hostId", AggDC, "user", "hostId"},
	} {
		p, err := Parse(tc.in)
		if err != nil {
			t.Fatalf("%s: %v", tc.in, err)
		}
		if p.Agg.Fn != tc.fn || p.Agg.Field != tc.field || p.Agg.By != tc.by {
			t.Errorf("%s → %+v", tc.in, p.Agg)
		}
	}
}

func TestParseStatsErrors(t *testing.T) {
	for _, in := range []string{
		"* | stats",                     // no fn
		"* | stats sum",                 // fn needs field
		"* | stats bogus(x)",            // unknown fn
		"* | stats count by",            // by needs field
		"* | stats sum(bytes",           // unbalanced paren
		"* | stats count by x | head 5", // agg must be terminal
	} {
		if _, err := Parse(in); err == nil {
			t.Errorf("%q: expected error", in)
		}
	}
}

func TestParseTop(t *testing.T) {
	p, err := Parse("* | top 10 srcIP")
	if err != nil {
		t.Fatal(err)
	}
	if p.Agg == nil || p.Agg.Kind != "top" || p.Agg.TopN != 10 || p.Agg.By != "srcIP" || p.Agg.Fn != AggCount {
		t.Errorf("agg = %+v", p.Agg)
	}
	if _, err := Parse("* | top srcIP"); err == nil {
		t.Error("expected error: top without N")
	}
	if _, err := Parse("* | top 0 srcIP"); err == nil {
		t.Error("expected error: top 0")
	}
}

func TestParseTimechart(t *testing.T) {
	p, err := Parse("* | timechart span=15m count by severity")
	if err != nil {
		t.Fatal(err)
	}
	if p.Agg == nil || p.Agg.Kind != "timechart" || p.Agg.Span != 15*time.Minute || p.Agg.By != "severity" {
		t.Errorf("agg = %+v", p.Agg)
	}

	// span defaults to 1h; by optional
	p, err = Parse("* | timechart count")
	if err != nil {
		t.Fatal(err)
	}
	if p.Agg.Span != time.Hour || p.Agg.By != "" {
		t.Errorf("agg = %+v", p.Agg)
	}

	for _, in := range []string{
		"* | timechart",                // missing count
		"* | timechart span=abc count", // bad span
		"* | timechart count by",       // by needs field
		"* | timechart sum(x)",         // only count supported
		"* | timechart count by a b",   // trailing tokens
	} {
		if _, err := Parse(in); err == nil {
			t.Errorf("%q: expected error", in)
		}
	}
}

func TestParseStatsWithPrecedingStages(t *testing.T) {
	p, err := Parse(`hostId:web-01 | where eventType:failed_login | stats count by srcIP`)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Filters) != 1 || p.Agg == nil {
		t.Errorf("plan = %+v", p)
	}
}
