# Obsidian Memory + Knowledge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** `amp -m grok45 --no-archive-after-execute -x '…'` — **never** Task/OpenAI/oracle/`-ox`. Isolate with git worktrees when using local `-x`. Every implementer task must pass **consulting-grok-review** before merge.

**Goal:** Ship files-first compounding memory (human-gated), project/global instruction files with strict load rules, Obsidian path-wikilinks + backlinks, and project-scoped FTS search — without vault/global search or graph canvas.

**Architecture:** Markdown on disk is SoT under global/vault/project roots. SQLite indexes knowledge notes (source, memory, instructions), wikilink edges, and FTS5. Compound writes only via proposal → human approve → atomic publish. Agents load SYSTEM→SOUL→AGENTS→lessons per session home; compounding skill from `{scope}/.agents/skills/compounding/SKILL.md` only on explicit Compound.

**Tech Stack:** Go 1.24+, SQLite (WAL, FTS5), existing `publish`/`fsroot`/`layout` patterns, Svelte 5 + TypeScript + Vitest, Node `>=22 <23`.

**Spec:** `docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md`  
**Lock:** `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge-lock.md`

## Global Constraints

- Spec wins on product conflicts; Canonical contracts win over draft snippets.
- API is sole writer; no external folder watcher in slice 1.
- Session load: **no** project←global fallback for SOUL/SYSTEM/AGENTS/memory.
- Vault: memory + compounding skill only — **no** vault SOUL/SYSTEM/AGENTS.
- Compound: **explicit user action only**; kinds `agents_patch` | `memory_detail` | `lessons_index_row`; never write `source/**` or `.agents/**` via compound.
- AGENTS must keep/seed Memory → `[[memory/lessons|lessons.md]]` pointer; compound must not strip it.
- Wikilinks: **path resolution** + optional `|title` display; no title-only resolve.
- Search slice 1: **current project corpus only**.
- Preserve promote/direct/review, session focus composer mount, workspace tool sandbox.
- Web: tokens in `app.css`; Node 22 on PATH; rebuild `web/dist` before Go static claims.
- Darwin remains first-class for FS tests.
- Do **not** merge/push product unless user asks; docs may commit locally always.

## Canonical contracts

### Paths (layout helpers — exact names plans must use)

```go
// internal/layout — add alongside existing ProjectRoot / SourceDir / EnsureProjectDirs

func GlobalRoot(dataDir string) string
// dataDir/files/global

func VaultRoot(dataDir, vaultID string) string
// dataDir/files/vaults/{vaultID}

func ProjectRoot(dataDir, vaultID, projectID string) string // existing

func InstructionPath(scopeRoot, name string) string
// name in {"SOUL.md","SYSTEM.md","AGENTS.md"} → scopeRoot/name

func MemoryDir(scopeRoot string) string        // scopeRoot/memory
func LessonsPath(scopeRoot string) string      // scopeRoot/memory/lessons.md
func AgentsSkillsDir(scopeRoot string) string  // scopeRoot/.agents/skills
func CompoundingSkillPath(scopeRoot string) string
// scopeRoot/.agents/skills/compounding/SKILL.md

func EnsureGlobalKnowledgeDirs(dataDir string) error
func EnsureVaultKnowledgeDirs(dataDir, vaultID string) error
// memory/ + .agents/skills/compounding/ + seed files

// EnsureProjectDirs: keep source/memory/soul dirs; ALSO seed
// SOUL.md, SYSTEM.md, AGENTS.md (if missing), memory/lessons.md,
// .agents/skills/compounding/SKILL.md from embedded default.
```

Scope root selection:

| Session home | Instruction+AGENTS root | Memory root | Skill root |
|--------------|-------------------------|-------------|------------|
| project | `ProjectRoot(...)` | same | same |
| vault | `GlobalRoot` | `VaultRoot` | `VaultRoot` |
| global | `GlobalRoot` | `GlobalRoot` | `GlobalRoot` |

