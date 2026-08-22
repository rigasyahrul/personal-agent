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

Search hits and knowledge tools return **`knowledge_id`** + `path` (scope-root). UI must not pass `knowledge_id` to v1 note body endpoints without resolve.

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

`note_id` in FTS = `knowledge_notes.id`.  
Project search SQL filters `knowledge_notes` to `project_id = ?` and kinds in corpus; join fts. Never search other projects. Always join `knowledge_notes` for scope (do not trust FTS alone).

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

### Agent tools (project session)

```go
// names locked
const (
  ToolReadKnowledge  = "read_knowledge"   // {path}
  ToolListKnowledge  = "list_knowledge"   // {path?}
  ToolSearchProject  = "search_project"   // {query, limit?}
)
// Workspace tools unchanged. No write_knowledge tool in slice 1.
```

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

---

## Phase drafts (assembled)

Source drafts: `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge-drafts/`. **Canonical contracts above win on conflict.**

### Review ledger

| Gate | Thread | Result |
|------|--------|--------|
| Design+plan lock | `T-01a02a38-9315-7176-b621-d3c7d122fea7` | NOT safe — Important path/ID/compound/API/uniques |
| Post-fix re-review | `T-01a02a3d-5bee-71d1-8d4c-7af5e68c6903` | **Safe to start P0** (Minor only; wikilink .md join key locked in follow-up) |

---

# Draft A — P0 Layout + Prompt (Tasks 1–12)

> Assembled into `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md`.  
> Canonical contracts in plan header win on conflict.

---

### Task 1: Layout path helpers

**Files:**
- Modify: `internal/layout/layout.go`
- Modify: `internal/layout/layout_test.go`

**Interfaces:**
- Produces:
  - `func GlobalRoot(dataDir string) string`
  - `func VaultRoot(dataDir, vaultID string) string`
  - `func InstructionPath(scopeRoot, name string) string` // name must be SOUL.md|SYSTEM.md|AGENTS.md
  - `func MemoryDir(scopeRoot string) string`
  - `func LessonsPath(scopeRoot string) string`
  - `func AgentsSkillsDir(scopeRoot string) string`
  - `func CompoundingSkillPath(scopeRoot string) string`

- [ ] **Step 1: Write the failing test**

```go
func TestKnowledgePaths(t *testing.T) {
	g := GlobalRoot("/data")
	if g != filepath.Join("/data", "files", "global") {
		t.Fatalf("global: %s", g)
	}
	v := VaultRoot("/data", "v1")
	if v != filepath.Join("/data", "files", "vaults", "v1") {
		t.Fatalf("vault: %s", v)
	}
	p := ProjectRoot("/data", "v1", "p1")
	if InstructionPath(p, "AGENTS.md") != filepath.Join(p, "AGENTS.md") {
		t.Fatal("agents path")
	}
	if LessonsPath(p) != filepath.Join(p, "memory", "lessons.md") {
		t.Fatal("lessons")
	}
	if CompoundingSkillPath(g) != filepath.Join(g, ".agents", "skills", "compounding", "SKILL.md") {
		t.Fatal("skill")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/layout/ -run TestKnowledgePaths -count=1`  
Expected: FAIL undefined

- [ ] **Step 3: Minimal implementation**

Add helpers in `layout.go` using `filepath.Join`. `InstructionPath` returns `filepath.Join(scopeRoot, name)` without validation (callers validate name).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/layout/ -run TestKnowledgePaths -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/layout/layout.go internal/layout/layout_test.go
git commit -m "feat(layout): knowledge path helpers for instructions memory skills"
```

---

### Task 2: Embedded default compounding skill

**Files:**
- Create: `internal/agent/skills/compounding/SKILL.md`
- Create: `internal/agent/skills/embed.go`
- Create: `internal/agent/skills/embed_test.go`

**Interfaces:**
- Produces:
  - `package skills` with `//go:embed compounding/SKILL.md` → `var DefaultCompoundingSkill string` or `[]byte`
  - `func DefaultCompoundingSkillMarkdown() string`

Skill body must include (spec §14): codify-first; selective detail; thin lessons index; preserve Memory pointer; path wikilinks; proposal JSON kinds only; compound ≠ diary; vault = memory only.

- [ ] **Step 1: Write the failing test**

```go
func TestDefaultCompoundingSkillEmbedded(t *testing.T) {
	s := DefaultCompoundingSkillMarkdown()
	for _, need := range []string{"codify", "lessons.md", "agents_patch", "memory_detail", "diary"} {
		if !strings.Contains(strings.ToLower(s), strings.ToLower(need)) {
			t.Fatalf("missing %q in skill", need)
		}
	}
	if len(s) < 400 {
		t.Fatalf("skill too short: %d", len(s))
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test ./internal/agent/skills/ -count=1`

- [ ] **Step 3: Implement embed + SKILL.md**

Write a complete default skill (~1–2 pages) matching Superpowers compounding pattern adapted to proposal schema in header.

