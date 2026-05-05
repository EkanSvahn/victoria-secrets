package metrics

import (
	"fmt"
	"io"
	"sync/atomic"
)

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

// PrometheusContentType is the exposition format Prometheus servers expect
// when scraping the /api/metrics endpoint.
const PrometheusContentType = "text/plain; version=0.0.4; charset=utf-8"

func (c *Counters) WriteText(w io.Writer) error {
	snap := c.Snapshot()
	entries := []struct {
		name  string
		help  string
		value uint64
	}{
		{"ephemeral_secrets_created_total", "Number of secrets successfully created.", snap.Create},
		{"ephemeral_secrets_consumed_total", "Number of secrets successfully consumed.", snap.Consume},
		{"ephemeral_secrets_not_found_total", "Number of preview or consume requests for missing secrets.", snap.NotFound},
		{"ephemeral_rate_limited_total", "Number of requests rejected by the rate limiter.", snap.RateLimited},
	}
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", e.name, e.help, e.name, e.name, e.value); err != nil {
			return err
		}
	}
	return nil
}
