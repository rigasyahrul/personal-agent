# Handoff: Master execution coordinator

**Date:** 2026-08-12  
**Role:** One **master** Amp thread coordinates the project. It does **not** implement bulk application code itself. It spawns, tracks, reviews, and ships via **worker** threads (and in-thread subagents when cheaper).  
**Plan SoT:** `docs/superpowers/plans/2026-08-12-personal-agent-v1.md`  
**Spec SoT:** `docs/superpowers/specs/2026-08-12-personal-agent-design.md`  
**Lessons:** `docs/memory/2026-08-12-lessons.md` + `AGENTS.md`

---

## Architecture

```text
┌─────────────────────────────────────────────────────────┐
│  MASTER THREAD (this handoff)                           │
│  — board owner, dispatch, merge order, final ship gate  │
│  — reads worker reports; updates STATUS.md              │
└────────────┬────────────────────────────▲───────────────┘
             │ spawn / brief              │ report back
             ▼                            │
   ┌─────────────────┐  ┌─────────────────┐  ┌────────────┐
   │ Worker: Phase N │  │ Worker: Phase M │  │ …          │
   │ implement→review│  │ (only if indep.)│  │            │
   │ → push → report │  │                 │  │            │
   └─────────────────┘  └─────────────────┘  └────────────┘
```

**Default:** phases are **sequential** (each depends on the previous). Master runs **one worker phase at a time** unless the board marks work **parallel-safe**.

**Inside a worker:** use Superpowers `subagent-driven-development` (fresh subagent per plan task + review). That is “parallelism inside one phase,” not multiple phases racing on `main`.

**Ship rule:** worker (or master) only claims shipped after **`git push`** and `git status -sb` is not ahead of origin. See lessons.

---

## 1) Start prompt — MASTER thread (paste into new thread)

```
You are the MASTER execution coordinator for personal-agent v1.

Use Superpowers. Load using-superpowers first. Then follow:
docs/superpowers/HANDOFF-master-execution.md

## Your job
- Own the execution board: docs/superpowers/STATUS-v1.md
- Drive the approved plan phase-by-phase:
  docs/superpowers/plans/2026-08-12-personal-agent-v1.md
- Spec SoT: docs/superpowers/specs/2026-08-12-personal-agent-design.md
- Lessons: docs/memory/2026-08-12-lessons.md and AGENTS.md
- Do NOT re-brainstorm product. Do NOT rewrite the plan unless blocked.
- Prefer NOT writing bulk app code in this thread. Coordinate workers.

## Loop (repeat until v1 done or BLOCKED)
1. Read STATUS-v1.md + git status + latest origin/main.
2. Pick the next phase that is ready (dependencies satisfied).
3. Open or continue a WORKER thread with the Worker start prompt
   from the handoff (fill PHASE, TASK range, branch name).
   Prefer: amp threads new  (or user pastes worker prompt in a new thread).
   If Amp Task/subagents are enough for a small phase, you may run
   subagent-driven-development in a dedicated worker-style session
   instead — still update the board the same way.
4. Wait for worker REPORT (template in handoff). If the worker is
   another Amp thread, poll with read_thread / threads list, or ask
   the user to paste the report.
5. Verify: tests, branch, push state, plan checkboxes if claimed done.
6. Mark phase done on STATUS-v1.md. Commit board updates. Push.
7. Only after ALL phases done: final hardening gate, tag/release notes
   if appropriate, compound a short lesson in docs/memory if needed.

## Parallelism rules
- Phases 1→7 are sequential by default (see STATUS-v1.md depends_on).
- Never two workers committing the same files on main without a merge plan.
- Parallel workers only when STATUS says parallel-safe AND branches
  don’t touch the same paths (master assigns path ownership).
- Inside one worker: subagent-driven-development per plan task is OK.

## Quality gates before accepting a phase
- go test ./... green (once go.mod exists)
- Plan Canonical contracts respected (not stale snippets)
- git pushed if worker said shipped
- No open P0 review findings

## Report to user
Keep a short running summary: phase, worker thread id/url, status, blockers.

Start by creating/updating docs/superpowers/STATUS-v1.md from the
template in the handoff if missing, then dispatch Phase 1.
```

---

## 2) Start prompt — WORKER thread (master fills blanks)

