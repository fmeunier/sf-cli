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
		_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":1,"summary":"First","status":"open","assigned_to":"alice","labels":["triaged"],"created_date":"2026-04-24T00:00:00Z"}],"count":1,"page":0,"limit":10,"milestones":[{"name":"m1","description":"","due_date":"","default":false,"complete":false,"closed":0,"total":1}]}`))
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
	pagination := result["pagination"].(map[string]any)
	if pagination["page"] != float64(0) {
		t.Fatalf("pagination.page = %v, want 0", pagination["page"])
	}
	if pagination["has_previous"] != false {
		t.Fatalf("pagination.has_previous = %v, want false", pagination["has_previous"])
	}
	if pagination["has_next"] != false {
		t.Fatalf("pagination.has_next = %v, want false", pagination["has_next"])
	}
	tickets := result["tickets"].([]any)
	firstTicket := tickets[0].(map[string]any)
	if firstTicket["status"] != "open" {
		t.Fatalf("ticket.status = %v, want %q", firstTicket["status"], "open")
	}
	if firstTicket["assigned_to"] != "alice" {
		t.Fatalf("ticket.assigned_to = %v, want %q", firstTicket["assigned_to"], "alice")
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
		_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":2,"summary":"Second","status":"open","assigned_to":"alice","labels":["triaged"],"created_date":"2026-04-24T00:00:00Z"}],"count":1,"page":0,"limit":25,"sort":"ticket_num_i desc","filter_choices":{"status":["open","closed"]}}`))
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
	pagination := result["pagination"].(map[string]any)
	if pagination["page"] != float64(0) {
		t.Fatalf("pagination.page = %v, want 0", pagination["page"])
	}
	if pagination["has_next"] != false {
		t.Fatalf("pagination.has_next = %v, want false", pagination["has_next"])
	}
	tickets := result["tickets"].([]any)
	firstTicket := tickets[0].(map[string]any)
	if firstTicket["status"] != "open" {
		t.Fatalf("ticket.status = %v, want %q", firstTicket["status"], "open")
	}
	if firstTicket["assigned_to"] != "alice" {
		t.Fatalf("ticket.assigned_to = %v, want %q", firstTicket["assigned_to"], "alice")
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

func TestTicketOutputsShareCommonOverlappingFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/rest/p/test/tickets":
			_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":42,"summary":"Answer","status":"open","reported_by":"alice","assigned_to":"bob","labels":["triaged"],"created_date":"2026-04-24T00:00:00Z","mod_date":"2026-04-24T01:00:00Z"}],"count":1,"page":0,"limit":25}`))
		case "/rest/p/test/tickets/search":
			_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":42,"summary":"Answer","status":"open","reported_by":"alice","assigned_to":"bob","labels":["triaged"],"created_date":"2026-04-24T00:00:00Z","mod_date":"2026-04-24T01:00:00Z"}],"count":1,"page":0,"limit":25}`))
		case "/rest/p/test/tickets/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","description":"Detailed ticket","status":"open","reported_by":"alice","assigned_to":"bob","labels":["triaged"],"private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42","subject":""},"created_date":"2026-04-24T00:00:00Z","mod_date":"2026-04-24T01:00:00Z"}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	listOut := &bytes.Buffer{}
	if status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "list", "--project", "test", "--tracker", "tickets"}, listOut); status != 0 {
		t.Fatalf("tickets list status = %d, want 0; output=%s", status, listOut.String())
	}
	searchOut := &bytes.Buffer{}
	if status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "search", "--project", "test", "--tracker", "tickets", "--query", "status:open"}, searchOut); status != 0 {
		t.Fatalf("tickets search status = %d, want 0; output=%s", status, searchOut.String())
	}
	getOut := &bytes.Buffer{}
	if status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "get", "--project", "test", "--tracker", "tickets", "--ticket", "42"}, getOut); status != 0 {
		t.Fatalf("tickets get status = %d, want 0; output=%s", status, getOut.String())
	}

	listTicket := decodeEnvelope(t, listOut.Bytes()).Result.(map[string]any)["tickets"].([]any)[0].(map[string]any)
	searchTicket := decodeEnvelope(t, searchOut.Bytes()).Result.(map[string]any)["tickets"].([]any)[0].(map[string]any)
	getTicket := decodeEnvelope(t, getOut.Bytes()).Result.(map[string]any)["ticket"].(map[string]any)

	for _, field := range []string{"ticket_num", "summary", "status", "reported_by", "assigned_to", "created_date", "mod_date"} {
		if listTicket[field] != getTicket[field] {
			t.Fatalf("list %s = %v, want %v", field, listTicket[field], getTicket[field])
		}
		if searchTicket[field] != getTicket[field] {
			t.Fatalf("search %s = %v, want %v", field, searchTicket[field], getTicket[field])
		}
	}

	listLabels := listTicket["labels"].([]any)
	searchLabels := searchTicket["labels"].([]any)
	getLabels := getTicket["labels"].([]any)
	if len(listLabels) != 1 || listLabels[0] != getLabels[0] {
		t.Fatalf("list labels = %v, want %v", listLabels, getLabels)
	}
	if len(searchLabels) != 1 || searchLabels[0] != getLabels[0] {
		t.Fatalf("search labels = %v, want %v", searchLabels, getLabels)
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
	if thread["id"] != "thread-42" {
		t.Fatalf("thread.id = %v, want %q", thread["id"], "thread-42")
	}
	comments := result["comments"].([]any)
	if len(comments) != 1 {
		t.Fatalf("len(comments) = %d, want 1", len(comments))
	}
	comment := comments[0].(map[string]any)
	if comment["body"] != "first comment" {
		t.Fatalf("comment.body = %v, want %q", comment["body"], "first comment")
	}
	if comment["id"] != "a1" {
		t.Fatalf("comment.id = %v, want %q", comment["id"], "a1")
	}
	if comment["created_at"] != "2026-04-24T00:00:00Z" {
		t.Fatalf("comment.created_at = %v, want timestamp", comment["created_at"])
	}
}

