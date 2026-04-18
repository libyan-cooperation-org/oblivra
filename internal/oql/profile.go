package oql

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type QueryProfile struct {
	mu   sync.Mutex
	root []*StageProfile
}

type StageProfile struct {
	Name, PlanNode     string
	Index              int
	StartTime, EndTime time.Time
	Duration           time.Duration
	RowsIn, RowsOut    int64
	BytesRead          int64
	MemoryPeak         int64
	MemoryCurrent      int64
	Details            map[string]interface{}
	Children           []*StageProfile
	Parent             *StageProfile
}

func NewQueryProfile() *QueryProfile { return &QueryProfile{} }

func (qp *QueryProfile) BeginStage(name string, index int, planNode string) *StageProfiler {
	s := &StageProfile{Name: name, Index: index, PlanNode: planNode, StartTime: time.Now(), Details: map[string]interface{}{}}
	qp.mu.Lock()
	qp.root = append(qp.root, s)
	qp.mu.Unlock()
	return &StageProfiler{stage: s, profile: qp}
}

func (qp *QueryProfile) Stages() []StageProfile {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	r := make([]StageProfile, len(qp.root))
	for i, s := range qp.root {
		r[i] = *s
	}
	return r
}

func (qp *QueryProfile) Summary() string {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	var b strings.Builder
	b.WriteString("Query Execution Profile\n" + strings.Repeat("═", 60) + "\n")
	total := time.Duration(0)
	for _, s := range qp.root {
		total += s.Duration
	}
	for _, s := range qp.root {
		writeStage(&b, s, total, 0)
	}
	b.WriteString(strings.Repeat("─", 60) + "\n")
	fmt.Fprintf(&b, "  Total: %s\n", total.Round(time.Millisecond))
	return b.String()
}

func writeStage(b *strings.Builder, s *StageProfile, total time.Duration, depth int) {
	indent := strings.Repeat("  ", depth)
	pct := float64(0)
	if total > 0 {
		pct = float64(s.Duration) / float64(total) * 100
	}
	sel := ""
	if s.RowsIn > 0 {
		sel = fmt.Sprintf(" (%.1f%% pass)", float64(s.RowsOut)/float64(s.RowsIn)*100)
	}
	bar := strings.Repeat("█", int(pct/5))
	if len(bar) == 0 && s.Duration > 0 {
		bar = "▏"
	}
	fmt.Fprintf(b, "%s  %-10s %8d → %-8d  %8s  %5.1f%%  %s%s\n",
		indent, s.Name, s.RowsIn, s.RowsOut, s.Duration.Round(time.Millisecond), pct, bar, sel)
	for k, v := range s.Details {
		fmt.Fprintf(b, "%s             └─ %s: %v\n", indent, k, v)
	}
	for _, child := range s.Children {
		writeStage(b, child, total, depth+1)
	}
}

type StageProfiler struct {
	stage   *StageProfile
	profile *QueryProfile
}

func (p *StageProfiler) TrackRowIn()            { p.stage.RowsIn++ }
func (p *StageProfiler) TrackRowOut()           { p.stage.RowsOut++ }
func (p *StageProfiler) TrackBytesRead(n int64) { p.stage.BytesRead += n }
func (p *StageProfiler) TrackMemory(n int64) {
	p.stage.MemoryCurrent += n
	if p.stage.MemoryCurrent > p.stage.MemoryPeak {
		p.stage.MemoryPeak = p.stage.MemoryCurrent
	}
}
func (p *StageProfiler) ReleaseMemory(n int64)             { p.stage.MemoryCurrent -= n }
func (p *StageProfiler) SetDetail(k string, v interface{}) { p.stage.Details[k] = v }
func (p *StageProfiler) Finish() {
	p.stage.EndTime = time.Now()
	p.stage.Duration = p.stage.EndTime.Sub(p.stage.StartTime)
}

func (p *StageProfiler) BeginChild(name string) *StageProfiler {
	child := &StageProfile{Name: name, StartTime: time.Now(), Details: map[string]interface{}{}, Parent: p.stage}
	p.profile.mu.Lock()
	p.stage.Children = append(p.stage.Children, child)
	p.profile.mu.Unlock()
	return &StageProfiler{stage: child, profile: p.profile}
}
