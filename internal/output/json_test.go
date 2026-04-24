package output

import (
	"bytes"
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