func TestTicketsCommentsNormalizesOrderingDeterministically(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test/tickets/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","discussion_thread":{"_id":"thread-42"}}}`))
		case "/rest/p/test/tickets/_discuss/thread/thread-42":
			_, _ = w.Write([]byte(`{"thread":{"_id":"thread-42","subject":"Discussion","posts":[{"author":"zoe","text":"no timestamp","is_meta":false,"slug":"z2"},{"author":"alice","text":"first","is_meta":false,"timestamp":"2026-04-24T00:00:00Z","slug":"a1"},{"author":"bob","text":"second same time","is_meta":false,"timestamp":"2026-04-24T00:00:00Z","slug":"b2"}]}}`))
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

	comments := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)["comments"].([]any)
	if comments[0].(map[string]any)["id"] != "a1" {
		t.Fatalf("comments[0].id = %v, want %q", comments[0].(map[string]any)["id"], "a1")
	}
	if comments[1].(map[string]any)["id"] != "b2" {
		t.Fatalf("comments[1].id = %v, want %q", comments[1].(map[string]any)["id"], "b2")
	}
	if comments[2].(map[string]any)["id"] != "z2" {
		t.Fatalf("comments[2].id = %v, want %q", comments[2].(map[string]any)["id"], "z2")
	}
}

func TestTicketsCommentsReturnsEmptyListWithoutThread(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/rest/p/test/tickets/42" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","discussion_thread":{"_id":""}}}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "comments", "--project", "test", "--tracker", "tickets", "--ticket", "42"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	comments := result["comments"].([]any)
	if len(comments) != 0 {
		t.Fatalf("len(comments) = %d, want 0", len(comments))
	}
	thread := result["thread"].(map[string]any)
	if len(thread) != 0 {
		t.Fatalf("thread = %v, want empty object", thread)
	}
	if _, ok := result["pagination"]; ok {
		t.Fatalf("pagination = %v, want omitted for unpaginated comments", result["pagination"])
	}
}

func TestTicketsListPaginationFirstPageExposesNextOnly(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":1,"summary":"First","status":"open"},{"ticket_num":2,"summary":"Second","status":"open"}],"count":5,"page":0,"limit":2}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "list", "--project", "test", "--tracker", "tickets", "--limit", "2"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	pagination := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)["pagination"].(map[string]any)
	if pagination["has_previous"] != false {
		t.Fatalf("has_previous = %v, want false", pagination["has_previous"])
	}
	if pagination["has_next"] != true {
		t.Fatalf("has_next = %v, want true", pagination["has_next"])
	}
	if pagination["next_page"] != float64(1) {
		t.Fatalf("next_page = %v, want 1", pagination["next_page"])
	}
	if pagination["previous_page"] != nil {
		t.Fatalf("previous_page = %v, want nil", pagination["previous_page"])
	}
}

func TestTicketsSearchPaginationMiddlePageExposesBothDirections(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":3,"summary":"Third","status":"open"},{"ticket_num":4,"summary":"Fourth","status":"open"}],"count":6,"page":1,"limit":2}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "search", "--project", "test", "--tracker", "tickets", "--query", "status:open", "--page", "1", "--limit", "2"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	pagination := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)["pagination"].(map[string]any)
	if pagination["has_previous"] != true {
		t.Fatalf("has_previous = %v, want true", pagination["has_previous"])
	}
	if pagination["previous_page"] != float64(0) {
		t.Fatalf("previous_page = %v, want 0", pagination["previous_page"])
	}
	if pagination["has_next"] != true {
		t.Fatalf("has_next = %v, want true", pagination["has_next"])
	}
	if pagination["next_page"] != float64(2) {
		t.Fatalf("next_page = %v, want 2", pagination["next_page"])
	}
}

func TestTicketsListPaginationFinalPageExposesNoNextPage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":5,"summary":"Fifth","status":"open"}],"count":5,"page":2,"limit":2}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "list", "--project", "test", "--tracker", "tickets", "--page", "2", "--limit", "2"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	pagination := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)["pagination"].(map[string]any)
	if pagination["has_previous"] != true {
		t.Fatalf("has_previous = %v, want true", pagination["has_previous"])
	}
	if pagination["previous_page"] != float64(1) {
		t.Fatalf("previous_page = %v, want 1", pagination["previous_page"])
	}
	if pagination["has_next"] != false {
		t.Fatalf("has_next = %v, want false", pagination["has_next"])
	}
	if pagination["next_page"] != nil {
		t.Fatalf("next_page = %v, want nil", pagination["next_page"])
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
