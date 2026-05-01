package cli

import (
	"bytes"
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

func TestRootHelpIncludesAgentGuidance(t *testing.T) {
	t.Parallel()

	help := rootUsage()
	for _, want := range []string{
		"Purpose:",
		"Output contract:",
		"Agent guidance:",
		"Review recently active tickets:",
		"sf tickets activity --project fuse-emulator --tracker bugs",
		"Use 'actions validate' before proposing or applying ticket create, label, or",
		"Current write-intent support:",
		"Supported action types today are 'ticket_create', 'ticket_labels', and",
	} {
		if !bytes.Contains([]byte(help), []byte(want)) {
			t.Fatalf("root help missing %q", want)
		}
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

func TestActionsValidateHelpIncludesInputAndOutputContract(t *testing.T) {
	t.Parallel()

	help := actionsValidateUsage()
	for _, want := range []string{
		"Supported action types today:",
		"Expected input shape:",
		"ticket_create",
		"Current ticket_create scope:",
		"ticket_labels",
		"Current ticket_labels scope:",
		"Current ticket_comment scope:",
		"reply posts are not modeled",
		"Validation output:",
		"Per-action result fields:",
		"canonical_identifiers",
	} {
		if !bytes.Contains([]byte(help), []byte(want)) {
			t.Fatalf("actions validate help missing %q", want)
		}
	}
}

func TestRunMissingCommandIncludesUsageGuidance(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run(nil, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	got := decodeEnvelope(t, stdout.Bytes())
	message := got.Error.Message
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

	got := decodeEnvelope(t, stdout.Bytes())
	message := got.Error.Message
	if !bytes.Contains([]byte(message), []byte("sf tickets <subcommand>")) {
		t.Fatalf("error.message = %q, want tickets usage", message)
	}
	if got.Command != "tickets" {
		t.Fatalf("command = %q, want %q", got.Command, "tickets")
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}
}
