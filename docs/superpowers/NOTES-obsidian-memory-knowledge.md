# Coordinator notes — Obsidian Memory + Knowledge

**Session:** 2026-08-22 (Pi, this terminal)  
**Role:** Master execution (adapted: no Amp)  
**Status:** **UNBLOCKED via Amp threads + origin/main** — resume at P1 Task 21

## Operating mode (user override)

Amp subscription gone (`amp` CLI present at `~/.amp/bin/amp` 0.0.1787428878 but **do not use**).  
User: do everything in this terminal; keep notes; check every skill.

| Original master rule | This session |
|---|---|
| Spawn `amp -m grok45 -x` implementers | Execute in-session (`executing-plans`) |
| `consulting-grok-review` via new Grok thread | `requesting-code-review` contract + structured self-review (no Amp). Ledger: `Task N: requesting-code-review PASS (no-amp, Critical none, Important none)` |
| One long Grok at a time | N/A — sequential in this thread |
| Worktrees for implementers | `using-git-worktrees` if we implement |
| Ship = push | Still applies |

Pi has no subagent tool (`using-superpowers/references/pi-tools.md`). Do not fabricate Task/amp calls.

## Skills checked (this session)

**Repo `.agents/skills/`**

| Skill | Verdict this session |
|---|---|
| `using-superpowers` | Loaded first + Pi adapter |
| `executing-plans` | Active — execute locked plan in this session |
| `using-git-worktrees` | Ready; currently on normal `main` checkout |
| `test-driven-development` | Required per task once unblocked |
| `subagent-driven-development` | Not usable (no Amp / no Pi subagent) |
| `consulting-grok-review` | Not usable (Amp credit gone) |
| `requesting-code-review` | Review substitute |
| `receiving-code-review` | For acting on review findings |
| `verification-before-completion` | Before any done claim |
| `finishing-a-development-branch` | After P3, present merge menu |
| `frontend-ui-craft` | When UI tasks start (browser vibe-pass HARD) |
| `compounding-engineering` | After ship / wrap |
| `synthesize-memory` | Not now |
| `systematic-debugging` | On test failures |
| `writing-plans` | Only if user authorizes a **new** plan (prompt said do not rewrite) |
| `brainstorming` | **Do not** re-brainstorm product (user + master prompt) |
| `writing-skills` | N/A |
| `dispatching-parallel-agents` | No subagents on Pi |

**User `~/.agents/skills/`**

| Skill | Verdict |
|---|---|
| `find-skills` | Not asked |
| `reviewing-learning-content` | Vault QA, not this feature |

## Missing sources of truth (blocker)

Searched: this repo (working tree + all branches + git history), `docs/superpowers/**`, `docs/memory/**`, `Workspace/`, `Downloads/`, `Desktop/`. **Zero hits for “obsidian”.**

Expected by the master prompt (none exist):

- `docs/superpowers/HANDOFF-obsidian-memory-knowledge-master.md`
- `docs/superpowers/STATUS-obsidian-memory-knowledge.md`
- `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md` (Canonical contracts first)
- `docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md`

This matches the orb-loss lesson: unpushed Amp-thread docs vanish. Amp threads cannot be restored without Amp credit.

## Repo state at session start

- Branch: `main` (normal checkout, not a worktree)
- After `git fetch`: **ahead 28, behind 41** vs `origin/main`
- Local tip `7aabad0` = unpushed UI polish (Enter-to-send, rail, etc.) + dirty rail/docs
- Origin tip `44cfaa2` = P0 done + P1 Task 20 done (Amp orb pushed)
- Merge-base: `3a91121` — histories diverged; **do not rebase local UI onto origin without a plan**
- SoT files live on **origin/main**, not this checkout
- Implement from worktree on `origin/main` (`feat/obsidian-p1-t21`)

## Amp thread recovery (2026-08-22 evening)

User authorized read-only `amp threads list` / `markdown` (no `-x` implementers).

| Thread | ID | Role |
|---|---|---|
| Coordinate Obsidian knowledge rollout | `T-01a02a65-f7b3-77fe-ac80-85624b3787a6` | **Master** — P0 1–12 + Task 20; next=21 |
| Project knowledge graph | `T-01a02a03-36e5-76aa-8ecd-f3b4dc2f18bb` | Design + plan assemble |
| Obsidian memory layout prompt | `T-01a02a32` | Early layout |
| Knowledge path helpers … Task 20 review | `T-01a02a67` … `T-01a02aa3` | Workers + reviews |

Master last state (thread archived mid-ship): P0 `done`, P1 `in_progress`, Task 20 `224f18d` + review `T-01a02aa3` PASS, tests green on orb. Next: **Task 21 Decide + timestamps**.

Exports: `/tmp/amp-obsidian/{master,knowledge-graph,layout-prompt,vault-phase}.md`

## Resume board (from origin STATUS)

| Phase | Status | Next |
|---|---|---|
| P0 1–12 | done + pushed | — |
| P1 20 | done `224f18d` | — |
| P1 21 | done `fa80874` | — |
| P1 22 | done `76ca6dd` | — |
| P1 23 | done `e365cd9` | — |
| P1 24 | done `2591eac` | **25 generate via Runner** |
| P1 25–35 | todo | after 25 |
| P2 / P3 / ship | blocked on P1 | |

## Divergence warning

Local `main` 28 UI commits are **not** on origin. Origin 41 obsidian commits are **not** in this working tree. Keep them separate until user asks to merge UI onto origin.