- [ ] **Step 4: PASS + commit**

```bash
git add internal/agent/skills/
git commit -m "feat(agent): embed default compounding skill"
```

---

### Task 3: Ensure seed helpers (global / vault / project)

**Files:**
- Modify: `internal/layout/layout.go`
- Modify: `internal/layout/layout_test.go`
- Modify: `EnsureProjectDirs` to also seed instruction files + lessons + skill when missing

**Interfaces:**
- Produces:
  - `func EnsureGlobalKnowledgeDirs(dataDir string, skillMarkdown string) error`
  - `func EnsureVaultKnowledgeDirs(dataDir, vaultID string, skillMarkdown string) error`
  - Extend `EnsureProjectDirs` → prefer new signature  
    `EnsureProjectDirs(dataDir, vaultID, projectID string, skillMarkdown string) error`  
    **or** keep old and add `EnsureProjectKnowledge(dataDir, vaultID, projectID, skillMarkdown string) error` called after dirs.  
    **LOCKED for implementers:** add `EnsureProjectKnowledge(...)` called from `ProjectStore.create` after `EnsureProjectDirs`, so existing tests that only call `EnsureProjectDirs` keep working; update project create path.

Seed rules:
- Create dirs `memory/`, `.agents/skills/compounding/` with 0700
- Write skill file if missing (0600) from `skillMarkdown`
- Write `memory/lessons.md` if missing with scaffold:

```markdown
# Lessons

> Thin index only. Detail files: `memory/YYYYMMDD-HHmm-slug.md`.

```

- Write `SOUL.md`, `SYSTEM.md`, `AGENTS.md` if missing:
  - SOUL/SYSTEM: empty file or single newline
  - AGENTS: default Memory block from Canonical contracts (exact markdown)

- [ ] **Step 1: Failing tests** for global, vault, project seed idempotency (second call does not overwrite edited AGENTS).

```go
func TestEnsureProjectKnowledgeIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureProjectDirs(dir, "", "p1"); err != nil { t.Fatal(err) }
	if err := EnsureProjectKnowledge(dir, "", "p1", "# skill\n"); err != nil { t.Fatal(err) }
	agents := InstructionPath(ProjectRoot(dir, "", "p1"), "AGENTS.md")
	if err := os.WriteFile(agents, []byte("custom\n"), 0600); err != nil { t.Fatal(err) }
	if err := EnsureProjectKnowledge(dir, "", "p1", "# skill\n"); err != nil { t.Fatal(err) }
	b, _ := os.ReadFile(agents)
	if string(b) != "custom\n" { t.Fatalf("overwrote agents: %q", b) }
}
```

- [ ] **Step 2–4:** implement, pass, commit

```bash
git commit -m "feat(layout): seed instructions memory and compounding skill"
```

---

### Task 4: Wire seed on vault + project create + app boot

**Files:**
- Modify: `internal/store/projects.go` — after `EnsureProjectDirs`, call `EnsureProjectKnowledge(..., skills.DefaultCompoundingSkillMarkdown())`
- Modify: `internal/store/vaults.go` — after vault row insert, `layout.EnsureVaultKnowledgeDirs` + ensure vault root exists
- Modify: app open/boot path (find where data dir is initialized — `cmd/personal-agent` or `internal/app`) to `EnsureGlobalKnowledgeDirs`
- Tests: `internal/store/projects_test.go`, `vaults` tests, boot test if present

- [ ] **Step 1:** Test `ProjectStore.Create` leaves `AGENTS.md` and skill file on disk.

```go
// in projects_test.go after create:
root := layout.ProjectRoot(dataDir, "", p.ID)
if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil { t.Fatal(err) }
if _, err := os.Stat(layout.CompoundingSkillPath(root)); err != nil { t.Fatal(err) }
```

- [ ] **Step 2–4:** implement, pass, commit

```bash
git commit -m "feat: seed knowledge files on project vault and global ensure"
```

---

### Task 5: Migration `002_knowledge.sql`

**Files:**
- Create: `internal/db/migrations/002_knowledge.sql`
- Ensure migrator picks numeric order (existing pattern)

**DDL (must match header contracts after consulting-grok-review):**
- `knowledge_notes` (id, kind, project_id, vault_id, is_global, relative_path, title, content_sha256, byte_size, frontmatter_json, status, source_note_id NULL REFERENCES notes(id), created_at, updated_at)
- Scope CHECK + **partial unique indexes** (project/vault/global) — not a single UNIQUE that breaks on NULL
- `relative_path` is **scope-root-relative** (`source/…`, `memory/…`, `AGENTS.md`)
- `compound_proposals`, `note_links`, `knowledge_fts` as header
- Include `ValidateKnowledgeRelPath` unit tests in Task 7, not promote `ValidateRelPath`
- [ ] **Step 1:** Test migration applies on empty DB (use existing db test helper).

- [ ] **Step 2–4:** add SQL, pass migrate test, commit

```bash
git commit -m "db: knowledge_notes compound_proposals note_links fts migration"
```

