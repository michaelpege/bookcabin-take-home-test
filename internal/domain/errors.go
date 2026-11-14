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
	// ErrProviderUnauthorized indicates rejected or missing credentials for a provider request.
	ErrProviderUnauthorized = errors.New("provider unauthorized")
	// ErrProviderInvalidResponse indicates an upstream payload could not be decoded cleanly.
	ErrProviderInvalidResponse = errors.New("provider invalid response")
	// ErrProviderContract indicates the provider returned a non-success contract envelope.
	ErrProviderContract = errors.New("provider contract error")
	// ErrAllProvidersFailed indicates every queried airline provider failed for the search.
	ErrAllProvidersFailed = errors.New("all flight providers failed")
	// ErrCacheMiss indicates the requested key is not present in the cache.
	ErrCacheMiss = errors.New("cache miss")
)
