package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveWorkspaceID_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "key-123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "wrkspc_abc", "name": "tosspaper"},
				{"id": "wrkspc_def", "name": "other"},
			},
		})
	}))
	defer srv.Close()

	orig := anthropicWorkspacesURL
	anthropicWorkspacesURL = srv.URL
	defer func() { anthropicWorkspacesURL = orig }()

	id, err := ResolveWorkspaceID(context.Background(), "key-123", "tosspaper")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "wrkspc_abc" {
		t.Errorf("expected wrkspc_abc, got %s", id)
	}
}

func TestResolveWorkspaceID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "wrkspc_abc", "name": "other"},
			},
		})
	}))
	defer srv.Close()

	orig := anthropicWorkspacesURL
	anthropicWorkspacesURL = srv.URL
	defer func() { anthropicWorkspacesURL = orig }()

	_, err := ResolveWorkspaceID(context.Background(), "key-123", "tosspaper")
	if err == nil {
		t.Fatal("expected error when workspace not found")
	}
}

func TestResolveWorkspaceID_DefaultWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "wrkspc_default", "name": ""},
			},
		})
	}))
	defer srv.Close()

	orig := anthropicWorkspacesURL
	anthropicWorkspacesURL = srv.URL
	defer func() { anthropicWorkspacesURL = orig }()

	id, err := ResolveWorkspaceID(context.Background(), "key-123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "wrkspc_default" {
		t.Errorf("expected wrkspc_default, got %s", id)
	}
}

func TestResolveWorkspaceID_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	orig := anthropicWorkspacesURL
	anthropicWorkspacesURL = srv.URL
	defer func() { anthropicWorkspacesURL = orig }()

	_, err := ResolveWorkspaceID(context.Background(), "bad-key", "tosspaper")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}