---

### Task 6: Domain types for knowledge + compound

**Files:**
- Create or modify: `internal/domain/knowledge.go`
- Modify: `internal/domain/models.go` if needed

**Interfaces:**
```go
type KnowledgeKind string
// constants from header

type KnowledgeNote struct {
  ID, RelativePath, Title string
  Kind KnowledgeKind
  ProjectID, VaultID string // empty if unused
  IsGlobal bool
  SourceNoteID string // optional; set when kind=source mirror of v1 notes.id
  ContentSHA256 string
  ByteSize int64
  FrontmatterJSON string
  Status string
  CreatedAt, UpdatedAt time.Time
}

type CompoundScope string // project|vault|global
type CompoundStatus string
type CompoundProposal struct {
  ID, SessionID string
  Scope CompoundScope
  ProjectID, VaultID string
  Status CompoundStatus
  RequestKey string
  ItemsJSON string
  Error string
  CreatedAt time.Time
  DecidedAt, FinishedAt *time.Time
}
```

- [ ] Test JSON tags round-trip optional; compile test enough. Commit: `feat(domain): knowledge and compound types`

---

### Task 7: Instruction store read/write + knowledge upsert

**Files:**
- Create: `internal/store/instructions.go`
- Create: `internal/store/instructions_test.go`

**Interfaces:**
```go
type InstructionName string // soul|system|agents → file SOUL.md etc.

func NormalizeInstructionFile(name string) (fileName string, kind domain.KnowledgeKind, err error)

// InstructionStore
func (s *InstructionStore) Get(ctx, scopeRoot string, name InstructionName) (content string, note domain.KnowledgeNote, err error)
func (s *InstructionStore) Put(ctx context.Context, meta ScopeMeta, name InstructionName, content string) (domain.KnowledgeNote, error)
// Put: validate, atomic write file, upsert knowledge_notes, reparse links deferred to P2 stub OK
```

`ScopeMeta`: `{DataDir, Home-equivalent: project|vault|global, ProjectID, VaultID}`

- [ ] Tests: put AGENTS, get back; reject `../x`; empty content allowed.

- [ ] Commit: `feat(store): instruction get/put with knowledge_notes upsert`

---

### Task 8: HTTP instruction handlers

**Files:**
- Create: `internal/httpapi/instruction_handlers.go`
- Create: `internal/httpapi/instruction_handlers_test.go`
- Modify: `internal/httpapi/server.go` routes

Routes per header:
- `GET/PUT /api/v1/projects/{id}/instructions/{name}`
- `GET/PUT /api/v1/global/instructions/{name}`

- [ ] Tests via existing httptest server helper: PUT agents, GET matches; 400 bad name.

- [ ] Commit: `feat(api): project and global instruction endpoints`

---

### Task 9: `BuildSessionPrompt`

**Files:**
- Create: `internal/agent/prompt.go`
- Create: `internal/agent/prompt_test.go`

**Interfaces:** exact `BuildPromptInput`, `PromptSection`, `BuildSessionPrompt` from header.

Runtime section content must mention: tools/safety, session home, compound only on explicit user action, path roots.

Caps: default MaxPerFileBytes=32768, MaxTotalBytes=96000; priority AGENTS>SYSTEM>SOUL>lessons when truncating.

- [ ] **Tests:**
  1. Project with only AGENTS → sections runtime+agents; no global file content even if global AGENTS exists.
  2. Vault session → global SYSTEM/SOUL/AGENTS + vault lessons only.
  3. Empty files skipped.
  4. Truncation sets `Truncated=true`.

- [ ] Commit: `feat(agent): BuildSessionPrompt with scoped load rules`

---

### Task 10: Wire prompt into Runner

**Files:**
- Modify: `internal/agent/runner.go`
- Modify: `internal/agent/runner_test.go`

Prepend one system `ChatMessage` (or multiple) from `BuildSessionPrompt` **before** history user/assistant messages. Do not duplicate if history already contains identical runtime blob (simple approach: always rebuild fresh system prefix each run; strip prior injected system messages marked with a stable prefix marker like `PA_RUNTIME_V1` if re-feeding full history — **LOCKED:** inject a single leading system message concatenating sections each run; when mapping history to provider, **skip** historical `role=system` messages that start with `PA_RUNTIME_V1` to avoid stacking).

- [ ] Test: execute with stub provider captures messages[0].role==system and contains AGENTS body.

- [ ] Commit: `feat(agent): inject scoped prompt sections into runner`

---

### Task 11: Lessons index read helper

**Files:**
- Create: `internal/store/memory_read.go` (or fold into instructions)
- Test: read lessons for prompt/path

```go
func ReadLessonsIndex(scopeRoot string) (string, error) // missing → "", nil
```

- [ ] Commit: `feat(store): read memory lessons index`

---

### Task 12: P0 verification gate

**Files:** none new

- [ ] **Step 1: Run full relevant tests**

