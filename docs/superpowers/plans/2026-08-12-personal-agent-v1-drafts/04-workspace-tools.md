## Phase 4: Workspace tools

### Task 21: Rooted workspace filesystem primitives and tools

**Files:**
- Create: `internal/fsroot/root.go`
- Create: `internal/fsroot/root_test.go`
- Create: `internal/agent/tools/workspace.go`
- Create: `internal/agent/tools/workspace_test.go`

**Interfaces:**
- Consumes: `paths.ValidateRelPath(string) (string, error)`, `paths.MaxMarkdownBytes`, and a session workspace directory returned by `layout.SessionWorkspace`.
- Produces: `fsroot.Open(path string) (*fsroot.Root, error)`, `(*fsroot.Root).ReadFile(path string, max int64) ([]byte, error)`, `WriteFileAtomic(path string, body []byte, perm fs.FileMode) error`, `EditFileAtomic(path, old, replacement string) error`, `MkdirAll(path string, perm fs.FileMode) error`, `Tree() ([]fsroot.Entry, error)`, and `tools.NewWorkspace(root *fsroot.Root) *tools.Workspace` with `Execute(context.Context, name string, arguments json.RawMessage) (tools.Result, error)`.

- [ ] **Step 1: Write the failing rooted-filesystem tests**

```go
package fsroot_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/fsroot"
)

func TestRootReadWriteEditMkdirAndTree(t *testing.T) {
	dir := t.TempDir()
	r, err := fsroot.Open(dir)
	if err != nil { t.Fatal(err) }
	defer r.Close()

	if err := r.MkdirAll("drafts/chapter", 0o755); err != nil { t.Fatal(err) }
	if err := r.WriteFileAtomic("drafts/chapter/notes.txt", []byte("alpha beta"), 0o644); err != nil { t.Fatal(err) }
	if err := r.EditFileAtomic("drafts/chapter/notes.txt", "beta", "gamma"); err != nil { t.Fatal(err) }
	got, err := r.ReadFile("drafts/chapter/notes.txt", 1024)
	if err != nil { t.Fatal(err) }
	if string(got) != "alpha gamma" { t.Fatalf("got %q", got) }

	entries, err := r.Tree()
	if err != nil { t.Fatal(err) }
	if len(entries) != 3 || entries[2].Path != "drafts/chapter/notes.txt" || entries[2].Kind != "file" {
		t.Fatalf("unexpected tree: %#v", entries)
	}
}

func TestRootRejectsTraversalAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("secret"), 0o600); err != nil { t.Fatal(err) }
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err != nil { t.Fatal(err) }
	r, err := fsroot.Open(dir)
	if err != nil { t.Fatal(err) }
	defer r.Close()

	for _, name := range []string{"../secret", "/etc/passwd", "escape/secret"} {
		if _, err := r.ReadFile(name, 1024); err == nil {
			t.Fatalf("ReadFile(%q) unexpectedly succeeded", name)
		}
	}
	if err := r.WriteFileAtomic("escape/new", []byte("owned"), 0o600); err == nil {
		t.Fatal("write through symlink unexpectedly succeeded")
	}
	if _, err := os.Stat(filepath.Join(outside, "new")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside file changed: %v", err)
	}
}

func TestAtomicWriteKeepsOldFileWhenReplacementCannotCommit(t *testing.T) {
	dir := t.TempDir()
	r, err := fsroot.Open(dir)
	if err != nil { t.Fatal(err) }
	defer r.Close()
	if err := r.WriteFileAtomic("note.md", []byte("old"), 0o644); err != nil { t.Fatal(err) }
	if err := r.WriteFileAtomic("missing/note.md", []byte("new"), 0o644); err == nil { t.Fatal("expected missing parent error") }
	got, err := r.ReadFile("note.md", 10)
	if err != nil || string(got) != "old" { t.Fatalf("old content lost: %q, %v", got, err) }
}
```

