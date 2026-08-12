## Phase 7: Hardening

### Task 37: Reject Hostile and Oversize Paths

**Files:**
- Modify: `internal/paths/paths_test.go`
- Modify: `internal/paths/paths.go`
- Modify: `internal/fsroot/root_test.go`
- Modify: `internal/fsroot/root.go`

**Interfaces:**
- Consumes: `paths.ValidateRelPath(string) (string, error)`, `paths.MaxPathBytes`, `paths.MaxDepth`, `paths.MaxComponentBytes`, `paths.MaxMarkdownBytes`, and the rooted open/read/write methods established on `fsroot.Root`.
- Produces: Uniform `*paths.PathError` rejection for traversal, absolute, reserved, malformed, and over-limit logical paths; rooted filesystem operations that reject symlink leaves and symlink ancestors.

- [ ] **Step 1: Write the failing path corpus and rooted symlink tests**

Append the following tests to `internal/paths/paths_test.go` (retaining its existing package declaration and imports, and adding `errors` and `strings`):

```go
func TestValidateRelPathRejectsHostileAndOversizeCorpus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		code string
	}{
		{"parent", "../secret.md", "invalid_path"},
		{"nested parent", "notes/../../secret.md", "invalid_path"},
		{"dot component", "notes/./secret.md", "invalid_path"},
		{"absolute unix", "/etc/passwd.md", "invalid_path"},
		{"absolute windows drive", `C:\\secret.md`, "invalid_path"},
		{"windows separator", `notes\\secret.md`, "invalid_path"},
		{"empty component", "notes//secret.md", "invalid_path"},
		{"reserved memory", "memory/secret.md", "reserved_path"},
		{"reserved soul", "soul/secret.md", "reserved_path"},
		{"reserved nested memory", "notes/memory/secret.md", "reserved_path"},
		{"control", "notes/secret\x00.md", "invalid_path"},
		{"too many components", strings.Repeat("a/", MaxDepth) + "x.md", "path_too_deep"},
		{"component too long", strings.Repeat("a", MaxComponentBytes+1) + ".md", "component_too_long"},
		{"path too long", strings.Repeat("abc/", 128) + "x.md", "path_too_long"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateRelPath(tc.path)
			var pe *PathError
			if !errors.As(err, &pe) {
				t.Fatalf("ValidateRelPath(%q) error = %v, want PathError", tc.path, err)
			}
			if pe.Code != tc.code {
				t.Fatalf("ValidateRelPath(%q) code = %q, want %q", tc.path, pe.Code, tc.code)
			}
		})
	}
}

func FuzzValidateRelPathNeverReturnsUnsafePath(f *testing.F) {
	for _, seed := range []string{"../x.md", "/x.md", "memory/x.md", "a/b.md", "a//b.md", "a\\b.md", "\x00.md"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		clean, err := ValidateRelPath(input)
		if err != nil {
			return
		}
		if clean == "" || strings.HasPrefix(clean, "/") || strings.Contains(clean, "\\") {
			t.Fatalf("accepted unsafe path %q as %q", input, clean)
		}
		parts := strings.Split(clean, "/")
		if len(parts) > MaxDepth || len(clean) > MaxPathBytes {
			t.Fatalf("accepted over-limit path %q", clean)
		}
		for _, part := range parts {
			if part == "" || part == "." || part == ".." || part == "memory" || part == "soul" || len(part) > MaxComponentBytes {
				t.Fatalf("accepted unsafe component %q in %q", part, clean)
			}
		}
	})
}

func TestValidateMarkdownBodyRejectsOversize(t *testing.T) {
	err := ValidateMarkdownBody([]byte(strings.Repeat("x", MaxMarkdownBytes+1)))
	var pe *PathError
	if !errors.As(err, &pe) || pe.Code != "body_too_large" {
		t.Fatalf("error = %v, want body_too_large PathError", err)
	}
	if err := ValidateMarkdownBody([]byte(strings.Repeat("x", MaxMarkdownBytes))); err != nil {
		t.Fatalf("exact limit rejected: %v", err)
	}
}
```

Append this test to `internal/fsroot/root_test.go`; it uses the `Open` constructor established for `fsroot.Root` in Phase 1:

```go
func TestRootRejectsSymlinkLeafAndAncestor(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(base, "leaf.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "linked")); err != nil {
		t.Fatal(err)
	}
	r, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, path := range []string{"leaf.md", "linked/secret.md"} {
		t.Run(path, func(t *testing.T) {
			if _, err := r.ReadFile(path); err == nil {
				t.Fatalf("ReadFile(%q) followed a symlink", path)
			}
			if err := r.WriteFileAtomic(path, []byte("changed"), 0o600); err == nil {
				t.Fatalf("WriteFileAtomic(%q) followed a symlink", path)
			}
		})
	}
	got, err := os.ReadFile(filepath.Join(outside, "secret.md"))
	if err != nil || string(got) != "secret" {
		t.Fatalf("outside file changed: body=%q err=%v", got, err)
	}
}
```

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./internal/paths ./internal/fsroot -run 'TestValidateRelPathRejectsHostileAndOversizeCorpus|TestValidateMarkdownBodyRejectsOversize|TestRootRejectsSymlinkLeafAndAncestor' -v`

Expected: FAIL because reserved path components, Windows forms, body size, or symlink traversal are not yet rejected consistently.

- [ ] **Step 3: Implement the minimum centralized validation and no-follow behavior**

In `internal/paths/paths.go`, make `ValidateRelPath` use the following checks before returning the original slash-separated path, and add `ValidateMarkdownBody`:

```go
func pathErr(code, message string) error { return &PathError{Code: code, Message: message} }

