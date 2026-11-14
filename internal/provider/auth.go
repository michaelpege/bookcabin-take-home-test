package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
)

// Authenticator attaches simulated credentials to outbound provider requests.
type Authenticator interface {
	Apply(ctx context.Context, header http.Header) error
}

// BearerTokenAuth simulates OAuth2/JWT bearer authentication.
type BearerTokenAuth struct {
	Token string
}

// Apply sets the Authorization: Bearer header.
func (a BearerTokenAuth) Apply(ctx context.Context, header http.Header) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.Token == "" {
		return fmt.Errorf("missing bearer token")
	}
	header.Set("Authorization", "Bearer "+a.Token)
	return nil
}

// APIKeyAuth simulates API-key header authentication.
type APIKeyAuth struct {
	HeaderName string
	Key        string
}

// Apply sets the configured API key header.
func (a APIKeyAuth) Apply(ctx context.Context, header http.Header) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.HeaderName == "" || a.Key == "" {
		return fmt.Errorf("missing api key config")
	}
	header.Set(a.HeaderName, a.Key)
	return nil
}

// BasicAuth simulates HTTP Basic authentication.
type BasicAuth struct {
	Username string
	Password string
}

// Apply sets the Authorization: Basic header.
func (a BasicAuth) Apply(ctx context.Context, header http.Header) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.Username == "" {
		return fmt.Errorf("missing basic auth username")
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(a.Username + ":" + a.Password))
	header.Set("Authorization", "Basic "+encoded)
	return nil
}
