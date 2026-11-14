package domain

import "errors"

var (
	// ErrFlightInvalid indicates a normalized flight failed domain invariant checks.
	ErrFlightInvalid = errors.New("invalid flight data")
	// ErrSearchInvalidCriteria indicates malformed or incomplete search parameters.
	ErrSearchInvalidCriteria = errors.New("invalid search criteria")
)