func ValidateRelPath(p string) (string, error) {
	if p == "" || strings.HasPrefix(p, "/") || filepath.IsAbs(p) || filepath.VolumeName(p) != "" || strings.Contains(p, "\\") {
		return "", pathErr("invalid_path", "path must be a relative POSIX path")
	}
	if len(p) > MaxPathBytes {
		return "", pathErr("path_too_long", "path exceeds 512 bytes")
	}
	parts := strings.Split(p, "/")
	if len(parts) > MaxDepth {
		return "", pathErr("path_too_deep", "path exceeds 16 components")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", pathErr("invalid_path", "path contains an unsafe component")
		}
		if len(part) > MaxComponentBytes {
			return "", pathErr("component_too_long", "path component exceeds 255 bytes")
		}
		if part == "memory" || part == "soul" {
			return "", pathErr("reserved_path", "memory and soul are reserved")
		}
		for _, r := range part {
			if r == 0 || unicode.IsControl(r) {
				return "", pathErr("invalid_path", "path contains a control character")
			}
		}
	}
	return p, nil
}

func ValidateMarkdownBody(body []byte) error {
	if len(body) > MaxMarkdownBytes {
		return pathErr("body_too_large", "markdown body exceeds 1 MiB")
	}
	return nil
}
```

Ensure `internal/fsroot/root.go` validates every logical path with `paths.ValidateRelPath`, opens ancestors through the Go 1.24 `os.Root` already established in Phase 1, and rejects any final node whose `FileInfo.Mode()&os.ModeSymlink != 0`. Atomic writes must create the temporary file in the validated parent, then perform the rename through the same root; never resolve with `filepath.EvalSymlinks` and never fall back to a host-absolute operation.

- [ ] **Step 4: Run unit tests and a short fuzz campaign**

Run: `go test ./internal/paths ./internal/fsroot -v && go test ./internal/paths -run '^$' -fuzz FuzzValidateRelPathNeverReturnsUnsafePath -fuzztime=5s`

Expected: PASS; the fuzz run completes without finding an accepted unsafe path.

- [ ] **Step 5: Commit**

```bash
git add internal/paths/paths.go internal/paths/paths_test.go internal/fsroot/root.go internal/fsroot/root_test.go
git commit -m "test: harden rooted path validation"
```

### Task 38: Serialize Multi-Tab Agent Starts

**Files:**
- Modify: `internal/store/runs.go`
- Modify: `internal/store/runs_test.go`
- Modify: `internal/agent/runner.go`
- Modify: `internal/agent/runner_test.go`
- Modify: `internal/httpapi/chat_handlers.go`

**Interfaces:**
- Consumes: `(*agent.Runner).Start(ctx, sessionID, requestKey, userMessage) (runID string, err error)`, AgentRun statuses, SQLite WAL, and the message mutation endpoint.
- Produces: `agent.ErrSessionBusy`, atomic single-active-run admission, and same-key idempotency returning the original run ID without appending a duplicate user message.

- [ ] **Step 1: Write concurrent different-key and same-key tests**

Append to `internal/agent/runner_test.go`, using the package's existing `newRunnerTestFixture` helper and blocking provider (the helper returns `runner`, `provider`, and `db`):

```go
func TestTwoTabsOneAgentRunDifferentKeys(t *testing.T) {
	fx := newRunnerTestFixture(t)
	fx.provider.Block()
	type result struct { id string; err error }
	start := make(chan struct{})
	out := make(chan result, 2)
	for _, key := range []string{"tab-a", "tab-b"} {
		key := key
		go func() {
			<-start
			id, err := fx.runner.Start(context.Background(), fx.sessionID, key, "explain this")
			out <- result{id, err}
		}()
	}
	close(start)
	a, b := <-out, <-out
	busy := 0
	started := 0
	for _, got := range []result{a, b} {
		switch {
		case got.err == nil:
			started++
		case errors.Is(got.err, ErrSessionBusy):
			busy++
		default:
			t.Fatalf("unexpected result: id=%q err=%v", got.id, got.err)
		}
	}
	if started != 1 || busy != 1 {
		t.Fatalf("started=%d busy=%d, want 1 and 1", started, busy)
	}
	fx.provider.Release()
}

