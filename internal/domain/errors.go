package domain

import "errors"

// Flight & Search Validation Errors.
var (
	// ErrFlightInvalid indicates a normalized flight failed domain invariant checks.
	ErrFlightInvalid = errors.New("invalid flight data")
	// ErrSearchInvalidCriteria indicates malformed or incomplete search parameters.
	ErrSearchInvalidCriteria = errors.New("invalid search criteria")
)

// Provider & Cache Boundary Errors.
var (
	// ErrProviderUnavailable indicates an upstream airline provider failed or timed out.
	ErrProviderUnavailable = errors.New("provider unavailable")
	// ErrCacheMiss indicates the requested key is not present in the cache.
	ErrCacheMiss = errors.New("cache miss")
)
