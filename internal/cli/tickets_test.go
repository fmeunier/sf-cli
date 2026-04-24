package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
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
	if pagination["count"] != float64(1) {
		t.Fatalf("pagination.count = %v, want 1", pagination["count"])
	}
	if pagination["has_more"] != false {
		t.Fatalf("pagination.has_more = %v, want false", pagination["has_more"])
	}
	if _, ok := pagination["next_cursor"]; ok {
		t.Fatalf("pagination.next_cursor = %v, want omitted", pagination["next_cursor"])
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
	if inputs["cursor"] != "" {
		t.Fatalf("proposal.inputs.cursor = %v, want empty string", inputs["cursor"])
	}
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
	if pagination["count"] != float64(1) {
		t.Fatalf("pagination.count = %v, want 1", pagination["count"])
	}
	if pagination["has_more"] != false {
		t.Fatalf("pagination.has_more = %v, want false", pagination["has_more"])
	}
	if _, ok := pagination["next_cursor"]; ok {
		t.Fatalf("pagination.next_cursor = %v, want omitted", pagination["next_cursor"])
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

func TestTicketsActivityReturnsMostRecentUpdates(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test/tickets":
			_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":1,"summary":"Older","status":"open","mod_date":"2026-04-24T01:00:00Z","discussion_thread":{"_id":"thread-1"}},{"ticket_num":2,"summary":"Newer comment","status":"open","mod_date":"2026-04-24T00:30:00Z","discussion_thread":{"_id":"thread-2"}}],"count":2,"page":0,"limit":25}`))
		case "/rest/p/test/tickets/1":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":1,"summary":"Older","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-1"},"mod_date":"2026-04-24T01:00:00Z"}}`))
		case "/rest/p/test/tickets/2":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":2,"summary":"Newer comment","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-2"},"mod_date":"2026-04-24T00:30:00Z"}}`))
		case "/rest/p/test/tickets/_discuss/thread/thread-1":
			_, _ = w.Write([]byte(`{"thread":{"_id":"thread-1","posts":[{"author":"alice","text":"older comment","is_meta":false,"timestamp":"2026-04-24T00:45:00Z","slug":"a1"}]}}`))
		case "/rest/p/test/tickets/_discuss/thread/thread-2":
			_, _ = w.Write([]byte(`{"thread":{"_id":"thread-2","posts":[{"author":"bob","text":"latest comment","is_meta":false,"timestamp":"2026-04-24T02:00:00Z","slug":"b1"}]}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "activity", "--project", "test", "--tracker", "tickets"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Command != "tickets.activity" {
		t.Fatalf("command = %q, want %q", got.Command, "tickets.activity")
	}
	if got.Proposal == nil || got.Proposal.Action != "list_ticket_activity" {
		t.Fatalf("proposal = %#v, want action %q", got.Proposal, "list_ticket_activity")
	}
	result := got.Result.(map[string]any)
	activities := result["tickets"].([]any)
	if len(activities) != 2 {
		t.Fatalf("len(tickets) = %d, want 2", len(activities))
	}
	first := activities[0].(map[string]any)
	if first["ticket_num"] != float64(2) {
		t.Fatalf("first.ticket_num = %v, want 2", first["ticket_num"])
	}
	if first["last_comment_at"] != "2026-04-24T02:00:00Z" {
		t.Fatalf("first.last_comment_at = %v, want latest timestamp", first["last_comment_at"])
	}
	if first["last_comment_author"] != "bob" {
		t.Fatalf("first.last_comment_author = %v, want %q", first["last_comment_author"], "bob")
	}
	if first["updated_at"] != "2026-04-24T02:00:00Z" {
		t.Fatalf("first.updated_at = %v, want latest activity timestamp", first["updated_at"])
	}
	pagination := result["pagination"].(map[string]any)
	if pagination["count"] != float64(2) {
		t.Fatalf("pagination.count = %v, want 2", pagination["count"])
	}
	if pagination["has_more"] != false {
		t.Fatalf("pagination.has_more = %v, want false", pagination["has_more"])
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

func TestTicketsListProjectsSelectedFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Fatalf("limit = %q, want %q", got, "10")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":1,"summary":"First","status":"open","assigned_to":"alice","labels":["triaged"],"created_date":"2026-04-24T00:00:00Z","mod_date":"2026-04-24T01:00:00Z"}],"count":1,"page":0,"limit":10}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "list", "--project", "test", "--tracker", "tickets", "--limit", "10", "--fields", "id,title,updated_at"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())
	result := got.Result.(map[string]any)
	ticket := result["tickets"].([]any)[0].(map[string]any)
	if len(ticket) != 3 {
		t.Fatalf("len(ticket) = %d, want 3", len(ticket))
	}
	if ticket["id"] != float64(1) {
		t.Fatalf("ticket.id = %v, want 1", ticket["id"])
	}
	if ticket["title"] != "First" {
		t.Fatalf("ticket.title = %v, want %q", ticket["title"], "First")
	}
	if ticket["updated_at"] != "2026-04-24T01:00:00Z" {
		t.Fatalf("ticket.updated_at = %v, want timestamp", ticket["updated_at"])
	}
	if _, ok := ticket["status"]; ok {
		t.Fatalf("ticket.status = %v, want omitted", ticket["status"])
	}
	inputs := got.Proposal.Inputs
	fields := inputs["fields"].([]any)
	if len(fields) != 3 {
		t.Fatalf("len(proposal.inputs.fields) = %d, want 3", len(fields))
	}
}

func TestTicketsGetProjectsSelectedFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","description":"Detailed ticket","status":"open","reported_by":"alice","assigned_to":"bob","labels":["triaged"],"private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42","subject":""},"custom_fields":{"_milestone":"unreleased"},"mod_date":"2026-04-24T01:00:00Z"}}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "get", "--project", "test", "--tracker", "tickets", "--ticket", "42", "--fields", "id,title,status,updated_at"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	ticket := result["ticket"].(map[string]any)
	if len(ticket) != 4 {
		t.Fatalf("len(ticket) = %d, want 4", len(ticket))
	}
	if ticket["id"] != float64(42) {
		t.Fatalf("ticket.id = %v, want 42", ticket["id"])
	}
	if ticket["title"] != "Answer" {
		t.Fatalf("ticket.title = %v, want %q", ticket["title"], "Answer")
	}
	if ticket["updated_at"] != "2026-04-24T01:00:00Z" {
		t.Fatalf("ticket.updated_at = %v, want timestamp", ticket["updated_at"])
	}
	if _, ok := ticket["description"]; ok {
		t.Fatalf("ticket.description = %v, want omitted", ticket["description"])
	}
}

func TestTicketsCommentsProjectsSelectedFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test/tickets/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","discussion_thread":{"_id":"thread-42"}}}`))
		case "/rest/p/test/tickets/_discuss/thread/thread-42":
			_, _ = w.Write([]byte(`{"thread":{"_id":"thread-42","subject":"Discussion","posts":[{"author":"alice","text":"first comment","is_meta":false,"timestamp":"2026-04-24T00:00:00Z","slug":"a1"}]}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "comments", "--project", "test", "--tracker", "tickets", "--ticket", "42", "--fields", "id,author,created_at"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	comment := result["comments"].([]any)[0].(map[string]any)
	if len(comment) != 3 {
		t.Fatalf("len(comment) = %d, want 3", len(comment))
	}
	if comment["id"] != "a1" {
		t.Fatalf("comment.id = %v, want %q", comment["id"], "a1")
	}
	if comment["author"] != "alice" {
		t.Fatalf("comment.author = %v, want %q", comment["author"], "alice")
	}
	if _, ok := comment["body"]; ok {
		t.Fatalf("comment.body = %v, want omitted", comment["body"])
	}
	if result["thread"].(map[string]any)["id"] != "thread-42" {
		t.Fatalf("thread.id = %v, want %q", result["thread"].(map[string]any)["id"], "thread-42")
	}
}

func TestTicketsActivityProjectsSelectedFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/p/test/tickets":
			_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":2,"summary":"Newer comment","status":"open","mod_date":"2026-04-24T00:30:00Z","discussion_thread":{"_id":"thread-2"}}],"count":1,"page":0,"limit":25}`))
		case "/rest/p/test/tickets/2":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":2,"summary":"Newer comment","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-2"},"mod_date":"2026-04-24T00:30:00Z"}}`))
		case "/rest/p/test/tickets/_discuss/thread/thread-2":
			_, _ = w.Write([]byte(`{"thread":{"_id":"thread-2","posts":[{"author":"bob","text":"latest comment","is_meta":false,"timestamp":"2026-04-24T02:00:00Z","slug":"b1"}]}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "activity", "--project", "test", "--tracker", "tickets", "--fields", "id,updated_at,last_comment_author"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	result := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)
	ticket := result["tickets"].([]any)[0].(map[string]any)
	if len(ticket) != 3 {
		t.Fatalf("len(ticket) = %d, want 3", len(ticket))
	}
	if ticket["id"] != float64(2) {
		t.Fatalf("ticket.id = %v, want 2", ticket["id"])
	}
	if ticket["last_comment_author"] != "bob" {
		t.Fatalf("ticket.last_comment_author = %v, want %q", ticket["last_comment_author"], "bob")
	}
	if _, ok := ticket["title"]; ok {
		t.Fatalf("ticket.title = %v, want omitted", ticket["title"])
	}
}

func TestTicketsSearchRejectsUnknownProjectedField(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"tickets", "search", "--project", "test", "--tracker", "tickets", "--query", "status:open", "--fields", "id,nope"}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Error == nil || got.Error.Code != "invalid_arguments" {
		t.Fatalf("error = %#v, want code %q", got.Error, "invalid_arguments")
	}
	if !bytes.Contains([]byte(got.Error.Message), []byte(`unsupported --fields value "nope"`)) {
		t.Fatalf("error.message = %q, want unsupported field guidance", got.Error.Message)
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

func TestTicketOutputsMatchCanonicalSchemaContract(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/rest/p/test/tickets":
			_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":42,"summary":"Answer","status":"open","reported_by":"alice","assigned_to":"bob","labels":["triaged"],"created_date":"2026-04-24T00:00:00Z","mod_date":"2026-04-24T01:00:00Z"}],"count":1,"page":0,"limit":25}`))
		case "/rest/p/test/tickets/search":
			_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":42,"summary":"Answer","status":"open","reported_by":"alice","assigned_to":"bob","labels":["triaged"],"created_date":"2026-04-24T00:00:00Z","mod_date":"2026-04-24T01:00:00Z"}],"count":1,"page":0,"limit":25}`))
		case "/rest/p/test/tickets/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","description":"Detailed ticket","status":"open","reported_by":"alice","assigned_to":"bob","labels":["triaged"],"private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42","subject":"Discussion"},"discussion_thread_url":"https://example.test/thread-42","custom_fields":{"_milestone":"unreleased"},"attachments":[{"name":"trace.txt"}],"related_artifacts":[{"type":"merge_request"}],"created_date":"2026-04-24T00:00:00Z","mod_date":"2026-04-24T01:00:00Z"}}`))
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

	assertTicketConformsCanonicalContract(t, decodeEnvelope(t, listOut.Bytes()).Result.(map[string]any)["tickets"].([]any)[0].(map[string]any), canonicalTicketListCommand)
	assertTicketConformsCanonicalContract(t, decodeEnvelope(t, searchOut.Bytes()).Result.(map[string]any)["tickets"].([]any)[0].(map[string]any), canonicalTicketSearchCommand)
	assertTicketConformsCanonicalContract(t, decodeEnvelope(t, getOut.Bytes()).Result.(map[string]any)["ticket"].(map[string]any), canonicalTicketGetCommand)

	getTicket := decodeEnvelope(t, getOut.Bytes()).Result.(map[string]any)["ticket"].(map[string]any)
	for _, field := range canonicalTicketFieldNames(canonicalTicketGetCommand) {
		if getTicket[field] == nil {
			t.Fatalf("ticket.%s = nil, want non-null canonical field", field)
		}
	}
}

func TestTicketOutputsOmitEmptyOptionalFieldsPerCanonicalContract(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/rest/p/test/tickets":
			_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":42,"summary":"Answer","status":"open"}],"count":1,"page":0,"limit":25}`))
		case "/rest/p/test/tickets/search":
			_, _ = w.Write([]byte(`{"tickets":[{"ticket_num":42,"summary":"Answer","status":"open"}],"count":1,"page":0,"limit":25}`))
		case "/rest/p/test/tickets/42":
			_, _ = w.Write([]byte(`{"ticket":{"ticket_num":42,"summary":"Answer","description":"","status":"open","private":false,"discussion_disabled":false,"discussion_thread":{"_id":"thread-42"}}}`))
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

	assertTicketConformsCanonicalContract(t, decodeEnvelope(t, listOut.Bytes()).Result.(map[string]any)["tickets"].([]any)[0].(map[string]any), canonicalTicketListCommand)
	assertTicketConformsCanonicalContract(t, decodeEnvelope(t, searchOut.Bytes()).Result.(map[string]any)["tickets"].([]any)[0].(map[string]any), canonicalTicketSearchCommand)
	assertTicketConformsCanonicalContract(t, decodeEnvelope(t, getOut.Bytes()).Result.(map[string]any)["ticket"].(map[string]any), canonicalTicketGetCommand)

	listTicket := decodeEnvelope(t, listOut.Bytes()).Result.(map[string]any)["tickets"].([]any)[0].(map[string]any)
	if _, ok := listTicket["description"]; ok {
		t.Fatalf("tickets list leaked detail-only field description: %v", listTicket["description"])
	}
	if _, ok := listTicket["discussion_thread"]; ok {
		t.Fatalf("tickets list leaked detail-only field discussion_thread: %v", listTicket["discussion_thread"])
	}
	getTicket := decodeEnvelope(t, getOut.Bytes()).Result.(map[string]any)["ticket"].(map[string]any)
	for _, field := range []string{"reported_by", "assigned_to", "labels", "created_date", "mod_date", "discussion_thread_url", "custom_fields", "attachments", "related_artifacts"} {
		if _, ok := getTicket[field]; ok {
			t.Fatalf("ticket.%s = %v, want omitted when optional value is empty", field, getTicket[field])
		}
	}
}

