package oql

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"
)

type AggState interface {
	Update(row Row, field *FieldRef)
	Merge(other AggState)
	Finalize() interface{}
	Clone() AggState
	SizeBytes() int64
	FuncName() string
}

type AggLimits struct {
	MaxDistinct int
	MaxValues   int
	MaxMedian   int
}

var DefaultAggLimits = AggLimits{MaxDistinct: 10_000, MaxValues: 1_000, MaxMedian: 100_000}
var globalAggLimits = DefaultAggLimits

func SetAggLimits(l AggLimits) { globalAggLimits = l }

func NewAggState(funcName string) AggState {
	switch funcName {
	case "count":
		return &CountState{}
	case "sum":
		return &SumState{}
	case "avg":
		return &AvgState{}
	case "min":
		return &MinState{val: math.MaxFloat64}
	case "max":
		return &MaxState{val: -math.MaxFloat64}
	case "dc", "distinct_count":
		return &DistinctState{exact: make(map[string]struct{}), limit: globalAggLimits.MaxDistinct}
	case "values", "list":
		return &ValuesState{limit: globalAggLimits.MaxValues}
	case "first":
		return &FirstState{}
	case "last":
		return &LastState{}
	case "median":
		return &MedianState{limit: globalAggLimits.MaxMedian}
	case "stdev":
		return &StdevState{}
	case "var":
		return &VarianceState{}
	case "range":
		return &RangeState{min: math.MaxFloat64, max: -math.MaxFloat64}
	default:
		return &CountState{}
	}
}

type CountState struct{ N int64 }

func (s *CountState) Update(_ Row, _ *FieldRef) { s.N++ }
func (s *CountState) Merge(o AggState)          { if x, ok := o.(*CountState); ok { s.N += x.N } }
func (s *CountState) Finalize() interface{}     { return s.N }
func (s *CountState) Clone() AggState           { return &CountState{N: s.N} }
func (s *CountState) SizeBytes() int64          { return 8 }
func (s *CountState) FuncName() string          { return "count" }

type SumState struct{ Total float64; Count int64 }

func (s *SumState) Update(row Row, f *FieldRef) {
	if f != nil {
		if v, ok := ToNumber(row[f.Canonical()]); ok {
			s.Total += v
			s.Count++
		}
	}
}
func (s *SumState) Merge(o AggState) {
	if x, ok := o.(*SumState); ok {
		s.Total += x.Total
		s.Count += x.Count
	}
}
func (s *SumState) Finalize() interface{} { return s.Total }
func (s *SumState) Clone() AggState       { return &SumState{Total: s.Total, Count: s.Count} }
func (s *SumState) SizeBytes() int64      { return 16 }
func (s *SumState) FuncName() string      { return "sum" }

type AvgState struct{ Total float64; Count int64 }

func (s *AvgState) Update(row Row, f *FieldRef) {
	if f != nil {
		if v, ok := ToNumber(row[f.Canonical()]); ok {
			s.Total += v
			s.Count++
		}
	}
}
func (s *AvgState) Merge(o AggState) {
	if x, ok := o.(*AvgState); ok {
		s.Total += x.Total
		s.Count += x.Count
	}
}
func (s *AvgState) Finalize() interface{} {
	if s.Count == 0 {
		return nil
	}
	return s.Total / float64(s.Count)
}
func (s *AvgState) Clone() AggState  { return &AvgState{Total: s.Total, Count: s.Count} }
func (s *AvgState) SizeBytes() int64 { return 16 }
func (s *AvgState) FuncName() string { return "avg" }

type MinState struct{ val float64; set bool }

func (s *MinState) Update(row Row, f *FieldRef) {
	if f != nil {
		if v, ok := ToNumber(row[f.Canonical()]); ok && (!s.set || v < s.val) {
			s.val = v
			s.set = true
		}
	}
}
func (s *MinState) Merge(o AggState) {
	if x, ok := o.(*MinState); ok && x.set && (!s.set || x.val < s.val) {
		s.val = x.val
		s.set = true
	}
}
func (s *MinState) Finalize() interface{} {
	if !s.set {
		return nil
	}
	return s.val
}
func (s *MinState) Clone() AggState  { return &MinState{val: s.val, set: s.set} }
func (s *MinState) SizeBytes() int64 { return 16 }
func (s *MinState) FuncName() string { return "min" }