### Default AGENTS Memory block (exact markdown to seed/preserve)

```markdown
## Memory
- Lesson index: [[memory/lessons|lessons.md]] — scan titles when stuck or before reinventing a fix.
- Detail files live under `memory/YYYYMMDD-HHmm-*.md`; open only what the index suggests.
- Prefer codifying durable rules here; keep evidence in memory (compound ≠ diary).
```

### Knowledge note identity (LOCKED — consulting-grok-review T-01a02a38)

```go
// domain
type KnowledgeKind string
const (
  KnowledgeKindSource       KnowledgeKind = "source"
  KnowledgeKindMemoryDetail KnowledgeKind = "memory_detail"
  KnowledgeKindMemoryIndex  KnowledgeKind = "memory_index"
  KnowledgeKindAgents       KnowledgeKind = "agents"
  KnowledgeKindSoul         KnowledgeKind = "soul"
  KnowledgeKindSystem       KnowledgeKind = "system"
)
```

**Two path namespaces (do not conflate):**

| Store | `relative_path` meaning | Example |
|-------|-------------------------|---------|
| v1 `notes` | **Source-relative only** (under `source/`). **Never** change this contract. | `articles/intro.md` |
| `knowledge_notes` | **Scope-root-relative** (project/vault/global root) | `source/articles/intro.md`, `memory/lessons.md`, `AGENTS.md` |

**Source mirror rule (mandatory):** on source publish / backfill:

```text
knowledge_notes.relative_path = "source/" + notes.relative_path   // POSIX join; reject if notes.rel has ".."
```

- Keep v1 `notes` table **as today** for promote/direct/review FKs. Do **not** prefix `notes.relative_path` with `source/`.
- New table `knowledge_notes` for all indexed markdown (source mirror + memory + instructions).
- `knowledge_notes.id` = **independent** UUIDs. **Never** assume `notes.id == knowledge_notes.id`.
- Optional column: `source_note_id TEXT NULL REFERENCES notes(id)` set on source mirror upsert (nullable for memory/instructions).
- Upsert key for source rows: `(project_id, relative_path)` where path is scope-root form `source/…`.

**Scope exclusivity + SQLite-safe uniques:**

```sql
-- CHECK exactly one scope owner:
--   (is_global=1 AND project_id IS NULL AND vault_id IS NULL)
--   OR (is_global=0 AND project_id IS NOT NULL AND vault_id IS NULL)  -- project (vault_id on project row is separate; knowledge uses project_id only for project corpus)
--   OR (is_global=0 AND project_id IS NULL AND vault_id IS NOT NULL)  -- vault memory/instructions-N/A
CREATE UNIQUE INDEX knowledge_notes_project_path ON knowledge_notes(project_id, relative_path)
  WHERE project_id IS NOT NULL;
CREATE UNIQUE INDEX knowledge_notes_vault_path ON knowledge_notes(vault_id, relative_path)
  WHERE vault_id IS NOT NULL AND is_global=0;
CREATE UNIQUE INDEX knowledge_notes_global_path ON knowledge_notes(relative_path)
  WHERE is_global=1;
```

**Path validators (do not reuse promote validator for knowledge):**

```go
// internal/paths or internal/knowledge
// Promote/direct keep existing ValidateRelPath (rejects memory/soul components under source/).
func ValidateKnowledgeRelPath(rel string) error
// Allow only:
//   AGENTS.md | SOUL.md | SYSTEM.md
//   memory/**  (no .., no absolute, .md files / lessons.md)
//   source/**  (for index/read of library only — compound MUST NOT write these)
// Reject: empty, .., absolute, .agents/**, sessions/**, soul/** directory targets
```

**Knowledge FS access (LOCKED — consulting-grok-review T-01a02a41):**

