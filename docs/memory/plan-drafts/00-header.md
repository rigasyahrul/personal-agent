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

---
