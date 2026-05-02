package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestActionsValidateAcceptsValidTicketCreateIntent(t *testing.T) {
	t.Parallel()

	var ticketRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			ticketRequests.Add(1)
			t.Fatalf("ticket_create validation should not fetch existing tickets")
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_create","project":"test","tracker":"bugs","summary":" New ticket ","description":"details","assigned_to":"alice","private":true,"custom_fields":{"_priority":"5"},"labels":[" triaged ","needs-review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	if result["ok"] != true {
		t.Fatalf("result.ok = %v, want true", result["ok"])
	}
	action := result["validated_actions"].([]any)[0].(map[string]any)
	if action["ok"] != true {
		t.Fatalf("validated action ok = %v, want true", action["ok"])
	}
	if ticketRequests.Load() != 0 {
		t.Fatalf("ticketRequests = %d, want 0", ticketRequests.Load())
	}
	normalized := action["action"].(map[string]any)
	if normalized["type"] != "ticket_create" {
		t.Fatalf("action.type = %v, want %q", normalized["type"], "ticket_create")
	}
	inputs := normalized["inputs"].(map[string]any)
	if inputs["summary"] != "New ticket" {
		t.Fatalf("inputs.summary = %v, want %q", inputs["summary"], "New ticket")
	}
	if inputs["status"] != "open" {
		t.Fatalf("inputs.status = %v, want %q", inputs["status"], "open")
	}
	if inputs["assigned_to"] != "alice" {
		t.Fatalf("inputs.assigned_to = %v, want %q", inputs["assigned_to"], "alice")
	}
	if inputs["private"] != true {
		t.Fatalf("inputs.private = %v, want true", inputs["private"])
	}
	customFields := inputs["custom_fields"].(map[string]any)
	if customFields["_priority"] != "5" {
		t.Fatalf("inputs.custom_fields._priority = %v, want %q", customFields["_priority"], "5")
	}
	labels := inputs["labels"].([]any)
	if len(labels) != 2 || labels[0] != "triaged" || labels[1] != "needs-review" {
		t.Fatalf("normalized labels = %v, want trimmed labels", labels)
	}
	canonical := action["canonical_identifiers"].(map[string]any)
	if canonical["project"] != "test" || canonical["tracker"] != "bugs" {
		t.Fatalf("canonical_identifiers = %v, want project/tracker", canonical)
	}
	if _, ok := canonical["ticket_num"]; ok {
		t.Fatalf("canonical_identifiers.ticket_num = %v, want omitted", canonical["ticket_num"])
	}
	if _, ok := action["issues"]; ok {
		t.Fatalf("issues = %v, want omitted", action["issues"])
	}
}

func TestActionsValidateRejectsUnsupportedTicketCreateFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_create","project":"test","tracker":"bugs","ticket":42,"summary":"New ticket","discussion_disabled":false}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	if result["ok"] != false {
		t.Fatalf("result.ok = %v, want false", result["ok"])
	}
	action := result["validated_actions"].([]any)[0].(map[string]any)
	issues := action["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "unsupported_ticket_target" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "unsupported_ticket_target")
	}
	unsupportedFields := map[string]bool{}
	for _, rawIssue := range issues[1:] {
		issue := rawIssue.(map[string]any)
		if issue["code"] != "unsupported_ticket_create_field" {
			t.Fatalf("issue.code = %v, want %q", issue["code"], "unsupported_ticket_create_field")
		}
		unsupportedFields[issue["field"].(string)] = true
	}
	for _, field := range []string{"discussion_disabled"} {
		if !unsupportedFields[field] {
			t.Fatalf("unsupported fields = %v, want %q present", unsupportedFields, field)
		}
	}
}

func TestActionsValidateReportsInvalidTicketCreateSummaryAndLabels(t *testing.T) {
	t.Parallel()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_create","project":"test","tracker":"bugs","summary":" ","labels":[" ","needs,review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	if result["ok"] != false {
		t.Fatalf("result.ok = %v, want false", result["ok"])
	}
	action := result["validated_actions"].([]any)[0].(map[string]any)
	issues := action["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "missing_summary" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "missing_summary")
	}
	if issues[1].(map[string]any)["code"] != "empty_label" {
		t.Fatalf("issues[1].code = %v, want %q", issues[1].(map[string]any)["code"], "empty_label")
	}
	if issues[2].(map[string]any)["code"] != "unsupported_label_value" {
		t.Fatalf("issues[2].code = %v, want %q", issues[2].(map[string]any)["code"], "unsupported_label_value")
	}
}