func TestTwoTabsOneAgentRunSameKeyIsIdempotent(t *testing.T) {
	fx := newRunnerTestFixture(t)
	fx.provider.Block()
	start := make(chan struct{})
	ids := make(chan string, 2)
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			id, err := fx.runner.Start(context.Background(), fx.sessionID, "same-key", "explain this")
			ids <- id
			errs <- err
		}()
	}
	close(start)
	id1, id2 := <-ids, <-ids
	if err1, err2 := <-errs, <-errs; err1 != nil || err2 != nil {
		t.Fatalf("errors = %v, %v", err1, err2)
	}
	if id1 == "" || id1 != id2 {
		t.Fatalf("run IDs = %q, %q", id1, id2)
	}
	var runs, userMessages int
	if err := fx.db.QueryRow(`SELECT count(*) FROM agent_runs WHERE session_id=?`, fx.sessionID).Scan(&runs); err != nil { t.Fatal(err) }
	if err := fx.db.QueryRow(`SELECT count(*) FROM messages WHERE session_id=? AND role='user'`, fx.sessionID).Scan(&userMessages); err != nil { t.Fatal(err) }
	if runs != 1 || userMessages != 1 {
		t.Fatalf("runs=%d user_messages=%d, want 1 and 1", runs, userMessages)
	}
	fx.provider.Release()
}
```

- [ ] **Step 2: Run tests to verify the admission race**

Run: `go test ./internal/agent -run 'TestTwoTabsOneAgentRunDifferentKeys|TestTwoTabsOneAgentRunSameKeyIsIdempotent' -count=20 -v`

Expected: FAIL intermittently or consistently with two runs, duplicate messages, unequal same-key IDs, or a SQLite uniqueness error leaking from `Start`.

- [ ] **Step 3: Make run admission one SQLite transaction**

In `internal/agent/runner.go`, export the sentinel and map store outcomes without starting a second provider call:

```go
var ErrSessionBusy = errors.New("session has an active agent run")

func (r *Runner) Start(ctx context.Context, sessionID, requestKey, userMessage string) (string, error) {
	admission, err := r.Runs.Admit(ctx, sessionID, requestKey, userMessage, r.Clock.Now())
	if err != nil {
		if errors.Is(err, store.ErrRunBusy) { return "", ErrSessionBusy }
		return "", err
	}
	if admission.Existing {
		return admission.RunID, nil
	}
	go r.execute(admission.RunID)
	return admission.RunID, nil
}
```

Implement `Runs.Admit` in `internal/store/runs.go` with `BEGIN IMMEDIATE` semantics on a dedicated `*sql.Conn`: first select `(session_id, request_key)` and return it as `Existing`; then reject a terminal session; then query any `queued`/`running` run and return `ErrRunBusy`; finally insert the user message and queued run in that same transaction. Preserve the existing partial unique index for one non-terminal run as defense in depth. If a unique conflict wins a race, re-read the same key first (idempotent success), otherwise return `ErrRunBusy`. In `chat_handlers.go`, map `agent.ErrSessionBusy` to HTTP `409` with code `session_busy`.

- [ ] **Step 4: Stress the focused tests**

Run: `go test ./internal/store ./internal/agent ./internal/httpapi -run 'TestTwoTabsOneAgentRun|TestStartRunBusyReturns409' -race -count=20`

Expected: PASS with exactly one run for every iteration; same-key starts return the same ID and different keys yield one success plus one busy response.

- [ ] **Step 5: Commit**

```bash
git add internal/store/runs.go internal/store/runs_test.go internal/agent/runner.go internal/agent/runner_test.go internal/httpapi/chat_handlers.go
git commit -m "fix: serialize multi-tab agent starts"
```

### Task 39: Serialize Session Delete Against Promotion

**Files:**
- Modify: `internal/store/sessions.go`
- Modify: `internal/httpapi/session_handlers.go`
- Modify: `internal/publish/machine.go`
- Modify: `internal/publish/machine_test.go`

**Interfaces:**
- Consumes: publication statuses, session status `active | terminal`, `publish.Machine.Run`, and session workspace deletion.
- Produces: A shared per-session mutation lock used by promote and delete; deletion either waits for a committed promotion or makes promotion fail before publication, never leaving a `ready` note whose file is absent.

- [ ] **Step 1: Write the delete-during-promote race test**

Append this deterministic hook-based test to `internal/publish/machine_test.go`; expose the test hook only as an optional function field on `Machine`:

```go
func TestSessionDeleteDuringPromoteHasNoOrphanReadyNote(t *testing.T) {
	fx := newMachineFixture(t)
	reachedFreeze := make(chan struct{})
	continuePromote := make(chan struct{})
	fx.machine.AfterTransition = func(status string) {
		if status == "frozen" {
			close(reachedFreeze)
			<-continuePromote
		}
	}
	promoteDone := make(chan error, 1)
	go func() {
		_, _, err := fx.machine.Run(context.Background(), fx.promoteInput("race.md"))
		promoteDone <- err
	}()
	<-reachedFreeze
	deleteDone := make(chan error, 1)
	go func() { deleteDone <- fx.sessions.Delete(context.Background(), fx.sessionID) }()
	close(continuePromote)
	promoteErr := <-promoteDone
	deleteErr := <-deleteDone
	if promoteErr != nil && deleteErr != nil {
		t.Fatalf("both operations failed: promote=%v delete=%v", promoteErr, deleteErr)
	}
	rows, err := fx.db.Query(`SELECT id, rel_path FROM notes WHERE status='ready'`)
	if err != nil { t.Fatal(err) }
	defer rows.Close()
	for rows.Next() {
		var id, rel string
		if err := rows.Scan(&id, &rel); err != nil { t.Fatal(err) }
		if _, err := os.Stat(filepath.Join(fx.projectRoot, "source", filepath.FromSlash(rel))); err != nil {
			t.Fatalf("orphan ready note %s at %s: %v", id, rel, err)
		}
	}
	if err := rows.Err(); err != nil { t.Fatal(err) }
	if _, err := os.Stat(fx.workspace); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("workspace still exists or stat failed: %v", err)
	}
}
```

- [ ] **Step 2: Run the race test to verify it fails**

Run: `go test ./internal/publish -run TestSessionDeleteDuringPromoteHasNoOrphanReadyNote -race -count=20 -v`

Expected: FAIL because delete can remove the workspace between promotion freeze/read transitions or because delete and promote do not share serialization.

- [ ] **Step 3: Add a shared keyed session lock and enforce terminal checks**

Add the following focused lock type beside the existing session store and inject one shared instance into both session deletion and `publish.Machine` during app wiring:

```go
type SessionLocks struct {
	mu sync.Mutex
	byID map[string]*sessionLock
}
type sessionLock struct { mu sync.Mutex; refs int }

