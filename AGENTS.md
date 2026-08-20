# Superpowers (obra/superpowers)

This project has [Superpowers](https://github.com/obra/superpowers) installed under `.agents/skills/`.

## Bootstrap (session start)

At the start of every conversation — before any clarifying questions, exploration, or action — load and follow the `using-superpowers` skill.

If there is even a small chance a Superpowers skill applies, invoke it with the Skill tool before proceeding. Process skills take priority:

- Creative / build work → `brainstorming` first (hard gate: no implementation until design is approved)
- Bugs / unexpected behavior → `systematic-debugging` first
- Implementing a planned task → `test-driven-development` (and related plan/execution skills)
- Frontend / UI visible changes → `frontend-ui-craft` (with TDD); browser vibe-pass before claiming UI done
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
- `frontend-ui-craft` — any frontend/UI touch: screen spec, browser vibe-pass, anti-AI-slop craft gate
- `compounding-engineering` — after a unit of work: codify into AGENTS.md / skills / tests / hooks; optional evidence as `docs/memory/YYYYMMDD-HHmm-slug.md` + thin index row in `docs/memory/lessons.md`
- `synthesize-memory` — promote recurring lesson entries (3+) into durable standing rules

## Compounding engineering (keep it simple)

Pattern: [Compounding Engineering](https://www.agentic-patterns.com/patterns/compounding-engineering-pattern/) — each unit of work should make the next easier. **Codify first** so agents stop repeating mistakes. A diary dump is not compounding.

| What | Where |
|------|--------|
| Specs | `docs/superpowers/specs/YYYY-MM-DD-*-design.md` |
| Plans (+ optional lock/drafts) | `docs/superpowers/plans/YYYY-MM-DD-*.md` |
| **Standing rules (hot path)** | **This file** (short bullets below) |
| **Skills / hooks / tests** | `.agents/skills/`, repo checks — preferred durable form |
| **Lessons index (list only)** | **`docs/memory/lessons.md`** |
| **Lesson entries (detail)** | **`docs/memory/YYYYMMDD-HHmm-<slug>.md`** — selective; not every session |

**Memory layers:** Standing rules here load every session. Skills load when the task matches. Scan `docs/memory/lessons.md` only when compounding / synthesizing / hunting a trap; open the linked entry file for detail — do **not** dump the corpus into every prompt, and do **not** write an entry for every wrap.

After non-trivial work or a user correction: run **`compounding-engineering`** → codify into AGENTS / skill / hook / test first → if evidence is needed, write **`docs/memory/YYYYMMDD-HHmm-slug.md`** and prepend a list row in **`docs/memory/lessons.md`**. Skip memory when a durable artifact fully captures it. Periodically run **`synthesize-memory`** when entries recur (3+).

### Standing rules

- **Ship = push.** Commit is not enough. After ship/done/archive: `git push`, confirm not `ahead of origin`, then archive the thread. → `docs/memory/` (ship entry)
- **Plans live under Superpowers**, not memory: `docs/superpowers/plans/`. Memory is selective lesson entries + index only.
- **Big multi-agent plans:** lock + one assembled plan + Canonical contracts; high-stakes review until Approved. → `docs/memory/`
- **Large writing-plans:** if the plan has many phases, **same turn** write `…-lock.md`, dispatch **parallel draft agents** into `…-drafts/`, then assemble one plan. Do **not** solo-stall a mega-plan in silence. → `docs/memory/`
- **No extra doc trees** (`docs/solutions/`, process/planning splits) unless the user asks. Prefer AGENTS + skills/tests; memory = index + timestamped entry files only.
- **Compound ≠ diary.** Codify into AGENTS / skill / hook / test first. Evidence (when needed): **`docs/memory/YYYYMMDD-HHmm-<slug>.md`** + thin row in **`docs/memory/lessons.md`**. Never put full lesson bodies in the index. Do not log mechanical work or restate git/PRs. → compounding-engineering skill
- **v1 execution:** master thread = `docs/superpowers/HANDOFF-master-execution.md` + board `STATUS-v1.md`. Workers implement phases; master coordinates merge/ship.
- **Every worker task MUST pass consulting-grok-review before merge.** After each implementer finishes a plan task: package diff → **new** `amp -m grok45 --no-archive-after-execute -x` + `consulting-grok-review`/`reviewer-prompt` (never Task/OpenAI/oracle/self-review/`-ox`). Fix Critical+Important before FF-merge. Ledger `Task N: consulting-grok-review PASS (thread T-…)` **before** `Task N: complete`. Run `.agents/skills/subagent-driven-development/scripts/check-review-gate progress.md` before claiming the plan done. No "tests green = reviewed." Whole-branch consulting-grok-review still required before ship. → skill + `docs/memory/20260820-1505-every-worker-must-pass-consulting-grok-review.md`
- **High-stakes / task review dispatch:** **`consulting-grok-review`** via a **new Grok 4.5 thread** + `reviewer-prompt` contract. Only `amp -m grok45 --no-archive-after-execute -x '…'` (never `-ox` with grok45). Do **not** use built-in `oracle` or Task/OpenAI subagents (no ChatGPT). Never substitute silent self-review. Mandatory per worker task before merge (see standing rule above). → `docs/memory/` + consulting-grok-review skill
- **Grok worker spawn (master):** same command as review. Local `-x` shares the master's workspace — **board/merge only from a separate clean `main` worktree** while a worker runs. Do not stall between unblocked phases (verify → review → merge/push → next). → `docs/memory/20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md`
- **Master merge when FF fails:** board-only or diverged worker-board tips block pure FF — cherry-pick **product** commits or merge-commit; never force-push worker. Re-run gates on the result. → `docs/memory/`
- **Web UI Node 22:** Vite/Vitest need Node `>=22 <23`. Put Node 22 first on `PATH` before `make web-test` / `npm test` (orb may default to Node 20). Build `web/dist` before Go static tests that read it. → `docs/memory/`
- **Darwin is a first-class test target for FS/shell.** Linux-only rename APIs, bash-4 empty arrays, “reject any ancestor symlink”, and chmod-immutable-before-rename all break macOS `make test`. Platform-split syscalls; resolve root path with `EvalSymlinks` then block links *inside* the root; seal trees only after rename. → `docs/memory/` (macOS gaps)
- **Makefile UX:** `.DEFAULT_GOAL` is `help`. Public targets need `## description`, a `.PHONY` entry, and inclusion in the matching `print-help-section` list. Do not let bare `make` run tests/build. Root binary from `make build` is gitignored (`/personal-agent`). → `docs/memory/`
- **Skill tool miss ≠ skip.** If Amp’s Skill tool says a skill in this file / `.agents/skills/` is “not found”, `Read .agents/skills/<name>/SKILL.md` and follow it (especially `compounding-engineering`, `synthesize-memory`). → `docs/memory/`
- **Polled SPA UIs:** never `innerHTML`-replace a focused composer/input on every poll. Patch messages/status/disabled in place; full shell rebuild only on session switch / missing shell. Keep a focus regression test. → `docs/memory/` (sessions focus)
- **Frontend UI craft:** any visible UI work loads `frontend-ui-craft` — short screen spec, browser vibe-pass when reachable (blocked ≠ passed), craft red flags before “done.” → `.agents/skills/frontend-ui-craft/`
- **UI tokens first:** new/edited screens use shared classes in `web/src/app.css` (`btn--*`, `panel`, `field-*`, `entity-card`, `metric-card`, `page-header`, …). Do **not** reintroduce one-off `bg-indigo-600` / scaffold soup. Drop redundant eyebrows (“Global desk”) on global routes; vault name eyebrow OK in vault context. → craft skill + `styles-baseline` / craft-scaffold tests
- **UI fix ≠ served fix.** Go serves **`web/dist`** (not Vite HMR) for `go run` / orb `personal-agent`. After UI edits: `npm --prefix web run build`, confirm `index.html` asset hashes + `curl` body contains the new class/string, then vibe-pass with **cache-bust** (`?v=<ts>#/route`). Baked Docker images need rebuild or `make docker-dev`. → `docs/memory/20260820-0953-ui-craft-tokens-and-dist-cache-bust.md`
- **Docker local loop:** prod compose stays image-baked (no host source mounts). Live API+web = `make docker-dev` (`docker-compose.yml` + `docker-compose.dev.yml` override, `air`, `..:/src`). Do not put live mounts on the prod compose file. → `docs/memory/`, `docs/ops/deploy.md`
- **Unpushed work dies with the orb.** Fresh orbs only have `origin/*`. Local-only design/plan/product commits vanish on recreate. Before handoff/archive (when user allows): push the feature branch, or at least push docs commits. On resume: recover from prior Amp threads via `read_thread` if git objects are gone. → `docs/memory/20260820-1248-orb-loss-unpushed-restore-from-threads.md`
- **Grok45: never `-ox`.** Use only `amp -m grok45 --no-archive-after-execute -x '…'`. `-ox` / `--orb-execute` fails with `Agent mode is invalid` for grok45 (hard, not flaky). One long Grok worker at a time; parallel streams often timeout. Interrupted workers: `amp threads continue T-… -x '…'` or finish small wrap-up on the master after verify. → spawn lesson + session-focus handoff
- **Session focus composer:** keep the Agent `<form>` **mounted** across file-tab switches (hide with `hidden`/`inert` OK). Never destroy/recreate the form on tab or poll. `SessionChat.focus.test.ts` is the gate. → `AGENTS.md` polled-SPA rule
- **Workspace file `kind` may be omitted.** `isPromotableWorkspaceFile` treats missing/`file` + `.md` as promotable; directories never. Normalize in SessionFileTab when loading. Covered by `promote.test.ts` + `SessionFileTab.test.ts`.
- **Ignore `.worktrees/`.** Isolated agent checkouts live under `.worktrees/`; keep that path gitignored so workers never dirty the index with nested checkouts.

## Notes for Amp orbs

- Skills live in `.agents/skills/` (project-local; survives orb recreation when committed).
- To refresh from upstream: clone `https://github.com/obra/superpowers` and copy `skills/*` into `.agents/skills/`.
- Optional: set `SUPERPOWERS_DISABLE_TELEMETRY=1` to disable brainstorming visual-companion telemetry.
