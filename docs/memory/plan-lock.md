# Plan lock — personal-agent v1

**Frozen for all plan-draft subagents.** Do not invent conflicting package paths, type names, or limits.
**Spec SoT:** `docs/superpowers/specs/2026-08-12-personal-agent-design.md`  
**Date:** 2026-08-12

---

## Writing-plans format (every task)

Each task MUST have:

```markdown
### Task N: Title

**Files:**
- Create: `path`
- Modify: `path`
- Test: `path`

**Interfaces:**
- Consumes: ...
- Produces: ...

- [ ] **Step 1: Write the failing test**
```go
// full test code
```

- [ ] **Step 2: Run test to verify it fails**
Run: `go test ./... -run TestName -v`
Expected: FAIL with ...

- [ ] **Step 3: Write minimal implementation**
```go
// full impl code
```

- [ ] **Step 4: Run test to verify it passes**
Run: `...`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add ...
git commit -m "..."
```
```

Rules:
- No TBD/TODO/placeholders/"similar to Task N"
- Full code in steps (tests + impl snippets an engineer can paste)
- TDD: red → green → commit
- One commit per task
- Bite-sized: each step ~2–5 minutes of work when possible; group only when a single deliverable needs scaffolding
- Assume greenfield except orb setup, Grok plugin, Superpowers, approved spec

---

## Resolved defaults (§15 open decisions)

| Decision | v1 lock |
|----------|---------|
| Go version | **1.24+** (`os.Root` requires 1.24; orb `.agents/setup` installs from go.dev if distro older) |
| Password hash | **argon2id** (golang.org/x/crypto/argon2); PHC-style encoded hash string in DB |
| Module path | `github.com/rigasyahrul/personal-agent` |
| SQLite driver | `modernc.org/sqlite` (pure Go, no CGO) |
| HTTP router | stdlib `net/http` ServeMux (Go 1.22+ patterns) |
| UI | Static SPA under `web/` (no bundler): vanilla JS modules + CSS; Go embeds `web/` via `embed.FS` in production; dev may serve `web/` from disk |
| Reverse proxy | **Caddy** in Compose (`caddy:2-alpine`) for TLS when domain set; direct `:8080` for localhost |
| Auth | Single owner; argon2id password hash in DB; bootstrap via `BOOTSTRAP_TOKEN` env (one-time set-password); session cookie `pa_session` HttpOnly Secure SameSite=Lax; CSRF double-submit cookie `pa_csrf` |
| IDs | UUIDv7 strings (or UUIDv4 if v7 lib heavy) — plan uses **UUIDv4** via `github.com/google/uuid` for simplicity |
| JSON times | RFC3339 UTC in API; owner timezone IANA string in settings for review "today" |
| Max `.md` body | **1 MiB** (1_048_576 bytes) |
| Max path length | **512** bytes UTF-8 |
| Max path depth | **16** components under root |
| Max component length | **255** bytes |
| Workspace tool grants default | `workspace_files: false` |
| SM-2-lite (`sm2-lite-v1`) | Again→10m (due_at +10m, interval_days stays or reset per table); Hard/Good/Easy use day intervals — see table below |
| Bite generator | JSON schema: `{ "bites": [ { "prompt": string, "answer": string } ] }` max 8 bites; `generator_version = "bites-v1"` |
| Vault UI | Optional `vault_id` / vault name on create project only; no vault browser |
| FTS | Deferred; no FTS in v1 |
| Agent provider v1 | OpenAI-compatible HTTP chat completions (`OPENAI_API_KEY`, `OPENAI_BASE_URL` optional); model list from env `PA_MODELS=provider:model_id,...` |
| Port | API listens `:8080` |
| Data dir | env `PA_DATA_DIR` default `./data` |
| Test data | Each test uses `t.TempDir()` |

### SM-2-lite-v1 schedule

Ratings: `again` | `hard` | `good` | `easy`

State fields: `stage` (int, starts 0), `interval_days` (float), `ease_factor` (float, default **2.5**, min **1.3**), `reps`, `lapses`, `due_at`, `last_reviewed_at`.

| Rating | Effect |
|--------|--------|
| again | `lapses++`; `reps=0`; `stage=0`; `interval_days=0`; `due_at = now+10m`; `ease_factor = max(1.3, ease_factor-0.2)` |
| hard | `reps++`; `ease_factor = max(1.3, ease_factor-0.15)`; if stage==0: interval=0.5d; else interval = interval_days * 1.2; `stage=max(stage,1)`; `due_at=now+interval` |
| good | `reps++`; if stage==0: interval=1d, stage=1; elif stage==1: interval=3d, stage=2; else interval = interval_days * ease_factor; `due_at=now+interval` |
| easy | `reps++`; `ease_factor += 0.15`; if stage<2: interval=4d, stage=2; else interval = interval_days * ease_factor * 1.3; `due_at=now+interval` |

Whole-note first create: stage=0, interval_days=0, ease_factor=2.5, reps=0, lapses=0, due_at=now (immediately due).

---

## Publication statuses (exact)

```
accepted → frozen → path_reserved → published_fs → finalized → review_enqueued → completed
failed (terminal error)
```

Note statuses: `pending` | `ready` | `failed`  
ReviewPending: `pending` | `leased` | `completed` | `failed`  
AgentRun: `queued` | `running` | `completed` | `failed` | `cancelled`  
Session status: `active` | `terminal`  
BackupRun: `running` | `succeeded` | `failed`

---

## Package / file map (create these paths)

```
go.mod
go.sum
cmd/personal-agent/main.go          # entrypoint
internal/config/config.go           # env load
internal/ids/ids.go                 # NewID()
internal/clock/clock.go             # Clock interface for tests
internal/paths/paths.go             # validate relative paths; limits
internal/paths/paths_test.go
internal/fsroot/root.go             # rooted FS helpers over os.Root / fallback
internal/fsroot/root_test.go
internal/db/db.go                   # open SQLite, WAL, migrations runner
internal/db/migrations/001_init.sql
internal/db/migrate_test.go
internal/domain/models.go           # shared structs / enums
internal/auth/password.go
internal/auth/password_test.go
internal/auth/session.go
internal/auth/session_test.go
internal/auth/csrf.go
internal/auth/bootstrap.go
internal/auth/bootstrap_test.go
internal/store/vaults.go
internal/store/projects.go
internal/store/sessions.go
internal/store/notes.go
internal/store/messages.go
internal/store/runs.go
internal/store/promote.go
internal/store/direct.go
internal/store/review.go
internal/store/backup.go
internal/store/settings.go
internal/store/*_test.go
internal/layout/layout.go           # derive workspace/source paths from home+ids
internal/layout/layout_test.go
internal/publish/machine.go         # promote + direct shared machine
internal/publish/machine_test.go
internal/publish/recover.go
internal/review/scheduler.go        # sm2-lite-v1
internal/review/scheduler_test.go
internal/review/queue.go
internal/review/bites.go            # bite job worker
internal/review/bites_test.go
internal/agent/provider.go          # Provider interface
internal/agent/openai_compat.go
internal/agent/runner.go            # run loop, idempotency, single active run
internal/agent/runner_test.go
internal/agent/tools/workspace.go   # rooted tools
internal/agent/tools/workspace_test.go
internal/backup/backup.go
internal/backup/backup_test.go
internal/backup/s3.go
internal/httpapi/server.go          # mux, middleware
internal/httpapi/middleware.go      # auth, csrf, request_id
internal/httpapi/health.go
internal/httpapi/auth_handlers.go
internal/httpapi/project_handlers.go
internal/httpapi/note_handlers.go
internal/httpapi/session_handlers.go
internal/httpapi/chat_handlers.go
internal/httpapi/promote_handlers.go
internal/httpapi/review_handlers.go
internal/httpapi/settings_handlers.go
internal/httpapi/backup_handlers.go
internal/httpapi/*_test.go          # httptest integration where needed
internal/app/app.go                 # wire dependencies
web/index.html
web/css/app.css
web/js/api.js
web/js/router.js
web/js/app.js
web/js/pages/home.js
web/js/pages/project.js
web/js/pages/notes.js
web/js/pages/sessions.js
web/js/pages/review.js
web/js/pages/settings.js
web/js/components/status-badges.js
web/js/components/markdown.js
deploy/docker-compose.yml
deploy/Dockerfile
deploy/Caddyfile
deploy/.env.example
docs/ops/backup-restore.md
docs/ops/deploy.md
Makefile
README.md
.amp/services.yaml                  # optional dev server portal
```

Tests live next to packages (`*_test.go`). Integration tests under `internal/httpapi` and `internal/publish` use temp dirs.

---

## Shared type signatures (plan code must match)

```go
// internal/ids
func NewID() string // uuid v4

// internal/clock
type Clock interface{ Now() time.Time }
type RealClock struct{}
func (RealClock) Now() time.Time
type FakeClock struct{ T time.Time }
func (f *FakeClock) Now() time.Time
func (f *FakeClock) Advance(d time.Duration)

// internal/paths
type PathError struct{ Code, Message string }
func ValidateRelPath(p string) (clean string, err error) // rejects abs, .., empty, controls
const MaxPathBytes = 512
const MaxDepth = 16
const MaxComponentBytes = 255
const MaxMarkdownBytes = 1 << 20

// internal/layout
type SessionHome string // "global" | "vault" | "project"
func ProjectRoot(dataDir, vaultID, projectID string) string // vaultID "" => global/projects/{id}
func SourceDir(projectRoot string) string
func SessionWorkspace(dataDir string, home SessionHome, vaultID, projectID, sessionID string) string

// internal/domain (selected)
type NoteStatus string // pending, ready, failed
type ReviewMode string // none, whole, bites
type Rating string // again, hard, good, easy

// internal/publish
type PublishInput struct {
    OpID, RequestKey, RequestFingerprint string
    Kind string // "promote" | "direct"
    SessionID string // promote only
    WorkspacePath string // promote only, rel
    Body []byte // direct only
    TargetProjectID, TargetRelPath string
    ReviewMode ReviewMode
    NoteID string // preallocated
}
type Machine struct {
    DB *sql.DB
    DataDir string
    Clock clock.Clock
}
func (m *Machine) Run(ctx context.Context, in PublishInput) (opStatus string, noteID string, err error)
func (m *Machine) RecoverAll(ctx context.Context) error

// internal/review
func ApplyRating(item ReviewItemState, rating Rating, now time.Time) ReviewItemState
type ReviewItemState struct {
    Stage int
    IntervalDays float64
    EaseFactor float64
    Reps, Lapses int
    DueAt time.Time
}

// internal/agent
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
type Runner struct { /* DB, Provider, Tools, Clock */ }
func (r *Runner) Start(ctx context.Context, sessionID, requestKey string, userMessage string) (runID string, err error)

