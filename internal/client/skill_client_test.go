package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

func TestSkillDeleteLeavesTheOnlyVersionToTheSkillDelete(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		mu.Unlock()
		switch {
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"version":"2"},{"version":"1"}]}`))
		case strings.HasSuffix(r.URL.Path, "/versions/1"):
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"cannot delete a Skill's only version. Delete the Skill, or create another version first"}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	orig := auth.BaseURL
	auth.BaseURL = srv.URL
	defer func() { auth.BaseURL = orig }()

	if err := NewSkillClient(auth.AdminAPIKey{Key: "k"}).Delete(context.Background(), "skill_1"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /v1/skills/skill_1/versions",
		"DELETE /v1/skills/skill_1/versions/2",
		"DELETE /v1/skills/skill_1/versions/1",
		"DELETE /v1/skills/skill_1",
	}
	if strings.Join(calls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("calls:\n%s", strings.Join(calls, "\n"))
	}
}
