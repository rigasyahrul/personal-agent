# Personal Agent — Design Spec

**Status:** Approved (2026-08-12) — ready for `writing-plans`  
**Date:** 2026-08-12  
**Repo:** `personal-agent`  
**v1 name:** Thin vertical — self-hosted learning dashboard  

---

## 1. Purpose

Build a **self-hosted, single-tenant personal agent** the user owns end-to-end:

- **Dashboard** in the browser (phone or laptop) against **one live host** (laptop or dedicated VPS + domain).
- **Learning / knowledge** first: projects, freeform source files, chat sessions with filesystem workspaces, promote-to-source, spaced review.
- **Portable data:** local volume is system of record; optional S3-compatible **backups** (not live storage).
- **Swappable AI:** one model per session; provider configured on the server; cancel a subscription without losing data.
- **Not** a multi-tenant SaaS; **not** multi-device file sync; **not** Amp-locked (Amp may call the same API later).

### Longer-term pillars (out of v1 UI)

| Pillar | Direction |
|--------|-----------|
| Personal ops | Global todos (app SoT), optional Google Calendar push; mail read-only later |
| Memory / learning | `memory/`, richer review, teaching loops |
| Research / writing | Capture from links, paste, exports |
| Identity | Reserved `soul/` for preferences / voice (TBD) |

v1 ships only the **learning thin vertical** while the data model leaves room for the rest.

---

## 2. Goals and non-goals

### Goals (v1)

1. Deploy with Docker Compose on a single host; open from any device via `localhost` or the user’s domain.
2. Create **projects** with or without a **vault**.
3. **Source tree** per project: freeform directories and Markdown files (human-navigable, exportable).
4. **Sessions** attached to a project (v1 UI): chat + persistent **workspace directory** on disk for the life of the session.
5. Agent tools (opt-in): read/write/edit/mkdir **inside the session workspace only**.
6. **Promote** workspace files → project `source/` with optional spaced review (whole note or generated bites).
7. **Direct** create Markdown under `source/` via API/UI (no chat required).
8. **Review** queue (project scope or all projects) with ratings and append-only history.
9. Optional **backup** snapshots to S3-compatible storage; app fully works without a bucket.
10. Owner **authentication** suitable for exposing the host on a domain.

### Non-goals (v1)

- Multi-tenant SaaS, concurrent multi-user collaboration
- Live S3 as runtime store; multi-writer sync across machines
- Nested product rules beyond freeform folders under `source/`
- Note **edit** / **delete** UI (deferred with explicit contracts later)
- File attachments / non-`.md` promote
- Global or vault-level session **UI** (schema ready; UI is project sessions only)
- Todos, Google, mail, cross-session chat UI
- Amp as required runtime
- Arbitrary host-side file edits while the app runs (no external reconcile importer)

---

## 3. Architecture

```
Any device (browser)
        │  HTTPS (user domain) or http://localhost
        ▼
┌───────────────────────────────────────────────┐
│  Single host (laptop OR dedicated VPS)        │
│  Docker Compose                               │
│                                               │
│  Dashboard (static/SPA)  ──HTTP──▶  Go API    │
│                                   (sole writer)│
│                                       │       │
│                         ┌─────────────▼─────┐ │
│                         │  Data volume SoT  │ │
│                         │  db/   (SQLite)   │ │
│                         │  files/           │ │
│                         │  staging/         │ │
│                         └─────────────┬─────┘ │
└───────────────────────────────────────┼───────┘
                                        │ optional snapshot job
                                        ▼
                              S3-compatible backup
```

| Rule | Detail |
|------|--------|
| System of record | Host **data volume** (DB + files + staging) |
| Clients | Browser in v1; later CLI/Amp → same API |
| Multi-device | All devices hit **one** deploy; no phone↔laptop file sync |
| S3 | Optional **backup only**; unset bucket = full app works |
| Tenancy | Single owner per deploy |
| Model | Server-configured providers; **immutable** `provider` + `model_id` per session |
| API sole writer | All durable mutations go through the Go API (including “new source file”) |

### Stack defaults (v1)

| Piece | Choice | Rationale |
|-------|--------|-----------|
| Language | Go | Matches orb setup; single binary service |
| DB | **SQLite** (WAL, short txns) | Single-host, sole-writer, simple Compose |
| Files | POSIX volume (Docker named volume or bind mount) | Local FS semantics; NFS not promised |
| UI | Simple web dashboard | Dashboard-first product |
| Backup target | Any S3-compatible API | R2, B2, AWS, MinIO, etc. |