// internal/auth
func HashPassword(pw string) (string, error)
func CheckPassword(hash, pw string) bool
func NewSessionToken() string
```

---

## API surface (v1) — prefix `/api/v1`

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/health` | liveness + storage writable (no auth) |
| GET | `/api/v1/setup/status` | bootstrap state |
| POST | `/api/v1/setup/bootstrap` | set owner password with bootstrap token |
| POST | `/api/v1/auth/login` | login |
| POST | `/api/v1/auth/logout` | logout |
| GET | `/api/v1/auth/me` | current owner |
| GET/PUT | `/api/v1/settings` | timezone, defaults |
| GET/POST | `/api/v1/vaults` | list/create (minimal) |
| GET/POST | `/api/v1/projects` | list/create |
| GET | `/api/v1/projects/{id}` | overview aggregates |
| GET | `/api/v1/projects/{id}/tree` | source tree |
| GET | `/api/v1/notes/{id}` | metadata + body |
| POST | `/api/v1/projects/{id}/folders` | mkdir under source |
| POST | `/api/v1/projects/{id}/direct-notes` | DirectCreateOperation |
| GET/POST | `/api/v1/projects/{id}/sessions` | list/create session |
| GET | `/api/v1/sessions/{id}` | session detail |
| DELETE | `/api/v1/sessions/{id}` | terminal + workspace delete |
| GET | `/api/v1/sessions/{id}/messages` | list messages |
| POST | `/api/v1/sessions/{id}/messages` | user message + start run (idempotency key) |
| GET | `/api/v1/sessions/{id}/runs/current` | current/non-terminal run |
| GET | `/api/v1/sessions/{id}/workspace/tree` | workspace tree |
| GET | `/api/v1/sessions/{id}/workspace/file` | read file `?path=` |
| POST | `/api/v1/sessions/{id}/promote` | PromoteOperation |
| GET | `/api/v1/operations/{id}` | promote/direct status |
| GET | `/api/v1/review/queue` | `?scope=all\|project:{id}` |
| POST | `/api/v1/review/items/{id}/rate` | rating + idempotency |
| POST | `/api/v1/review/items/{id}/suspend` | suspend |
| POST | `/api/v1/review/pending/{id}/retry` | retry bite job |
| GET/POST | `/api/v1/backups` | list / Backup now |
| GET | `/api/v1/home` | home dashboard DTO |

