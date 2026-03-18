package oql

import (
	"context"
	"fmt"
	"time"
)

type JoinGuardrails struct {
	MaxSubqueryRows     int
	MaxSubqueryMemBytes int64
	MaxOutputRows       int64
	SubqueryTimeout     time.Duration
}

var DefaultJoinGuardrails = JoinGuardrails{
	MaxSubqueryRows: 50_000, MaxSubqueryMemBytes: 256 << 20,
	MaxOutputRows: 1_000_000, SubqueryTimeout: 30 * time.Second,
}

func ExecuteSafeJoin(ctx context.Context, left, right <-chan Row, field FieldRef, joinType string, g JoinGuardrails, mt *MemTracker, prof *StageProfiler, meta *QueryMeta) <-chan Row {
	out := make(chan Row, 256)
	go func() {
		defer close(out)
		ht := make(map[string][]Row)
		cnt := 0
		var mem int64
		trunc := false
		subCtx, cancel := context.WithTimeout(ctx, g.SubqueryTimeout)
		for row := range right {
			select {
			case <-subCtx.Done():
				trunc = true
				goto drain
			default:
			}
			cnt++
			rs := EstimateRowBytes(row)
			mem += rs
			if cnt > g.MaxSubqueryRows || mem > g.MaxSubqueryMemBytes {
				trunc = true
				meta.Warnings = append(meta.Warnings, fmt.Sprintf("Join subquery truncated at %d rows", cnt))
				goto drain
			}
			mt.Alloc(rs)
			k := fmt.Sprint(row[field.Canonical()])
			ht[k] = append(ht[k], row)
		}
	drain:
		if trunc {
			for range right {
			}
		}
		cancel()
		prof.SetDetail("right_rows", cnt)
		prof.SetDetail("right_truncated", trunc)
		outCnt := int64(0)
		i := 0
		for lr := range left {
			if cancelled(ctx, i) {
				return
			}
			i++
			prof.TrackRowIn()
			k := fmt.Sprint(lr[field.Canonical()])
			matches := ht[k]
			emit := func(r Row) bool {
				outCnt++
				if outCnt > g.MaxOutputRows {
					meta.Truncated = true
					for range left {
					}
					return false
				}
				prof.TrackRowOut()
				select {
				case out <- r:
					return true
				case <-ctx.Done():
					return false
				}
			}
			switch joinType {
			case "inner":
				for _, rr := range matches {
					if !emit(mergeRows(lr, rr)) {
						return
					}
				}
			case "left", "outer":
				if len(matches) == 0 {
					if !emit(lr) {
						return
					}
				} else {
					for _, rr := range matches {
						if !emit(mergeRows(lr, rr)) {
							return
						}
					}
				}
			}
		}
		prof.SetDetail("output_rows", outCnt)
	}()
	return out
}

func mergeRows(a, b Row) Row {
	m := make(Row, len(a)+len(b))
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	return m
}
