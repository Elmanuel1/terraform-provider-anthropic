package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

type DeploymentAgentRef struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Version int64  `json:"version"`
}

type DeploymentContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type DeploymentInitialEvent struct {
	Type    string                   `json:"type"`
	Content []DeploymentContentBlock `json:"content"`
}

type DeploymentSchedule struct {
	Type           string   `json:"type"`
	Expression     string   `json:"expression"`
	Timezone       string   `json:"timezone"`
	LastRunAt      *string  `json:"last_run_at"`
	UpcomingRunsAt []string `json:"upcoming_runs_at"`
}

type DeploymentPausedReason struct {
	Type  string `json:"type"`
	Error *struct {
		Type string `json:"type"`
	} `json:"error"`
}

type DeploymentResponse struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	Description   *string                  `json:"description"`
	Agent         DeploymentAgentRef       `json:"agent"`
	EnvironmentID string                   `json:"environment_id"`
	InitialEvents []DeploymentInitialEvent `json:"initial_events"`
	Metadata      map[string]string        `json:"metadata"`
	Schedule      *DeploymentSchedule      `json:"schedule"`
	Status        string                   `json:"status"`
	PausedReason  *DeploymentPausedReason  `json:"paused_reason"`
	VaultIDs      []string                 `json:"vault_ids"`
	CreatedAt     string                   `json:"created_at"`
	UpdatedAt     string                   `json:"updated_at"`
	ArchivedAt    *string                  `json:"archived_at"`
}

type DeploymentClient struct {
	creds      auth.Credentials
	httpClient *http.Client
}

func NewDeploymentClient(creds auth.Credentials) *DeploymentClient {
	return &DeploymentClient{creds: creds, httpClient: defaultHTTPClient}
}

func (c *DeploymentClient) Create(ctx context.Context, body map[string]any) (*DeploymentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/deployments", body)
	if err != nil {
		return nil, fmt.Errorf("creating deployment: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("creating deployment returned HTTP %d: %s", status, raw)
	}
	return parseDeployment(raw)
}

func (c *DeploymentClient) Read(ctx context.Context, id string) (*DeploymentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodGet, "/v1/deployments/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("reading deployment: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("reading deployment returned HTTP %d: %s", status, raw)
	}
	return parseDeployment(raw)
}

func (c *DeploymentClient) Update(ctx context.Context, id string, body map[string]any) (*DeploymentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/deployments/"+url.PathEscape(id), body)
	if err != nil {
		return nil, fmt.Errorf("updating deployment: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("updating deployment returned HTTP %d: %s", status, raw)
	}
	return parseDeployment(raw)
}

func (c *DeploymentClient) Pause(ctx context.Context, id string) (*DeploymentResponse, error) {
	return c.action(ctx, id, "pause")
}

func (c *DeploymentClient) Unpause(ctx context.Context, id string) (*DeploymentResponse, error) {
	return c.action(ctx, id, "unpause")
}

func (c *DeploymentClient) action(ctx context.Context, id, verb string) (*DeploymentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/deployments/"+url.PathEscape(id)+"/"+verb, nil)
	if err != nil {
		return nil, fmt.Errorf("%s deployment: %w", verb, err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%s deployment returned HTTP %d: %s", verb, status, raw)
	}
	return parseDeployment(raw)
}

func (c *DeploymentClient) Archive(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/deployments/"+url.PathEscape(id)+"/archive", nil)
	if err != nil {
		return fmt.Errorf("archiving deployment: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("archiving deployment returned HTTP %d", status)
	}
	return nil
}

func parseDeployment(raw []byte) (*DeploymentResponse, error) {
	var d DeploymentResponse
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("parsing deployment response: %w", err)
	}
	return &d, nil
}