```go
package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/agent/tools"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
)

func TestWorkspaceToolsAcceptMarkdownAndText(t *testing.T) {
	r, err := fsroot.Open(t.TempDir())
	if err != nil { t.Fatal(err) }
	defer r.Close()
	w := tools.NewWorkspace(r)

	cases := []struct{name string; args string}{
		{"mkdir", `{"path":"research"}`},
		{"write_file", `{"path":"research/raw.txt","content":"first draft"}`},
		{"edit_file", `{"path":"research/raw.txt","old":"first","replacement":"second"}`},
		{"read_file", `{"path":"research/raw.txt"}`},
	}
	for _, tc := range cases {
		if _, err := w.Execute(context.Background(), tc.name, json.RawMessage(tc.args)); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
	got, _ := r.ReadFile("research/raw.txt", 100)
	if string(got) != "second draft" { t.Fatalf("got %q", got) }
}

func TestWorkspaceToolsRejectUnknownFieldsTraversalAndUnknownTool(t *testing.T) {
	r, err := fsroot.Open(t.TempDir())
	if err != nil { t.Fatal(err) }
	defer r.Close()
	w := tools.NewWorkspace(r)
	for _, tc := range []struct{name, args string}{
		{"write_file", `{"path":"../x","content":"x"}`},
		{"read_file", `{"path":"x","extra":true}`},
		{"shell", `{"command":"id"}`},
	} {
		if _, err := w.Execute(context.Background(), tc.name, json.RawMessage(tc.args)); err == nil {
			t.Fatalf("%s unexpectedly accepted %s", tc.name, tc.args)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/fsroot ./internal/agent/tools -v`

Expected: FAIL because `internal/fsroot` and `internal/agent/tools` do not exist.

- [ ] **Step 3: Implement rooted access with Go 1.24 `os.Root` and strict tool decoding**

```go
// internal/fsroot/root.go
package fsroot

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	pathcontract "github.com/rigasyahrul/personal-agent/internal/paths"
)

type Root struct{ root *os.Root }
type Entry struct { Path string `json:"path"`; Kind string `json:"kind"`; Size int64 `json:"size,omitempty"` }

func Open(name string) (*Root, error) {
	r, err := os.OpenRoot(name)
	if err != nil { return nil, err }
	return &Root{root: r}, nil
}
func (r *Root) Close() error { return r.root.Close() }

func clean(name string) (string, error) { return pathcontract.ValidateRelPath(name) }

func (r *Root) rejectSymlinks(name string, allowMissingLeaf bool) error {
	parts := strings.Split(name, "/")
	for i := range parts {
		current := strings.Join(parts[:i+1], "/")
		info, err := r.root.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) && allowMissingLeaf && i == len(parts)-1 { return nil }
		if err != nil { return err }
		if info.Mode()&fs.ModeSymlink != 0 { return fmt.Errorf("symlink forbidden: %s", current) }
	}
	return nil
}

func (r *Root) ReadFile(name string, max int64) ([]byte, error) {
	name, err := clean(name); if err != nil { return nil, err }
	if err := r.rejectSymlinks(name, false); err != nil { return nil, err }
	f, err := r.root.Open(name); if err != nil { return nil, err }; defer f.Close()
	info, err := f.Stat(); if err != nil { return nil, err }
	if !info.Mode().IsRegular() { return nil, fmt.Errorf("not a regular file: %s", name) }
	b, err := io.ReadAll(io.LimitReader(f, max+1)); if err != nil { return nil, err }
	if int64(len(b)) > max { return nil, fmt.Errorf("file exceeds %d bytes", max) }
	return b, nil
}

func (r *Root) WriteFileAtomic(name string, body []byte, perm fs.FileMode) error {
	name, err := clean(name); if err != nil { return err }
	if err := r.rejectSymlinks(path.Dir(name), false); err != nil && path.Dir(name) != "." { return err }
	if _, err := r.root.Lstat(name); err == nil {
		if err := r.rejectSymlinks(name, false); err != nil { return err }
	} else if !errors.Is(err, fs.ErrNotExist) { return err }
	var tmp string
	var f *os.File
	for attempts := 0; attempts < 8; attempts++ {
		nonce := make([]byte, 8)
		if _, err := rand.Read(nonce); err != nil { return err }
		tmp = path.Join(path.Dir(name), ".pa-write-"+hex.EncodeToString(nonce))
		f, err = r.root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if err == nil { break }
		if !errors.Is(err, fs.ErrExist) { return err }
	}
	if f == nil { return errors.New("cannot allocate atomic-write temporary file") }
	ok := false
	defer func() { if !ok { _ = r.root.Remove(tmp) } }()
	if _, err = f.Write(body); err == nil { err = f.Sync() }
	if closeErr := f.Close(); err == nil { err = closeErr }
	if err != nil { return err }
	if err := r.root.Rename(tmp, name); err != nil { return err }
	ok = true
	return nil
}

func (r *Root) EditFileAtomic(name, old, replacement string) error {
	b, err := r.ReadFile(name, pathcontract.MaxMarkdownBytes); if err != nil { return err }
	if old == "" || bytes.Count(b, []byte(old)) != 1 { return errors.New("old text must occur exactly once") }
	return r.WriteFileAtomic(name, bytes.Replace(b, []byte(old), []byte(replacement), 1), 0o644)
}

func (r *Root) MkdirAll(name string, perm fs.FileMode) error {
	name, err := clean(name); if err != nil { return err }
	current := ""
	for _, component := range strings.Split(name, "/") {
		current = path.Join(current, component)
		info, statErr := r.root.Lstat(current)
		if errors.Is(statErr, fs.ErrNotExist) { if err := r.root.Mkdir(current, perm); err != nil { return err }; continue }
		if statErr != nil { return statErr }
		if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() { return fmt.Errorf("unsafe directory component: %s", current) }
	}
	return nil
}

func (r *Root) Tree() ([]Entry, error) {
	var out []Entry
	var walk func(string) error
	walk = func(dir string) error {
		f, err := r.root.Open(dir); if err != nil { return err }; defer f.Close()
		items, err := f.ReadDir(-1); if err != nil { return err }
		for _, item := range items {
			name := item.Name(); if strings.HasPrefix(name, ".pa-write-") { continue }
			p := name; if dir != "." { p = path.Join(dir, name) }
			info, err := r.root.Lstat(p); if err != nil { return err }
			if info.Mode()&fs.ModeSymlink != 0 { return fmt.Errorf("symlink forbidden: %s", p) }
			kind := "file"; if info.IsDir() { kind = "directory" } else if !info.Mode().IsRegular() { return fmt.Errorf("special file forbidden: %s", p) }
			out = append(out, Entry{Path:p, Kind:kind, Size:info.Size()})
			if info.IsDir() { if err := walk(p); err != nil { return err } }
		}
		return nil
	}
	if err := walk("."); err != nil { return nil, err }
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
```