func TestActionsValidateAcceptsValidTicketCommentIntent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"}}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"hello"}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Command != "actions.validate" {
		t.Fatalf("command = %q, want %q", got.Command, "actions.validate")
	}
	result := got.Result.(map[string]any)
	if result["ok"] != true {
		t.Fatalf("result.ok = %v, want true", result["ok"])
	}
	validated := result["validated_actions"].([]any)
	action := validated[0].(map[string]any)
	if action["ok"] != true {
		t.Fatalf("validated action ok = %v, want true", action["ok"])
	}
	normalized := action["action"].(map[string]any)
	if normalized["type"] != "ticket_comment" {
		t.Fatalf("action.type = %v, want %q", normalized["type"], "ticket_comment")
	}
	canonical := action["canonical_identifiers"].(map[string]any)
	if canonical["ticket_num"] != float64(42) {
		t.Fatalf("canonical_identifiers.ticket_num = %v, want 42", canonical["ticket_num"])
	}
	if canonical["discussion_thread_id"] != "thread-42" {
		t.Fatalf("canonical_identifiers.discussion_thread_id = %v, want %q", canonical["discussion_thread_id"], "thread-42")
	}
	if _, ok := action["issues"]; ok {
		t.Fatalf("issues = %v, want omitted", action["issues"])
	}
}

func TestActionsValidateReportsInvalidTicketCommentIntent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/999":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"ticket not found"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":999,"body":"   "}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	if result["ok"] != false {
		t.Fatalf("result.ok = %v, want false", result["ok"])
	}
	action := result["validated_actions"].([]any)[0].(map[string]any)
	canonical := action["canonical_identifiers"].(map[string]any)
	if canonical["project"] != "test" {
		t.Fatalf("canonical_identifiers.project = %v, want %q", canonical["project"], "test")
	}
	if canonical["tracker"] != "bugs" {
		t.Fatalf("canonical_identifiers.tracker = %v, want %q", canonical["tracker"], "bugs")
	}
	if canonical["ticket_num"] != float64(999) {
		t.Fatalf("canonical_identifiers.ticket_num = %v, want 999", canonical["ticket_num"])
	}
	issues := action["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "empty_body" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "empty_body")
	}
}

func TestActionsValidateRejectsTicketCommentWhenDiscussionDisabled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":true,"discussion_thread":{"_id":"thread-42"}}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"hello"}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	if result["ok"] != false {
		t.Fatalf("result.ok = %v, want false", result["ok"])
	}
	action := result["validated_actions"].([]any)[0].(map[string]any)
	issues := action["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "ticket_discussion_disabled" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "ticket_discussion_disabled")
	}
	canonical := action["canonical_identifiers"].(map[string]any)
	if canonical["discussion_thread_id"] != "thread-42" {
		t.Fatalf("canonical_identifiers.discussion_thread_id = %v, want %q", canonical["discussion_thread_id"], "thread-42")
	}
}

func TestActionsValidateRejectsTicketCommentWithoutDiscussionThreadID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":""}}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"hello"}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	if result["ok"] != false {
		t.Fatalf("result.ok = %v, want false", result["ok"])
	}
	action := result["validated_actions"].([]any)[0].(map[string]any)
	issues := action["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "discussion_thread_unavailable" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "discussion_thread_unavailable")
	}
	canonical := action["canonical_identifiers"].(map[string]any)
	if _, ok := canonical["discussion_thread_id"]; ok {
		t.Fatalf("canonical_identifiers.discussion_thread_id = %v, want omitted", canonical["discussion_thread_id"])
	}
}