Live `fsroot.Open` + `paths.ValidateRelPath` **rejects** any path component `memory` or `soul` (`internal/paths/paths.go`, `internal/fsroot/root.go`). That is **correct for promote/source** and must **not** be loosened.

Knowledge/compound I/O must **not** call `ValidateRelPath` on scope-root paths like `memory/lessons.md`.

**Required approach (pick implementation detail, both OK):**

1. **Sub-root open (preferred):**  
   - Library: `fsroot.Open(SourceDir(projectRoot))` + **source-relative** path (same as v1 notes).  
   - Memory: `fsroot.Open(MemoryDir(scopeRoot))` + path relative to `memory/` only (e.g. `lessons.md`, `YYYYMMDD-HHmm-slug.md`).  
   - Instructions: write/read single files under `scopeRoot` via rooted open of `scopeRoot` with **single-segment** names `AGENTS.md`|`SOUL.md`|`SYSTEM.md` only (still never pass `memory/...` through ValidateRelPath as multi-component under project root if that hits reserved check — use direct `InstructionPath` + atomic write helper that validates with `ValidateKnowledgeRelPath` / name allowlist, not promote validator).

2. **Or** extend fsroot with an explicit `OpenAllowing(validateFn)` / `knowledge.OpenScopeFile(scopeRoot, rel)` that uses **only** `ValidateKnowledgeRelPath`.

**Explicit forbid:** removing `memory`/`soul` from promote `ValidateRelPath` to “make compound work.”
### compound_proposals

```sql
-- indicative; draft A/B must align to this shape
CREATE TABLE compound_proposals (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES sessions(id),
  scope TEXT NOT NULL CHECK(scope IN ('project','vault','global')),
  project_id TEXT REFERENCES projects(id),
  vault_id TEXT REFERENCES vaults(id),
  status TEXT NOT NULL CHECK(status IN ('pending','approved','rejected','failed')),
  request_key TEXT NOT NULL,
  items_json TEXT NOT NULL CHECK(json_valid(items_json)),
  error TEXT,
  created_at TEXT NOT NULL,
  decided_at TEXT,
  finished_at TEXT,
  UNIQUE(session_id, request_key)
);
```

Item JSON element:

```json
{
  "kind": "agents_patch|memory_detail|lessons_index_row",
  "path": "AGENTS.md|memory/....md|memory/lessons.md",
  "action": "create|update",
  "title": "optional",
  "content": "full markdown body after edit",
  "content_sha256": "hex"
}
```

Statuses: `pending` → (`approved`|`rejected`) → if approved publish → set `finished_at` on success or `failed`+`finished_at` on publish error.  
Reject sets `decided_at`+`finished_at` (no publish). Terminal success keeps `status=approved` with `finished_at` set (no separate `completed` status).

**Compound validation (LOCKED):**

```go
func ValidateCompoundItems(scope CompoundScope, items []CompoundItem) error
// Enforce on CreatePending AND on Decide(approve) with the FINAL items
// (including human edits) AND again inside PublishApproved before any write.
// Rules:
// - kind ∈ {agents_patch, memory_detail, lessons_index_row}
// - vault scope: agents_patch forbidden
// - path allowlist ONLY: AGENTS.md (project|global) OR memory/**
// - NEVER source/**, NEVER .agents/**, NEVER SOUL.md/SYSTEM.md via compound
// - ValidateKnowledgeRelPath on every path
// - memory_detail path regex: ^memory/[0-9]{8}-[0-9]{4}-[a-z0-9-]+\.md$
// - if any memory_detail → require lessons_index_row with path memory/lessons.md
// - content_sha256 == sha256(content); size/item caps
// Decide when already terminal → return existing (idempotent); do not re-publish
```

**Compound scope binding (LOCKED):**

