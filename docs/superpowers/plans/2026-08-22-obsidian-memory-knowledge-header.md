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

### Knowledge note identity

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

// Scope: exactly one of project_id, vault_id (memory-only vault rows), or global flag.
// Prefer columns: project_id NULL, vault_id NULL, is_global INTEGER NOT NULL DEFAULT 0
// CHECK exactly one scope owner.
// relative_path: POSIX under scope root, e.g. "source/a.md", "memory/lessons.md", "AGENTS.md"
// notes table: EXTEND existing `notes` OR new `knowledge_notes` — ** Canonical choice: new table
// `knowledge_notes` ** and keep v1 `notes` for source promote/review compatibility;
// source files appear in BOTH (knowledge_notes mirrors ready source notes) OR
// knowledge_notes is supersource and notes.knowledge_note_id FK — **pick in Task 1 migration:
// Decision LOCKED: extend `notes` with kind + scope columns is riskier for review FKs.
// LOCKED: create `knowledge_notes` for all indexed md; keep `notes` as today for source review;
// on source publish, upsert matching knowledge_notes row (same relative path under project).
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

Statuses: `pending` → (`approved`|`rejected`) → if approved publish → set `finished_at` on success or `failed`+`finished_at` on publish error. Reject sets `decided_at`+`finished_at` (no publish).

### Wikilink regex / normalize

- Match: `\[\[([^\]|#]+)(?:\|([^\]]+))?\]\]`
- Trim target; strip optional trailing `.md`; reject targets with `..`, absolute, empty.
- Store edge raw target + normalized relative path string.
- Resolution root = knowledge note’s scope root.

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

Project search SQL filters `knowledge_notes` to `project_id = ?` and kinds in scope corpus; join fts. Never search other projects.

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

### HTTP routes (mount under existing auth)

```
GET  /api/projects/{id}/instructions/{name}     name=soul|system|agents
PUT  /api/projects/{id}/instructions/{name}
GET  /api/global/instructions/{name}
PUT  /api/global/instructions/{name}

POST /api/sessions/{id}/compound                body: {request_key, user_context?}
GET  /api/sessions/{id}/compound/{proposal_id}
POST /api/sessions/{id}/compound/{proposal_id}/decide
     body: {request_key, decision: approve|reject, items?: edited items}

GET  /api/projects/{id}/notes/{note_id}/backlinks
GET  /api/projects/{id}/search?q=&limit=

GET  /api/projects/{id}/knowledge/tree          # optional if reuse notes tree + memory
GET  /api/projects/{id}/knowledge/read?path=
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