func NewSessionLocks() *SessionLocks { return &SessionLocks{byID: make(map[string]*sessionLock)} }

func (s *SessionLocks) Lock(id string) func() {
	s.mu.Lock()
	l := s.byID[id]
	if l == nil { l = &sessionLock{}; s.byID[id] = l }
	l.refs++
	s.mu.Unlock()
	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		s.mu.Lock()
		l.refs--
		if l.refs == 0 { delete(s.byID, id) }
		s.mu.Unlock()
	}
}
```

At the beginning of promote `Run`, acquire `unlock := m.SessionLocks.Lock(in.SessionID)` and defer `unlock()`. Under that lock, verify the session is active before accepting/freezing. Session deletion acquires the same lock, transactionally marks the session terminal (which blocks runs/tools/new promotes), removes only its workspace with the rooted helper, and retains the session tombstone, source notes, operations, review items, and review events. If filesystem removal fails, return a clean error and leave the terminal tombstone so no new mutation can begin. The `AfterTransition` field is nil in production and called immediately after a committed transition solely to make crash/race tests deterministic.

- [ ] **Step 4: Run publication, session, and race tests**

Run: `go test ./internal/publish ./internal/store ./internal/httpapi -run 'TestSessionDelete|TestPromote|TestDeleteSession' -race -count=10`

Expected: PASS; repeated runs never report a ready note without its source file, and deleted sessions retain source/review history while their workspace is absent.

- [ ] **Step 5: Commit**

```bash
git add internal/store/sessions.go internal/httpapi/session_handlers.go internal/publish/machine.go internal/publish/machine_test.go internal/app/app.go
git commit -m "fix: serialize session deletion and promotion"
```

### Task 40: Enforce Authentication, CSRF, and One-Time Bootstrap

**Files:**
- Modify: `internal/httpapi/middleware.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/auth_handlers.go`
- Modify: `internal/httpapi/auth_test.go`
- Modify: `internal/auth/bootstrap.go`
- Modify: `internal/auth/bootstrap_test.go`

**Interfaces:**
- Consumes: `pa_session`, `pa_csrf`, owner bootstrap storage, `BOOTSTRAP_TOKEN`, and the v1 ServeMux mutation routes.
- Produces: HTTP `401` for every unauthenticated domain mutation, `403` for authenticated CSRF mismatch, and `409 owner_exists` for any bootstrap attempt after owner creation.

- [ ] **Step 1: Write table-driven security boundary tests**

Add to `internal/httpapi/auth_test.go`, using the existing HTTP fixture's `server`, `login(t) (sessionCookie, csrfCookie)`, and JSON request helper:

```go
func TestUnauthenticatedMutationsReturn401(t *testing.T) {
	fx := newHTTPFixture(t)
	tests := []struct{ method, path, body string }{
		{"PUT", "/api/v1/settings", `{"timezone":"UTC"}`},
		{"POST", "/api/v1/projects", `{"name":"x"}`},
		{"POST", "/api/v1/projects/p1/folders", `{"path":"x"}`},
		{"POST", "/api/v1/projects/p1/direct-notes", `{"path":"x.md","body":"x"}`},
		{"POST", "/api/v1/projects/p1/sessions", `{"title":"x"}`},
		{"DELETE", "/api/v1/sessions/s1", ``},
		{"POST", "/api/v1/sessions/s1/messages", `{"message":"x"}`},
		{"POST", "/api/v1/sessions/s1/promote", `{"path":"x.md"}`},
		{"POST", "/api/v1/review/items/r1/rate", `{"rating":"good"}`},
		{"POST", "/api/v1/review/items/r1/suspend", `{}`},
		{"POST", "/api/v1/review/pending/p1/retry", `{}`},
		{"POST", "/api/v1/backups", `{}`},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			res := fx.request(tc.method, tc.path, tc.body, nil)
			if res.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
		})
	}
}

