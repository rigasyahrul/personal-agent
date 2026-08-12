## Phase 3: Sessions + chat + agent run

### Task 15: Store project sessions and create their workspaces

**Files:**
- Create: `internal/store/sessions.go`
- Test: `internal/store/sessions_test.go`

**Interfaces:**
- Consumes: `ids.NewID() string`, `layout.SessionWorkspace(dataDir string, home layout.SessionHome, vaultID, projectID, sessionID string) string`, the migrated `projects` and `sessions` tables, and `domain.Session`.
- Produces: `type SessionStore struct { DB *sql.DB; DataDir string; Now func() time.Time }`, `type CreateSessionInput struct { ProjectID, Title, Provider, ModelID, ModelParametersJSON, ToolGrantsJSON string }`, `func (s *SessionStore) CreateProject(ctx context.Context, in CreateSessionInput) (domain.Session, error)`, and `func (s *SessionStore) ListByProject(ctx context.Context, projectID string) ([]domain.Session, error)`.

- [ ] **Step 1: Write the failing store tests**

```go
package store_test

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/rigasyahrul/personal-agent/internal/db"
    "github.com/rigasyahrul/personal-agent/internal/layout"
    "github.com/rigasyahrul/personal-agent/internal/store"
)

func TestSessionStoreCreateProjectAndList(t *testing.T) {
    data := t.TempDir()
    conn := db.OpenTest(t, data)
    now := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
    _, err := conn.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','Vault',?,?);
        INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','Project',?,?)`, now, now, now, now)
    if err != nil { t.Fatal(err) }
    ss := &store.SessionStore{DB: conn, DataDir: data, Now: func() time.Time { return now }}

    got, err := ss.CreateProject(context.Background(), store.CreateSessionInput{
        ProjectID: "p1", Title: "Learn", Provider: "openai", ModelID: "gpt-test",
        ModelParametersJSON: `{}`, ToolGrantsJSON: `{"workspace_files":false}`,
    })
    if err != nil { t.Fatal(err) }
    if got.Home != layout.SessionHome("project") || got.VaultID == nil || *got.VaultID != "v1" || got.ProjectID == nil || *got.ProjectID != "p1" {
        t.Fatalf("wrong scope: %#v", got)
    }
    workspace := layout.SessionWorkspace(data, got.Home, "v1", "p1", got.ID)
    if info, err := os.Stat(workspace); err != nil || !info.IsDir() { t.Fatalf("workspace: %v", err) }
    listed, err := ss.ListByProject(context.Background(), "p1")
    if err != nil || len(listed) != 1 || listed[0].ID != got.ID { t.Fatalf("list: %#v %v", listed, err) }
}