```go
// internal/agent/tools/workspace.go
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/paths"
)

type Result struct { Content string `json:"content,omitempty"`; ChangedPath string `json:"changed_path,omitempty"` }
type Workspace struct{ root *fsroot.Root }
func NewWorkspace(root *fsroot.Root) *Workspace { return &Workspace{root: root} }

func decode(raw json.RawMessage, dst any) error {
	d := json.NewDecoder(bytes.NewReader(raw)); d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil { return err }
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) { return errors.New("exactly one JSON object required") }
	return nil
}

func (w *Workspace) Execute(ctx context.Context, name string, raw json.RawMessage) (Result, error) {
	if err := ctx.Err(); err != nil { return Result{}, err }
	switch name {
	case "read_file":
		var a struct{ Path string `json:"path"` }; if err := decode(raw, &a); err != nil { return Result{}, err }
		b, err := w.root.ReadFile(a.Path, paths.MaxMarkdownBytes); return Result{Content:string(b)}, err
	case "write_file":
		var a struct{ Path string `json:"path"`; Content string `json:"content"` }; if err := decode(raw, &a); err != nil { return Result{}, err }
		if len(a.Content) > paths.MaxMarkdownBytes { return Result{}, fmt.Errorf("content exceeds %d bytes", paths.MaxMarkdownBytes) }
		if err := w.root.WriteFileAtomic(a.Path, []byte(a.Content), 0o644); err != nil { return Result{}, err }; return Result{ChangedPath:a.Path}, nil
	case "edit_file":
		var a struct{ Path string `json:"path"`; Old string `json:"old"`; Replacement string `json:"replacement"` }; if err := decode(raw, &a); err != nil { return Result{}, err }
		if err := w.root.EditFileAtomic(a.Path, a.Old, a.Replacement); err != nil { return Result{}, err }; return Result{ChangedPath:a.Path}, nil
	case "mkdir":
		var a struct{ Path string `json:"path"` }; if err := decode(raw, &a); err != nil { return Result{}, err }
		if err := w.root.MkdirAll(a.Path, 0o755); err != nil { return Result{}, err }; return Result{ChangedPath:a.Path}, nil
	default:
		return Result{}, fmt.Errorf("workspace tool %q is not allowed", name)
	}
}
```

