package provider

import (
	"context"
	"math/rand/v2"
	"time"
)

// SimulateDelay waits for a uniform random duration in [minDelay, maxDelay]
// while respecting context cancellation.
func SimulateDelay(ctx context.Context, minDelay, maxDelay time.Duration) error {
	if maxDelay <= 0 {
		return ctx.Err()
	}
	delay := minDelay
	if spread := maxDelay - minDelay; spread > 0 {
		delay += time.Duration(rand.Int64N(int64(spread) + 1))
	}
	if delay <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