---

## 4. Filesystem layout

```
data/
  db/
    personal-agent.sqlite
  files/
    global/
      sessions/{session_id}/          # home=global workspaces (schema-ready)
      projects/{project_id}/
        source/**                     # freeform knowledge tree
        memory/                       # reserved (later)
        soul/                         # reserved (later)
        sessions/{session_id}/        # home=project, no vault
    vaults/{vault_id}/
      sessions/{session_id}/          # home=vault workspaces (schema-ready)
      projects/{project_id}/
        source/**
        memory/
        soul/
        sessions/{session_id}/        # home=project under vault
  staging/
    promote/{operation_id}/           # backend-owned; not user-visible trees
    direct/{operation_id}/
  backups/
    local/{backup_run_id}/            # staged bundles before/without upload
```

### Layout rules

1. **`source/**`** — freeform directories and files. No forced slug service. Create folders freely.
2. **`memory/`**, **`soul/`** — reserved names under each project; empty or absent in v1; not user-promoted targets for arbitrary use until defined.
3. **Session workspace** — freeform tree under that session’s directory; lives for the life of the session; deleted with the session.
4. **IDs in paths** — `project_id` / `vault_id` / `session_id` directories use stable IDs (not display names) so renames of titles do not move trees. **Note filenames** inside `source/` are user/agent-chosen.
5. **`staging/`** — only the API; never exposed as a browsable “notes” tree.
6. **Export a project** — copy `…/projects/{project_id}/` excluding `sessions/` (optional include); `source/` remains human-readable offline.

---

## 5. Data model

### 5.1 Session scope

| `home` | `vault_id` | `project_id` | Meaning |
|--------|------------|--------------|---------|
| `global` | NULL | NULL | App-level session |
| `vault` | NOT NULL | NULL | Vault-level session |
| `project` | see below | NOT NULL | Project session |

**Project session vault rule**

- If `project.vault_id` IS NULL → `session.vault_id` MUST be NULL.
- If `project.vault_id` IS NOT NULL → `session.vault_id` MUST equal `project.vault_id`.

Enforce with DB `CHECK` on `(home, vault_id, project_id)` shape **plus** trigger or equivalent for null-safe equality with `project.vault_id`. API validates the same.

**Immutability after create:** `home`, `vault_id`, `project_id`, `provider`, `model_id`, `model_parameters` never change.

**v1 UI** creates only `home = project`. Schema supports all three homes.

**Project placement:** `project.vault_id` is **immutable in v1** (no move global ↔ vault).

### 5.2 Entities

#### Vault
- `id`, `name`, `created_at`, `updated_at`
- Restrict delete while projects/sessions reference it (v1: no delete UI required)

#### Project
- `id`, `vault_id` NULLABLE (immutable), `name`, `created_at`, `updated_at`
- Directory under `global/projects/{id}` or `vaults/{vault_id}/projects/{id}`

#### Session
- `id`, `home`, `vault_id`, `project_id` (per scope)
- `status`: `active` | `terminal` (terminal during/after delete)
- `provider`, `model_id`, `model_parameters_json` (immutable)
- `tool_grants_json` e.g. `{ "workspace_files": false }` default
- `title`, `created_at`, `updated_at`, `deleted_at?`
- Workspace path derived from home + ids (never user-supplied absolute paths)

#### AgentRun
- `id`, `session_id`, `request_key` (idempotency)
- `status`: `queued` | `running` | `completed` | `failed` | `cancelled`
- `started_at`, `completed_at`, `error?`
- **At most one** non-terminal run per session (`UNIQUE` partial index on active/running)

#### Message
- `id`, `session_id`, `run_id?`
- `sequence` UNIQUE `(session_id, sequence)`
- `role`: `system` | `user` | `assistant` | `tool`
- `content`, `tool_calls_json?`, `tool_call_id?`
- `status`, `created_at`

#### Note (source index only — **no body column**)
- `id` (stable identity)
- `project_id`
- `relative_path` — logical POSIX path under that project’s `source/` (e.g. `articles/intro.md`)
- `content_sha256`, `byte_size`
- `status`: `pending` | `ready` | `failed`
- `origin_session_id?` (nullable; session may later be tombstoned)
- `origin_workspace_path?`
- `revision` (monotonic; v1 create-only bumps from 0→1)
- `created_at`, `updated_at`
- **UNIQUE** active `(project_id, relative_path)`
- Reads: open file from volume; hash mismatch → integrity error (do not silently rewrite metadata)
- Soft-delete / edit: **not in v1 UI**; if added later, preserve `id` and review history contracts