type MaxState struct{ val float64; set bool }

func (s *MaxState) Update(row Row, f *FieldRef) {
	if f != nil {
		if v, ok := ToNumber(row[f.Canonical()]); ok && (!s.set || v > s.val) {
			s.val = v
			s.set = true
		}
	}
}
func (s *MaxState) Merge(o AggState) {
	if x, ok := o.(*MaxState); ok && x.set && (!s.set || x.val > s.val) {
		s.val = x.val
		s.set = true
	}
}
func (s *MaxState) Finalize() interface{} {
	if !s.set {
		return nil
	}
	return s.val
}
func (s *MaxState) Clone() AggState  { return &MaxState{val: s.val, set: s.set} }
func (s *MaxState) SizeBytes() int64 { return 16 }
func (s *MaxState) FuncName() string { return "max" }

type DistinctState struct {
	exact       map[string]struct{}
	hll         *hllSketch
	approximate bool
	limit       int
}

func (s *DistinctState) Update(row Row, f *FieldRef) {
	if f == nil {
		return
	}
	v, ok := row[f.Canonical()]
	if !ok {
		return
	}
	str := fmt.Sprint(v)
	if s.approximate {
		s.hll.Add(str)
		return
	}
	s.exact[str] = struct{}{}
	if len(s.exact) > s.limit {
		s.hll = newHLLSketch(14)
		for k := range s.exact {
			s.hll.Add(k)
		}
		s.exact = nil
		s.approximate = true
	}
}

func (s *DistinctState) Merge(o AggState) {
	x, ok := o.(*DistinctState)
	if !ok {
		return
	}
	if !s.approximate && !x.approximate {
		for k := range x.exact {
			s.exact[k] = struct{}{}
		}
		if len(s.exact) > s.limit {
			s.hll = newHLLSketch(14)
			for k := range s.exact {
				s.hll.Add(k)
			}
			s.exact = nil
			s.approximate = true
		}
		return
	}
	if !s.approximate {
		s.hll = newHLLSketch(14)
		for k := range s.exact {
			s.hll.Add(k)
		}
		s.exact = nil
		s.approximate = true
	}
	if x.approximate {
		s.hll.Merge(x.hll)
	} else {
		for k := range x.exact {
			s.hll.Add(k)
		}
	}
}

func (s *DistinctState) Finalize() interface{} {
	if s.approximate {
		return int64(s.hll.Count())
	}
	return int64(len(s.exact))
}

func (s *DistinctState) Clone() AggState {
	if s.approximate {
		return &DistinctState{approximate: true, hll: s.hll.Clone(), limit: s.limit}
	}
	c := &DistinctState{exact: make(map[string]struct{}, len(s.exact)), limit: s.limit}
	for k := range s.exact {
		c.exact[k] = struct{}{}
	}
	return c
}

func (s *DistinctState) SizeBytes() int64 {
	if s.approximate {
		return int64(s.hll.Size())
	}
	return EstimateMapOverhead(len(s.exact)) + int64(len(s.exact))*32
}
func (s *DistinctState) FuncName() string { return "dc" }

type hllSketch struct {
	p         uint8
	m         uint32
	registers []uint8
}

func newHLLSketch(precision uint8) *hllSketch {
	m := uint32(1) << precision
	return &hllSketch{p: precision, m: m, registers: make([]uint8, m)}
}

func (h *hllSketch) Add(value string) {
	hash := fnvHash64(value)
	idx := hash >> (64 - h.p)
	remaining := (hash << h.p) | (1 << (h.p - 1))
	rank := uint8(1)
	for remaining&(1<<63) == 0 && rank < 64 {
		rank++
		remaining <<= 1
	}
	if rank > h.registers[idx] {
		h.registers[idx] = rank
	}
}