Keep workspace content type-neutral: `.md`, `.txt`, and other regular text files are valid in the workspace; the `.md` restriction applies later when promoting to `source/`. Random temporary siblings remain under the same rooted parent, so `os.Root.Rename` performs an atomic replacement on one filesystem.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/fsroot ./internal/agent/tools -v`

Expected: PASS, including traversal, absolute path, symlink, special-node, size, exact-edit, and atomic replacement cases.

- [ ] **Step 5: Commit**

```bash
git add internal/fsroot/root.go internal/fsroot/root_test.go internal/agent/tools/workspace.go internal/agent/tools/workspace_test.go
git commit -m "feat: add rooted workspace file tools"
```

### Task 22: Opt-in agent tool-call loop

**Files:**
- Modify: `internal/agent/provider.go`
- Modify: `internal/agent/runner.go`
- Modify: `internal/agent/runner_test.go`

**Interfaces:**
- Consumes: session `tool_grants.workspace_files`, `layout.SessionWorkspace`, `fsroot.Open`, `tools.Workspace.Execute`, ordered messages, and the existing idempotent `Runner.Start` run lifecycle.
- Produces: `ToolDefinition`, `ToolCall`, tool-capable `ChatRequest`/`ChatResponse`, and a bounded runner loop that exposes only `read_file`, `write_file`, `edit_file`, and `mkdir` when the persisted grant is true.

- [ ] **Step 1: Write failing grant and untrusted-argument tests**

```go
package agent_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/agent"
)

type scriptedProvider struct { requests []agent.ChatRequest; responses []agent.ChatResponse }
func (p *scriptedProvider) Chat(_ context.Context, req agent.ChatRequest) (agent.ChatResponse, error) {
	p.requests = append(p.requests, req); out := p.responses[0]; p.responses = p.responses[1:]; return out, nil
}

func TestRunnerDoesNotAdvertiseOrExecuteToolsWithoutGrant(t *testing.T) {
	p := &scriptedProvider{responses: []agent.ChatResponse{{Content:"plain answer"}}}
	r, db := newRunnerFixture(t, p, false)
	if _, err := r.Start(context.Background(), "session-1", "request-1", "write x"); err != nil { t.Fatal(err) }
	if len(p.requests) != 1 || len(p.requests[0].Tools) != 0 { t.Fatalf("tools leaked: %#v", p.requests) }
	assertNoToolMessages(t, db, "session-1")
}

func TestRunnerExecutesRootedToolsAndReportsChanges(t *testing.T) {
	p := &scriptedProvider{responses: []agent.ChatResponse{
		{ToolCalls: []agent.ToolCall{{ID:"call-1", Name:"write_file", Arguments:json.RawMessage(`{"path":"draft.txt","content":"hello"}`)}}},
		{Content:"saved"},
	}}
	r, db := newRunnerFixture(t, p, true)
	if _, err := r.Start(context.Background(), "session-1", "request-1", "save it"); err != nil { t.Fatal(err) }
	if len(p.requests[0].Tools) != 4 { t.Fatalf("got %d tools", len(p.requests[0].Tools)) }
	assertToolChange(t, db, "session-1", "call-1", "draft.txt")
}

func TestRunnerTreatsModelArgumentsAsUntrustedAndHasNoShell(t *testing.T) {
	for _, call := range []agent.ToolCall{
		{ID:"escape", Name:"read_file", Arguments:json.RawMessage(`{"path":"../../etc/passwd"}`)},
		{ID:"shell", Name:"shell", Arguments:json.RawMessage(`{"command":"id"}`)},
	} {
		p := &scriptedProvider{responses: []agent.ChatResponse{{ToolCalls:[]agent.ToolCall{call}}, {Content:"done"}}}
		r, db := newRunnerFixture(t, p, true)
		if _, err := r.Start(context.Background(), "session-1", call.ID, "try"); err != nil { t.Fatal(err) }
		assertToolError(t, db, "session-1", call.ID)
	}
}
```

The fixture creates a temporary database/session/workspace using the Phase 3 helpers, sets the persisted grant explicitly, and wires the real rooted workspace executor. The assertions query stored tool messages by tool-call ID and verify that a successful mutation stores `changed_path`, while rejected arguments store a safe error without exposing host paths.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/agent -run 'TestRunner(DoesNotAdvertise|ExecutesRooted|TreatsModel)' -v`

Expected: FAIL because provider DTOs have no tool calls and the runner performs only one provider call.

