package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

const defaultOperationTimeout = 200 * time.Millisecond

// Option configures a FlightCache instance.
type Option func(*FlightCache)

// WithOperationTimeout overrides the per-command Redis timeout used to fail open
// promptly when Redis is slow or unreachable.
func WithOperationTimeout(d time.Duration) Option {
	return func(c *FlightCache) {
		if d > 0 {
			c.opTimeout = d
		}
	}
}

// FlightCache implements domain.FlightCache backed by Redis.
type FlightCache struct {
	client    *goredis.Client
	opTimeout time.Duration
}

// NewFlightCache connects to Redis at addr with tight fail-open timeouts.
func NewFlightCache(addr string, opts ...Option) *FlightCache {
	client := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  defaultOperationTimeout,
		WriteTimeout: defaultOperationTimeout,
	})
	return NewFlightCacheFromClient(client, opts...)
}

// NewFlightCacheFromClient wraps an existing go-redis Client.
func NewFlightCacheFromClient(client *goredis.Client, opts ...Option) *FlightCache {
	c := &FlightCache{
		client:    client,
		opTimeout: defaultOperationTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Get fetches cached bytes for key, returning domain.ErrCacheMiss when absent.
func (c *FlightCache) Get(ctx context.Context, key string) ([]byte, error) {
	opCtx, cancel := c.withOpTimeout(ctx)
	defer cancel()

	val, err := c.client.Get(opCtx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, domain.ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get %q: %w", key, err)
	}
	return val, nil
}

// Set stores value under key for ttl duration. Non-positive TTLs are ignored.
func (c *FlightCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	opCtx, cancel := c.withOpTimeout(ctx)
	defer cancel()

	if err := c.client.Set(opCtx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set %q: %w", key, err)
	}
	return nil
}

// Ping verifies Redis connectivity within the operation timeout.
func (c *FlightCache) Ping(ctx context.Context) error {
	opCtx, cancel := c.withOpTimeout(ctx)
	defer cancel()

	if err := c.client.Ping(opCtx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// Close releases underlying Redis client connections.
func (c *FlightCache) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *FlightCache) withOpTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.opTimeout <= 0 {
		return ctx, func() {}
	}
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= c.opTimeout {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.opTimeout)
}
