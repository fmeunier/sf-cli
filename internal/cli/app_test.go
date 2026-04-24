package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveTokenPrefersFlag(t *testing.T) {
	t.Parallel()

	got := resolveToken("flag-token", "env-token")
	if got != "flag-token" {
		t.Fatalf("resolveToken() = %q, want %q", got, "flag-token")
	}
}

func TestResolveTokenFallsBackToEnv(t *testing.T) {
	t.Parallel()

	got := resolveToken("", " env-token ")
	if got != "env-token" {
		t.Fatalf("resolveToken() = %q, want %q", got, "env-token")
	}
}

func TestRunUsesBearerTokenFromEnv(t *testing.T) {
	t.Setenv(envBearerToken, "env-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer env-token" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer env-token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tickets":[],"count":0,"page":0,"limit":25}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL, "tickets", "list", "--project", "test", "--tracker", "bugs"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}
}

func TestRunPrefersFlagTokenOverEnv(t *testing.T) {
	t.Setenv(envBearerToken, "env-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer flag-token" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer flag-token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tickets":[],"count":0,"page":0,"limit":25}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL, "--token", "flag-token", "tickets", "list", "--project", "test", "--tracker", "bugs"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}
}