- [ ] **Step 3: Add provider DTOs and the bounded tool loop**

```go
// Add to internal/agent/provider.go.
type ToolDefinition struct {
	Name string `json:"name"`
	Description string `json:"description"`
	Parameters map[string]any `json:"parameters"`
}
type ToolCall struct { ID string `json:"id"`; Name string `json:"name"`; Arguments json.RawMessage `json:"arguments"` }
type ChatRequest struct { Messages []Message `json:"messages"`; Tools []ToolDefinition `json:"tools,omitempty"` }
type ChatResponse struct { Content string `json:"content,omitempty"`; ToolCalls []ToolCall `json:"tool_calls,omitempty"` }

var workspaceToolDefinitions = []ToolDefinition{
	{Name:"read_file", Description:"Read a regular workspace file", Parameters:objectSchema("path")},
	{Name:"write_file", Description:"Atomically replace a workspace file", Parameters:objectSchema("path", "content")},
	{Name:"edit_file", Description:"Replace one exact occurrence in a workspace file", Parameters:objectSchema("path", "old", "replacement")},
	{Name:"mkdir", Description:"Create workspace directories", Parameters:objectSchema("path")},
}

func objectSchema(required ...string) map[string]any {
	properties := map[string]any{}
	for _, name := range required { properties[name] = map[string]any{"type":"string"} }
	return map[string]any{"type":"object", "properties":properties, "required":required, "additionalProperties":false}
}
```

```go
// Use this loop inside the existing run execution path in internal/agent/runner.go.
const maxToolRounds = 8

func (r *Runner) completeTurn(ctx context.Context, session Session, messages []Message) (string, error) {
	req := ChatRequest{Messages:messages}
	var workspace *tools.Workspace
	var root *fsroot.Root
	if session.ToolGrants.WorkspaceFiles {
		opened, err := fsroot.Open(layout.SessionWorkspace(r.DataDir, session.Home, session.VaultID, session.ProjectID, session.ID))
		if err != nil { return "", err }
		root = opened; defer root.Close()
		workspace = tools.NewWorkspace(root)
		req.Tools = workspaceToolDefinitions
	}
	for round := 0; round < maxToolRounds; round++ {
		response, err := r.Provider.Chat(ctx, req)
		if err != nil { return "", err }
		if len(response.ToolCalls) == 0 { return response.Content, nil }
		if workspace == nil { return "", errors.New("provider returned a tool call without a workspace grant") }
		for _, call := range response.ToolCalls {
			result, toolErr := workspace.Execute(ctx, call.Name, call.Arguments)
			toolMessage := Message{Role:"tool", ToolCallID:call.ID}
			if toolErr != nil { toolMessage.Content = safeToolError(toolErr) } else {
				encoded, _ := json.Marshal(result); toolMessage.Content = string(encoded); toolMessage.ChangedPath = result.ChangedPath
			}
			if err := r.Messages.AppendTool(ctx, session.ID, toolMessage); err != nil { return "", err }
			req.Messages = append(req.Messages, toolMessage)
		}
	}
	return "", errors.New("tool round limit exceeded")
}

func safeToolError(err error) string {
	return `{"error":"workspace tool request rejected"}`
}
```

Adapt the existing OpenAI-compatible request/response conversion to preserve tool-call IDs and JSON arguments verbatim. Do not add a shell definition, command runner, generic process tool, or provider-selected root. The runner derives the root only from trusted persisted session IDs and scope fields. Execute calls sequentially, cap the loop at eight rounds, persist each result before the next provider request, and finish the existing run as `failed` when the round limit or provider call fails.

- [ ] **Step 4: Run agent tests to verify they pass**

Run: `go test ./internal/agent ./internal/agent/tools -v`

Expected: PASS; tools are absent when the grant is false, rooted tools work when true, hostile JSON is rejected, and `shell` remains unknown.

- [ ] **Step 5: Commit**

```bash
git add internal/agent/provider.go internal/agent/openai_compat.go internal/agent/runner.go internal/agent/runner_test.go
git commit -m "feat: run opt-in workspace tool calls"
```

### Task 23: Authenticated workspace tree and file-read API

**Files:**
- Modify: `internal/httpapi/chat_handlers.go`
- Modify: `internal/httpapi/server.go`
- Create: `internal/httpapi/workspace_test.go`

