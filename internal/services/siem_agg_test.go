package services

import (
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/oql"
)

func aggEvent(host, srcIP, user, bytes string, ts time.Time) events.Event {
	f := map[string]string{}
	if srcIP != "" {
		f["srcIP"] = srcIP
	}
	if user != "" {
		f["user"] = user
	}
	if bytes != "" {
		f["bytes"] = bytes
	}
	return events.Event{HostID: host, EventType: "failed_login", Fields: f, Timestamp: ts}
}

func TestApplyAggCountBy(t *testing.T) {
	now := time.Now().UTC()
	in := []events.Event{
		aggEvent("web-01", "10.0.0.1", "root", "", now),
		aggEvent("web-01", "10.0.0.1", "admin", "", now),
		aggEvent("web-01", "10.0.0.2", "root", "", now),
	}
	res := applyAgg(in, &oql.Agg{Kind: "stats", Fn: oql.AggCount, By: "srcIP"})
	if len(res.Buckets) != 2 {
		t.Fatalf("buckets = %+v", res.Buckets)
	}
	// sorted by value desc
	if res.Buckets[0].Key != "10.0.0.1" || res.Buckets[0].Value != 2 {
		t.Errorf("bucket[0] = %+v", res.Buckets[0])
	}
	if res.Buckets[1].Key != "10.0.0.2" || res.Buckets[1].Value != 1 {
		t.Errorf("bucket[1] = %+v", res.Buckets[1])
	}
}

func TestApplyAggSingleBucketCount(t *testing.T) {
	now := time.Now().UTC()
	in := []events.Event{aggEvent("a", "", "", "", now), aggEvent("b", "", "", "", now)}
	res := applyAgg(in, &oql.Agg{Kind: "stats", Fn: oql.AggCount})
	if len(res.Buckets) != 1 || res.Buckets[0].Value != 2 {
		t.Fatalf("buckets = %+v", res.Buckets)
	}
}

func TestApplyAggNumericFns(t *testing.T) {
	now := time.Now().UTC()
	in := []events.Event{
		aggEvent("h", "", "", "100", now),
		aggEvent("h", "", "", "300", now),
		aggEvent("h", "", "", "not-a-number", now), // skipped
	}
	for _, tc := range []struct {
		fn   oql.AggFn
		want float64
	}{
		{oql.AggSum, 400}, {oql.AggAvg, 200}, {oql.AggMin, 100}, {oql.AggMax, 300},
	} {
		res := applyAgg(in, &oql.Agg{Kind: "stats", Fn: tc.fn, Field: "bytes"})
		if res.Buckets[0].Value != tc.want {
			t.Errorf("%s = %v, want %v", tc.fn, res.Buckets[0].Value, tc.want)
		}
	}
}

func TestApplyAggDistinctCount(t *testing.T) {
	now := time.Now().UTC()
	in := []events.Event{
		aggEvent("web-01", "", "root", "", now),
		aggEvent("web-01", "", "root", "", now),
		aggEvent("web-01", "", "admin", "", now),
		aggEvent("db-01", "", "root", "", now),
	}
	res := applyAgg(in, &oql.Agg{Kind: "stats", Fn: oql.AggDC, Field: "user", By: "hostId"})
	if len(res.Buckets) != 2 {
		t.Fatalf("buckets = %+v", res.Buckets)
	}
	if res.Buckets[0].Key != "web-01" || res.Buckets[0].Value != 2 {
		t.Errorf("bucket[0] = %+v", res.Buckets[0])
	}
	if res.Buckets[1].Key != "db-01" || res.Buckets[1].Value != 1 {
		t.Errorf("bucket[1] = %+v", res.Buckets[1])
	}
}

func TestApplyAggTopN(t *testing.T) {
	now := time.Now().UTC()
	var in []events.Event
	for i := 0; i < 5; i++ {
		in = append(in, aggEvent("h", "10.0.0.1", "", "", now))
	}
	for i := 0; i < 3; i++ {
		in = append(in, aggEvent("h", "10.0.0.2", "", "", now))
	}
	in = append(in, aggEvent("h", "10.0.0.3", "", "", now))
	res := applyAgg(in, &oql.Agg{Kind: "top", Fn: oql.AggCount, By: "srcIP", TopN: 2})
	if len(res.Buckets) != 2 {
		t.Fatalf("top 2 returned %d buckets", len(res.Buckets))
	}
	if res.Buckets[0].Key != "10.0.0.1" || res.Buckets[1].Key != "10.0.0.2" {
		t.Errorf("buckets = %+v", res.Buckets)
	}
}

func TestApplyAggMissingByValue(t *testing.T) {
	now := time.Now().UTC()
	in := []events.Event{aggEvent("h", "", "", "", now)} // no srcIP field
	res := applyAgg(in, &oql.Agg{Kind: "stats", Fn: oql.AggCount, By: "srcIP"})
	if len(res.Buckets) != 1 || res.Buckets[0].Key != "(missing)" {
		t.Fatalf("buckets = %+v", res.Buckets)
	}
}

func TestTimechartBucketsAndSeries(t *testing.T) {
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	in := []events.Event{
		{HostID: "h", Severity: "warning", Timestamp: base.Add(5 * time.Minute)},
		{HostID: "h", Severity: "warning", Timestamp: base.Add(10 * time.Minute)},
		{HostID: "h", Severity: "critical", Timestamp: base.Add(20 * time.Minute)},
		{HostID: "h", Severity: "warning", Timestamp: base.Add(70 * time.Minute)}, // next bucket
	}
	res := applyAgg(in, &oql.Agg{Kind: "timechart", Fn: oql.AggCount, Span: time.Hour, By: "severity"})
	if res.Span != "1h0m0s" {
		t.Errorf("span = %q", res.Span)
	}
	if len(res.Buckets) != 2 {
		t.Fatalf("buckets = %+v", res.Buckets)
	}
	b0 := res.Buckets[0]
	if b0.Key != base.Format(time.RFC3339) || b0.Value != 3 {
		t.Errorf("bucket[0] = %+v", b0)
	}
	if b0.Series["warning"] != 2 || b0.Series["critical"] != 1 {
		t.Errorf("series = %+v", b0.Series)
	}
	if res.Buckets[1].Value != 1 {
		t.Errorf("bucket[1] = %+v", res.Buckets[1])
	}
	// oldest-first ordering
	if !(res.Buckets[0].Key < res.Buckets[1].Key) {
		t.Error("timechart buckets not oldest-first")
	}
}
