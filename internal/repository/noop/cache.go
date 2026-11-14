package noop

import (
	"context"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

// Cache is a no-op domain.FlightCache implementation for unit tests and
// cache-disabled execution paths.
type Cache struct{}

// New constructs a no-op FlightCache adapter.
func New() *Cache {
	return &Cache{}
}

// Get always returns domain.ErrCacheMiss.
func (c *Cache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, domain.ErrCacheMiss
}

// Set discards the entry without error.
func (c *Cache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