**Interfaces:**
- Consumes: authenticated session lookup, persisted workspace grant, `layout.SessionWorkspace`, `fsroot.Open`, `Root.Tree`, and `Root.ReadFile`.
- Produces: `GET /api/v1/sessions/{id}/workspace/tree` returning `{entries:[{path,kind,size}]}` and `GET /api/v1/sessions/{id}/workspace/file?path=` returning `{path,content}`.

- [ ] **Step 1: Write failing HTTP tests for grant checks and rooted reads**

```go
package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceTreeAndFile(t *testing.T) {
	fx := newAuthenticatedFixture(t)
	session := fx.createSession(true)
	workspace := fx.workspacePath(session.ID)
	if err := os.MkdirAll(filepath.Join(workspace, "drafts"), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(workspace, "drafts", "note.txt"), []byte("hello"), 0o644); err != nil { t.Fatal(err) }

	tree := fx.get("/api/v1/sessions/"+session.ID+"/workspace/tree")
	if tree.Code != http.StatusOK { t.Fatalf("tree: %d %s", tree.Code, tree.Body.String()) }
	var payload struct{ Entries []struct{ Path, Kind string; Size int64 } `json:"entries"` }
	if err := json.Unmarshal(tree.Body.Bytes(), &payload); err != nil { t.Fatal(err) }
	if len(payload.Entries) != 2 || payload.Entries[1].Path != "drafts/note.txt" { t.Fatalf("%#v", payload) }

	file := fx.get("/api/v1/sessions/"+session.ID+"/workspace/file?path="+url.QueryEscape("drafts/note.txt"))
	if file.Code != http.StatusOK || !json.Valid(file.Body.Bytes()) { t.Fatalf("file: %d %s", file.Code, file.Body.String()) }
	if !containsJSON(file.Body.Bytes(), `"content":"hello"`) { t.Fatalf("body %s", file.Body.String()) }
}

func TestWorkspaceEndpointsRequireGrantAndRejectEscape(t *testing.T) {
	fx := newAuthenticatedFixture(t)
	off := fx.createSession(false)
	if got := fx.get("/api/v1/sessions/"+off.ID+"/workspace/tree"); got.Code != http.StatusForbidden { t.Fatalf("got %d", got.Code) }
	on := fx.createSession(true)
	if got := fx.get("/api/v1/sessions/"+on.ID+"/workspace/file?path=..%2Fdb%2Fpersonal-agent.sqlite"); got.Code != http.StatusBadRequest { t.Fatalf("got %d", got.Code) }
}

func TestWorkspaceEndpointsRequireOwnerAuthentication(t *testing.T) {
	fx := newAuthenticatedFixture(t)
	session := fx.createSession(true)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+session.ID+"/workspace/tree", nil)
	res := httptest.NewRecorder(); fx.Handler.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized { t.Fatalf("got %d", res.Code) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/httpapi -run 'TestWorkspace' -v`

Expected: FAIL with 404 because workspace routes are not registered.

- [ ] **Step 3: Register read-only workspace handlers**

```go
// Add to internal/httpapi/chat_handlers.go.
func (s *Server) workspaceRoot(ctx context.Context, sessionID string) (*fsroot.Root, error) {
	session, err := s.Sessions.Get(ctx, sessionID)
	if err != nil { return nil, err }
	if !session.ToolGrants.WorkspaceFiles { return nil, errWorkspaceFilesDisabled }
	return fsroot.Open(layout.SessionWorkspace(s.DataDir, session.Home, session.VaultID, session.ProjectID, session.ID))
}

func (s *Server) workspaceTree(w http.ResponseWriter, r *http.Request) {
	root, err := s.workspaceRoot(r.Context(), r.PathValue("id"))
	if errors.Is(err, errWorkspaceFilesDisabled) { writeError(w, http.StatusForbidden, "workspace_files_disabled"); return }
	if err != nil { writeStoreError(w, err); return }
	defer root.Close()
	entries, err := root.Tree()
	if err != nil { writeError(w, http.StatusBadRequest, "unsafe_workspace_tree"); return }
	writeJSON(w, http.StatusOK, map[string]any{"entries":entries})
}

func (s *Server) workspaceFile(w http.ResponseWriter, r *http.Request) {
	name, err := paths.ValidateRelPath(r.URL.Query().Get("path"))
	if err != nil { writeError(w, http.StatusBadRequest, "invalid_path"); return }
	root, err := s.workspaceRoot(r.Context(), r.PathValue("id"))
	if errors.Is(err, errWorkspaceFilesDisabled) { writeError(w, http.StatusForbidden, "workspace_files_disabled"); return }
	if err != nil { writeStoreError(w, err); return }
	defer root.Close()
	body, err := root.ReadFile(name, paths.MaxMarkdownBytes)
	if err != nil { writeError(w, http.StatusBadRequest, "workspace_file_unreadable"); return }
	writeJSON(w, http.StatusOK, map[string]string{"path":name, "content":string(body)})
}
```

