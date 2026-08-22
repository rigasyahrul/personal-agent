# Copy-paste: Master thread prompt (Obsidian Memory + Knowledge P0→P3)

**Use:** Open a **new** Amp thread and paste the fenced block below as the first message.  
**Full handoff:** `docs/superpowers/HANDOFF-obsidian-memory-knowledge-master.md`  
**Board:** `docs/superpowers/STATUS-obsidian-memory-knowledge.md`  
**Plan:** `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md`  
**Design/plan final gate:** consulting-grok-review `T-01a02a5c` — locked **Safe with Minor**

---

```
You are the MASTER execution coordinator for personal-agent: Obsidian Memory + Knowledge (P0→P3).

## Bootstrap (do first)
1. Load skill `using-superpowers`.
2. Read and obey:
   - docs/superpowers/HANDOFF-obsidian-memory-knowledge-master.md
   - docs/superpowers/STATUS-obsidian-memory-knowledge.md
   - docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md
     → read **Canonical contracts** section FIRST (wins over draft task prose)
   - docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md
3. AGENTS.md standing rules apply.

## Mission
Drive the locked plan end-to-end: **P0 (tasks 1–12) → P1 (20–35) → P2 (40–52) → P3 (60–72) → ship**.

- Do NOT re-brainstorm product.
- Do NOT rewrite Canonical contracts unless a consulting-grok-review returns Critical/Important that requires a docs fix.
- Prefer coordinating workers over solo bulk coding in this thread.
- Update STATUS board continuously; commit + **push** board/progress.

## Hard rules (non-negotiable)
1. **Ship = push.** After each task merge and each phase: `git push`; confirm not `ahead of origin`.
2. **Every implementer task MUST pass consulting-grok-review before merge.**
   - Ledger BEFORE complete: `Task N: consulting-grok-review PASS (thread T-…)`
   - Then: `Task N: complete (commit …, pushed)`
3. **Worker + reviewer dispatch ONLY:**
   `amp -m grok45 --no-archive-after-execute -x '…'`
   - Reviewer: add `-l consulting-grok-review` and use `.agents/skills/consulting-grok-review/reviewer-prompt.md` contract.
   - **NEVER** Amp Task tool, OpenAI subagents, built-in oracle, or `amp -m grok45 -ox`.
4. **One long Grok at a time** (implementer OR reviewer). No parallel Grok writers.
5. If Grok stream times out / no output ~3–4 min: **do not triple-retry** — `amp threads continue T-… -x '…'` once, or small wrap on master after verify.
6. **Worktrees:** local `-x` shares workspace — isolate implementers with git worktrees (`using-git-worktrees`). Master merges from clean main worktree.
7. **TDD** per plan task. Node `>=22 <23` on PATH for web. Rebuild `web/dist` before Go static UI claims.
8. Phases are **sequential**. Do not start P(n+1) until P(n) is merged, tests green, board `done`, pushed.

## Canonical traps (do not let workers “fix” these wrong)
- Dual paths: v1 `notes.relative_path` = source-relative; `knowledge_notes` = scope-root with `source/` + notes.rel
- Knowledge FS: MemoryDir/SourceDir sub-roots — **never** loosen promote `ValidateRelPath` reserved memory/soul
- Migrator must apply `002` (db.go is 001-only today)
- Knowledge tools: multi-handler Runner; work with `workspace_files=false`; no write_knowledge
- Compound: session-derived scope only; ValidateCompoundItems on create+decide+publish; Decide CAS; lessons **server merge**; tools off on generate; PA_COMPOUND_V1 ephemeral
- API `/api/v1` + CSRF mutation on POSTs/PUTs
- UI: `knowledge_id` + open contract (not v1 notes.id for memory)
- Recovery: approved && finished_at null → re-drive publish

## Loop
1. `git fetch`; `git status -sb`; read STATUS + Canonical.
2. Mark phase `in_progress` on board.
3. For each task N in phase order:
   a. Worktree + branch for implementer.
   b. Spawn ONE Grok implementer with handoff §2 worker template (fill phase, task N, branch).
   c. Verify: commits, `go test` (and web tests if needed) on touched packages.
   d. Package diff BASE..HEAD; write reviewer prompt (handoff §3); spawn NEW Grok consulting-grok-review.
   e. Fix Critical+Important; one scoped re-review if needed.
   f. Ledger PASS → FF-merge (or cherry-pick if FF blocked; never force-push worker) → **push** → STATUS row.
4. Phase gate: plan verification task (12 / 35 / 52 / 72) + full relevant tests → board phase=done → push.
5. After P3: whole-branch consulting-grok-review on full diff → fix → push main → STATUS ship=done.
6. Optional: compounding-engineering after ship.

## Merge policy
- Prefer FF-merge worker → main (or phase branch → main).
- If FF fails: cherry-pick product commits or merge-commit; never force-push worker; re-run gates on result.
- Board/merge only from clean main while worker runs in worktree.

## Quality bar to accept a task
- Plan task steps done (TDD)
- Canonical contracts respected
- consulting-grok-review Verdict safe, Critical=none, Important=none (or fixed+re-PASS)
- Pushed

## Start now
1. Read board + Canonical + `git status -sb` + `git log -3 --oneline`.
2. Set P0 `in_progress` on STATUS; commit+push board if needed.
3. Dispatch Task 1 implementer (Grok -x only).
4. Continue until P0–P3 shipped or BLOCKED (report blocker clearly).
```

---

## After you paste

Master should immediately update `STATUS-obsidian-memory-knowledge.md` and start Task 1. You only need to approve/unblock if it asks.
