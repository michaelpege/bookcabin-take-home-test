package garuda

import (
	_ "embed"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
)

const defaultProviderName = "Garuda Indonesia"

//go:embed testdata/search_response.json
var defaultPayload []byte

// Option configures a Garuda Client instance.
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

// Client is the Garuda Indonesia FlightProvider adapter.
type Client struct {
	name     string
	auth     provider.Authenticator
	payload  []byte
	minDelay time.Duration
	maxDelay time.Duration
}

// NewClient constructs a Garuda Client with PDF-specified 50-100ms latency
// and mock Bearer JWT authentication.
func NewClient(opts ...Option) *Client {
	c := &Client{
		name:     defaultProviderName,
		auth:     provider.BearerTokenAuth{Token: "mock-garuda-jwt-token"},
		payload:  defaultPayload,
		minDelay: 50 * time.Millisecond,
		maxDelay: 100 * time.Millisecond,
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
