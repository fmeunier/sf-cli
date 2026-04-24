package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectToolsExecutesAPIRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/p/fuse-emulator" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/rest/p/fuse-emulator")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"shortname":"fuse-emulator","name":"Fuse","url":"https://sourceforge.net/p/fuse-emulator/","tools":[{"name":"tickets","mount_point":"bugs","mount_label":"Bugs","url":"https://sourceforge.net/p/fuse-emulator/bugs/","api_url":"https://sourceforge.net/rest/p/fuse-emulator/bugs/"},{"name":"tickets","mount_point":"feature-requests","mount_label":"Feature Requests","url":"https://sourceforge.net/p/fuse-emulator/feature-requests/","api_url":"https://sourceforge.net/rest/p/fuse-emulator/feature-requests/"}]}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	status := Run([]string{"--base-url", server.URL + "/rest", "project", "tools", "--project", "fuse-emulator"}, stdout)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; output=%s", status, stdout.String())
	}

	got := decodeEnvelope(t, stdout.Bytes())

	if got.Command != "project.tools" {
		t.Fatalf("command = %q, want %q", got.Command, "project.tools")
	}
	if !got.OK {
		t.Fatalf("ok = %v, want true", got.OK)
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}

	if got.Proposal == nil || got.Proposal.Action != "list_project_tools" {
		t.Fatalf("proposal = %#v, want action %q", got.Proposal, "list_project_tools")
	}
	result := got.Result.(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("len(result.tools) = %d, want 2", len(tools))
	}
	firstTool := tools[0].(map[string]any)
	if firstTool["mount_point"] != "bugs" {
		t.Fatalf("first tool mount_point = %v, want %q", firstTool["mount_point"], "bugs")
	}
}

func TestProjectToolsRequiresProject(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	status := Run([]string{"project", "tools"}, stdout)
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}

	got := decodeEnvelope(t, stdout.Bytes())
	if got.Error == nil || got.Error.Code != "invalid_arguments" {
		t.Fatalf("error = %#v, want code %q", got.Error, "invalid_arguments")
	}
	if got.Command != "project.tools" {
		t.Fatalf("command = %q, want %q", got.Command, "project.tools")
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %v, want empty", got.Warnings)
	}
}