func TestActionsValidateAcceptsValidTicketLabelsIntent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"},"labels":["triaged"]}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_labels","project":"test","tracker":"bugs","ticket":42,"labels":[" triaged ","needs-review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	if result["ok"] != true {
		t.Fatalf("result.ok = %v, want true", result["ok"])
	}
	action := result["validated_actions"].([]any)[0].(map[string]any)
	if action["ok"] != true {
		t.Fatalf("validated action ok = %v, want true", action["ok"])
	}
	normalized := action["action"].(map[string]any)
	if normalized["type"] != "ticket_labels" {
		t.Fatalf("action.type = %v, want %q", normalized["type"], "ticket_labels")
	}
	labels := normalized["inputs"].(map[string]any)["labels"].([]any)
	if len(labels) != 2 || labels[0] != "triaged" || labels[1] != "needs-review" {
		t.Fatalf("normalized labels = %v, want trimmed labels", labels)
	}
	canonical := action["canonical_identifiers"].(map[string]any)
	if canonical["ticket_num"] != float64(42) {
		t.Fatalf("canonical_identifiers.ticket_num = %v, want 42", canonical["ticket_num"])
	}
	if _, ok := canonical["discussion_thread_id"]; ok {
		t.Fatalf("canonical_identifiers.discussion_thread_id = %v, want omitted", canonical["discussion_thread_id"])
	}
	if _, ok := action["issues"]; ok {
		t.Fatalf("issues = %v, want omitted", action["issues"])
	}
}

func TestActionsValidateReportsUnsupportedTicketLabelsIntent(t *testing.T) {
	t.Parallel()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_labels","project":"test","tracker":"bugs","ticket":42,"labels":[" ","needs,review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	if result["ok"] != false {
		t.Fatalf("result.ok = %v, want false", result["ok"])
	}
	action := result["validated_actions"].([]any)[0].(map[string]any)
	if action["ok"] != false {
		t.Fatalf("validated action ok = %v, want false", action["ok"])
	}
	issues := action["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "empty_label" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "empty_label")
	}
	if issues[1].(map[string]any)["code"] != "unsupported_label_value" {
		t.Fatalf("issues[1].code = %v, want %q", issues[1].(map[string]any)["code"], "unsupported_label_value")
	}
}

func TestActionsValidateAcceptsLongTicketCommentBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"}}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	longBody := strings.Repeat("a", 70000)
	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"`+longBody+`"}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "validate", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	if result["ok"] != true {
		t.Fatalf("result.ok = %v, want true", result["ok"])
	}
	action := result["validated_actions"].([]any)[0].(map[string]any)
	if action["ok"] != true {
		t.Fatalf("validated action ok = %v, want true", action["ok"])
	}
	if _, ok := action["issues"]; ok {
		t.Fatalf("issues = %v, want omitted", action["issues"])
	}
}

func TestActionsApplyDefaultsToDryRunWithoutConfirm(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"}}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"hello"}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "apply", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "dry_run" {
		t.Fatalf("mode = %q, want %q", got.Mode, "dry_run")
	}
	if got.Command != "actions.apply" {
		t.Fatalf("command = %q, want %q", got.Command, "actions.apply")
	}
	result := got.Result.(map[string]any)
	if result["confirmed"] != false {
		t.Fatalf("result.confirmed = %v, want false", result["confirmed"])
	}
	if result["executed"] != false {
		t.Fatalf("result.executed = %v, want false", result["executed"])
	}
	validated := result["validated_actions"].([]any)
	if len(validated) != 1 {
		t.Fatalf("len(validated_actions) = %d, want 1", len(validated))
	}
	if validated[0].(map[string]any)["ok"] != true {
		t.Fatalf("validated action ok = %v, want true", validated[0].(map[string]any)["ok"])
	}
	if got.Proposal == nil || got.Proposal.Action != "apply_actions_file" {
		t.Fatalf("proposal = %#v, want action %q", got.Proposal, "apply_actions_file")
	}
	if got.Proposal.Inputs["confirm"] != false {
		t.Fatalf("proposal.inputs.confirm = %v, want false", got.Proposal.Inputs["confirm"])
	}
}

