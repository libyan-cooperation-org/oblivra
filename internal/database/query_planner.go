package database

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// QueryLimits defines the maximum resources a single query may consume.
// Exceeding any limit causes the query planner to reject the query before
// it touches the SIEM store — protecting the platform under heavy analyst load.
type QueryLimits struct {
	MaxScanRows   int64         // Max rows to scan (default 1,000,000)
	MaxExecTime   time.Duration // Max wall-clock execution time (default 10s)
	MaxResultRows int           // Max rows returned (default 10,000)
}

// DefaultQueryLimits are conservative defaults suitable for a shared SOC workstation.
var DefaultQueryLimits = QueryLimits{
	MaxScanRows:   1_000_000,
	MaxExecTime:   10 * time.Second,
	MaxResultRows: 10_000,
}

// HeavyQueryLimits are relaxed limits for scheduled reports or administrative queries.
var HeavyQueryLimits = QueryLimits{
	MaxScanRows:   50_000_000,
	MaxExecTime:   60 * time.Second,
	MaxResultRows: 100_000,
}

// QueryCost is the estimated resource cost of a query before execution.
type QueryCost struct {
	EstimatedRows int64
	EstimatedTime time.Duration
	ScanFullTable bool
	HasTimeRange  bool
	TimeRangeSecs int64
	Reason        string
}

// QueryPlanner estimates query cost and enforces limits before execution.
type QueryPlanner struct {
	limits     QueryLimits
	rowsPerSec int64 // estimated scan throughput for cost calculation
}

// NewQueryPlanner creates a planner with the given limits.
// rowsPerSec is the estimated scan throughput of the underlying store (default 500k/s).
func NewQueryPlanner(limits QueryLimits) *QueryPlanner {
	return &QueryPlanner{
		limits:     limits,
		rowsPerSec: 500_000, // conservative BadgerDB scan estimate
	}
}

// Plan estimates the cost of a search query and returns a QueryCost.
// query is the raw search string; mode is "logql"|"lucene"|"sql".
func (p *QueryPlanner) Plan(query, mode string, timeRangeSecs int64) (*QueryCost, error) {
	cost := &QueryCost{}

	// Estimate rows based on time range
	if timeRangeSecs > 0 {
		cost.HasTimeRange = true
		cost.TimeRangeSecs = timeRangeSecs
		// Rough heuristic: assume 1k EPS average = 1000 rows/sec
		cost.EstimatedRows = timeRangeSecs * 1_000
	} else {
		// No time range = full table scan
		cost.ScanFullTable = true
		cost.EstimatedRows = 100_000_000 // assume 100M rows worst case
		cost.Reason = "no time range specified — full table scan"
	}

	// Wildcard / match-all patterns are expensive
	q := strings.TrimSpace(query)
	if q == "" || q == "*" || q == "{}" {
		cost.EstimatedRows = cost.EstimatedRows * 10
		cost.Reason = "match-all query pattern"
	}

	// SQL queries with no WHERE clause
	if strings.EqualFold(mode, "sql") {
		lower := strings.ToLower(q)
		if !strings.Contains(lower, "where") && !strings.Contains(lower, "limit") {
			cost.ScanFullTable = true
			cost.EstimatedRows = 100_000_000
			cost.Reason = "SQL query without WHERE or LIMIT clause"
		}
	}

	// Estimate time based on rows and throughput
	if p.rowsPerSec > 0 {
		cost.EstimatedTime = time.Duration(cost.EstimatedRows/p.rowsPerSec) * time.Second
		if cost.EstimatedTime < 100*time.Millisecond {
			cost.EstimatedTime = 100 * time.Millisecond
		}
	}

	return cost, nil
}

// Validate checks whether a query cost is within the configured limits.
// Returns a descriptive error if any limit would be exceeded.
func (p *QueryPlanner) Validate(cost *QueryCost) error {
	if cost.EstimatedRows > p.limits.MaxScanRows {
		return fmt.Errorf(
			"query would scan ~%d rows (limit %d): %s — add a time range filter to narrow the query",
			cost.EstimatedRows, p.limits.MaxScanRows,
			cost.Reason,
		)
	}

	if cost.EstimatedTime > p.limits.MaxExecTime {
		return fmt.Errorf(
			"query estimated execution time %s exceeds limit %s — narrow the time range or add filters",
			cost.EstimatedTime.Round(time.Second),
			p.limits.MaxExecTime.Round(time.Second),
		)
	}

	return nil
}

// BoundedContext wraps a parent context with the query's execution time limit.
// The caller must call the returned cancel func when the query completes.
func (p *QueryPlanner) BoundedContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, p.limits.MaxExecTime)
}

// LimitResults caps a result slice to the configured max rows.
// Returns the (possibly truncated) slice and whether it was truncated.
func (p *QueryPlanner) LimitResults(rows int) (limit int, truncated bool) {
	if rows > p.limits.MaxResultRows {
		return p.limits.MaxResultRows, true
	}
	return rows, false
}