```go
// Register inside the authenticated /api/v1 mux in internal/httpapi/server.go.
mux.HandleFunc("GET /api/v1/sessions/{id}/workspace/tree", s.workspaceTree)
mux.HandleFunc("GET /api/v1/sessions/{id}/workspace/file", s.workspaceFile)
```

Return 404 for a session outside the owner-visible store query, 403 when `workspace_files` is off, and 400 for invalid paths, symlinks, non-regular files, invalid UTF-8 content, or files over 1 MiB. These are read-only GET routes and therefore require authentication but no CSRF token. Never expose an absolute workspace path in JSON or errors.

- [ ] **Step 4: Run HTTP tests to verify they pass**

Run: `go test ./internal/httpapi -run 'TestWorkspace' -v`

Expected: PASS for tree/file reads, authentication, grant enforcement, and traversal rejection.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/chat_handlers.go internal/httpapi/server.go internal/httpapi/workspace_test.go
git commit -m "feat: expose workspace tree and file reads"
```

### Task 24: Workspace file-tree panel and agent change indicators

**Files:**
- Modify: `web/js/api.js`
- Modify: `web/js/pages/sessions.js`
- Create: `web/js/components/workspace.mjs`
- Create: `web/js/components/workspace.test.mjs`
- Modify: `web/css/app.css`

**Interfaces:**
- Consumes: session detail `tool_grants.workspace_files`, message/tool result `changed_path`, `GET /api/v1/sessions/{id}/workspace/tree`, and `GET /api/v1/sessions/{id}/workspace/file?path=`.
- Produces: `workspaceTree(sessionID)`, `workspaceFile(sessionID, path)`, `renderWorkspacePanel`, and a tools-on-only panel that refreshes after agent file changes.

- [ ] **Step 1: Write failing DOM-independent component tests**

```js
// web/js/components/workspace.test.mjs
import test from 'node:test';
import assert from 'node:assert/strict';
import { changedPaths, workspaceRows } from './workspace.mjs';

test('workspaceRows escapes labels and marks changed files', () => {
  const html = workspaceRows(
    [{ path: 'drafts', kind: 'directory' }, { path: 'drafts/<note>.txt', kind: 'file', size: 5 }],
    new Set(['drafts/<note>.txt']),
  );
  assert.match(html, /drafts\/&lt;note&gt;\.txt/);
  assert.match(html, /data-path="drafts\/&lt;note&gt;\.txt"/);
  assert.match(html, /workspace-entry--changed/);
  assert.doesNotMatch(html, /<note>/);
});

