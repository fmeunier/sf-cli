package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sf-cli/internal/model"
)

func decodeEnvelope(t *testing.T, data []byte) model.Envelope {
	t.Helper()

	var got model.Envelope
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	return got
}

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

func TestSuccessEnvelopeKeepsResultProposalAndWarningsTogether(t *testing.T) {
	t.Parallel()

	got := successEnvelope(
		"tickets.list",
		proposal("tickets.list", "list_tickets", map[string]any{"project": "test"}, map[string]any{"limit": 10}),
		map[string]any{"count": 1},
		"partial metadata from upstream",
	)

	if !got.OK {
		t.Fatalf("OK = %v, want true", got.OK)
	}
	if got.Error != nil {
		t.Fatalf("Error = %#v, want nil", got.Error)
	}
	if got.Proposal == nil || got.Proposal.Action != "list_tickets" {
		t.Fatalf("Proposal = %#v, want action %q", got.Proposal, "list_tickets")
	}
	if got.Result.(map[string]any)["count"] != 1 {
		t.Fatalf("Result.count = %v, want 1", got.Result.(map[string]any)["count"])
	}
	if len(got.Warnings) != 1 || got.Warnings[0] != "partial metadata from upstream" {
		t.Fatalf("Warnings = %v, want single warning", got.Warnings)
	}
}

func TestErrorEnvelopeKeepsProposalAndClearsResult(t *testing.T) {
	t.Parallel()

	got := errorEnvelope(
		"tickets.get",
		proposal("tickets.get", "get_ticket", map[string]any{"project": "test", "ticket": 42}, nil),
		"not_found",
		"ticket not found",
	)

	if got.OK {
		t.Fatalf("OK = %v, want false", got.OK)
	}
	if got.Result != nil {
		t.Fatalf("Result = %#v, want nil", got.Result)
	}
	if got.Error == nil || got.Error.Code != "not_found" {
		t.Fatalf("Error = %#v, want code %q", got.Error, "not_found")
	}
	if got.Proposal == nil || got.Proposal.Action != "get_ticket" {
		t.Fatalf("Proposal = %#v, want action %q", got.Proposal, "get_ticket")
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("Warnings = %v, want empty", got.Warnings)
	}
}