func TestActionsApplyDryRunPreviewsTicketLabelsWithoutExecution(t *testing.T) {
	t.Parallel()

	var saveRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"},"labels":["triaged"]}}`))
		case "/rest/p/test/bugs/42/save":
			saveRequests.Add(1)
			t.Fatalf("ticket label save should not run during dry-run")
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_labels","project":"test","tracker":"bugs","ticket":42,"labels":["triaged","needs-review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "apply", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "dry_run" {
		t.Fatalf("mode = %q, want %q", got.Mode, "dry_run")
	}
	result := got.Result.(map[string]any)
	if result["confirmed"] != false {
		t.Fatalf("result.confirmed = %v, want false", result["confirmed"])
	}
	if result["executed"] != false {
		t.Fatalf("result.executed = %v, want false", result["executed"])
	}
	if saveRequests.Load() != 0 {
		t.Fatalf("saveRequests = %d, want 0", saveRequests.Load())
	}
	validated := result["validated_actions"].([]any)
	if validated[0].(map[string]any)["ok"] != true {
		t.Fatalf("validated action ok = %v, want true", validated[0].(map[string]any)["ok"])
	}
}

func TestActionsApplyDryRunPreviewsTicketCreateWithoutExecution(t *testing.T) {
	t.Parallel()

	var createRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/new":
			createRequests.Add(1)
			t.Fatalf("ticket create should not run during dry-run")
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_create","project":"test","tracker":"bugs","summary":" New ticket ","description":"details","assigned_to":"alice","private":true,"custom_fields":{"_priority":"5"},"labels":[" triaged ","needs-review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "apply", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "dry_run" {
		t.Fatalf("mode = %q, want %q", got.Mode, "dry_run")
	}
	result := got.Result.(map[string]any)
	if result["confirmed"] != false {
		t.Fatalf("result.confirmed = %v, want false", result["confirmed"])
	}
	if result["executed"] != false {
		t.Fatalf("result.executed = %v, want false", result["executed"])
	}
	if createRequests.Load() != 0 {
		t.Fatalf("createRequests = %d, want 0", createRequests.Load())
	}
	validated := result["validated_actions"].([]any)
	if validated[0].(map[string]any)["ok"] != true {
		t.Fatalf("validated action ok = %v, want true", validated[0].(map[string]any)["ok"])
	}
	inputs := validated[0].(map[string]any)["action"].(map[string]any)["inputs"].(map[string]any)
	if inputs["summary"] != "New ticket" {
		t.Fatalf("inputs.summary = %v, want %q", inputs["summary"], "New ticket")
	}
	if inputs["status"] != "open" {
		t.Fatalf("inputs.status = %v, want %q", inputs["status"], "open")
	}
	labels := inputs["labels"].([]any)
	if len(labels) != 2 || labels[0] != "triaged" || labels[1] != "needs-review" {
		t.Fatalf("normalized labels = %v, want trimmed labels", labels)
	}
}

func TestActionsApplyRejectsInvalidActionsBeforeExecution(t *testing.T) {
	t.Parallel()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"   "}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "dry_run" {
		t.Fatalf("mode = %q, want %q", got.Mode, "dry_run")
	}
	if got.Error == nil || got.Error.Code != "invalid_actions" {
		t.Fatalf("error = %#v, want code %q", got.Error, "invalid_actions")
	}
	result := got.Result.(map[string]any)
	if result["confirmed"] != true {
		t.Fatalf("result.confirmed = %v, want true", result["confirmed"])
	}
	if result["executed"] != false {
		t.Fatalf("result.executed = %v, want false", result["executed"])
	}
	validated := result["validated_actions"].([]any)
	issues := validated[0].(map[string]any)["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "empty_body" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "empty_body")
	}
}