test('changedPaths returns only agent tool mutations', () => {
  const messages = [
    { role: 'user', changed_path: 'ignored.txt' },
    { role: 'tool', changed_path: 'draft.md' },
    { role: 'tool', content: '{"changed_path":"notes/raw.txt"}' },
    { role: 'assistant', content: 'done' },
  ];
  assert.deepEqual([...changedPaths(messages)], ['draft.md', 'notes/raw.txt']);
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `node --test web/js/components/workspace.test.mjs`

Expected: FAIL with `ERR_MODULE_NOT_FOUND` for `workspace.mjs`.

- [ ] **Step 3: Implement the panel, API reads, and change refresh**

```js
// web/js/components/workspace.mjs
const escapeHTML = (value) => String(value)
  .replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
  .replaceAll('"', '&quot;').replaceAll("'", '&#39;');

export function changedPaths(messages) {
  const paths = new Set();
  for (const message of messages) {
    if (message.role !== 'tool') continue;
    let path = message.changed_path;
    if (!path && message.content) {
      try { path = JSON.parse(message.content).changed_path; } catch { path = ''; }
    }
    if (path) paths.add(path);
  }
  return paths;
}

export function workspaceRows(entries, changed) {
  return entries.map((entry) => {
    const path = escapeHTML(entry.path);
    const changedClass = changed.has(entry.path) ? ' workspace-entry--changed' : '';
    const disabled = entry.kind === 'directory' ? ' disabled' : '';
    return `<button class="workspace-entry workspace-entry--${entry.kind}${changedClass}" data-path="${path}"${disabled}>${path}</button>`;
  }).join('');
}

export async function renderWorkspacePanel({ container, sessionID, messages, api }) {
  const { entries } = await api.workspaceTree(sessionID);
  container.innerHTML = `<section class="workspace-panel"><h2>Workspace files</h2><div class="workspace-tree">${workspaceRows(entries, changedPaths(messages))}</div><pre class="workspace-preview" aria-live="polite">Select a file</pre></section>`;
  container.querySelectorAll('.workspace-entry--file').forEach((button) => button.addEventListener('click', async () => {
    const file = await api.workspaceFile(sessionID, button.dataset.path);
    container.querySelector('.workspace-preview').textContent = file.content;
  }));
}
```

```js
// Add to web/js/api.js.
export const workspaceTree = (sessionID) => apiJSON(`/api/v1/sessions/${encodeURIComponent(sessionID)}/workspace/tree`);
export const workspaceFile = (sessionID, path) => apiJSON(`/api/v1/sessions/${encodeURIComponent(sessionID)}/workspace/file?path=${encodeURIComponent(path)}`);
```

```js
// Integrate in web/js/pages/sessions.js after session detail and messages load.
import { renderWorkspacePanel } from '../components/workspace.mjs';
import * as api from '../api.js';

const workspaceMount = page.querySelector('[data-workspace-panel]');
if (session.tool_grants.workspace_files) {
  workspaceMount.hidden = false;
  await renderWorkspacePanel({container:workspaceMount, sessionID:session.id, messages, api});
} else {
  workspaceMount.hidden = true;
  workspaceMount.replaceChildren();
}

// Call this from the existing message/run refresh path after replacing `messages`.
if (session.tool_grants.workspace_files) {
  await renderWorkspacePanel({container:workspaceMount, sessionID:session.id, messages, api});
}
```

```css
/* Add to web/css/app.css. */
.session-layout { display: grid; grid-template-columns: minmax(0, 2fr) minmax(16rem, 1fr); gap: 1rem; }
.workspace-panel { border: 1px solid var(--border); border-radius: .5rem; padding: .75rem; }
.workspace-tree { display: grid; gap: .25rem; max-height: 18rem; overflow: auto; }
.workspace-entry { border: 0; background: transparent; color: inherit; cursor: pointer; padding: .35rem; text-align: left; }
.workspace-entry--directory { font-weight: 700; cursor: default; }
.workspace-entry--changed::after { content: " changed by agent"; color: var(--accent); font-size: .8em; }
.workspace-preview { max-height: 24rem; overflow: auto; white-space: pre-wrap; }
@media (max-width: 760px) { .session-layout { grid-template-columns: 1fr; } }
```

Add `<aside data-workspace-panel hidden></aside>` beside the existing chat column inside the session layout. The panel is absent when tools are off, uses `textContent` for file bodies, escapes every tree label/attribute, and refreshes whenever polling observes new tool messages so agent mutations become visible without a page reload. A failed read leaves chat usable and renders a concise panel error; it must not enable sending when the AI provider is unavailable.

- [ ] **Step 4: Run focused and full verification**

Run: `node --test web/js/components/workspace.test.mjs && go test ./internal/fsroot ./internal/agent/... ./internal/httpapi/...`

Expected: PASS; the panel helper escapes hostile names and identifies tool changes, while all workspace backend tests remain green.

- [ ] **Step 5: Commit**

```bash
git add web/js/api.js web/js/pages/sessions.js web/js/components/workspace.mjs web/js/components/workspace.test.mjs web/css/app.css
git commit -m "feat: show session workspace file changes"
```

### Phase self-check

- Spec §4: session workspaces remain freeform trees rooted at the scoped session directory.
- Spec §6: all paths use the shared logical POSIX contract; Go 1.24 `os.Root` operations, component checks, and atomic same-root replacement prevent traversal and symlink escape.
- Spec §8 and §9 F3: the tools-on session screen shows workspace files and agent changes; tools-off sessions remain messages-only, and file history stays readable independently of provider availability.
- Spec §11: model arguments are untrusted, grants default off and are checked from persisted session state, and no shell or host-root selection is exposed.
