package domain

import "errors"

var (
	// ErrInvalidFlight indicates a normalized flight failed domain invariant checks.
	ErrInvalidFlight = errors.New("invalid flight data")
	// ErrInvalidSearchCriteria indicates malformed or incomplete search parameters.
	ErrInvalidSearchCriteria = errors.New("invalid search criteria")
)