func (h *hllSketch) Count() uint64 {
	sum := 0.0
	zeros := 0
	for _, v := range h.registers {
		sum += 1.0 / float64(uint64(1)<<v)
		if v == 0 {
			zeros++
		}
	}
	m := float64(h.m)
	alpha := 0.7213 / (1 + 1.079/m)
	estimate := alpha * m * m / sum
	if estimate <= 2.5*m && zeros > 0 {
		estimate = m * math.Log(m/float64(zeros))
	}
	return uint64(estimate)
}

func (h *hllSketch) Merge(other *hllSketch) {
	if other == nil || h.m != other.m {
		return
	}
	for i := uint32(0); i < h.m; i++ {
		if other.registers[i] > h.registers[i] {
			h.registers[i] = other.registers[i]
		}
	}
}

func (h *hllSketch) Clone() *hllSketch {
	c := &hllSketch{p: h.p, m: h.m, registers: make([]uint8, h.m)}
	copy(c.registers, h.registers)
	return c
}

func (h *hllSketch) Size() int { return int(h.m) + 16 }

func fnvHash64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

type ValuesState struct {
	vals      []interface{}
	limit     int
	truncated bool
}

func (s *ValuesState) Update(row Row, f *FieldRef) {
	if f == nil {
		return
	}
	if v, ok := row[f.Canonical()]; ok {
		if len(s.vals) < s.limit {
			s.vals = append(s.vals, v)
		} else {
			s.truncated = true
		}
	}
}
func (s *ValuesState) Merge(o AggState) {
	if x, ok := o.(*ValuesState); ok {
		rem := s.limit - len(s.vals)
		if rem <= 0 {
			s.truncated = true
			return
		}
		if len(x.vals) <= rem {
			s.vals = append(s.vals, x.vals...)
		} else {
			s.vals = append(s.vals, x.vals[:rem]...)
			s.truncated = true
		}
	}
}
func (s *ValuesState) Finalize() interface{} { return s.vals }
func (s *ValuesState) Clone() AggState {
	c := &ValuesState{vals: make([]interface{}, len(s.vals)), limit: s.limit, truncated: s.truncated}
	copy(c.vals, s.vals)
	return c
}
func (s *ValuesState) SizeBytes() int64 { return int64(len(s.vals)) * 24 }
func (s *ValuesState) FuncName() string { return "values" }

type FirstState struct{ val interface{}; set bool }

func (s *FirstState) Update(row Row, f *FieldRef) {
	if s.set || f == nil {
		return
	}
	if v, ok := row[f.Canonical()]; ok {
		s.val = v
		s.set = true
	}
}
func (s *FirstState) Merge(o AggState) {
	if !s.set {
		if x, ok := o.(*FirstState); ok && x.set {
			s.val = x.val
			s.set = true
		}
	}
}
func (s *FirstState) Finalize() interface{} { return s.val }
func (s *FirstState) Clone() AggState       { return &FirstState{val: s.val, set: s.set} }
func (s *FirstState) SizeBytes() int64      { return 24 }
func (s *FirstState) FuncName() string      { return "first" }

type LastState struct{ val interface{} }

func (s *LastState) Update(row Row, f *FieldRef) {
	if f != nil {
		if v, ok := row[f.Canonical()]; ok {
			s.val = v
		}
	}
}
func (s *LastState) Merge(o AggState) {
	if x, ok := o.(*LastState); ok && x.val != nil {
		s.val = x.val
	}
}
func (s *LastState) Finalize() interface{} { return s.val }
func (s *LastState) Clone() AggState       { return &LastState{val: s.val} }
func (s *LastState) SizeBytes() int64      { return 24 }
func (s *LastState) FuncName() string      { return "last" }

type MedianState struct {
	vals      []float64
	limit     int
	count     int64
	reservoir bool
}