func TestCSRFFailureReturns403(t *testing.T) {
	fx := newHTTPFixture(t)
	session, csrf := fx.login(t)
	res := fx.request("POST", "/api/v1/projects", `{"name":"x"}`, []*http.Cookie{session, csrf})
	if res.Code != http.StatusForbidden {
		t.Fatalf("missing header status=%d body=%s", res.Code, res.Body.String())
	}
	reqCookies := []*http.Cookie{session, {Name: "pa_csrf", Value: "cookie-value"}}
	res = fx.requestWithHeaders("POST", "/api/v1/projects", `{"name":"x"}`, reqCookies, map[string]string{"X-CSRF-Token": "header-value"})
	if res.Code != http.StatusForbidden {
		t.Fatalf("mismatch status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestBootstrapTakeoverBlockedWhenOwnerExists(t *testing.T) {
	fx := newHTTPFixture(t)
	first := fx.requestWithHeaders("POST", "/api/v1/setup/bootstrap", `{"password":"first secure password"}`, nil, map[string]string{"Authorization": "Bearer " + fx.bootstrapToken})
	if first.Code != http.StatusCreated { t.Fatalf("first status=%d body=%s", first.Code, first.Body.String()) }
	second := fx.requestWithHeaders("POST", "/api/v1/setup/bootstrap", `{"password":"attacker password"}`, nil, map[string]string{"Authorization": "Bearer " + fx.bootstrapToken})
	if second.Code != http.StatusConflict { t.Fatalf("second status=%d body=%s", second.Code, second.Body.String()) }
	if fx.loginPassword("first secure password").Code != http.StatusOK { t.Fatal("original owner password no longer works") }
	if fx.loginPassword("attacker password").Code == http.StatusOK { t.Fatal("takeover password was accepted") }
}
```

- [ ] **Step 2: Run tests to verify security gaps fail**

Run: `go test ./internal/httpapi ./internal/auth -run 'TestUnauthenticatedMutationsReturn401|TestCSRFFailureReturns403|TestBootstrapTakeoverBlockedWhenOwnerExists' -v`

Expected: FAIL on any mutation omitted from the secured route group, CSRF mismatch not mapped to 403, or a second bootstrap that changes owner credentials.

- [ ] **Step 3: Centralize route policy and make bootstrap compare-and-set**

In `internal/httpapi/middleware.go`, compose mutation middleware in this order so unauthenticated requests receive 401 before CSRF evaluation:

```go
func (s *Server) securedMutation(next http.Handler) http.Handler {
	return s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("pa_csrf")
		if err != nil || cookie.Value == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(r.Header.Get("X-CSRF-Token"))) != 1 {
			writeError(w, http.StatusForbidden, "csrf_failed", "CSRF token missing or invalid")
			return
		}
		next.ServeHTTP(w, r)
	}))
}
```

Register every mutation listed in Step 1 through `securedMutation`; bootstrap and login remain outside it, logout is secured because it mutates the session. In `internal/auth/bootstrap.go`, start a transaction, query owner count, return `ErrOwnerExists` before checking or writing credentials when count is nonzero, compare the supplied bearer token to the configured token in constant time, insert the owner with a unique singleton key, and commit. Map `ErrOwnerExists` to `409 owner_exists`; never expose whether a supplied bootstrap token was correct after an owner exists.

- [ ] **Step 4: Run all auth and HTTP API tests**

Run: `go test ./internal/auth ./internal/httpapi -race -v`

Expected: PASS; all listed unauthenticated mutations are 401, valid-session CSRF failures are 403, and bootstrap credentials cannot be replaced.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/middleware.go internal/httpapi/server.go internal/httpapi/auth_handlers.go internal/httpapi/auth_test.go internal/auth/bootstrap.go internal/auth/bootstrap_test.go
git commit -m "fix: enforce auth csrf and bootstrap boundaries"
```

### Task 41: Add the Spec Acceptance Integration Suite

**Files:**
- Create: `internal/acceptance/acceptance_test.go`
- Create: `internal/acceptance/harness_test.go`
- Modify: `internal/app/app.go`

