package fpl

import (
	"context"
	"sync"
	"time"
)

// tokenBucket limits outbound FPL API calls to maxTokens/sec with a burst of maxTokens.
type tokenBucket struct {
	mu     sync.Mutex
	tokens float64
	max    float64
	perSec float64
	last   time.Time
}

func newTokenBucket(rps int) *tokenBucket {
	if rps < 1 {
		rps = 5
	}
	return &tokenBucket{
		tokens: float64(rps),
		max:    float64(rps),
		perSec: float64(rps),
		last:   time.Now(),
	}
}

func (b *tokenBucket) Wait(ctx context.Context) error {
	for {
		b.mu.Lock()
		now := time.Now()
		b.tokens += now.Sub(b.last).Seconds() * b.perSec
		if b.tokens > b.max {
			b.tokens = b.max
		}
		b.last = now
		if b.tokens >= 1 {
			b.tokens--
			b.mu.Unlock()
			return nil
		}
		wait := time.Duration((1-b.tokens)/b.perSec*float64(time.Second)) + time.Millisecond
		b.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
