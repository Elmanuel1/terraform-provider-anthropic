package client

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
)

func TestDoRequest_NilConfig(t *testing.T) {
	_, _, err := DoRequest(context.Background(), nil, http.MethodGet, "/v1/agents", nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestDoRequest_NilWIF(t *testing.T) {
	cfg := &Config{APIKey: "key"}
	_, _, err := DoRequest(context.Background(), cfg, http.MethodGet, "/v1/agents", nil)
	if err == nil {
		t.Fatal("expected error for nil WIF")
	}
}

func TestDoRequest_PropagatesWorkspaceError(t *testing.T) {
	// WIF is set but APIKey is empty — workspace resolution will fail
	cfg := &Config{
		WIF:    &auth.WIFConfig{},
		APIKey: "",
	}
	_, _, err := DoRequest(context.Background(), cfg, http.MethodGet, "/v1/agents", nil)
	if err == nil {
		t.Fatal("expected error when workspace resolution fails")
	}
}

func TestHTTPClient_DefaultTimeout(t *testing.T) {
	cfg := &Config{}
	c := cfg.httpClient()
	if c.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", c.Timeout)
	}
}

func TestHTTPClient_Custom(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	cfg := &Config{HTTPClient: custom}
	if cfg.httpClient() != custom {
		t.Error("expected the injected client to be returned")
	}
}
