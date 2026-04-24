package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrackerSchemaReturnsBestEffortMetadata(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test/bugs":
			_, _ = w.Write([]byte(`{"milestones":[{"name":"v1","description":"first","due_date":"","default":"on","complete":false,"closed":1,"total":2}],"tracker_config":{"options":{"mount_label":"Bugs","mount_point":"bugs","EnableVoting":true}},"saved_bins":[{"_id":"bin-1","summary":"Open","terms":"status:open","sort":"ticket_num_i desc"}]}`))
		case "/rest/p/test/bugs/search":
			if got := r.URL.Query().Get("q"); got != "*:*" {
				t.Fatalf("q = %q, want %q", got, "*:*")
			}
			if got := r.URL.Query().Get("limit"); got != "1" {
				t.Fatalf("limit = %q, want %q", got, "1")
			}
			_, _ = w.Write([]byte(`{"filter_choices":{"_milestone":[["v1",2]],"status":[["open",4],["closed",1]],"assigned_to":["alice","bob"]}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tracker", "schema", "--project", "test", "--tracker", "bugs"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got["command"] != "tracker.schema" {
		t.Fatalf("command = %v, want %q", got["command"], "tracker.schema")
	}
	if got["ok"] != true {
		t.Fatalf("ok = %v, want true", got["ok"])
	}

	proposal := got["proposal"].(map[string]any)
	if proposal["action"] != "get_tracker_schema" {
		t.Fatalf("proposal.action = %v, want %q", proposal["action"], "get_tracker_schema")
	}
	result := got["result"].(map[string]any)
	options := result["options"].(map[string]any)
	if options["mount_label"] != "Bugs" {
		t.Fatalf("result.options.mount_label = %v, want %q", options["mount_label"], "Bugs")
	}
	milestones := result["milestones"].([]any)
	if len(milestones) != 1 {
		t.Fatalf("len(result.milestones) = %d, want 1", len(milestones))
	}
	fields := result["fields"].([]any)
	if len(fields) != 3 {
		t.Fatalf("len(result.fields) = %d, want 3", len(fields))
	}
	fieldByName := map[string]map[string]any{}
	for _, rawField := range fields {
		field := rawField.(map[string]any)
		fieldByName[field["name"].(string)] = field
	}
	statusField := fieldByName["status"]
	if statusField["name"] != "status" {
		t.Fatalf("status field name = %v, want %q", statusField["name"], "status")
	}
	statusValues := statusField["values"].([]any)
	firstStatus := statusValues[0].(map[string]any)
	if firstStatus["value"] != "open" {
		t.Fatalf("status value[0] = %v, want %q", firstStatus["value"], "open")
	}
	if _, ok := result["warnings"]; ok {
		t.Fatalf("result.warnings present, want omitted")
	}
	if _, ok := got["error"]; ok && got["error"] != nil {
		t.Fatalf("error = %v, want nil", got["error"])
	}
	if len(result["saved_bins"].([]any)) != 1 {
		t.Fatalf("len(result.saved_bins) = %d, want 1", len(result["saved_bins"].([]any)))
	}
	assignedField := fieldByName["assigned_to"]
	assignedValues := assignedField["values"].([]any)
	if assignedValues[0].(map[string]any)["value"] != "alice" {
		t.Fatalf("assigned_to value[0] = %v, want %q", assignedValues[0].(map[string]any)["value"], "alice")
	}
}

func TestTrackerSchemaSucceedsWhenSearchMetadataIsUnavailable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test/bugs":
			_, _ = w.Write([]byte(`{"tracker_config":{"options":{"mount_label":"Bugs"}}}`))
		case "/rest/p/test/bugs/search":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tracker", "schema", "--project", "test", "--tracker", "bugs"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	result := got["result"].(map[string]any)
	warnings := result["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("len(result.warnings) = %d, want 1", len(warnings))
	}
	if result["options"].(map[string]any)["mount_label"] != "Bugs" {
		t.Fatalf("result.options.mount_label = %v, want %q", result["options"].(map[string]any)["mount_label"], "Bugs")
	}
	if got["ok"] != true {
		t.Fatalf("ok = %v, want true", got["ok"])
	}
}

func TestTrackerSchemaRequiresProjectAndTracker(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"tracker", "schema", "--tracker", "bugs"}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got["command"] != "tracker.schema" {
		t.Fatalf("command = %v, want %q", got["command"], "tracker.schema")
	}
	errorValue := got["error"].(map[string]any)
	if errorValue["code"] != "invalid_arguments" {
		t.Fatalf("error.code = %v, want %q", errorValue["code"], "invalid_arguments")
	}
}