```bash
go test ./internal/layout/ ./internal/store/ ./internal/agent/ ./internal/agent/skills/ ./internal/httpapi/ -count=1
```

Expected: PASS

- [ ] **Step 2: Manual checklist in commit message**
  - New project has AGENTS Memory block + skill file
  - Prompt isolation test green

- [ ] **Step 3: Commit** docs note only if needed — or empty verification commit:

```bash
git commit --allow-empty -m "test: P0 layout prompt verification gate"
```

DRAFT_A_COMPLETE

---

# Draft B — P1 Compound (Tasks 20–35)

> Assembled into final plan. Header Canonical contracts win.

---

### Task 20: Compound proposal store — create pending

**Files:**
- Create: `internal/store/compound.go`
- Create: `internal/store/compound_test.go`

**Interfaces:**
```go
type CompoundItem struct {
  Kind string `json:"kind"` // agents_patch|memory_detail|lessons_index_row
  Path string `json:"path"`
  Action string `json:"action"` // create|update
  Title string `json:"title,omitempty"`
  Content string `json:"content"`
  ContentSHA256 string `json:"content_sha256"`
}

type CreateProposalInput struct {
  SessionID string
  RequestKey string
  Scope domain.CompoundScope
  ProjectID, VaultID string
  Items []CompoundItem
  Now time.Time
}

func (s *CompoundStore) CreatePending(ctx context.Context, in CreateProposalInput) (domain.CompoundProposal, error)
// Idempotent: same session_id+request_key+same items fingerprint → return existing
// Different fingerprint → ErrConflict
```

Validation via shared `ValidateCompoundItems` (Canonical LOCKED):
- Run on **CreatePending**, on **Decide(approve) final items** (including human edits), and again in **PublishApproved** before any write
- kind/path/action allowed for scope (vault cannot agents_patch)
- path allowlist ONLY `AGENTS.md` or `memory/**` — NEVER `source/**`, `.agents/**`, SOUL/SYSTEM
- memory_detail path regex `^memory/[0-9]{8}-[0-9]{4}-[a-z0-9-]+\.md$`
- if any memory_detail → require lessons_index_row path `memory/lessons.md`
- `ValidateKnowledgeRelPath` on every path (not promote `ValidateRelPath`)
- content sha256 must match sha256(content); max 256KiB/item; max 20 items
- Decide when already terminal → idempotent return; do not re-publish
- [ ] **Step 1: Failing tests** — happy path; vault rejects agents_patch; detail without index row rejects; path escape rejects.

- [ ] **Step 2–4:** implement, pass, commit `feat(store): compound proposal create pending`

---

### Task 21: Compound decide + timestamps

**Files:**
- Modify: `internal/store/compound.go`

**Interfaces:**
```go
type DecideInput struct {
  ProposalID string
  RequestKey string // decide idempotency optional separate key — use body request_key UNIQUE per decide op OR reuse: LOCKED use decide_request_key column optional; simpler: idempotent decide by proposal status
  Decision string // approve|reject
  Items []CompoundItem // optional edits replacing items_json when approve
  Now time.Time
}

func (s *CompoundStore) Decide(ctx context.Context, in DecideInput) (domain.CompoundProposal, error)
// reject: status=rejected, decided_at=now, finished_at=now
// approve: status=approved, decided_at=now, finished_at still null until publish completes
func (s *CompoundStore) MarkFinished(ctx context.Context, id string, status string, errMsg string, now time.Time) error
// status approved→ stays approved with finished_at; or failed with error+finished_at
func (s *CompoundStore) Get(ctx, id string) (domain.CompoundProposal, error)
func (s *CompoundStore) GetBySessionRequest(ctx, sessionID, requestKey string) (domain.CompoundProposal, error)
```

- [ ] Tests: reject sets both timestamps; approve sets decided_at only; MarkFinished sets finished_at.

- [ ] Commit: `feat(store): compound decide and finished_at`

---

### Task 22: Compound publish writer

**Files:**
- Create: `internal/compound/publish.go`
- Create: `internal/compound/publish_test.go`

**Interfaces:**
```go
type Publisher struct {
  DataDir string
  DB *sql.DB
  // knowledge upsert + optional link reparse hook
  Notes *store.KnowledgeStore // or InstructionStore + MemoryWriter
  Clock clock.Clock
  Barrier mutBarrier
}

func (p *Publisher) PublishApproved(ctx context.Context, proposal domain.CompoundProposal) error
// For each item: atomic write under scope root; upsert knowledge_notes; update lessons/agents
// Preserve AGENTS Memory block: if agents_patch content strips ## Memory pointer, reject before write (ValidateAgentsMemoryPointer)
// On full success caller MarkFinished approved; on error MarkFinished failed
```

Use `fsroot` / temp+rename same volume patterns from `internal/publish`.

- [ ] Tests: approve agents_patch writes file; strips Memory block → error; memory detail + lessons row both land.

- [ ] Commit: `feat(compound): publish approved proposal items to disk`

---

### Task 23: Load compounding skill for session scope

