# Personal Agent v1 (Thin Vertical) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the approved self-hosted, single-tenant learning dashboard thin vertical for projects, source notes, agent sessions, promotion, spaced review, and backup.

**Architecture:** A browser connects to a single-host Docker Compose deployment, where the Go API is the sole writer for all durable mutations. SQLite plus files and staging on a local data volume are the system of record; optional S3-compatible storage receives backup snapshots only.

**Tech Stack:** Go 1.24+, SQLite (`modernc.org/sqlite`), stdlib `net/http`, vanilla JS SPA, Docker Compose, Caddy, optional S3

## Global Constraints

- Module path: `github.com/rigasyahrul/personal-agent`.
- Go version: **1.24+** (`os.Root` requires 1.24); orb `.agents/setup` installs from go.dev if the distro version is older.
- SQLite driver: `modernc.org/sqlite` (pure Go, no CGO); use WAL and short transactions.
- HTTP router: stdlib `net/http` ServeMux with Go 1.22+ patterns; API listens on `:8080`.
- UI: plain HTML, vanilla JavaScript modules, and minimal CSS under `web/` (no bundler); Go embeds `web/` via `embed.FS` in production; dev may serve `web/` from disk.
- Data directory: `PA_DATA_DIR`, default `./data`; the host data volume containing SQLite, files, and staging is the system of record, and the Go API is its sole writer.
- IDs: UUIDv4 strings from `github.com/google/uuid`.
- API timestamps: RFC3339 UTC; owner timezone is an IANA timezone string and controls review “today.”
- Path limits: Markdown body maximum **1 MiB** (`1_048_576` bytes); relative path maximum **512** UTF-8 bytes; depth maximum **16** components; component maximum **255** bytes.
- Paths are relative logical POSIX paths under a trusted root. Reject empty and absolute paths, `.`, `..`, NUL/control characters, and empty components; promote/direct accept regular `.md` files only, never symlinks, devices, sockets, or FIFOs, and rooted APIs must prevent escape.
- `memory/` and `soul/` are reserved project names and are not v1 promotion destinations; staging is backend-only. URLs and review references use `note_id`, not paths.
- Authentication is single-owner. Bootstrap uses one-time `BOOTSTRAP_TOKEN`; store an **argon2id** PHC-style password hash; use `pa_session` (`HttpOnly`, `Secure`, `SameSite=Lax`) and CSRF double-submit cookie `pa_csrf`.
- All mutations require the auth cookie and CSRF except bootstrap, login, and health as designed; secrets come from environment variables or secret mounts.
- Workspace tool grants default to `workspace_files: false`; tools are rooted inside the session workspace, model arguments are untrusted, and there is no arbitrary shell tool.
- Scheduler name is exactly `sm2-lite-v1`; ratings are `again`, `hard`, `good`, and `easy`.
- Bite generator version is `bites-v1`; output schema is `{ "bites": [ { "prompt": string, "answer": string } ] }` with at most 8 bites.
- Publication statuses are exactly `accepted → frozen → path_reserved → published_fs → finalized → review_enqueued → completed`, with terminal error status `failed`.
- Other exact statuses: Note `pending | ready | failed`; ReviewPending `pending | leased | completed | failed`; AgentRun `queued | running | completed | failed | cancelled`; Session `active | terminal`; BackupRun `running | succeeded | failed`.
- One model per session; `home`, `vault_id`, `project_id`, `provider`, `model_id`, and `model_parameters` are immutable. v1 UI creates project sessions only; project placement is immutable.
- Agent provider v1 uses OpenAI-compatible HTTP chat completions via `OPENAI_API_KEY` and optional `OPENAI_BASE_URL`; models come from `PA_MODELS=provider:model_id,...`.
- Enforce one non-terminal AgentRun per session and idempotency for agent runs, promote, direct create, and review ratings; multi-tab and multi-device browsers are expected.
- Optional S3 is backup-only and the application must work without a bucket; backup uses a mutation barrier, consistent SQLite backup, immutable file/operation bundle, manifest, and checksums.
- Deferred/non-goals: FTS; multi-tenant SaaS; multi-writer or multi-device file sync; live S3 storage; note edit/delete UI; attachments and non-Markdown promotion; global/vault session UI; todos, Google, mail, and cross-session chat; Amp as a runtime requirement; arbitrary external host edits and reconcile importing.
- Tests use `t.TempDir()`; integration tests use temporary directories. Follow TDD red → green → commit, keep steps about 2–5 minutes where possible, and make exactly one commit per task.
- No `TBD`, `TODO`, placeholders, or “similar to Task N”; task test and implementation steps contain complete pasteable code.

## File Structure

```text
go.mod                                      # Declare the locked module path, Go version, and dependencies.
go.sum                                      # Pin dependency checksums.
cmd/personal-agent/main.go                  # Start the configured application process.
internal/config/config.go                   # Load and validate environment configuration.
internal/ids/ids.go                         # Generate UUIDv4 entity IDs through NewID.
internal/clock/clock.go                     # Provide real and fake clocks for deterministic behavior.
internal/paths/paths.go                     # Validate rooted relative paths and fixed limits.
internal/paths/paths_test.go                # Verify path validation, limits, and rejection cases.
internal/fsroot/root.go                     # Perform safe rooted filesystem operations over os.Root/fallback.
internal/fsroot/root_test.go                # Verify containment and unsafe-node rejection.
internal/db/db.go                           # Open SQLite, configure WAL, and run migrations.
internal/db/migrations/001_init.sql         # Create the complete initial schema, constraints, and indexes.
internal/db/migrate_test.go                 # Verify clean and repeat migration behavior.
internal/domain/models.go                   # Define shared domain structs, enums, and exact statuses.
internal/auth/password.go                   # Hash and verify the owner password.
internal/auth/password_test.go              # Test password hashing and rejection.
internal/auth/session.go                    # Issue, persist, validate, and revoke owner sessions.
internal/auth/session_test.go               # Test session token lifecycle and cookie semantics.
internal/auth/csrf.go                       # Implement double-submit CSRF checks.
internal/auth/bootstrap.go                  # Enforce one-time owner bootstrap.
internal/auth/bootstrap_test.go             # Test bootstrap takeover prevention and one-time behavior.
internal/store/vaults.go                    # Persist and query minimal vault records.
internal/store/projects.go                  # Persist projects and immutable vault placement.
internal/store/sessions.go                  # Persist sessions and enforce scope/lifecycle invariants.
internal/store/notes.go                     # Index note metadata and verify filesystem integrity.
internal/store/messages.go                  # Append and list ordered session messages.
internal/store/runs.go                      # Persist idempotent agent runs and single-active-run rules.
internal/store/promote.go                   # Persist promote operations and transitions.
internal/store/direct.go                    # Persist direct-create operations and transitions.
internal/store/review.go                    # Persist review jobs, items, ratings, and events.
internal/store/backup.go                    # Persist backup run state and history.
internal/store/settings.go                  # Persist owner timezone and defaults.
internal/store/*_test.go                    # Test store queries, constraints, concurrency, and idempotency.
internal/layout/layout.go                   # Derive project source and scoped workspace locations from IDs.
internal/layout/layout_test.go              # Verify global, vault, and project path derivation.
internal/publish/machine.go                 # Execute the shared promote/direct publication state machine.
internal/publish/machine_test.go            # Test publication, no-clobber, retries, and transitions.
internal/publish/recover.go                 # Resume non-terminal publication operations at startup.
internal/review/scheduler.go                # Apply the exact sm2-lite-v1 schedule.
internal/review/scheduler_test.go           # Verify every schedule rating and boundary.
internal/review/queue.go                    # Build explicitly scoped due-item queues.
internal/review/bites.go                    # Lease and execute bite-generation jobs.
internal/review/bites_test.go               # Test schema limits, retries, and deduplication.
internal/agent/provider.go                  # Define provider requests, responses, and interface.
internal/agent/openai_compat.go             # Call OpenAI-compatible chat completions.
internal/agent/runner.go                    # Run idempotent session turns with one active run.
internal/agent/runner_test.go               # Test run lifecycle, provider failures, and concurrency.
internal/agent/tools/workspace.go           # Expose opt-in rooted workspace file tools.
internal/agent/tools/workspace_test.go       # Test grants, atomic writes, and path/symlink escape rejection.
internal/backup/backup.go                   # Create consistent local snapshot bundles under a mutation barrier.
internal/backup/backup_test.go              # Verify bundle manifests and restore-ready snapshots.
internal/backup/s3.go                       # Upload optional snapshots to S3-compatible storage.
internal/httpapi/server.go                  # Assemble the v1 ServeMux and static SPA serving.
internal/httpapi/middleware.go              # Enforce request IDs, authentication, and CSRF.
internal/httpapi/health.go                  # Report liveness and writable-storage health without auth.
internal/httpapi/auth_handlers.go           # Handle setup, login, logout, and current owner.
internal/httpapi/project_handlers.go        # Handle vaults, projects, folders, trees, and home aggregates.
internal/httpapi/note_handlers.go           # Handle direct creation and integrity-checked note reads.
internal/httpapi/session_handlers.go        # Handle session creation, detail, listing, and deletion.
internal/httpapi/chat_handlers.go           # Handle messages, runs, and workspace browsing.
internal/httpapi/promote_handlers.go        # Handle promote requests and operation status.
internal/httpapi/review_handlers.go         # Handle scoped queues, ratings, suspension, and retry.
internal/httpapi/settings_handlers.go       # Handle timezone and application defaults.
internal/httpapi/backup_handlers.go         # Handle backup history and manual snapshots.
internal/httpapi/*_test.go                  # Exercise authenticated API behavior with httptest.
internal/app/app.go                         # Wire configuration, DB, stores, workers, API, and recovery.
web/index.html                              # Provide the SPA shell and navigation mount points.
web/css/app.css                             # Style dashboard, responsive screens, and status states.
web/js/api.js                               # Wrap JSON API calls, CSRF, and error handling.
web/js/router.js                            # Route browser locations to SPA pages.
web/js/app.js                               # Bootstrap shared UI state and navigation.
web/js/pages/home.js                        # Render home projects, activity, health, and due review.
web/js/pages/project.js                     # Render project overview and project navigation.
web/js/pages/notes.js                       # Render source tree, note viewer, folder, and direct-create UI.
web/js/pages/sessions.js                    # Render session list, chat, workspace, and promote flow.
web/js/pages/review.js                      # Render explicit-scope review cards and rating actions.
web/js/pages/settings.js                    # Render bootstrap, timezone, model, health, and backup settings.
web/js/components/status-badges.js          # Render durable run, publication, and bite-job statuses.
web/js/components/markdown.js               # Safely render Markdown note content.
deploy/docker-compose.yml                   # Run the API and Caddy on one host with a persistent volume.
deploy/Dockerfile                           # Build and package the Go service and static SPA.
deploy/Caddyfile                            # Proxy localhost/domain traffic and terminate domain TLS.
deploy/.env.example                         # Document deploy configuration without secrets.
docs/ops/backup-restore.md                  # Document backup configuration and tested restore drill.
docs/ops/deploy.md                          # Document localhost and domain deployment procedures.
Makefile                                    # Provide repeatable build, test, and run targets.
README.md                                   # Explain product setup, use, and operational links.
.amp/services.yaml                          # Optionally expose the development server through an orb portal.
```

Tests live beside their packages as `*_test.go`; integration tests in `internal/httpapi` and `internal/publish` use temporary directories.

## Resolved Defaults

| Decision | v1 lock |
|---|---|
| Go version | **1.24+** (`os.Root`); install from go.dev in orb setup if distro older |
| Module path | `github.com/rigasyahrul/personal-agent` |
| SQLite driver | `modernc.org/sqlite` (pure Go, no CGO) |
| HTTP router | stdlib `net/http` ServeMux (Go 1.22+ patterns) |
| UI | Vanilla JS static SPA under `web/` (no bundler); embed via `embed.FS` |
| Reverse proxy | Caddy via `caddy:2-alpine`; TLS when a domain is set, direct `:8080` for localhost |
| Auth | Single owner; **argon2id** DB password hash; one-time `BOOTSTRAP_TOKEN`; `pa_session` HttpOnly Secure SameSite=Lax; `pa_csrf` double-submit cookie |
| Password hash | **argon2id** (`golang.org/x/crypto/argon2`); PHC-style encoded string |
| IDs | UUIDv4 via `github.com/google/uuid` |
| JSON times | RFC3339 UTC; IANA owner timezone for review “today” |
| Max `.md` body | **1 MiB** (`1_048_576` bytes) |
| Max path length | **512** bytes UTF-8 |
| Max path depth | **16** components under root |
| Max component length | **255** bytes |
| Workspace tool grants default | `workspace_files: false` |
| Scheduler | `sm2-lite-v1`; Again is +10m and other ratings use day intervals below |
| Bite generator | `{ "bites": [ { "prompt": string, "answer": string } ] }`, maximum 8; `generator_version = "bites-v1"` |
| Vault UI | Optional `vault_id` / vault name during project creation only; no vault browser |
| FTS | Deferred; no FTS in v1 |
| Agent provider | OpenAI-compatible HTTP chat completions; `OPENAI_API_KEY`, optional `OPENAI_BASE_URL`; `PA_MODELS=provider:model_id,...` |
| Port | `:8080` |
| Data directory | `PA_DATA_DIR`, default `./data` |
| Test data | Every test uses `t.TempDir()` |

SM-2 state starts with `stage=0`, `interval_days=0`, `ease_factor=2.5` (minimum `1.3`), `reps=0`, `lapses=0`, and `due_at=now`; state also records `last_reviewed_at`.

| Rating | Exact effect |
|---|---|
| `again` | `lapses++`; `reps=0`; `stage=0`; `interval_days=0`; `due_at=now+10m`; `ease_factor=max(1.3, ease_factor-0.2)` |
| `hard` | `reps++`; `ease_factor=max(1.3, ease_factor-0.15)`; if `stage==0`, interval `0.5d`, otherwise `interval_days*1.2`; `stage=max(stage,1)`; `due_at=now+interval` |
| `good` | `reps++`; if `stage==0`, interval `1d` and stage `1`; else if `stage==1`, interval `3d` and stage `2`; otherwise `interval_days*ease_factor`; `due_at=now+interval` |
| `easy` | `reps++`; `ease_factor+=0.15`; if `stage<2`, interval `4d` and stage `2`; otherwise `interval_days*ease_factor*1.3`; `due_at=now+interval` |

## API Surface

All API routes use prefix `/api/v1` except `/health`.

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Liveness and writable storage; no auth |
| GET | `/api/v1/setup/status` | Read bootstrap state |
| POST | `/api/v1/setup/bootstrap` | Set owner password with bootstrap token |
| POST | `/api/v1/auth/login` | Log in |
| POST | `/api/v1/auth/logout` | Log out |
| GET | `/api/v1/auth/me` | Read current owner |
| GET/PUT | `/api/v1/settings` | Read/update timezone and defaults |
| GET | `/api/v1/models` | List server-configured provider/model pairs |
| GET/POST | `/api/v1/vaults` | List/create minimal vaults |
| GET/POST | `/api/v1/projects` | List/create projects |
| GET | `/api/v1/projects/{id}` | Read project overview aggregates |
| GET | `/api/v1/projects/{id}/tree` | Browse source tree |
| GET | `/api/v1/notes/{id}` | Read note metadata and body |
| POST | `/api/v1/projects/{id}/folders` | Create folder under source |
| POST | `/api/v1/projects/{id}/direct-notes` | Start DirectCreateOperation |
| GET/POST | `/api/v1/projects/{id}/sessions` | List/create project sessions |
| GET | `/api/v1/sessions/{id}` | Read session detail |
| DELETE | `/api/v1/sessions/{id}` | Mark terminal and delete workspace |
| GET | `/api/v1/sessions/{id}/messages` | List ordered messages |
| POST | `/api/v1/sessions/{id}/messages` | Add user message and idempotently start run |
| GET | `/api/v1/sessions/{id}/runs/current` | Read current/non-terminal run |
| GET | `/api/v1/sessions/{id}/workspace/tree` | Browse workspace tree |
| GET | `/api/v1/sessions/{id}/workspace/file` | Read workspace file with `?path=` |
| POST | `/api/v1/sessions/{id}/promote` | Start PromoteOperation |
| GET | `/api/v1/operations/{id}` | Read promote/direct operation status |
| GET | `/api/v1/review/queue` | Read queue with `?scope=all\|project:{id}` |
| POST | `/api/v1/review/items/{id}/rate` | Rate item idempotently |
| POST | `/api/v1/review/items/{id}/suspend` | Suspend item |
| POST | `/api/v1/review/pending/{id}/retry` | Retry bite job |
| GET/POST | `/api/v1/backups` | List backups or start Backup now |
| GET | `/api/v1/home` | Read home dashboard DTO |

## Task Index

| Tasks | Phase | Draft |
|---|---|---|
| 1–8 | Skeleton | `01-skeleton.md` |
| 9–14 | Projects + source | `02-projects-source.md` |
| 15–20 | Sessions + chat | `03-sessions-chat.md` |
| 21–24 | Workspace tools | `04-workspace-tools.md` |
| 25–32 | Promote + review | `05-promote-review.md` |
| 33–36 | Backup | `06-backup.md` |
| 37–42 | Hardening | `07-hardening.md` |

## How to execute

Implement one task at a time using TDD (red → green → commit), preserving the locked interfaces and exact defaults. Each phase leaves working, testable software and should be verified before the next phase begins.

### Implementation completeness rule

Some task steps compress long functions into prose (“implement finalize”, “wire barrier”). That is **not** permission to invent conflicting APIs. When a step is prose-only:

1. Implement against **Canonical contracts** and the migration in Task 3.
2. Keep functions small and tested; add extra `*_test.go` cases as needed.
3. Prefer extending existing types over new parallel ones.
4. Before marking a task done: `go test ./...` passes and the task’s **Interfaces: Produces** symbols exist with the locked signatures.


---


## Canonical contracts (authoritative over task snippets)

If any task code snippet conflicts with this section, **follow this section**. Task snippets remain illustrative; implementers must compile against these contracts.

### Database path

- Runtime DB file: `$PA_DATA_DIR/db/personal-agent.sqlite` (WAL/SHM siblings beside it).
- Tests: `database.Open(ctx, filepath.Join(t.TempDir(), "db", "personal-agent.sqlite"))`.
- Helper allowed in test packages:

```go
// testutil or package-local
func openTestDB(t *testing.T) (*sql.DB, string) {
    t.Helper()
    dir := t.TempDir()
    db, err := database.Open(context.Background(), filepath.Join(dir, "db", "personal-agent.sqlite"))
    if err != nil { t.Fatal(err) }
    t.Cleanup(func() { _ = db.Close() })
    return db, dir
}
```

Never use a nonexistent `testutil.OpenDB` symbol — use `openTestDB` / `database.Open` as above.

**Test DB helper lives in `internal/testutil` (exported package), not `internal/db/*_test.go`.**

Task 3 must Create `internal/testutil/db.go`:

```go
package testutil

import (
  "context"
  "database/sql"
  "path/filepath"
  "testing"

  database "github.com/rigasyahrul/personal-agent/internal/db"
)

func OpenDB(t *testing.T, dataDir string) *sql.DB {
  t.Helper()
  db, err := database.Open(context.Background(), filepath.Join(dataDir, "db", "personal-agent.sqlite"))
  if err != nil { t.Fatal(err) }
  t.Cleanup(func() { _ = db.Close() })
  return db
}

func TempDB(t *testing.T) (db *sql.DB, dataDir string) {
  t.Helper()
  dataDir = t.TempDir()
  return OpenDB(t, dataDir), dataDir
}
```

Replace every snippet call `testutil.OpenDB(t, x)` with `testutil.OpenDB(t, x)` and `func() *sql.DB { db, _ := testutil.TempDB(t); return db }()` with `testutil.TempDB(t)`.


### Schema table names (migration 001)

Exact names: `owner`, `settings`, `auth_sessions`, `vaults`, `projects`, `sessions`, `agent_runs`, `messages`, `notes`, `promote_ops`, `direct_ops`, `review_pending`, `review_items`, `review_events`, `backup_runs`.


### Model configuration (exact)

`internal/config.Config` MUST include (Task 1 create; Task 8 env example):

```go
type ModelRef struct {
    Provider string // e.g. "openai"
    ModelID  string // e.g. "gpt-4o-mini"
}
type Config struct {
    DataDir, Addr, BootstrapToken string
    SecureCookies bool
    OpenAIAPIKey  string // OPENAI_API_KEY
    OpenAIBaseURL string // OPENAI_BASE_URL, optional
    Models        []ModelRef // from PA_MODELS=provider:model_id,provider:model_id
}
```

Parse `PA_MODELS` as comma-separated `provider:model_id` pairs; empty list allowed (read-only app). Reject session create if `provider`+`model_id` not in configured list when list non-empty; if list empty, reject create with 503/400 "no models configured".

API:

- `GET /api/v1/models` → `{ "models": [ {"provider":"...","model_id":"..."} ] }` (auth required)
- New session UI (Task 20) populates `<select>` from this endpoint — no free-text provider/model when models are configured.

Wire `agent.OpenAICompat` from `OpenAIAPIKey`/`OpenAIBaseURL` in `app.New`.

### Config env vars

| Env | Field | Default |
|-----|-------|---------|
| `PA_DATA_DIR` | `DataDir` | `./data` |
| `PA_ADDR` | `Addr` | `:8080` |
| `BOOTSTRAP_TOKEN` | `BootstrapToken` | empty |
| `PA_SECURE_COOKIES` | `SecureCookies` | `true` unless set to `false` |
| `OPENAI_API_KEY` | provider | |
| `OPENAI_BASE_URL` | provider | optional |
| `PA_MODELS` | model list | |

### `internal/agent` types (single definition)

```go
package agent

type ChatMessage struct {
    Role       string          `json:"role"`
    Content    string          `json:"content"`
    ToolCallID string          `json:"tool_call_id,omitempty"`
    ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
}

type ToolCall struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Arguments string `json:"arguments"` // JSON object string
}

type ToolDefinition struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Parameters  any    `json:"parameters"` // JSON Schema object
}

type ChatRequest struct {
    Model      string           `json:"model"`
    Messages   []ChatMessage    `json:"messages"`
    Tools      []ToolDefinition `json:"tools,omitempty"`
    Parameters map[string]any   `json:"-"`
}

type ChatResponse struct {
    Content   string
    ToolCalls []ToolCall
    Raw       json.RawMessage // optional
}

type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

type MessageStore interface {
    List(ctx context.Context, sessionID string) ([]domain.Message, error)
    Append(ctx context.Context, msg domain.Message) error
}

type RunStore interface {
    BeginOrGet(ctx context.Context, sessionID, requestKey string) (runID string, existing bool, err error)
    MarkRunning(ctx context.Context, runID string) error
    MarkDone(ctx context.Context, runID, status string, errMsg string) error
}

type SessionReader interface {
    Get(ctx context.Context, id string) (domain.Session, error)
}

type Runner struct {
    DB       *sql.DB
    DataDir  string
    Provider Provider
    Messages MessageStore
    Runs     RunStore
    Sessions SessionReader
    Clock    clock.Clock
    // Tools factory: nil tools when grant off
    Workspace func(session domain.Session) (*tools.Workspace, error)
}

func (r *Runner) Start(ctx context.Context, sessionID, requestKey, userMessage string) (runID string, err error)
// Start is idempotent on requestKey; returns ErrBusy if another non-terminal run exists with different key.
// execute(ctx, runID, session) runs the provider loop (used internally and from recovery).
func (r *Runner) execute(ctx context.Context, runID string, session domain.Session) error
```

Task 18 creates base Runner without tools. Task 22 sets `Runner.Workspace` and extends the loop to handle `ToolCalls` when `session.ToolGrants.WorkspaceFiles` is true. Do not redefine `ChatRequest` in Task 22 — only add tool fields already shown above.

### `internal/publish.Machine` (single machine, both kinds)

```go
type PublishInput struct {
    OpID, RequestKey, RequestFingerprint string
    Kind              string // "promote" | "direct"
    SessionID         string // required if promote
    WorkspacePath     string // required if promote (rel)
    Body              []byte // required if direct
    TargetProjectID   string
    TargetRelPath     string
    ReviewMode        domain.ReviewMode // none|whole|bites
    NoteID            string
}

type Machine struct {
    DB      *sql.DB
    DataDir string
    Clock   clock.Clock
}

func (m *Machine) Run(ctx context.Context, in PublishInput) (opStatus string, noteID string, err error)
func (m *Machine) RecoverAll(ctx context.Context) error
```

- Task 13 implements `Kind=="direct"` end-to-end against table `direct_ops`.
- Task 25 **extends** the same `Run` for `Kind=="promote"` against `promote_ops` (do not create a second machine type; handle promote; keep direct path).
- Both kinds share freeze→reserve→publish_fs→finalize; promote freezes from workspace via rooted read; direct freezes from `Body`.
- Vault placement: always `SELECT vault_id FROM projects WHERE id=?` and pass to `layout.ProjectRoot` — never hard-code empty vault.

### Publication vault placement (mandatory)

```go
var vaultID sql.NullString
if err := m.DB.QueryRowContext(ctx, `SELECT vault_id FROM projects WHERE id=?`, in.TargetProjectID).Scan(&vaultID); err != nil {
    return "", in.NoteID, err
}
root := layout.ProjectRoot(m.DataDir, vaultID.String, in.TargetProjectID) // vaultID.String is "" when NULL
```

Never hard-code `vaultID := ""` in promote/direct/recover paths.

```go
func loadProjectVault(db *sql.DB, projectID string) string {
    var v sql.NullString
    if err := db.QueryRow(`SELECT vault_id FROM projects WHERE id=?`, projectID).Scan(&v); err != nil {
        panic(err) // production code returns error; tests may use must*
    }
    return v.String
}
```


