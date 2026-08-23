# Handoff: Master execution — Obsidian Memory + Knowledge (P0→P3)

**Date:** 2026-08-22  
**Role:** One **master** Amp thread coordinates. Prefer **not** bulk-implementing all code in the master; spawn **Grok 4.5 workers** per task/phase, merge only after gates.  
**Plan SoT:** `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md`  
**Spec SoT:** `docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md`  
**Board:** `docs/superpowers/STATUS-obsidian-memory-knowledge.md`  
**Canonical contracts:** plan header (wins over draft task prose)  
**Design/plan final gate:** consulting-grok-review `T-01a02a5c` — package **locked Safe with Minor**

**2026-08-23 ship:** P0–P3 + recovery + laptop UI merge are on `origin/main`. Workers were `pi -p`. Recovery Important fixed in `0ccb51f` (decide retry + SessionChat keep card). Board **ship=`done`**.

---

## Non-negotiable rules (from AGENTS.md)

1. **Ship = push.** Commit alone is not done. After phase/ship: `git push`, confirm not `ahead of origin`.
2. **Every implementer task MUST pass consulting-grok-review** before merge.  
   Ledger: `Task N: consulting-grok-review PASS (thread T-…)` **before** `Task N: complete`.
3. **Worker dispatch:** only  
   `amp -m grok45 --no-archive-after-execute -x '…'`  
   **Never** Task tool, OpenAI subagents, built-in oracle, or `amp -m grok45 -ox`.
4. **One long Grok worker at a time.** If stream times out / no file ~3–4 min: do **not** triple-retry — continue once or finish small wrap on master after verify.
5. **Local `-x` shares workspace** — use **git worktrees** for implementers while master holds clean merge base (`using-git-worktrees`).
6. **Canonical contracts win** over draft snippets. Spec wins on product behavior.
7. **TDD** per plan task. Node 22 on PATH for web tests. Rebuild `web/dist` before Go static UI claims.
8. After phase done: update STATUS board, commit board, **push**.

---

## Phase map

| Phase | Tasks | Goal |
|-------|-------|------|
| **P0** | 1–12 | Layout seed, migrator `002`, instructions API, `BuildSessionPrompt`, runner inject |
| **P1** | 20–35 | Explicit compound, skill, proposal/decide/publish, review card, rail memory |
| **P2** | 40–52 | Frontmatter, path wikilinks, backlinks API/UI |
| **P3** | 60–72 | FTS, search UI, knowledge tools (`workspace_files=false`), craft + final gate |
| **Ship** | — | Whole-branch consulting-grok-review + merge/push main |

Phases are **sequential**. Do not start P(n+1) until P(n) is merged, pushed, and board `done`.

---

## 1) MASTER start prompt (paste into a **new** Amp thread)

Copy everything inside the fence:

```
You are the MASTER execution coordinator for personal-agent: Obsidian Memory + Knowledge (P0→P3).

## Bootstrap
1. Load skill `using-superpowers` first.
2. Read and follow:
   - docs/superpowers/HANDOFF-obsidian-memory-knowledge-master.md
   - docs/superpowers/STATUS-obsidian-memory-knowledge.md
   - docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md  (Canonical contracts section FIRST)
   - docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md
3. AGENTS.md standing rules apply (ship=push, consulting-grok-review every task, grok45 -x only, one Grok at a time, worktrees).

## Your job
- Own the board STATUS-obsidian-memory-knowledge.md
- Drive the locked plan phase-by-phase (P0 → P1 → P2 → P3 → ship)
- Do NOT re-brainstorm product. Do NOT rewrite Canonical contracts unless a consulting-grok-review Critical/Important forces a docs fix.
- Prefer coordinating Grok workers over solo bulk implementation.
- Never use Task/OpenAI/oracle/-ox for workers or reviews.

## Hard gates
- Every plan task: implement (TDD) → package diff → NEW amp -m grok45 --no-archive-after-execute -x with consulting-grok-review/reviewer-prompt → fix Critical+Important → then FF-merge/push.
- Ledger line BEFORE complete: Task N: consulting-grok-review PASS (thread T-…)
- Run .agents/skills/subagent-driven-development/scripts/check-review-gate progress.md if you maintain progress.md — or keep STATUS ledger equivalent.
- Phase gate: all tasks in phase PASS + go test ./... (+ web tests if phase touched web) + push.
- Final ship: whole-branch consulting-grok-review PASS + push main + board ship=done.

## Loop (until all phases done or BLOCKED)
1. git fetch; git status -sb; read STATUS board; read Canonical contracts.
2. Pick next ready phase (dependencies satisfied).
3. For each task in phase order:
   a. Ensure isolated worktree for implementer (using-git-worktrees) when using local -x.
   b. Spawn ONE Grok implementer:
      amp -m grok45 --no-archive-after-execute -x "$(cat <<'EOF'
      … worker prompt from handoff §2, filled …
      EOF
      )"
   c. Poll amp threads list / markdown until done; verify commits + tests.
   d. Spawn NEW Grok reviewer (consulting-grok-review skill + reviewer-prompt):
      amp -m grok45 --no-archive-after-execute -l consulting-grok-review -x "$(cat review-prompt.txt)"
   e. If Critical/Important: fix (implementer or master small fix) → scoped re-review once.
   f. Ledger PASS → merge to integration branch or main per handoff → push → update STATUS.
4. After phase: phase verification commands from plan Task 12/35/52/72 → board phase=done → push.
5. After P3: whole-branch consulting-grok-review → ship push → compound if needed.

## Parallelism
- One implementer Grok at a time.
- One reviewer Grok at a time (can follow implementer immediately after).
- Never two writers on same worktree.

## Merge policy
- Prefer FF-merge from worker branch to main (or phase branch then main).
- If FF fails: cherry-pick product commits or merge-commit; never force-push worker; re-run gates.
- Board/merge only from clean main worktree while worker runs elsewhere.

## Report to user when blocked
- Missing secrets, flaky infra, Canonical contradiction, review deadlock after one re-review.

Start now: read board + plan Canonical + git status; set P0 in_progress; dispatch Task 1.
```