**Files:**
- Create: `internal/agent/compound_skill.go`
- Test: `internal/agent/compound_skill_test.go`

```go
func LoadCompoundingSkill(dataDir string, home layout.SessionHome, vaultID, projectID string) (string, string, error)
// returns (markdown, sourcePathOr"embedded", err)
// missing/empty file → embedded default
```

- [ ] Commit: `feat(agent): load scoped compounding skill with embed fallback`

---

### Task 24: HTTP POST start compound (proposal from model — phase split)

**Design lock for P1:** Two substeps in product:

**24a UI/API path without LLM (testable):** client may POST items directly for tests/dev.  
**24b Agent path:** Runner mode `compound` loads skill + user_context and expects model to return JSON items (parse from assistant message fenced json).

**Files:**
- Create: `internal/httpapi/compound_handlers.go`
- Tests

```
POST /api/v1/sessions/{id}/compound
{ "request_key": "...", "user_context": "optional", "items": [ ... optional prebuilt ...] }
```

If `items` present → validate+CreatePending (no model).  
If `items` absent → start compound agent run (Task 25) OR return 501 until 25 done — **LOCKED:** implement items-present path first in this task; Task 25 adds generation.

- [ ] Test: POST items → 200 proposal pending.

- [ ] Commit: `feat(api): create compound proposal from items`

---

### Task 25: Compound generation via Runner

**Files:**
- Modify: `internal/agent/runner.go` or new `internal/agent/compound_run.go`
- HTTP: when items omitted, admit a run with system= skill + instructions to emit JSON array of items only; parse; CreatePending; return proposal id.

**Interfaces:**
```go
func ParseCompoundItemsFromAssistant(content string) ([]store.CompoundItem, error)
// extract first ```json ... ``` or raw JSON array
```

- [ ] Test with fake provider returning fixed JSON → proposal rows created.

- [ ] Commit: `feat(agent): generate compound proposal items from model`

---

### Task 26: HTTP GET proposal + decide

**Files:**
- Modify: `compound_handlers.go`

```
GET  /api/v1/sessions/{id}/compound/{proposal_id}
POST /api/v1/sessions/{id}/compound/{proposal_id}/decide
{ "request_key": "...", "decision": "approve"|"reject", "items": [optional edits] }
```

Decide approve → Publisher.PublishApproved → MarkFinished.  
All under auth+CSRF.

- [ ] Tests: reject; approve writes AGENTS on disk; wrong session 404.

- [ ] Commit: `feat(api): get and decide compound proposals`

---

### Task 27: Frontend API client types

**Files:**
- Modify: `web/src/lib/api/types.ts`
- Create: `web/src/lib/api/compound.ts`
- Test: unit parse types if applicable

```ts
export type CompoundItem = { kind: string; path: string; action: string; title?: string; content: string; content_sha256: string }
export type CompoundProposal = { id: string; status: string; items: CompoundItem[]; created_at: string; decided_at?: string; finished_at?: string }
export function createCompound(sessionId: string, body: {...}): Promise<CompoundProposal>
export function decideCompound(sessionId: string, proposalId: string, body: {...}): Promise<CompoundProposal>
```

- [ ] Commit: `feat(web): compound API client`

---

### Task 28: `CompoundReviewCard.svelte`

**Files:**
- Create: `web/src/components/sessions/CompoundReviewCard.svelte`
- Create: `web/src/components/sessions/CompoundReviewCard.test.ts`
- Styles: `web/src/app.css` — `.compound-card`, `.compound-item`, etc. tokens

**Props:**
```ts
proposal: CompoundProposal
onconfirm: (decision: 'approve'|'reject', items: CompoundItem[]) => void
oncancel?: () => void
busy?: boolean
```

UI: list items with path/kind; textarea edit content; approve/reject buttons; confirm.

- [ ] Test: renders paths; edit updates local items; approve callback.

- [ ] Commit: `feat(web): CompoundReviewCard for human gate`

---

### Task 29: Wire Compound control into SessionChat

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `SessionChat.focus.test.ts` — ensure compound UI does not remount composer

- [ ] Add **Compound** button (explicit only); on click POST compound with user_context=last messages summary or empty; show CompoundReviewCard when pending.
- [ ] On approve/reject call decide endpoint; toast/error.
- [ ] Focus regression test still PASS.

- [ ] Commit: `feat(web): explicit Compound action in session chat`

---

### Task 30: Vault/global compound scope enforcement tests

**Files:**
- `internal/store/compound_test.go` + API tests

- [ ] Project session cannot write vault memory paths outside project root.
- [ ] Vault session CreatePending accepts memory_detail under vault; rejects agents_patch.
- [ ] Global session writes global AGENTS/memory only.

- [ ] Commit: `test: compound scope enforcement`

---

### Task 31: AGENTS Memory pointer preservation helper

**Files:**
- Create: `internal/compound/agents_pointer.go`
- Test

