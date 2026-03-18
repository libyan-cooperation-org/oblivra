package oql

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

type QueryBudget struct {
	MaxWallTime        time.Duration
	MaxCPUTime         time.Duration
	MaxMemoryBytes     int64
	MaxScanBytes       int64
	MaxRowsScanned     int64
	MaxRowsOutput      int64
	MaxGroupKeys       int
	MaxJoinMaterialize int64
}

var (
	InteractiveBudget = QueryBudget{
		MaxWallTime: 60 * time.Second, MaxCPUTime: 30 * time.Second,
		MaxMemoryBytes: 512 << 20, MaxScanBytes: 10 << 30,
		MaxRowsScanned: 50_000_000, MaxRowsOutput: 100_000,
		MaxGroupKeys: 50_000, MaxJoinMaterialize: 50_000,
	}
	ScheduledBudget = QueryBudget{
		MaxWallTime: 300 * time.Second, MaxCPUTime: 120 * time.Second,
		MaxMemoryBytes: 1 << 30, MaxScanBytes: 50 << 30,
		MaxRowsScanned: 200_000_000, MaxRowsOutput: 1_000_000,
		MaxGroupKeys: 100_000, MaxJoinMaterialize: 100_000,
	}
	AdminBudget = QueryBudget{
		MaxWallTime: 600 * time.Second, MaxCPUTime: 300 * time.Second,
		MaxMemoryBytes: 2 << 30, MaxScanBytes: 200 << 30,
		MaxRowsScanned: 1_000_000_000, MaxRowsOutput: 10_000_000,
		MaxGroupKeys: 500_000, MaxJoinMaterialize: 500_000,
	}
)

func BudgetForUser(role, queryType string) QueryBudget {
	if role == "admin" {
		return AdminBudget
	}
	if queryType == "scheduled" {
		return ScheduledBudget
	}
	return InteractiveBudget
}

type BudgetMonitor struct {
	budget       QueryBudget
	cancel       context.CancelFunc
	startTime    time.Time
	bytesScanned atomic.Int64
	rowsScanned  atomic.Int64
	rowsOutput   atomic.Int64
	memoryUsed   atomic.Int64
	groupKeys    atomic.Int64
	violation    atomic.Value
	done         chan struct{}
}

func NewBudgetMonitor(ctx context.Context, budget QueryBudget) (context.Context, *BudgetMonitor) {
	ctx, cancel := context.WithTimeout(ctx, budget.MaxWallTime)
	bm := &BudgetMonitor{budget: budget, cancel: cancel, startTime: time.Now(), done: make(chan struct{})}
	go bm.loop()
	return ctx, bm
}

func (bm *BudgetMonitor) loop() {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if m := bm.memoryUsed.Load(); m > bm.budget.MaxMemoryBytes {
				bm.violate("memory", m, bm.budget.MaxMemoryBytes,
					fmt.Sprintf("Query exceeded memory limit (%d MB / %d MB)", m>>20, bm.budget.MaxMemoryBytes>>20))
				return
			}
		case <-bm.done:
			return
		}
	}
}

func (bm *BudgetMonitor) Stop() {
	select {
	case <-bm.done:
	default:
		close(bm.done)
	}
}

func (bm *BudgetMonitor) violate(limit string, current, maximum int64, msg string) {
	bm.violation.Store(&BudgetViolation{Limit: limit, Current: current, Maximum: maximum, Message: msg})
	bm.cancel()
}

func (bm *BudgetMonitor) Violation() *BudgetViolation {
	v := bm.violation.Load()
	if v == nil {
		return nil
	}
	return v.(*BudgetViolation)
}

func (bm *BudgetMonitor) TrackScan(bytes, rows int64) {
	if b := bm.bytesScanned.Add(bytes); b > bm.budget.MaxScanBytes {
		bm.violate("scan_bytes", b, bm.budget.MaxScanBytes,
			fmt.Sprintf("Scanned %d GB (limit %d GB)", b>>30, bm.budget.MaxScanBytes>>30))
		return
	}
	if r := bm.rowsScanned.Add(rows); r > bm.budget.MaxRowsScanned {
		bm.violate("rows_scanned", r, bm.budget.MaxRowsScanned,
			fmt.Sprintf("Scanned %d events (limit %d)", r, bm.budget.MaxRowsScanned))
	}
}

func (bm *BudgetMonitor) TrackOutput(n int64) {
	if r := bm.rowsOutput.Add(n); r > bm.budget.MaxRowsOutput {
		bm.violate("rows_output", r, bm.budget.MaxRowsOutput,
			fmt.Sprintf("Output %d rows (limit %d)", r, bm.budget.MaxRowsOutput))
	}
}

func (bm *BudgetMonitor) TrackMemory(delta int64) { bm.memoryUsed.Add(delta) }

func (bm *BudgetMonitor) TrackGroupKey() bool {
	if k := bm.groupKeys.Add(1); k > int64(bm.budget.MaxGroupKeys) {
		bm.violate("group_keys", k, int64(bm.budget.MaxGroupKeys),
			fmt.Sprintf("Aggregation created %d groups (limit %d)", k, bm.budget.MaxGroupKeys))
		return false
	}
	return true
}

func (bm *BudgetMonitor) Snapshot() BudgetSnapshot {
	return BudgetSnapshot{
		WallTime: time.Since(bm.startTime), BytesScanned: bm.bytesScanned.Load(),
		RowsScanned: bm.rowsScanned.Load(), RowsOutput: bm.rowsOutput.Load(),
		MemoryUsed: bm.memoryUsed.Load(), GroupKeys: bm.groupKeys.Load(), Budget: bm.budget,
	}
}

type BudgetSnapshot struct {
	WallTime                                          time.Duration
	BytesScanned, RowsScanned, RowsOutput             int64
	MemoryUsed, GroupKeys                              int64
	Budget                                             QueryBudget
}

func (s BudgetSnapshot) Utilization() map[string]float64 {
	return map[string]float64{
		"wall_time":    float64(s.WallTime) / float64(s.Budget.MaxWallTime) * 100,
		"scan_bytes":   float64(s.BytesScanned) / float64(s.Budget.MaxScanBytes) * 100,
		"rows_scanned": float64(s.RowsScanned) / float64(s.Budget.MaxRowsScanned) * 100,
		"rows_output":  float64(s.RowsOutput) / float64(s.Budget.MaxRowsOutput) * 100,
		"memory":       float64(s.MemoryUsed) / float64(s.Budget.MaxMemoryBytes) * 100,
		"group_keys":   float64(s.GroupKeys) / float64(s.Budget.MaxGroupKeys) * 100,
	}
}
