package oql

import (
	"context"
	"sync/atomic"
)

type BackpressureChannel struct {
	ch         chan Row
	sendBlocks atomic.Int64
	totalSent  atomic.Int64
}

func NewBackpressureChannel(size int) *BackpressureChannel {
	return &BackpressureChannel{ch: make(chan Row, size)}
}

func (bc *BackpressureChannel) Send(ctx context.Context, row Row) error {
	select {
	case bc.ch <- row:
		bc.totalSent.Add(1)
		return nil
	default:
		bc.sendBlocks.Add(1)
		select {
		case bc.ch <- row:
			bc.totalSent.Add(1)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (bc *BackpressureChannel) Recv() <-chan Row { return bc.ch }
func (bc *BackpressureChannel) Close()          { close(bc.ch) }

func (bc *BackpressureChannel) Stats() BackpressureStats {
	sent := bc.totalSent.Load()
	pct := float64(0)
	if sent > 0 {
		pct = float64(bc.sendBlocks.Load()) / float64(sent) * 100
	}
	return BackpressureStats{TotalSent: sent, SendBlocks: bc.sendBlocks.Load(), BackpressurePct: pct, QueueDepth: len(bc.ch), QueueCap: cap(bc.ch)}
}

type BackpressureStats struct {
	TotalSent, SendBlocks int64
	BackpressurePct       float64
	QueueDepth, QueueCap  int
}
