package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTicketsListExecutesAPIRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/p/test/tickets" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/rest/p/test/tickets")
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Fatalf("limit = %q, want %q", got, "10")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":1,"summary":"First"}],"count":1,"page":0,"limit":10,"milestones":[{"name":"m1","description":"","due_date":"","default":false,"complete":false,"closed":0,"total":1}]}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "list", "--project", "test", "--tracker", "tickets", "--limit", "10"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())

	if got.Command != "tickets.list" {
		t.Fatalf("command = %q, want %q", got.Command, "tickets.list")
	}
	if !got.OK {
		t.Fatalf("ok = %v, want true", got.OK)
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}

	result := got.Result.(map[string]any)
	if result["count"] != float64(1) {
		t.Fatalf("result.count = %v, want 1", result["count"])
	}
	if result["limit"] != float64(10) {
		t.Fatalf("result.limit = %v, want 10", result["limit"])
	}
	if got.Proposal == nil || got.Proposal.Action != "list_tickets" {
		t.Fatalf("proposal = %#v, want action %q", got.Proposal, "list_tickets")
	}
	inputs := got.Proposal.Inputs
	if inputs["limit"] != float64(10) {
		t.Fatalf("proposal.inputs.limit = %v, want 10", inputs["limit"])
	}
}

func TestTicketsSearchExecutesAPIRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/p/test/tickets/search" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/rest/p/test/tickets/search")
		}
		if got := r.URL.Query().Get("q"); got != "status:open" {
			t.Fatalf("q = %q, want %q", got, "status:open")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":2,"summary":"Second"}],"count":1,"page":0,"limit":25,"sort":"ticket_num_i desc","filter_choices":{"status":["open","closed"]}}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "search", "--project", "test", "--tracker", "tickets", "--query", "status:open"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())

	if got.Command != "tickets.search" {
		t.Fatalf("command = %q, want %q", got.Command, "tickets.search")
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}
	inputs := got.Proposal.Inputs
	if inputs["query"] != "status:open" {
		t.Fatalf("proposal.inputs.query = %v, want %q", inputs["query"], "status:open")
	}
	result := got.Result.(map[string]any)
	if result["sort"] != "ticket_num_i desc" {
		t.Fatalf("result.sort = %v, want %q", result["sort"], "ticket_num_i desc")
	}
}

func TestTicketsListRequiresProjectAndTracker(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"tickets", "list", "--tracker", "tickets"}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Command != "tickets.list" {
		t.Fatalf("command = %q, want %q", got.Command, "tickets.list")
	}
	if got.Error == nil || got.Error.Code != "invalid_arguments" {
		t.Fatalf("error = %#v, want code %q", got.Error, "invalid_arguments")
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}
}

func TestTicketsListRejectsQueryFlagWithGuidance(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"tickets", "list", "--project", "fuse-emulator", "--tracker", "bugs", "--query", "status:open"}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	got := decodeEnvelope(t, stdout.Bytes())

	if got.Command != "tickets.list" {
		t.Fatalf("command = %q, want %q", got.Command, "tickets.list")
	}
	target := got.Proposal.Target
	if target["project"] != "" {
		t.Fatalf("proposal.target.project = %v, want empty string", target["project"])
	}
	message := got.Error.Message
	if !bytes.Contains([]byte(message), []byte("tickets search")) {
		t.Fatalf("error.message = %q, want guidance mentioning tickets search", message)
	}
	if got.Error == nil || got.Error.Code != "invalid_arguments" {
		t.Fatalf("error = %#v, want code %q", got.Error, "invalid_arguments")
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}
}

func TestTicketsGetExecutesAPIRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/p/test/tickets/42" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/rest/p/test/tickets/42")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","description":"Detailed ticket","status":"open","reported_by":"alice","assigned_to":"bob","labels":["triaged"],"private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42","subject":""},"custom_fields":{"_milestone":"unreleased"}}}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "get", "--project", "test", "--tracker", "tickets", "--ticket", "42"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Command != "tickets.get" {
		t.Fatalf("command = %q, want %q", got.Command, "tickets.get")
	}
	target := got.Proposal.Target
	if target["ticket"] != float64(42) {
		t.Fatalf("proposal.target.ticket = %v, want 42", target["ticket"])
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}
	result := got.Result.(map[string]any)
	ticket := result["ticket"].(map[string]any)
	if ticket["summary"] != "Answer" {
		t.Fatalf("ticket.summary = %v, want %q", ticket["summary"], "Answer")
	}
}

func TestTicketsCommentsFetchesTicketThenThread(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test/tickets/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","discussion_thread":{"_id":"thread-42"}}}`))
		case "/rest/p/test/tickets/_discuss/thread/thread-42":
			_, _ = w.Write([]byte(`{"thread":{"_id":"thread-42","subject":"","posts":[{"author":"alice","text":"first comment","is_meta":false,"timestamp":"2026-04-24T00:00:00Z","slug":"a1"}]}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "comments", "--project", "test", "--tracker", "tickets", "--ticket", "42"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}
	if requestCount != 2 {
		t.Fatalf("requestCount = %d, want 2", requestCount)
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Command != "tickets.comments" {
		t.Fatalf("command = %q, want %q", got.Command, "tickets.comments")
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}
	result := got.Result.(map[string]any)
	thread := result["thread"].(map[string]any)
	posts := thread["posts"].([]any)
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want 1", len(posts))
	}
	post := posts[0].(map[string]any)
	if post["text"] != "first comment" {
		t.Fatalf("post.text = %v, want %q", post["text"], "first comment")
	}
}

func TestTicketsCommentsReturnsAPINotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"ticket not found"}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "comments", "--project", "test", "--tracker", "tickets", "--ticket", "42"}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Command != "tickets.comments" {
		t.Fatalf("command = %q, want %q", got.Command, "tickets.comments")
	}
	if got.Error == nil || got.Error.Code != "api_error" {
		t.Fatalf("error = %#v, want code %q", got.Error, "api_error")
	}
	message := got.Error.Message
	if !bytes.Contains([]byte(message), []byte("ticket not found")) {
		t.Fatalf("error.message = %q, want to mention ticket not found", message)
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}
}
