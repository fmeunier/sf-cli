package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRunShowsRootHelpText(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--help"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0", status)
	}
	if got := stdout.String(); got != rootUsage() {
		t.Fatalf("root help = %q, want %q", got, rootUsage())
	}
}

func TestRunShowsSubcommandHelpText(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"tickets", "search", "--help"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0", status)
	}
	if got := stdout.String(); got != ticketsSearchUsage() {
		t.Fatalf("tickets search help = %q, want %q", got, ticketsSearchUsage())
	}
}

func TestRunShowsHelpCommandOutput(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"help", "tracker", "schema"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0", status)
	}
	if got := stdout.String(); got != trackerSchemaUsage() {
		t.Fatalf("help output = %q, want %q", got, trackerSchemaUsage())
	}
}

func TestRunMissingCommandIncludesUsageGuidance(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run(nil, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	message := got["error"].(map[string]any)["message"].(string)
	if !bytes.Contains([]byte(message), []byte("Usage:")) {
		t.Fatalf("error.message = %q, want usage guidance", message)
	}
	if !bytes.Contains([]byte(message), []byte("sf [--base-url URL]")) {
		t.Fatalf("error.message = %q, want root usage", message)
	}
}

func TestRunMissingTicketsSubcommandIncludesUsageGuidance(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"tickets"}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	message := got["error"].(map[string]any)["message"].(string)
	if !bytes.Contains([]byte(message), []byte("sf tickets <subcommand>")) {
		t.Fatalf("error.message = %q, want tickets usage", message)
	}
	if got["command"] != "tickets" {
		t.Fatalf("command = %v, want %q", got["command"], "tickets")
	}
}
