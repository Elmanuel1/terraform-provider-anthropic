package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

func withBaseURL(t *testing.T, srv *httptest.Server) {
	t.Helper()
	orig := auth.BaseURL
	auth.BaseURL = srv.URL
	t.Cleanup(func() { auth.BaseURL = orig })
}

func TestDeploymentClient_Create(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
		json.NewEncoder(w).Encode(map[string]any{
			"id":             "depl_abc",
			"name":           "weekly",
			"environment_id": "env_1",
			"agent":          map[string]any{"id": "agent_1", "type": "agent", "version": 3},
			"status":         "active",
			"created_at":     "2026-06-12T00:00:00Z",
			"updated_at":     "2026-06-12T00:00:00Z",
		})
	}))
	defer srv.Close()
	withBaseURL(t, srv)

	c := NewDeploymentClient(auth.WorkspaceAPIKey{Key: "k"})
	body := map[string]any{
		"name":           "weekly",
		"environment_id": "env_1",
		"agent":          map[string]any{"id": "agent_1", "type": "agent"},
		"initial_events": []map[string]any{{"type": "user.message", "content": []map[string]any{{"type": "text", "text": "hi"}}}},
		"schedule":       map[string]any{"type": "cron", "expression": "0 9 * * 1-5", "timezone": "UTC"},
	}
	dep, err := c.Create(context.Background(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/deployments" {
		t.Errorf("expected POST /v1/deployments, got %s %s", gotMethod, gotPath)
	}
	if gotBody["environment_id"] != "env_1" {
		t.Errorf("environment_id not forwarded: %v", gotBody["environment_id"])
	}
	if _, ok := gotBody["schedule"]; !ok {
		t.Error("schedule not forwarded")
	}
	if dep.ID != "depl_abc" || dep.Agent.Version != 3 {
		t.Errorf("unexpected parsed response: %+v", dep)
	}
}

func TestDeploymentClient_Read_FullObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/deployments/depl_abc" {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		w.Header().Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
		w.Write([]byte(`{
			"id":"depl_abc","name":"weekly","environment_id":"env_1",
			"agent":{"id":"agent_1","type":"agent","version":2},
			"initial_events":[{"type":"user.message","content":[{"type":"text","text":"hi"}]}],
			"schedule":{"type":"cron","expression":"0 9 * * 1-5","timezone":"UTC","upcoming_runs_at":["2026-06-13T09:00:00Z"]},
			"status":"paused","paused_reason":{"type":"manual"},
			"vault_ids":["vault_1"],"created_at":"t","updated_at":"t"
		}`))
	}))
	defer srv.Close()
	withBaseURL(t, srv)

	dep, err := NewDeploymentClient(auth.WorkspaceAPIKey{Key: "k"}).Read(context.Background(), "depl_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep == nil {
		t.Fatal("expected deployment, got nil")
	}
	if dep.Status != "paused" || dep.PausedReason == nil || dep.PausedReason.Type != "manual" {
		t.Errorf("paused_reason not parsed: %+v", dep)
	}
	if dep.Schedule == nil || len(dep.Schedule.UpcomingRunsAt) != 1 {
		t.Errorf("schedule not parsed: %+v", dep.Schedule)
	}
	if len(dep.InitialEvents) != 1 || dep.InitialEvents[0].Content[0].Text != "hi" {
		t.Errorf("initial_events not parsed: %+v", dep.InitialEvents)
	}
}

func TestDeploymentClient_Read_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()
	withBaseURL(t, srv)

	dep, err := NewDeploymentClient(auth.WorkspaceAPIKey{Key: "k"}).Read(context.Background(), "depl_missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep != nil {
		t.Errorf("expected nil for 404, got %+v", dep)
	}
}

func TestDeploymentClient_PauseAndArchive(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
		w.Write([]byte(`{"id":"depl_abc","status":"paused"}`))
	}))
	defer srv.Close()
	withBaseURL(t, srv)

	c := NewDeploymentClient(auth.WorkspaceAPIKey{Key: "k"})
	if _, err := c.Pause(context.Background(), "depl_abc"); err != nil {
		t.Fatalf("pause error: %v", err)
	}
	if err := c.Archive(context.Background(), "depl_abc"); err != nil {
		t.Fatalf("archive error: %v", err)
	}
	want := []string{"POST /v1/deployments/depl_abc/pause", "POST /v1/deployments/depl_abc/archive"}
	for i, p := range want {
		if i >= len(paths) || paths[i] != p {
			t.Errorf("expected call %q, got %v", p, paths)
		}
	}
}
