package batik

import (
	_ "embed"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
)

const defaultProviderName = "Batik Air"

//go:embed testdata/search_response.json
var defaultPayload []byte

// Option configures a Batik Air Client instance.
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

// Client is the Batik Air FlightProvider adapter.
type Client struct {
	name     string
	auth     provider.Authenticator
	payload  []byte
	minDelay time.Duration
	maxDelay time.Duration
}

// NewClient constructs a Batik Air Client with PDF-specified 200-400ms latency
// and mock HTTP Basic authentication.
func NewClient(opts ...Option) *Client {
	c := &Client{
		name:     defaultProviderName,
		auth:     provider.BasicAuth{Username: "bookcabin", Password: "mock-batik-secret"},
		payload:  defaultPayload,
		minDelay: 200 * time.Millisecond,
		maxDelay: 400 * time.Millisecond,
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