```go
const AgentsMemoryMarker = "## Memory"
func EnsureAgentsMemoryPointer(content string) (string, error)
// if missing Memory section, append Canonical block; if present keep
func ValidateAgentsMemoryPointer(content string) error
// require [[memory/lessons
```

- [ ] Used by publish + optional Put instruction.

- [ ] Commit: `feat(compound): ensure AGENTS memory lessons pointer`

---

### Task 32: Replace ProjectRail fake memory textarea (read-only summary)

**Files:**
- Modify: `web/src/components/ProjectRail.svelte`
- Modify: `ProjectRail.test.ts`
- API: GET project knowledge read `memory/lessons.md` or instructions

- [ ] Remove non-persistent bind:value memory dump.
- [ ] Show lessons index preview (first N lines) + link “Open memory”.
- [ ] Empty state when no lessons.

- [ ] Commit: `fix(web): ProjectRail shows real memory index not fake textarea`

---

### Task 33: Compound timestamps metric helper (optional UI)

**Files:**
- Create: `web/src/lib/compoundMetrics.ts`
- Test: time-to-finish ms from created_at/finished_at ISO strings

- [ ] Commit: `feat(web): compound time-to-finish helper`

---

### Task 34: Integration test Go — full compound approve path

**Files:**
- Create: `internal/httpapi/compound_handlers_test.go` end-to-end with temp data dir

Flow: create project+session → POST items → decide approve → read AGENTS from disk contains rule + Memory pointer → proposal finished_at set.

- [ ] Commit: `test(api): compound approve end-to-end`

---

### Task 35: P1 verification gate

```bash
go test ./internal/compound/ ./internal/store/ ./internal/agent/ ./internal/httpapi/ -count=1
export PATH="$HOME/.local/node-v22/bin:/usr/bin:$PATH"
npm --prefix web test -- CompoundReviewCard SessionChat ProjectRail
```

- [ ] Expected PASS  
- [ ] Commit empty or changelog: `test: P1 compound verification gate`

DRAFT_B_COMPLETE

---

# Draft C — P2 Obsidian links + backlinks (Tasks 40–52)

---

### Task 40: Frontmatter parse

**Files:**
- Create: `internal/knowledge/frontmatter.go`
- Create: `internal/knowledge/frontmatter_test.go`

```go
type Frontmatter struct {
  Title string
  Date string
  Tags []string
  CodifiedInto []string
  Raw map[string]any
}

func SplitFrontmatter(md string) (fm Frontmatter, body string, err error)
// --- yaml --- body; missing fm → empty, full body
func TitleOrStem(fm Frontmatter, relativePath string) string
```

- [ ] Tests: with/without fm; title fallback stem `memory/x.md` → `x`.

- [ ] Commit: `feat(knowledge): yaml frontmatter split`

---

### Task 41: Wikilink parse

**Files:**
- Create: `internal/knowledge/wikilink.go`
- Create: `internal/knowledge/wikilink_test.go`

```go
type Wikilink struct {
  RawTarget string
  Alias string
  NormalizedPath string // no leading ./ ; strip .md
}

func ParseWikilinks(body string) []Wikilink
func NormalizeWikilinkTarget(target string) (string, error)
// reject .., absolute, empty, NUL
```

Regex from header: `\[\[([^\]|#]+)(?:\|([^\]]+))?\]\]`

- [ ] Tests: `[[memory/a|Title]]`, `[[source/x.md]]`, reject `[[../x]]`.

- [ ] Commit: `feat(knowledge): path wikilink parse and normalize`

---

### Task 42: KnowledgeStore upsert + reindex links

**Files:**
- Create: `internal/store/knowledge.go`
- Create: `internal/store/knowledge_test.go`

```go
type UpsertKnowledgeInput struct {
  Kind domain.KnowledgeKind
  ProjectID, VaultID string
  IsGlobal bool
  RelativePath string
  Content []byte
  Status string // ready
}

func (s *KnowledgeStore) UpsertFromContent(ctx context.Context, in UpsertKnowledgeInput) (domain.KnowledgeNote, error)
// hash, frontmatter title, write DB row, replace note_links from parse, try resolve to_note_id
func (s *KnowledgeStore) ByID(ctx, id string) (domain.KnowledgeNote, error)
func (s *KnowledgeStore) ByScopePath(ctx, projectID, vaultID string, isGlobal bool, rel string) (domain.KnowledgeNote, error)
func (s *KnowledgeStore) DeleteLinksFrom(ctx, fromID string) error
```

- [ ] Test: upsert note with link to second path; edge row; resolve when target exists.

- [ ] Commit: `feat(store): knowledge upsert and note_links reindex`

---

### Task 43: Hook source publish → knowledge_notes

**Files:**
- Modify: `internal/publish/machine.go` finalize path
- Test: promote/direct completes → knowledge_notes row kind=source with
  `relative_path = "source/" + notes.relative_path` and `source_note_id = notes.id`
  (v1 `notes.relative_path` stays source-relative; never prefix notes with `source/`)

- [ ] Commit: `feat(publish): upsert knowledge_notes on source publish`

