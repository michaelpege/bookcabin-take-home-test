package lion

import (
	_ "embed"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/provider"
)

const defaultProviderName = "Lion Air"

//go:embed testdata/search_response.json
var defaultPayload []byte

// Option configures a Lion Air Client instance.
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

// Client is the Lion Air FlightProvider adapter.
type Client struct {
	name     string
	auth     provider.Authenticator
	payload  []byte
	minDelay time.Duration
	maxDelay time.Duration
}

// NewClient constructs a Lion Air Client with PDF-specified 100-200ms latency
// and mock API-key authentication.
func NewClient(opts ...Option) *Client {
	c := &Client{
		name:     defaultProviderName,
		auth:     provider.APIKeyAuth{HeaderName: "X-API-Key", Key: "mock-lion-api-key"},
		payload:  defaultPayload,
		minDelay: 100 * time.Millisecond,
		maxDelay: 200 * time.Millisecond,
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