- Op tables: `promote_ops` / `direct_ops` only (never `promote_operations` / `direct_create_operations`).

### Settings + first-run UI (required; fold into Tasks 5–7)

Because the file map lists settings files, implementers MUST create them even if early task snippets omit full code:

| File | Responsibility |
|------|----------------|
| `internal/store/settings.go` | Get/Put owner timezone (IANA), default_provider, default_model_id |
| `internal/httpapi/settings_handlers.go` | `GET/PUT /api/v1/settings` (auth+CSRF on PUT) |
| `web/js/pages/settings.js` | Timezone field, AI defaults display (keys never shown), backup status panel (filled Task 35) |
| `web/js/pages/setup.js` (or home gate) | If `setup/status` says needs bootstrap → password form; else if unauthenticated → login form |

Task 5 creates bootstrap/login/me/CSRF APIs. Task 6 wires them. Task 7 static shell MUST include setup+login screens calling those APIs (not only empty Home). Task 35 only **extends** settings page with backup controls.

### Backup bundle layout (exact)

Production data dir:

```
$PA_DATA_DIR/
  db/personal-agent.sqlite
  files/global/... and files/vaults/...
  staging/...
  backups/local/{run_id}/...
```

Bundle written to `$PA_DATA_DIR/backups/local/{run_id}/`:

```
manifest.json
database.sqlite                 # from SQLite Backup API only
files/global/...                # mirror of $PA_DATA_DIR/files/**
files/vaults/...
staging/...                     # mirror of $PA_DATA_DIR/staging/** if any
```

Snapshot algorithm:

1. Acquire mutation barrier; wait in-flight publish ops to terminal or durable pause.
2. `database.sqlite` ← SQLite online backup from open DB (not file copy of live db).
3. Copy tree `$PA_DATA_DIR/files` → bundle `files/` (Preserve relative paths under files/).
4. Copy tree `$PA_DATA_DIR/staging` → bundle `staging/` if exists.
5. Write `manifest.json` with sha256 of every bundled file + cutoff_at.
6. Release barrier. Upload optional. Mark BackupRun.

**Do not** walk `$PA_DATA_DIR` root and re-prefix arbitrarily. **Do not** include `db/` or `backups/` in the file walk.

Restore drill / TestAcceptance10:

1. Stop app.
2. New empty `restoreDir`.
3. `restoreDir/db/personal-agent.sqlite` ← bundle `database.sqlite`.
4. `restoreDir/files/**` ← bundle `files/**`.
5. `restoreDir/staging/**` ← bundle `staging/**` if present.
6. Remove wal/shm if any.
7. `database.Open(ctx, restoreDir+"/db/personal-agent.sqlite")` and assert known note row + file bytes on disk under `restoreDir/files/...`.
8. Optional: boot `app.New` with `DataDir=restoreDir` and `GET /api/v1/notes/{id}`.

Task 33 tests must seed notes under `dataDir/files/global/projects/{id}/source/...` via layout helpers, not `dataDir/global/...`.


### Backup schedule (v1)

Spec requires schedule + Backup now. v1 lock:

- Settings fields: `backup_schedule` enum `off` | `daily` (default `off`), stored in `settings` table (add column in 001 or via Task 35 migration-free JSON settings blob if already present).
- When `daily`: in-process ticker in `app.New` fires at next local 03:00 owner-timezone (or UTC if unset) and calls the same `backup.Service.Run` as Backup now. Missed ticks do not catch up more than once on startup.
- UI (Task 35): radio/select Off vs Daily + Backup now button + last success/fail.
- Bucket config remains env-only (`PA_S3_*`); UI shows whether sink is configured (boolean), not secret values.

If `settings` table in Task 3 lacks schedule column, add `backup_schedule TEXT NOT NULL DEFAULT 'off'` in Task 3 migration (preferred) or document ALTER in Task 35.

### Backup artifact + S3 contract (exact, overrides Task 33–34 snippets)

**Local artifact is always a directory** (not a required `.tar.gz`):

```
$PA_DATA_DIR/backups/local/{run_id}/
  manifest.json
  database.sqlite
  files/**          # mirror of $PA_DATA_DIR/files/**
  staging/**        # optional mirror
```

`backup_runs.local_path` stores that directory path (absolute or data-dir-relative).

**`backup.Sink` interface** (replace any `PutObject` single-file-only API in snippets):

```go
package backup

// Sink uploads a completed local bundle directory. nil Sink => local-only success.
type Sink interface {
    // Upload mirrors the directory tree to object storage under objectPrefix/.
    // Every file under localDir is uploaded; keys are objectPrefix + relative path.
    Upload(ctx context.Context, localDir string, objectPrefix string) error
}

// S3Sink implements Sink via s3-compatible client (PutObject per file).
type S3Sink struct {
    Client interface {
        PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) error
    }
    Bucket string
}
```

Optional: after directory bundle is complete, an implementer MAY also write `bundle.tar.gz` beside it for operator convenience, but **BackupRun success and S3 upload MUST be defined on the directory tree + manifest**, not solely on a tarball. Task 33–34 snippets mentioning only `.tar.gz` / single `PutObject(LocalPath)` are void — use directory `Sink.Upload`.

Task 34 test: memSink records uploaded keys; assert `manifest.json`, `database.sqlite`, and at least one `files/...` key.


### Acceptance harness (Task 41)

Create `internal/apitest` or `internal/httpapi/acceptance_test.go` helpers:

```go
type Fixture struct {
    T       *testing.T
    DataDir string
    DB      *sql.DB
    Server  *httptest.Server
    Client  *http.Client
    CSRF    string
    // cookies jar on Client
}

func NewFixture(t *testing.T) *Fixture // openTest DB, migrate, bootstrap owner, login, start httptest with real httpapi.New
func (f *Fixture) AuthJSON(method, path string, body any) *http.Response
```

Wiring:

- Provider: fake `agent.Provider` injected via `ServerDeps.Provider`.
- BiteGenerator: fake implementing `review.BiteGenerator` OR provider returning fixed JSON; inject via `ServerDeps.BiteGenerator`.
- Backup sink: `backup.Sink` interface `{ Upload(ctx, localDir, objectKey string) error }`; nil sink = local-only success; tests use `memSink`.
- Workspace files for promote tests: write with `os.WriteFile` into `layout.SessionWorkspace(...)` or `tools.Workspace` — **never** `inProcessWorkspaceWrite` HTTP.

Delete any snippet named `workspaceWrite` — replace with in-process file create.

Constructors:

```go
// ServerDeps must include everything app wires:
type ServerDeps struct {
    DB, DataDir, Clock, BootstrapToken, SecureCookies, Static,
    Provider agent.Provider,
    BiteGenerator review.BiteGenerator,
    BackupSink backup.Sink, // may be nil
    Runner *agent.Runner,
    Publish *publish.Machine,
    // stores...
}
```

If a Task 41 snippet references undefined symbols, implement the harness using only the above — ignore the stale symbol names in the snippet.

### Bite generation

```go
// internal/review/bites.go
type BiteGenerator interface {
    Generate(ctx context.Context, noteBody string) ([]Bite, error)
}
type Bite struct{ Prompt, Answer string }
// Default impl uses agent.Provider with fixed system prompt; tests use fake.
```

### Review rating

One DB transaction: read item with `row_version`, apply `scheduler.ApplyRating`, bump `row_version`, insert `review_events` with UNIQUE `request_key`.

### HTTP workspace

No general workspace write HTTP API in v1. Tests seed workspace files via `os`/`tools.Workspace` in-process, not HTTP. Agent writes only through tools when granted.

### Error sentinels (shared)

```go
var (
    ErrNotFound = errors.New("not found")
    ErrConflict = errors.New("conflict")
    ErrBusy      = errors.New("busy")
    ErrUnauthorized = errors.New("unauthorized")
    ErrForbidden = errors.New("forbidden")
    ErrValidation = errors.New("validation")
    ErrIntegrity = errors.New("integrity")
)
```

Map to HTTP: 404/409/409/401/403/400/409 as appropriate.

---


<!-- Assembled from docs/memory/plan-drafts/ — implementers: this file is authoritative -->


## Phase 1: Skeleton

### Task 1: Go module, configuration, IDs, clock, and Makefile

**Config errata:** `Config` MUST include `OpenAIAPIKey`, `OpenAIBaseURL`, and `Models []ModelRef` parsed from `PA_MODELS` per Canonical **Model configuration**. Tests cover default empty models and parsing `openai:gpt-4o-mini`.


**Files:**
- Create: `go.mod`, `Makefile`, `internal/config/config.go`, `internal/config/config_test.go`, `internal/ids/ids.go`, `internal/ids/ids_test.go`, `internal/clock/clock.go`, `internal/clock/clock_test.go`

**Interfaces:**
- Consumes: environment variables `PA_DATA_DIR`, `PA_ADDR`, `BOOTSTRAP_TOKEN`, `PA_SECURE_COOKIES`
- Produces: `func config.Load() (config.Config, error)`; `func ids.NewID() string`; `type clock.Clock interface{ Now() time.Time }`; `type clock.RealClock struct{}`; `func (clock.RealClock) Now() time.Time`; `type clock.FakeClock struct{ T time.Time }`; `func (*clock.FakeClock) Now() time.Time`; `func (*clock.FakeClock) Advance(time.Duration)`

- [ ] **Step 1: Write the failing tests**

```go
// internal/config/config_test.go
package config
import "testing"
func TestLoadDefaults(t *testing.T) { t.Setenv("PA_DATA_DIR", ""); t.Setenv("PA_ADDR", ""); c, err := Load(); if err != nil || c.DataDir != "./data" || c.Addr != ":8080" { t.Fatalf("%+v %v", c, err) } }

// internal/ids/ids_test.go
package ids
import ("testing"; "github.com/google/uuid")
func TestNewIDIsUUID4(t *testing.T) { u, err := uuid.Parse(NewID()); if err != nil || u.Version() != 4 { t.Fatalf("%v %v", u, err) } }

// internal/clock/clock_test.go
package clock
import ("testing"; "time")
func TestFakeClockAdvance(t *testing.T) { f := &FakeClock{T: time.Unix(0, 0)}; f.Advance(time.Minute); if !f.Now().Equal(time.Unix(60, 0)) { t.Fatal(f.Now()) } }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config ./internal/ids ./internal/clock -v`
Expected: FAIL because the module and packages do not exist.

- [ ] **Step 3: Write the minimal implementation**

```go
// go.mod
module github.com/rigasyahrul/personal-agent

go 1.24

require github.com/google/uuid v1.6.0

// internal/config/config.go
package config
import ("errors"; "os")
type Config struct { DataDir, Addr, BootstrapToken string; SecureCookies bool }
func Load() (Config, error) { c := Config{DataDir: os.Getenv("PA_DATA_DIR"), Addr: os.Getenv("PA_ADDR"), BootstrapToken: os.Getenv("BOOTSTRAP_TOKEN"), SecureCookies: os.Getenv("PA_SECURE_COOKIES") != "false"}; if c.DataDir == "" { c.DataDir = "./data" }; if c.Addr == "" { c.Addr = ":8080" }; if c.Addr[0] != ':' { return c, errors.New("PA_ADDR must begin with ':'") }; return c, nil }

// internal/ids/ids.go
package ids
import "github.com/google/uuid"
func NewID() string { return uuid.NewString() }

// internal/clock/clock.go
package clock
import "time"
type Clock interface{ Now() time.Time }
type RealClock struct{}
func (RealClock) Now() time.Time { return time.Now().UTC() }
type FakeClock struct{ T time.Time }
func (f *FakeClock) Now() time.Time { return f.T }
func (f *FakeClock) Advance(d time.Duration) { f.T = f.T.Add(d) }

// Makefile
.PHONY: test run build
test:
	go test ./...
run:
	go run ./cmd/personal-agent
build:
	go build ./cmd/personal-agent
```