```go
// HTTP body may only carry: request_key, user_context?, items?
// Handler MUST derive scope + project_id + vault_id ONLY from sessions row:
//   home=project → scope=project, project_id=session.project_id, vault_id=session.vault_id
//   home=vault   → scope=vault, vault_id=session.vault_id, project_id=""
//   home=global  → scope=global, both ids empty
// Ignore/forbid any client-supplied scope or target ids (do not put them on the wire).
// 403 if session missing/terminal/wrong owner.
```

**Decide CAS + publish atomicity (LOCKED):**

```sql
-- Decide transition must be compare-and-swap:
UPDATE compound_proposals SET status=?, decided_at=?, items_json=?
 WHERE id=? AND status='pending'
-- 0 rows → return current row (idempotent) or 409 if conflicting decision; never double-publish
```

```go
// PublishApproved:
// 1. Re-ValidateCompoundItems on final items
// 2. Stage all file bytes (temp on same volume)
// 3. Apply all disk renames + knowledge upserts in one barrier/txn plan
// 4. On any item failure: MarkFinished failed, finished_at set; do NOT leave status=approved
//    with only a subset applied (compensate staged files / document rollback of partial renames)
// Prefer all-or-nothing; partial success is a bug.
//
// Recovery: on app boot or GET proposal, if status=approved AND finished_at IS NULL,
// re-drive PublishApproved once (idempotent file writes) or MarkFinished failed.
// Never leave approved+null-finished without a recovery path.
```

**Compound generation run (LOCKED — Task 25):**

```go
// When items omitted on POST compound:
// - Use same session Admit as chat (one-active-run): if busy → 409 session_busy
// - Tools DISABLED for compound generation runs (model must emit JSON items only)
// - Skill + compound instructions injected as ephemeral system (PA_COMPOUND_V1 prefix);
//   strip from provider history like PA_RUNTIME_V1 — do not pollute durable chat semantics
// - Parse JSON items → CreatePending; do not write AGENTS/memory until human decide
// Items-POST path: no agent run.
// UI: disable Compound button while session has queued|running run.
```

**lessons_index_row semantics (LOCKED):**

```go
// Prefer server-side MERGE for lessons_index_row:
// - Parse existing memory/lessons.md bullets
// - Upsert/prepend the proposed row(s) by target path; do not delete unrelated rows
// - If client sends full content, server still merges by path key rather than blind overwrite
// OR if full-replace kept: ValidateCompoundItems MUST require every memory_detail path
// appear as a wikilink/target in the lessons content, AND tests prove multi-row index
// survives a second compound that only adds one lesson (implementer must read-merge).
// LOCKED choice: **server merge/prepend by detail path**; never blind truncate index.
```

**content_sha256 on decide (LOCKED):**

```go
// On Decide(approve), server recomputes sha256 from final item content and overwrites
// content_sha256 (client hash advisory only). Human edits in UI need not recompute.
```

**HTTP CSRF (LOCKED):**

```go
// All compound POST/decide and instruction PUT routes use the same mutation middleware
// + RequireCSRF as POST /api/v1/sessions/{id}/messages (see server.go).
```

### Wikilink regex / normalize

- Match: `\[\[([^\]|#]+)(?:\|([^\]]+))?\]\]`
- Trim target; reject targets with `..`, absolute, empty.
- **Join key LOCKED:** `note_links.to_path` and resolve lookups use the **same** scope-root form as `knowledge_notes.relative_path`, **including `.md`** (e.g. `source/intro.md`, `memory/a.md`, `AGENTS.md`).
- When parsing a link, if the target has no `.md` suffix and is not a bare instruction stem that maps to `AGENTS.md`|`SOUL.md`|`SYSTEM.md`, append `.md` before store/resolve.
- Bare `AGENTS` / `SOUL` / `SYSTEM` → `AGENTS.md` / `SOUL.md` / `SYSTEM.md`.
- Display alias (`|title`) never affects the join key.
- Resolution root = knowledge note’s scope root.
- Source library links in markdown MUST use `source/…` prefix (not v1 notes-relative bare paths).
### note_links