**Interfaces:**
- Consumes: the fully wired application, temporary `PA_DATA_DIR`, fake provider/clock/S3 sink, publication transition hooks, HTTP endpoints, and backup restore helper from Phases 1–6.
- Produces: Eleven named integration tests, one per spec §13 criterion, run by `go test ./internal/acceptance -v`; `app.Dependencies` permits deterministic provider, clock, publication hook, and backup sink injection without production-only branches.

- [ ] **Step 1: Create a failing acceptance manifest with exact test names**

Create `internal/acceptance/acceptance_test.go` with this explicit manifest and eleven tests. Each named function must perform the indicated observable assertions through the harness; component tests alone do not satisfy this task.

```go
package acceptance

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestAcceptance01PromoteRetrySameKeyOneNoteOneReviewSet(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-01")
	h.workspaceFile(s, "lesson.md", "# Lesson")
	a := h.promote(s, "lesson.md", "notes/lesson.md", "whole", "promote-key")
	b := h.promote(s, "lesson.md", "notes/lesson.md", "whole", "promote-key")
	if a.NoteID != b.NoteID { t.Fatalf("note IDs differ: %s %s", a.NoteID, b.NoteID) }
	h.assertCount("notes", "id=?", 1, a.NoteID)
	h.assertCount("review_items", "note_id=?", 1, a.NoteID)
}

func TestAcceptance02CrashAfterFSPublishRecoveryConverges(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-02")
	h.workspaceFile(s, "crash.md", "# Durable")
	h.crashAfter("published_fs")
	op := h.promoteExpectInterrupted(s, "crash.md", "notes/crash.md", "whole", "crash-key")
	h.restart()
	h.recover()
	h.assertOperationStatus(op.ID, "completed")
	h.assertReadyNoteFile(op.NoteID, "# Durable")
}

func TestAcceptance03BiteFailureRetryNoDuplicateNote(t *testing.T) {
	h := newHarness(t)
	h.bites.failNext(errors.New("generator unavailable"))
	n := h.directNote("notes/bites.md", "# Bites", "bites", "bite-key")
	h.runBiteWorker()
	h.assertPendingStatus(n.PendingID, "failed")
	h.retryPending(n.PendingID)
	h.runBiteWorker()
	h.assertCount("notes", "id=?", 1, n.NoteID)
	h.assertCount("review_items", "note_id=?", h.bites.generatedCount(), n.NoteID)
}

func TestAcceptance04InvalidSessionScopeRejectedAPIAndDB(t *testing.T) {
	h := newHarness(t)
	res := h.rawJSON("POST", "/api/v1/projects/"+h.projectID+"/sessions", `{"home":"vault","vault_id":"wrong","title":"bad","provider":"openai","model_id":"test"}`, true)
	if res.Code != http.StatusBadRequest { t.Fatalf("API status=%d body=%s", res.Code, res.Body.String()) }
	if _, err := h.db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,title,provider,model_id,status,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, "bad", "vault", "wrong", h.projectID, "bad", "openai", "test", "active", h.now()); err == nil {
		t.Fatal("database accepted mismatched vault/project scope")
	}
}

func TestAcceptance05TraversalAndSymlinkEscapeRejected(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-05")
	for _, p := range []string{"../secret.md", "/tmp/secret.md"} {
		if got := h.workspaceWrite(s, p, "stolen"); got.Code < 400 { t.Fatalf("accepted %q", p) }
	}
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil { t.Fatal(err) }
	if err := h.workspaceSymlink(s, "escape.md", outside); err != nil { t.Skipf("symlink unavailable: %v", err) }
	if got := h.workspaceRead(s, "escape.md"); got.Code < 400 { t.Fatal("followed symlink escape") }
}

func TestAcceptance06DestinationExists409NoOverwrite(t *testing.T) {
	h := newHarness(t)
	h.directNote("notes/existing.md", "original", "none", "first-key")
	res := h.directNoteResponse("notes/existing.md", "replacement", "none", "second-key")
	if res.Code != http.StatusConflict { t.Fatalf("status=%d body=%s", res.Code, res.Body.String()) }
	if got := h.sourceBody("notes/existing.md"); got != "original" { t.Fatalf("body=%q", got) }
}

func TestAcceptance07SessionDeleteRemovesWorkspaceOnly(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-07")
	h.workspaceFile(s, "keep.md", "keep")
	n := h.promote(s, "keep.md", "notes/keep.md", "whole", "keep-key")
	h.deleteSession(s)
	h.assertWorkspaceAbsent(s)
	h.assertReadyNoteFile(n.NoteID, "keep")
	h.assertCount("review_items", "note_id=?", 1, n.NoteID)
}

func TestAcceptance08RatingRetrySameKeyOneEvent(t *testing.T) {
	h := newHarness(t)
	n := h.directNote("notes/rate.md", "rate", "whole", "rate-note-key")
	h.rate(n.ReviewItemID, "good", "rating-key")
	h.rate(n.ReviewItemID, "good", "rating-key")
	h.assertCount("review_events", "review_item_id=? AND request_key=?", 1, n.ReviewItemID, "rating-key")
}

func TestAcceptance09TwoTabsOneAgentRun(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-09")
	h.provider.block()
	a, b := h.parallelMessages(s, "tab-a", "tab-b")
	if !((a.Code == http.StatusAccepted && b.Code == http.StatusConflict) || (b.Code == http.StatusAccepted && a.Code == http.StatusConflict)) {
		t.Fatalf("statuses=%d,%d", a.Code, b.Code)
	}
	h.assertCount("agent_runs", "session_id=? AND status IN ('queued','running')", 1, s)
	h.provider.release()
}

func TestAcceptance10BackupRestoreLastBundleSucceeds(t *testing.T) {
	h := newHarness(t)
	n := h.directNote("notes/backup.md", "restored", "whole", "backup-note-key")
	bundle := h.backupNow()
	restored := h.restoreBundle(bundle)
	restored.assertReadyNoteFile(n.NoteID, "restored")
	restored.assertManifestChecksums()
}

func TestAcceptance11UnauthenticatedMutationRejected(t *testing.T) {
	h := newHarness(t)
	res := h.rawJSON("POST", "/api/v1/projects", `{"name":"takeover"}`, false)
	if res.Code != http.StatusUnauthorized { t.Fatalf("status=%d body=%s", res.Code, res.Body.String()) }
	h.assertCount("projects", "name=?", 0, "takeover")
}
```