Run: `go mod tidy`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config ./internal/ids ./internal/clock -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum Makefile internal/config internal/ids internal/clock
git commit -m "chore: initialize Go skeleton"
```

### Task 2: Validate relative paths

**Files:**
- Create: `internal/paths/paths.go`, `internal/paths/paths_test.go`

**Interfaces:**
- Consumes: logical POSIX UTF-8 relative paths
- Produces: `type paths.PathError struct{ Code, Message string }`; `func paths.ValidateRelPath(string) (string, error)`; constants `MaxPathBytes = 512`, `MaxDepth = 16`, `MaxComponentBytes = 255`, `MaxMarkdownBytes = 1 << 20`

- [ ] **Step 1: Write the failing test**

```go
package paths
import "testing"
func TestValidateRelPath(t *testing.T) {
	valid := map[string]string{"notes/a.md":"notes/a.md", "a//b":"a/b"}
	for in, want := range valid { got, err := ValidateRelPath(in); if err != nil || got != want { t.Errorf("%q: %q %v", in, got, err) } }
	bad := []string{"", ".", "..", "../a", "/a", "a/../b", "a/./b", "a\x00b", "a\nb"}
	for _, in := range bad { if _, err := ValidateRelPath(in); err == nil { t.Errorf("accepted %q", in) } }
	if _, err := ValidateRelPath(string(make([]byte, MaxPathBytes+1))); err == nil { t.Fatal("accepted long path") }
	deep := "a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a"; if _, err := ValidateRelPath(deep); err == nil { t.Fatal("accepted deep path") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/paths -run TestValidateRelPath -v`
Expected: FAIL with `undefined: ValidateRelPath`.

- [ ] **Step 3: Write minimal implementation**

```go
package paths
import ("fmt"; "path"; "strings"; "unicode/utf8")
const ( MaxPathBytes = 512; MaxDepth = 16; MaxComponentBytes = 255; MaxMarkdownBytes = 1 << 20 )
type PathError struct{ Code, Message string }
func (e *PathError) Error() string { return e.Code + ": " + e.Message }
func reject(code, message string) (string, error) { return "", &PathError{Code: code, Message: message} }
func ValidateRelPath(p string) (string, error) {
	if p == "" || !utf8.ValidString(p) { return reject("invalid_path", "path must be non-empty UTF-8") }
	if len(p) > MaxPathBytes || strings.HasPrefix(p, "/") { return reject("invalid_path", "path is absolute or too long") }
	for _, r := range p { if r < 0x20 || r == 0x7f { return reject("invalid_path", "control characters are forbidden") } }
	for _, c := range strings.Split(p, "/") { if c == "." || c == ".." { return reject("invalid_path", "dot components are forbidden") } }
	clean := path.Clean(p); parts := strings.Split(clean, "/")
	if clean == "." || len(parts) > MaxDepth { return reject("invalid_path", "path is empty or too deep") }
	for _, c := range parts { if len(c) == 0 || len(c) > MaxComponentBytes { return reject("invalid_path", fmt.Sprintf("invalid component %q", c)) } }
	return clean, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/paths -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/paths
git commit -m "feat: validate relative paths"
```

### Task 3: SQLite WAL and complete initial migration

**Settings errata:** `settings` table MUST include `backup_schedule TEXT NOT NULL DEFAULT 'off'` (and timezone etc.).

**Also Create:** `internal/testutil/db.go` with `OpenDB`/`TempDB` (see Canonical contracts).

**Files:**
- Create: `internal/db/db.go`, `internal/db/migrations/001_init.sql`, `internal/db/migrate_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: `context.Context`, database file path
- Produces: `func db.Open(context.Context, string) (*sql.DB, error)`; embedded, idempotently recorded migration `001_init.sql`

- [ ] **Step 1: Write the failing test**

```go
package db
import ("context"; "path/filepath"; "testing")
func TestOpenMigratesAllTablesAndWAL(t *testing.T) { d, err := Open(context.Background(), filepath.Join(t.TempDir(), "db", "app.sqlite")); if err != nil { t.Fatal(err) }; defer d.Close(); var mode string; if err := d.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" { t.Fatalf("%q %v", mode, err) }; tables := []string{"owner","settings","auth_sessions","vaults","projects","sessions","agent_runs","messages","notes","promote_ops","direct_ops","review_pending","review_items","review_events","backup_runs"}; for _, name := range tables { var n int; if err := d.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&n); err != nil || n != 1 { t.Fatalf("table %s: %d %v", name, n, err) } }; if _, err := Open(context.Background(), filepath.Join(t.TempDir(), "other.sqlite")); err != nil { t.Fatal(err) } }
func TestSessionScopeCheck(t *testing.T) { d, _ := Open(context.Background(), filepath.Join(t.TempDir(), "x.sqlite")); defer d.Close(); _, err := d.Exec(`INSERT INTO sessions(id,home,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('s','project','active','p','m','{}','{}','t','x','x')`); if err == nil { t.Fatal("invalid project scope accepted") } }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/db -v`
Expected: FAIL because `Open` is undefined.

- [ ] **Step 3: Write minimal implementation and the full schema**

```go
// internal/db/db.go
package db
import ("context"; "database/sql"; "embed"; "fmt"; "os"; "path/filepath"; _ "modernc.org/sqlite")
//go:embed migrations/*.sql
var migrations embed.FS
func Open(ctx context.Context, file string) (*sql.DB, error) { if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil { return nil, err }; d, err := sql.Open("sqlite", file); if err != nil { return nil, err }; d.SetMaxOpenConns(1); for _, q := range []string{"PRAGMA foreign_keys=ON", "PRAGMA journal_mode=WAL", "PRAGMA busy_timeout=5000"} { if _, err = d.ExecContext(ctx, q); err != nil { d.Close(); return nil, err } }; if _, err = d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil { d.Close(); return nil, err }; var n int; if err = d.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='001'`).Scan(&n); err != nil { d.Close(); return nil, err }; if n == 0 { b, e := migrations.ReadFile("migrations/001_init.sql"); if e != nil { d.Close(); return nil, e }; tx, e := d.BeginTx(ctx, nil); if e == nil { _, e = tx.ExecContext(ctx, string(b)) }; if e == nil { _, e = tx.ExecContext(ctx, `INSERT INTO schema_migrations VALUES('001',strftime('%Y-%m-%dT%H:%M:%fZ','now'))`) }; if e == nil { e = tx.Commit() } else { tx.Rollback() }; if e != nil { d.Close(); return nil, fmt.Errorf("migration 001: %w", e) } }; return d, nil }
```

```sql
-- internal/db/migrations/001_init.sql
CREATE TABLE owner (id INTEGER PRIMARY KEY CHECK(id=1), password_hash TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE settings (id INTEGER PRIMARY KEY CHECK(id=1), timezone TEXT NOT NULL DEFAULT 'UTC', default_provider TEXT, default_model_id TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
INSERT INTO settings(id,timezone,created_at,updated_at) VALUES(1,'UTC',strftime('%Y-%m-%dT%H:%M:%fZ','now'),strftime('%Y-%m-%dT%H:%M:%fZ','now'));
CREATE TABLE auth_sessions (token_hash TEXT PRIMARY KEY, owner_id INTEGER NOT NULL DEFAULT 1 REFERENCES owner(id) ON DELETE CASCADE, csrf_token TEXT NOT NULL, expires_at TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE vaults (id TEXT PRIMARY KEY, name TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE projects (id TEXT PRIMARY KEY, vault_id TEXT REFERENCES vaults(id) ON DELETE RESTRICT, name TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE sessions (id TEXT PRIMARY KEY, home TEXT NOT NULL CHECK(home IN ('global','vault','project')), vault_id TEXT REFERENCES vaults(id) ON DELETE RESTRICT, project_id TEXT REFERENCES projects(id) ON DELETE RESTRICT, status TEXT NOT NULL CHECK(status IN ('active','terminal')), provider TEXT NOT NULL, model_id TEXT NOT NULL, model_parameters_json TEXT NOT NULL CHECK(json_valid(model_parameters_json)), tool_grants_json TEXT NOT NULL CHECK(json_valid(tool_grants_json)), title TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT, CHECK((home='global' AND vault_id IS NULL AND project_id IS NULL) OR (home='vault' AND vault_id IS NOT NULL AND project_id IS NULL) OR (home='project' AND project_id IS NOT NULL)));
CREATE TRIGGER sessions_project_vault_insert BEFORE INSERT ON sessions WHEN NEW.home='project' AND NOT EXISTS(SELECT 1 FROM projects p WHERE p.id=NEW.project_id AND p.vault_id IS NEW.vault_id) BEGIN SELECT RAISE(ABORT,'session project vault mismatch'); END;
CREATE TRIGGER sessions_immutable_update BEFORE UPDATE OF home,vault_id,project_id,provider,model_id,model_parameters_json ON sessions BEGIN SELECT RAISE(ABORT,'immutable session fields'); END;
CREATE TABLE agent_runs (id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id), request_key TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('queued','running','completed','failed','cancelled')), started_at TEXT, completed_at TEXT, error TEXT, created_at TEXT NOT NULL, UNIQUE(session_id,request_key));
CREATE UNIQUE INDEX one_active_run ON agent_runs(session_id) WHERE status IN ('queued','running');
CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id), run_id TEXT REFERENCES agent_runs(id), sequence INTEGER NOT NULL, role TEXT NOT NULL CHECK(role IN ('system','user','assistant','tool')), content TEXT NOT NULL, tool_calls_json TEXT CHECK(tool_calls_json IS NULL OR json_valid(tool_calls_json)), tool_call_id TEXT, status TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(session_id,sequence));
CREATE TABLE notes (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), relative_path TEXT NOT NULL, content_sha256 TEXT, byte_size INTEGER CHECK(byte_size IS NULL OR byte_size>=0), status TEXT NOT NULL CHECK(status IN ('pending','ready','failed')), origin_session_id TEXT REFERENCES sessions(id), origin_workspace_path TEXT, revision INTEGER NOT NULL DEFAULT 0 CHECK(revision>=0), created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(project_id,relative_path));
CREATE TABLE promote_ops (id TEXT PRIMARY KEY, request_key TEXT NOT NULL UNIQUE, request_fingerprint TEXT NOT NULL, session_id TEXT NOT NULL REFERENCES sessions(id), workspace_path TEXT NOT NULL, target_project_id TEXT NOT NULL REFERENCES projects(id), target_relative_path TEXT NOT NULL, review_mode TEXT NOT NULL CHECK(review_mode IN ('none','whole','bites')), note_id TEXT NOT NULL, frozen_sha256 TEXT, frozen_size INTEGER, status TEXT NOT NULL CHECK(status IN ('accepted','frozen','path_reserved','published_fs','finalized','review_enqueued','completed','failed')), error TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE direct_ops (id TEXT PRIMARY KEY, request_key TEXT NOT NULL UNIQUE, request_fingerprint TEXT NOT NULL, target_project_id TEXT NOT NULL REFERENCES projects(id), target_relative_path TEXT NOT NULL, review_mode TEXT NOT NULL CHECK(review_mode IN ('none','whole','bites')), note_id TEXT NOT NULL, frozen_sha256 TEXT, frozen_size INTEGER, status TEXT NOT NULL CHECK(status IN ('accepted','frozen','path_reserved','published_fs','finalized','review_enqueued','completed','failed')), error TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE review_pending (id TEXT PRIMARY KEY, note_id TEXT NOT NULL REFERENCES notes(id), source_sha256 TEXT NOT NULL, generator_version TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('pending','leased','completed','failed')), attempts INTEGER NOT NULL DEFAULT 0, lease_until TEXT, last_error TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE UNIQUE INDEX review_pending_generation ON review_pending(note_id,source_sha256,generator_version) WHERE status IN ('pending','leased');
CREATE TABLE review_items (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), note_id TEXT NOT NULL REFERENCES notes(id), kind TEXT NOT NULL CHECK(kind IN ('whole','bite')), source_sha256 TEXT NOT NULL, source_revision INTEGER NOT NULL, prompt TEXT NOT NULL, answer TEXT, generation_id TEXT REFERENCES review_pending(id), ordinal INTEGER, stage INTEGER NOT NULL DEFAULT 0, due_at TEXT NOT NULL, interval_days REAL NOT NULL DEFAULT 0, ease_factor REAL NOT NULL DEFAULT 2.5 CHECK(ease_factor>=1.3), reps INTEGER NOT NULL DEFAULT 0, lapses INTEGER NOT NULL DEFAULT 0, last_reviewed_at TEXT, row_version INTEGER NOT NULL DEFAULT 1, status TEXT NOT NULL CHECK(status IN ('active','suspended','retired')), scheduler_version TEXT NOT NULL, CHECK((kind='whole' AND generation_id IS NULL AND ordinal IS NULL) OR (kind='bite' AND answer IS NOT NULL AND generation_id IS NOT NULL AND ordinal IS NOT NULL)), UNIQUE(generation_id,ordinal));
CREATE UNIQUE INDEX review_whole_active ON review_items(note_id,source_revision) WHERE kind='whole' AND status='active';
CREATE TABLE review_events (id TEXT PRIMARY KEY, review_item_id TEXT NOT NULL REFERENCES review_items(id), request_key TEXT NOT NULL UNIQUE, rating TEXT NOT NULL CHECK(rating IN ('again','hard','good','easy')), previous_state_json TEXT NOT NULL CHECK(json_valid(previous_state_json)), resulting_state_json TEXT NOT NULL CHECK(json_valid(resulting_state_json)), scheduler_version TEXT NOT NULL, reviewed_at TEXT NOT NULL, duration_ms INTEGER CHECK(duration_ms IS NULL OR duration_ms>=0));
CREATE TABLE backup_runs (id TEXT PRIMARY KEY, status TEXT NOT NULL CHECK(status IN ('running','succeeded','failed')), cutoff_at TEXT NOT NULL, local_path TEXT, object_key TEXT, manifest_hash TEXT, started_at TEXT NOT NULL, completed_at TEXT, error TEXT);
```

Run: `go get modernc.org/sqlite && go mod tidy`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/db -v`
Expected: PASS with WAL enabled, all 15 domain tables present, and invalid scope rejected.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/db
git commit -m "feat: add SQLite schema and migrations"
```

### Task 4: Argon2id passwords and session tokens

**Files:**
- Create: `internal/auth/password.go`, `internal/auth/password_test.go`, `internal/auth/session.go`, `internal/auth/session_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: plaintext password, PHC hash string, cryptographic random source
- Produces: `func auth.HashPassword(string) (string, error)`; `func auth.CheckPassword(string, string) bool`; `func auth.NewSessionToken() string`; `func auth.TokenHash(string) string`

- [ ] **Step 1: Write the failing tests**

```go
package auth
import ("strings"; "testing")
func TestPasswordRoundTrip(t *testing.T) { h, err := HashPassword("correct horse battery staple"); if err != nil { t.Fatal(err) }; if !strings.HasPrefix(h,"$argon2id$") || !CheckPassword(h,"correct horse battery staple") || CheckPassword(h,"wrong") { t.Fatal("argon2id verification failed") } }
func TestSessionTokens(t *testing.T) { a, b := NewSessionToken(), NewSessionToken(); if a == "" || a == b || TokenHash(a) == a || TokenHash(a) != TokenHash(a) { t.Fatal("unsafe token") } }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/auth -run 'TestPasswordRoundTrip|TestSessionTokens' -v`
Expected: FAIL with undefined functions.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/auth/password.go
package auth
import ("crypto/rand"; "crypto/subtle"; "encoding/base64"; "fmt"; "strings"; "golang.org/x/crypto/argon2")
func HashPassword(pw string) (string,error) { salt := make([]byte,16); if _,err:=rand.Read(salt); err!=nil{return "",err}; sum:=argon2.IDKey([]byte(pw),salt,3,64*1024,2,32); return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s",base64.RawStdEncoding.EncodeToString(salt),base64.RawStdEncoding.EncodeToString(sum)),nil }
func CheckPassword(encoded,pw string) bool { p:=strings.Split(encoded,"$"); if len(p)!=6 || p[1]!="argon2id" || p[2]!="v=19" || p[3]!="m=65536,t=3,p=2" {return false}; salt,e1:=base64.RawStdEncoding.DecodeString(p[4]); want,e2:=base64.RawStdEncoding.DecodeString(p[5]); if e1!=nil||e2!=nil{return false}; got:=argon2.IDKey([]byte(pw),salt,3,64*1024,2,32); return subtle.ConstantTimeCompare(got,want)==1 }

// internal/auth/session.go
package auth
import ("crypto/rand"; "crypto/sha256"; "encoding/base64"; "encoding/hex")
func NewSessionToken() string { b:=make([]byte,32); if _,err:=rand.Read(b); err!=nil { panic(err) }; return base64.RawURLEncoding.EncodeToString(b) }
func TokenHash(token string) string { h:=sha256.Sum256([]byte(token)); return hex.EncodeToString(h[:]) }
```

Run: `go get golang.org/x/crypto && go mod tidy`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/auth -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/auth
git commit -m "feat: add argon2id authentication primitives"
```

### Task 5: Owner bootstrap, login, logout, me, and CSRF

**Files:**
- Create: `internal/auth/bootstrap.go`, `internal/auth/bootstrap_test.go`, `internal/auth/csrf.go`, `internal/httpapi/auth_handlers.go`, `internal/httpapi/middleware.go`, `internal/httpapi/auth_handlers_test.go`

**Interfaces:**
- Consumes: `*sql.DB`, `clock.Clock`, bootstrap token, `pa_session` and `pa_csrf` cookies, `X-CSRF-Token`
- Produces: `func auth.Bootstrap(context.Context, *sql.DB, string, string, string, time.Time) error`; `func httpapi.AuthRoutes(*http.ServeMux, AuthDeps)`; `func httpapi.RequireAuth(*sql.DB, http.Handler) http.Handler`; `func httpapi.RequireCSRF(http.Handler) http.Handler`; endpoints `GET /api/v1/setup/status`, `POST /api/v1/setup/bootstrap`, `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`

- [ ] **Step 1: Write the failing integration test**

```go
package httpapi
import ("bytes"; "context"; "encoding/json"; "net/http"; "net/http/httptest"; "path/filepath"; "testing"; "time"; "github.com/rigasyahrul/personal-agent/internal/clock"; database "github.com/rigasyahrul/personal-agent/internal/db")
func TestBootstrapLoginMeLogoutAndCSRF(t *testing.T) { d,err:=database.Open(context.Background(),filepath.Join(t.TempDir(),"a.sqlite")); if err!=nil{t.Fatal(err)}; defer d.Close(); mux:=http.NewServeMux(); AuthRoutes(mux,AuthDeps{DB:d,Clock:&clock.FakeClock{T:time.Unix(1000,0).UTC()},BootstrapToken:"secret",SecureCookies:false}); post:=func(path,body string,cookies []*http.Cookie,csrf string)*httptest.ResponseRecorder{ r:=httptest.NewRequest(http.MethodPost,path,bytes.NewBufferString(body)); r.Header.Set("Content-Type","application/json"); if csrf!=""{r.Header.Set("X-CSRF-Token",csrf)}; for _,c:=range cookies{r.AddCookie(c)}; w:=httptest.NewRecorder(); mux.ServeHTTP(w,r); return w }; w:=post("/api/v1/setup/bootstrap",`{"token":"secret","password":"long-enough-password"}`,nil,""); if w.Code!=201{t.Fatal(w.Code,w.Body.String())}; if post("/api/v1/setup/bootstrap",`{"token":"secret","password":"another-long-password"}`,nil,"").Code!=409{t.Fatal("second bootstrap accepted")}; w=post("/api/v1/auth/login",`{"password":"long-enough-password"}`,nil,""); if w.Code!=204{t.Fatal(w.Code,w.Body.String())}; cookies:=w.Result().Cookies(); var csrf string; for _,c:=range cookies{if c.Name=="pa_csrf"{csrf=c.Value}}; r:=httptest.NewRequest(http.MethodGet,"/api/v1/auth/me",nil); for _,c:=range cookies{r.AddCookie(c)}; me:=httptest.NewRecorder(); mux.ServeHTTP(me,r); var out map[string]any; json.NewDecoder(me.Body).Decode(&out); if me.Code!=200 || out["owner"]!=true{t.Fatal(me.Code,out)}; if post("/api/v1/auth/logout",`{}`,cookies,"wrong").Code!=403{t.Fatal("csrf bypass")}; if post("/api/v1/auth/logout",`{}`,cookies,csrf).Code!=204{t.Fatal("logout failed")} }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestBootstrapLoginMeLogoutAndCSRF -v`
Expected: FAIL because `AuthRoutes` and `AuthDeps` are undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/auth/bootstrap.go
package auth
import ("context"; "database/sql"; "errors"; "time")
var ErrBootstrapped=errors.New("owner already bootstrapped"); var ErrBootstrapToken=errors.New("invalid bootstrap token")
func Bootstrap(ctx context.Context,d *sql.DB,configured,provided,pw string,now time.Time) error { var n int; if err:=d.QueryRowContext(ctx,"SELECT count(*) FROM owner").Scan(&n);err!=nil{return err}; if n!=0{return ErrBootstrapped}; if configured==""||provided!=configured{return ErrBootstrapToken}; h,err:=HashPassword(pw);if err!=nil{return err}; _,err=d.ExecContext(ctx,"INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,?,?,?)",h,now.Format(time.RFC3339Nano),now.Format(time.RFC3339Nano));return err }

// internal/auth/csrf.go
package auth
import "crypto/subtle"
func ValidCSRF(cookie,header string) bool { return cookie!=""&&len(cookie)==len(header)&&subtle.ConstantTimeCompare([]byte(cookie),[]byte(header))==1 }

// internal/httpapi/middleware.go
package httpapi
import ("context"; "database/sql"; "net/http"; "time"; "github.com/rigasyahrul/personal-agent/internal/auth")
type ownerKey struct{}
func RequireAuth(d *sql.DB,next http.Handler) http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){c,e:=r.Cookie("pa_session");if e!=nil{http.Error(w,"unauthorized",401);return};var expiry string;if d.QueryRowContext(r.Context(),"SELECT expires_at FROM auth_sessions WHERE token_hash=?",auth.TokenHash(c.Value)).Scan(&expiry)!=nil{http.Error(w,"unauthorized",401);return};t,e:=time.Parse(time.RFC3339Nano,expiry);if e!=nil||!t.After(time.Now().UTC()){http.Error(w,"unauthorized",401);return};next.ServeHTTP(w,r.WithContext(context.WithValue(r.Context(),ownerKey{},true)))})}
func RequireCSRF(next http.Handler) http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){c,e:=r.Cookie("pa_csrf");if e!=nil||!auth.ValidCSRF(c.Value,r.Header.Get("X-CSRF-Token")){http.Error(w,"csrf",403);return};next.ServeHTTP(w,r)})}
```

```go
// internal/httpapi/auth_handlers.go
package httpapi
import ("database/sql"; "encoding/json"; "errors"; "net/http"; "time"; "github.com/rigasyahrul/personal-agent/internal/auth"; "github.com/rigasyahrul/personal-agent/internal/clock")
type AuthDeps struct{DB *sql.DB;Clock clock.Clock;BootstrapToken string;SecureCookies bool}
func AuthRoutes(m *http.ServeMux,d AuthDeps){ write:=func(w http.ResponseWriter,v any){w.Header().Set("Content-Type","application/json");json.NewEncoder(w).Encode(v)}; m.HandleFunc("GET /api/v1/setup/status",func(w http.ResponseWriter,r *http.Request){var n int;d.DB.QueryRowContext(r.Context(),"SELECT count(*) FROM owner").Scan(&n);write(w,map[string]bool{"bootstrapped":n==1})});m.HandleFunc("POST /api/v1/setup/bootstrap",func(w http.ResponseWriter,r *http.Request){var in struct{Token,Password string};if json.NewDecoder(r.Body).Decode(&in)!=nil||len(in.Password)<12{http.Error(w,"invalid request",400);return};err:=auth.Bootstrap(r.Context(),d.DB,d.BootstrapToken,in.Token,in.Password,d.Clock.Now());if errors.Is(err,auth.ErrBootstrapped){http.Error(w,"already bootstrapped",409);return};if err!=nil{http.Error(w,"forbidden",403);return};w.WriteHeader(201)});m.HandleFunc("POST /api/v1/auth/login",func(w http.ResponseWriter,r *http.Request){var in struct{Password string};json.NewDecoder(r.Body).Decode(&in);var h string;if d.DB.QueryRowContext(r.Context(),"SELECT password_hash FROM owner WHERE id=1").Scan(&h)!=nil||!auth.CheckPassword(h,in.Password){http.Error(w,"unauthorized",401);return};token,csrf:=auth.NewSessionToken(),auth.NewSessionToken();now:=d.Clock.Now();_,e:=d.DB.ExecContext(r.Context(),"INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)",auth.TokenHash(token),csrf,now.Add(30*24*time.Hour).Format(time.RFC3339Nano),now.Format(time.RFC3339Nano));if e!=nil{http.Error(w,"session",500);return};http.SetCookie(w,&http.Cookie{Name:"pa_session",Value:token,Path:"/",HttpOnly:true,Secure:d.SecureCookies,SameSite:http.SameSiteLaxMode});http.SetCookie(w,&http.Cookie{Name:"pa_csrf",Value:csrf,Path:"/",Secure:d.SecureCookies,SameSite:http.SameSiteLaxMode});w.WriteHeader(204)});m.Handle("GET /api/v1/auth/me",RequireAuth(d.DB,http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){write(w,map[string]bool{"owner":true})})));m.Handle("POST /api/v1/auth/logout",RequireAuth(d.DB,RequireCSRF(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){c,_:=r.Cookie("pa_session");d.DB.ExecContext(r.Context(),"DELETE FROM auth_sessions WHERE token_hash=?",auth.TokenHash(c.Value));http.SetCookie(w,&http.Cookie{Name:"pa_session",Path:"/",MaxAge:-1,HttpOnly:true,Secure:d.SecureCookies});w.WriteHeader(204)})))}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/auth ./internal/httpapi -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth internal/httpapi
git commit -m "feat: add owner bootstrap and browser auth"
```

### Task 6: Health, setup status, empty Home, and server wiring

**Files:**
- Create: `internal/httpapi/health.go`, `internal/httpapi/server.go`, `internal/httpapi/server_test.go`, `internal/app/app.go`, `cmd/personal-agent/main.go`

**Interfaces:**
- Consumes: `config.Config`, `*sql.DB`, `clock.Clock`, static `http.FileSystem`
- Produces: `func httpapi.New(ServerDeps) http.Handler`; `func app.New(context.Context, config.Config) (*app.App, error)`; `func (*app.App) Handler() http.Handler`; endpoints `GET /health`, `GET /api/v1/home`

- [ ] **Step 1: Write the failing test**

```go
package httpapi
import ("context"; "encoding/json"; "net/http"; "net/http/httptest"; "path/filepath"; "testing"; "time"; "github.com/rigasyahrul/personal-agent/internal/clock"; database "github.com/rigasyahrul/personal-agent/internal/db")
func TestHealthSetupAndEmptyHome(t *testing.T){dir:=t.TempDir();d,err:=database.Open(context.Background(),filepath.Join(dir,"db","a.sqlite"));if err!=nil{t.Fatal(err)};defer d.Close();h:=New(ServerDeps{DB:d,DataDir:dir,Clock:&clock.FakeClock{T:time.Unix(0,0)},BootstrapToken:"x"});for _,path:=range []string{"/health","/api/v1/setup/status","/api/v1/home"}{r:=httptest.NewRequest(http.MethodGet,path,nil);w:=httptest.NewRecorder();h.ServeHTTP(w,r);if w.Code!=200{t.Fatalf("%s: %d %s",path,w.Code,w.Body.String())};var v map[string]any;if json.NewDecoder(w.Body).Decode(&v)!=nil{t.Fatalf("%s not JSON",path)}}}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestHealthSetupAndEmptyHome -v`
Expected: FAIL because `New` and `ServerDeps` are undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/httpapi/health.go
package httpapi
import ("encoding/json"; "net/http"; "os"; "path/filepath")
func healthHandler(dataDir string)http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){p:=filepath.Join(dataDir,".health-write");err:=os.WriteFile(p,[]byte("ok"),0600);if err==nil{err=os.Remove(p)};w.Header().Set("Content-Type","application/json");if err!=nil{w.WriteHeader(503)};json.NewEncoder(w).Encode(map[string]any{"ok":err==nil,"storage_writable":err==nil})}}

// internal/httpapi/server.go
package httpapi
import ("database/sql"; "encoding/json"; "net/http"; "github.com/rigasyahrul/personal-agent/internal/clock")
type ServerDeps struct{DB *sql.DB;DataDir string;Clock clock.Clock;BootstrapToken string;SecureCookies bool;Static http.FileSystem}
func New(d ServerDeps)http.Handler{m:=http.NewServeMux();AuthRoutes(m,AuthDeps{DB:d.DB,Clock:d.Clock,BootstrapToken:d.BootstrapToken,SecureCookies:d.SecureCookies});m.Handle("GET /health",healthHandler(d.DataDir));m.HandleFunc("GET /api/v1/home",func(w http.ResponseWriter,r *http.Request){w.Header().Set("Content-Type","application/json");json.NewEncoder(w).Encode(map[string]any{"projects":[]any{},"due_count":0,"last_project":nil})});if d.Static!=nil{m.Handle("GET /",http.FileServer(d.Static))};return m}

// internal/app/app.go
package app
import ("context"; "net/http"; "path/filepath"; "github.com/rigasyahrul/personal-agent/internal/clock"; "github.com/rigasyahrul/personal-agent/internal/config"; database "github.com/rigasyahrul/personal-agent/internal/db"; "github.com/rigasyahrul/personal-agent/internal/httpapi")
type App struct{handler http.Handler}
func New(ctx context.Context,c config.Config)(*App,error){d,e:=database.Open(ctx,filepath.Join(c.DataDir,"db","personal-agent.sqlite"));if e!=nil{return nil,e};return &App{handler:httpapi.New(httpapi.ServerDeps{DB:d,DataDir:c.DataDir,Clock:clock.RealClock{},BootstrapToken:c.BootstrapToken,SecureCookies:c.SecureCookies,Static:http.Dir("web")})},nil}
func(a *App)Handler()http.Handler{return a.handler}

// cmd/personal-agent/main.go
package main
import("context";"log";"net/http";"github.com/rigasyahrul/personal-agent/internal/app";"github.com/rigasyahrul/personal-agent/internal/config")
func main(){c,e:=config.Load();if e!=nil{log.Fatal(e)};a,e:=app.New(context.Background(),c);if e!=nil{log.Fatal(e)};log.Printf("listening on %s",c.Addr);log.Fatal(http.ListenAndServe(c.Addr,a.Handler()))}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi internal/app cmd/personal-agent
git commit -m "feat: serve health setup and empty home APIs"
```

### Task 7: Static empty Home shell

**Files:**
- Create: `web/index.html`, `web/css/app.css`, `web/js/api.js`, `web/js/router.js`, `web/js/app.js`, `web/js/pages/home.js`, `internal/httpapi/static_test.go`

**Interfaces:**
- Consumes: `GET /api/v1/home`, `GET /health`, `GET /api/v1/setup/status`
- Produces: static SPA at `GET /`; `api.get(path)`; `home.render(element)`

- [ ] **Step 1: Write the failing test**

```go
package httpapi
import("net/http";"net/http/httptest";"os";"strings";"testing")
func TestStaticShell(t *testing.T){if _,err:=os.Stat("../../web/index.html");err!=nil{t.Fatal(err)};h:=http.FileServer(http.Dir("../../web"));r:=httptest.NewRequest("GET","/",nil);w:=httptest.NewRecorder();h.ServeHTTP(w,r);if w.Code!=200||!strings.Contains(w.Body.String(),"Personal Agent")||!strings.Contains(w.Body.String(),`type="module"`){t.Fatalf("%d %s",w.Code,w.Body.String())}}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestStaticShell -v`
Expected: FAIL because `web/index.html` does not exist.

- [ ] **Step 3: Write minimal implementation**

```html
<!-- web/index.html -->
<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Personal Agent</title><link rel="stylesheet" href="/css/app.css"></head><body><header><h1>Personal Agent</h1><span id="health">Checking storage…</span></header><main id="app" aria-live="polite">Loading…</main><script type="module" src="/js/app.js"></script></body></html>
```

```css
/* web/css/app.css */
:root{font-family:system-ui,sans-serif;color:#172033;background:#f6f7fb}body{max-width:64rem;margin:auto;padding:1rem}header{display:flex;justify-content:space-between;align-items:center}main{background:white;border-radius:.75rem;padding:2rem;box-shadow:0 2px 12px #0001}.muted{color:#667085}button{padding:.65rem 1rem;border:0;border-radius:.5rem;background:#2457d6;color:white}
```

```js
// web/js/api.js
export async function get(path){const response=await fetch(path,{headers:{Accept:'application/json'}});if(!response.ok)throw new Error(`${response.status} ${await response.text()}`);return response.json()}

// web/js/router.js
export function route(){return location.pathname==='/'?'home':'not-found'}

// web/js/pages/home.js
import{get}from'../api.js';
export async function render(root){const home=await get('/api/v1/home');root.innerHTML=home.projects.length?'<h2>Projects</h2>':`<h2>Your learning home</h2><p class="muted">No projects yet. Create your first project to begin collecting notes and sessions.</p><button disabled>Create project (next phase)</button>`}

// web/js/app.js
import{get}from'./api.js';import{route}from'./router.js';import{render as home}from'./pages/home.js';
const root=document.querySelector('#app');Promise.all([get('/health'),get('/api/v1/setup/status')]).then(([health,setup])=>{document.querySelector('#health').textContent=`Storage ${health.storage_writable?'ready':'unavailable'} · ${setup.bootstrapped?'Owner ready':'Setup required'}`}).catch(e=>document.querySelector('#health').textContent=e.message);if(route()==='home')home(root).catch(e=>root.textContent=e.message);else root.textContent='Not found';
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... && test "$(grep -c '/api/v1/home' web/js/pages/home.js)" -eq 1`
Expected: PASS and the Home module contains exactly one Home API reference.

- [ ] **Step 5: Commit**

```bash
git add web internal/httpapi/static_test.go
git commit -m "feat: add empty Home web shell"
```


**Errata (required):** Include setup bootstrap form and login form in the static shell; gate Home on `GET /api/v1/auth/me`. Create `web/js/pages/setup.js` and wire settings stub page that loads timezone from `GET /api/v1/settings` once authenticated. Create `internal/store/settings.go` + `settings_handlers.go` if not already added in Task 5–6.

### Task 8: Container deployment and Go 1.24 orb setup

**Files:**
- Create: `deploy/Dockerfile`, `deploy/docker-compose.yml`, `deploy/Caddyfile`, `deploy/.env.example`, `README.md`, `deploy/deploy_test.go`
- Modify: `.agents/setup`

**Interfaces:**
- Consumes: module build, port `8080`, `PA_DATA_DIR`, `BOOTSTRAP_TOKEN`, `PA_DOMAIN`
- Produces: `personal-agent` container, persistent `pa-data` volume, optional-domain Caddy reverse proxy, documented local startup, orb installation of Go 1.24+

- [ ] **Step 1: Write the failing deployment test**

```go
package deploy_test
import("os";"strings";"testing")
func TestDeploymentFiles(t *testing.T){checks:=map[string][]string{"Dockerfile":{"golang:1.24","CMD"},"docker-compose.yml":{"personal-agent:","caddy:","pa-data:"},"Caddyfile":{"reverse_proxy personal-agent:8080"},".env.example":{"BOOTSTRAP_TOKEN=","PA_DOMAIN="},"../README.md":{"docker compose"},"../.agents/setup":{"1.24"}};for file,need:=range checks{b,err:=os.ReadFile(file);if err!=nil{t.Fatal(file,err)};for _,s:=range need{if !strings.Contains(string(b),s){t.Errorf("%s missing %q",file,s)}}}}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./deploy -run TestDeploymentFiles -v`
Expected: FAIL because deployment files do not exist.

- [ ] **Step 3: Write minimal implementation**

```dockerfile
# deploy/Dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /personal-agent ./cmd/personal-agent
FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=build /personal-agent /usr/local/bin/personal-agent
COPY web ./web
EXPOSE 8080
CMD ["personal-agent"]
```

```yaml
# deploy/docker-compose.yml
services:
  personal-agent:
    build: {context: .., dockerfile: deploy/Dockerfile}
    environment:
      PA_DATA_DIR: /data
      PA_ADDR: :8080
      PA_SECURE_COOKIES: ${PA_SECURE_COOKIES:-true}
      BOOTSTRAP_TOKEN: ${BOOTSTRAP_TOKEN:?set BOOTSTRAP_TOKEN}
    volumes: [pa-data:/data]
    ports: ["8080:8080"]
    restart: unless-stopped
  caddy:
    image: caddy:2-alpine
    profiles: [domain]
    environment: [PA_DOMAIN]
    ports: ["80:80", "443:443"]
    volumes: ["./Caddyfile:/etc/caddy/Caddyfile:ro", "caddy-data:/data"]
    depends_on: [personal-agent]
    restart: unless-stopped
volumes: {pa-data: {}, caddy-data: {}}
```

```caddyfile
# deploy/Caddyfile
{$PA_DOMAIN:localhost} {
    reverse_proxy personal-agent:8080
}
```

```dotenv
# deploy/.env.example
BOOTSTRAP_TOKEN=replace-with-at-least-32-random-characters
PA_DOMAIN=agent.example.com
PA_SECURE_COOKIES=true
```

```markdown
<!-- README.md -->
# Personal Agent

Self-hosted, single-owner learning dashboard. Requires Go 1.24+ or Docker.

## Development

Run `make test`, then set `BOOTSTRAP_TOKEN` and run `make run`. Open port 8080, bootstrap the owner once, and log in. Runtime data defaults to `./data`.

## Docker Compose

Copy `deploy/.env.example` to `deploy/.env`, replace the bootstrap token, then run `docker compose -f deploy/docker-compose.yml up --build`. For a real domain, set `PA_DOMAIN`, keep secure cookies enabled, and add `--profile domain`; Caddy terminates HTTPS. Model and backup credentials belong in environment variables, never the database.
```

```bash
# .agents/setup
#!/usr/bin/env bash
set -euo pipefail
required=1.24.0
current=$(go version 2>/dev/null | sed -n 's/.*go\([0-9.]*\).*/\1/p' || true)
if [ -z "$current" ] || [ "$(printf '%s\n' "$required" "$current" | sort -V | head -n1)" != "$required" ]; then
  curl -fsSLo /tmp/go.tgz https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
  sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
fi
export PATH=/usr/local/go/bin:$PATH
go mod download
```

- [ ] **Step 4: Run focused and phase verification**

Run: `go test ./deploy -v && go test ./... && docker compose -f deploy/docker-compose.yml config >/dev/null`
Expected: all Go tests PASS and Compose configuration validates. Then run `BOOTSTRAP_TOKEN=01234567890123456789012345678901 PA_SECURE_COOKIES=false timeout 10s go run ./cmd/personal-agent` and expect the log to contain `listening on :8080` before timeout.

- [ ] **Step 5: Commit**

```bash
git add deploy README.md .agents/setup
git commit -m "chore: add single-host deployment skeleton"
```

## Phase self-check

