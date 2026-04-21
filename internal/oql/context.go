package oql

import (
	"crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
	"time"
)

type EvalContext struct {
	Now       time.Time
	QueryID   string
	User      string
	Role      string
	QueryType string
	TimeRange TimeRange
	rng       *mathrand.Rand
	secure    bool
}

func NewEvalContext(queryID, user string, tr TimeRange) *EvalContext {
	seed := int64(0)
	for _, b := range []byte(queryID) {
		seed = seed*31 + int64(b)
	}
	return &EvalContext{
		Now:       time.Now().UTC(),
		QueryID:   queryID,
		User:      user,
		TimeRange: tr,
		rng:       mathrand.New(mathrand.NewSource(seed)),
	}
}

func NewSecureEvalContext(queryID, user string, tr TimeRange) *EvalContext {
	ctx := NewEvalContext(queryID, user, tr)
	ctx.secure = true
	return ctx
}

func (ctx *EvalContext) Random() float64 {
	if ctx.secure {
		var buf [8]byte
		rand.Read(buf[:])
		return float64(binary.LittleEndian.Uint64(buf[:])) / float64(^uint64(0))
	}
	return ctx.rng.Float64()
}

func (ctx *EvalContext) NowTime() time.Time                     { return ctx.Now }
func (ctx *EvalContext) RelativeTime(d time.Duration) time.Time { return ctx.Now.Add(d) }
func (ctx *EvalContext) IsSecure() bool                         { return ctx.secure }