func assertTicketMatchesCanonicalContract(t *testing.T, ticket map[string]any, command string) {
	t.Helper()

	assertTicketConformsCanonicalContract(t, ticket, command)

	gotFields := sortedTicketFields(ticket)
	wantFields := sortedFieldNames(canonicalTicketFieldNames(command))
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Fatalf("%s fields = %v, want %v", command, gotFields, wantFields)
	}

}

func assertTicketConformsCanonicalContract(t *testing.T, ticket map[string]any, command string) {
	t.Helper()

	allowedFields := make(map[string]ticketSchemaField, len(canonicalTicketSchema))
	for _, field := range canonicalTicketSchema {
		if field.supports(command) {
			allowedFields[field.name] = field
		}
	}

	for field := range ticket {
		if _, ok := allowedFields[field]; !ok {
			t.Fatalf("%s emitted unexpected field %q", command, field)
		}
	}

	for _, field := range canonicalTicketSchema {
		if !field.supports(command) || field.nullable {
			continue
		}
		if field.required {
			if _, ok := ticket[field.name]; !ok {
				t.Fatalf("%s missing required field %q", command, field.name)
			}
		}
		if value, ok := ticket[field.name]; ok && value == nil {
			t.Fatalf("%s field %q = null, want non-null field", command, field.name)
		}
	}
}