- Spec §3: single Go API, local SQLite/files data directory, static browser client, and Compose/Caddy topology are established.
- Spec §5 and §11: all entities are represented in migration 001; owner bootstrap, Argon2id, hashed sessions, secure cookie settings, and CSRF are test-covered; session scope has a shape CHECK and project-vault trigger.
- Spec §8 Home and §9 F0: writable storage, bootstrap state, empty dashboard DTO, and first-run shell are served.
- Spec §14 phase 1: `go test ./...` is green; `go run ./cmd/personal-agent` serves health, setup, auth, empty Home, and static shell; SQLite WAL migration and Compose deployment exist.


## Phase 2: Projects + source tree

### Task 9: Derive and create project layout

**Files:**
- Create: `internal/layout/layout.go`
- Test: `internal/layout/layout_test.go`

**Interfaces:**
- Consumes: trusted `dataDir`, database IDs, and `internal/fsroot` containment guarantees from Phase 1.
- Produces: `type SessionHome string`, `ProjectRoot(dataDir, vaultID, projectID string) string`, `SourceDir(projectRoot string) string`, `SessionWorkspace(dataDir string, home SessionHome, vaultID, projectID, sessionID string) string`, and `EnsureProjectDirs(dataDir, vaultID, projectID string) error`.

- [ ] **Step 1: Write the failing test**

```go
package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectLayoutAndCreation(t *testing.T) {
	d := t.TempDir()
	if got, want := ProjectRoot(d, "", "p1"), filepath.Join(d, "files", "global", "projects", "p1"); got != want { t.Fatalf("root=%q want %q", got, want) }
	if got, want := ProjectRoot(d, "v1", "p1"), filepath.Join(d, "files", "vaults", "v1", "projects", "p1"); got != want { t.Fatalf("vault root=%q want %q", got, want) }
	if got := SourceDir(ProjectRoot(d, "", "p1")); got != filepath.Join(d, "files", "global", "projects", "p1", "source") { t.Fatal(got) }
	if got := SessionWorkspace(d, SessionHome("project"), "v1", "p1", "s1"); got != filepath.Join(d, "files", "vaults", "v1", "projects", "p1", "sessions", "s1") { t.Fatal(got) }
	if err := EnsureProjectDirs(d, "v1", "p1"); err != nil { t.Fatal(err) }
	for _, name := range []string{"source", "memory", "soul"} {
		if st, err := os.Stat(filepath.Join(ProjectRoot(d, "v1", "p1"), name)); err != nil || !st.IsDir() { t.Fatalf("%s: %v", name, err) }
	}
}

func TestSessionWorkspaceAllHomes(t *testing.T) {
	d := t.TempDir()
	cases := map[SessionHome]string{
		"global": filepath.Join(d, "files", "global", "sessions", "s"),
		"vault": filepath.Join(d, "files", "vaults", "v", "sessions", "s"),
		"project": filepath.Join(d, "files", "global", "projects", "p", "sessions", "s"),
	}
	for home, want := range cases {
		if got := SessionWorkspace(d, home, "v", "p", "s"); got != want { t.Errorf("%s: %q want %q", home, got, want) }
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/layout -run 'TestProjectLayout|TestSessionWorkspace' -v`
Expected: FAIL because package `internal/layout` and its functions do not exist.

- [ ] **Step 3: Write minimal implementation**

```go
package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

type SessionHome string

func ProjectRoot(dataDir, vaultID, projectID string) string {
	if vaultID == "" { return filepath.Join(dataDir, "files", "global", "projects", projectID) }
	return filepath.Join(dataDir, "files", "vaults", vaultID, "projects", projectID)
}

func SourceDir(projectRoot string) string { return filepath.Join(projectRoot, "source") }

func SessionWorkspace(dataDir string, home SessionHome, vaultID, projectID, sessionID string) string {
	switch home {
	case "global": return filepath.Join(dataDir, "files", "global", "sessions", sessionID)
	case "vault": return filepath.Join(dataDir, "files", "vaults", vaultID, "sessions", sessionID)
	case "project": return filepath.Join(ProjectRoot(dataDir, vaultID, projectID), "sessions", sessionID)
	default: panic(fmt.Sprintf("invalid session home %q", home))
	}
}

func EnsureProjectDirs(dataDir, vaultID, projectID string) error {
	root := ProjectRoot(dataDir, vaultID, projectID)
	for _, name := range []string{"source", "memory", "soul"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil { return fmt.Errorf("create project %s: %w", name, err) }
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/layout -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/layout/layout.go internal/layout/layout_test.go
git commit -m "feat: add rooted project layout"
```

### Task 10: Persist vaults and projects with immutable placement

**Files:**
- Create: `internal/store/vaults.go`
- Create: `internal/store/projects.go`
- Test: `internal/store/projects_test.go`

**Interfaces:**
- Consumes: `*sql.DB`, `ids.NewID()`, `clock.Clock`, the Phase 1 `vaults`/`projects` schema, and `layout.EnsureProjectDirs`.
- Produces: `NewVaultStore(db, clock) *VaultStore`, `Create(ctx, name) (domain.Vault, error)`, `List(ctx) ([]domain.Vault, error)`, `NewProjectStore(db, dataDir, clock) *ProjectStore`, `Create(ctx, name, vaultID) (domain.Project, error)`, `List(ctx) ([]domain.Project, error)`, and `Get(ctx, id) (domain.Project, error)`.

- [ ] **Step 1: Write the failing test**

```go
package store_test

import (
	"context"
	"testing"
	"time"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	dbtest "github.com/rigasyahrul/personal-agent/internal/db"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

func TestVaultAndProjectCRUD(t *testing.T) {
	d := t.TempDir(); database, err := dbtest.Open(d); if err != nil { t.Fatal(err) }; defer database.Close()
	c := &clock.FakeClock{T: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)}
	vs := store.NewVaultStore(database, c); v, err := vs.Create(context.Background(), "Learning"); if err != nil { t.Fatal(err) }
	ps := store.NewProjectStore(database, d, c); p, err := ps.Create(context.Background(), "Go", v.ID); if err != nil { t.Fatal(err) }
	if p.VaultID != v.ID || p.Name != "Go" { t.Fatalf("%+v", p) }
	if _, err := ps.Create(context.Background(), "Bad", "missing"); err == nil { t.Fatal("expected unknown vault failure") }
	got, err := ps.Get(context.Background(), p.ID); if err != nil || got.VaultID != v.ID { t.Fatalf("%+v %v", got, err) }
	list, err := ps.List(context.Background()); if err != nil || len(list) != 1 { t.Fatalf("%+v %v", list, err) }
	if err := layout.EnsureProjectDirs(d, "", p.ID); err != nil { t.Fatal(err) }
	if _, err := database.Exec(`UPDATE projects SET vault_id=NULL WHERE id=?`, p.ID); err == nil { t.Fatal("vault placement must be immutable") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run TestVaultAndProjectCRUD -v`
Expected: FAIL because the stores and constructors are undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/store/vaults.go
package store

import (
	"context"
	"database/sql"
	"strings"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
)

type VaultStore struct { db *sql.DB; clock clock.Clock }
func NewVaultStore(db *sql.DB, c clock.Clock) *VaultStore { return &VaultStore{db: db, clock: c} }
func (s *VaultStore) Create(ctx context.Context, name string) (domain.Vault, error) {
	name = strings.TrimSpace(name); v := domain.Vault{ID: ids.NewID(), Name: name, CreatedAt: s.clock.Now().UTC(), UpdatedAt: s.clock.Now().UTC()}
	if name == "" { return domain.Vault{}, ErrInvalid }
	_, err := s.db.ExecContext(ctx, `INSERT INTO vaults(id,name,created_at,updated_at) VALUES(?,?,?,?)`, v.ID,v.Name,v.CreatedAt,v.UpdatedAt); return v, err
}
func (s *VaultStore) List(ctx context.Context) ([]domain.Vault, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,created_at,updated_at FROM vaults ORDER BY name,id`); if err != nil { return nil, err }; defer rows.Close()
	out := []domain.Vault{}; for rows.Next() { var v domain.Vault; if err := rows.Scan(&v.ID,&v.Name,&v.CreatedAt,&v.UpdatedAt); err != nil { return nil,err }; out=append(out,v) }; return out,rows.Err()
}
```

```go
// internal/store/projects.go
package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

var ErrInvalid = errors.New("invalid input")
type ProjectStore struct { db *sql.DB; dataDir string; clock clock.Clock }
func NewProjectStore(db *sql.DB, dataDir string, c clock.Clock) *ProjectStore { return &ProjectStore{db:db,dataDir:dataDir,clock:c} }
func (s *ProjectStore) Create(ctx context.Context, name, vaultID string) (domain.Project, error) {
	name=strings.TrimSpace(name); if name=="" { return domain.Project{},ErrInvalid }
	if vaultID!="" { var n int; if err:=s.db.QueryRowContext(ctx,`SELECT count(*) FROM vaults WHERE id=?`,vaultID).Scan(&n); err!=nil||n!=1 { return domain.Project{},ErrInvalid } }
	now:=s.clock.Now().UTC(); p:=domain.Project{ID:ids.NewID(),VaultID:vaultID,Name:name,CreatedAt:now,UpdatedAt:now}
	tx,err:=s.db.BeginTx(ctx,nil); if err!=nil{return domain.Project{},err}; defer tx.Rollback()
	var nullable any; if vaultID!="" { nullable=vaultID }
	if _,err=tx.ExecContext(ctx,`INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES(?,?,?,?,?)`,p.ID,nullable,p.Name,p.CreatedAt,p.UpdatedAt);err!=nil{return domain.Project{},err}
	if err=layout.EnsureProjectDirs(s.dataDir,vaultID,p.ID);err!=nil{return domain.Project{},err}; if err=tx.Commit();err!=nil{return domain.Project{},err}; return p,nil
}
func scanProject(row interface{ Scan(...any) error }) (domain.Project,error) { var p domain.Project; var v sql.NullString; err:=row.Scan(&p.ID,&v,&p.Name,&p.CreatedAt,&p.UpdatedAt); if v.Valid { p.VaultID=v.String }; return p,err }
func (s *ProjectStore) Get(ctx context.Context,id string)(domain.Project,error){return scanProject(s.db.QueryRowContext(ctx,`SELECT id,vault_id,name,created_at,updated_at FROM projects WHERE id=?`,id))}
func (s *ProjectStore) List(ctx context.Context)([]domain.Project,error){rows,err:=s.db.QueryContext(ctx,`SELECT id,vault_id,name,created_at,updated_at FROM projects ORDER BY updated_at DESC,id`);if err!=nil{return nil,err};defer rows.Close();out:=[]domain.Project{};for rows.Next(){p,e:=scanProject(rows);if e!=nil{return nil,e};out=append(out,p)};return out,rows.Err()}
```

Add `Vault` and `Project` structs with the fields used above to `internal/domain/models.go`, and add an immutable-placement trigger to `001_init.sql`:

```sql
CREATE TRIGGER projects_vault_immutable BEFORE UPDATE OF vault_id ON projects
WHEN NEW.vault_id IS NOT OLD.vault_id BEGIN SELECT RAISE(ABORT, 'project vault_id is immutable'); END;
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store -run TestVaultAndProjectCRUD -v`
Expected: PASS with one persisted project and rejected placement update.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/models.go internal/db/migrations/001_init.sql internal/store/vaults.go internal/store/projects.go internal/store/projects_test.go
git commit -m "feat: persist vaults and projects"
```

### Task 11: Serve vault, project, overview, and home data

**Files:**
- Create: `internal/httpapi/project_handlers.go`
- Modify: `internal/httpapi/server.go`
- Test: `internal/httpapi/project_handlers_test.go`

**Interfaces:**
- Consumes: authenticated/CSRF-protected Phase 1 mux, `VaultStore`, `ProjectStore`, and SQLite aggregate tables.
- Produces: `GET/POST /api/v1/vaults`, `GET/POST /api/v1/projects`, `GET /api/v1/projects/{id}`, and `GET /api/v1/home`; project DTO fields are `id`, `vault_id`, `vault_name`, `name`, `note_count`, `session_count`, and `due_count`.

- [ ] **Step 1: Write the failing test**

```go
package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"github.com/rigasyahrul/personal-agent/internal/httpapi"
)

func TestProjectAPI(t *testing.T) {
	s := newAuthenticatedTestServer(t)
	w := httptest.NewRecorder(); r := httptest.NewRequest(http.MethodPost,"/api/v1/projects",strings.NewReader(`{"name":"Go","vault_id":null}`)); r.Header.Set("Content-Type","application/json"); addAuthAndCSRF(r)
	s.ServeHTTP(w,r); if w.Code!=http.StatusCreated { t.Fatalf("%d %s",w.Code,w.Body.String()) }
	w=httptest.NewRecorder(); r=httptest.NewRequest(http.MethodGet,"/api/v1/home",nil); addAuth(r); s.ServeHTTP(w,r)
	if w.Code!=http.StatusOK || !strings.Contains(w.Body.String(),`"note_count":0`) || !strings.Contains(w.Body.String(),`"session_count":0`) || !strings.Contains(w.Body.String(),`"due_count":0`) { t.Fatalf("%d %s",w.Code,w.Body.String()) }
	_ = httpapi.ProjectDTO{}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestProjectAPI -v`
Expected: FAIL because the routes and `ProjectDTO` are absent.

- [ ] **Step 3: Write minimal implementation**

```go
package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type ProjectDTO struct { ID string `json:"id"`; VaultID string `json:"vault_id,omitempty"`; VaultName string `json:"vault_name,omitempty"`; Name string `json:"name"`; NoteCount int `json:"note_count"`; SessionCount int `json:"session_count"`; DueCount int `json:"due_count"` }
type ProjectHandlers struct { Vaults *store.VaultStore; Projects *store.ProjectStore }
func jsonOut(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
func (h ProjectHandlers) vaults(w http.ResponseWriter,r *http.Request){if r.Method==http.MethodGet{v,e:=h.Vaults.List(r.Context());if e!=nil{http.Error(w,e.Error(),500);return};jsonOut(w,200,v);return};var in struct{Name string `json:"name"`};if json.NewDecoder(r.Body).Decode(&in)!=nil{http.Error(w,"invalid json",400);return};v,e:=h.Vaults.Create(r.Context(),in.Name);if e!=nil{http.Error(w,e.Error(),400);return};jsonOut(w,201,v)}
func (h ProjectHandlers) projects(w http.ResponseWriter,r *http.Request){if r.Method==http.MethodGet{ps,e:=h.Projects.List(r.Context());if e!=nil{http.Error(w,e.Error(),500);return};out:=make([]ProjectDTO,len(ps));for i,p:=range ps{out[i]=ProjectDTO{ID:p.ID,VaultID:p.VaultID,Name:p.Name}};jsonOut(w,200,out);return};var in struct{Name string `json:"name"`;VaultID *string `json:"vault_id"`};if json.NewDecoder(r.Body).Decode(&in)!=nil{http.Error(w,"invalid json",400);return};v:="";if in.VaultID!=nil{v=*in.VaultID};p,e:=h.Projects.Create(r.Context(),in.Name,v);if e!=nil{http.Error(w,e.Error(),400);return};jsonOut(w,201,ProjectDTO{ID:p.ID,VaultID:p.VaultID,Name:p.Name})}
func (h ProjectHandlers) project(w http.ResponseWriter,r *http.Request){p,e:=h.Projects.Get(r.Context(),r.PathValue("id"));if e!=nil{http.Error(w,"project not found",404);return};jsonOut(w,200,ProjectDTO{ID:p.ID,VaultID:p.VaultID,Name:p.Name})}
func (h ProjectHandlers) home(w http.ResponseWriter,r *http.Request){ps,e:=h.Projects.List(r.Context());if e!=nil{http.Error(w,e.Error(),500);return};out:=make([]ProjectDTO,len(ps));for i,p:=range ps{out[i]=ProjectDTO{ID:p.ID,VaultID:p.VaultID,Name:p.Name}};jsonOut(w,200,map[string]any{"projects":out,"due_count":0,"generated_at":time.Now().UTC()})}
func (h ProjectHandlers) Register(mux *http.ServeMux){mux.HandleFunc("GET /api/v1/vaults",h.vaults);mux.HandleFunc("POST /api/v1/vaults",h.vaults);mux.HandleFunc("GET /api/v1/projects",h.projects);mux.HandleFunc("POST /api/v1/projects",h.projects);mux.HandleFunc("GET /api/v1/projects/{id}",h.project);mux.HandleFunc("GET /api/v1/home",h.home)}
```

Wire `ProjectHandlers.Register(mux)` in `server.go` inside the existing auth/CSRF middleware. Keep aggregate values zero until their tables gain rows; Task 12 supplies real note counts and Phase 3 supplies sessions.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/httpapi -run TestProjectAPI -v`
Expected: PASS and the home DTO includes all three count fields.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/server.go internal/httpapi/project_handlers.go internal/httpapi/project_handlers_test.go
git commit -m "feat: expose project and home APIs"
```

### Task 12: Index and integrity-check the source tree

**Files:**
- Create: `internal/store/notes.go`
- Create: `internal/httpapi/note_handlers.go`
- Modify: `internal/httpapi/project_handlers.go`
- Test: `internal/store/notes_test.go`
- Test: `internal/httpapi/note_handlers_test.go`

**Interfaces:**
- Consumes: `paths.ValidateRelPath`, `layout.SourceDir`, ready Note rows, project placement, and rooted filesystem reads.
- Produces: `NewNoteStore(db, dataDir) *NoteStore`, `Tree(ctx, projectID) ([]TreeEntry, error)`, `Get(ctx, noteID) (NoteDocument, error)`, `POST /api/v1/projects/{id}/folders`, `GET /api/v1/projects/{id}/tree`, and `GET /api/v1/notes/{id}`; URLs accept note IDs only.

- [ ] **Step 1: Write the failing test**

```go
package store_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

func TestNoteTreeAndIntegrity(t *testing.T) {
	d,db,p := projectFixture(t); body:=[]byte("# Safe\n"); path:=filepath.Join(projectSource(t,d,p),"guide","safe.md")
	if err:=os.MkdirAll(filepath.Dir(path),0700);err!=nil{t.Fatal(err)};if err:=os.WriteFile(path,body,0600);err!=nil{t.Fatal(err)};sum:=sha256.Sum256(body)
	if _,err:=db.Exec(`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n1',?,?,?,?,'ready',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,p.ID,"guide/safe.md",fmt.Sprintf("%x",sum),len(body));err!=nil{t.Fatal(err)}
	s:=store.NewNoteStore(db,d); tree,err:=s.Tree(context.Background(),p.ID);if err!=nil||len(tree)!=2||tree[1].NoteID!="n1"{t.Fatalf("%+v %v",tree,err)}
	doc,err:=s.Get(context.Background(),"n1");if err!=nil||string(doc.Body)!=string(body){t.Fatalf("%+v %v",doc,err)}
	if err:=os.WriteFile(path,[]byte("changed"),0600);err!=nil{t.Fatal(err)};_,err=s.Get(context.Background(),"n1");if !errors.Is(err,store.ErrIntegrity){t.Fatalf("got %v",err)}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run TestNoteTreeAndIntegrity -v`
Expected: FAIL because `NoteStore`, `TreeEntry`, and `ErrIntegrity` are undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

var ErrIntegrity=errors.New("note integrity check failed")
type TreeEntry struct{Path string `json:"path"`;Kind string `json:"kind"`;NoteID string `json:"note_id,omitempty"`}
type NoteDocument struct{ID string `json:"id"`;ProjectID string `json:"project_id"`;RelativePath string `json:"relative_path"`;ContentSHA256 string `json:"content_sha256"`;ByteSize int64 `json:"byte_size"`;Revision int `json:"revision"`;Body []byte `json:"body"`}
type NoteStore struct{db *sql.DB;dataDir string}
func NewNoteStore(db *sql.DB,dataDir string)*NoteStore{return &NoteStore{db:db,dataDir:dataDir}}
func (s *NoteStore) projectRoot(ctx context.Context,id string)(string,error){var v sql.NullString;if err:=s.db.QueryRowContext(ctx,`SELECT vault_id FROM projects WHERE id=?`,id).Scan(&v);err!=nil{return "",err};return layout.ProjectRoot(s.dataDir,v.String,id),nil}
func (s *NoteStore) Tree(ctx context.Context,projectID string)([]TreeEntry,error){root,e:=s.projectRoot(ctx,projectID);if e!=nil{return nil,e};ids:=map[string]string{};rows,e:=s.db.QueryContext(ctx,`SELECT relative_path,id FROM notes WHERE project_id=? AND status='ready'`,projectID);if e!=nil{return nil,e};defer rows.Close();for rows.Next(){var p,id string;if e=rows.Scan(&p,&id);e!=nil{return nil,e};ids[p]=id};out:=[]TreeEntry{};e=filepath.WalkDir(layout.SourceDir(root),func(p string,d os.DirEntry,e error)error{if e!=nil{return e};if p==layout.SourceDir(root){return nil};if d.Type()&os.ModeSymlink!=0{return ErrIntegrity};rel,e:=filepath.Rel(layout.SourceDir(root),p);if e!=nil{return e};rel=filepath.ToSlash(rel);if d.IsDir(){out=append(out,TreeEntry{Path:rel,Kind:"folder"});return nil};if filepath.Ext(rel)!=".md"||!d.Type().IsRegular(){return ErrIntegrity};id,ok:=ids[rel];if !ok{return ErrIntegrity};out=append(out,TreeEntry{Path:rel,Kind:"note",NoteID:id});return nil});sort.Slice(out,func(i,j int)bool{return out[i].Path<out[j].Path});return out,e}
func (s *NoteStore) Get(ctx context.Context,id string)(NoteDocument,error){var n NoteDocument;var vault sql.NullString;e:=s.db.QueryRowContext(ctx,`SELECT n.id,n.project_id,n.relative_path,n.content_sha256,n.byte_size,n.revision,p.vault_id FROM notes n JOIN projects p ON p.id=n.project_id WHERE n.id=? AND n.status='ready'`,id).Scan(&n.ID,&n.ProjectID,&n.RelativePath,&n.ContentSHA256,&n.ByteSize,&n.Revision,&vault);if e!=nil{return n,e};if strings.Contains(n.RelativePath,"\\"){return n,ErrIntegrity};b,e:=os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(s.dataDir,vault.String,n.ProjectID)),filepath.FromSlash(n.RelativePath)));if e!=nil{return n,ErrIntegrity};sum:=fmt.Sprintf("%x",sha256.Sum256(b));if sum!=n.ContentSHA256||int64(len(b))!=n.ByteSize{return n,ErrIntegrity};n.Body=b;return n,nil}
```

Implement handlers using the existing JSON helper. Folder creation must call `paths.ValidateRelPath`, reject a final `.md` component, and call a rooted `Mkdir` under the selected project's `source`; return `409` when it exists and never follow a symlink. Tree returns `[]TreeEntry`. Note read maps `ErrIntegrity` to `409 {"code":"integrity_error"}` and encodes `body` as a JSON string (not base64). Add note counts with `COUNT(*) FILTER (WHERE status='ready')` to project/home DTO queries.

```go
func (h NoteHandlers) get(w http.ResponseWriter,r *http.Request){n,e:=h.Notes.Get(r.Context(),r.PathValue("id"));if errors.Is(e,store.ErrIntegrity){jsonOut(w,409,map[string]string{"code":"integrity_error"});return};if e!=nil{http.Error(w,"note not found",404);return};jsonOut(w,200,map[string]any{"id":n.ID,"project_id":n.ProjectID,"relative_path":n.RelativePath,"content_sha256":n.ContentSHA256,"byte_size":n.ByteSize,"revision":n.Revision,"body":string(n.Body)})}
func (h NoteHandlers) tree(w http.ResponseWriter,r *http.Request){v,e:=h.Notes.Tree(r.Context(),r.PathValue("id"));if e!=nil{http.Error(w,e.Error(),409);return};jsonOut(w,200,v)}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store ./internal/httpapi -run 'TestNoteTreeAndIntegrity|TestNoteHandlers' -v`
Expected: PASS; changing the file after indexing produces HTTP 409 and no metadata rewrite.

- [ ] **Step 5: Commit**

```bash
git add internal/store/notes.go internal/store/notes_test.go internal/httpapi/note_handlers.go internal/httpapi/note_handlers_test.go internal/httpapi/project_handlers.go internal/httpapi/server.go
git commit -m "feat: browse integrity-checked source notes"
```

### Task 13: Publish direct Markdown through the shared machine

**Files:**
- Create: `internal/store/direct.go`
- Create: `internal/publish/machine.go`
- Create: `internal/publish/recover.go`
- Create: `internal/httpapi/note_handlers.go`
- Test: `internal/publish/machine_test.go`

**Interfaces:**
- Consumes: `PublishInput` exactly as locked, `paths.ValidateRelPath`, `paths.MaxMarkdownBytes`, rooted no-clobber filesystem operations, Note/direct-operation schema, and `clock.Clock`.
- Produces: `type Machine struct { DB *sql.DB; DataDir string; Clock clock.Clock }`, `Run(ctx, in PublishInput) (opStatus string, noteID string, err error)`, `RecoverAll(ctx) error`, and `POST /api/v1/projects/{id}/direct-notes`. Promote kind remains rejected until Phase 5 rather than pretending success.

- [ ] **Step 1: Write the failing test**

```go
package publish_test

func TestDirectCreateIsIdempotentAndNeverOverwrites(t *testing.T) {
	d,db,p,c:=publishFixture(t);m:=publish.Machine{DB:db,DataDir:d,Clock:c}
	in:=publish.PublishInput{OpID:"op1",RequestKey:"key1",RequestFingerprint:"fp1",Kind:"direct",Body:[]byte("# One\n"),TargetProjectID:p.ID,TargetRelPath:"guide/one.md",ReviewMode:domain.ReviewMode("none"),NoteID:"n1"}
	status,noteID,err:=m.Run(context.Background(),in);if err!=nil||status!="completed"||noteID!="n1"{t.Fatalf("%s %s %v",status,noteID,err)}
	status,noteID,err=m.Run(context.Background(),in);if err!=nil||status!="completed"||noteID!="n1"{t.Fatalf("retry: %s %s %v",status,noteID,err)}
	other:=in;other.OpID="op2";other.RequestKey="key2";other.RequestFingerprint="fp2";other.NoteID="n2";other.Body=[]byte("overwrite")
	if _,_,err=m.Run(context.Background(),other);!errors.Is(err,publish.ErrConflict){t.Fatalf("got %v",err)}
	b,err:=os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(d,p.VaultID,p.ID)),"guide","one.md"));if err!=nil||string(b)!="# One\n"{t.Fatalf("%q %v",b,err)}
}

