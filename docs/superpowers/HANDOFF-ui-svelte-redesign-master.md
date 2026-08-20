# Handoff: Master execution — UI Svelte redesign

**Date:** 2026-08-19  
**Role:** One **master** Amp thread (orb) coordinates. It does **not** implement bulk application code. It owns the board, spawns **Grok 4.5 workers only**, runs **consulting-grok-review only** for high-stakes review, merges, and ships.  
**Plan SoT:** `docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md`  
**Spec SoT:** `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md`  
**Lock:** `docs/superpowers/plans/2026-08-19-ui-svelte-redesign-lock.md`  
**Board:** `docs/superpowers/STATUS-ui-svelte-redesign.md` (master creates on start)  
**Lessons:** `docs/memory/lessons.md` + `AGENTS.md`

**Pushed baseline on `origin/main`:** includes spec `20b62cd`, plan `d3a2d3a`, compound wrap `d962894` (or later tips on main).

---

## Hard rules (non-negotiable)

| Concern | Rule |
|---------|------|
| **Worker / execution model** | **Grok 4.5 only** — `amp -m grok45 --no-archive-after-execute -x '…'` (prompt file recommended). **`-ox` optional:** if `Agent mode is invalid`, drop `-ox` immediately. No Claude/GPT workers for implementation. |
| **High-stakes review** | **`consulting-grok-review` only** — new Grok 4.5 thread + skill contract (same spawn as workers). **Never** built-in `oracle`, Task/OpenAI reviewers, or silent self-review as a substitute. |
| **Master workspace** | Coordinate only; no bulk app code. Board/merge from a **separate clean `main` worktree** — local Grok `-x` shares the caller's checkout with the worker. |
| **Master** | May edit board, handoffs, tiny merge fixes. Do **not** pause between unblocked phases. |
| **Plan authority** | **Canonical contracts** in the assembled plan win over task snippets. Spec wins over plan if product conflict. |
| **Ship** | Commit ≠ ship. After phase accept: merge/FF as appropriate, **`git push`**, confirm not ahead of origin. |
| **Parallelism** | Phases sequential by default. Parallel workers only if board marks parallel-safe **and** path ownership does not overlap. |

---

## Architecture

```text
MASTER (this thread) — board/merge in isolated main worktree
   │ amp -m grok45 --no-archive-after-execute -x '…'
   │ (-ox best-effort only; fall back to local -x)
   ▼
WORKER phases (Tasks 1–8 → 10–15 → 20–25 → 30–35 → 40–46 → 50–55)
   │ phase done → master: verify (Node 22 web + go) → consulting-grok-review
   ▼
MASTER accept → board commit → git push → dispatch next phase (no stall)
```

---

## 1) MASTER start prompt (paste into new orb thread)

```
You are the MASTER execution coordinator for personal-agent **UI Svelte redesign**.

## Bootstrap
1. Load Superpowers: using-superpowers first.
2. Follow this handoff exactly:
   docs/superpowers/HANDOFF-ui-svelte-redesign-master.md
3. Read AGENTS.md standing rules + docs/memory/lessons.md Index (esp. big plans, docker-dev, sessions focus, consulting-grok-review).

## Sources of truth (do not re-brainstorm)
- Spec: docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md
- Plan: docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md
- Lock: docs/superpowers/plans/2026-08-19-ui-svelte-redesign-lock.md
- Canonical contracts in the plan OVERRIDE conflicting task body snippets.

## Your job
- Own the board: docs/superpowers/STATUS-ui-svelte-redesign.md (create from template in the handoff if missing).
- Drive the plan **phase-by-phase** (task ranges below).
- Spawn **WORKER threads only with Grok 4.5** (`amp -m grok45 --no-archive-after-execute -x …`; `-ox` best-effort). Board/merge from a separate main worktree. No other model for implementation.
- High-stakes review **only** via skill **consulting-grok-review** in a **new Grok 4.5 thread**. Forbidden substitutes: built-in oracle, Task/OpenAI subagent review, ChatGPT, silent self-review claiming “looks good.”
- Prefer NOT writing bulk application code in this master thread. Coordinate, verify, merge, push.
- Do NOT rewrite product design. Plan changes only if truly blocked (document on board).

## Phases (sequential unless board says parallel-safe)

| Phase | Tasks | Focus |
|-------|-------|--------|
| A Tooling + docker HMR | 1–8 | Vite/Svelte scaffold, web/dist, PA_UI_DEV_PROXY, make docker-dev HMR |
| B Shell + auth | 10–15 | Router, shell context, API client, auth, tokens |
| C Global grids | 20–25 | Home, Projects, Vaults searchable grids |
| D Vault context | 30–35 | Enter/leave vault, vault-scoped pages |
| E Project surfaces | 40–46 | Hub, notes, sessions focus-safe, promote |
| F Review + harden | 50–55 | Review, settings, drop legacy, Go tests, docs, gate |

## Loop (until all phases done or BLOCKED)
1. git fetch; read STATUS board; git status -sb; confirm on latest origin/main (or known integration branch).
2. Pick next ready phase (dependencies done).
3. Create worker branch name e.g. impl/ui-svelte-phase-A-tooling.
4. Open a **new Grok 4.5 worker thread** (`amp -m grok45 --no-archive-after-execute -x …`) with the WORKER prompt from the handoff (fill PHASE, TASK range, BRANCH, PATH ownership). Prefer a prompt file. Master stays on a clean main worktree.
5. Wait for WORKER REPORT (template in handoff). Poll thread or ask user to paste.
6. Verify before accept:
   - Claimed tests actually run green (go test ./..., make web-test / npm test as applicable)
   - Canonical contracts respected
   - docker-dev HMR story intact for UI phases (prod compose still baked)
   - Session focus invariant not regressed when sessions land
   - git pushed if worker claimed ship
7. High-stakes gate: run **consulting-grok-review** (new grok45 thread) on the phase diff before merge when the phase is non-trivial (always for A, E, F; use judgment for tiny doc-only — default YES for code phases).
8. Merge/FF to main (or integration branch), update STATUS, commit board, **git push**, confirm not ahead of origin.
9. Dispatch next phase.

## Parallelism
- Default: one worker phase at a time on disjoint branches.
- Inside a worker: subagent-driven-development is OK **only if those subagents are also Grok 4.5** — if the harness cannot guarantee that, worker executes tasks inline on Grok 4.5 instead.
- Never two workers editing the same paths without master’s explicit path split.

## Quality gates (phase accept)
- Plan checkboxes for the range can be marked done only after evidence
- go test ./... green when Go touched
- Frontend tests green when web/ touched
- No open Critical/Important findings from consulting-grok-review
- Ship = push

## Report to user
Short running summary each turn: phase, worker thread URL, status, blockers, last push SHA.

## Start now
1. Create docs/superpowers/STATUS-ui-svelte-redesign.md from the template in the handoff.
2. Confirm origin/main has the plan/spec commits.
3. Dispatch Phase A (Tasks 1–8) to a Grok 4.5 worker.
```

