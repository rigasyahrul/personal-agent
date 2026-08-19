# Superpowers (obra/superpowers)

This project has [Superpowers](https://github.com/obra/superpowers) installed under `.agents/skills/`.

## Bootstrap (session start)

At the start of every conversation — before any clarifying questions, exploration, or action — load and follow the `using-superpowers` skill.

If there is even a small chance a Superpowers skill applies, invoke it with the Skill tool before proceeding. Process skills take priority:

- Creative / build work → `brainstorming` first (hard gate: no implementation until design is approved)
- Bugs / unexpected behavior → `systematic-debugging` first
- Implementing a planned task → `test-driven-development` (and related plan/execution skills)
- About to claim done → `verification-before-completion`
- After non-trivial work / user asks to compound → `compounding-engineering`

Announce which skill you are using, then follow it exactly.

## Installed Superpowers skills

- `using-superpowers` — mandatory skill-check bootstrap
- `brainstorming` — design before code
- `writing-plans` / `executing-plans` / `subagent-driven-development`
- `test-driven-development` / `systematic-debugging` / `verification-before-completion`
- `using-git-worktrees` / `finishing-a-development-branch`
- `requesting-code-review` / `receiving-code-review`
- `dispatching-parallel-agents` / `writing-skills`
- `compounding-engineering` — after a unit of work: append lesson + codify into AGENTS.md / tests / skills
- `synthesize-memory` — promote recurring lessons (3+) into durable standing rules

## Compounding engineering (keep it simple)

Pattern: [Compounding Engineering](https://www.agentic-patterns.com/patterns/compounding-engineering-pattern/) — each unit of work should make the next easier. Codify learnings so agents stop repeating mistakes.

| What | Where |
|------|--------|
| Specs | `docs/superpowers/specs/YYYY-MM-DD-*-design.md` |
| Plans (+ optional lock/drafts) | `docs/superpowers/plans/YYYY-MM-DD-*.md` |
| **Lessons learned** | **`docs/memory/lessons.md`** (one stable file; newest first) |
| Standing rules | **This file** (short bullets below) |

**Memory layers:** Standing rules here load every session. `docs/memory/lessons.md` is on-demand evidence (compound / synthesize / repeated trap) — do not dump the whole diary into every prompt.

After non-trivial work or a user correction: run **`compounding-engineering`** (prepend lesson under `docs/memory/lessons.md`, standing bullet here if it should load every session). Periodically run **`synthesize-memory`** when lessons recur.

### Standing rules

- **Ship = push.** Commit is not enough. After ship/done/archive: `git push`, confirm not `ahead of origin`, then archive the thread. → `docs/memory/lessons.md`
- **Plans live under Superpowers**, not memory: `docs/superpowers/plans/`. Memory is lessons only (`docs/memory/lessons.md`).
- **Big multi-agent plans:** lock + one assembled plan + Canonical contracts; high-stakes review until Approved. → `docs/memory/lessons.md`
- **No extra doc trees** (`docs/solutions/`, process/planning splits) unless the user asks. Prefer one lessons file + AGENTS.md.
- **Lessons path is stable:** always `docs/memory/lessons.md`. Prepend newest-first; refresh the Index. Never create per-session `YYYY-MM-DD-lessons.md`.
- **v1 execution:** master thread = `docs/superpowers/HANDOFF-master-execution.md` + board `STATUS-v1.md`. Workers implement phases; master coordinates merge/ship.
- **High-stakes review on workers:** **`consulting-grok-review`** via a **new Grok 4.5 thread** (`amp --mode grok45 -ox -x '…'` + `reviewer-prompt` contract). Do **not** use built-in `oracle` or Task/OpenAI subagents (no ChatGPT). Never substitute silent self-review. → `docs/memory/lessons.md`
- **Darwin is a first-class test target for FS/shell.** Linux-only rename APIs, bash-4 empty arrays, “reject any ancestor symlink”, and chmod-immutable-before-rename all break macOS `make test`. Platform-split syscalls; resolve root path with `EvalSymlinks` then block links *inside* the root; seal trees only after rename. → `docs/memory/lessons.md` (macOS gaps)
- **Makefile UX:** `.DEFAULT_GOAL` is `help`. Public targets need `## description`, a `.PHONY` entry, and inclusion in the matching `print-help-section` list. Do not let bare `make` run tests/build. Root binary from `make build` is gitignored (`/personal-agent`). → `docs/memory/lessons.md`
- **Skill tool miss ≠ skip.** If Amp’s Skill tool says a skill in this file / `.agents/skills/` is “not found”, `Read .agents/skills/<name>/SKILL.md` and follow it (especially `compounding-engineering`, `synthesize-memory`). → `docs/memory/lessons.md`
- **Polled SPA UIs:** never `innerHTML`-replace a focused composer/input on every poll. Patch messages/status/disabled in place; full shell rebuild only on session switch / missing shell. Keep a focus regression test. → `docs/memory/lessons.md` (sessions focus)
- **UI fix ≠ served fix.** If user hits `localhost:8080`, check who listens (`lsof`/Docker) and that served asset bytes match the edit (`curl …/js/…`) before claiming success. Baked images do not see host `web/` until rebuild or a dev mount.
- **Docker local loop:** prod compose stays image-baked (no host source mounts). Live API+web = `make docker-dev` (`docker-compose.yml` + `docker-compose.dev.yml` override, `air`, `..:/src`). Do not put live mounts on the prod compose file. → `docs/memory/lessons.md`, `docs/ops/deploy.md`

## Notes for Amp orbs

- Skills live in `.agents/skills/` (project-local; survives orb recreation when committed).
- To refresh from upstream: clone `https://github.com/obra/superpowers` and copy `skills/*` into `.agents/skills/`.
- Optional: set `SUPERPOWERS_DISABLE_TELEMETRY=1` to disable brainstorming visual-companion telemetry.
