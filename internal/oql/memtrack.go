package oql

import (
	"runtime"
	"sync/atomic"
	"time"
)

type MemTracker struct {
	tracked  atomic.Int64
	monitor  *BudgetMonitor
	baseHeap uint64
	done     chan struct{}
}

func NewMemTracker(monitor *BudgetMonitor) *MemTracker {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	mt := &MemTracker{monitor: monitor, baseHeap: ms.HeapInuse, done: make(chan struct{})}
	go mt.loop()
	return mt
}

func (mt *MemTracker) Stop() {
	select {
	case <-mt.done:
	default:
		close(mt.done)
	}
}
func (mt *MemTracker) Alloc(n int64) { mt.tracked.Add(n); mt.monitor.TrackMemory(n) }
func (mt *MemTracker) Free(n int64)  { mt.tracked.Add(-n); mt.monitor.TrackMemory(-n) }

func (mt *MemTracker) loop() {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			heapDelta := int64(0)
			if ms.HeapInuse > mt.baseHeap {
				heapDelta = int64(ms.HeapInuse - mt.baseHeap)
			}
			tracked := mt.tracked.Load()
			if heapDelta > tracked*3 && heapDelta-tracked > 50<<20 {
				correction := (heapDelta - tracked) / 4
				mt.monitor.TrackMemory(correction)
				mt.tracked.Add(correction)
			}
		case <-mt.done:
			return
		}
	}
}

func EstimateMapOverhead(entries int) int64 {
	return (int64(entries/6) + 1) * 208
}

type TrackedMap struct {
	data    map[string]interface{}
	tracker *MemTracker
	size    int64
}

func NewTrackedMap(t *MemTracker) *TrackedMap {
	o := EstimateMapOverhead(0)
	t.Alloc(o)
	return &TrackedMap{data: make(map[string]interface{}), tracker: t, size: o}
}
func (m *TrackedMap) Set(k string, v interface{}) {
	if _, exists := m.data[k]; !exists {
		s := int64(len(k)) + 56
		m.tracker.Alloc(s)
		m.size += s
		if n := len(m.data); n&(n-1) == 0 && n > 8 {
			g := EstimateMapOverhead(n) - EstimateMapOverhead(n/2)
			m.tracker.Alloc(g)
			m.size += g
		}
	}
	m.data[k] = v
}
func (m *TrackedMap) Get(k string) (interface{}, bool) { v, ok := m.data[k]; return v, ok }
func (m *TrackedMap) Len() int                         { return len(m.data) }
func (m *TrackedMap) Release()                         { m.tracker.Free(m.size); m.data = nil }

type TrackedBuffer struct {
	data    []Row
	tracker *MemTracker
	size    int64
}

func NewTrackedBuffer(t *MemTracker) *TrackedBuffer {
	t.Alloc(24)
	return &TrackedBuffer{tracker: t, size: 24}
}
func (b *TrackedBuffer) Append(r Row) {
	s := EstimateRowBytes(r)
	b.tracker.Alloc(s)
	b.size += s
	b.data = append(b.data, r)
	if n := len(b.data); n > 1 && n&(n-1) == 0 {
		g := int64(n) * 8
		b.tracker.Alloc(g)
		b.size += g
	}
}
func (b *TrackedBuffer) Rows() []Row { return b.data }
func (b *TrackedBuffer) Len() int    { return len(b.data) }
func (b *TrackedBuffer) Release()    { b.tracker.Free(b.size); b.data = nil }
