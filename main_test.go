package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recorded struct {
	method, uri, body, auth, contentType string
}

// serve starts a test server that records the request and sends the given
// status and response body.
func serve(t *testing.T, status int, response string) *recorded {
	t.Helper()
	got := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*got = recorded{r.Method, r.RequestURI, string(b), r.Header.Get("Authorization"), r.Header.Get("Content-Type")}
		w.WriteHeader(status)
		io.WriteString(w, response)
	}))
	t.Cleanup(srv.Close)
	// The trailing slash checks that the client removes it.
	t.Setenv("NOTEDISCOVERY_URL", srv.URL+"/")
	t.Setenv("NOTEDISCOVERY_API_KEY", "test-key")
	return got
}

func TestCommands(t *testing.T) {
	file := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(file, []byte("from file"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		args       []string
		stdin      string
		response   string
		wantMethod string
		wantURI    string
		wantBody   string
		wantOut    string
	}{
		{name: "health", args: []string{"health"},
			response: `{"status":"healthy"}`, wantMethod: "GET", wantURI: "/health",
			wantOut: `{"status":"healthy"}` + "\n"},
		{name: "ls", args: []string{"ls"},
			response: `{"notes":[]}`, wantMethod: "GET", wantURI: "/api/notes",
			wantOut: `{"notes":[]}` + "\n"},
		{name: "ls with pagination", args: []string{"ls", "--limit", "20", "--offset", "40"},
			response: `{"notes":[]}`, wantMethod: "GET", wantURI: "/api/notes?limit=20&offset=40",
			wantOut: `{"notes":[]}` + "\n"},
		{name: "get", args: []string{"get", "my folder/a note.md"},
			response:   `{"path":"my folder/a note.md","content":"# Title\nbody","backlinks":[]}`,
			wantMethod: "GET", wantURI: "/api/notes/my%20folder/a%20note.md",
			wantOut: "# Title\nbody"},
		{name: "get json", args: []string{"get", "a.md", "--json"},
			response:   `{"path":"a.md","content":"x","backlinks":[]}`,
			wantMethod: "GET", wantURI: "/api/notes/a.md",
			wantOut: `{"path":"a.md","content":"x","backlinks":[]}` + "\n"},
		{name: "put from stdin", args: []string{"put", "dir/a.md"}, stdin: "# Hello\n",
			response: `{"success":true}`, wantMethod: "POST", wantURI: "/api/notes/dir/a.md",
			wantBody: `{"content":"# Hello\n"}`, wantOut: `{"success":true}` + "\n"},
		{name: "put from file", args: []string{"put", "a.md", file}, stdin: "ignored",
			response: `{"success":true}`, wantMethod: "POST", wantURI: "/api/notes/a.md",
			wantBody: `{"content":"from file"}`, wantOut: `{"success":true}` + "\n"},
		{name: "append", args: []string{"append", "a.md"}, stdin: "more",
			response: `{"success":true}`, wantMethod: "PATCH", wantURI: "/api/notes/a.md",
			wantBody: `{"add_timestamp":false,"content":"more"}`, wantOut: `{"success":true}` + "\n"},
		{name: "append with timestamp", args: []string{"append", "a.md", file, "--timestamp"},
			response: `{"success":true}`, wantMethod: "PATCH", wantURI: "/api/notes/a.md",
			wantBody: `{"add_timestamp":true,"content":"from file"}`, wantOut: `{"success":true}` + "\n"},
		{name: "mv", args: []string{"mv", "a.md", "dir/b c.md"},
			response: `{"success":true}`, wantMethod: "POST", wantURI: "/api/notes/move",
			wantBody: `{"newPath":"dir/b c.md","oldPath":"a.md"}`, wantOut: `{"success":true}` + "\n"},
		{name: "rm", args: []string{"rm", "dir/a.md"},
			response: `{"success":true}`, wantMethod: "DELETE", wantURI: "/api/notes/dir/a.md",
			wantOut: `{"success":true}` + "\n"},
		{name: "search", args: []string{"search", "road map", "--limit", "5"},
			response: `{"results":[]}`, wantMethod: "GET", wantURI: "/api/search?limit=5&q=road+map",
			wantOut: `{"results":[]}` + "\n"},
		{name: "tags", args: []string{"tags"},
			response: `{"tags":{}}`, wantMethod: "GET", wantURI: "/api/tags",
			wantOut: `{"tags":{}}` + "\n"},
		{name: "tags with tag", args: []string{"tags", "to do"},
			response: `{"tag":"to do"}`, wantMethod: "GET", wantURI: "/api/tags/to%20do",
			wantOut: `{"tag":"to do"}` + "\n"},
		{name: "mkdir", args: []string{"mkdir", "Projects/2025"},
			response: `{"success":true}`, wantMethod: "POST", wantURI: "/api/folders",
			wantBody: `{"path":"Projects/2025"}`, wantOut: `{"success":true}` + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serve(t, http.StatusOK, tt.response)
			var stdout, stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
			}
			if got.method != tt.wantMethod {
				t.Errorf("method = %q, want %q", got.method, tt.wantMethod)
			}
			if got.uri != tt.wantURI {
				t.Errorf("URI = %q, want %q", got.uri, tt.wantURI)
			}
			if got.body != tt.wantBody {
				t.Errorf("body = %q, want %q", got.body, tt.wantBody)
			}
			if got.auth != "Bearer test-key" {
				t.Errorf("Authorization = %q, want %q", got.auth, "Bearer test-key")
			}
			wantType := ""
			if tt.wantBody != "" {
				wantType = "application/json"
			}
			if got.contentType != wantType {
				t.Errorf("Content-Type = %q, want %q", got.contentType, wantType)
			}
			if stdout.String() != tt.wantOut {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantOut)
			}
		})
	}
}