```sql
CREATE TABLE note_links (
  id TEXT PRIMARY KEY,
  from_note_id TEXT NOT NULL REFERENCES knowledge_notes(id) ON DELETE CASCADE,
  raw_target TEXT NOT NULL,
  to_path TEXT NOT NULL,
  to_note_id TEXT REFERENCES knowledge_notes(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX note_links_to_note ON note_links(to_note_id);
CREATE INDEX note_links_to_path ON note_links(to_path);
CREATE INDEX note_links_from ON note_links(from_note_id);
```

`from_note_id` / `to_note_id` are always **`knowledge_notes.id`**, never v1 `notes.id`.

### Backlinks / ID resolve (LOCKED)

```
# Preferred (unambiguous):
GET /api/v1/projects/{id}/knowledge/backlinks?path={scope-root-rel}
GET /api/v1/projects/{id}/knowledge/backlinks?knowledge_id={knowledge_notes.id}

# Optional convenience for source library UI that only has v1 notes.id:
GET /api/v1/projects/{id}/notes/{note_id}/backlinks
  → server resolves knowledge row by
     project_id = id AND relative_path = 'source/' || notes.relative_path
  → 404 if no knowledge mirror row
```

Search hits and knowledge tools return **`knowledge_id`** + `path` (scope-root) + `kind`.  
JSON field name **`knowledge_id`** (not ambiguous `note_id`).

**UI open contract (LOCKED):**

```
onOpen(hit):
  if hit.kind == source && hit.source_note_id:
    navigate v1 notes UI with source_note_id  // existing #/projects/.../notes/{notes.id}
  else:
    open via GET /api/v1/projects/{id}/knowledge/read?path={hit.path}
// Never pass knowledge_id to GET /api/v1/notes/{id} body routes.
BacklinksPanel onopen uses knowledge_id or path → knowledge/read; source convenience optional.
```
### FTS

```sql
CREATE VIRTUAL TABLE knowledge_fts USING fts5(
  note_id UNINDEXED,
  title,
  path,
  body,
  tokenize = 'unicode61'
);
```

`note_id` column in FTS table = `knowledge_notes.id` (internal). API responses use **`knowledge_id`**.  
Project search SQL filters `knowledge_notes` to `project_id = ?` and kinds in corpus; join fts. Never search other projects. Always join `knowledge_notes` for scope (do not trust FTS alone).

**FTS query safety (LOCKED):** bind/escape user query for FTS5 (strip or quote `"` `*` Boolean ops as needed); invalid query → empty hits or 400, never panic/500 SQL error leak.

**Frontmatter caps (LOCKED):** max frontmatter block 64KiB; fail closed on parse bomb / oversize (skip fm fields, still index body, or reject write — prefer reject compound/instruction put; for promote source prefer skip fm + log).
### Migrations (LOCKED)

```go
// internal/db/db.go today hard-codes version '001' only.
// Task 5 MUST change Open/migrate to apply embedded migrations/002_knowledge.sql
// (loop migrations/*.sql by sorted version OR explicit 002 after 001).
// Test: after db.Open on empty dir, knowledge_notes / compound_proposals / note_links exist.
// Never ship 002 SQL file without migrator change.
```

### Prompt assembly API

```go
// internal/agent/prompt.go (new)
type PromptSection struct {
  Name    string // "runtime"|"system"|"soul"|"agents"|"lessons"
  Path    string // empty for runtime
  Content string
  Truncated bool
}

type BuildPromptInput struct {
  DataDir string
  Home    layout.SessionHome
  VaultID string // may be empty
  ProjectID string // may be empty
  // caps
  MaxPerFileBytes int // default 32_768
  MaxTotalBytes   int // default 96_000
}

func BuildSessionPrompt(in BuildPromptInput) ([]PromptSection, error)
// Reads files from disk; skips missing/empty; applies caps AGENTS>SYSTEM>SOUL>lessons
```