func TestDirectCreateValidation(t *testing.T){d,db,p,c:=publishFixture(t);m:=publish.Machine{DB:db,DataDir:d,Clock:c};for _,path:=range []string{"../x.md","x.txt","memory/x.md"}{_,_,err:=m.Run(context.Background(),publish.PublishInput{OpID:"o"+path,RequestKey:"k"+path,RequestFingerprint:"f"+path,Kind:"direct",Body:[]byte("x"),TargetProjectID:p.ID,TargetRelPath:path,ReviewMode:"none",NoteID:"n"+path});if err==nil{t.Fatalf("accepted %q",path)}}}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/publish -run 'TestDirectCreate' -v`
Expected: FAIL because the publish machine does not exist.

- [ ] **Step 3: Write minimal implementation**

```go
package publish

import("context";"crypto/sha256";"database/sql";"errors";"fmt";"os";"path/filepath";"strings";"github.com/rigasyahrul/personal-agent/internal/clock";"github.com/rigasyahrul/personal-agent/internal/domain";"github.com/rigasyahrul/personal-agent/internal/layout";pathcheck "github.com/rigasyahrul/personal-agent/internal/paths")
var ErrConflict=errors.New("publication conflict")
type PublishInput struct{OpID,RequestKey,RequestFingerprint string;Kind string;SessionID string;WorkspacePath string;Body []byte;TargetProjectID,TargetRelPath string;ReviewMode domain.ReviewMode;NoteID string}
type Machine struct{DB *sql.DB;DataDir string;Clock clock.Clock}
func (m *Machine) Run(ctx context.Context,in PublishInput)(string,string,error){
	if in.Kind!="direct"{return "",in.NoteID,errors.New("promote is not enabled")};clean,e:=pathcheck.ValidateRelPath(in.TargetRelPath);if e!=nil||!strings.HasSuffix(clean,".md")||len(in.Body)>pathcheck.MaxMarkdownBytes||clean=="memory"||strings.HasPrefix(clean,"memory/")||clean=="soul"||strings.HasPrefix(clean,"soul/"){return "",in.NoteID,ErrInvalid}
	var oldFP,status,note string;e=m.DB.QueryRowContext(ctx,`SELECT request_fingerprint,status,note_id FROM direct_ops WHERE request_key=?`,in.RequestKey).Scan(&oldFP,&status,&note);if e==nil{if oldFP!=in.RequestFingerprint{return status,note,ErrConflict};return status,note,nil};if !errors.Is(e,sql.ErrNoRows){return "",in.NoteID,e}
	var vault sql.NullString;if e=m.DB.QueryRowContext(ctx,`SELECT vault_id FROM projects WHERE id=?`,in.TargetProjectID).Scan(&vault);e!=nil{return "",in.NoteID,e};now:=m.Clock.Now().UTC();_,e=m.DB.ExecContext(ctx,`INSERT INTO direct_ops(id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,'accepted',?,?)`,in.OpID,in.RequestKey,in.RequestFingerprint,in.TargetProjectID,clean,in.ReviewMode,in.NoteID,now,now);if e!=nil{return "",in.NoteID,e}
	stage:=filepath.Join(m.DataDir,"staging","direct",in.OpID,"body.md");if e=os.MkdirAll(filepath.Dir(stage),0700);e==nil{e=os.WriteFile(stage,in.Body,0600)};if e!=nil{return "",in.NoteID,e};sum:=fmt.Sprintf("%x",sha256.Sum256(in.Body));_,e=m.DB.ExecContext(ctx,`UPDATE direct_ops SET status='frozen',frozen_sha256=?,frozen_size=?,updated_at=? WHERE id=?`,sum,len(in.Body),now,in.OpID);if e!=nil{return "",in.NoteID,e}
	tx,e:=m.DB.BeginTx(ctx,nil);if e!=nil{return "",in.NoteID,e};defer tx.Rollback();_,e=tx.ExecContext(ctx,`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES(?,?,?,'',0,'pending',0,?,?)`,in.NoteID,in.TargetProjectID,clean,now,now);if e!=nil{return "",in.NoteID,ErrConflict};if _,e=tx.ExecContext(ctx,`UPDATE direct_ops SET status='path_reserved',updated_at=? WHERE id=?`,now,in.OpID);e!=nil{return "",in.NoteID,e};if e=tx.Commit();e!=nil{return "",in.NoteID,e}
	dst:=filepath.Join(layout.SourceDir(layout.ProjectRoot(m.DataDir,vault.String,in.TargetProjectID)),filepath.FromSlash(clean));if e=os.MkdirAll(filepath.Dir(dst),0700);e!=nil{return "",in.NoteID,e};f,e:=os.OpenFile(dst,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if errors.Is(e,os.ErrExist){return "",in.NoteID,ErrConflict};if e!=nil{return "",in.NoteID,e};if _,e=f.Write(in.Body);e==nil{e=f.Sync()};closeErr:=f.Close();if e==nil{e=closeErr};if e!=nil{return "",in.NoteID,e};_,e=m.DB.ExecContext(ctx,`UPDATE direct_ops SET status='published_fs',updated_at=? WHERE id=?`,now,in.OpID);if e!=nil{return "",in.NoteID,e}
	tx,e=m.DB.BeginTx(ctx,nil);if e!=nil{return "",in.NoteID,e};defer tx.Rollback();if _,e=tx.ExecContext(ctx,`UPDATE notes SET content_sha256=?,byte_size=?,status='ready',revision=1,updated_at=? WHERE id=?`,sum,len(in.Body),now,in.NoteID);e!=nil{return "",in.NoteID,e};if in.ReviewMode=="whole"{_,e=tx.ExecContext(ctx,`INSERT INTO review_items(id,project_id,note_id,kind,source_sha256,source_revision,prompt,stage,due_at,interval_days,ease_factor,reps,lapses,row_version,status,scheduler_version) VALUES(lower(hex(randomblob(16))),?,?, 'whole',?,1,'Review this note',0,?,0,2.5,0,0,1,'active','sm2-lite-v1')`,in.TargetProjectID,in.NoteID,sum,now)}else if in.ReviewMode=="bites"{_,e=tx.ExecContext(ctx,`INSERT INTO review_pending(id,note_id,source_sha256,generator_version,status,attempts) VALUES(lower(hex(randomblob(16))),?,?,'bites-v1','pending',0)`,in.NoteID,sum)};if e!=nil{return "",in.NoteID,e};if _,e=tx.ExecContext(ctx,`UPDATE direct_ops SET status='completed',updated_at=? WHERE id=?`,now,in.OpID);e!=nil{return "",in.NoteID,e};if e=tx.Commit();e!=nil{return "",in.NoteID,e};_ = os.RemoveAll(filepath.Dir(stage));return "completed",in.NoteID,nil
}
```

`RecoverAll` queries direct operations not in `completed,failed`, reconstructs immutable input from staging plus operation columns, and resumes from the recorded status; each transition must first reconcile the corresponding DB/FS artifact. The direct HTTP handler requires `Idempotency-Key`, computes `request_fingerprint = sha256(project_id + NUL + path + NUL + review_mode + NUL + body)`, preallocates UUID operation/note IDs, and maps `ErrConflict` to 409. Empty keys, invalid review modes, non-`.md`, bodies over 1 MiB, and unsafe paths return 400. Never overwrite an existing destination.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/publish ./internal/httpapi -run 'TestDirectCreate|TestDirectNoteHandler' -v`
Expected: PASS; retry returns the same note, conflicting fingerprints/paths return 409, and original bytes remain unchanged.

- [ ] **Step 5: Commit**

```bash
git add internal/store/direct.go internal/publish/machine.go internal/publish/recover.go internal/publish/machine_test.go internal/httpapi/note_handlers.go internal/httpapi/note_handlers_test.go internal/httpapi/server.go
git commit -m "feat: publish direct source notes"
```

### Task 14: Add projects and notes screens

**Files:**
- Modify: `web/index.html`
- Modify: `web/css/app.css`
- Modify: `web/js/api.js`
- Modify: `web/js/router.js`
- Modify: `web/js/app.js`
- Create: `web/js/pages/home.js`
- Create: `web/js/pages/project.js`
- Create: `web/js/pages/notes.js`
- Test: `web/js/pages/projects.test.mjs`

**Interfaces:**
- Consumes: project/home/tree/note/folder/direct-note endpoints from Tasks 11–13 and CSRF support from Phase 1.
- Produces: project cards and new-project form, project overview, source tree, note-by-ID viewer, and new Markdown file/folder forms; no edit/delete UI.

- [ ] **Step 1: Write the failing test**

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import { projectCard } from './home.js';
import { treeRows, directPayload } from './notes.js';

test('project card includes counts and vault badge', () => {
  const html = projectCard({id:'p1',name:'Go',vault_name:'Learning',note_count:2,session_count:0,due_count:1});
  assert.match(html, /Go/); assert.match(html, /Learning/); assert.match(html, /2 notes/); assert.match(html, /1 due/);
});
test('tree links notes by id, never by path', () => {
  const html = treeRows('p1', [{kind:'note',path:'guide/a.md',note_id:'n1'}]);
  assert.match(html, /notes\/n1/); assert.doesNotMatch(html, /notes\/guide/);
});
test('direct create preserves locked request fields', () => {
  assert.deepEqual(directPayload('guide/a.md','# A','whole'), {relative_path:'guide/a.md',body:'# A',review_mode:'whole'});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test web/js/pages/projects.test.mjs`
Expected: FAIL because the page modules and exports do not exist.

- [ ] **Step 3: Write minimal implementation**

```js
// web/js/api.js
export async function api(path, options={}) {
  const headers = {'Accept':'application/json', ...(options.headers||{})};
  if (options.body && typeof options.body !== 'string') { headers['Content-Type']='application/json'; options.body=JSON.stringify(options.body); }
  if (!['GET','HEAD'].includes(options.method||'GET')) headers['X-CSRF-Token']=document.cookie.split('; ').find(v=>v.startsWith('pa_csrf='))?.split('=')[1]||'';
  const response=await fetch(`/api/v1${path}`,{...options,headers}); const data=await response.json().catch(()=>({}));
  if(!response.ok) throw new Error(data.message||data.code||`HTTP ${response.status}`); return data;
}
```

```js
// web/js/pages/home.js
import {api} from '../api.js';
const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
export const projectCard=p=>`<article class="card"><a href="#/projects/${encodeURIComponent(p.id)}"><h2>${esc(p.name)}</h2></a>${p.vault_name?`<span class="badge">${esc(p.vault_name)}</span>`:''}<p>${p.note_count||0} notes · ${p.session_count||0} sessions · ${p.due_count||0} due</p></article>`;
export async function renderHome(root){const data=await api('/home');root.innerHTML=`<header><h1>Projects</h1><button id="new-project">New project</button></header><form id="project-form" hidden><label>Name <input name="name" required></label><label>Vault ID (optional) <input name="vault_id"></label><button>Create</button></form><section class="cards">${data.projects.map(projectCard).join('')||'<p>Create your first project.</p>'}</section>`;root.querySelector('#new-project').onclick=()=>root.querySelector('#project-form').hidden=false;root.querySelector('#project-form').onsubmit=async e=>{e.preventDefault();const f=new FormData(e.currentTarget);const vault=f.get('vault_id').trim();const p=await api('/projects',{method:'POST',body:{name:f.get('name'),vault_id:vault||null}});location.hash=`#/projects/${p.id}`}}
```

```js
// web/js/pages/notes.js
import {api} from '../api.js';
const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
export const treeRows=(projectID,entries)=>entries.map(e=>e.kind==='folder'?`<li>📁 ${esc(e.path)}</li>`:`<li>📄 <a href="#/projects/${encodeURIComponent(projectID)}/notes/${encodeURIComponent(e.note_id)}">${esc(e.path)}</a></li>`).join('');
export const directPayload=(relative_path,body,review_mode)=>({relative_path,body,review_mode});
export async function renderNotes(root,projectID,noteID){const tree=await api(`/projects/${encodeURIComponent(projectID)}/tree`);let viewer='';if(noteID){const n=await api(`/notes/${encodeURIComponent(noteID)}`);viewer=`<article><h2>${esc(n.relative_path)}</h2><pre>${esc(n.body)}</pre></article>`}root.innerHTML=`<nav><a href="#/projects/${projectID}">Overview</a></nav><h1>Notes</h1><ul>${treeRows(projectID,tree)}</ul>${viewer}<details><summary>New folder</summary><form id="folder"><input name="path" maxlength="512" required><button>Create folder</button></form></details><details><summary>New file</summary><form id="file"><input name="path" maxlength="512" pattern=".*\\.md$" required><textarea name="body" maxlength="1048576" required></textarea><select name="review"><option value="none">No review</option><option value="whole">Whole note</option><option value="bites">Bites</option></select><button>Create file</button></form></details>`;root.querySelector('#folder').onsubmit=async e=>{e.preventDefault();await api(`/projects/${projectID}/folders`,{method:'POST',body:{relative_path:new FormData(e.currentTarget).get('path')}});return renderNotes(root,projectID)};root.querySelector('#file').onsubmit=async e=>{e.preventDefault();const f=new FormData(e.currentTarget);const result=await api(`/projects/${projectID}/direct-notes`,{method:'POST',headers:{'Idempotency-Key':crypto.randomUUID()},body:directPayload(f.get('path'),f.get('body'),f.get('review'))});location.hash=`#/projects/${projectID}/notes/${result.note_id}`}}
```

```js
// web/js/pages/project.js
import {api} from '../api.js';
export async function renderProject(root,id){const p=await api(`/projects/${encodeURIComponent(id)}`);root.innerHTML=`<h1>${p.name}</h1><nav><a href="#/projects/${id}/notes">Notes</a></nav><section class="cards"><p>${p.note_count||0} notes</p><p>${p.session_count||0} sessions</p><p>${p.due_count||0} due</p></section><a class="button" href="#/projects/${id}/notes">New source file</a>`}
```

Update `router.js` to match `#/projects/:id/notes/:noteID`, `#/projects/:id/notes`, and `#/projects/:id` in that order, calling the exports above; default to `renderHome`. Update `app.js` to rerender on `hashchange`, add `<main id="app">` in `index.html`, and add responsive `.cards`, `.card`, `.badge`, form, tree, and `pre { white-space: pre-wrap }` rules to `app.css`. Render Markdown as escaped text in this phase; a vetted Markdown renderer arrives only through `components/markdown.js`, never through raw `innerHTML` from note bodies.

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test web/js/pages/projects.test.mjs && go test ./...`
Expected: PASS; tree links contain `note_id`, direct file input is `.md`-constrained and 1 MiB-limited, and all Go tests remain green.

- [ ] **Step 5: Commit**

```bash
git add web/index.html web/css/app.css web/js/api.js web/js/router.js web/js/app.js web/js/pages/home.js web/js/pages/project.js web/js/pages/notes.js web/js/pages/projects.test.mjs
git commit -m "feat: add projects and source notes UI"
```

## Phase self-check

- Covers spec §4 project layout (`source/`, reserved `memory/`/`soul/`) and rooted project placement.
- Covers §5 Vault, immutable Project placement, ready Note metadata, stable note identity, and integrity checking.
- Covers §6 direct publication through `publish.Machine`, `.md`/1 MiB/path limits, idempotency, and 409 no-clobber behavior.
- Covers §8 Home, Projects, overview, Notes tree/view, and new file/folder UI without edit/delete.
- Covers §9 F1 create project, F5 direct source file, and F6 browse by `note_id` with integrity errors.


## Phase 3: Sessions + chat + agent run

### Task 15: Store project sessions and create their workspaces

**Model errata:** `Create` accepts provider/model_id only if present in the configured model list passed into the store/service (from config). Persist immutable provider/model_id/model_parameters_json.


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
    conn := testutil.OpenDB(t, data)
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
    ss := &store.SessionStore{DB: testutil.OpenDB(t, data), DataDir: data, Now: time.Now}
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
    conn := func() *sql.DB { db, _ := testutil.TempDB(t); return db }()
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
    data := t.TempDir(); conn := testutil.OpenDB(t, data); seedVaultProject(t, conn, "v1", "p1")
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

**Runner errata:** Implement Canonical `agent` types exactly (`ChatRequest` with optional Tools, `Runner` with `Provider`, `Clock`, `DataDir`, `execute`). Tests use `testutil.OpenDB` / `TempDB`.


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
    r := &agent.Runner{Sessions:&store.SessionStore{DB:conn}, Messages:&store.MessageStore{DB:conn,Now:time.Now}, Runs:&store.RunStore{DB:conn,Now:time.Now}, Provider:map[string]agent.Provider{"openai":fake}}
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
type Runner struct { Sessions *store.SessionStore; Messages *store.MessageStore; Runs *store.RunStore; Provider Provider // single provider v1 }
func (r *Runner) Start(ctx context.Context, sessionID, requestKey, userMessage string) (string,error) {
    run,created,err:=r.Runs.CreateOrGet(ctx,sessionID,requestKey); if err!=nil{return "",err}; if !created{return run.ID,nil}
    session,err:=r.Sessions.Get(ctx,sessionID); if err!=nil{return run.ID,err}
    if _,err=r.Messages.Append(ctx,sessionID,&run.ID,"user",userMessage,"complete"); err!=nil { _=r.Runs.SetStatus(ctx,run.ID,"failed",err.Error()); return run.ID,err }
    _=r.Runs.SetStatus(ctx,run.ID,"running","")
    history,err:=r.Messages.List(ctx,sessionID); if err!=nil{return run.ID,err}; req:=ChatRequest{Model:session.ModelID}
    if err=json.Unmarshal([]byte(session.ModelParametersJSON),&req.Parameters); err!=nil{return run.ID,err}; for _,m:=range history { req.Messages=append(req.Messages,ChatMessage{Role:m.Role,Content:m.Content}) }
    provider,ok:=r.Provider[session.Provider]; if !ok { err=fmt.Errorf("provider %q unavailable",session.Provider) } else { var response ChatResponse; response,err=provider.Chat(ctx,req); if err==nil { _,err=r.Messages.Append(ctx,sessionID,&run.ID,"assistant",response.Content,"complete") } }
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

**Model errata:** Expose `GET /api/v1/models`. Session create handler validates body.provider/model_id against config models before store create.


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

**Model errata:** New-session form uses `<select>` options from `GET /api/v1/models`, not free-text provider/model fields (optional advanced text only if models list empty is not allowed — show setup CTA instead).


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

**Runner errata:** Extend Task 18 `Runner` / Canonical `ChatRequest`. `ToolCall.Arguments` is a JSON object string. Use `r.Provider` (singular). Call `r.execute` as on Runner. Do not invent a second ChatRequest type.


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
type ToolCall struct { ID string `json:"id"`; Name string `json:"name"`; Arguments string `json:"arguments"` } // JSON object string; canonical
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


## Phase 5: Promote + review

### Task 25: Recoverable promote publication machine

**Machine errata:** Modify Task 13 `publish.Machine.Run` for `Kind=="promote"` as well as direct. Tables: `promote_ops` / `direct_ops`. Vault via `loadProjectVault`. Exact status chain from lock. Snippets with empty vault hard-codes or wrong table names are void.


**Files:**
- Create: `internal/store/promote.go`
- Modify: `internal/publish/machine.go` (created in Task 13)
- Create: `internal/publish/machine_test.go`

**Interfaces:**
- Extends Task 13 `publish.Machine` for `Kind=promote` (keep direct support).
- **Interfaces:**
- Consumes: `publish.PublishInput`, `layout.SessionWorkspace`, `layout.ProjectRoot`, `paths.ValidateRelPath`, `fsroot` rooted file operations, `ids.NewID`, and the Phase 1 schema.
- Produces: `(*publish.Machine).Run(context.Context, publish.PublishInput) (string, string, error)`, durable exact transitions `accepted → frozen → path_reserved → published_fs → finalized → review_enqueued → completed`, `publish.ConflictError`, and idempotent operation lookup by request key and fingerprint.

- [ ] **Step 1: Write the failing publication tests**

```go
package publish_test

func TestPromotePublishesOnceAndRejectsChangedFingerprint(t *testing.T) {
	ctx := context.Background()
	db, dataDir, projectID, sessionID := newPublishFixture(t)
	workspace := layout.SessionWorkspace(dataDir, layout.SessionHome("project"), "", projectID, sessionID)
	require.NoError(t, os.MkdirAll(workspace, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "draft.md"), []byte("# frozen\n"), 0o644))
	m := &publish.Machine{DB: db, DataDir: dataDir, Clock: &clock.FakeClock{T: time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)}}
	in := publish.PublishInput{OpID: ids.NewID(), RequestKey: "promote-1", RequestFingerprint: "fp-1", Kind: "promote", SessionID: sessionID, WorkspacePath: "draft.md", TargetProjectID: projectID, TargetRelPath: "guides/frozen.md", ReviewMode: domain.ReviewWhole, NoteID: ids.NewID()}

	status, noteID, err := m.Run(ctx, in)
	require.NoError(t, err)
	require.Equal(t, "completed", status)
	require.Equal(t, in.NoteID, noteID)
	body, err := os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, loadProjectVault(db, projectID), projectID)), "guides/frozen.md"))
	require.NoError(t, err)
	require.Equal(t, "# frozen\n", string(body))

	status, gotNoteID, err := m.Run(ctx, in)
	require.NoError(t, err)
	require.Equal(t, "completed", status)
	require.Equal(t, noteID, gotNoteID)
	var notes, items int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM notes WHERE id=?`, noteID).Scan(&notes))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM review_items WHERE note_id=?`, noteID).Scan(&items))
	require.Equal(t, 1, notes)
	require.Equal(t, 1, items)

	changed := in
	changed.RequestFingerprint = "fp-changed"
	_, _, err = m.Run(ctx, changed)
	var conflict *publish.ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, "idempotency_key_reused", conflict.Code)
}

func TestPromoteRejectsAnotherProjectAndExistingDestination(t *testing.T) {
	ctx := context.Background()
	db, dataDir, projectID, sessionID := newPublishFixture(t)
	m := &publish.Machine{DB: db, DataDir: dataDir, Clock: clock.RealClock{}}
	otherProject := insertProject(t, db, "other")
	in := validPromoteInput(t, dataDir, projectID, sessionID)
	in.TargetProjectID = otherProject
	_, _, err := m.Run(ctx, in)
	require.ErrorContains(t, err, "session project is the only promote target")

	in = validPromoteInput(t, dataDir, projectID, sessionID)
	destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, loadProjectVault(db, projectID), projectID)), filepath.FromSlash(in.TargetRelPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0o755))
	require.NoError(t, os.WriteFile(destination, []byte("keep me"), 0o644))
	_, _, err = m.Run(ctx, in)
	var conflict *publish.ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, "destination_exists", conflict.Code)
	require.Equal(t, "keep me", string(mustRead(t, destination)))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/publish -run 'TestPromote(PublishesOnce|RejectsAnother)' -v`
Expected: FAIL because `Machine.Run` and `ConflictError` do not exist.

- [ ] **Step 3: Implement the minimal durable state machine**

```go
package publish

type ConflictError struct{ Code string }
func (e *ConflictError) Error() string { return e.Code }

