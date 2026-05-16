package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/build4africa/terraform-provider-anthropic-wif/internal/auth"
)

const (
	BaseURL    = "https://api.anthropic.com"
	BetaHeader = "managed-agents-2026-04-01"
	APIVersion = "2023-06-01"
)

type Config struct {
	WIF           *auth.WIFConfig
	APIKey        string
	WorkspaceName string
}

func DoRequest(ctx context.Context, cfg *Config, method, path string, body any) ([]byte, int, error) {
	workspaceID, err := auth.ResolveWorkspaceID(ctx, cfg.APIKey, cfg.WorkspaceName)
	if err != nil {
		return nil, 0, fmt.Errorf("workspace resolution: %w", err)
	}

	token, err := auth.MintToken(ctx, cfg.WIF, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("minting token: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, BaseURL+path, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", APIVersion)
	req.Header.Set("anthropic-beta", BetaHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, nil
}
