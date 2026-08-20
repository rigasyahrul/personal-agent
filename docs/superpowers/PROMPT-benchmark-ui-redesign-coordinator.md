# Coordinator prompt — Benchmark UI redesign (paste into new orb / thread)

Copy everything below the line into a **new** Amp thread (coordinator / master). Do not implement product UI in the first message before loading skills.

---

You are the **master coordinator** for the personal-agent **Benchmark UI redesign**. You do **not** implement all 12 tasks yourself in one blob. You run **subagent-driven development** until the plan is fully done, every task has **consulting-grok-review PASS**, and the branch is **pushed**.

## Bootstrap (mandatory, in order)

1. Load skill `using-superpowers`, then `subagent-driven-development`, then `consulting-grok-review`, then `verification-before-completion`. For UI tasks also load `frontend-ui-craft` and `test-driven-development`.
2. Read and obey:
   - `AGENTS.md` (standing rules — especially Grok spawn, review gate, ship=push, Node 22, dist cache-bust)
   - `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md`
   - `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign.md` (**Tasks 1–12**)
   - `docs/superpowers/HANDOFF-benchmark-ui-redesign.md`
3. Create board: `docs/superpowers/STATUS-benchmark-ui-redesign.md` with task list 1–12, status columns: `todo | implementing | review | PASS | complete`, and a **review ledger** section.
4. Create progress file for the gate script (e.g. `docs/superpowers/progress-benchmark-ui-redesign.md`) in the format expected by `.agents/skills/subagent-driven-development/scripts/check-review-gate` — every task must eventually show:
   - `Task N: consulting-grok-review PASS (thread T-…)`
   - `Task N: complete`
5. Work on a **feature branch** (e.g. `impl/benchmark-ui-redesign`), not loose unpushed main-only if the repo convention uses branches. Isolate workers with **git worktrees** under `.worktrees/` when using local `amp -m grok45 -x` so master merge stays clean.

## Hard rules (non-negotiable)

| Rule | Detail |
|------|--------|
| **Workers** | Only `amp -m grok45 --no-archive-after-execute -x '…'`. **Never** `-ox`. **Never** Amp Task/OpenAI/oracle as implementer or reviewer. |
| **One long Grok at a time** | Do not parallelize two heavy Grok workers; they timeout. |
| **Review every task** | After each task’s tests green: package diff → **new** Grok thread with `consulting-grok-review` / reviewer-prompt contract. Fix **Critical + Important** before merge. Ledger PASS **before** complete. |
| **No “tests green = reviewed”** | Forbidden. |
| **Gate script** | Before claiming the plan done: run `.agents/skills/subagent-driven-development/scripts/check-review-gate progress-benchmark-ui-redesign.md` (or your progress path) — must exit 0. |
| **Whole-branch review** | After Task 12 product work: one more consulting-grok-review on full branch diff before ship. |
| **Ship = push** | `git push`, confirm not `ahead of origin`. |
| **UI verify** | Node `>=22 <23` on PATH. After UI: `npm --prefix web run build`, cache-bust vibe-pass vs refs `claude.png` `claude-2.png` `grok.png` `grok-2.png` `amp.png` (repo root or `.amp/in/artifacts/`). |
| **Composer** | Never remount focused session composer; `SessionChat.focus.test.ts` must stay green. |
| **Memory tab** | Chrome only — no fake save success. |

## Per-task loop (repeat for Task 1 → 12)

For each task in plan order:

1. **Dispatch implementer** (Grok `-x` or worktree worker) with: task section pasted from the plan, file paths, TDD steps, “do not skip review.”
2. **Verify:** run the task’s test commands yourself (or require worker evidence). Node 22.
3. **Review:** new thread:
   ```bash
   amp -m grok45 --no-archive-after-execute -x '…'
   ```
   Load `consulting-grok-review`. Paste: task intent, file list, `git diff` range, test output summary. Require verdict: PASS / FAIL with Critical/Important/Suggestions.
4. If FAIL → fix (worker or master) → re-review until PASS.
5. **Ledger** in STATUS + progress file:
   - `Task N: consulting-grok-review PASS (thread T-xxxxxxxx)`
   - `Task N: complete`
6. **Merge** to integration branch (FF preferred; never force-push worker). Push branch periodically so orb loss does not wipe work.
7. **Next task** immediately when unblocked — do not stall.

## Task map (do not reorder without reason)

| N | Summary |
|---|--------|
| 1 | Shell nav density (`align-content: start`, pad 12×10) |
| 2 | `Modal.svelte` + `.modal` tokens |
| 3 | Projects/Vaults/VaultProjects creates → Modal |
| 4 | `ProjectRail` + hub/rail CSS tokens |
| 5 | Rewrite `ProjectHubPage` Claude stack + rail |
| 6 | `#/projects/:id/sessions` → hub |
| 7 | SessionChat `openPath` / embeddedInHub from rail |
| 8 | Dense bottom composer + assistant copy |
| 9 | Hub embeds SessionChat; Back restores start |
| 10 | Vault projects name-first list |
| 11 | `frontend-ui-craft` benchmark gate (may already be partially done on main — verify/complete) |
| 12 | Full test suite + dist + vibe-pass vs 5 refs |

## Done criteria (all required)

- [ ] All Tasks 1–12 marked complete in STATUS
- [ ] Progress file: every task has `consulting-grok-review PASS (thread …)` then `complete`
- [ ] `check-review-gate` exits 0
- [ ] Whole-branch consulting-grok-review PASS
- [ ] `export PATH="$HOME/.local/node-v22/bin:$PATH" && cd web && npm test` green
- [ ] `npm --prefix web run build` done; served UI cache-bust checked
- [ ] Vibe-pass evidence vs all five benchmark refs (structural)
- [ ] Branch **pushed**; `git status` not ahead of origin (or PR opened if that is the ship path)
- [ ] Final message: task ledger table + review thread IDs + push SHA

## If blocked

- Grok worker timeout / no file: continue on master for small wrap-up or `amp threads continue T-… -x` — do **not** triple-retry the same spawn.
- Missing Node 22: install or put `~/.local/node-v22/bin` first on PATH.
- App not running for vibe-pass: start it or mark **blocked** (never fake pass).

## Start now

1. Confirm git clean-ish; pull/rebase as needed; create branch + STATUS + progress file.  
2. Dispatch **Task 1** only.  
3. Run until Done criteria are all checked.  

You are responsible for finishing — not stopping after “plan read.”