func (s *MedianState) Update(row Row, f *FieldRef) {
	if f == nil {
		return
	}
	v, ok := ToNumber(row[f.Canonical()])
	if !ok {
		return
	}
	s.count++
	if len(s.vals) < s.limit {
		s.vals = append(s.vals, v)
	} else {
		s.reservoir = true
		idx := int(s.count % int64(s.limit))
		s.vals[idx] = v
	}
}

func (s *MedianState) Merge(o AggState) {
	if x, ok := o.(*MedianState); ok {
		rem := s.limit - len(s.vals)
		if rem > 0 {
			take := len(x.vals)
			if take > rem {
				take = rem
			}
			s.vals = append(s.vals, x.vals[:take]...)
		}
		s.count += x.count
	}
}

func (s *MedianState) Finalize() interface{} {
	if len(s.vals) == 0 {
		return nil
	}
	sorted := make([]float64, len(s.vals))
	copy(sorted, s.vals)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}
func (s *MedianState) Clone() AggState {
	c := &MedianState{vals: make([]float64, len(s.vals)), limit: s.limit, count: s.count, reservoir: s.reservoir}
	copy(c.vals, s.vals)
	return c
}
func (s *MedianState) SizeBytes() int64 { return int64(len(s.vals)) * 8 }
func (s *MedianState) FuncName() string { return "median" }

type StdevState struct{ Count int64; Mean, M2 float64 }

func (s *StdevState) Update(row Row, f *FieldRef) {
	if f == nil {
		return
	}
	v, ok := ToNumber(row[f.Canonical()])
	if !ok {
		return
	}
	s.Count++
	delta := v - s.Mean
	s.Mean += delta / float64(s.Count)
	delta2 := v - s.Mean
	s.M2 += delta * delta2
}

func (s *StdevState) Merge(o AggState) {
	x, ok := o.(*StdevState)
	if !ok || x.Count == 0 {
		return
	}
	if s.Count == 0 {
		s.Count = x.Count
		s.Mean = x.Mean
		s.M2 = x.M2
		return
	}
	total := s.Count + x.Count
	delta := x.Mean - s.Mean
	s.M2 = s.M2 + x.M2 + delta*delta*float64(s.Count)*float64(x.Count)/float64(total)
	s.Mean = (s.Mean*float64(s.Count) + x.Mean*float64(x.Count)) / float64(total)
	s.Count = total
}

func (s *StdevState) Finalize() interface{} {
	if s.Count < 2 {
		return float64(0)
	}
	return math.Sqrt(s.M2 / float64(s.Count-1))
}
func (s *StdevState) Clone() AggState  { return &StdevState{Count: s.Count, Mean: s.Mean, M2: s.M2} }
func (s *StdevState) SizeBytes() int64 { return 24 }
func (s *StdevState) FuncName() string { return "stdev" }

type VarianceState struct{ StdevState }

func (s *VarianceState) Finalize() interface{} {
	if s.Count < 2 {
		return float64(0)
	}
	return s.M2 / float64(s.Count-1)
}
func (s *VarianceState) Clone() AggState  { return &VarianceState{StdevState{s.Count, s.Mean, s.M2}} }
func (s *VarianceState) FuncName() string { return "var" }

type RangeState struct{ min, max float64; set bool }

func (s *RangeState) Update(row Row, f *FieldRef) {
	if f == nil {
		return
	}
	v, ok := ToNumber(row[f.Canonical()])
	if !ok {
		return
	}
	if !s.set || v < s.min {
		s.min = v
	}
	if !s.set || v > s.max {
		s.max = v
	}
	s.set = true
}
func (s *RangeState) Merge(o AggState) {
	if x, ok := o.(*RangeState); ok && x.set {
		if !s.set || x.min < s.min {
			s.min = x.min
		}
		if !s.set || x.max > s.max {
			s.max = x.max
		}
		s.set = true
	}
}
func (s *RangeState) Finalize() interface{} {
	if !s.set {
		return nil
	}
	return s.max - s.min
}
func (s *RangeState) Clone() AggState  { return &RangeState{s.min, s.max, s.set} }
func (s *RangeState) SizeBytes() int64 { return 24 }
func (s *RangeState) FuncName() string { return "range" }