```
You are a WORKER thread under the personal-agent MASTER coordinator.

Use Superpowers. Load using-superpowers, then:
- test-driven-development
- subagent-driven-development (preferred) OR executing-plans
- verification-before-completion before claiming done
- finishing-a-development-branch when the phase is complete

## Scope (filled by master)
- Phase: __PHASE_N_NAME__
- Plan tasks: __TASK_START__–__TASK_END__
- Plan: docs/superpowers/plans/2026-08-12-personal-agent-v1.md
- Spec: docs/superpowers/specs/2026-08-12-personal-agent-design.md
- Lock (contracts): docs/superpowers/plans/2026-08-12-personal-agent-v1-lock.md
- Branch: __BRANCH__   (e.g. impl/v1-phase-1-skeleton)
- Path ownership (only touch): __PATHS__
- Do NOT start tasks outside your range.
- Canonical contracts in the plan OVERRIDE conflicting task snippets.

## Rules
- TDD: red → green → commit per plan task.
- Follow plan Interfaces / Global Constraints.
- Go 1.24+, module github.com/rigasyahrul/personal-agent.
- Ship = push your branch (and PR/merge only if master asked).
- Append lessons only for real mistakes: docs/memory/YYYY-MM-DD-lessons.md
- Standing rules: AGENTS.md

## Process
1. git fetch && checkout -b __BRANCH__ from origin/main (or master-specified base).
2. Execute tasks __TASK_START__–__TASK_END__ with subagent-driven-development.
3. After each task: tests pass, commit.
4. End of phase: full go test ./... ; fix until green.
5. Request code review (skill) on the phase diff; fix until approved.
6. Push branch: git push -u origin HEAD
7. Post REPORT to master (template below). Do not archive master.
8. Optional: archive THIS worker thread only after master accepts.

## REPORT template (paste back to master / end of worker thread)

### Worker report
- Phase:
- Tasks:
- Branch:
- Thread:
- Status: DONE | BLOCKED | NEEDS_MASTER
- Commits: (shas)
- Pushed: yes/no (git status -sb)
- Tests: go test ./... → pass/fail (paste tail)
- Review: approved / findings left
- Files touched: (top-level list)
- Blockers: 
- Notes for next phase:
```

---

## 3) Phase board (master owns)

Copy into `docs/superpowers/STATUS-v1.md` on first master run if missing:

| Phase | Tasks | Depends on | Parallel-safe? | Status | Branch | Worker thread | Notes |
|-------|-------|------------|----------------|--------|--------|---------------|-------|
| 1 Skeleton | 1–8 | — | no | todo | impl/v1-p1-skeleton | | Compose, SQLite, auth, empty Home |
| 2 Projects + source | 9–14 | 1 | no | todo | impl/v1-p2-projects | | layout, projects, direct publish |
| 3 Sessions + chat | 15–20 | 2 | no | todo | impl/v1-p3-sessions | | sessions, runner, chat UI |
| 4 Workspace tools | 21–24 | 3 | no | todo | impl/v1-p4-tools | | rooted tools + UI |
| 5 Promote + review | 25–32 | 4 | no | todo | impl/v1-p5-promote-review | | machine, SM-2, review UI |
| 6 Backup | 33–36 | 5 | no | todo | impl/v1-p6-backup | | barrier, S3 optional, restore |
| 7 Hardening | 37–42 | 6 | no | todo | impl/v1-p7-hardening | | §13 acceptance, polish |

**Status values:** `todo` | `running` | `review` | `done` | `blocked`

**Merge policy (default):** each phase merges to `main` (PR or fast-forward) **before** the next worker starts, so workers always branch from green `main`. Master performs or approves the merge after accepting the worker report.

---

## 4) How master “manages other threads”

| Mechanism | When |
|-----------|------|
| **New Amp thread** + Worker start prompt | Default for a full phase (clean context) |
| **`amp threads continue <id>`** | Resume a blocked worker |
| **`read_thread` / thread URL** | Master pulls report without user paste |
| **In-master Task subagents** | Tiny fixes, board updates, merge conflict assist — not whole phases |
| **User paste** | Fallback if tools can’t see worker output |

Master message when dispatching:

```text
Dispatching Phase N → worker thread.
Branch: impl/v1-pN-…
Tasks: A–B
Path ownership: …
Paste the Worker start prompt (filled) into a new Amp thread
(or I create it if CLI allows). Report back with the REPORT template.
```

---

## 5) Final ship (master only)

When all phases `done`:

1. On `main`: `go test ./...`
2. Smoke: health + bootstrap path if feasible in orb
3. Confirm plan acceptance §13 covered by tests
4. `git push` if anything pending
5. Append compound lesson if process improved
6. Summarize to user: what shipped, how to run Compose
7. Archive **worker** threads; archive **master** only when user says project v1 closed

---

## 6) Already done (do not redo)

| Item | Location |
|------|----------|
| Design approved | `docs/superpowers/specs/2026-08-12-personal-agent-design.md` |
| Plan approved (Oracle) | `docs/superpowers/plans/2026-08-12-personal-agent-v1.md` |
| Plan lock / drafts | `…-v1-lock.md`, `…-v1-drafts/` |
| Lessons + AGENTS rules | `docs/memory/2026-08-12-lessons.md`, `AGENTS.md` |
| Greenfield app | still no `go.mod` until Phase 1 |

---

## 7) Anti-patterns

- Master implementing all 42 tasks alone in one megacontext  
- Two phases on one branch fighting  
- “Done” without push  
- Rewriting the product spec mid-flight  
- Ignoring Canonical contracts in the plan  
- Putting plan drafts back under `docs/memory/`
