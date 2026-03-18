package oql

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type QueryScheduler struct {
	maxConcurrent, maxPerUser, maxPerTenant, maxQueueDepth int
	maxQueueWait                                           time.Duration
	active                                                 atomic.Int64
	activePerUser, activePerTenant                          sync.Map
	mu                                                     sync.Mutex
	userQueues                                             map[string]*list.List
	userOrder                                              []string
	totalQueue                                             int
	totalSubmitted, totalExecuted, totalRejected, totalTimedOut atomic.Int64
}

type QuerySlot struct {
	QueryID, User, Tenant string
	Priority              int
	Acquired              time.Time
	sched                 *QueryScheduler
}

type SchedulerConfig struct {
	MaxConcurrent, MaxPerUser, MaxPerTenant, MaxQueueDepth int
	MaxQueueWait                                           time.Duration
}

var DefaultSchedulerConfig = SchedulerConfig{
	MaxConcurrent: 20, MaxPerUser: 5, MaxPerTenant: 10,
	MaxQueueDepth: 100, MaxQueueWait: 30 * time.Second,
}

func NewQueryScheduler(c SchedulerConfig) *QueryScheduler {
	return &QueryScheduler{
		maxConcurrent: c.MaxConcurrent, maxPerUser: c.MaxPerUser,
		maxPerTenant: c.MaxPerTenant, maxQueueDepth: c.MaxQueueDepth,
		maxQueueWait: c.MaxQueueWait, userQueues: make(map[string]*list.List),
	}
}

type queueEntry struct {
	queryID, user, tenant string
	priority              int
	enqueued              time.Time
	ready                 chan struct{}
}

func CostAwarePriority(basePriority int, cost QueryCost) int {
	adj := basePriority
	if cost.EstimatedBytes > 1<<30 {
		adj++
	}
	if cost.EstimatedBytes > 10<<30 {
		adj++
	}
	if cost.EstimatedEvents > 10_000_000 {
		adj++
	}
	if adj > 5 {
		adj = 5
	}
	return adj
}

func (qs *QueryScheduler) Acquire(ctx context.Context, queryID, user, tenant string, priority int) (*QuerySlot, error) {
	qs.totalSubmitted.Add(1)
	if qs.tryAcquire(user, tenant) {
		qs.totalExecuted.Add(1)
		return &QuerySlot{QueryID: queryID, User: user, Tenant: tenant, Priority: priority, Acquired: time.Now(), sched: qs}, nil
	}
	qs.mu.Lock()
	if qs.totalQueue >= qs.maxQueueDepth {
		qs.mu.Unlock()
		qs.totalRejected.Add(1)
		return nil, fmt.Errorf("scheduler: queue full (%d)", qs.totalQueue)
	}
	entry := &queueEntry{queryID: queryID, user: user, tenant: tenant, priority: priority, enqueued: time.Now(), ready: make(chan struct{})}
	q, exists := qs.userQueues[user]
	if !exists {
		q = list.New()
		qs.userQueues[user] = q
		qs.userOrder = append(qs.userOrder, user)
	}
	inserted := false
	for e := q.Front(); e != nil; e = e.Next() {
		if entry.priority < e.Value.(*queueEntry).priority {
			q.InsertBefore(entry, e)
			inserted = true
			break
		}
	}
	if !inserted {
		q.PushBack(entry)
	}
	qs.totalQueue++
	qs.mu.Unlock()

	waitCtx, cancel := context.WithTimeout(ctx, qs.maxQueueWait)
	defer cancel()
	select {
	case <-entry.ready:
		qs.totalExecuted.Add(1)
		return &QuerySlot{QueryID: queryID, User: user, Tenant: tenant, Priority: priority, Acquired: time.Now(), sched: qs}, nil
	case <-waitCtx.Done():
		qs.mu.Lock()
		qs.removeEntry(user, entry)
		qs.mu.Unlock()
		qs.totalTimedOut.Add(1)
		return nil, fmt.Errorf("scheduler: timed out (%d running, %d queued)", qs.active.Load(), qs.totalQueue)
	}
}

