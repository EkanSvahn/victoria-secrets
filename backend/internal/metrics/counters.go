package metrics

import "sync/atomic"

type Counters struct {
	createCount      atomic.Uint64
	consumeCount     atomic.Uint64
	notFoundCount    atomic.Uint64
	rateLimitedCount atomic.Uint64
}

type Snapshot struct {
	Create      uint64 `json:"create"`
	Consume     uint64 `json:"consume"`
	NotFound    uint64 `json:"not_found"`
	RateLimited uint64 `json:"rate_limited"`
}

func NewCounters() *Counters {
	return &Counters{}
}

func (c *Counters) IncCreate() {
	c.createCount.Add(1)
}

func (c *Counters) IncConsume() {
	c.consumeCount.Add(1)
}

func (c *Counters) IncNotFound() {
	c.notFoundCount.Add(1)
}

func (c *Counters) IncRateLimited() {
	c.rateLimitedCount.Add(1)
}

func (c *Counters) Snapshot() Snapshot {
	return Snapshot{
		Create:      c.createCount.Load(),
		Consume:     c.consumeCount.Load(),
		NotFound:    c.notFoundCount.Load(),
		RateLimited: c.rateLimitedCount.Load(),
	}
}