func TestActionsApplyExecutesConfirmedTicketComment(t *testing.T) {
	t.Parallel()

	var postRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"}}}`))
		case "/rest/p/test/bugs/_discuss/thread/thread-42/new":
			postRequests.Add(1)
			if r.Method != http.MethodPost {
				t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
				t.Fatalf("Authorization header = %q, want %q", got, "Bearer secret-token")
			}
			if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/x-www-form-urlencoded") {
				t.Fatalf("Content-Type = %q, want form encoding", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if got := string(body); got != "text=hello" {
				t.Fatalf("request body = %q, want %q", got, "text=hello")
			}
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"hello"}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "--token", "secret-token", "actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "apply" {
		t.Fatalf("mode = %q, want %q", got.Mode, "apply")
	}
	if got.Error != nil {
		t.Fatalf("error = %#v, want nil", got.Error)
	}
	result := got.Result.(map[string]any)
	if result["confirmed"] != true {
		t.Fatalf("result.confirmed = %v, want true", result["confirmed"])
	}
	if result["executed"] != true {
		t.Fatalf("result.executed = %v, want true", result["executed"])
	}
	applied := result["applied_actions"].([]any)
	if len(applied) != 1 {
		t.Fatalf("len(applied_actions) = %d, want 1", len(applied))
	}
	if applied[0].(map[string]any)["ok"] != true {
		t.Fatalf("applied action ok = %v, want true", applied[0].(map[string]any)["ok"])
	}
	if _, ok := applied[0].(map[string]any)["issues"]; ok {
		t.Fatalf("issues = %v, want omitted", applied[0].(map[string]any)["issues"])
	}
	if postRequests.Load() != 1 {
		t.Fatalf("postRequests = %d, want 1", postRequests.Load())
	}
}

func TestActionsApplyReturnsAPIErrorWhenTicketCommentPostFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"}}}`))
		case "/rest/p/test/bugs/_discuss/thread/thread-42/new":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"upstream failed"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"hello"}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "--token", "secret-token", "actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "apply" {
		t.Fatalf("mode = %q, want %q", got.Mode, "apply")
	}
	if got.Error == nil || got.Error.Code != "api_error" {
		t.Fatalf("error = %#v, want code %q", got.Error, "api_error")
	}
	result := got.Result.(map[string]any)
	if result["executed"] != true {
		t.Fatalf("result.executed = %v, want true", result["executed"])
	}
	applied := result["applied_actions"].([]any)
	issues := applied[0].(map[string]any)["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "api_error" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "api_error")
	}
}

func TestActionsApplyExecutesConfirmedTicketCreate(t *testing.T) {
	t.Parallel()

	var createRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/new":
			createRequests.Add(1)
			if r.Method != http.MethodPost {
				t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
				t.Fatalf("Authorization header = %q, want %q", got, "Bearer secret-token")
			}
			if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/x-www-form-urlencoded") {
				t.Fatalf("Content-Type = %q, want form encoding", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if got := string(body); got != "ticket_form.assigned_to=alice&ticket_form.custom_fields._priority=5&ticket_form.description=details&ticket_form.labels=triaged%2Cneeds-review&ticket_form.private=on&ticket_form.status=open&ticket_form.summary=New+ticket" {
				t.Fatalf("request body = %q, want %q", got, "ticket_form.assigned_to=alice&ticket_form.custom_fields._priority=5&ticket_form.description=details&ticket_form.labels=triaged%2Cneeds-review&ticket_form.private=on&ticket_form.status=open&ticket_form.summary=New+ticket")
			}
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_create","project":"test","tracker":"bugs","summary":" New ticket ","description":"details","assigned_to":"alice","private":true,"custom_fields":{"_priority":"5"},"labels":[" triaged ","needs-review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "--token", "secret-token", "actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "apply" {
		t.Fatalf("mode = %q, want %q", got.Mode, "apply")
	}
	if got.Error != nil {
		t.Fatalf("error = %#v, want nil", got.Error)
	}
	result := got.Result.(map[string]any)
	if result["confirmed"] != true {
		t.Fatalf("result.confirmed = %v, want true", result["confirmed"])
	}
	if result["executed"] != true {
		t.Fatalf("result.executed = %v, want true", result["executed"])
	}
	applied := result["applied_actions"].([]any)
	if len(applied) != 1 {
		t.Fatalf("len(applied_actions) = %d, want 1", len(applied))
	}
	if applied[0].(map[string]any)["ok"] != true {
		t.Fatalf("applied action ok = %v, want true", applied[0].(map[string]any)["ok"])
	}
	if _, ok := applied[0].(map[string]any)["issues"]; ok {
		t.Fatalf("issues = %v, want omitted", applied[0].(map[string]any)["issues"])
	}
	if createRequests.Load() != 1 {
		t.Fatalf("createRequests = %d, want 1", createRequests.Load())
	}
}

