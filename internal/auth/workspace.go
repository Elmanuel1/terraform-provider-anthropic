package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var anthropicWorkspacesURL = "https://api.anthropic.com/v1/organizations/workspaces"

func ResolveWorkspaceID(ctx context.Context, creds Credentials, name string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, anthropicWorkspacesURL, nil)
	if err != nil {
		return "", fmt.Errorf("building workspaces request: %w", err)
	}
	if err := creds.Authenticate(ctx, req); err != nil {
		return "", fmt.Errorf("authenticating workspaces request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("listing workspaces: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("listing workspaces returned HTTP %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parsing workspaces response: %w", err)
	}

	var available []string
	var defaultID string
	for _, w := range result.Data {
		if w.Name == name {
			return w.ID, nil
		}
		if w.Name == "" {
			defaultID = w.ID
		}
		available = append(available, fmt.Sprintf("%q", w.Name))
	}

	if name == "" && defaultID != "" {
		return defaultID, nil
	}

	return "", fmt.Errorf("workspace %q not found — available workspaces: [%s]", name, strings.Join(available, ", "))
}
