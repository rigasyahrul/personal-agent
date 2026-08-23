# Pi master + SQLite 1-conn + ship recovery

**Date:** 2026-08-23  
**Tags:** pi, master, sqlite, knowledge, compound, ship, vibe-pass, worktree

**Task:** Drive Obsidian Memory + Knowledge P1–P3 on Pi (`pi -p` worktrees from `origin/main`).

**Wrong / mistakes:**
- Holding a SQLite `Query` cursor while calling `ByScopePath` / another query deadlocks (`SetMaxOpenConns(1)`). Task 69 hung until rows were drained first.
- Open memory emitted `memory/lessons.md` as project-note without `note_id` → `workspaceFile` → `ValidateRelPath` reserved → 400.
- Laptop `make docker-dev` `:8080` is not the worktree (and may be SSH). Vibe-pass must serve the worktree.
- `pi -p` killed at 300s often has an empty tee log; session jsonl still has the work. Triple-spawn wastes the session.
- Whole-branch ship review: GET recovery exists, but decide retry skips publish when `before != pending`, and SessionChat dismisses on any HTTP 200 (`failed` / unfinished included).

**What worked:** Disjoint `pi -p` worktrees from `origin/main`; cherry-pick onto one tip; FF `HEAD:main`. Knowledge FS = MemoryDir/SourceDir, never promote `ValidateRelPath` on `memory/…`. `finishRun(context.Background())` for compound.

**Rule (next agent):** Use the new `AGENTS.md` Pi / SQLite / knowledge-path / compound-recovery / worktree-vibe bullets. Do not mark ship=done until decide+UI re-drive `approved && finished_at == null`.

**Codified into:** `AGENTS.md` standing rules; `.agents/skills/frontend-ui-craft/SKILL.md` worktree vibe-pass; `docs/superpowers/HANDOFF-obsidian-memory-knowledge-master.md` wrap note; existing gates `TestBackfillReadySourceNotes*`, `SessionFileTab.test.ts`, `TestCompoundGETRecoversApprovedUnfinished`

**Evidence:** `origin/main` through `179dbed`; ship review session `ship-review`; backfill fix `de7f3ad`; Open memory fix `de09a43`