---

### Task 44: Hook instruction put + compound publish → reindex

**Files:**
- Modify: `internal/store/instructions.go`, `internal/compound/publish.go`

- [ ] After successful write call KnowledgeStore.UpsertFromContent.

- [ ] Commit: `feat: reindex knowledge on instruction and compound writes`

---

### Task 45: Backlinks query

**Files:**
- Modify: `internal/store/knowledge.go`

```go
type Backlink struct {
  FromNoteID string
  FromPath string
  FromTitle string
  Snippet string // optional empty in v1
}

func (s *KnowledgeStore) Backlinks(ctx context.Context, noteID string) ([]Backlink, error)
// WHERE to_note_id = ? OR to_path = note.relative_path within same scope
```

- [ ] Test: A links B → Backlinks(B) includes A.

- [ ] Commit: `feat(store): backlinks query`

---

### Task 46: HTTP backlinks + knowledge read by path

**Files:**
- Create: `internal/httpapi/knowledge_handlers.go`
- Modify: `server.go`

```
GET /api/v1/projects/{id}/knowledge/backlinks?path=|knowledge_id=
GET /api/v1/projects/{id}/notes/{note_id}/backlinks  // resolve via source/ + notes.rel
GET /api/v1/projects/{id}/knowledge/read?path=       // scope-root-relative
GET /api/v1/projects/{id}/knowledge/tree             // source + memory; exclude .agents
```
note_links IDs are knowledge_notes.id only; never assume notes.id == knowledge id.
- [ ] Tests: read memory path; backlinks JSON.

- [ ] Commit: `feat(api): knowledge read tree and backlinks`

---

### Task 47: `BacklinksPanel.svelte`

**Files:**
- Create: `web/src/components/notes/BacklinksPanel.svelte`
- Create: `web/src/components/notes/BacklinksPanel.test.ts`
- CSS tokens `.backlinks`, `.backlinks__item`

**Props:** `items: { title: string; path: string; noteId: string }[]; onopen: (noteId: string) => void`

- [ ] Empty state “No backlinks yet.”

- [ ] Commit: `feat(web): BacklinksPanel`

---

### Task 48: Note viewer integration

**Files:**
- Modify note viewer route/component (find existing notes UI under `web/src/components/notes/` or hub file tab)
- When a project note is open, fetch backlinks and show panel

- [ ] Test: mock API shows backlink title.

- [ ] Commit: `feat(web): show backlinks on project note view`

---

### Task 49: Render wikilinks in MarkdownView (display)

**Files:**
- Modify: `web/src/lib/markdown/render.ts` and/or `MarkdownView.svelte`
- Test: `[[path|Title]]` becomes link with text Title and data-path

**LOCKED:** click handling may be app-level; render `<a class="wikilink" data-path="...">` without navigating externally.

- [ ] Commit: `feat(web): render path wikilinks with title mask`

---

### Task 50: Seed memory detail example parser fixtures

**Files:**
- Create: `internal/knowledge/testdata/sample_memory.md`
- Used by parse tests

- [ ] Commit: `test(knowledge): sample memory fixture`

---

### Task 51: Cross-kind link test (memory → source → AGENTS)

**Files:**
- `internal/store/knowledge_test.go`

- [ ] memory detail links `[[source/intro|Intro]]` and `[[AGENTS]]`; backlinks on source and agents resolve after all upserted.

- [ ] Commit: `test(store): wikilinks across memory source agents`

---

### Task 52: P2 verification gate

```bash
go test ./internal/knowledge/ ./internal/store/ ./internal/publish/ ./internal/httpapi/ -count=1
export PATH="$HOME/.local/node-v22/bin:/usr/bin:$PATH"
npm --prefix web test -- BacklinksPanel markdown
```

- [ ] PASS + commit `test: P2 obsidian links verification gate`

DRAFT_C_COMPLETE

---

# Draft D — P3 Project search (Tasks 60–72)

---

### Task 60: FTS upsert/delete helpers

**Files:**
- Modify: `internal/store/knowledge.go`
- Create: `internal/store/knowledge_fts_test.go`

```go
func (s *KnowledgeStore) ReindexFTS(ctx context.Context, note domain.KnowledgeNote, title, body string) error
// DELETE FROM knowledge_fts WHERE note_id=?; INSERT ...
func (s *KnowledgeStore) RemoveFTS(ctx context.Context, noteID string) error
```

Call from UpsertFromContent after body load.

- [ ] Test: upsert then search finds token.

- [ ] Commit: `feat(store): knowledge FTS5 reindex on upsert`

---

### Task 61: Project search query

**Files:**
- Modify: `internal/store/knowledge.go`

```go
type SearchHit struct {
  NoteID string
  Path string
  Title string
  Snippet string
  Kind domain.KnowledgeKind
  Rank float64
}

func (s *KnowledgeStore) SearchProject(ctx context.Context, projectID, query string, limit int) ([]SearchHit, error)
// empty query → empty hits, nil err
// limit default 20 max 50
// filter knowledge_notes.project_id = projectID AND kind in (source,memory_*,agents,soul,system)
// NEVER join other projects
```