func TestActionsApplyReturnsAPIErrorWhenTicketCreateFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/new":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"upstream failed"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_create","project":"test","tracker":"bugs","summary":"New ticket"}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "--token", "secret-token", "actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Error == nil || got.Error.Code != "api_error" {
		t.Fatalf("error = %#v, want code %q", got.Error, "api_error")
	}
	result := got.Result.(map[string]any)
	if result["executed"] != true {
		t.Fatalf("result.executed = %v, want true", result["executed"])
	}
	applied := result["applied_actions"].([]any)
	issues := applied[0].(map[string]any)["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "api_error" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "api_error")
	}
}

func TestActionsApplyRequiresTokenForConfirmedWrites(t *testing.T) {
	t.Setenv(envBearerToken, "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"},"labels":["triaged"]}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_labels","project":"test","tracker":"bugs","ticket":42,"labels":["triaged","needs-review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "dry_run" {
		t.Fatalf("mode = %q, want %q", got.Mode, "dry_run")
	}
	if got.Error == nil || got.Error.Code != "authentication_required" {
		t.Fatalf("error = %#v, want code %q", got.Error, "authentication_required")
	}
	result := got.Result.(map[string]any)
	if result["confirmed"] != true {
		t.Fatalf("result.confirmed = %v, want true", result["confirmed"])
	}
	if result["executed"] != false {
		t.Fatalf("result.executed = %v, want false", result["executed"])
	}
	if _, ok := result["applied_actions"]; ok {
		t.Fatalf("applied_actions = %v, want omitted", result["applied_actions"])
	}
}

func TestActionsApplyExecutesConfirmedTicketLabels(t *testing.T) {
	t.Parallel()

	var saveRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"},"labels":["triaged"]}}`))
		case "/rest/p/test/bugs/42/save":
			saveRequests.Add(1)
			if r.Method != http.MethodPost {
				t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
				t.Fatalf("Authorization header = %q, want %q", got, "Bearer secret-token")
			}
			if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/x-www-form-urlencoded") {
				t.Fatalf("Content-Type = %q, want form encoding", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if got := string(body); got != "ticket_form.labels=triaged%2Cneeds-review" {
				t.Fatalf("request body = %q, want %q", got, "ticket_form.labels=triaged%2Cneeds-review")
			}
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_labels","project":"test","tracker":"bugs","ticket":42,"labels":["triaged","needs-review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "--token", "secret-token", "actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "apply" {
		t.Fatalf("mode = %q, want %q", got.Mode, "apply")
	}
	if got.Error != nil {
		t.Fatalf("error = %#v, want nil", got.Error)
	}
	result := got.Result.(map[string]any)
	if result["executed"] != true {
		t.Fatalf("result.executed = %v, want true", result["executed"])
	}
	applied := result["applied_actions"].([]any)
	if len(applied) != 1 {
		t.Fatalf("len(applied_actions) = %d, want 1", len(applied))
	}
	if applied[0].(map[string]any)["ok"] != true {
		t.Fatalf("applied action ok = %v, want true", applied[0].(map[string]any)["ok"])
	}
	if _, ok := applied[0].(map[string]any)["issues"]; ok {
		t.Fatalf("issues = %v, want omitted", applied[0].(map[string]any)["issues"])
	}
	if saveRequests.Load() != 1 {
		t.Fatalf("saveRequests = %d, want 1", saveRequests.Load())
	}
}

func TestActionsApplyReturnsAPIErrorWhenTicketLabelSaveFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"},"labels":["triaged"]}}`))
		case "/rest/p/test/bugs/42/save":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"upstream failed"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_labels","project":"test","tracker":"bugs","ticket":42,"labels":["triaged","needs-review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "--token", "secret-token", "actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Error == nil || got.Error.Code != "api_error" {
		t.Fatalf("error = %#v, want code %q", got.Error, "api_error")
	}
	result := got.Result.(map[string]any)
	if result["executed"] != true {
		t.Fatalf("result.executed = %v, want true", result["executed"])
	}
	applied := result["applied_actions"].([]any)
	issues := applied[0].(map[string]any)["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "api_error" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "api_error")
	}
}

