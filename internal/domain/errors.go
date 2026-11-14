package domain

import "errors"

var (
	// ErrInvalidFlight indicates a normalized flight failed domain invariant checks.
	ErrInvalidFlight = errors.New("invalid flight data")
	// ErrInvalidSearchCriteria indicates malformed or incomplete search parameters.
	ErrInvalidSearchCriteria = errors.New("invalid search criteria")
	// ErrProviderUnavailable indicates an upstream airline provider failed or timed out.
	ErrProviderUnavailable = errors.New("provider unavailable")
	// ErrCacheMiss indicates the requested key is not present in the cache.
	ErrCacheMiss = errors.New("cache miss")
)
