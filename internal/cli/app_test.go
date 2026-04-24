package cli

import "testing"

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