func TestActionsApplyExecutesMixedCreateCommentAndLabelsFile(t *testing.T) {
	t.Parallel()

	var createRequests atomic.Int32
	var postRequests atomic.Int32
	var saveRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/new":
			createRequests.Add(1)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"},"labels":["triaged"]}}`))
		case "/rest/p/test/bugs/_discuss/thread/thread-42/new":
			postRequests.Add(1)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/rest/p/test/bugs/42/save":
			saveRequests.Add(1)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_create","project":"test","tracker":"bugs","summary":"new ticket"},{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"hello"},{"type":"ticket_labels","project":"test","tracker":"bugs","ticket":42,"labels":["triaged","needs-review"]}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "--token", "secret-token", "actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Mode != "apply" {
		t.Fatalf("mode = %q, want %q", got.Mode, "apply")
	}
	if got.Error != nil {
		t.Fatalf("error = %#v, want nil", got.Error)
	}
	result := got.Result.(map[string]any)
	if result["executed"] != true {
		t.Fatalf("result.executed = %v, want true", result["executed"])
	}
	applied := result["applied_actions"].([]any)
	if len(applied) != 3 {
		t.Fatalf("len(applied_actions) = %d, want 3", len(applied))
	}
	if applied[0].(map[string]any)["ok"] != true || applied[1].(map[string]any)["ok"] != true || applied[2].(map[string]any)["ok"] != true {
		t.Fatalf("applied actions = %v, want both ok", applied)
	}
	if createRequests.Load() != 1 {
		t.Fatalf("createRequests = %d, want 1", createRequests.Load())
	}
	if postRequests.Load() != 1 {
		t.Fatalf("postRequests = %d, want 1", postRequests.Load())
	}
	if saveRequests.Load() != 1 {
		t.Fatalf("saveRequests = %d, want 1", saveRequests.Load())
	}
}

func TestActionsApplyRejectsMixedFilesContainingUnsupportedTypes(t *testing.T) {
	t.Parallel()

	var createRequests atomic.Int32
	var postRequests atomic.Int32
	var saveRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test":
			_, _ = w.Write([]byte(`{"shortname":"test","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs"}]}`))
		case "/rest/p/test/bugs/new":
			createRequests.Add(1)
			t.Fatalf("ticket create should not run when unsupported actions are present")
		case "/rest/p/test/bugs/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"},"labels":["triaged"]}}`))
		case "/rest/p/test/bugs/_discuss/thread/thread-42/new":
			postRequests.Add(1)
			t.Fatalf("comment post should not run when unsupported actions are present")
		case "/rest/p/test/bugs/42/save":
			saveRequests.Add(1)
			t.Fatalf("ticket label save should not run when unsupported actions are present")
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	actionsPath := writeActionsFile(t, `{"actions":[{"type":"ticket_comment","project":"test","tracker":"bugs","ticket":42,"body":"hello"},{"type":"wiki_edit","project":"test","tracker":"bugs"}]}`)
	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "--token", "secret-token", "actions", "apply", "--confirm", actionsPath}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Error == nil || got.Error.Code != "invalid_actions" {
		t.Fatalf("error = %#v, want code %q", got.Error, "invalid_actions")
	}
	result := got.Result.(map[string]any)
	if result["executed"] != false {
		t.Fatalf("result.executed = %v, want false", result["executed"])
	}
	validated := result["validated_actions"].([]any)
	issues := validated[1].(map[string]any)["issues"].([]any)
	if issues[0].(map[string]any)["code"] != "unsupported_action_type" {
		t.Fatalf("issues[0].code = %v, want %q", issues[0].(map[string]any)["code"], "unsupported_action_type")
	}
	if createRequests.Load() != 0 {
		t.Fatalf("createRequests = %d, want 0", createRequests.Load())
	}
	if postRequests.Load() != 0 {
		t.Fatalf("postRequests = %d, want 0", postRequests.Load())
	}
	if saveRequests.Load() != 0 {
		t.Fatalf("saveRequests = %d, want 0", saveRequests.Load())
	}
}

func writeActionsFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "actions.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
