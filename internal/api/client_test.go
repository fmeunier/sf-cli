package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRequestSetsBearerHeader(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Options{BaseURL: "https://example.com/rest", Token: "secret-token"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "p/test/tickets", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "Bearer secret-token" {
		t.Fatalf("Authorization header = %q, want %q", got, "Bearer secret-token")
	}
	if !client.HasToken() {
		t.Fatal("HasToken() = false, want true")
	}
}

func TestHasTokenIsFalseWhenUnset(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Options{BaseURL: "https://example.com/rest"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.HasToken() {
		t.Fatal("HasToken() = true, want false")
	}
}

func TestGetJSONReturnsAPIErrorMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"missing token"}`))
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.GetJSON(context.Background(), "p/test/tickets", nil, nil)
	if err == nil {
		t.Fatal("GetJSON() error = nil, want error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("GetJSON() error type = %T, want *APIError", err)
	}

	if apiErr.Message != "missing token" {
		t.Fatalf("APIError.Message = %q, want %q", apiErr.Message, "missing token")
	}
}
