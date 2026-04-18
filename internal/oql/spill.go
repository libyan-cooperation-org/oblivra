package oql

import (
	"bufio"
	"container/heap"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var globalSpillSem = make(chan struct{}, 4)

func acquireSpill() { globalSpillSem <- struct{}{} }
func releaseSpill() { <-globalSpillSem }

type SpillManager struct {
	dir       string
	mu        sync.Mutex
	files     []string
	totalSize atomic.Int64
	maxDisk   int64
}

func NewSpillManager(tmpDir string, maxBytes int64) (*SpillManager, error) {
	dir := filepath.Join(tmpDir, fmt.Sprintf("oql-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &SpillManager{dir: dir, maxDisk: maxBytes}, nil
}

func (sm *SpillManager) Cleanup() { os.RemoveAll(sm.dir) }

func (sm *SpillManager) NewWriter(prefix string) (*SpillWriter, error) {
	sm.mu.Lock()
	path := filepath.Join(sm.dir, fmt.Sprintf("%s-%d.bin", prefix, len(sm.files)))
	sm.files = append(sm.files, path)
	sm.mu.Unlock()
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &SpillWriter{file: f, buf: bufio.NewWriterSize(f, 128<<10), path: path, mgr: sm}, nil
}

type SpillWriter struct {
	file  *os.File
	buf   *bufio.Writer
	path  string
	mgr   *SpillManager
	count int64
	bytes int64
}

func (w *SpillWriter) WriteRow(row Row) error {
	acquireSpill()
	defer releaseSpill()
	startBytes := w.bytes
	if err := binary.Write(w.buf, binary.LittleEndian, uint32(len(row))); err != nil {
		return err
	}
	w.bytes += 4
	for k, v := range row {
		if err := binary.Write(w.buf, binary.LittleEndian, uint16(len(k))); err != nil {
			return err
		}
		w.buf.WriteString(k)
		w.bytes += 2 + int64(len(k))
		switch val := v.(type) {
		case nil:
			w.buf.WriteByte(0)
			w.bytes++
		case string:
			w.buf.WriteByte(1)
			binary.Write(w.buf, binary.LittleEndian, uint32(len(val)))
			w.buf.WriteString(val)
			w.bytes += 1 + 4 + int64(len(val))
		case float64:
			w.buf.WriteByte(2)
			binary.Write(w.buf, binary.LittleEndian, val)
			w.bytes += 9
		case bool:
			w.buf.WriteByte(3)
			if val {
				w.buf.WriteByte(1)
			} else {
				w.buf.WriteByte(0)
			}
			w.bytes += 2
		case int64:
			w.buf.WriteByte(4)
			binary.Write(w.buf, binary.LittleEndian, val)
			w.bytes += 9
		case int:
			w.buf.WriteByte(4)
			binary.Write(w.buf, binary.LittleEndian, int64(val))
			w.bytes += 9
		default:
			s := fmt.Sprint(val)
			w.buf.WriteByte(1)
			binary.Write(w.buf, binary.LittleEndian, uint32(len(s)))
			w.buf.WriteString(s)
			w.bytes += 1 + 4 + int64(len(s))
		}
	}
	w.count++
	written := w.bytes - startBytes
	w.mgr.totalSize.Add(written)
	if w.mgr.totalSize.Load() > w.mgr.maxDisk {
		return fmt.Errorf("spill exceeded disk budget (%d MB)", w.mgr.maxDisk>>20)
	}
	return nil
}

func (w *SpillWriter) Close() error { w.buf.Flush(); return w.file.Close() }
func (w *SpillWriter) Path() string { return w.path }
func (w *SpillWriter) Count() int64 { return w.count }

type SpillReader struct {
	file *os.File
	buf  *bufio.Reader
}

func OpenSpillReader(path string) (*SpillReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &SpillReader{file: f, buf: bufio.NewReaderSize(f, 128<<10)}, nil
}

func (r *SpillReader) NextRow() (Row, error) {
	var fieldCount uint32
	if err := binary.Read(r.buf, binary.LittleEndian, &fieldCount); err != nil {
		return nil, err
	}
	row := make(Row, fieldCount)
	for i := uint32(0); i < fieldCount; i++ {
		var keyLen uint16
		if err := binary.Read(r.buf, binary.LittleEndian, &keyLen); err != nil {
			return nil, err
		}
		keyBytes := make([]byte, keyLen)
		if _, err := io.ReadFull(r.buf, keyBytes); err != nil {
			return nil, err
		}
		key := string(keyBytes)
		tag, err := r.buf.ReadByte()
		if err != nil {
			return nil, err
		}
		switch tag {
		case 0:
			row[key] = nil
		case 1:
			var slen uint32
			binary.Read(r.buf, binary.LittleEndian, &slen)
			sb := make([]byte, slen)
			io.ReadFull(r.buf, sb)
			row[key] = string(sb)
		case 2:
			var f float64
			binary.Read(r.buf, binary.LittleEndian, &f)
			row[key] = f
		case 3:
			b, _ := r.buf.ReadByte()
			row[key] = b == 1
		case 4:
			var n int64
			binary.Read(r.buf, binary.LittleEndian, &n)
			row[key] = n
		default:
			row[key] = nil
		}
	}
	return row, nil
}

func (r *SpillReader) Close() error { return r.file.Close() }

type SpillingSort struct {
	specs    []SortSpec
	memLimit int64
	mgr      *SpillManager
	tracker  *MemTracker
}

func NewSpillingSort(specs []SortSpec, memLimit int64, mgr *SpillManager, t *MemTracker) *SpillingSort {
	return &SpillingSort{specs: specs, memLimit: memLimit, mgr: mgr, tracker: t}
}

func (ss *SpillingSort) Sort(ctx context.Context, in <-chan Row, out chan<- Row, prof *StageProfiler) error {
	var chunks []*SpillWriter
	var chunk []Row
	var mem int64
	i := 0
	for row := range in {
		if cancelled(ctx, i) {
			return ctx.Err()
		}
		i++
		prof.TrackRowIn()
		chunk = append(chunk, row)
		mem += EstimateRowBytes(row)
		if mem >= ss.memLimit {
			sortRows(chunk, ss.specs)
			w, err := ss.mgr.NewWriter("sort")
			if err != nil {
				return err
			}
			for _, r := range chunk {
				if err := w.WriteRow(r); err != nil {
					w.Close()
					return err
				}
			}
			w.Close()
			chunks = append(chunks, w)
			chunk = chunk[:0]
			mem = 0
		}
	}
	if len(chunks) == 0 {
		sortRows(chunk, ss.specs)
		for _, r := range chunk {
			prof.TrackRowOut()
			select {
			case out <- r:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		prof.SetDetail("spilled", false)
		return nil
	}
	if len(chunk) > 0 {
		sortRows(chunk, ss.specs)
		w, _ := ss.mgr.NewWriter("sort")
		for _, r := range chunk {
			w.WriteRow(r)
		}
		w.Close()
		chunks = append(chunks, w)
	}
	prof.SetDetail("spilled", true)
	prof.SetDetail("chunks", len(chunks))
	return ss.heapMerge(ctx, chunks, out, prof)
}

type mergeItem struct {
	row    Row
	source int
}
type mergeHeap struct {
	items []mergeItem
	specs []SortSpec
}

func (h mergeHeap) Len() int            { return len(h.items) }
func (h mergeHeap) Less(i, j int) bool  { return !shouldSwap(h.items[i].row, h.items[j].row, h.specs) }
func (h mergeHeap) Swap(i, j int)       { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *mergeHeap) Push(x interface{}) { h.items = append(h.items, x.(mergeItem)) }
func (h *mergeHeap) Pop() interface{} {
	old := h.items
	n := len(old)
	it := old[n-1]
	h.items = old[:n-1]
	return it
}

func (ss *SpillingSort) heapMerge(ctx context.Context, chunks []*SpillWriter, out chan<- Row, prof *StageProfiler) error {
	readers := make([]*SpillReader, len(chunks))
	for i, c := range chunks {
		r, err := OpenSpillReader(c.Path())
		if err != nil {
			return err
		}
		defer r.Close()
		readers[i] = r
	}
	h := &mergeHeap{specs: ss.specs}
	heap.Init(h)
	for i, r := range readers {
		row, err := r.NextRow()
		if err == nil {
			heap.Push(h, mergeItem{row: row, source: i})
		}
	}
	j := 0
	for h.Len() > 0 {
		if cancelled(ctx, j) {
			return ctx.Err()
		}
		j++
		item := heap.Pop(h).(mergeItem)
		prof.TrackRowOut()
		select {
		case out <- item.row:
		case <-ctx.Done():
			return ctx.Err()
		}
		nextRow, err := readers[item.source].NextRow()
		if err == nil {
			heap.Push(h, mergeItem{row: nextRow, source: item.source})
		}
	}
	return nil
}

type SpillingAggregator struct {
	groups     map[string]map[string]AggState
	aggDefs    []AggExpr
	memLimit   int64
	currMem    int64
	mgr        *SpillManager
	tracker    *MemTracker
	spilled    bool
	spillFiles []*SpillWriter
}

func NewSpillingAggregator(aggs []AggExpr, memLimit int64, mgr *SpillManager, t *MemTracker) *SpillingAggregator {
	return &SpillingAggregator{
		groups: make(map[string]map[string]AggState), aggDefs: aggs,
		memLimit: memLimit, mgr: mgr, tracker: t,
	}
}

func (sa *SpillingAggregator) Add(key string, row Row) error {
	states, exists := sa.groups[key]
	if !exists {
		states = make(map[string]AggState, len(sa.aggDefs))
		for _, a := range sa.aggDefs {
			states[a.Alias] = NewAggState(a.Func)
		}
		sa.groups[key] = states
		sa.currMem += int64(len(key)) + 64
	}
	for _, a := range sa.aggDefs {
		states[a.Alias].Update(row, a.Field)
	}
	if sa.currMem >= sa.memLimit {
		return sa.spill()
	}
	return nil
}

func (sa *SpillingAggregator) spill() error {
	sa.spilled = true
	w, err := sa.mgr.NewWriter("agg")
	if err != nil {
		return err
	}
	for key, states := range sa.groups {
		row := Row{"_gk": key}
		for alias, state := range states {
			row["_fn_"+alias] = state.FuncName()
			row[alias] = state.Finalize()
			switch s := state.(type) {
			case *AvgState:
				row["_cnt_"+alias] = s.Count
				row["_sum_"+alias] = s.Total
			case *StdevState:
				row["_cnt_"+alias] = s.Count
				row["_mean_"+alias] = s.Mean
				row["_m2_"+alias] = s.M2
			case *VarianceState:
				row["_cnt_"+alias] = s.Count
				row["_mean_"+alias] = s.Mean
				row["_m2_"+alias] = s.M2
			case *CountState:
				row["_cnt_"+alias] = s.N
			case *SumState:
				row["_sum_"+alias] = s.Total
				row["_cnt_"+alias] = s.Count
			case *MinState:
				if s.set {
					row["_min_"+alias] = s.val
				}
			case *MaxState:
				if s.set {
					row["_max_"+alias] = s.val
				}
			case *RangeState:
				if s.set {
					row["_min_"+alias] = s.min
					row["_max_"+alias] = s.max
				}
			}
		}
		if err := w.WriteRow(row); err != nil {
			w.Close()
			return err
		}
	}
	w.Close()
	sa.spillFiles = append(sa.spillFiles, w)
	sa.groups = make(map[string]map[string]AggState)
	sa.currMem = 0
	return nil
}

func (sa *SpillingAggregator) Spilled() bool { return sa.spilled }

func (sa *SpillingAggregator) Finalize(groupByFields []FieldRef) ([]Row, error) {
	if !sa.spilled {
		var out []Row
		for key, states := range sa.groups {
			row := make(Row)
			for alias, state := range states {
				row[alias] = state.Finalize()
			}
			addGroupByFields(row, key, groupByFields)
			out = append(out, row)
		}
		return out, nil
	}
	merged := make(map[string]map[string]AggState)
	for key, states := range sa.groups {
		cloned := make(map[string]AggState, len(states))
		for alias, state := range states {
			cloned[alias] = state.Clone()
		}
		merged[key] = cloned
	}
	for _, sf := range sa.spillFiles {
		reader, err := OpenSpillReader(sf.Path())
		if err != nil {
			return nil, err
		}
		for {
			row, err := reader.NextRow()
			if err != nil {
				break
			}
			key, _ := row["_gk"].(string)
			existing, exists := merged[key]
			if !exists {
				existing = make(map[string]AggState, len(sa.aggDefs))
				for _, a := range sa.aggDefs {
					existing[a.Alias] = NewAggState(a.Func)
				}
				merged[key] = existing
			}
			for _, a := range sa.aggDefs {
				spilledState := NewAggState(a.Func)
				restoreSpilledState(spilledState, a.Alias, row)
				existing[a.Alias].Merge(spilledState)
			}
		}
		reader.Close()
	}
	var out []Row
	for key, states := range merged {
		row := make(Row)
		for alias, state := range states {
			row[alias] = state.Finalize()
		}
		addGroupByFields(row, key, groupByFields)
		out = append(out, row)
	}
	return out, nil
}

func restoreSpilledState(state AggState, alias string, row Row) {
	switch s := state.(type) {
	case *CountState:
		if v, ok := row["_cnt_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.N = int64(n)
			}
		}
	case *SumState:
		if v, ok := row["_sum_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.Total = n
			}
		}
		if v, ok := row["_cnt_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.Count = int64(n)
			}
		}
	case *AvgState:
		if v, ok := row["_sum_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.Total = n
			}
		}
		if v, ok := row["_cnt_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.Count = int64(n)
			}
		}
	case *MinState:
		if v, ok := row["_min_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.val = n
				s.set = true
			}
		}
	case *MaxState:
		if v, ok := row["_max_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.val = n
				s.set = true
			}
		}
	case *StdevState:
		if v, ok := row["_cnt_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.Count = int64(n)
			}
		}
		if v, ok := row["_mean_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.Mean = n
			}
		}
		if v, ok := row["_m2_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.M2 = n
			}
		}
	case *VarianceState:
		if v, ok := row["_cnt_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.Count = int64(n)
			}
		}
		if v, ok := row["_mean_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.Mean = n
			}
		}
		if v, ok := row["_m2_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.M2 = n
			}
		}
	case *RangeState:
		if v, ok := row["_min_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.min = n
				s.set = true
			}
		}
		if v, ok := row["_max_"+alias]; ok {
			if n, ok := ToNumber(v); ok {
				s.max = n
				s.set = true
			}
		}
	}
}

var _ = math.MaxFloat64
