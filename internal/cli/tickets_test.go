package cli

import (
	"bytes"
	"encoding/json"
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

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got["command"] != "tickets.list" {
		t.Fatalf("command = %v, want %q", got["command"], "tickets.list")
	}
	if got["ok"] != true {
		t.Fatalf("ok = %v, want true", got["ok"])
	}

	result := got["result"].(map[string]any)
	if result["count"] != float64(1) {
		t.Fatalf("result.count = %v, want 1", result["count"])
	}
	if result["limit"] != float64(10) {
		t.Fatalf("result.limit = %v, want 10", result["limit"])
	}
	proposal := got["proposal"].(map[string]any)
	if proposal["action"] != "list_tickets" {
		t.Fatalf("proposal.action = %v, want %q", proposal["action"], "list_tickets")
	}
	inputs := proposal["inputs"].(map[string]any)
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

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got["command"] != "tickets.search" {
		t.Fatalf("command = %v, want %q", got["command"], "tickets.search")
	}
	proposal := got["proposal"].(map[string]any)
	inputs := proposal["inputs"].(map[string]any)
	if inputs["query"] != "status:open" {
		t.Fatalf("proposal.inputs.query = %v, want %q", inputs["query"], "status:open")
	}
	result := got["result"].(map[string]any)
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

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got["command"] != "tickets.list" {
		t.Fatalf("command = %v, want %q", got["command"], "tickets.list")
	}
	errorValue := got["error"].(map[string]any)
	if errorValue["code"] != "invalid_arguments" {
		t.Fatalf("error.code = %v, want %q", errorValue["code"], "invalid_arguments")
	}
}

func TestTicketsListRejectsQueryFlagWithGuidance(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"tickets", "list", "--project", "fuse-emulator", "--tracker", "bugs", "--query", "status:open"}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got["command"] != "tickets.list" {
		t.Fatalf("command = %v, want %q", got["command"], "tickets.list")
	}
	proposal := got["proposal"].(map[string]any)
	target := proposal["target"].(map[string]any)
	if target["project"] != "" {
		t.Fatalf("proposal.target.project = %v, want empty string", target["project"])
	}
	errorValue := got["error"].(map[string]any)
	message := errorValue["message"].(string)
	if !bytes.Contains([]byte(message), []byte("tickets search")) {
		t.Fatalf("error.message = %q, want guidance mentioning tickets search", message)
	}
	if errorValue["code"] != "invalid_arguments" {
		t.Fatalf("error.code = %v, want %q", errorValue["code"], "invalid_arguments")
	}
}