func (slot *QuerySlot) Release() {
	qs := slot.sched
	qs.active.Add(-1)
	qs.decUser(slot.User)
	if slot.Tenant != "" {
		qs.decTenant(slot.Tenant)
	}
	qs.processQueue()
}

func (qs *QueryScheduler) tryAcquire(user, tenant string) bool {
	if qs.active.Load() >= int64(qs.maxConcurrent) {
		return false
	}
	if qs.getUserCount(user) >= int64(qs.maxPerUser) {
		return false
	}
	if tenant != "" && qs.getTenantCount(tenant) >= int64(qs.maxPerTenant) {
		return false
	}
	qs.active.Add(1)
	qs.incUser(user)
	if tenant != "" {
		qs.incTenant(tenant)
	}
	return true
}

func (qs *QueryScheduler) processQueue() {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	if qs.totalQueue == 0 {
		return
	}
	maxServe := qs.maxConcurrent - int(qs.active.Load())
	if maxServe <= 0 {
		return
	}
	served := 0
	for round := 0; round < 2 && served < maxServe; round++ {
		for _, user := range qs.userOrder {
			if served >= maxServe {
				break
			}
			q, exists := qs.userQueues[user]
			if !exists || q.Len() == 0 {
				continue
			}
			front := q.Front()
			if front == nil {
				continue
			}
			entry := front.Value.(*queueEntry)
			if qs.tryAcquire(entry.user, entry.tenant) {
				q.Remove(front)
				qs.totalQueue--
				close(entry.ready)
				served++
			}
		}
	}
}

func (qs *QueryScheduler) removeEntry(user string, entry *queueEntry) {
	if q, ok := qs.userQueues[user]; ok {
		for e := q.Front(); e != nil; e = e.Next() {
			if e.Value.(*queueEntry) == entry {
				q.Remove(e)
				qs.totalQueue--
				break
			}
		}
		if q.Len() == 0 {
			delete(qs.userQueues, user)
			for i, u := range qs.userOrder {
				if u == user {
					qs.userOrder = append(qs.userOrder[:i], qs.userOrder[i+1:]...)
					break
				}
			}
		}
	}
}

func (qs *QueryScheduler) getUserCount(u string) int64 {
	if v, ok := qs.activePerUser.Load(u); ok {
		return v.(*atomic.Int64).Load()
	}
	return 0
}
func (qs *QueryScheduler) incUser(u string) {
	v, _ := qs.activePerUser.LoadOrStore(u, &atomic.Int64{})
	v.(*atomic.Int64).Add(1)
}
func (qs *QueryScheduler) decUser(u string) {
	if v, ok := qs.activePerUser.Load(u); ok {
		v.(*atomic.Int64).Add(-1)
	}
}
func (qs *QueryScheduler) getTenantCount(t string) int64 {
	if v, ok := qs.activePerTenant.Load(t); ok {
		return v.(*atomic.Int64).Load()
	}
	return 0
}
func (qs *QueryScheduler) incTenant(t string) {
	v, _ := qs.activePerTenant.LoadOrStore(t, &atomic.Int64{})
	v.(*atomic.Int64).Add(1)
}
func (qs *QueryScheduler) decTenant(t string) {
	if v, ok := qs.activePerTenant.Load(t); ok {
		v.(*atomic.Int64).Add(-1)
	}
}

func (qs *QueryScheduler) Stats() SchedulerStats {
	return SchedulerStats{
		ActiveQueries: qs.active.Load(), QueueDepth: int64(qs.totalQueue),
		MaxConcurrent: int64(qs.maxConcurrent), TotalSubmitted: qs.totalSubmitted.Load(),
		TotalExecuted: qs.totalExecuted.Load(), TotalRejected: qs.totalRejected.Load(),
		TotalTimedOut: qs.totalTimedOut.Load(),
	}
}

type SchedulerStats struct {
	ActiveQueries, QueueDepth, MaxConcurrent                    int64
	TotalSubmitted, TotalExecuted, TotalRejected, TotalTimedOut int64
}