func TestNoAPIKeySendsNoAuthHeader(t *testing.T) {
	got := serve(t, http.StatusOK, `{}`)
	t.Setenv("NOTEDISCOVERY_API_KEY", "")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"ls"}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if got.method != "GET" || got.auth != "" {
		t.Errorf("method = %q, Authorization = %q, want GET and no header", got.method, got.auth)
	}
}

func TestHTTPErrorPrintsDetail(t *testing.T) {
	tests := []struct{ name, response, want string }{
		{"string detail", `{"detail":"Note not found"}`, "ndcli: HTTP 404: Note not found\n"},
		{"list detail", `{"detail":[{"msg":"bad"}]}`, `ndcli: HTTP 404: [{"msg":"bad"}]` + "\n"},
		{"no detail", "gateway down\n", "ndcli: HTTP 404: gateway down\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serve(t, http.StatusNotFound, tt.response)
			var stdout, stderr bytes.Buffer
			code := run([]string{"get", "missing.md"}, nil, &stdout, &stderr)
			if code != 1 || stdout.Len() != 0 || stderr.String() != tt.want {
				t.Errorf("code = %d, stdout = %q, stderr = %q, want 1, empty, %q", code, stdout.String(), stderr.String(), tt.want)
			}
		})
	}
}

func TestVersionUsesNoNetwork(t *testing.T) {
	got := serve(t, http.StatusOK, `{}`)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if stdout.String() != "ndcli "+version+"\n" || got.method != "" {
		t.Errorf("stdout = %q, request method = %q, want the version and no request", stdout.String(), got.method)
	}
}

func TestUsageErrorsSendNoRequest(t *testing.T) {
	for _, args := range [][]string{{}, {"nope"}, {"get"}, {"get", ""}, {"mv", "a.md"}, {"rm", "a.md", "b.md"}, {"ls", "--bogus"}} {
		got := serve(t, http.StatusOK, `{}`)
		var stdout, stderr bytes.Buffer
		if code := run(args, strings.NewReader(""), &stdout, &stderr); code != 2 || got.method != "" {
			t.Errorf("args %q: code = %d, request method = %q, want 2 and no request", args, code, got.method)
		}
	}
}