---

## 2) WORKER start prompt (master fills blanks; Grok 4.5 only)

```
You are a WORKER under the personal-agent UI Svelte redesign MASTER.

## Model / review constraints
- You run as **Grok 4.5** only. Do not hand off implementation to other models.
- Do **not** use built-in oracle or ChatGPT/Task reviewers.
- When MASTER asks for high-stakes review, that is MASTER’s job via consulting-grok-review — you still self-verify with tests (verification-before-completion).

## Bootstrap
Load using-superpowers, then:
- test-driven-development
- subagent-driven-development OR executing-plans (only if subagents are Grok 4.5; else execute inline)
- verification-before-completion before claiming done
- finishing-a-development-branch when phase complete

## Scope (MASTER FILLS)
- Phase: __PHASE_LETTER_NAME__
- Plan tasks: __TASK_START__–__TASK_END__
- Plan: docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md
- Spec: docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md
- Lock: docs/superpowers/plans/2026-08-19-ui-svelte-redesign-lock.md
- Branch: __BRANCH__
- Path ownership (only touch): __PATHS__
- Do NOT start tasks outside your range.
- Canonical contracts in the plan OVERRIDE conflicting task snippets.

## Rules
- TDD: red → green → commit per plan task (checkbox steps).
- Prefix commands with rtk when available.
- Production compose stays image-baked; live UI = make docker-dev + PA_UI_DEV_PROXY as plan Tasks 6–8.
- Polled session UI must never steal composer focus (when you touch sessions).
- Ship = push your branch. Report merge readiness; MASTER merges unless told otherwise.
- Lessons only for real mistakes: docs/memory/lessons.md (newest first + Index). Standing: AGENTS.md.

## Done criteria
1. All tasks in range implemented + tests green.
2. Branch pushed to origin.
3. Fill WORKER REPORT below and stop (or wait for master).

## WORKER REPORT (paste back to master)

### Phase
### Tasks completed
### Branch / HEAD SHA
### Pushed? (yes/no + remote)
### Tests run + results
### Files touched (summary)
### Deviations from plan (if any)
### Blockers / risks
### Ready for consulting-grok-review? (yes/no)
```

---

## 3) Board template — `docs/superpowers/STATUS-ui-svelte-redesign.md`

Master creates this file on start:

```markdown
# UI Svelte redesign — execution board

**Master owns this file.**  
**Plan:** docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md  
**Handoff:** docs/superpowers/HANDOFF-ui-svelte-redesign-master.md  
**Rules:** Workers = Grok 4.5 only. Review = consulting-grok-review only.

| Phase | Tasks | Depends | Parallel-safe | Status | Branch | Worker thread | Review | Updated |
|-------|-------|---------|---------------|--------|--------|---------------|--------|---------|
| A Tooling + HMR | 1–8 | — | no | todo | | | | |
| B Shell + auth | 10–15 | A | no | todo | | | | |
| C Global grids | 20–25 | B | no | todo | | | | |
| D Vault context | 30–35 | C | no | todo | | | | |
| E Project surfaces | 40–46 | D | no | todo | | | | |
| F Review + harden | 50–55 | E | no | todo | | | | |

**Status:** todo | running | review | done | blocked

## Log
| When | Event |
|------|--------|
| | Master started |

## Active blockers
_None._

## Master thread
- URL: (fill)
```

---

## 4) consulting-grok-review gate (master)

Before accepting a code phase:

1. Load skill `consulting-grok-review` (if Skill tool misses: Read `.agents/skills/consulting-grok-review/SKILL.md`).
2. Dispatch **new Grok 4.5 thread** with Oracle-shaped task (unresolved question, already checked, decision impact, intended behavior, settled constraints, @files, git range).
3. Act on Critical/Important findings before merge.
4. Record review thread URL on the board.

Example unresolved questions by phase:
- **A:** Does PA_UI_DEV_PROXY + entrypoint give HMR on :8080 without baking mounts into prod compose?
- **E:** Can session poll replace the focused composer DOM node?
- **F:** Do Go web_test/static_test and docker-dev/prod invariants still hold?

---

## 5) Final ship gate (master)

- [ ] All phases `done` on board  
- [ ] `go test ./...` green  
- [ ] Frontend tests green  
- [ ] `make docker-dev` HMR smoke (or documented manual check)  
- [ ] Prod image build path serves `web/dist`  
- [ ] `git push`; main not ahead of origin  
- [ ] Optional: compound lesson if new traps appeared  
```