#### PromoteOperation
- `id`, `request_key` UNIQUE, `request_fingerprint` (hash of immutable request payload)
- `session_id`, `workspace_path` (relative to session root)
- `target_project_id`, `target_relative_path` (under `source/`)
- `review_mode`: `none` | `whole` | `bites`
- `note_id` (allocated at freeze)
- `frozen_sha256`, `frozen_size`
- `status`: see §6
- `error?`, `created_at`, `updated_at`
- **One operation = one source file.** UI batch = N operations.

#### DirectCreateOperation
- Same publication primitive as promote; input bytes from request body instead of workspace freeze
- Idempotency key + fingerprint; allocates `note_id`; same status machine ending in ready Note + optional review

#### ReviewPending (bite job)
- `id`, `note_id`, `source_sha256`, `generator_version`
- `status`: `pending` | `leased` | `completed` | `failed`
- `attempts`, `lease_until?`, `last_error?`
- Uniqueness: one active generation per `(note_id, source_sha256, generator_version)` as designed

#### ReviewItem
- `id`, `project_id`, `note_id`
- `kind`: `whole` | `bite`
- `source_sha256` / `source_revision` snapshot of what was reviewed against at creation
- `prompt`, `answer?` (**required for bites**; snapshot — never silently rewrite on note change)
- `generation_id?`, `ordinal?` (bites)
- Scheduling: `stage`, `due_at`, `interval_days`, `ease_factor`, `reps`, `lapses`, `last_reviewed_at?`
- `row_version` (optimistic concurrency)
- `status`: `active` | `suspended` | `retired`
- `scheduler_version` e.g. `sm2-lite-v1`
- Uniqueness:
  - whole: one active item per note + source revision (or superseded policy — prefer supersede prior generation, keep events)
  - bites: unique `(generation_id, ordinal)`

#### ReviewEvent (append-only)
- `id`, `review_item_id`
- `request_key` UNIQUE (rating idempotency)
- `rating`: `again` | `hard` | `good` | `easy`
- `previous_state_json`, `resulting_state_json`
- `scheduler_version`, `reviewed_at`, `duration_ms?`

#### BackupRun
- `id`, `status`: `running` | `succeeded` | `failed`
- `cutoff_at`, `local_path?`, `object_key?`, `manifest_hash?`
- `started_at`, `completed_at?`, `error?`

#### Owner / auth (minimal)
- Single owner credentials (or bootstrap token → set password)
- Session cookies (secure, HTTP-only, SameSite) for browser
- CSRF protection on mutating routes
- Secrets (model keys, S3) in env / secret files — not in DB plaintext if avoidable

---

## 6. Publication (promote & direct create)

Shared **recoverable** state machine. UI departure does **not** cancel work; badges read durable status.

### Statuses

`accepted` → `frozen` → `path_reserved` → `published_fs` → `finalized` → (`review_enqueued` |) `completed`  
Failures: `failed` with `error`; bites may leave note `ready` and job retryable.

### Steps

1. **Validate** session (active), paths, target project rules, review mode, idempotency key.
2. **Insert operation** with immutable fingerprint; allocate `note_id`.
   - Same key + same fingerprint → return existing op.
   - Same key + different fingerprint → **409 conflict**.
3. **Freeze input**
   - Promote: copy workspace file bytes into `staging/{op}/` (workspace writes use atomic replace so freeze sees one version).
   - Direct: write request body into staging.
   - Record `frozen_sha256`, `frozen_size`.
4. **Reserve path in DB** — Note `pending` for `(project_id, relative_path)`; collision with existing ready/pending → **409 never overwrite**.
5. **Publish file** — write temp on **same volume**, fsync, **atomic no-clobber** rename into `source/{relative_path}`.
6. **Finalize DB (one txn)** — Note `ready` + hashes/sizes; mark op complete for source; if `whole`, insert ReviewItem; if `bites`, insert ReviewPending.
7. **Bite worker** — lease `ReviewPending` rows; generate bites; insert items; mark job completed. Failure: note remains; job retryable with backoff; never delete source note.
8. **Startup recovery** — scan non-terminal ops; resume from durable status (reconcile FS vs DB).

