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
- Modify: `internal/db/db.go` — **must** apply `002` (today only hard-codes `001`; change to loop or explicit 002)
- Test: empty `Open` → `knowledge_notes` table exists
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
