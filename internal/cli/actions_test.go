package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func writeActionsFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "actions.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
