package client

import (
	"net/http"
	"testing"
	"time"
)

func TestResolveHTTPClient_Default(t *testing.T) {
	c := resolveHTTPClient(nil)
	if c.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", c.Timeout)
	}
}

func TestResolveHTTPClient_Custom(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	if resolveHTTPClient(custom) != custom {
		t.Error("expected the injected client to be returned")
	}
}
