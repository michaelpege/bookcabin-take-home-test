package provider

import "time"

// ResiliencePolicy defines per-provider client resilience settings for
// rate limiting, per-attempt timeouts, and exponential backoff retries.
type ResiliencePolicy struct {
	Timeout      time.Duration
	MaxRetries   int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
	RateLimitRPS float64
	RateBurst    int
}

// DefaultPolicies returns per-airline resilience policies tuned to each provider's
// expected latency and reliability profile.
func DefaultPolicies() map[string]ResiliencePolicy {
	return map[string]ResiliencePolicy{
		"Garuda Indonesia": {
			Timeout:      500 * time.Millisecond,
			MaxRetries:   1,
			BaseBackoff:  25 * time.Millisecond,
			MaxBackoff:   150 * time.Millisecond,
			RateLimitRPS: 50,
			RateBurst:    10,
		},
		"Lion Air": {
			Timeout:      600 * time.Millisecond,
			MaxRetries:   1,
			BaseBackoff:  30 * time.Millisecond,
			MaxBackoff:   200 * time.Millisecond,
			RateLimitRPS: 40,
			RateBurst:    10,
		},
		"Batik Air": {
			Timeout:      800 * time.Millisecond,
			MaxRetries:   1,
			BaseBackoff:  40 * time.Millisecond,
			MaxBackoff:   250 * time.Millisecond,
			RateLimitRPS: 30,
			RateBurst:    8,
		},
		"AirAsia": {
			Timeout:      500 * time.Millisecond,
			MaxRetries:   2,
			BaseBackoff:  30 * time.Millisecond,
			MaxBackoff:   200 * time.Millisecond,
			RateLimitRPS: 50,
			RateBurst:    10,
		},
	}
}