All mutations require auth cookie + CSRF (except bootstrap/login/health as designed).

---

## Task ID ranges (no overlap)

| Phase | Tasks | Draft file |
|-------|-------|------------|
| 0 Header + file map + constraints | (assembler) | `00-header.md` |
| 1 Skeleton | 1–8 | `01-skeleton.md` |
| 2 Projects + source | 9–14 | `02-projects-source.md` |
| 3 Sessions + chat | 15–20 | `03-sessions-chat.md` |
| 4 Workspace tools | 21–24 | `04-workspace-tools.md` |
| 5 Promote + review | 25–32 | `05-promote-review.md` |
| 6 Backup | 33–36 | `06-backup.md` |
| 7 Hardening | 37–42 | `07-hardening.md` |
| 8 Assembly notes | (assembler) | — |

Each phase draft starts with `## Phase N: Title` and contains only its tasks.

---

## Spec coverage checklist (assembler verifies)

- [ ] Compose + single host deploy
- [ ] SQLite WAL + migrations
- [ ] Auth bootstrap + cookies + CSRF
- [ ] Health + empty Home
- [ ] Vaults minimal + projects ± vault
- [ ] FS layout global/vaults/staging
- [ ] Source tree browse + direct create + mkdir
- [ ] Notes read by note_id; integrity hash
- [ ] Sessions project-only UI; schema all homes
- [ ] Scope CHECK + vault_id match project
- [ ] Immutable provider/model
- [ ] Messages + AgentRun single active
- [ ] Workspace tools rooted opt-in
- [ ] Promote machine + recovery
- [ ] Direct create same machine
- [ ] Review whole/bites + sm2-lite + events
- [ ] Bite worker lease/retry
- [ ] Session delete workspace only
- [ ] Backup barrier + S3 optional + restore docs
- [ ] Acceptance tests §13
- [ ] Path limits + no symlink
- [ ] Multi-tab concurrency

---

## Subagent instructions summary

1. Read this lock + full design spec.
2. Write ONLY your phase file under `docs/memory/plan-drafts/`.
3. Follow writing-plans task format exactly.
4. Use exact paths/types from this lock.
5. Include real Go test + impl code in steps.
6. Do not implement application code outside `docs/memory/`.
7. End file with a short "Phase self-check" listing spec sections covered.