func sortedTicketFields(ticket map[string]any) []string {
	fields := make([]string, 0, len(ticket))
	for field := range ticket {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

func sortedFieldNames(fields []string) []string {
	cloned := append([]string(nil), fields...)
	sort.Strings(cloned)
	return cloned
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
	if pagination["has_more"] != true {
		t.Fatalf("has_more = %v, want true", pagination["has_more"])
	}
	if pagination["next_cursor"] != "1" {
		t.Fatalf("next_cursor = %v, want %q", pagination["next_cursor"], "1")
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
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "search", "--project", "test", "--tracker", "tickets", "--query", "status:open", "--cursor", "1", "--limit", "2"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	pagination := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)["pagination"].(map[string]any)
	if pagination["has_more"] != true {
		t.Fatalf("has_more = %v, want true", pagination["has_more"])
	}
	if pagination["next_cursor"] != "2" {
		t.Fatalf("next_cursor = %v, want %q", pagination["next_cursor"], "2")
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
	status := Run([]string{"--base-url", server.URL + "/rest", "tickets", "list", "--project", "test", "--tracker", "tickets", "--cursor", "2", "--limit", "2"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	pagination := decodeEnvelope(t, stdout.Bytes()).Result.(map[string]any)["pagination"].(map[string]any)
	if pagination["has_more"] != false {
		t.Fatalf("has_more = %v, want false", pagination["has_more"])
	}
	if _, ok := pagination["next_cursor"]; ok {
		t.Fatalf("next_cursor = %v, want omitted", pagination["next_cursor"])
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
