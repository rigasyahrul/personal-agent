# Superpowers (obra/superpowers)

This project has [Superpowers](https://github.com/obra/superpowers) installed under `.agents/skills/`.

## Bootstrap (session start)

At the start of every conversation — before any clarifying questions, exploration, or action — load and follow the `using-superpowers` skill.

If there is even a small chance a Superpowers skill applies, invoke it with the Skill tool before proceeding. Process skills take priority:

- Creative / build work → `brainstorming` first (hard gate: no implementation until design is approved)
- Bugs / unexpected behavior → `systematic-debugging` first
- Implementing a planned task → `test-driven-development` (and related plan/execution skills)
- About to claim done → `verification-before-completion`

Announce which skill you are using, then follow it exactly.

## Installed Superpowers skills

- `using-superpowers` — mandatory skill-check bootstrap
- `brainstorming` — design before code
- `writing-plans` / `executing-plans` / `subagent-driven-development`
- `test-driven-development` / `systematic-debugging` / `verification-before-completion`
- `using-git-worktrees` / `finishing-a-development-branch`
- `requesting-code-review` / `receiving-code-review`
- `dispatching-parallel-agents` / `writing-skills`

## Compounding engineering (keep it simple)

Pattern: [Compounding Engineering](https://www.agentic-patterns.com/patterns/compounding-engineering-pattern/) — each unit of work should make the next easier. Codify learnings so agents stop repeating mistakes.

| What | Where |
|------|--------|
| Specs | `docs/superpowers/specs/YYYY-MM-DD-*-design.md` |
| Plans (+ optional lock/drafts) | `docs/superpowers/plans/YYYY-MM-DD-*.md` |
| **Lessons learned** | **`docs/memory/YYYY-MM-DD-lessons.md`** (one file; append) |
| Standing rules | **This file** (short bullets below) |

After non-trivial work or a user correction: append a lesson under `docs/memory/`, and add a standing bullet here if it should load every session.

### Standing rules

- **Ship = push.** Commit is not enough. After ship/done/archive: `git push`, confirm not `ahead of origin`, then archive the thread. → details in `docs/memory/2026-08-12-lessons.md`
- **Plans live under Superpowers**, not memory: `docs/superpowers/plans/`. Memory is lessons only.
- **Big multi-agent plans:** lock + one assembled plan + Canonical contracts; Oracle until Approved. → `docs/memory/2026-08-12-lessons.md`
- **No extra doc trees** (`docs/solutions/`, process/planning splits) unless the user asks. Prefer one lessons file + AGENTS.md.
- **v1 execution:** master thread = `docs/superpowers/HANDOFF-master-execution.md` + board `STATUS-v1.md`. Workers implement phases; master coordinates merge/ship.
- **High-stakes review on workers:** default to skill **`consulting-grok-review`** (Task + `reviewer-prompt.md`). Do **not** call built-in `oracle` (no ChatGPT subscription). Never substitute silent self-review for that gate. → `docs/memory/2026-08-12-lessons.md`

## Notes for Amp orbs

- Skills live in `.agents/skills/` (project-local; survives orb recreation when committed).
- To refresh from upstream: clone `https://github.com/obra/superpowers` and copy `skills/*` into `.agents/skills/`.
- Optional: set `SUPERPOWERS_DISABLE_TELEMETRY=1` to disable brainstorming visual-companion telemetry.
