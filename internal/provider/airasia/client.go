package airasia

import (
	_ "embed"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
)

const (
	defaultProviderName = "AirAsia"
	defaultFailureRate  = 0.10
)

//go:embed testdata/search_response.json
var defaultPayload []byte

// Option configures an AirAsia Client instance.
type Option func(*Client)

// WithDelay overrides the simulated network latency range.
func WithDelay(minDelay, maxDelay time.Duration) Option {
	return func(c *Client) {
		c.minDelay = minDelay
		c.maxDelay = maxDelay
	}
}

// WithPayload overrides the mock JSON response payload (useful for testing).
func WithPayload(raw []byte) Option {
	return func(c *Client) {
		c.payload = raw
	}
}

// WithAuth overrides the mock authenticator.
func WithAuth(auth provider.Authenticator) Option {
	return func(c *Client) {
		c.auth = auth
	}
}

// WithFailureRate overrides the simulated transient error probability in [0.0, 1.0].
func WithFailureRate(rate float64) Option {
	return func(c *Client) {
		c.failureRate = rate
	}
}

// Client is the AirAsia FlightProvider adapter.
type Client struct {
	name        string
	auth        provider.Authenticator
	payload     []byte
	minDelay    time.Duration
	maxDelay    time.Duration
	failureRate float64
}

// NewClient constructs an AirAsia Client with PDF-specified 50-150ms latency,
// 90% availability (10% transient failure rate), and mock Bearer token authentication.
func NewClient(opts ...Option) *Client {
	c := &Client{
		name:        defaultProviderName,
		auth:        provider.BearerTokenAuth{Token: "mock-airasia-jwt-token"},
		payload:     defaultPayload,
		minDelay:    50 * time.Millisecond,
		maxDelay:    150 * time.Millisecond,
		failureRate: defaultFailureRate,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns the canonical provider name.
func (c *Client) Name() string {
	return c.name
}
