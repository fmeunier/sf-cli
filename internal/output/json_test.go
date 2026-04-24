package output

import (
	"bytes"
	"strings"
	"testing"

	"sf-cli/internal/model"
)

func TestWriteJSONFormatsIndentedEnvelope(t *testing.T) {
	t.Parallel()

	envelope := model.Envelope{
		Version:  "v1",
		Mode:     "read_only",
		Command:  "tickets.list",
		OK:       true,
		Warnings: []string{},
		Proposal: &model.Proposal{
			Action:  "list_tickets",
			Effects: []model.Effect{},
		},
		Result: map[string]any{"count": 1},
	}

	var out bytes.Buffer
	if err := WriteJSON(&out, envelope); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	want := "{\n" +
		"  \"version\": \"v1\",\n" +
		"  \"mode\": \"read_only\",\n" +
		"  \"command\": \"tickets.list\",\n" +
		"  \"ok\": true,\n" +
		"  \"warnings\": [],\n" +
		"  \"proposal\": {\n" +
		"    \"action\": \"list_tickets\",\n" +
		"    \"effects\": []\n" +
		"  },\n" +
		"  \"result\": {\n" +
		"    \"count\": 1\n" +
		"  },\n" +
		"  \"error\": null\n" +
		"}\n"

	if out.String() != want {
		t.Fatalf("WriteJSON() output = %q, want %q", out.String(), want)
	}
}

func TestWriteJSONOmitsNilProposalAndPreservesWarnings(t *testing.T) {
	t.Parallel()

	envelope := model.Envelope{
		Version:  "v1",
		Mode:     "read_only",
		Command:  "actions.validate",
		OK:       true,
		Warnings: []string{"comment body exceeds the recommended length"},
		Result:   map[string]any{"ok": true},
		Error:    nil,
	}

	var out bytes.Buffer
	if err := WriteJSON(&out, envelope); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	got := out.String()
	if strings.Contains(got, "\"proposal\"") {
		t.Fatalf("WriteJSON() output = %q, want proposal omitted", got)
	}
	if !strings.Contains(got, "\"warnings\": [\n    \"comment body exceeds the recommended length\"\n  ]") {
		t.Fatalf("WriteJSON() output = %q, want warnings preserved", got)
	}
	if !strings.Contains(got, "\"result\": {\n    \"ok\": true\n  }") {
		t.Fatalf("WriteJSON() output = %q, want result preserved", got)
	}
}
