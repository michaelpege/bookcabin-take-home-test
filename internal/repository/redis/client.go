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

// Option configures a Redis cache Client instance.
type Option func(*Client)

// WithOperationTimeout overrides the per-command Redis timeout used to fail open
// promptly when Redis is slow or unreachable.
func WithOperationTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.opTimeout = d
		}
	}
}

// Client implements domain.FlightCache backed by Redis.
type Client struct {
	client    *goredis.Client
	opTimeout time.Duration
}

// NewClient connects to Redis at addr with tight fail-open timeouts and verifies
// initial connectivity via Ping. It always returns a usable fail-open Client even
// when Ping returns an error.
func NewClient(addr string, opts ...Option) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  defaultOperationTimeout,
		WriteTimeout: defaultOperationTimeout,
	})
	c := NewClientFromGoRedis(rdb, opts...)
	return c, c.Ping(context.Background())
}

// NewClientFromGoRedis wraps an existing go-redis Client.
func NewClientFromGoRedis(rdb *goredis.Client, opts ...Option) *Client {
	c := &Client{
		client:    rdb,
		opTimeout: defaultOperationTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Get fetches cached bytes for key, returning domain.ErrCacheMiss when absent.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
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
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
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
func (c *Client) Ping(ctx context.Context) error {
	opCtx, cancel := c.withOpTimeout(ctx)
	defer cancel()

	if err := c.client.Ping(opCtx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// Close releases underlying Redis client connections.
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *Client) withOpTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.opTimeout <= 0 {
		return ctx, func() {}
	}
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= c.opTimeout {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.opTimeout)
}
