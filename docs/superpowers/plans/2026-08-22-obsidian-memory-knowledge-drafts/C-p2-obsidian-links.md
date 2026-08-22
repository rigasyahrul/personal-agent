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
  // NormalizedPath: scope-root form WITH .md — same as knowledge_notes.relative_path
  NormalizedPath string
}

func ParseWikilinks(body string) []Wikilink
func NormalizeWikilinkTarget(target string) (string, error)
// reject .., absolute, empty, NUL
// append .md if missing; bare AGENTS|SOUL|SYSTEM → AGENTS.md|SOUL.md|SYSTEM.md
// do NOT strip .md (Canonical join key)
```

Regex from header: `\[\[([^\]|#]+)(?:\|([^\]]+))?\]\]`

- [ ] Tests: `[[memory/a|Title]]` → `memory/a.md`; `[[source/x.md]]` → `source/x.md`; `[[AGENTS]]` → `AGENTS.md`; reject `[[../x]]`.

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
