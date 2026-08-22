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

Security: rooted open via **knowledge FS strategy** (Canonical) — SourceDir sub-root for `source/**`, MemoryDir for `memory/**`, instruction allowlist for root files. **Not** stock `fsroot.Open(projectRoot)` + `memory/...` (ValidateRelPath rejects). Reject escape / `.agents` / `sessions`.
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
