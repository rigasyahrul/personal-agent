# Grok draft worker A — P0 layout + prompt

You are a plan-draft worker for personal-agent. **Write only** this file:

`docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge-drafts/A-p0-layout-prompt.md`

Do **not** implement product code. Do **not** write the assembled plan. Do **not** use Task/OpenAI.

## Read first (required)

1. `docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md`
2. `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge-lock.md`
3. `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge-header.md` (Canonical contracts — **obey names/paths**)
4. Skim: `internal/layout/layout.go`, `internal/store/projects.go`, `internal/store/vaults.go`, `internal/agent/runner.go`, `internal/db/migrations/001_init.sql`

## Your task range: Tasks 1–12 (P0 only)

Cover:

1. Layout helpers + tests (GlobalRoot, VaultRoot, InstructionPath, MemoryDir, LessonsPath, CompoundingSkillPath, Ensure* seed)
2. Embedded default compounding skill file path `internal/agent/skills/compounding/SKILL.md` (content outline OK; full skill prose can be short but must meet spec §14)
3. Migration for `knowledge_notes` skeleton (enough for instructions/memory rows; links/FTS can be stubbed or full DDL if clean — align header). Do **not** implement compound API (that's B) but migration may create empty compound_proposals table if cleaner one migration — prefer **one migration** `002_knowledge.sql` with knowledge_notes + compound_proposals + note_links + fts as in header.
4. Seed on: app init/global ensure, vault create, project create
5. Instruction GET/PUT store + HTTP for project + global (soul|system|agents)
6. `BuildSessionPrompt` + tests for project/vault/global load isolation (no fallback)
7. Wire runner to prepend prompt sections before model call
8. Replace/prepare ProjectRail memory tab contract note only if needed for P0 read path — prefer minimal; full rail UI is B/C

## Format (mandatory)

For each Task N:

```markdown
### Task N: Title

**Files:**
- Create: ...
- Modify: ...
- Test: ...

**Interfaces:**
- Consumes: ...
- Produces: exact Go/TS signatures

- [ ] **Step 1: Write the failing test**
(code)

- [ ] **Step 2: Run test to verify it fails**
Run: `...`
Expected: ...

- [ ] **Step 3: Minimal implementation**
(code)

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

- No TBD/TODO/placeholders
- No "similar to Task N" without full content
- TDD every task
- Use Canonical contract names exactly
- Tasks 1–12 only (leave 13–19 unused if needed for gaps — renumber within 1–12)
- End file with a one-line: `DRAFT_A_COMPLETE`

When done, ensure the draft file exists and is non-trivial (>100 lines).