### HTTP routes (mount under existing auth — **`/api/v1` prefix LOCKED**)

Match live `internal/httpapi/server.go` (`/api/v1/...`). Never mount bare `/api/...` for these.

```
GET  /api/v1/projects/{id}/instructions/{name}     name=soul|system|agents
PUT  /api/v1/projects/{id}/instructions/{name}
GET  /api/v1/global/instructions/{name}
PUT  /api/v1/global/instructions/{name}

POST /api/v1/sessions/{id}/compound                body: {request_key, user_context?, items?}
GET  /api/v1/sessions/{id}/compound/{proposal_id}
POST /api/v1/sessions/{id}/compound/{proposal_id}/decide
     body: {request_key, decision: approve|reject, items?: edited items}

GET  /api/v1/projects/{id}/knowledge/backlinks?path=|knowledge_id=
GET  /api/v1/projects/{id}/notes/{note_id}/backlinks   # convenience resolve — see Backlinks section
GET  /api/v1/projects/{id}/search?q=&limit=

GET  /api/v1/projects/{id}/knowledge/tree
GET  /api/v1/projects/{id}/knowledge/read?path=       # path = scope-root-relative
```

Errors: 400 validation, 404 missing, 409 idempotency conflict, 403 wrong session scope.

### Agent tools (project session) — LOCKED T-01a02a53

```go
// names locked
const (
  ToolReadKnowledge  = "read_knowledge"   // {path} scope-root-relative
  ToolListKnowledge  = "list_knowledge"   // {path?}
  ToolSearchProject  = "search_project"   // {query, limit?}
)
// No write_knowledge tool in slice 1.
```

**Runner dispatch (Critical lock):**

```go
// Live runner today only workspace.Execute — MUST change in Task 65:
// 1. Build tool definitions for project home ALWAYS including the three knowledge tools
//    independent of session.tool_grants.workspace_files.
// 2. On tool call:
//    - workspace tool names → existing Workspace path (requires workspace_files grant + root)
//    - knowledge tool names → KnowledgeToolHandler (no workspace grant required)
//    - unknown → reject
// 3. Tests MUST pass with workspace_files=false: search_project + read_knowledge succeed.
// 4. Vault/global sessions: do NOT register knowledge tools in slice 1.
// 5. Forbid routing knowledge paths through Workspace/ValidateRelPath.
```

**Tool result caps:** knowledge read max body (e.g. 256KiB same order as workspace markdown); search max hits 50; snippet max ~400 runes; truncate with clear marker.

### UI components (tokens in app.css)

| Component | Role |
|-----------|------|
| `CompoundReviewCard.svelte` | pending proposal items; approve/edit/reject/confirm |
| `BacklinksPanel.svelte` | list inbound links |
| `KnowledgeSearch.svelte` | project search field + results |
| `InstructionEditor.svelte` | minimal SOUL/SYSTEM/AGENTS editor |
| `ProjectRail.svelte` | remove fake textarea; show memory summary / open lessons |

### Timestamps

RFC3339Nano UTC strings in DB (match existing store helpers).  
Metric: time-to-finish = parse(finished_at)-parse(created_at).

### Embedded skill

- Source of default body: `internal/agent/skills/compounding/SKILL.md` (go:embed)
- Copied to scope path on ensure/create if missing
- Load order on Compound: scope file if non-empty else embedded

### TDD / verify commands

```bash
go test ./internal/layout/ ./internal/store/ ./internal/agent/ ./internal/httpapi/ ./internal/publish/ -count=1
export PATH="$HOME/.local/node-v22/bin:/usr/bin:$PATH"   # adjust to Node 22
npm --prefix web test
```

---

## File map (summary)

See lock. Drafts must not invent parallel doc trees.

---

<!-- DRAFTS ASSEMBLED BELOW THIS LINE -->