Create `internal/acceptance/harness_test.go` with a `harness` that: opens all state beneath `t.TempDir()`; constructs the real `app.App` with injected fake clock/provider/bite generator/object sink; bootstraps and logs in through HTTP; records cookies and CSRF; exposes the exact helpers called above; closes/reopens the app for `restart`; invokes the real recovery and restore paths; and executes SQL assertions only for postcondition inspection. Every helper must call `t.Helper()`, fail immediately on unexpected status/error, and register cleanup. Do not reproduce business logic in the harness.

- [ ] **Step 2: Run the acceptance package to verify it fails**

Run: `go test ./internal/acceptance -v`

Expected: FAIL to compile until the harness and deterministic dependency injection are complete; after compilation, at least one invariant should fail if earlier phase hardening is incomplete.

- [ ] **Step 3: Complete the real integration harness and minimal dependency injection**

Add this production-neutral seam to `internal/app/app.go`, adapting field concrete types to the exact Phase 1–6 interfaces while keeping defaults in `New`:

```go
type Dependencies struct {
	Clock clock.Clock
	Provider agent.Provider
	BiteGenerator review.BiteGenerator
	ObjectSink backup.ObjectSink
	AfterPublishTransition func(string)
}

func DefaultDependencies(cfg config.Config) Dependencies {
	return Dependencies{
		Clock: clock.RealClock{},
		Provider: agent.NewOpenAICompatible(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey),
		BiteGenerator: review.NewProviderBiteGenerator(),
		ObjectSink: backup.NewConfiguredObjectSink(cfg),
	}
}
```

Have `NewWithDependencies(cfg, deps)` construct the same app as `New(cfg)`, pass one shared `SessionLocks` to delete/promote, and expose only lifecycle methods required by operators and tests: `Handler() http.Handler`, `Recover(context.Context) error`, and `Close() error`. Implement every `harness_test.go` helper against these public lifecycle methods and the real HTTP API. The crash hook must return a sentinel from the state-machine test seam without terminating the test process. Restore must target a fresh temporary data directory, validate the manifest/checksums, reopen SQLite, and then build a second real app over restored state.

- [ ] **Step 4: Run every named acceptance test and the race detector**

Run: `go test ./internal/acceptance -run '^TestAcceptance(01|02|03|04|05|06|07|08|09|10|11)' -race -count=1 -v`

Expected: PASS for all eleven exact names. The output is the executable mapping for spec §13.1–11:

| Spec criterion | Concrete acceptance test |
|---|---|
| §13.1 promote retry | `TestAcceptance01PromoteRetrySameKeyOneNoteOneReviewSet` |
| §13.2 crash recovery | `TestAcceptance02CrashAfterFSPublishRecoveryConverges` |
| §13.3 bite retry | `TestAcceptance03BiteFailureRetryNoDuplicateNote` |
| §13.4 session scope | `TestAcceptance04InvalidSessionScopeRejectedAPIAndDB` |
| §13.5 path escape | `TestAcceptance05TraversalAndSymlinkEscapeRejected` |
| §13.6 destination conflict | `TestAcceptance06DestinationExists409NoOverwrite` |
| §13.7 session deletion | `TestAcceptance07SessionDeleteRemovesWorkspaceOnly` |
| §13.8 rating idempotency | `TestAcceptance08RatingRetrySameKeyOneEvent` |
| §13.9 multi-tab run | `TestAcceptance09TwoTabsOneAgentRun` |
| §13.10 restore drill | `TestAcceptance10BackupRestoreLastBundleSucceeds` |
| §13.11 unauthenticated mutation | `TestAcceptance11UnauthenticatedMutationRejected` |