func (m *Machine) Run(ctx context.Context, in PublishInput) (string, string, error) {
	if in.if in.Kind != "promote" && in.Kind != "direct" { return "", "", ErrValidation }
	workspacePath, err := paths.ValidateRelPath(in.WorkspacePath); if err != nil { return "", "", err }
	targetPath, err := paths.ValidateRelPath(in.TargetRelPath); if err != nil { return "", "", err }
	if path.Ext(targetPath) != ".md" { return "", "", fmt.Errorf("promotion requires a .md destination") }
	if in.ReviewMode != domain.ReviewNone && in.ReviewMode != domain.ReviewWhole && in.ReviewMode != domain.ReviewBites { return "", "", fmt.Errorf("invalid review mode") }

	tx, err := m.DB.BeginTx(ctx, nil); if err != nil { return "", "", err }
	defer tx.Rollback()
	var existingID, fingerprint, status, noteID string
	err = tx.QueryRowContext(ctx, `SELECT id,request_fingerprint,status,note_id FROM promote_ops WHERE request_key=?`, in.RequestKey).Scan(&existingID, &fingerprint, &status, &noteID)
	if err == nil {
		if fingerprint != in.RequestFingerprint { return "", "", &ConflictError{Code: "idempotency_key_reused"} }
		tx.Commit()
		if status == "completed" || status == "failed" { return status, noteID, nil }
		in.OpID, in.NoteID = existingID, noteID
	} else if !errors.Is(err, sql.ErrNoRows) { return "", "", err
	} else {
		var home, sessionProject, vaultID, sessionStatus string
		if err := tx.QueryRowContext(ctx, `SELECT home,project_id,coalesce(vault_id,''),status FROM sessions WHERE id=?`, in.SessionID).Scan(&home, &sessionProject, &vaultID, &sessionStatus); err != nil { return "", "", err }
		if sessionStatus != "active" { return "", "", fmt.Errorf("session is terminal") }
		if home != "project" || sessionProject != in.TargetProjectID { return "", "", fmt.Errorf("session project is the only promote target") }
		_, err = tx.ExecContext(ctx, `INSERT INTO promote_ops(id,request_key,request_fingerprint,session_id,workspace_path,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,'accepted',?,?)`, in.OpID,in.RequestKey,in.RequestFingerprint,in.SessionID,workspacePath,in.TargetProjectID,targetPath,in.ReviewMode,in.NoteID,m.Clock.Now(),m.Clock.Now())
		if err != nil { return "", "", err }
		if err := tx.Commit(); err != nil { return "", "", err }
	}

	stagingDir := filepath.Join(m.DataDir, "staging", in.OpID)
	stagingFile := filepath.Join(stagingDir, "frozen.md")
	if err := os.MkdirAll(stagingDir, 0o700); err != nil { return m.fail(ctx, in, err) }
	if statusBefore(ctx,m.DB,in.OpID) == "accepted" {
		source := layout.SessionWorkspace(m.DataDir, layout.SessionHome("project"), "", in.TargetProjectID, in.SessionID)
		body, err := readRootedRegular(source, workspacePath, paths.MaxMarkdownBytes); if err != nil { return m.fail(ctx,in,err) }
		if err := writeSync(stagingFile, body); err != nil { return m.fail(ctx,in,err) }
		hash := sha256.Sum256(body)
		if err := transition(ctx,m.DB,in.OpID,"accepted","frozen",hex.EncodeToString(hash[:]),int64(len(body))); err != nil { return "", "", err }
	}
	if statusBefore(ctx,m.DB,in.OpID) == "frozen" {
		_, err := m.DB.ExecContext(ctx, `INSERT INTO notes(id,project_id,relative_path,status,origin_session_id,origin_workspace_path,revision,created_at,updated_at) SELECT note_id,target_project_id,target_relative_path,'pending',session_id,workspace_path,0,?,? FROM promote_ops WHERE id=?`,m.Clock.Now(),m.Clock.Now(),in.OpID)
		if isUnique(err) { return m.failConflict(ctx,in,"destination_exists") }; if err != nil { return m.fail(ctx,in,err) }
		if err := setStatus(ctx,m.DB,in.OpID,"frozen","path_reserved",m.Clock.Now()); err != nil { return "", "", err }
	}
	if statusBefore(ctx,m.DB,in.OpID) == "path_reserved" {
		destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(m.DataDir, loadProjectVault(m.DB, in.TargetProjectID), in.TargetProjectID)),filepath.FromSlash(targetPath))
		if err := atomicNoClobber(stagingFile,destination); err != nil { if errors.Is(err,fs.ErrExist) { return m.failConflict(ctx,in,"destination_exists") }; return m.fail(ctx,in,err) }
		if err := setStatus(ctx,m.DB,in.OpID,"path_reserved","published_fs",m.Clock.Now()); err != nil { return "", "", err }
	}
	return m.finalize(ctx,in)
}
```

In `finalize`, use one transaction to copy frozen hash/size into the pending Note, set `status='ready', revision=1`, move the operation `published_fs → finalized`, insert the whole `ReviewItem` or bite `ReviewPending` with uniqueness constraints, move `finalized → review_enqueued` whenever review mode is not `none`, and finally move to `completed`. `atomicNoClobber` must create and fsync a temporary file in the destination directory, use an OS no-replace operation, fsync that directory, and never remove an existing destination. Any pre-publication conflict marks the operation `failed` with its error while preserving existing bytes.

- [ ] **Step 4: Run the publication package tests**

Run: `go test ./internal/publish -v`
Expected: PASS; the retry has one Note/review set, changed fingerprints conflict, cross-project promotion is rejected, and existing bytes remain unchanged.

- [ ] **Step 5: Commit**

```bash
git add internal/store/promote.go internal/publish/machine.go internal/publish/machine_test.go
git commit -m "feat: add recoverable promote publication machine"
```

### Task 26: Startup recovery of non-terminal publications

**Files:**
- Create: `internal/publish/recover.go`
- Modify: `internal/publish/machine_test.go`
- Modify: `internal/app/app.go`

**Interfaces:**
- Consumes: Task 25 operation rows, staging files, source files, hashes, and `Machine.Run` resumable transitions.
- Produces: `(*publish.Machine).RecoverAll(context.Context) error`, reconciliation after a crash at every non-terminal status, and startup recovery before HTTP serving.

- [ ] **Step 1: Write the failing crash-recovery test**

```go
func TestRecoverAllConvergesAfterFilesystemPublish(t *testing.T) {
	ctx := context.Background()
	db, dataDir, projectID, sessionID := newPublishFixture(t)
	in := validPromoteInput(t, dataDir, projectID, sessionID)
	insertOperationAndPendingNote(t, db, in, "published_fs", "abc", 3)
	destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, loadProjectVault(db, projectID), projectID)),filepath.FromSlash(in.TargetRelPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(destination),0o755))
	require.NoError(t, os.WriteFile(destination,[]byte("abc"),0o644))
	m := &publish.Machine{DB:db,DataDir:dataDir,Clock:&clock.FakeClock{T:time.Date(2026,8,12,12,0,0,0,time.UTC)}}

	require.NoError(t,m.RecoverAll(ctx))
	var opStatus, noteStatus string
	require.NoError(t,db.QueryRow(`SELECT status FROM promote_ops WHERE id=?`,in.OpID).Scan(&opStatus))
	require.NoError(t,db.QueryRow(`SELECT status FROM notes WHERE id=?`,in.NoteID).Scan(&noteStatus))
	require.Equal(t,"completed",opStatus)
	require.Equal(t,"ready",noteStatus)
	require.NoError(t,m.RecoverAll(ctx))
	var count int
	require.NoError(t,db.QueryRow(`SELECT count(*) FROM review_items WHERE note_id=?`,in.NoteID).Scan(&count))
	require.Equal(t,1,count)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/publish -run TestRecoverAllConvergesAfterFilesystemPublish -v`
Expected: FAIL because `RecoverAll` is undefined.

- [ ] **Step 3: Implement recovery and startup wiring**

```go
func (m *Machine) RecoverAll(ctx context.Context) error {
	rows, err := m.DB.QueryContext(ctx, `SELECT id,request_key,request_fingerprint,session_id,workspace_path,target_project_id,target_relative_path,review_mode,note_id FROM promote_ops WHERE status NOT IN ('completed','failed') ORDER BY created_at,id`)
	if err != nil { return err }
	defer rows.Close()
	var inputs []PublishInput
	for rows.Next() {
		var in PublishInput
		if err := rows.Scan(&in.OpID,&in.RequestKey,&in.RequestFingerprint,&in.SessionID,&in.WorkspacePath,&in.TargetProjectID,&in.TargetRelPath,&in.ReviewMode,&in.NoteID); err != nil { return err }
		in.Kind = "promote"
		inputs = append(inputs,in)
	}
	if err := rows.Err(); err != nil { return err }
	for _, in := range inputs {
		if err := m.reconcilePublishedFile(ctx,in); err != nil { return fmt.Errorf("recover %s: %w",in.OpID,err) }
		if _,_,err := m.Run(ctx,in); err != nil { return fmt.Errorf("recover %s: %w",in.OpID,err) }
	}
	return nil
}

func (m *Machine) reconcilePublishedFile(ctx context.Context, in PublishInput) error {
	status := statusBefore(ctx,m.DB,in.OpID)
	if status != "published_fs" && status != "finalized" && status != "review_enqueued" { return nil }
	var want string
	if err := m.DB.QueryRowContext(ctx,`SELECT frozen_sha256 FROM promote_ops WHERE id=?`,in.OpID).Scan(&want); err != nil { return err }
	destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(m.DataDir, loadProjectVault(m.DB, in.TargetProjectID), in.TargetProjectID)),filepath.FromSlash(in.TargetRelPath))
	body, err := os.ReadFile(destination); if err != nil { return err }
	got := sha256.Sum256(body)
	if hex.EncodeToString(got[:]) != want { return fmt.Errorf("published file hash mismatch") }
	return nil
}
```

In `internal/app/app.go`, construct the machine after migrations and call `machine.RecoverAll(ctx)` before constructing or serving the HTTP mux; return the wrapped recovery error so startup fails safely rather than serving inconsistent state.

- [ ] **Step 4: Run focused and application tests**

Run: `go test ./internal/publish ./internal/app -v`
Expected: PASS, including repeated recovery producing exactly one review item.

- [ ] **Step 5: Commit**

```bash
git add internal/publish/recover.go internal/publish/machine_test.go internal/app/app.go
git commit -m "feat: recover unfinished publications at startup"
```

### Task 27: Exact sm2-lite-v1 scheduler

**Files:**
- Create: `internal/review/scheduler.go`
- Create: `internal/review/scheduler_test.go`

**Interfaces:**
- Consumes: `domain.Rating` values `again | hard | good | easy` and UTC review time.
- Produces: pure `review.ApplyRating(ReviewItemState, domain.Rating, time.Time) ReviewItemState` implementing the exact locked table.

- [ ] **Step 1: Write table-driven failing tests for every branch**

```go
package review_test