### Target project rules

| Session home | Promote target |
|--------------|----------------|
| `project` (v1) | **Only** that project |
| `vault` (later) | Project in that vault only |
| `global` (later) | Any project after explicit pick |

### Path contract (security)

- Paths are UTF-8, relative, logical POSIX components under the trusted root (`source/` or session workspace).
- **Reject:** empty, absolute, `.`, `..`, NUL/controls, empty components, reserved backend names (`memory`/`soul` as promote destinations only if explicitly allowed later — v1 promote target is under `source/` only).
- **v1 promote/direct:** regular files ending in `.md` only; max size / depth / path length limits (concrete numbers in implementation plan).
- **No symlinks**, devices, sockets, FIFOs in managed trees for v1.
- Operations use **rooted** FS APIs (e.g. Go `os.Root` / `openat2`); never “validate string then open unconstrained absolute path.”
- Model tool arguments are **untrusted**.
- **No arbitrary shell tool** as sandbox.
- App URLs and review references use **`note_id`**, not path.

---

## 7. Review semantics

| Topic | v1 lock |
|-------|---------|
| Scheduler | `sm2-lite-v1` (Again/Hard/Good/Easy); exact intervals in implementation plan |
| “Today” | Owner timezone from settings (default UTC until set) |
| Caught up | No `active` items with `due_at <= now` in the **explicit** selected scope |
| Queue | Single queue for new + scheduled; optional UI label “Today’s lesson” on first 1–2 cards — **not** a second mode |
| Whole item | Prompt + open current file body in panel; store creation `source_sha256` |
| Bite item | Snapshot prompt/answer; never auto-rewrite if file changes |
| Rating | One DB txn: bump `row_version`, update schedule, insert `ReviewEvent` with idempotency key |
| Scope | `project:{id}` or `all`; never silently widen; URL/state carries scope |
| AI down | Existing items reviewable; new bite generation disabled; whole-note OK |

---

## 8. Screens and navigation (v1)

```
Home
├── Projects list (+ New project)
├── Last activity (derived)
├── Today’s review (all projects)
├── Setup / health strip
└── Settings

Project
├── Overview          ← default on open
├── Notes             ← source/** tree
├── Sessions          ← list + chat + workspace
└── Review            ← this project scope
```

No separate multi-vault gateway required for v1; vault is optional metadata on projects + future session homes.

### Home
- Project cards: name, vault badge if any, note count, session count, due count
- Due today + Start review (all-project scope)
- Open last project
- First-run CTAs when empty / misconfigured

### Project overview
- Recent source files, recent sessions (with status badges), due count
- Actions: New session, New source file, Review

### Notes
- Directory tree under `source/**`
- Open `.md` → render from disk
- New file / new folder via API
- **No edit, no delete** in v1

### Sessions / Chat
- List: title, model badge, file count, promote badges
- New session: title, model once, `workspace_files` default **off**
- Chat messages in sequence order
- If tools on: file tree + agent edits scoped to workspace root
- **Save to source** on selected `.md` → destination path + review mode modal
- One active agent run per session; reconnect does not start a second run

### Review
- Shared component; scope chip: This project | All projects
- Card UX: bite reveal / whole panel; rate; skip/suspend
- Caught-up empty state
- Explicit “Continue all projects” after project queue empties (optional)

### Settings / first-run
- Storage health (volume writable)
- Owner auth bootstrap
- Default AI provider/model; API keys via env/secrets
- Backup: bucket optional, schedule, last success/fail, Backup now
- Owner timezone

### Status badges (outside chat too)
- Promoting… / Promote failed — Retry  
- Note saved; cards pending… / Cards failed — Retry cards  
- Ready  

Published notes appear in Notes as soon as FS+DB finalize, even if bites still pending.

---

## 9. Key flows

### F0 — First run
1. Volume writable?  
2. Owner bootstrap (env secret or loopback-only setup until password set) — **required before domain exposure**.  
3. AI configured? (optional for read paths)  
4. Any projects? → else create first project (with or without vault).  

### F1 — Create project
Name + optional vault → directories created → overview.

### F2 — New session (v1 from project)
Title, model, tools → enforce scope invariants → create workspace dir → chat.

### F3 — Chat + workspace
Tools off: messages only. Tools on: rooted workspace tools. AI down: read-only history/files; no send.

### F4 — Promote
Select workspace `.md` → modal path + review → PromoteOperation machine (§6) → badges.

