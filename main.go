// ndcli is a command-line client for the NoteDiscovery REST API.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// version is overridden at release build time via -ldflags "-X main.version=...".
var version = "dev"

const usage = `usage: ndcli <command> [arguments]

commands:
  health                               check the server
  ls [--limit N] [--offset N]          list notes
  get <path> [--json]                  print a note
  put <path> [file]                    create or replace a note
  append <path> [file] [--timestamp]   add to the end of a note
  mv <old> <new>                       move or rename a note
  rm <path>                            delete a note
  search <query> [--limit N]           search note contents
  tags [tag]                           list tags, or the notes with a tag
  mkdir <path>                         create a folder
  --version                            print the version

put and append read stdin if there is no file argument.

environment:
  NOTEDISCOVERY_URL       base URL (default http://localhost:8000)
  NOTEDISCOVERY_API_KEY   optional API key
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	cmd, rest := args[0], args[1:]
	fs := flag.NewFlagSet("ndcli "+cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		method = http.MethodGet
		path   string
		query  = url.Values{}
		body   map[string]any
		file   string
		asJSON = true
		pos    []string
		ok     bool
	)
	switch cmd {
	case "--version", "-version":
		fmt.Fprintln(stdout, "ndcli", version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return 0
	case "health":
		_, ok = parse(fs, rest, 0, 0)
		path = "/health"
	case "ls":
		limit := fs.Int("limit", 0, "max notes to return (0 = all)")
		offset := fs.Int("offset", 0, "notes to skip")
		_, ok = parse(fs, rest, 0, 0)
		path = "/api/notes"
		setInt(query, "limit", *limit)
		setInt(query, "offset", *offset)
	case "get":
		full := fs.Bool("json", false, "print the full server response")
		pos, ok = parse(fs, rest, 1, 1)
		path = "/api/notes/" + escape(pos[0])
		asJSON = *full
	case "put":
		pos, ok = parse(fs, rest, 1, 2)
		method, path = http.MethodPost, "/api/notes/"+escape(pos[0])
		body, file = map[string]any{}, pos[1]
	case "append":
		timestamp := fs.Bool("timestamp", false, "add a timestamp header before the content")
		pos, ok = parse(fs, rest, 1, 2)
		method, path = http.MethodPatch, "/api/notes/"+escape(pos[0])
		body, file = map[string]any{"add_timestamp": *timestamp}, pos[1]
	case "mv":
		pos, ok = parse(fs, rest, 2, 2)
		method, path = http.MethodPost, "/api/notes/move"
		body = map[string]any{"oldPath": pos[0], "newPath": pos[1]}
	case "rm":
		pos, ok = parse(fs, rest, 1, 1)
		method, path = http.MethodDelete, "/api/notes/"+escape(pos[0])
	case "search":
		limit := fs.Int("limit", 0, "max results to return (0 = all)")
		pos, ok = parse(fs, rest, 1, 1)
		path = "/api/search"
		query.Set("q", pos[0])
		setInt(query, "limit", *limit)
	case "tags":
		pos, ok = parse(fs, rest, 0, 1)
		path = "/api/tags"
		if pos[0] != "" {
			path += "/" + escape(pos[0])
		}
	case "mkdir":
		pos, ok = parse(fs, rest, 1, 1)
		method, path = http.MethodPost, "/api/folders"
		body = map[string]any{"path": pos[0]}
	default:
		fmt.Fprintf(stderr, "ndcli: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
	if !ok {
		return 2
	}

	if cmd == "put" || cmd == "append" {
		content, err := readContent(file, stdin)
		if err != nil {
			fmt.Fprintln(stderr, "ndcli:", err)
			return 1
		}
		body["content"] = content
	}

	base := strings.TrimRight(os.Getenv("NOTEDISCOVERY_URL"), "/")
	if base == "" {
		base = "http://localhost:8000"
	}
	u := base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	data, err := call(method, u, os.Getenv("NOTEDISCOVERY_API_KEY"), body)
	if err != nil {
		fmt.Fprintln(stderr, "ndcli:", err)
		return 1
	}

	if !asJSON {
		var note struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(data, &note); err != nil {
			fmt.Fprintln(stderr, "ndcli: the server response is not JSON:", err)
			return 1
		}
		// No newline is added, so "get" output is the exact note content.
		fmt.Fprint(stdout, note.Content)
		return 0
	}
	stdout.Write(data)
	if !bytes.HasSuffix(data, []byte("\n")) {
		fmt.Fprintln(stdout)
	}
	return 0
}

// parse collects the positional arguments. It accepts a flag before or after
// them, because the flag package stops at the first positional argument. The
// result always has maxArgs elements, and an absent optional argument is "".
func parse(fs *flag.FlagSet, args []string, minArgs, maxArgs int) ([]string, bool) {
	pos := make([]string, 0, maxArgs)
	padded := func() []string { return append(pos, make([]string, maxArgs)...)[:maxArgs] }
	for {
		if err := fs.Parse(args); err != nil {
			return padded(), false
		}
		if fs.NArg() == 0 {
			break
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(pos) < minArgs || len(pos) > maxArgs || slices.Contains(pos[:minArgs], "") {
		fmt.Fprint(fs.Output(), usage)
		return padded(), false
	}
	return padded(), true
}

func setInt(query url.Values, key string, n int) {
	if n > 0 {
		query.Set(key, strconv.Itoa(n))
	}
}

// escape escapes each segment of a note path and keeps the "/" separators.
func escape(p string) string {
	segments := strings.Split(strings.Trim(p, "/"), "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

func readContent(file string, stdin io.Reader) (string, error) {
	if file == "" {
		b, err := io.ReadAll(stdin)
		return string(b), err
	}
	b, err := os.ReadFile(file)
	return string(b), err
}

func call(method, u, key string, body map[string]any) ([]byte, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, u, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, detail(data))
	}
	return data, nil
}

// detail returns the "detail" field of an error response. The server sends a
// string for most errors and a list for a validation error.
func detail(data []byte) string {
	var e struct {
		Detail json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(data, &e) != nil || e.Detail == nil {
		return strings.TrimSpace(string(data))
	}
	var s string
	if json.Unmarshal(e.Detail, &s) == nil {
		return s
	}
	return string(e.Detail)
}
