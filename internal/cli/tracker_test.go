package cli

import (
	"bytes"
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

	got := decodeEnvelope(t, stdout.Bytes())

	if got.Command != "tracker.schema" {
		t.Fatalf("command = %q, want %q", got.Command, "tracker.schema")
	}
	if !got.OK {
		t.Fatalf("ok = %v, want true", got.OK)
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}

	if got.Proposal == nil || got.Proposal.Action != "get_tracker_schema" {
		t.Fatalf("proposal = %#v, want action %q", got.Proposal, "get_tracker_schema")
	}
	result := got.Result.(map[string]any)
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
	statusValidation := statusField["validation"].(map[string]any)
	if statusValidation["type"] != "choice" {
		t.Fatalf("status validation.type = %v, want %q", statusValidation["type"], "choice")
	}
	statusValues := statusField["values"].([]any)
	firstStatus := statusValues[0].(map[string]any)
	if firstStatus["value"] != "open" {
		t.Fatalf("status value[0] = %v, want %q", firstStatus["value"], "open")
	}
	allowedValues := statusValidation["allowed_values"].([]any)
	if allowedValues[0].(map[string]any)["value"] != "open" {
		t.Fatalf("status validation.allowed_values[0] = %v, want %q", allowedValues[0].(map[string]any)["value"], "open")
	}
	if _, ok := result["warnings"]; ok {
		t.Fatalf("result.warnings present, want omitted")
	}
	if got.Error != nil {
		t.Fatalf("error = %v, want nil", got.Error)
	}
	if len(result["saved_bins"].([]any)) != 1 {
		t.Fatalf("len(result.saved_bins) = %d, want 1", len(result["saved_bins"].([]any)))
	}
	assignedField := fieldByName["assigned_to"]
	assignedValues := assignedField["values"].([]any)
	if assignedValues[0].(map[string]any)["value"] != "alice" {
		t.Fatalf("assigned_to value[0] = %v, want %q", assignedValues[0].(map[string]any)["value"], "alice")
	}
	milestoneField := fieldByName["_milestone"]
	milestoneValidation := milestoneField["validation"].(map[string]any)
	if milestoneValidation["default"] != "v1" {
		t.Fatalf("milestone validation.default = %v, want %q", milestoneValidation["default"], "v1")
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

	got := decodeEnvelope(t, stdout.Bytes())

	result := got.Result.(map[string]any)
	if len(got.Warnings) != 1 {
		t.Fatalf("len(warnings) = %d, want 1", len(got.Warnings))
	}
	if result["options"].(map[string]any)["mount_label"] != "Bugs" {
		t.Fatalf("result.options.mount_label = %v, want %q", result["options"].(map[string]any)["mount_label"], "Bugs")
	}
	if !got.OK {
		t.Fatalf("ok = %v, want true", got.OK)
	}
	if _, ok := result["warnings"]; ok {
		t.Fatalf("result.warnings present, want omitted")
	}
}

func TestTrackerSchemaSkipsMalformedFilterChoices(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test/bugs":
			_, _ = w.Write([]byte(`{"tracker_config":{"options":{"mount_label":"Bugs"}}}`))
		case "/rest/p/test/bugs/search":
			_, _ = w.Write([]byte(`{"filter_choices":{"status":[[],["open","2"],{"bad":true}],"assigned_to":null}}`))
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

	got := decodeEnvelope(t, stdout.Bytes())

	result := got.Result.(map[string]any)
	fields := result["fields"].([]any)
	if len(fields) != 2 {
		t.Fatalf("len(result.fields) = %d, want 2", len(fields))
	}
	statusField := fields[1].(map[string]any)
	values := statusField["values"].([]any)
	if len(values) != 2 {
		t.Fatalf("len(status.values) = %d, want 2", len(values))
	}
	openValue := values[0].(map[string]any)
	if openValue["value"] != "open" {
		t.Fatalf("status.values[0].value = %v, want %q", openValue["value"], "open")
	}
	if openValue["count"] != float64(2) {
		t.Fatalf("status.values[0].count = %v, want 2", openValue["count"])
	}
	malformedValue := values[1].(map[string]any)
	if malformedValue["value"] == nil {
		t.Fatal("status.values[1].value = nil, want retained malformed object")
	}
	if _, ok := malformedValue["count"]; ok {
		t.Fatalf("status.values[1].count present, want omitted")
	}
	validation := statusField["validation"].(map[string]any)
	if validation["type"] != "unknown" {
		t.Fatalf("status validation.type = %v, want %q", validation["type"], "unknown")
	}
	if _, ok := validation["allowed_values"]; ok {
		t.Fatalf("status validation.allowed_values present, want omitted")
	}
}

func TestTrackerSchemaRequiresProjectAndTracker(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"tracker", "schema", "--tracker", "bugs"}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	got := decodeEnvelope(t, stdout.Bytes())

	if got.Command != "tracker.schema" {
		t.Fatalf("command = %q, want %q", got.Command, "tracker.schema")
	}
	if got.Error == nil || got.Error.Code != "invalid_arguments" {
		t.Fatalf("error = %#v, want code %q", got.Error, "invalid_arguments")
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}
}