---

## 2) WORKER implementer prompt template (master fills braces)

```
You are a Grok implementer worker for personal-agent.

## Bootstrap
Load using-superpowers. Then test-driven-development for this task.
If UI: also frontend-ui-craft.

## Authority
- Plan SoT: docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md
- Canonical contracts in that plan WIN over any draft/task snip contradiction.
- Spec: docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md
- Do NOT reopen product decisions. Do NOT skip tests.

## Your assignment
- Phase: {P0|P1|P2|P3}
- Task number(s): {N} only (do not start N+1 unless master said batch)
- Branch/worktree: {branch}
- End when Task N steps complete, tests green for touched packages, committed.

## Hard rules
- TDD: failing test → impl → pass → commit
- amp worker already is grok45; do not spawn Task/OpenAI
- Knowledge FS: MemoryDir/SourceDir sub-roots; NEVER loosen promote ValidateRelPath for memory
- Routes under /api/v1; CSRF mutation on PUTs/POSTs
- notes.relative_path stays source-relative; knowledge uses source/ + notes.rel
- No write_knowledge tool; compound never writes source/**

## Deliverable report (end of turn)
- Commits SHAs
- Tests run + results
- Files touched
- Anything blocked / Canonical ambiguity
- Ready for consulting-grok-review: YES/NO

Implement Task {N} now.
```

---

## 3) REVIEWER prompt template (consulting-grok-review)

Master writes a file then:

```bash
amp -m grok45 --no-archive-after-execute -l consulting-grok-review \
  -x "$(cat /path/to/task-N-review-prompt.txt)"
```

Prompt body must use `.agents/skills/consulting-grok-review/reviewer-prompt.md` structure with:

```
UNRESOLVED QUESTION:
Does the Task {N} diff correctly implement the plan task and Canonical contracts without Critical/Important bugs (security, path escape, scope bleed, v1 notes collision, missing tests)?

ALREADY CHECKED:
- Plan task {N} text
- git diff {BASE}..{HEAD} --stat
- tests: {commands + results}

DECISION IMPACT:
- If NOT safe: list Critical/Important; master blocks merge
- If safe: master may merge Task N

INTENDED BEHAVIOR:
{quote plan task + relevant Canonical bullets}

SETTLED CONSTRAINTS:
- Canonical wins; no product reopen; TDD required

SCOPE:
- @paths from the task Files: section
- git range BASE..HEAD

IGNORE:
- Unrelated files; style-only nits; future phases
```

Require mandatory sections: Verdict, Evidence, If wrong, Smallest fix, Severity (Critical/Important/Minor), Reversal, Out of scope.

**Merge only if Verdict is safe AND Critical=none AND Important=none** (or Important fixed + re-review PASS).

---

## 4) Phase verification commands

```bash
# Always
go test ./... -count=1

# If web touched
export PATH="$HOME/.local/node-v22/bin:/usr/bin:$PATH"
npm --prefix web test
npm --prefix web run build   # before Go static / vibe claims
```

P0 gate ≈ plan Task 12; P1 ≈ 35; P2 ≈ 52; P3 ≈ 72.

---

## 5) Whole-branch ship gate

After P3 complete on main (or release branch):

1. `go test ./...` + web test + web build  
2. Whole-branch consulting-grok-review (diff origin/main…HEAD or phase base…HEAD)  
3. Fix Critical/Important  
4. `git push`  
5. STATUS ship=done  
6. Optional: compounding-engineering  

---

## 6) Master checklist (print)

- [ ] P0 all tasks review PASS + pushed  
- [ ] P1 all tasks review PASS + pushed  
- [ ] P2 all tasks review PASS + pushed  
- [ ] P3 all tasks review PASS + pushed  
- [ ] Whole-branch consulting-grok-review PASS  
- [ ] main not ahead of origin  
- [ ] STATUS board final  

---

## Quick copy: user “run this later”

1. Open **new** Amp thread.  
2. Paste **§1 MASTER start prompt**.  
3. Let master drive; you only unblock secrets/approvals.  
4. Do not paste worker prompts yourself unless master asks.  