func TestApplyRatingExactTable(t *testing.T) {
	now := time.Date(2026,8,12,9,0,0,0,time.UTC)
	tests := []struct{name string; in review.ReviewItemState; rating domain.Rating; stage,reps,lapses int; interval,ease float64; due time.Time}{
		{"again", review.ReviewItemState{Stage:3,IntervalDays:10,EaseFactor:1.4,Reps:8,Lapses:2},domain.RatingAgain,0,0,3,0,1.3,now.Add(10*time.Minute)},
		{"hard-new",review.ReviewItemState{EaseFactor:2.5},domain.RatingHard,1,1,0,.5,2.35,now.Add(12*time.Hour)},
		{"hard-later",review.ReviewItemState{Stage:2,IntervalDays:10,EaseFactor:1.4,Reps:2},domain.RatingHard,2,3,0,12,1.3,now.Add(12*24*time.Hour)},
		{"good-new",review.ReviewItemState{EaseFactor:2.5},domain.RatingGood,1,1,0,1,2.5,now.Add(24*time.Hour)},
		{"good-stage-one",review.ReviewItemState{Stage:1,IntervalDays:1,EaseFactor:2.5,Reps:1},domain.RatingGood,2,2,0,3,2.5,now.Add(72*time.Hour)},
		{"good-later",review.ReviewItemState{Stage:2,IntervalDays:4,EaseFactor:2.5,Reps:2},domain.RatingGood,2,3,0,10,2.5,now.Add(10*24*time.Hour)},
		{"easy-new",review.ReviewItemState{EaseFactor:2.5},domain.RatingEasy,2,1,0,4,2.65,now.Add(4*24*time.Hour)},
		{"easy-later",review.ReviewItemState{Stage:2,IntervalDays:4,EaseFactor:2.5,Reps:2},domain.RatingEasy,2,3,0,13.78,2.65,now.Add(time.Duration(13.78*24*float64(time.Hour)))},
	}
	for _,tt := range tests { t.Run(tt.name,func(t *testing.T){
		got := review.ApplyRating(tt.in,tt.rating,now)
		require.Equal(t,tt.stage,got.Stage); require.Equal(t,tt.reps,got.Reps); require.Equal(t,tt.lapses,got.Lapses)
		require.InDelta(t,tt.interval,got.IntervalDays,1e-9); require.InDelta(t,tt.ease,got.EaseFactor,1e-9); require.Equal(t,tt.due,got.DueAt)
	}) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/review -run TestApplyRatingExactTable -v`
Expected: FAIL because `ApplyRating` is undefined.

- [ ] **Step 3: Implement the pure scheduler**

```go
package review

type ReviewItemState struct { Stage int; IntervalDays, EaseFactor float64; Reps, Lapses int; DueAt time.Time }

func ApplyRating(item ReviewItemState, rating domain.Rating, now time.Time) ReviewItemState {
	switch rating {
	case domain.RatingAgain:
		item.Lapses++; item.Reps=0; item.Stage=0; item.IntervalDays=0
		item.EaseFactor=math.Max(1.3,item.EaseFactor-.2); item.DueAt=now.Add(10*time.Minute)
	case domain.RatingHard:
		item.Reps++; item.EaseFactor=math.Max(1.3,item.EaseFactor-.15)
		if item.Stage==0 { item.IntervalDays=.5 } else { item.IntervalDays*=1.2 }
		if item.Stage<1 { item.Stage=1 }; item.DueAt=addDays(now,item.IntervalDays)
	case domain.RatingGood:
		item.Reps++
		if item.Stage==0 { item.IntervalDays=1; item.Stage=1 } else if item.Stage==1 { item.IntervalDays=3; item.Stage=2 } else { item.IntervalDays*=item.EaseFactor }
		item.DueAt=addDays(now,item.IntervalDays)
	case domain.RatingEasy:
		item.Reps++; item.EaseFactor+=.15
		if item.Stage<2 { item.IntervalDays=4; item.Stage=2 } else { item.IntervalDays=item.IntervalDays*item.EaseFactor*1.3 }
		item.DueAt=addDays(now,item.IntervalDays)
	default: panic("invalid rating")
	}
	return item
}
func addDays(t time.Time, days float64) time.Time { return t.Add(time.Duration(days*24*float64(time.Hour))) }
```

- [ ] **Step 4: Run scheduler tests**

Run: `go test ./internal/review -run TestApplyRatingExactTable -v`
Expected: PASS for Again +10m, Hard, Good, Easy, and the 1.3 ease floor.

- [ ] **Step 5: Commit**

```bash
git add internal/review/scheduler.go internal/review/scheduler_test.go
git commit -m "feat: implement exact sm2-lite-v1 scheduler"
```

### Task 28: Whole-note finalization and idempotent ratings

**Files:**
- Create: `internal/store/review.go`
- Modify: `internal/publish/machine.go`
- Create: `internal/store/review_test.go`

**Interfaces:**
- Consumes: Task 27 `ApplyRating`, finalized Note metadata, request key, expected `row_version`, and `clock.Clock`.
- Produces: immediately due whole-note items (`stage=0`, `interval_days=0`, `ease_factor=2.5`, `reps=0`, `lapses=0`), `store.ReviewStore.Rate`, one append-only `ReviewEvent`, idempotent retries, and optimistic concurrency conflicts.

- [ ] **Step 1: Write failing store tests**

```go
func TestRateIsAtomicIdempotentAndVersioned(t *testing.T) {
	db,itemID := reviewStoreFixture(t)
	s := store.ReviewStore{DB:db,Clock:&clock.FakeClock{T:time.Date(2026,8,12,9,0,0,0,time.UTC)}}
	got,err := s.Rate(context.Background(),itemID,"rate-1",0,domain.RatingGood,1250)
	require.NoError(t,err); require.Equal(t,int64(1),got.RowVersion); require.Equal(t,3.0,got.IntervalDays)

	again,err := s.Rate(context.Background(),itemID,"rate-1",0,domain.RatingGood,1250)
	require.NoError(t,err); require.Equal(t,got,again)
	var events int; require.NoError(t,db.QueryRow(`SELECT count(*) FROM review_events WHERE request_key='rate-1'`).Scan(&events)); require.Equal(t,1,events)

	_,err=s.Rate(context.Background(),itemID,"rate-2",0,domain.RatingEasy,20)
	var conflict *store.RowVersionConflict; require.ErrorAs(t,err,&conflict)
}

func TestWholeReviewItemStartsImmediatelyDue(t *testing.T) {
	db,dataDir,projectID,sessionID:=newPublishFixture(t); now:=time.Date(2026,8,12,9,0,0,0,time.UTC)
	in:=validPromoteInput(t,dataDir,projectID,sessionID); in.ReviewMode=domain.ReviewWhole
	_,_,err:=(&publish.Machine{DB:db,DataDir:dataDir,Clock:&clock.FakeClock{T:now}}).Run(context.Background(),in); require.NoError(t,err)
	var stage,reps,lapses int; var interval,ease float64; var due time.Time
	require.NoError(t,db.QueryRow(`SELECT stage,interval_days,ease_factor,reps,lapses,due_at FROM review_items WHERE note_id=?`,in.NoteID).Scan(&stage,&interval,&ease,&reps,&lapses,&due))
	require.Equal(t,[]any{0,0.0,2.5,0,0,now},[]any{stage,interval,ease,reps,lapses,due})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/store ./internal/publish -run 'Test(RateIsAtomic|WholeReview)' -v`
Expected: FAIL because `ReviewStore.Rate` and atomic whole-item finalization are absent.

- [ ] **Step 3: Implement atomic rating and whole-item insertion**

```go
func (s ReviewStore) Rate(ctx context.Context,itemID,key string,version int64,rating domain.Rating,durationMS int64)(RatedItem,error){
	tx,err:=s.DB.BeginTx(ctx,nil); if err!=nil{return RatedItem{},err}; defer tx.Rollback()
	if prior,ok,err:=eventResult(ctx,tx,key); err!=nil{return RatedItem{},err}else if ok{return prior,nil}
	var current RatedItem
	err=tx.QueryRowContext(ctx,`SELECT stage,interval_days,ease_factor,reps,lapses,due_at,row_version FROM review_items WHERE id=? AND status='active'`,itemID).Scan(&current.Stage,&current.IntervalDays,&current.EaseFactor,&current.Reps,&current.Lapses,&current.DueAt,&current.RowVersion)
	if err!=nil{return RatedItem{},err}; if current.RowVersion!=version{return RatedItem{},&RowVersionConflict{Current:current.RowVersion}}
	next:=review.ApplyRating(current.State(),rating,s.Clock.Now())
	previousJSON,_:=json.Marshal(current); resulting:=current.WithState(next); resulting.RowVersion++; resulting.LastReviewedAt=s.Clock.Now(); resultingJSON,_:=json.Marshal(resulting)
	res,err:=tx.ExecContext(ctx,`UPDATE review_items SET stage=?,interval_days=?,ease_factor=?,reps=?,lapses=?,due_at=?,last_reviewed_at=?,row_version=row_version+1 WHERE id=? AND row_version=?`,next.Stage,next.IntervalDays,next.EaseFactor,next.Reps,next.Lapses,next.DueAt,s.Clock.Now(),itemID,version)
	if err!=nil{return RatedItem{},err}; if n,_:=res.RowsAffected();n!=1{return RatedItem{},&RowVersionConflict{Current:version+1}}
	_,err=tx.ExecContext(ctx,`INSERT INTO review_events(id,review_item_id,request_key,rating,previous_state_json,resulting_state_json,scheduler_version,reviewed_at,duration_ms) VALUES(?,?,?,?,?,?,?,?,?)`,ids.NewID(),itemID,key,rating,previousJSON,resultingJSON,"sm2-lite-v1",s.Clock.Now(),durationMS)
	if err!=nil{return RatedItem{},err}; if err:=tx.Commit();err!=nil{return RatedItem{},err}; return resulting,nil
}
```

In the Task 25 finalize transaction, insert a `whole` item with `source_sha256`, `source_revision=1`, prompt `Review this note`, `due_at=now`, `status='active'`, `row_version=0`, and `scheduler_version='sm2-lite-v1'`; use a uniqueness-preserving insert so recovery cannot duplicate it.

- [ ] **Step 4: Run store and publication tests**

Run: `go test ./internal/store ./internal/publish -v`
Expected: PASS; same-key rating retries return the first result with one event, stale versions conflict, and whole items are immediately due.

- [ ] **Step 5: Commit**

```bash
git add internal/store/review.go internal/store/review_test.go internal/publish/machine.go
git commit -m "feat: finalize whole reviews and rate idempotently"
```

### Task 29: Bite generation lease worker and retry

**Files:**
- Create: `internal/review/bites.go`
- Create: `internal/review/bites_test.go`
- Modify: `internal/publish/machine.go`

**Interfaces:**
- Consumes: `agent.Provider`, finalized note bytes, `ReviewPending` rows, generator version `bites-v1`, and schema `{ "bites": [{"prompt": string, "answer": string}] }` limited to 8.
- Produces: `review.BiteWorker.LeaseAndRun(context.Context) (bool, error)`, transactional bite item creation, expired-lease recovery, failed retry state, and no effect on ready Notes.

- [ ] **Step 1: Write failing worker tests with a fake provider**

```go
type fakeProvider struct{ response string; err error }
func(f fakeProvider)Chat(context.Context,agent.ChatRequest)(agent.ChatResponse,error){return agent.ChatResponse{Content:f.response},f.err}

func TestBiteWorkerFailureKeepsNoteAndRetryCreatesItemsOnce(t *testing.T){
	db,dataDir,pendingID,noteID:=biteFixture(t)
	now:=time.Date(2026,8,12,9,0,0,0,time.UTC); c:=&clock.FakeClock{T:now}
	w:=review.BiteWorker{DB:db,DataDir:dataDir,Clock:c,Provider:fakeProvider{err:errors.New("provider down")},Lease:time.Minute}
	didWork,err:=w.LeaseAndRun(context.Background()); require.True(t,didWork); require.ErrorContains(t,err,"provider down")
	var noteStatus,pendingStatus string; require.NoError(t,db.QueryRow(`SELECT status FROM notes WHERE id=?`,noteID).Scan(&noteStatus)); require.Equal(t,"ready",noteStatus)
	require.NoError(t,db.QueryRow(`SELECT status FROM review_pending WHERE id=?`,pendingID).Scan(&pendingStatus)); require.Equal(t,"failed",pendingStatus)

	require.NoError(t,store.RetryReviewPending(context.Background(),db,pendingID))
	w.Provider=fakeProvider{response:`{"bites":[{"prompt":"What is A?","answer":"B"},{"prompt":"What is C?","answer":"D"}]}`}
	didWork,err=w.LeaseAndRun(context.Background()); require.True(t,didWork); require.NoError(t,err)
	var count int; require.NoError(t,db.QueryRow(`SELECT count(*) FROM review_items WHERE generation_id=?`,pendingID).Scan(&count)); require.Equal(t,2,count)
	didWork,err=w.LeaseAndRun(context.Background()); require.False(t,didWork); require.NoError(t,err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/review -run TestBiteWorkerFailureKeepsNoteAndRetryCreatesItemsOnce -v`
Expected: FAIL because `BiteWorker` does not exist.

- [ ] **Step 3: Implement leasing, strict decoding, and transactional insertion**

```go
type BiteWorker struct{DB *sql.DB; DataDir string; Clock clock.Clock; Provider agent.Provider; Lease time.Duration}
type biteOutput struct{Bites []struct{Prompt string `json:"prompt"`; Answer string `json:"answer"`} `json:"bites"`}

func(w *BiteWorker)LeaseAndRun(ctx context.Context)(bool,error){
	tx,err:=w.DB.BeginTx(ctx,nil);if err!=nil{return false,err};defer tx.Rollback()
	row:=tx.QueryRowContext(ctx,`SELECT id,note_id,source_sha256 FROM review_pending WHERE status='pending' OR (status='leased' AND lease_until<=?) ORDER BY id LIMIT 1`,w.Clock.Now())
	var jobID,noteID,sourceHash string;if err:=row.Scan(&jobID,&noteID,&sourceHash);errors.Is(err,sql.ErrNoRows){return false,nil}else if err!=nil{return false,err}
	res,err:=tx.ExecContext(ctx,`UPDATE review_pending SET status='leased',attempts=attempts+1,lease_until=? WHERE id=? AND (status='pending' OR lease_until<=?)`,w.Clock.Now().Add(w.Lease),jobID,w.Clock.Now());if err!=nil{return false,err};if n,_:=res.RowsAffected();n!=1{return false,nil};if err:=tx.Commit();err!=nil{return false,err}
	body,projectID,revision,err:=w.readReadyNote(ctx,noteID,sourceHash);if err!=nil{return true,w.fail(ctx,jobID,err)}
	resp,err:=w.Provider.Chat(ctx,agent.ChatRequest{Messages:[]agent.ChatMessage{{Role:"system",Content:"Return JSON only: {\"bites\":[{\"prompt\":string,\"answer\":string}]}, with 1 to 8 non-empty bites."},{Role:"user",Content:string(body)}}});if err!=nil{return true,w.fail(ctx,jobID,err)}
	var out biteOutput;dec:=json.NewDecoder(strings.NewReader(resp.Content));dec.DisallowUnknownFields();if err:=dec.Decode(&out);err!=nil{return true,w.fail(ctx,jobID,err)}
	if len(out.Bites)<1||len(out.Bites)>8{return true,w.fail(ctx,jobID,fmt.Errorf("generator returned %d bites",len(out.Bites)))}
	for _,b:=range out.Bites{if strings.TrimSpace(b.Prompt)==""||strings.TrimSpace(b.Answer)==""{return true,w.fail(ctx,jobID,fmt.Errorf("bite prompt and answer must be non-empty"))}}
	return true,w.complete(ctx,jobID,noteID,projectID,sourceHash,revision,out)
}
```

`complete` must use one transaction, verify the row is still `leased`, insert each bite with `(generation_id, ordinal)` uniqueness, initial whole-note scheduling defaults, and then set `ReviewPending.status='completed'`. `fail` sets only the pending row to `failed`, clears `lease_until`, records `last_error`, and never updates or deletes the Note. Finalization inserts one active `ReviewPending` for `(note_id, source_sha256, 'bites-v1')` and moves the operation through `review_enqueued` to `completed`.

- [ ] **Step 4: Run bite and publication tests**

Run: `go test ./internal/review ./internal/publish -v`
Expected: PASS; provider failure leaves the Note ready, retry creates exactly two items, and no duplicate job is processed.

- [ ] **Step 5: Commit**

```bash
git add internal/review/bites.go internal/review/bites_test.go internal/publish/machine.go
git commit -m "feat: generate retryable review bites"
```

### Task 30: Explicitly scoped review queue API

**Files:**
- Create: `internal/review/queue.go`
- Create: `internal/httpapi/review_handlers.go`
- Create: `internal/httpapi/review_handlers_test.go`
- Modify: `internal/httpapi/server.go`

**Interfaces:**
- Consumes: active due review items, owner time, Task 28 rating store, Task 29 retry, auth/CSRF middleware, and query scope `all | project:{id}`.
- Produces: `GET /api/v1/review/queue`, rate/suspend/retry mutations, explicit scope echo, and `caught_up` computed only in the selected scope.

- [ ] **Step 1: Write failing HTTP scope and mutation tests**

```go
func TestReviewQueueNeverWidensProjectScope(t *testing.T){
	srv,csrf,p1,p2,item1,item2:=reviewHTTPFixture(t);_ = item2
	r:=authedRequest(t,http.MethodGet,"/api/v1/review/queue?scope=project:"+p1,nil,csrf);w:=httptest.NewRecorder();srv.ServeHTTP(w,r)
	require.Equal(t,http.StatusOK,w.Code)
	var got struct{Scope string `json:"scope"`;CaughtUp bool `json:"caught_up"`;Items []struct{ID string `json:"id"`} `json:"items"`}
	require.NoError(t,json.NewDecoder(w.Body).Decode(&got));require.Equal(t,"project:"+p1,got.Scope);require.False(t,got.CaughtUp);require.Equal(t,item1,got.Items[0].ID);require.Len(t,got.Items,1)

	r=authedRequest(t,http.MethodGet,"/api/v1/review/queue?scope=project:"+p2,nil,csrf);w=httptest.NewRecorder();srv.ServeHTTP(w,r)
	require.Equal(t,http.StatusOK,w.Code);require.NoError(t,json.NewDecoder(w.Body).Decode(&got));require.Equal(t,"project:"+p2,got.Scope)
}

func TestReviewRateRetryAndSuspend(t *testing.T){
	srv,csrf,p1,_,item,_:=reviewHTTPFixture(t)
	body:=strings.NewReader(`{"rating":"good","request_key":"rating-1","row_version":0,"duration_ms":50}`)
	r:=authedRequest(t,http.MethodPost,"/api/v1/review/items/"+item+"/rate",body,csrf);w:=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusOK,w.Code)
	body=strings.NewReader(`{"rating":"good","request_key":"rating-1","row_version":0,"duration_ms":50}`)
	r=authedRequest(t,http.MethodPost,"/api/v1/review/items/"+item+"/rate",body,csrf);w=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusOK,w.Code)
	r=authedRequest(t,http.MethodPost,"/api/v1/review/items/"+item+"/suspend",strings.NewReader(`{}`),csrf);w=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusNoContent,w.Code)
	r=authedRequest(t,http.MethodGet,"/api/v1/review/queue?scope=project:"+p1,nil,csrf);w=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Contains(t,w.Body.String(),`"caught_up":true`)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/httpapi -run 'TestReview(Queue|Rate)' -v`
Expected: FAIL with 404 because review routes are not registered.

- [ ] **Step 3: Implement exact scope parsing and handlers**

```go
func ParseScope(raw string)(Scope,error){
	if raw=="all"{return Scope{Raw:"all"},nil}
	if strings.HasPrefix(raw,"project:")&&len(strings.TrimPrefix(raw,"project:"))>0{return Scope{Raw:raw,ProjectID:strings.TrimPrefix(raw,"project:")},nil}
	return Scope{},fmt.Errorf("scope must be all or project:{id}")
}
func(q Queue)Due(ctx context.Context,scope Scope)(QueueDTO,error){
	query:=`SELECT id,project_id,note_id,kind,prompt,answer,row_version,due_at FROM review_items WHERE status='active' AND due_at<=?`
	args:=[]any{q.Clock.Now()};if scope.ProjectID!=""{query+=` AND project_id=?`;args=append(args,scope.ProjectID)};query+=` ORDER BY due_at,id LIMIT 50`
	items,err:=scanQueue(ctx,q.DB,query,args...);if err!=nil{return QueueDTO{},err}
	return QueueDTO{Scope:scope.Raw,CaughtUp:len(items)==0,Items:items},nil
}
```

Register all four locked routes. Reject missing/invalid scope with 400 rather than defaulting. Rate validates the four exact ratings and maps `RowVersionConflict` to 409. Suspend updates only the named active item to `suspended` and is idempotent. Pending retry changes `failed → pending`, clears lease/error, and returns 409 for a row not in `failed`; all three mutations remain behind auth and CSRF.

- [ ] **Step 4: Run HTTP API tests**

Run: `go test ./internal/httpapi -run 'TestReview(Queue|Rate)' -v`
Expected: PASS; project responses contain no other-project item and caught-up changes after suspension.

- [ ] **Step 5: Commit**

```bash
git add internal/review/queue.go internal/httpapi/review_handlers.go internal/httpapi/review_handlers_test.go internal/httpapi/server.go
git commit -m "feat: expose explicitly scoped review queue"
```

### Task 31: Promote and operation-status HTTP APIs

**Files:**
- Create: `internal/httpapi/promote_handlers.go`
- Create: `internal/httpapi/promote_handlers_test.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/store/promote.go`

**Interfaces:**
- Consumes: Task 25 `Machine.Run`, session/project lookup, request `Idempotency-Key`, immutable canonical request fingerprint, and durable operation rows.
- Produces: `POST /api/v1/sessions/{id}/promote`, `GET /api/v1/operations/{id}`, 409 conflict responses, and badge DTO fields for promotion and card-generation retry states.

- [ ] **Step 1: Write failing endpoint tests**

```go
func TestPromoteEndpointIsIdempotentAndReportsBadges(t *testing.T){
	srv,csrf,sessionID:=promoteHTTPFixture(t)
	payload:=`{"workspace_path":"draft.md","target_relative_path":"saved/draft.md","review_mode":"bites"}`
	request:=func(key string) *httptest.ResponseRecorder { r:=authedRequest(t,http.MethodPost,"/api/v1/sessions/"+sessionID+"/promote",strings.NewReader(payload),csrf);r.Header.Set("Idempotency-Key",key);w:=httptest.NewRecorder();srv.ServeHTTP(w,r);return w }
	w1:=request("promote-http-1");require.Equal(t,http.StatusAccepted,w1.Code)
	w2:=request("promote-http-1");require.Equal(t,http.StatusAccepted,w2.Code);require.JSONEq(t,w1.Body.String(),w2.Body.String())
	var accepted struct{OperationID string `json:"operation_id"`};require.NoError(t,json.Unmarshal(w1.Body.Bytes(),&accepted))
	r:=authedRequest(t,http.MethodGet,"/api/v1/operations/"+accepted.OperationID,nil,csrf);w:=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusOK,w.Code)
	require.Contains(t,w.Body.String(),`"publication_status":"completed"`);require.Contains(t,w.Body.String(),`"badge":"Note saved; cards pending…"`);require.Contains(t,w.Body.String(),`"retry_cards":false`)
}

func TestPromoteEndpointMapsConflictsTo409(t *testing.T){
	srv,csrf,sessionID:=promoteHTTPFixture(t)
	r:=authedRequest(t,http.MethodPost,"/api/v1/sessions/"+sessionID+"/promote",strings.NewReader(`{"workspace_path":"draft.md","target_relative_path":"a.md","review_mode":"none"}`),csrf);r.Header.Set("Idempotency-Key","same");w:=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusAccepted,w.Code)
	r=authedRequest(t,http.MethodPost,"/api/v1/sessions/"+sessionID+"/promote",strings.NewReader(`{"workspace_path":"draft.md","target_relative_path":"b.md","review_mode":"none"}`),csrf);r.Header.Set("Idempotency-Key","same");w=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusConflict,w.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/httpapi -run TestPromoteEndpoint -v`
Expected: FAIL with 404 because promote routes do not exist.

- [ ] **Step 3: Implement promote submission, fingerprinting, and status DTOs**

```go
type promoteRequest struct{WorkspacePath string `json:"workspace_path"`;TargetRelativePath string `json:"target_relative_path"`;ReviewMode domain.ReviewMode `json:"review_mode"`}
func(h PromoteHandler)Create(w http.ResponseWriter,r *http.Request){
	key:=strings.TrimSpace(r.Header.Get("Idempotency-Key"));if key==""{writeError(w,400,"idempotency_key_required");return}
	var req promoteRequest;if err:=decodeStrictJSON(r,&req);err!=nil{writeError(w,400,"invalid_request");return}
	projectID,err:=h.Store.SessionProject(r.Context(),r.PathValue("id"));if err!=nil{writeStoreError(w,err);return}
	canonical,_:=json.Marshal(struct{SessionID,WorkspacePath,TargetProjectID,TargetRelativePath string;ReviewMode domain.ReviewMode}{r.PathValue("id"),req.WorkspacePath,projectID,req.TargetRelativePath,req.ReviewMode})
	fingerprint:=sha256.Sum256(canonical)
	in:=publish.PublishInput{OpID:ids.NewID(),RequestKey:key,RequestFingerprint:hex.EncodeToString(fingerprint[:]),Kind:"promote",SessionID:r.PathValue("id"),WorkspacePath:req.WorkspacePath,TargetProjectID:projectID,TargetRelPath:req.TargetRelativePath,ReviewMode:req.ReviewMode,NoteID:ids.NewID()}
	status,noteID,err:=h.Machine.Run(r.Context(),in);if err!=nil{if _,ok:=err.(*publish.ConflictError);ok{writeError(w,409,err.Error());return};writeError(w,422,err.Error());return}
	writeJSON(w,202,map[string]any{"operation_id":in.OpID,"note_id":noteID,"status":status})
}
```

When an existing idempotency key is returned, expose its stored operation ID rather than the newly allocated ID. The status handler returns exact `publication_status`, Note status, pending ID/status, and derives only these copies: non-terminal publication=`Promoting…`; failed publication=`Promote failed — Retry`; ready note plus pending/leased cards=`Note saved; cards pending…`; ready note plus failed cards=`Cards failed — Retry cards`; otherwise ready=`Ready`. Set `retry_cards=true` only for failed `ReviewPending`; do not retry publication with a new destination or silently overwrite.

- [ ] **Step 4: Run HTTP tests**

Run: `go test ./internal/httpapi -run TestPromoteEndpoint -v`
Expected: PASS for same-key replay, changed-fingerprint 409, operation status, and card badge data.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/promote_handlers.go internal/httpapi/promote_handlers_test.go internal/httpapi/server.go internal/store/promote.go
git commit -m "feat: expose promote operation APIs"
```

### Task 32: Save-to-source and review web UI

**Files:**
- Modify: `web/js/api.js`
- Modify: `web/js/pages/sessions.js`
- Create: `web/js/pages/review.js`
- Create: `web/js/components/status-badges.js`
- Modify: `web/js/router.js`
- Modify: `web/css/app.css`
- Test: `internal/httpapi/web_test.go`

**Interfaces:**
- Consumes: Tasks 30–31 review/promote APIs, session project ID, selected workspace `.md`, operation badge DTOs, and URL query scope.
- Produces: Save to source modal, project-only target, review cards for whole/bite items, explicit scope chip, rating/suspend actions, caught-up UI, and durable operation/card badges with retry-cards action.

- [ ] **Step 1: Write a failing embedded-web contract test**

```go
func TestWebContainsPromoteAndReviewContracts(t *testing.T){
	tests:=map[string][]string{
		"../../../web/js/pages/sessions.js":{"Save to source","target_relative_path","review_mode","operation_id"},
		"../../../web/js/pages/review.js":{"project:","scope=","caught_up","row_version","duration_ms"},
		"../../../web/js/components/status-badges.js":{"Promoting…","Promote failed — Retry","Note saved; cards pending…","Cards failed — Retry cards","Ready"},
	}
	for file,wants:=range tests{t.Run(filepath.Base(file),func(t *testing.T){body,err:=os.ReadFile(file);require.NoError(t,err);for _,want:=range wants{require.Contains(t,string(body),want)}})}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestWebContainsPromoteAndReviewContracts -v`
Expected: FAIL because the review page and badge component do not exist.

- [ ] **Step 3: Implement the modal, cards, scope state, and badges**

```js
// web/js/components/status-badges.js
export function operationBadge(operation, onRetryCards) {
  const el = document.createElement('div'); el.className = `status-badge status-${operation.publication_status}`;
  el.textContent = operation.badge || 'Ready';
  if (operation.retry_cards) { const button=document.createElement('button'); button.textContent='Retry cards'; button.onclick=onRetryCards; el.append(' ',button); }
  return el;
}

// web/js/pages/review.js
export async function renderReview(root,{projectId}) {
  const params=new URLSearchParams(location.search); const fallback=projectId?`project:${projectId}`:'all'; const scope=params.get('scope')||fallback;
  if(scope!=='all'&&!scope.startsWith('project:')) throw new Error('Invalid review scope');
  const data=await api.get(`/api/v1/review/queue?scope=${encodeURIComponent(scope)}`);
  const chip=document.createElement('nav'); chip.className='scope-chip'; chip.innerHTML=`<button data-scope="project:${projectId}">This project</button><button data-scope="all">All projects</button>`;
  chip.onclick=e=>{const next=e.target.dataset.scope;if(!next)return;params.set('scope',next);history.pushState({},'',`${location.pathname}?${params}`);renderReview(root,{projectId});};
  root.replaceChildren(chip);
  if(data.caught_up){const empty=document.createElement('p');empty.className='caught-up';empty.textContent=`Caught up in ${data.scope==='all'?'all projects':'this project'}.`;root.append(empty);return;}
  for(const item of data.items){const started=performance.now();const card=document.createElement('article');card.className='review-card';card.innerHTML=`<h2>${escapeHTML(item.prompt)}</h2>${item.kind==='whole'?'<button class="open-note">Open current note</button>':`<button class="reveal">Reveal answer</button><p class="answer" hidden>${escapeHTML(item.answer)}</p>`}<div class="ratings"></div>`;
    card.querySelector('.reveal')?.addEventListener('click',()=>card.querySelector('.answer').hidden=false);
    for(const rating of ['again','hard','good','easy']){const b=document.createElement('button');b.textContent=rating;b.onclick=async()=>{await api.post(`/api/v1/review/items/${item.id}/rate`,{rating,request_key:crypto.randomUUID(),row_version:item.row_version,duration_ms:Math.round(performance.now()-started)});renderReview(root,{projectId});};card.querySelector('.ratings').append(b);}
    const suspend=document.createElement('button');suspend.textContent='Suspend';suspend.onclick=async()=>{await api.post(`/api/v1/review/items/${item.id}/suspend`,{});renderReview(root,{projectId});};card.append(suspend);root.append(card);}
}
```

In `sessions.js`, show **Save to source** only for a selected regular `.md` workspace file. The modal collects `target_relative_path` and radio values `none | whole | bites`; it displays the immutable session project name without a target-project picker, posts with a fresh `Idempotency-Key`, stores `operation_id`, polls `/api/v1/operations/{id}` across navigation/reload, renders `operationBadge`, and invokes `/api/v1/review/pending/{id}/retry` only when `retry_cards` is true. In `router.js`, project Review links set `?scope=project:{projectId}` and Home review sets `?scope=all`. Add focused modal/card/scope/badge styles without introducing a bundler.

- [ ] **Step 4: Run web contract and full tests**

Run: `go test ./internal/httpapi -run TestWebContainsPromoteAndReviewContracts -v && go test ./...`
Expected: PASS; static contracts include explicit scope, optimistic rating fields, modal payload fields, and all exact badge copy.

- [ ] **Step 5: Commit**

```bash
git add web/js/api.js web/js/pages/sessions.js web/js/pages/review.js web/js/components/status-badges.js web/js/router.js web/css/app.css internal/httpapi/web_test.go
git commit -m "feat: add promote and review web flows"
```

## Phase self-check

- Publication §6 and F4: Tasks 25–26 and 31–32 cover freezing, staging, reservation, no-clobber publish, exact durable statuses, finalization, idempotency/fingerprints, 409 conflicts, project-home target restriction, recovery, operation polling, and badges.
- Review §7 and F7: Tasks 27–30 and 32 cover the exact `sm2-lite-v1` table, immediately due whole items, bite snapshots, explicit `project:{id} | all` scope, rating transactions/events/versioning, suspension, and caught-up behavior.
- Data model §5: Tasks 25, 28, and 29 use PromoteOperation, Note, ReviewPending, ReviewItem, and ReviewEvent contracts with exact locked statuses and versions.
- Failure/status UX §9: Tasks 29, 31, and 32 preserve ready Notes after generation failure and expose retry-card controls plus all exact badge text.
- Acceptance §13: Task 25 proves same-key uniqueness and destination no-overwrite (1, 6); Task 26 proves post-publish convergence (2); Task 29 proves bite failure preserves the Note (3); Task 25 enforces project session scope (4); Phase 4 supplies rooted path/symlink enforcement used here; Task 28 proves one event on rating retry (8).


## Phase 6: Backup

### Task 33: Create a mutation-safe local backup bundle

**Path errata:** Seed and pack only under `$PA_DATA_DIR/files/**` and `$PA_DATA_DIR/staging/**` using `layout.ProjectRoot` (includes `files/global|vaults`). Bundle/restore MUST follow Canonical **Backup bundle layout (exact)**. Ignore snippets writing `dataDir/global/...` without `files/` or restoring `database.sqlite` to data dir root.


**Artifact errata:** Output a **directory bundle** per Canonical backup artifact contract. Do not require `.tar.gz` as the sole LocalPath. `backup_runs.local_path` = bundle directory.

**Files:**
- Create: `internal/backup/backup.go`
- Create: `internal/backup/backup_test.go`
- Create: `internal/store/backup.go`
- Modify: `internal/domain/models.go`
- Modify: `internal/app/app.go`

**Interfaces:**
- Consumes: the application `*sql.DB`, `PA_DATA_DIR`, `clock.Clock`, `ids.NewID()`, and the existing application composition root.
- Produces: `backup.Barrier`, `backup.Service.Run(context.Context) (domain.BackupRun, error)`, `backup.Service.List(context.Context) ([]domain.BackupRun, error)`, and immutable `.tar.gz` bundles containing `database.sqlite`, `files/`, and a final `manifest.json` with SHA-256 checksums.

- [ ] **Step 1: Write the failing local-backup tests**

Create `internal/backup/backup_test.go`:

```go
package backup_test

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/backup"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	dbopen "github.com/rigasyahrul/personal-agent/internal/db"
)

func TestRunCreatesConsistentLocalBundleAndSucceededRun(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	db, err := dbopen.Open(filepath.Join(dataDir, "db", "personal-agent.sqlite"))
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','Known','2026-08-12T10:00:00Z','2026-08-12T10:00:00Z')`); err != nil { t.Fatal(err) }
	source := filepath.Join(dataDir, "global", "projects", "p1", "source")
	if err := os.MkdirAll(source, 0o700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(source, "known.md"), []byte("# known note\n"), 0o600); err != nil { t.Fatal(err) }

	barrier := &backup.Barrier{}
	svc := backup.NewService(db, dataDir, barrier, &clock.FakeClock{T: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)}, nil)
	run, err := svc.Run(ctx)
	if err != nil { t.Fatal(err) }
	if run.Status != "succeeded" || run.LocalPath == "" || run.ManifestHash == "" { t.Fatalf("run=%+v", run) }
	manifest, files := readBundle(t, run.LocalPath)
	if manifest.CutoffAt != "2026-08-12T10:00:00Z" { t.Fatalf("cutoff=%q", manifest.CutoffAt) }
	for name, want := range manifest.Files {
		sum := sha256.Sum256(files[name])
		if hex.EncodeToString(sum[:]) != want { t.Fatalf("checksum mismatch for %s", name) }
	}
	snapshot := filepath.Join(t.TempDir(), "snapshot.sqlite")
	if err := os.WriteFile(snapshot, files["database.sqlite"], 0o600); err != nil { t.Fatal(err) }
	restored, err := sql.Open("sqlite", snapshot)
	if err != nil { t.Fatal(err) }
	defer restored.Close()
	var name string
	if err := restored.QueryRow(`SELECT name FROM projects WHERE id='p1'`).Scan(&name); err != nil { t.Fatal(err) }
	if name != "Known" || string(files["files/global/projects/p1/source/known.md"]) != "# known note\n" { t.Fatal("bundle omitted known data") }
}

func TestBarrierBlocksMutationDuringSnapshot(t *testing.T) {
	b := &backup.Barrier{}
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() { _ = b.Snapshot(func() error { close(entered); <-release; return nil }) }()
	<-entered
	go func() { _ = b.Mutate(func() error { close(done); return nil }) }()
	select { case <-done: t.Fatal("mutation crossed snapshot barrier"); case <-time.After(30 * time.Millisecond): }
	close(release)
	select { case <-done: case <-time.After(time.Second): t.Fatal("mutation remained blocked") }
}

type testManifest struct { CutoffAt string `json:"cutoff_at"`; Files map[string]string `json:"files"` }

func readBundle(t *testing.T, path string) (testManifest, map[string][]byte) {
	t.Helper()
	f, err := os.Open(path); if err != nil { t.Fatal(err) }; defer f.Close()
	gz, err := gzip.NewReader(f); if err != nil { t.Fatal(err) }; defer gz.Close()
	tr := tar.NewReader(gz); files := map[string][]byte{}
	for { h, err := tr.Next(); if err == io.EOF { break }; if err != nil { t.Fatal(err) }; b, err := io.ReadAll(tr); if err != nil { t.Fatal(err) }; files[h.Name] = b }
	var m testManifest
	if err := json.Unmarshal(files["manifest.json"], &m); err != nil { t.Fatal(err) }
	return m, files
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/backup -run 'TestRunCreates|TestBarrier' -v`

Expected: FAIL because package `internal/backup` and its service do not exist.

- [ ] **Step 3: Implement the local snapshot, manifest, and run store**

Add this model to `internal/domain/models.go`:

```go
type BackupRun struct {
	ID, Status, CutoffAt, StartedAt string
	LocalPath, ObjectKey, ManifestHash, CompletedAt, Error string
}
```

Create `internal/store/backup.go` with `CreateBackupRun`, `CompleteBackupRun`, and `ListBackupRuns`; use the migration's `backup_runs` columns, order lists by `started_at DESC`, and translate nullable columns with `COALESCE(column,'')`:

```go
package store

import (
	"context"
	"database/sql"
	"github.com/rigasyahrul/personal-agent/internal/domain"
)

func CreateBackupRun(ctx context.Context, db *sql.DB, r domain.BackupRun) error {
	_, err := db.ExecContext(ctx, `INSERT INTO backup_runs(id,status,cutoff_at,started_at) VALUES(?,?,?,?)`, r.ID, r.Status, r.CutoffAt, r.StartedAt)
	return err
}

func CompleteBackupRun(ctx context.Context, db *sql.DB, r domain.BackupRun) error {
	_, err := db.ExecContext(ctx, `UPDATE backup_runs SET status=?,local_path=NULLIF(?,''),object_key=NULLIF(?,''),manifest_hash=NULLIF(?,''),completed_at=NULLIF(?,''),error=NULLIF(?,'') WHERE id=?`, r.Status, r.LocalPath, r.ObjectKey, r.ManifestHash, r.CompletedAt, r.Error, r.ID)
	return err
}

func ListBackupRuns(ctx context.Context, db *sql.DB) ([]domain.BackupRun, error) {
	rows, err := db.QueryContext(ctx, `SELECT id,status,cutoff_at,COALESCE(local_path,''),COALESCE(object_key,''),COALESCE(manifest_hash,''),started_at,COALESCE(completed_at,''),COALESCE(error,'') FROM backup_runs ORDER BY started_at DESC`)
	if err != nil { return nil, err }; defer rows.Close()
	var out []domain.BackupRun
	for rows.Next() { var r domain.BackupRun; if err := rows.Scan(&r.ID,&r.Status,&r.CutoffAt,&r.LocalPath,&r.ObjectKey,&r.ManifestHash,&r.StartedAt,&r.CompletedAt,&r.Error); err != nil { return nil, err }; out = append(out,r) }
	return out, rows.Err()
}
```

Create `internal/backup/backup.go`. The SQLite driver connection exposes the online backup API through `NewBackup`; write the database snapshot before walking data files, skip the live DB/WAL/SHM and `backups/`, sort archive names, write `manifest.json` after every payload member, rename the temporary archive atomically, and record failures:

```go
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"modernc.org/sqlite"
)

type Barrier struct{ mu sync.RWMutex }
func (b *Barrier) Mutate(fn func() error) error { b.mu.RLock(); defer b.mu.RUnlock(); return fn() }
func (b *Barrier) Snapshot(fn func() error) error { b.mu.Lock(); defer b.mu.Unlock(); return fn() }

type Uploader interface { Upload(context.Context, string, string) error }
type Service struct { DB *sql.DB; DataDir string; Barrier *Barrier; Clock clock.Clock; Uploader Uploader; Bucket string }
func NewService(db *sql.DB, dataDir string, barrier *Barrier, c clock.Clock, uploader Uploader) *Service { return &Service{DB:db,DataDir:dataDir,Barrier:barrier,Clock:c,Uploader:uploader} }
func (s *Service) List(ctx context.Context) ([]domain.BackupRun,error) { return store.ListBackupRuns(ctx,s.DB) }

type manifest struct { Version int `json:"version"`; CutoffAt string `json:"cutoff_at"`; Files map[string]string `json:"files"` }