- [ ] **Step 5: Commit**

```bash
git add internal/acceptance/acceptance_test.go internal/acceptance/harness_test.go internal/app/app.go
git commit -m "test: cover all v1 acceptance invariants"
```

### Task 42: Polish Deployment and Developer Operations

**Files:**
- Create: `.amp/services.yaml`
- Create: `Makefile`
- Create: `README.md`
- Modify: `docs/ops/deploy.md`
- Modify: `docs/ops/backup-restore.md`

**Interfaces:**
- Consumes: `cmd/personal-agent`, port `8080`, `PA_DATA_DIR`, Compose/Caddy assets, bootstrap/auth configuration, and all Go tests.
- Produces: Reproducible orb development service, documented localhost and HTTPS deployment paths, test/lint targets, and a final verified repository entry point.

- [ ] **Step 1: Write failing documentation/configuration smoke checks**

Run this shell check before creating the files:

```bash
test -f .amp/services.yaml && \
test -f Makefile && \
grep -q '^test:' Makefile && \
grep -q '^lint:' Makefile && \
grep -q 'BOOTSTRAP_TOKEN' README.md && \
grep -q 'docker compose' docs/ops/deploy.md && \
grep -q 'restore drill' docs/ops/backup-restore.md
```

Expected: FAIL because one or more final developer/deployment entry points are absent or incomplete.

- [ ] **Step 2: Add the dev service and Make targets**

Create `.amp/services.yaml`:

```yaml
services:
  personal-agent:
    command: go run ./cmd/personal-agent
    working_dir: .
    environment:
      PA_DATA_DIR: .amp/state/personal-agent
      PA_LISTEN_ADDR: :8080
      PA_COOKIE_SECURE: "false"
      PA_MODELS: openai:test
      BOOTSTRAP_TOKEN: dev-only-change-me
    port: 8080
    healthcheck:
      path: /health
```

Create `Makefile`:

```make
.PHONY: test lint fmt-check run

test:
	go test ./...

lint: fmt-check
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || \
	  (echo "Go files need gofmt"; gofmt -l $$(find . -name '*.go' -not -path './.git/*'); exit 1)

run:
	go run ./cmd/personal-agent
```

- [ ] **Step 3: Write final operator-facing documentation**

Write `README.md` with: product scope and non-goals; Go 1.24+ prerequisite; `make test`, `make lint`, and `make run`; required `BOOTSTRAP_TOKEN` plus optional OpenAI/S3 variables without real secrets; first-run bootstrap/login; data layout; `.amp/services.yaml` usage via `amp orb services ensure`; links to the design, deployment, and backup/restore docs; and the warning that domain deployment requires HTTPS and secure cookies.

Update `docs/ops/deploy.md` with exact localhost and Compose commands, persistent volume ownership/writability checks, `.env` creation from `deploy/.env.example`, Caddy domain/TLS setup, bootstrap-before-exposure ordering, health check, upgrade sequence (backup, pull/build, migrate/start, health check), rollback boundaries, and secret rotation. Never instruct operators to commit `.env`.

Update `docs/ops/backup-restore.md` with the exact stop-writers → select last successful bundle → verify manifest/checksums → restore into an empty data directory → start app → run health/read/integrity checks sequence; include the command supplied by Phase 6 for the automated restore drill and state that a successful drill is required before considering backups operational.

- [ ] **Step 4: Verify configuration, lint, and the complete test suite**

Run:

```bash
test -f .amp/services.yaml && \
grep -q '^test:' Makefile && \
grep -q '^lint:' Makefile && \
grep -q 'BOOTSTRAP_TOKEN' README.md && \
grep -q 'docker compose' docs/ops/deploy.md && \
grep -qi 'restore drill' docs/ops/backup-restore.md && \
make lint && \
go test ./...
```

Expected: PASS; `go vet`, formatting verification, every package test, and all eleven acceptance tests succeed.

- [ ] **Step 5: Commit**

```bash
git add .amp/services.yaml Makefile README.md docs/ops/deploy.md docs/ops/backup-restore.md
git commit -m "docs: finalize development and deployment operations"
```

### Phase self-check

- Spec §9 F9/F10: session deletion is serialized with promotion and cleanly handles busy, terminal, path, provider, auth, CSRF, and bootstrap failure boundaries.
- Spec §10: different-key concurrent starts yield one run and one 409 busy response; same-key starts return one idempotent run; promote and session deletion share serialization.
- Spec §11: rooted paths reject traversal/symlinks, every domain mutation is authenticated and CSRF-protected, and owner bootstrap is one-time.
- Spec §13.1–11: every criterion maps to one exact `TestAcceptanceNN...` integration test listed in Task 41.
- Operational finish: `.amp/services.yaml`, README, deploy/restore instructions, Make test/lint targets, and final `go test ./...` verification are explicit.
