package auth

import (
	"context"
	"fmt"
	"net/http"
)

// Credentials authenticates an outbound HTTP request (e.g. sets x-api-key).
type Credentials interface {
	Authenticate(ctx context.Context, req *http.Request) error
}

// AdminAPIKey authenticates using a static Anthropic Admin API key.
type AdminAPIKey struct {
	Key string
}

func (a AdminAPIKey) Authenticate(_ context.Context, req *http.Request) error {
	if a.Key == "" {
		return fmt.Errorf("admin API key is empty")
	}
	req.Header.Set("x-api-key", a.Key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "admin-api-2025-05-21")
	return nil
}
