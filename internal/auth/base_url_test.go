package auth

import (
	"context"
	"net/http"
	"testing"
)

func TestWithBaseURLRoutesEveryRequestToTheBase(t *testing.T) {
	base, err := ParseBaseURL("http://agentd.internal:8080/prefix/")
	if err != nil {
		t.Fatal(err)
	}
	creds := WithBeta(WithBaseURL(AdminAPIKey{Key: "k"}, base), AgentsBeta)
	req, _ := http.NewRequest(http.MethodGet, BaseURL+"/v1/memory_stores/memstore_1?limit=2", nil)
	if err := creds.Authenticate(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got, want := req.URL.String(), "http://agentd.internal:8080/prefix/v1/memory_stores/memstore_1?limit=2"; got != want {
		t.Fatalf("url = %s, want %s", got, want)
	}
	if req.Host != "agentd.internal:8080" || req.Header.Get(HeaderAPIKey) != "k" || req.Header.Get(HeaderBeta) != AgentsBeta {
		t.Fatalf("host %q, headers %v", req.Host, req.Header)
	}
}

func TestWithBaseURLNilLeavesTheRequest(t *testing.T) {
	creds := WithBaseURL(AdminAPIKey{Key: "k"}, nil)
	req, _ := http.NewRequest(http.MethodGet, BaseURL+"/v1/agents", nil)
	if err := creds.Authenticate(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if req.URL.String() != BaseURL+"/v1/agents" {
		t.Fatalf("url = %s", req.URL)
	}
}

func TestParseBaseURLRejectsWhatIsNotABase(t *testing.T) {
	for _, raw := range []string{"", "agentd:8080", "ftp://x", "http://x?y=1", "/v1"} {
		if _, err := ParseBaseURL(raw); err == nil {
			t.Errorf("%q: want an error", raw)
		}
	}
}