func (s *Service) Run(ctx context.Context) (run domain.BackupRun, err error) {
	now := s.Clock.Now().UTC(); run = domain.BackupRun{ID:ids.NewID(),Status:"running",CutoffAt:now.Format(time.RFC3339),StartedAt:now.Format(time.RFC3339)}
	if err = store.CreateBackupRun(ctx,s.DB,run); err != nil { return run,err }
	defer func(){ if err != nil { run.Status="failed"; run.Error=err.Error(); run.CompletedAt=s.Clock.Now().UTC().Format(time.RFC3339); _=store.CompleteBackupRun(context.Background(),s.DB,run) } }()
	err = s.Barrier.Snapshot(func() error { var e error; run.LocalPath,run.ManifestHash,e=s.local(ctx,run); return e })
	if err != nil { return run,err }
	run.Status="succeeded"; run.CompletedAt=s.Clock.Now().UTC().Format(time.RFC3339)
	if err=store.CompleteBackupRun(ctx,s.DB,run); err!=nil{return run,err}
	return run,nil
}

func (s *Service) local(ctx context.Context, run domain.BackupRun) (string,string,error) {
	dir:=filepath.Join(s.DataDir,"backups"); if err:=os.MkdirAll(dir,0o700); err!=nil{return "","",err}
	work,err:=os.MkdirTemp(dir,"snapshot-"); if err!=nil{return "","",err}; defer os.RemoveAll(work)
	dbPath:=filepath.Join(work,"database.sqlite")
	conn,err:=s.DB.Conn(ctx); if err!=nil{return "","",err}
	err=conn.Raw(func(raw any) error { c,ok:=raw.(interface{ NewBackup(string)(*sqlite.Backup,error) }); if !ok{return fmt.Errorf("sqlite connection lacks backup API")}; b,e:=c.NewBackup("file:"+dbPath); if e!=nil{return e}; defer b.Finish(); for { more,e:=b.Step(128); if e!=nil{return e}; if !more{return nil} } }); conn.Close()
	if err!=nil{return "","",err}
	entries:=map[string]string{"database.sqlite":dbPath}
	err=filepath.WalkDir(s.DataDir,func(path string,d os.DirEntry,e error) error { if e!=nil{return e}; rel,e:=filepath.Rel(s.DataDir,path); if e!=nil{return e}; if rel=="."{return nil}; first:=strings.Split(filepath.ToSlash(rel),"/")[0]; if d.IsDir() && (first=="backups"||first=="db"){return filepath.SkipDir}; if first=="db"||first=="backups"{return nil}; if d.IsDir(){return nil}; // only pack files/ and staging/
 if first!="files" && first!="staging"{return nil}; entries[filepath.ToSlash(rel)]=path; return nil }); if err!=nil{return "","",err}
	final:=filepath.Join(dir,run.ID+".tar.gz"); tmp:=final+".tmp"; f,err:=os.OpenFile(tmp,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0o600); if err!=nil{return "","",err}
	gz:=gzip.NewWriter(f); tw:=tar.NewWriter(gz); sums:=map[string]string{}; names:=make([]string,0,len(entries)); for n:=range entries{names=append(names,n)}; sort.Strings(names)
	write:=func(name string,b []byte) error { if err:=tw.WriteHeader(&tar.Header{Name:name,Mode:0o600,Size:int64(len(b)),ModTime:time.Unix(0,0)});err!=nil{return err}; _,err:=tw.Write(b); return err }
	for _,name:=range names { b,e:=os.ReadFile(entries[name]); if e!=nil{err=e;break}; sum:=sha256.Sum256(b); sums[name]=hex.EncodeToString(sum[:]); if e=write(name,b);e!=nil{err=e;break} }
	m:=manifest{Version:1,CutoffAt:run.CutoffAt,Files:sums}; mb,_:=json.Marshal(m); if err==nil{err=write("manifest.json",mb)}; if e:=tw.Close();err==nil{err=e}; if e:=gz.Close();err==nil{err=e}; if e:=f.Close();err==nil{err=e}; if err!=nil{os.Remove(tmp);return "","",err}; if err=os.Rename(tmp,final);err!=nil{return "","",err}; mh:=sha256.Sum256(mb); return final,hex.EncodeToString(mh[:]),nil
}

var _ = io.EOF
```

In `internal/app/app.go`, construct exactly one `barrier := &backup.Barrier{}` and pass it both to `backup.NewService(...)` and every mutation entry point. Wrap database-plus-filesystem mutation bodies (project/session creation and deletion, workspace writes, publication/recovery, bite generation, and review rating) with `barrier.Mutate`; do not lock read-only paths. This ensures the exclusive snapshot waits for in-flight file operations to reach their durable operation state and prevents new ones until the manifest is complete.

- [ ] **Step 4: Run focused and regression tests**

Run: `gofmt -w internal/backup internal/store/backup.go internal/domain/models.go internal/app/app.go && go test ./internal/backup ./internal/store ./internal/app`

Expected: PASS; the bundle test proves both the online database snapshot and source file are checksummed and readable.

- [ ] **Step 5: Commit**

```bash
git add internal/backup/backup.go internal/backup/backup_test.go internal/store/backup.go internal/domain/models.go internal/app/app.go
git commit -m "feat: create mutation-safe local backups"
```

### Task 34: Upload backups to optional S3 storage
**S3 errata:** Implement `backup.Sink.Upload(localDir, objectPrefix)` uploading each file under the directory. Ignore single-file `PutObject(LocalPath)` snippets. nil sink => local succeeded without upload.


**Files:**
- Create: `internal/backup/s3.go`
- Modify: `internal/backup/backup.go`
- Modify: `internal/backup/backup_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/app/app.go`

**Interfaces:**
- Consumes: Task 33's local immutable bundle, `backup.Uploader`, and environment configuration.
- Produces: `backup.NewS3Uploader(client S3PutObjectAPI, bucket string) Uploader`; unset `PA_BACKUP_S3_BUCKET` requires no AWS credentials and local completion is sufficient, while a configured bucket requires successful upload before `BackupRun.status=succeeded`.

- [ ] **Step 1: Write failing optional-upload tests**

Append to `internal/backup/backup_test.go`:

```go
type mockUploader struct { calls int; key string; err error }
func (m *mockUploader) Upload(_ context.Context, path,key string) error { m.calls++; m.key=key; if _,err:=os.Stat(path);err!=nil{return err}; return m.err }

func TestRunWithoutBucketSucceedsLocallyWithoutUpload(t *testing.T) {
	svc, db := testService(t); up := &mockUploader{}; svc.Uploader=up
	run, err := svc.Run(context.Background())
	if err != nil || run.Status != "succeeded" || up.calls != 0 { t.Fatalf("run=%+v calls=%d err=%v",run,up.calls,err) }
	assertStoredStatus(t,db,run.ID,"succeeded")
}

func TestConfiguredUploadControlsFinalStatus(t *testing.T) {
	svc, db := testService(t); up := &mockUploader{}; svc.Uploader=up; svc.Bucket="archive"
	run, err := svc.Run(context.Background())
	if err != nil || run.Status != "succeeded" || up.calls != 1 || run.ObjectKey == "" { t.Fatalf("run=%+v calls=%d err=%v",run,up.calls,err) }
	assertStoredStatus(t,db,run.ID,"succeeded")

	svc, db = testService(t); svc.Bucket="archive"; svc.Uploader=&mockUploader{err:fmt.Errorf("S3 unavailable")}
	run, err = svc.Run(context.Background())
	if err == nil || run.Status != "failed" || run.LocalPath == "" { t.Fatalf("run=%+v err=%v",run,err) }
	assertStoredStatus(t,db,run.ID,"failed")
}
```

Add helpers using `dbopen.Open(filepath.Join(t.TempDir(), "db", "personal-agent.sqlite"))`, a fake clock, a new barrier, and a query of `backup_runs.status`. Each helper must register `db.Close` with `t.Cleanup`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/backup -run 'TestRunWithoutBucket|TestConfiguredUpload' -v`

Expected: `TestRunWithoutBucketSucceedsLocallyWithoutUpload` passes and `TestConfiguredUploadControlsFinalStatus` fails because a configured upload is not yet invoked.

- [ ] **Step 3: Implement optional upload and final status transition**

Replace the tail of `Service.Run` after `s.Barrier.Snapshot(...)` with:

```go
	if err != nil { return run, err }
	if s.Bucket != "" {
		if s.Uploader == nil { err = fmt.Errorf("backup bucket configured without uploader"); return run, err }
		run.ObjectKey = "backups/" + filepath.Base(run.LocalPath)
		if err = s.Uploader.Upload(ctx, run.LocalPath, run.ObjectKey); err != nil { return run, err }
	}
	run.Status = "succeeded"
	run.CompletedAt = s.Clock.Now().UTC().Format(time.RFC3339)
	if err = store.CompleteBackupRun(ctx, s.DB, run); err != nil { return run, err }
	return run, nil
```

Create `internal/backup/s3.go` using the AWS SDK's narrow mockable interface:

```go
package backup

import (
	"context"
	"os"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3PutObjectAPI interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}
type s3Uploader struct { client S3PutObjectAPI; bucket string }
func NewS3Uploader(client S3PutObjectAPI, bucket string) Uploader { return &s3Uploader{client:client,bucket:bucket} }
func (u *s3Uploader) Upload(ctx context.Context,path,key string) error {
	f,err:=os.Open(path); if err!=nil{return err}; defer f.Close()
	_,err=u.client.PutObject(ctx,&s3.PutObjectInput{Bucket:&u.bucket,Key:&key,Body:f})
	return err
}
```

Add `BackupS3Bucket string` to config and load it from `PA_BACKUP_S3_BUCKET`. In `internal/app/app.go`, only when the bucket is non-empty, load AWS default config, construct `s3.NewFromConfig`, and assign `service.Bucket` and `service.Uploader`; with an empty bucket, do not load AWS config. Add `github.com/aws/aws-sdk-go-v2/config` and `github.com/aws/aws-sdk-go-v2/service/s3` with `go get`.

- [ ] **Step 4: Run focused and configuration tests**

Run: `gofmt -w internal/backup internal/config internal/app && go test ./internal/backup ./internal/config ./internal/app`

Expected: PASS, including local-only success, upload success, and upload-failure persistence.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/backup internal/config/config.go internal/app/app.go
git commit -m "feat: optionally upload backup bundles to S3"
```

### Task 35: Expose backup controls and status in Settings

**Schedule errata:** Implement Canonical **Backup schedule (v1)**: settings read/write `backup_schedule` (`off`|`daily`), UI control, and wire daily ticker in app to `backup.Service.Run`. Manual Backup now remains. Show sink-configured boolean from env, never secrets.


**Files:**
- Create: `internal/httpapi/backup_handlers.go`
- Create: `internal/httpapi/backup_handlers_test.go`
- Modify: `internal/httpapi/server.go`
- Modify or create if missing: `internal/httpapi/settings_handlers.go` (required since Tasks 5–7 errata)
- Modify: `web/js/api.js`
- Modify or create if missing: `web/js/pages/settings.js`

**Interfaces:**
- Consumes: `backup.Service.List`, `backup.Service.Run`, existing authenticated/CSRF middleware, and the existing settings DTO.
- Produces: authenticated `GET /api/v1/backups`, CSRF-protected `POST /api/v1/backups`, and settings fields `backup.last_success` and `backup.last_failure` plus a “Backup now” control.

- [ ] **Step 1: Write failing HTTP contract tests**

Create `internal/httpapi/backup_handlers_test.go` following the package's existing authenticated test-server helper:

```go
package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBackupsRequireAuthAndPostRequiresCSRF(t *testing.T) {
	s := newTestServer(t)
	for _,tc:=range []struct{method string; csrf bool; want int}{{"GET",false,http.StatusUnauthorized},{"POST",false,http.StatusUnauthorized}} {
		r:=httptest.NewRequest(tc.method,"/api/v1/backups",nil); w:=httptest.NewRecorder(); s.ServeHTTP(w,r)
		if w.Code!=tc.want { t.Fatalf("%s got %d",tc.method,w.Code) }
	}
	r:=authenticatedRequest(t,"POST","/api/v1/backups",strings.NewReader(`{}`),false); w:=httptest.NewRecorder(); s.ServeHTTP(w,r)
	if w.Code!=http.StatusForbidden { t.Fatalf("got %d",w.Code) }
}

func TestBackupNowThenListAndSettingsStatus(t *testing.T) {
	s:=newTestServer(t)
	r:=authenticatedRequest(t,"POST","/api/v1/backups",strings.NewReader(`{}`),true); w:=httptest.NewRecorder(); s.ServeHTTP(w,r)
	if w.Code!=http.StatusCreated || !strings.Contains(w.Body.String(),`"status":"succeeded"`) { t.Fatalf("%d %s",w.Code,w.Body.String()) }
	for _,path:=range []string{"/api/v1/backups","/api/v1/settings"} {
		r=authenticatedRequest(t,"GET",path,nil,false); w=httptest.NewRecorder(); s.ServeHTTP(w,r)
		if w.Code!=http.StatusOK || !strings.Contains(w.Body.String(),`"last_success"`) { t.Fatalf("%s: %d %s",path,w.Code,w.Body.String()) }
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/httpapi -run 'TestBackups|TestBackupNow' -v`

Expected: FAIL with `404` because the backup routes are not registered.

- [ ] **Step 3: Implement handlers, settings summary, and UI**

Create `internal/httpapi/backup_handlers.go`:

```go
package httpapi

import (
	"net/http"
	"github.com/rigasyahrul/personal-agent/internal/backup"
)

type backupHandlers struct{ service *backup.Service }
func (h backupHandlers) list(w http.ResponseWriter,r *http.Request) { runs,err:=h.service.List(r.Context()); if err!=nil{writeError(w,http.StatusInternalServerError,"backup_list_failed",err.Error());return}; writeJSON(w,http.StatusOK,map[string]any{"backups":runs}) }
func (h backupHandlers) create(w http.ResponseWriter,r *http.Request) { run,err:=h.service.Run(r.Context()); if err!=nil{writeJSON(w,http.StatusInternalServerError,run);return}; writeJSON(w,http.StatusCreated,run) }
```

Register `GET /api/v1/backups` behind authentication and `POST /api/v1/backups` behind authentication plus CSRF in `server.go`. Ensure the generic mutation-barrier middleware excludes this POST: `Service.Run` itself acquires the exclusive lock, so wrapping it in `Barrier.Mutate` would deadlock.

In `settings_handlers.go`, list runs and add:

```go
type backupSummary struct { LastSuccess *domain.BackupRun `json:"last_success"`; LastFailure *domain.BackupRun `json:"last_failure"` }
func summarizeBackups(runs []domain.BackupRun) backupSummary {
	var s backupSummary
	for i:=range runs { r:=runs[i]; if r.Status=="succeeded" && s.LastSuccess==nil{s.LastSuccess=&r}; if r.Status=="failed" && s.LastFailure==nil{s.LastFailure=&r} }
	return s
}
```

Expose it as the `backup` property of the settings response. In `web/js/api.js`, add `listBackups: () => request('/api/v1/backups')` and `backupNow: () => request('/api/v1/backups', {method:'POST', body:'{}'})`. In `web/js/pages/settings.js`, render “Never backed up” when `last_success` is null, otherwise “Last successful backup: <completed_at>”; render “Last attempt failed: <error>” when the newest failure is newer than the last success; and wire a disabled-while-running “Backup now” button that refreshes settings and displays the API error.

- [ ] **Step 4: Run API tests and static checks**

Run: `gofmt -w internal/httpapi && go test ./internal/httpapi && node --check web/js/api.js && node --check web/js/pages/settings.js`

Expected: PASS; unauthenticated and missing-CSRF requests remain rejected, while backup creation/list/settings status work.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/backup_handlers.go internal/httpapi/backup_handlers_test.go internal/httpapi/server.go internal/httpapi/settings_handlers.go web/js/api.js web/js/pages/settings.js
git commit -m "feat: expose backup controls and status"
```

### Task 36: Document and automate the restore drill

**Restore errata:** Follow Canonical backup layout. Automated test restores into fresh `PA_DATA_DIR` (`db/personal-agent.sqlite`, `files/**`) and opens via `database.Open` + file bytes and/or HTTP note get. Ignore snippets leaving `database.sqlite` at restore root.


**Files:**
- Create: `docs/ops/backup-restore.md`
- Modify: `internal/backup/backup_test.go`

**Interfaces:**
- Consumes: Task 33's bundle format and manifest checksum contract.
- Produces: an operator stop/verify/restore/start drill and acceptance coverage proving a restored bundle opens as SQLite and contains a known indexed note and source body.

- [ ] **Step 1: Write the failing restore acceptance test**

Append a `TestRestoreDrillFindsKnownNote` test to `internal/backup/backup_test.go`. Seed project `p1`, write `global/projects/p1/source/known.md`, calculate its SHA-256, insert a ready `notes` row for that path, run a backup, extract only names listed in the manifest into a fresh restore directory using `filepath.Clean` plus a prefix check, open `database.sqlite`, query `notes` by ID, and assert the restored source bytes and metadata hash agree. Initially call a not-yet-created test helper `restoreBundle(t, run.LocalPath, restoreDir)` so the test is red.

```go
func TestRestoreDrillFindsKnownNote(t *testing.T) {
	ctx:=context.Background(); dataDir:=t.TempDir(); db,err:=dbopen.Open(filepath.Join(dataDir, "db", "personal-agent.sqlite")); if err!=nil{t.Fatal(err)}; t.Cleanup(func(){db.Close()})
	body:=[]byte("# restore me\n"); sum:=sha256.Sum256(body); source:=filepath.Join(dataDir,"global","projects","p1","source"); if err:=os.MkdirAll(source,0o700);err!=nil{t.Fatal(err)}; if err:=os.WriteFile(filepath.Join(source,"known.md"),body,0o600);err!=nil{t.Fatal(err)}
	_,err=db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','Known','2026-08-12T10:00:00Z','2026-08-12T10:00:00Z'); INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n1','p1','known.md',?,?, 'ready',1,'2026-08-12T10:00:00Z','2026-08-12T10:00:00Z')`,hex.EncodeToString(sum[:]),len(body)); if err!=nil{t.Fatal(err)}
	svc:=backup.NewService(db,dataDir,&backup.Barrier{},&clock.FakeClock{T:time.Date(2026,8,12,10,0,0,0,time.UTC)},nil); run,err:=svc.Run(ctx); if err!=nil{t.Fatal(err)}
	restoreDir:=t.TempDir(); restoreBundle(t,run.LocalPath,restoreDir)
	restored,err:=sql.Open("sqlite",filepath.Join(restoreDir,"database.sqlite")); if err!=nil{t.Fatal(err)}; defer restored.Close()
	var path,hash string; if err:=restored.QueryRow(`SELECT relative_path,content_sha256 FROM notes WHERE id='n1' AND status='ready'`).Scan(&path,&hash);err!=nil{t.Fatal(err)}
	restoredBody,err:=os.ReadFile(filepath.Join(restoreDir,"global","projects","p1","source",filepath.FromSlash(path)));if err!=nil{t.Fatal(err)}; got:=sha256.Sum256(restoredBody)
	if hash!=hex.EncodeToString(got[:]) || string(restoredBody)!=string(body){t.Fatal("restored note failed integrity check")}
}
```

- [ ] **Step 2: Run the restore test to verify it fails**

Run: `go test ./internal/backup -run TestRestoreDrillFindsKnownNote -v`

Expected: FAIL to compile with `undefined: restoreBundle`.

- [ ] **Step 3: Add the safe extraction helper and operator procedure**

Implement `restoreBundle` in the test file by reading and validating `manifest.json` first, rejecting absolute names or names whose cleaned form starts with `..`, verifying every payload checksum before writing it beneath `restoreDir`, mapping `files/<relative>` to `<restoreDir>/<relative>`, and writing `database.sqlite` at the restore root. This test-only extractor intentionally mirrors the documented manual verification without adding a v1 restore API.

Create `docs/ops/backup-restore.md` with these exact operational sections and commands:

```markdown
# Backup and restore

Backups are immutable `.tar.gz` bundles in `$PA_DATA_DIR/backups`. With no `PA_BACKUP_S3_BUCKET`, a verified local bundle is a successful backup. With a bucket configured, success means the same bundle was uploaded. RPO is the time since the last successful run; for a daily run, worst-case RPO is 24 hours. RTO depends on bundle size and operator download/verification time.

## Restore drill

1. In Settings, run **Backup now** and confirm its status is `succeeded`. Record its bundle path or S3 object key and `manifest_hash`.
2. Stop writes and the application: `docker compose -f deploy/docker-compose.yml stop personal-agent` (use the actual application service name from the Compose file if it differs).
3. Preserve the current volume: `cp -a "$PA_DATA_DIR" "${PA_DATA_DIR}.before-restore"`.
4. Download the selected S3 object when applicable. Extract the bundle into an empty temporary directory, never over the live volume: `mkdir -p /tmp/pa-restore && tar -xzf BACKUP.tar.gz -C /tmp/pa-restore`.
5. Recompute `sha256sum` for every file named by `manifest.json`; compare each result to `files[name]`. Recompute SHA-256 over the exact `manifest.json` bytes and compare it with the recorded `manifest_hash`. Abort on a missing, extra, or mismatched payload.
6. Verify the database before replacement: `sqlite3 /tmp/pa-restore/database.sqlite 'PRAGMA integrity_check;'`; the only output must be `ok`.
7. Build a fresh data directory by placing `database.sqlite` at `$PA_DATA_DIR/db/personal-agent.sqlite` and copying the contents under `/tmp/pa-restore/files/` to the same relative paths beneath `$PA_DATA_DIR`. Do not restore the bundle's `backups/` directory and remove stale `db/personal-agent.sqlite-wal` or `db/personal-agent.sqlite-shm` files.
8. Start the application: `docker compose -f deploy/docker-compose.yml start personal-agent`.
9. Verify `/health`, sign in, open a known note, and confirm its body renders without an integrity error. Confirm projects, review queue, and the latest durable operation states are present.
10. Record drill date, backup run ID, cutoff, manifest hash, elapsed restore time, and verification result. If any check fails, stop the app, restore `${PA_DATA_DIR}.before-restore`, and investigate before deleting either copy.

## Automated acceptance drill

`go test ./internal/backup -run TestRestoreDrillFindsKnownNote -v` creates a bundle, restores it into a fresh directory, opens the restored SQLite database, finds a known ready Note, and verifies its source body's SHA-256. Run it after every bundle-format change.
```

- [ ] **Step 4: Run restore and full backup verification**

Run: `gofmt -w internal/backup/backup_test.go && go test ./internal/backup -v && go test ./...`

Expected: PASS, including `TestRestoreDrillFindsKnownNote`; the full suite confirms the shared mutation barrier did not regress publication, review, session, or HTTP mutations.

- [ ] **Step 5: Commit**

```bash
git add docs/ops/backup-restore.md internal/backup/backup_test.go
git commit -m "docs: add verified backup restore drill"
```

### Phase self-check

- Spec §5 `BackupRun`: running/succeeded/failed lifecycle and all local/upload/manifest/error fields are persisted.
- Spec §9 F8 and §12: the shared mutation barrier drains durable file operations, SQLite uses its online backup API, files and operation state are bundled, the manifest is written last, and upload gates success only when configured.
- The application remains functional with `PA_BACKUP_S3_BUCKET` unset; local verified completion is sufficient and AWS configuration is not loaded.
- HTTP `GET/POST /api/v1/backups`, Settings never/success/failure states, authentication, and CSRF are covered.
- Spec §13 #10: the automated restore drill opens the restored database, finds a known Note, and verifies the restored source body; the operator stop/restore/start procedure is documented.


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

**Runner errata:** Use Canonical `Runner.Start` / `ErrBusy` / `Clock`. No `Runner.Providers` map.


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

**Harness errata:** Use Canonical **Acceptance harness** only (`apitest.NewFixture`, fake Provider, `review.BiteGenerator`, `backup.Sink`, in-process workspace seeding). If a code block references `workspaceWrite`, undefined constructors, or non-canonical types, do not copy those symbols — rewrite against the harness. Keep the eleven `TestAcceptance0x...` names.


**Files:**
- Follow Canonical contracts for `openTestDB`, `BiteGenerator`, backup paths, and in-process workspace seeding (no HTTP workspace write).
- **Files:**
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
		if _, err := tools.Open(wsRoot).WriteFile(p, []byte("stolen")); err == nil { t.Fatalf("accepted %q", p) }
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
      PA_ADDR: :8080
      PA_SECURE_COOKIES: "false"
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


---

## Spec coverage matrix

| Spec area | Tasks |
|-----------|-------|
| §3 Compose/Go/SQLite/UI | 1, 6–8 |
| §4 FS layout | 3, 9, 15, 21 |
| §5 Data model + scope | 3, 10, 15–17 |
| §5 Owner auth | 4–5, 40 |
| §6 Publication machine | 13, 25–26, 31 |
| §7 Review / SM-2 | 27–30, 32 |
| §8 Screens | 7, 14, 20, 24, 32, 35 |
| §9 Flows F0–F10 | 5–8, 10–16, 18–19, 25, 28–36, 39–40 |
| §10 Concurrency | 17–18, 38–39 |
| §11 Security | 2, 5, 21, 37, 40 |
| §12 Backup/restore | 33–36 |
| §13 Acceptance 1–11 | 41 (+ supporting tasks) |
| §15 Open decisions | Header Resolved Defaults |

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-08-12-personal-agent-v1.md`.

**For implementers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans`.

Do not implement application code until execution is explicitly approved.

**Spec SoT:** `docs/superpowers/specs/2026-08-12-personal-agent-design.md`  
**Coordination scratch:** `docs/memory/` (optional; this plan file is authoritative)