Snippet: simple substring window around first match in body (or FTS snippet if available).

- [ ] Tests: two projects same word → only current project; title match preferred if easy (optional ORDER BY).

- [ ] Commit: `feat(store): SearchProject FTS`

---

### Task 62: HTTP search endpoint

**Files:**
- Modify: `internal/httpapi/knowledge_handlers.go`

```
GET /api/v1/projects/{id}/search?q=&limit=
→ { "hits": [ SearchHit... ] }  // note_id field = knowledge_notes.id; include path scope-root
```

- [ ] Test: index two notes, search returns one.

- [ ] Commit: `feat(api): project search endpoint`

---

### Task 63: Agent tool `search_project`

**Files:**
- Modify: `internal/agent/tools/` (new file `knowledge.go` or extend registry)
- Modify: runner tool list when session home=project

```go
// ToolSearchProject = "search_project"
// args: {"query": string, "limit": optional int}
// result JSON: {hits: [...]}
```

Workspace tools unchanged. Grant: always on for project sessions in slice 1 (no extra flag).

- [ ] Test: tool registry includes search_project; handler calls store.

- [ ] Commit: `feat(agent): search_project tool`

---

### Task 64: Agent tools `read_knowledge` + `list_knowledge`

**Files:**
- Same tools package

```go
// read_knowledge {path} — read under project root source|memory|instructions only
// list_knowledge {path?} — list directory entries, no .agents, no sessions
```

Security: rooted open via fsroot; reject escape.

- [ ] Tests: read AGENTS.md; reject `../`.

- [ ] Commit: `feat(agent): read_knowledge and list_knowledge tools`

---

### Task 65: Wire tools into Runner for project home

**Files:**
- Modify: `internal/agent/runner.go`

- [ ] When building tools list, if session.Home==project include knowledge tools; vault/global may include read limited later — **slice 1:** knowledge tools **project only**.

- [ ] Commit: `feat(agent): enable knowledge tools on project sessions`

---

### Task 66: `KnowledgeSearch.svelte`

**Files:**
- Create: `web/src/components/notes/KnowledgeSearch.svelte`
- Create: `web/src/components/notes/KnowledgeSearch.test.ts`
- CSS: `.knowledge-search`

**Props:** `projectId: string; onopen: (hit) => void`

Debounced input → GET search → list hits (title, path, snippet).

- [ ] Commit: `feat(web): KnowledgeSearch component`

---

### Task 67: Place search on project notes / hub

**Files:**
- Modify project notes view / `ProjectHubPage` / notes panel — wherever source tree lives
- Test: search field present when project open

- [ ] Commit: `feat(web): project knowledge search entry point`

---

### Task 68: Spec stub comment for vault/global search grants

**Files:**
- Modify: `docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md` only if needed — prefer code comment in session types:

```go
// Future: session.tool_grants search_vault / search_global — not in slice 1
```

- [ ] Commit: `docs: note deferred vault global search grants in code`

---

### Task 69: Reindex existing source notes migration helper

**Files:**
- Create: `internal/store/knowledge_backfill.go`
- Call on app boot once: for each ready notes row, if knowledge missing upsert from disk

- [ ] Test: existing source file gets fts row after backfill.

- [ ] Commit: `feat(store): backfill knowledge_notes from source notes`

---

### Task 70: Instruction editor minimal UI (global + project)

**Files:**
- Create: `web/src/components/settings/InstructionEditor.svelte`
- Wire settings page + project overview tabs for SOUL/SYSTEM/AGENTS

- [ ] Test: save calls PUT.

- [ ] Commit: `feat(web): instruction editors for soul system agents`

---

### Task 71: frontend-ui-craft pass checklist

**Files:** craft-sensitive components only

- [ ] Load `frontend-ui-craft` skill during implementation
- [ ] Tokens only; no indigo scaffold
- [ ] Rebuild `web/dist` before any Go static UI claim

```bash
export PATH="$HOME/.local/node-v22/bin:/usr/bin:$PATH"
npm --prefix web run build
```

- [ ] Commit: `style(web): craft pass knowledge compound search ui`

---

### Task 72: Final slice verification gate

```bash
go test ./... -count=1
export PATH="$HOME/.local/node-v22/bin:/usr/bin:$PATH"
npm --prefix web test
npm --prefix web run build
```

Manual checklist:
- [ ] Compound approve updates AGENTS+memory; finished_at set
- [ ] Backlinks show on linked notes
- [ ] Project search returns source+memory hits only for that project
- [ ] Prompt load isolation still green
- [ ] Session focus composer test green

```bash
git commit --allow-empty -m "test: slice1 obsidian memory knowledge verification gate"
```

**Out of scope confirmation (do not implement):** graph canvas, vault/global FTS grants UI, auto-compound, title-only wikilinks.

DRAFT_D_COMPLETE
