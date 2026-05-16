package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
)

const (
	BaseURL    = "https://api.anthropic.com"
	BetaHeader = "managed-agents-2026-04-01"
	APIVersion = "2023-06-01"
)

type Config struct {
	WIF        *auth.WIFConfig
	APIKey     string // admin API key, used by workspace resource and token datasource
	HTTPClient *http.Client
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// DoRequest calls the Managed Agents API using a WIF-minted bearer token.
// workspaceID must be resolved by the caller (stored in resource state).
func DoRequest(ctx context.Context, cfg *Config, workspaceID, method, path string, body any) ([]byte, int, error) {
	if cfg == nil {
		return nil, 0, fmt.Errorf("missing client config")
	}
	if cfg.WIF == nil {
		return nil, 0, fmt.Errorf("missing WIF config")
	}

	token, err := auth.MintToken(ctx, cfg.WIF, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("minting token: %w", err)
	}

	raw, status, err := doHTTP(ctx, cfg.httpClient(), method, BaseURL+path, body,
		func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+token.AccessToken)
			req.Header.Set("anthropic-version", APIVersion)
			req.Header.Set("anthropic-beta", BetaHeader)
		})
	return raw, status, err
}

// DoAdminRequest calls the Anthropic Admin API using the provided credentials.
func DoAdminRequest(ctx context.Context, cfg *Config, creds auth.Credentials, method, path string, body any) ([]byte, int, error) {
	if cfg == nil {
		return nil, 0, fmt.Errorf("missing client config")
	}

	req, err := buildRequest(ctx, method, BaseURL+path, body)
	if err != nil {
		return nil, 0, err
	}
	if err := creds.Authenticate(ctx, req); err != nil {
		return nil, 0, fmt.Errorf("authenticating request: %w", err)
	}

	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("admin API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, nil
}

func doHTTP(ctx context.Context, hc *http.Client, method, url string, body any, setHeaders func(*http.Request)) ([]byte, int, error) {
	req, err := buildRequest(ctx, method, url, body)
	if err != nil {
		return nil, 0, err
	}
	setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, nil
}

func buildRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