### F5 — Direct source file
Notes → path + body → DirectCreateOperation → same publication primitive.

### F6 — Browse source
Tree navigate; render markdown; integrity error if hash mismatch / missing file.

### F7 — Review
Entry with explicit scope → queue → rate idempotently → caught up.

### F8 — Backup
If bucket configured: mutation barrier → SQLite backup API → files + ops state bundle → manifest/checksums → upload → mark BackupRun.  
UI: never backed up | last success | last attempt failed.  
Restore: operator procedure in docs; test drill required before calling backup “done.”

### F9 — Delete session
Mark terminal → block runs/tools/promotes → delete workspace tree only → tombstone session; **never** delete source notes or cascade review history. In-flight promotes: complete or fail cleanly without orphan ready notes.

### F10 — Failure catalog (non-exhaustive)
Path conflict 409 · stale/missing workspace file · disk full · hash mismatch · session busy/terminal · generation failure · provider failure · backup failure · auth expired · first-run takeover blocked · CSRF failure.

---

## 10. Concurrency

- Single-tenant **owner**, but **multi-tab / multi-device browsers** are normal.
- Serialize: one active `AgentRun` per session; workspace mutations vs promote vs session delete.
- Idempotency keys on promote, direct create, review rate, agent run.
- Do not rely on UI disabling alone.

---

## 11. Security

| Topic | v1 requirement |
|-------|----------------|
| Exposure | Assume VPS may be on the public internet |
| Auth | Owner login; secure cookies; CSRF on mutations |
| Bootstrap | Prevent open signup / first-request admin takeover |
| Secrets | Model + S3 credentials via environment/secret mounts |
| Tools | Workspace rooted; no host shell; no path escape |
| HTTPS | Terminated at reverse proxy or Compose companion; required for real domain deploy |

---

## 12. Backup and restore

1. Acquire application mutation barrier.  
2. Drain/wait file ops to durable states.  
3. SQLite online backup API → consistent DB file.  
4. Capture files + staging/ops metadata into immutable bundle.  
5. Manifest + checksums last.  
6. Release barrier.  
7. Upload; only then `BackupRun=succeeded`.  

**RPO:** time since last successful snapshot (e.g. up to 24h on daily cron).  
**RTO / restore:** documented stop → restore volume → start; verify with drill.

---

## 13. Testing invariants (acceptance)

1. Promote retry same key → one Note, one review set.  
2. Crash after FS publish before DB finalize → recovery converges.  
3. Bite job fails → Note remains; retry does not duplicate Note.  
4. Invalid session scope rejected at API and DB.  
5. Path `../` / symlink escape rejected.  
6. Destination exists → 409, no overwrite.  
7. Session delete removes workspace only.  
8. Rating retry with same key → one event.  
9. Two tabs cannot run two agent runs on one session.  
10. Backup restore drill succeeds from last bundle.  
11. Unauthenticated mutating request on “secured” mode rejected.

---

## 14. Implementation phases (indicative — detailed plan later)

1. **Skeleton** — Compose, SQLite, auth bootstrap, health, empty Home.  
2. **Projects + source tree** — create project, browse/create `.md`, read render.  
3. **Sessions + chat** — messages, model config, agent run loop.  
4. **Workspace tools** — rooted FS tools, file tree UI.  
5. **Promote + review** — publication machine, whole/bites, review UI.  
6. **Backup** — snapshot + optional S3 + restore docs.  
7. **Hardening** — recovery tests, path fuzz, multi-tab checks.

---

## 15. Open decisions deferred to implementation plan

- Concrete size/depth limits and SQLite vs embed choices for FTS later  
- Exact SM-2-lite interval table  
- Reverse proxy image (Caddy/Traefik/nginx) in Compose  
- Bite generation prompt/schema  
- Whether vaults appear in v1 UI beyond “optional field on project”  

These do not block approving this design.

---

## 16. Document history

| Date | Change |
|------|--------|
| 2026-08-12 | Initial locked design from brainstorming + multiple Oracle passes; local SoT; session workspace; freeform source; flexible session scope; S3 backup-only; publication machine; auth/concurrency/path contracts |

---

## 17. Approval

**Approved by user on 2026-08-12.**

Next step: invoke **writing-plans** and save the implementation plan to:

`docs/superpowers/plans/2026-08-12-personal-agent-v1.md`

(Do not implement code until that plan exists and is reviewed.)