func TestSessionStoreRejectsMissingProjectWithoutDirectory(t *testing.T) {
    data := t.TempDir()
    ss := &store.SessionStore{DB: db.OpenTest(t, data), DataDir: data, Now: time.Now}
    _, err := ss.CreateProject(context.Background(), store.CreateSessionInput{ProjectID: "missing", Provider: "openai", ModelID: "m", ModelParametersJSON: `{}`, ToolGrantsJSON: `{}`})
    if err == nil { t.Fatal("expected missing project error") }
    entries, readErr := os.ReadDir(filepath.Join(data, "files"))
    if readErr == nil && len(entries) != 0 { t.Fatalf("unexpected directories: %v", entries) }
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/store -run 'TestSessionStore(CreateProjectAndList|RejectsMissingProjectWithoutDirectory)' -v`
Expected: FAIL because `SessionStore`, `CreateSessionInput`, and its methods do not exist.

- [ ] **Step 3: Implement project-scoped creation and listing**

```go
package store

import (
    "context"
    "database/sql"
    "errors"
    "os"
    "time"

    "github.com/rigasyahrul/personal-agent/internal/domain"
    "github.com/rigasyahrul/personal-agent/internal/ids"
    "github.com/rigasyahrul/personal-agent/internal/layout"
)

var ErrNotFound = errors.New("not found")
var ErrInvalidScope = errors.New("invalid session scope")
var ErrSessionTerminal = errors.New("session terminal")
var ErrSessionBusy = errors.New("session busy")

type SessionStore struct { DB *sql.DB; DataDir string; Now func() time.Time }
type CreateSessionInput struct { ProjectID, Title, Provider, ModelID, ModelParametersJSON, ToolGrantsJSON string }

func (s *SessionStore) CreateProject(ctx context.Context, in CreateSessionInput) (domain.Session, error) {
    var out domain.Session
    var vault sql.NullString
    if err := s.DB.QueryRowContext(ctx, `SELECT vault_id FROM projects WHERE id=?`, in.ProjectID).Scan(&vault); err != nil {
        if errors.Is(err, sql.ErrNoRows) { return out, ErrNotFound }; return out, err
    }
    now, id := s.Now().UTC(), ids.NewID()
    var vaultID any
    vaultText := ""
    if vault.Valid { vaultID, vaultText = vault.String, vault.String }
    _, err := s.DB.ExecContext(ctx, `INSERT INTO sessions
        (id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
        VALUES(?, 'project', ?, ?, 'active', ?, ?, ?, ?, ?, ?, ?)`,
        id, vaultID, in.ProjectID, in.Provider, in.ModelID, in.ModelParametersJSON, in.ToolGrantsJSON, in.Title, now, now)
    if err != nil { return out, err }
    workspace := layout.SessionWorkspace(s.DataDir, layout.SessionHome("project"), vaultText, in.ProjectID, id)
    if err := os.MkdirAll(workspace, 0700); err != nil {
        _, _ = s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id=?`, id)
        return out, err
    }
    return s.Get(ctx, id)
}

func (s *SessionStore) ListByProject(ctx context.Context, projectID string) ([]domain.Session, error) {
    rows, err := s.DB.QueryContext(ctx, `SELECT id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at,deleted_at FROM sessions WHERE project_id=? ORDER BY created_at DESC,id`, projectID)
    if err != nil { return nil, err }; defer rows.Close()
    out := []domain.Session{}
    for rows.Next() { var v domain.Session; if err := scanSession(rows, &v); err != nil { return nil, err }; out = append(out, v) }
    return out, rows.Err()
}
```

Add a local `scanner` interface and `scanSession(scanner, *domain.Session) error` in the same file; scan nullable IDs/timestamp into `sql.NullString`/`sql.NullTime`, then assign pointers only when valid. This is concrete mechanical mapping of the exact SELECT column order above and is shared by `Get` in Task 16.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/store -run 'TestSessionStore(CreateProjectAndList|RejectsMissingProjectWithoutDirectory)' -v`
Expected: PASS; the row inherits the project's vault and its derived workspace exists.

- [ ] **Step 5: Commit**

```bash
git add internal/store/sessions.go internal/store/sessions_test.go
git commit -m "feat: store project sessions and workspaces"
```

### Task 16: Enforce session scope, immutability, retrieval, and safe deletion

**Files:**
- Modify: `internal/db/migrations/001_init.sql`
- Modify: `internal/store/sessions.go`
- Modify: `internal/store/sessions_test.go`
- Test: `internal/db/migrate_test.go`

**Interfaces:**
- Consumes: `SessionStore` and `scanSession` from Task 15, `layout.SessionWorkspace`, and session status values `active`/`terminal`.
- Produces: DB scope/immutability enforcement, `func (s *SessionStore) Get(ctx context.Context, id string) (domain.Session, error)`, and `func (s *SessionStore) Delete(ctx context.Context, id string) error`.

- [ ] **Step 1: Write failing DB and store tests**

```go
func TestSessionScopeAndImmutableModel(t *testing.T) {
    conn := OpenTest(t, t.TempDir())
    now := time.Now().UTC()
    _, _ = conn.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v','V',?,?);
      INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p','v','P',?,?)`, now, now, now, now)
    bad := []string{
        `INSERT INTO sessions(id,home,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('s1','project','active','p','m','{}','{}','',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
        `INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('s2','project','wrong','p','active','p','m','{}','{}','',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
    }
    for _, q := range bad { if _, err := conn.Exec(q); err == nil { t.Fatalf("accepted invalid scope: %s", q) } }
    if _, err := conn.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('ok','project','v','p','active','p','m','{}','{}','',?,?)`, now, now); err != nil { t.Fatal(err) }
    if _, err := conn.Exec(`UPDATE sessions SET model_id='other' WHERE id='ok'`); err == nil { t.Fatal("model mutation accepted") }
}
```

```go
func TestSessionDeleteRemovesOnlyWorkspace(t *testing.T) {
    data := t.TempDir(); conn := db.OpenTest(t, data); seedVaultProject(t, conn, "v1", "p1")
    ss := &store.SessionStore{DB: conn, DataDir: data, Now: time.Now}
    session, err := ss.CreateProject(context.Background(), store.CreateSessionInput{ProjectID:"p1", Provider:"p", ModelID:"m", ModelParametersJSON:`{}`, ToolGrantsJSON:`{}`})
    if err != nil { t.Fatal(err) }
    workspace := layout.SessionWorkspace(data, session.Home, "v1", "p1", session.ID)
    if err := os.WriteFile(filepath.Join(workspace, "draft.md"), []byte("draft"), 0600); err != nil { t.Fatal(err) }
    source := filepath.Join(layout.SourceDir(layout.ProjectRoot(data, "v1", "p1")), "kept.md")
    if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil { t.Fatal(err) }
    if err := os.WriteFile(source, []byte("source"), 0600); err != nil { t.Fatal(err) }
    if err := ss.Delete(context.Background(), session.ID); err != nil { t.Fatal(err) }
    got, _ := ss.Get(context.Background(), session.ID)
    if got.Status != "terminal" || got.DeletedAt == nil { t.Fatalf("not tombstoned: %#v", got) }
    if _, err := os.Stat(workspace); !os.IsNotExist(err) { t.Fatalf("workspace remains: %v", err) }
    if body, err := os.ReadFile(source); err != nil || string(body) != "source" { t.Fatalf("source changed: %q %v", body, err) }
}
```

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./internal/db ./internal/store -run 'TestSession(ScopeAndImmutableModel|DeleteRemovesOnlyWorkspace)' -v`
Expected: FAIL because invalid/mismatched scope and model updates are accepted, and `Get`/`Delete` are missing.

- [ ] **Step 3: Add DB guards and safe tombstone deletion**

```sql
CREATE TRIGGER sessions_project_vault_insert
BEFORE INSERT ON sessions WHEN NEW.home = 'project'
BEGIN
  SELECT CASE WHEN NOT EXISTS (
    SELECT 1 FROM projects p WHERE p.id=NEW.project_id AND p.vault_id IS NEW.vault_id
  ) THEN RAISE(ABORT, 'session project vault mismatch') END;
END;

CREATE TRIGGER sessions_immutable_scope_model
BEFORE UPDATE OF home,vault_id,project_id,provider,model_id,model_parameters_json ON sessions
BEGIN
  SELECT RAISE(ABORT, 'session scope and model are immutable');
END;
```

Ensure the existing `sessions` definition includes:

```sql
CHECK (
 (home='global' AND vault_id IS NULL AND project_id IS NULL) OR
 (home='vault' AND vault_id IS NOT NULL AND project_id IS NULL) OR
 (home='project' AND project_id IS NOT NULL)
)
```

```go
func (s *SessionStore) Get(ctx context.Context, id string) (domain.Session, error) {
    var out domain.Session
    err := scanSession(s.DB.QueryRowContext(ctx, `SELECT id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at,deleted_at FROM sessions WHERE id=?`, id), &out)
    if errors.Is(err, sql.ErrNoRows) { return out, ErrNotFound }
    return out, err
}

func (s *SessionStore) Delete(ctx context.Context, id string) error {
    tx, err := s.DB.BeginTx(ctx, nil); if err != nil { return err }
    defer tx.Rollback()
    var home layout.SessionHome; var vault, project sql.NullString; var status string
    if err := tx.QueryRowContext(ctx, `SELECT home,vault_id,project_id,status FROM sessions WHERE id=?`, id).Scan(&home,&vault,&project,&status); err != nil {
        if errors.Is(err, sql.ErrNoRows) { return ErrNotFound }; return err
    }
    if status == "terminal" { return tx.Commit() }
    now := s.Now().UTC()
    result, err := tx.ExecContext(ctx, `UPDATE sessions SET status='terminal',deleted_at=?,updated_at=? WHERE id=? AND status='active'`, now, now, id)
    if err != nil { return err }; n, _ := result.RowsAffected(); if n != 1 { return ErrSessionBusy }
    if err := tx.Commit(); err != nil { return err }
    workspace := layout.SessionWorkspace(s.DataDir, home, nullableText(vault), nullableText(project), id)
    return os.RemoveAll(workspace)
}

func nullableText(v sql.NullString) string { if v.Valid { return v.String }; return "" }
```

The API will expose no update route, so provider/model are immutable both through the product API and direct DB writes. Tombstoning commits before deleting only the derived session workspace; project `source/`, notes, and review rows are never addressed.

- [ ] **Step 4: Run the focused tests to verify they pass**

Run: `go test ./internal/db ./internal/store -run 'TestSession(ScopeAndImmutableModel|DeleteRemovesOnlyWorkspace)' -v`
Expected: PASS, including preservation of the source note.

- [ ] **Step 5: Commit**

```bash
git add internal/db/migrations/001_init.sql internal/db/migrate_test.go internal/store/sessions.go internal/store/sessions_test.go
git commit -m "feat: enforce session lifecycle invariants"
```

### Task 17: Store ordered messages and one active agent run

**Files:**
- Create: `internal/store/messages.go`
- Create: `internal/store/runs.go`
- Test: `internal/store/messages_test.go`
- Test: `internal/store/runs_test.go`

**Interfaces:**
- Consumes: migrated `messages`/`agent_runs` tables, `ids.NewID()`, and `domain.Message`/`domain.AgentRun`.
- Produces: `MessageStore.Append`, `MessageStore.List`, `RunStore.CreateOrGet`, `RunStore.Current`, `RunStore.SetStatus`, and sentinel `store.ErrRequestKeyConflict`.

- [ ] **Step 1: Write failing ordering, idempotency, and concurrency tests**

```go
func TestMessagesAppendInSequence(t *testing.T) {
    conn, sid := seededSession(t)
    ms := &store.MessageStore{DB: conn, Now: time.Now}
    first, _ := ms.Append(context.Background(), sid, nil, "user", "hello", "complete")
    second, _ := ms.Append(context.Background(), sid, nil, "assistant", "hi", "complete")
    got, err := ms.List(context.Background(), sid)
    if err != nil || first.Sequence != 1 || second.Sequence != 2 || got[0].Content != "hello" || got[1].Content != "hi" { t.Fatalf("messages: %#v %v", got, err) }
}

func TestRunStoreOneActiveAndRequestKeyIdempotent(t *testing.T) {
    conn, sid := seededSession(t); rs := &store.RunStore{DB: conn, Now: time.Now}
    one, created, err := rs.CreateOrGet(context.Background(), sid, "key-1")
    if err != nil || !created { t.Fatalf("first: %#v %v", one, err) }
    same, created, err := rs.CreateOrGet(context.Background(), sid, "key-1")
    if err != nil || created || same.ID != one.ID { t.Fatalf("retry: %#v %v", same, err) }
    if _, _, err := rs.CreateOrGet(context.Background(), sid, "key-2"); !errors.Is(err, store.ErrSessionBusy) { t.Fatalf("want busy, got %v", err) }
    current, err := rs.Current(context.Background(), sid)
    if err != nil || current.ID != one.ID { t.Fatalf("current: %#v %v", current, err) }
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/store -run 'Test(MessagesAppendInSequence|RunStoreOneActiveAndRequestKeyIdempotent)' -v`
Expected: FAIL because message/run stores do not exist.

- [ ] **Step 3: Implement transactional sequence allocation and active-run uniqueness**

```go
type MessageStore struct { DB *sql.DB; Now func() time.Time }
func (s *MessageStore) Append(ctx context.Context, sessionID string, runID *string, role, content, status string) (domain.Message, error) {
    tx, err := s.DB.BeginTx(ctx, nil); if err != nil { return domain.Message{}, err }; defer tx.Rollback()
    var sequence int
    if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM messages WHERE session_id=?`, sessionID).Scan(&sequence); err != nil { return domain.Message{}, err }
    out := domain.Message{ID:ids.NewID(), SessionID:sessionID, RunID:runID, Sequence:sequence, Role:role, Content:content, Status:status, CreatedAt:s.Now().UTC()}
    _, err = tx.ExecContext(ctx, `INSERT INTO messages(id,session_id,run_id,sequence,role,content,status,created_at) VALUES(?,?,?,?,?,?,?,?)`, out.ID,out.SessionID,out.RunID,out.Sequence,out.Role,out.Content,out.Status,out.CreatedAt)
    if err != nil { return domain.Message{}, err }; return out, tx.Commit()
}
func (s *MessageStore) List(ctx context.Context, sessionID string) ([]domain.Message, error) {
    rows, err := s.DB.QueryContext(ctx, `SELECT id,session_id,run_id,sequence,role,content,tool_calls_json,tool_call_id,status,created_at FROM messages WHERE session_id=? ORDER BY sequence`, sessionID)
    if err != nil { return nil, err }; defer rows.Close(); out := []domain.Message{}
    for rows.Next() { var m domain.Message; var run, calls, callID sql.NullString; if err := rows.Scan(&m.ID,&m.SessionID,&run,&m.Sequence,&m.Role,&m.Content,&calls,&callID,&m.Status,&m.CreatedAt); err != nil{return nil,err}; if run.Valid{m.RunID=&run.String}; if calls.Valid{m.ToolCallsJSON=&calls.String}; if callID.Valid{m.ToolCallID=&callID.String}; out=append(out,m) }
    return out, rows.Err()
}
```

```go
var ErrRequestKeyConflict = errors.New("request key conflict")
type RunStore struct { DB *sql.DB; Now func() time.Time }
func (s *RunStore) CreateOrGet(ctx context.Context, sessionID, requestKey string) (domain.AgentRun, bool, error) {
    if got, err := s.byKey(ctx, sessionID, requestKey); err == nil { return got, false, nil } else if !errors.Is(err, ErrNotFound) { return got,false,err }
    now, id := s.Now().UTC(), ids.NewID()
    _, err := s.DB.ExecContext(ctx, `INSERT INTO agent_runs(id,session_id,request_key,status,started_at) SELECT ?,?,?,'queued',? WHERE EXISTS(SELECT 1 FROM sessions WHERE id=? AND status='active')`, id,sessionID,requestKey,now,sessionID)
    if err != nil {
        if got, e := s.byKey(ctx, sessionID, requestKey); e == nil { return got,false,nil }
        if strings.Contains(err.Error(), "agent_runs_one_active") { return domain.AgentRun{},false,ErrSessionBusy }
        return domain.AgentRun{},false,err
    }
    got, err := s.byKey(ctx, sessionID, requestKey); return got, true, err
}
func (s *RunStore) Current(ctx context.Context, sessionID string) (domain.AgentRun,error) { return scanRun(s.DB.QueryRowContext(ctx, `SELECT id,session_id,request_key,status,started_at,completed_at,error FROM agent_runs WHERE session_id=? AND status IN ('queued','running')`,sessionID)) }
func (s *RunStore) SetStatus(ctx context.Context, id, status, message string) error { _,err:=s.DB.ExecContext(ctx, `UPDATE agent_runs SET status=?,completed_at=CASE WHEN ? IN ('completed','failed','cancelled') THEN ? END,error=NULLIF(?,'') WHERE id=?`,status,status,s.Now().UTC(),message,id); return err }
```

Implement `byKey` and `scanRun` with the same explicit seven-column scan. Ensure migration `001_init.sql` contains `CREATE UNIQUE INDEX agent_runs_one_active ON agent_runs(session_id) WHERE status IN ('queued','running');` and `UNIQUE(session_id,request_key)`. SQLite serializes competing inserts; the partial unique index, not UI state, decides the winner.

- [ ] **Step 4: Run store tests**

Run: `go test ./internal/store -run 'Test(MessagesAppendInSequence|RunStoreOneActiveAndRequestKeyIdempotent)' -v`
Expected: PASS; a retry returns the original run and a different key gets `ErrSessionBusy`.

- [ ] **Step 5: Commit**

```bash
git add internal/store/messages.go internal/store/messages_test.go internal/store/runs.go internal/store/runs_test.go internal/db/migrations/001_init.sql
git commit -m "feat: store messages and serialize agent runs"
```

### Task 18: Add the OpenAI-compatible provider and idempotent runner

**Files:**
- Create: `internal/agent/provider.go`
- Create: `internal/agent/openai_compat.go`
- Create: `internal/agent/runner.go`
- Create: `internal/agent/runner_test.go`

**Interfaces:**
- Consumes: locked `Provider.Chat(ctx, req)`, `store.MessageStore`, `store.RunStore`, and immutable provider/model values from `SessionStore.Get`.
- Produces: `ChatRequest`, `ChatResponse`, `OpenAICompat`, `Runner`, and locked `func (r *Runner) Start(ctx context.Context, sessionID, requestKey string, userMessage string) (runID string, err error)`.

- [ ] **Step 1: Write failing provider and runner tests with a fake**

```go
type fakeProvider struct { calls atomic.Int32; block chan struct{}; err error }
func (f *fakeProvider) Chat(ctx context.Context, req agent.ChatRequest) (agent.ChatResponse,error) {
    f.calls.Add(1); if f.block != nil { select { case <-f.block: case <-ctx.Done(): return agent.ChatResponse{},ctx.Err() } }
    return agent.ChatResponse{Content:"answer"}, f.err
}
func TestRunnerStartIsIdempotentAndCompletes(t *testing.T) {
    conn, sid := seededSession(t); fake := &fakeProvider{}
    r := &agent.Runner{Sessions:&store.SessionStore{DB:conn}, Messages:&store.MessageStore{DB:conn,Now:time.Now}, Runs:&store.RunStore{DB:conn,Now:time.Now}, Providers:map[string]agent.Provider{"openai":fake}}
    one, err := r.Start(context.Background(),sid,"request-1","question"); if err != nil { t.Fatal(err) }
    two, err := r.Start(context.Background(),sid,"request-1","question"); if err != nil || two != one { t.Fatalf("retry %q %v",two,err) }
    if fake.calls.Load()!=1 { t.Fatalf("provider calls=%d",fake.calls.Load()) }
    messages, _ := r.Messages.List(context.Background(),sid); if len(messages)!=2 || messages[1].Content!="answer" { t.Fatalf("%#v",messages) }
}
func TestRunnerProviderFailureKeepsHistory(t *testing.T) {
    conn,sid:=seededSession(t); fake:=&fakeProvider{err:errors.New("offline")}; r:=newRunner(conn,fake)
    runID,err:=r.Start(context.Background(),sid,"request-2","saved question"); if err==nil { t.Fatal("want provider error") }
    run,_:=r.Runs.ByID(context.Background(),runID); if run.Status!="failed" { t.Fatalf("%#v",run) }
    messages,listErr:=r.Messages.List(context.Background(),sid); if listErr!=nil || len(messages)!=1 || messages[0].Content!="saved question" { t.Fatalf("history %#v %v",messages,listErr) }
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/agent -run 'TestRunner(StartIsIdempotentAndCompletes|ProviderFailureKeepsHistory)' -v`
Expected: FAIL because the provider and runner types are absent.

- [ ] **Step 3: Implement provider adapter and synchronous run lifecycle**

```go
package agent
type ChatMessage struct { Role string `json:"role"`; Content string `json:"content"` }
type ChatRequest struct { Model string `json:"model"`; Messages []ChatMessage `json:"messages"`; Parameters map[string]any `json:"-"` }
type ChatResponse struct { Content string }
type Provider interface { Chat(context.Context, ChatRequest) (ChatResponse,error) }
```

```go
type OpenAICompat struct { BaseURL, APIKey string; Client *http.Client }
func (p *OpenAICompat) Chat(ctx context.Context, in ChatRequest) (ChatResponse,error) {
    body:=map[string]any{"model":in.Model,"messages":in.Messages}; for k,v:=range in.Parameters { body[k]=v }
    encoded,err:=json.Marshal(body); if err!=nil{return ChatResponse{},err}
    req,err:=http.NewRequestWithContext(ctx,http.MethodPost,strings.TrimRight(p.BaseURL,"/")+"/chat/completions",bytes.NewReader(encoded)); if err!=nil{return ChatResponse{},err}
    req.Header.Set("Authorization","Bearer "+p.APIKey); req.Header.Set("Content-Type","application/json")
    resp,err:=p.Client.Do(req); if err!=nil{return ChatResponse{},err}; defer resp.Body.Close()
    if resp.StatusCode/100!=2 { b,_:=io.ReadAll(io.LimitReader(resp.Body,4096)); return ChatResponse{},fmt.Errorf("provider status %d: %s",resp.StatusCode,b) }
    var out struct{ Choices []struct{ Message ChatMessage `json:"message"` } `json:"choices"` }
    if err:=json.NewDecoder(resp.Body).Decode(&out); err!=nil{return ChatResponse{},err}; if len(out.Choices)==0{return ChatResponse{},errors.New("provider returned no choices")}; return ChatResponse{Content:out.Choices[0].Message.Content},nil
}
```

```go
type Runner struct { Sessions *store.SessionStore; Messages *store.MessageStore; Runs *store.RunStore; Providers map[string]Provider }
func (r *Runner) Start(ctx context.Context, sessionID, requestKey, userMessage string) (string,error) {
    run,created,err:=r.Runs.CreateOrGet(ctx,sessionID,requestKey); if err!=nil{return "",err}; if !created{return run.ID,nil}
    session,err:=r.Sessions.Get(ctx,sessionID); if err!=nil{return run.ID,err}
    if _,err=r.Messages.Append(ctx,sessionID,&run.ID,"user",userMessage,"complete"); err!=nil { _=r.Runs.SetStatus(ctx,run.ID,"failed",err.Error()); return run.ID,err }
    _=r.Runs.SetStatus(ctx,run.ID,"running","")
    history,err:=r.Messages.List(ctx,sessionID); if err!=nil{return run.ID,err}; req:=ChatRequest{Model:session.ModelID}
    if err=json.Unmarshal([]byte(session.ModelParametersJSON),&req.Parameters); err!=nil{return run.ID,err}; for _,m:=range history { req.Messages=append(req.Messages,ChatMessage{Role:m.Role,Content:m.Content}) }
    provider,ok:=r.Providers[session.Provider]; if !ok { err=fmt.Errorf("provider %q unavailable",session.Provider) } else { var response ChatResponse; response,err=provider.Chat(ctx,req); if err==nil { _,err=r.Messages.Append(ctx,sessionID,&run.ID,"assistant",response.Content,"complete") } }
    if err!=nil { _=r.Runs.SetStatus(ctx,run.ID,"failed",err.Error()); return run.ID,err }; return run.ID,r.Runs.SetStatus(ctx,run.ID,"completed","")
}
```

Add `RunStore.ByID` as the same seven-column `scanRun` query. Tools are intentionally absent: `tool_grants_json` is persisted but this phase sends only message history. A duplicate request key returns before appending/calling, so reconnect cannot double-start.

- [ ] **Step 4: Run agent tests**

Run: `go test ./internal/agent -run 'TestRunner(StartIsIdempotentAndCompletes|ProviderFailureKeepsHistory)' -v`
Expected: PASS; provider failure leaves the user message readable and marks the run failed.

- [ ] **Step 5: Commit**

```bash
git add internal/agent/provider.go internal/agent/openai_compat.go internal/agent/runner.go internal/agent/runner_test.go internal/store/runs.go
git commit -m "feat: add idempotent chat runner"
```

### Task 19: Expose session and chat HTTP endpoints

**Files:**
- Create: `internal/httpapi/session_handlers.go`
- Create: `internal/httpapi/chat_handlers.go`
- Modify: `internal/httpapi/server.go`
- Create: `internal/httpapi/session_handlers_test.go`
- Create: `internal/httpapi/chat_handlers_test.go`

**Interfaces:**
- Consumes: `SessionStore`, `MessageStore`, `RunStore`, `agent.Runner`, auth/CSRF middleware, and `/api/v1` ServeMux conventions.
- Produces: project session list/create, session get/delete, message list/post, and current-run handlers; project create always forces `home=project` and unsupported `home=global|vault` returns HTTP 400.

- [ ] **Step 1: Write failing HTTP integration tests**

```go
func TestSessionAPIRejectsInvalidScopeAndKeepsModelImmutable(t *testing.T) {
    app, projectID := testServer(t)
    for _,home:=range []string{"global","vault"} {
        res:=authedJSON(t,app,"POST","/api/v1/projects/"+projectID+"/sessions",map[string]any{"home":home,"title":"x","provider":"openai","model_id":"m"},"key")
        if res.Code!=http.StatusBadRequest { t.Fatalf("home %s: %d",home,res.Code) }
    }
    created:=authedJSON(t,app,"POST","/api/v1/projects/"+projectID+"/sessions",map[string]any{"home":"project","title":"x","provider":"openai","model_id":"m","model_parameters":map[string]any{}},"key")
    if created.Code!=http.StatusCreated { t.Fatal(created.Body.String()) }
    var session domain.Session; json.Unmarshal(created.Body.Bytes(),&session)
    update:=authedJSON(t,app,"PUT","/api/v1/sessions/"+session.ID,map[string]any{"model_id":"changed"},"key")
    if update.Code!=http.StatusMethodNotAllowed { t.Fatalf("model update status=%d",update.Code) }
}
func TestChatAPIRetryDoesNotDoubleStartAndHistorySurvivesProviderFailure(t *testing.T) {
    app,sid,fake:=testChatServer(t); fake.err=errors.New("offline")
    first:=authedJSON(t,app,"POST","/api/v1/sessions/"+sid+"/messages",map[string]string{"content":"hello","request_key":"same"},"csrf")
    second:=authedJSON(t,app,"POST","/api/v1/sessions/"+sid+"/messages",map[string]string{"content":"hello","request_key":"same"},"csrf")
    if first.Code!=http.StatusBadGateway || second.Code!=http.StatusAccepted || fake.calls.Load()!=1 { t.Fatalf("codes %d/%d calls %d",first.Code,second.Code,fake.calls.Load()) }
    history:=authedJSON(t,app,"GET","/api/v1/sessions/"+sid+"/messages",nil,"")
    if history.Code!=http.StatusOK || !strings.Contains(history.Body.String(),"hello") { t.Fatalf("history: %d %s",history.Code,history.Body.String()) }
}
```

- [ ] **Step 2: Run HTTP tests to verify they fail**

Run: `go test ./internal/httpapi -run 'Test(SessionAPIRejectsInvalidScopeAndKeepsModelImmutable|ChatAPIRetryDoesNotDoubleStartAndHistorySurvivesProviderFailure)' -v`
Expected: FAIL with unregistered routes (404/405).

- [ ] **Step 3: Register and implement the handlers**

```go
type sessionCreateRequest struct { Home,Title,Provider,ModelID string; ModelParameters map[string]any `json:"model_parameters"`; ToolGrants map[string]bool `json:"tool_grants"` }
func (s *Server) projectSessions(w http.ResponseWriter,r *http.Request) {
    projectID:=r.PathValue("id")
    if r.Method==http.MethodGet { out,err:=s.Sessions.ListByProject(r.Context(),projectID); writeResult(w,out,err); return }
    var in sessionCreateRequest; if err:=json.NewDecoder(r.Body).Decode(&in); err!=nil { writeError(w,400,"invalid_json"); return }
    if in.Home!="" && in.Home!="project" { writeError(w,400,"invalid_scope"); return }
    params,_:=json.Marshal(in.ModelParameters); if in.ToolGrants==nil { in.ToolGrants=map[string]bool{"workspace_files":false} }; grants,_:=json.Marshal(in.ToolGrants)
    out,err:=s.Sessions.CreateProject(r.Context(),store.CreateSessionInput{ProjectID:projectID,Title:in.Title,Provider:in.Provider,ModelID:in.ModelID,ModelParametersJSON:string(params),ToolGrantsJSON:string(grants)})
    if errors.Is(err,store.ErrNotFound){writeError(w,404,"project_not_found");return}; if err!=nil{writeError(w,500,"create_failed");return}; writeJSON(w,201,out)
}
func (s *Server) session(w http.ResponseWriter,r *http.Request) { id:=r.PathValue("id"); if r.Method==http.MethodDelete { err:=s.Sessions.Delete(r.Context(),id); writeNoContentOrError(w,err); return }; out,err:=s.Sessions.Get(r.Context(),id); writeResult(w,out,err) }
```

```go
func (s *Server) messages(w http.ResponseWriter,r *http.Request) {
    sid:=r.PathValue("id"); if r.Method==http.MethodGet { out,err:=s.Messages.List(r.Context(),sid); writeResult(w,out,err); return }
    var in struct{ Content string `json:"content"`; RequestKey string `json:"request_key"` }; if json.NewDecoder(r.Body).Decode(&in)!=nil || in.Content=="" || in.RequestKey=="" { writeError(w,400,"invalid_message");return }
    runID,err:=s.Runner.Start(r.Context(),sid,in.RequestKey,in.Content)
    if errors.Is(err,store.ErrSessionBusy){writeError(w,409,"session_busy");return}; if err!=nil { writeJSON(w,502,map[string]any{"run_id":runID,"error":"provider_unavailable"});return }; writeJSON(w,202,map[string]string{"run_id":runID})
}
func (s *Server) currentRun(w http.ResponseWriter,r *http.Request) { out,err:=s.Runs.Current(r.Context(),r.PathValue("id")); if errors.Is(err,store.ErrNotFound){w.WriteHeader(204);return}; writeResult(w,out,err) }
```

Register exact methods so no update endpoint can mutate provider/model:

```go
mux.Handle("GET /api/v1/projects/{id}/sessions", secured(http.HandlerFunc(s.projectSessions)))
mux.Handle("POST /api/v1/projects/{id}/sessions", secured(http.HandlerFunc(s.projectSessions)))
mux.Handle("GET /api/v1/sessions/{id}", secured(http.HandlerFunc(s.session)))
mux.Handle("DELETE /api/v1/sessions/{id}", secured(http.HandlerFunc(s.session)))
mux.Handle("GET /api/v1/sessions/{id}/messages", secured(http.HandlerFunc(s.messages)))
mux.Handle("POST /api/v1/sessions/{id}/messages", secured(http.HandlerFunc(s.messages)))
mux.Handle("GET /api/v1/sessions/{id}/runs/current", secured(http.HandlerFunc(s.currentRun)))
```

Use the server's existing `writeJSON`, `writeError`, authentication, and CSRF helpers; GET remains available when AI is down. There are deliberately no tool routes in this task.

- [ ] **Step 4: Run HTTP tests**

Run: `go test ./internal/httpapi -run 'Test(SessionAPIRejectsInvalidScopeAndKeepsModelImmutable|ChatAPIRetryDoesNotDoubleStartAndHistorySurvivesProviderFailure)' -v`
Expected: PASS; retry uses the durable request key, history GET works during provider failure, and PUT is 405.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/session_handlers.go internal/httpapi/chat_handlers.go internal/httpapi/server.go internal/httpapi/session_handlers_test.go internal/httpapi/chat_handlers_test.go
git commit -m "feat: expose sessions and chat API"
```

### Task 20: Build project sessions and polling chat UI

**Files:**
- Modify: `web/js/api.js`
- Modify: `web/js/router.js`
- Create: `web/js/pages/sessions.js`
- Modify: `web/css/app.css`
- Create: `web/js/pages/sessions.test.js`

**Interfaces:**
- Consumes: Task 19 JSON routes and the existing vanilla-JS router/API request helper.
- Produces: project sessions list/new form, immutable model display, sequenced chat, message sending with one stable request key per submission, and polling of messages/current run.

- [ ] **Step 1: Write the failing browser-module test**

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import { createSessionsPage } from './sessions.js';

test('re-render polling does not submit twice and history remains readable after send failure', async () => {
  const calls = [];
  const api = async (path, options = {}) => {
    calls.push([path, options]);
    if (path.endsWith('/messages') && options.method === 'POST') throw new Error('AI unavailable');
    if (path.endsWith('/messages')) return [{sequence: 1, role: 'user', content: 'kept'}];
    if (path.endsWith('/runs/current')) return null;
    return [];
  };
  const root = document.createElement('main');
  const page = createSessionsPage({root, api, projectID: 'p1', randomUUID: () => 'stable-key', setInterval: () => 7, clearInterval: () => {}});
  await page.openChat({id:'s1', title:'Chat', provider:'openai', model_id:'m'});
  root.querySelector('[name=message]').value = 'hello';
  await root.querySelector('form[data-chat]').onsubmit({preventDefault(){}});
  await page.poll(); await page.poll();
  assert.equal(calls.filter(([p,o]) => p.endsWith('/messages') && o.method === 'POST').length, 1);
  assert.match(root.textContent, /kept/); assert.match(root.textContent, /AI unavailable/);
  page.destroy();
});
```

- [ ] **Step 2: Run the UI test to verify it fails**

Run: `node --test web/js/pages/sessions.test.js`
Expected: FAIL because `sessions.js` and `createSessionsPage` do not exist (use the DOM shim already established by prior web tests).

- [ ] **Step 3: Implement the sessions page and polling chat**

```js
export function createSessionsPage({root, api, projectID, randomUUID=crypto.randomUUID.bind(crypto), setInterval=window.setInterval.bind(window), clearInterval=window.clearInterval.bind(window)}) {
  let session=null, timer=null, sending=false, error='';
  const esc = value => String(value).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  async function list() {
    const sessions=await api(`/api/v1/projects/${projectID}/sessions`);
    root.innerHTML=`<h1>Sessions</h1><form data-new><input name="title" required placeholder="Title"><input name="provider" required placeholder="Provider"><input name="model_id" required placeholder="Model"><label><input type="checkbox" name="workspace_files"> Workspace files</label><button>New session</button></form><ul>${sessions.map(s=>`<li><button data-session="${esc(s.id)}">${esc(s.title)} — ${esc(s.provider)}:${esc(s.model_id)}</button></li>`).join('')}</ul>`;
    root.querySelector('form[data-new]').onsubmit=async e=>{e.preventDefault();const f=new FormData(e.currentTarget);const created=await api(`/api/v1/projects/${projectID}/sessions`,{method:'POST',body:{home:'project',title:f.get('title'),provider:f.get('provider'),model_id:f.get('model_id'),model_parameters:{},tool_grants:{workspace_files:f.get('workspace_files')==='on'}}});await openChat(created)};
    root.querySelectorAll('[data-session]').forEach(button=>button.onclick=()=>openChat(sessions.find(s=>s.id===button.dataset.session)));
  }
  function render(messages=[],run=null) {
    root.innerHTML=`<button data-back>Sessions</button><h1>${esc(session.title)}</h1><p class="model-badge">${esc(session.provider)}:${esc(session.model_id)}</p><ol class="messages">${messages.sort((a,b)=>a.sequence-b.sequence).map(m=>`<li class="${esc(m.role)}"><b>${esc(m.role)}</b> ${esc(m.content)}</li>`).join('')}</ol><p class="run-status">${run?esc(run.status):''}</p><p role="alert">${esc(error)}</p><form data-chat><textarea name="message" required></textarea><button ${sending||run?'disabled':''}>Send</button></form>`;
    root.querySelector('[data-back]').onclick=()=>{destroy();list()}; root.querySelector('form[data-chat]').onsubmit=send;
  }
  async function poll(){if(!session)return;const [messages,run]=await Promise.all([api(`/api/v1/sessions/${session.id}/messages`),api(`/api/v1/sessions/${session.id}/runs/current`)]);render(messages,run)}
  async function send(e){e.preventDefault();if(sending)return;sending=true;error='';const content=e.currentTarget.elements.message.value,key=randomUUID();try{await api(`/api/v1/sessions/${session.id}/messages`,{method:'POST',body:{content,request_key:key}})}catch(err){error=err.message}finally{sending=false;await poll()}}
  async function openChat(value){session=value;await poll();if(timer===null)timer=setInterval(poll,1500)}
  function destroy(){if(timer!==null)clearInterval(timer);timer=null;session=null}
  return {list,openChat,poll,destroy};
}
```

Add API methods that JSON-encode `options.body` and include CSRF through the existing helper, route `/projects/:id/sessions` to `createSessionsPage(...).list()`, and add focused `.messages`, `.model-badge`, and `.run-status` styles. The model is displayed but never editable after creation. Polling only performs GETs; only a deliberate form submit creates a UUID, so reconnect/re-render cannot start another run. Failed sends retain fetched history and show the AI outage inline.

- [ ] **Step 4: Run UI and full focused phase checks**

Run: `node --test web/js/pages/sessions.test.js && go test ./internal/store ./internal/agent ./internal/httpapi`
Expected: PASS; the UI submits once, keeps history visible on AI failure, and all session/chat backend tests pass.

- [ ] **Step 5: Commit**

```bash
git add web/js/api.js web/js/router.js web/js/pages/sessions.js web/js/pages/sessions.test.js web/css/app.css
git commit -m "feat: add sessions and polling chat UI"
```

### Phase self-check

- Covers §5 Session/AgentRun/Message: all schema scopes remain representable, project-vault equality is DB-enforced, API creates project scope only, provider/model configuration is immutable, messages are sequenced, and the partial unique index permits one non-terminal run.
- Covers §9 F2/F3/F9: project session creation creates its derived workspace, chat remains readable when AI is unavailable, and deletion tombstones the session then removes only its workspace—not source notes or review history.
- Covers §10 and §13: durable request-key idempotency prevents reconnect retries from double-starting; the DB partial unique index arbitrates two tabs rather than relying on disabled buttons.
- Covers §8 Sessions/Chat: sessions list, one-time model selection, tools-off default, ordered chat, run status, and polling/reconnect behavior. Workspace tools and tree remain intentionally deferred to Phase 4.
