# Orb loss: unpushed commits vanish; restore from Amp threads

**Date:** 2026-08-20  
**Tags:** orb, git, handoff, amp, grok45, session-focus, restore

**Task:** Resume Session Focus Layout after orb recreate; prior master had 5+ local commits and Task 1 on a worker branch — none on `origin`.

**Wrong / mistakes:**
1. Design, plan, lock, handoff, and Task 1 (`session-prefs`) lived only on the old orb’s git objects (`main` ahead of origin, `w/session-focus-t1`). Fresh orb = shallow clone at `origin/main` only — SHAs invalid, worktrees gone.
2. Assuming handoff paths on disk without checking `git status -sb` / `git cat-file` first burned time on ENOENT.
3. Grok Task 11 vibe-pass worker hung after screenshots/uncommitted fixes; `amp threads continue` can error while the first `-x` is still live — master should finish small verify+commit wraps rather than stall.

**What worked:**
1. **Recover authority docs** from prior master + design/worker threads via `read_thread` (full plan/lock/handoff; full design from design worker; prefs impl from Task 1 worker).
2. **Re-seed feature branch** `impl/session-focus-layout` under `.worktrees/session-focus-layout`, restore Task 1 files, TDD-verify, then SDD Tasks 2→11 with **one** `amp -m grok45 --no-archive-after-execute -x` worker worktree each; FF-merge into feature after verify.
3. **Never `-ox` with grok45** (hard fail). Node 22 first on `PATH`. Keep `SessionChat.focus.test.ts` green; composer form stays mounted across file tabs.
4. **Promote kind-omit** found in vibe-pass: API file payloads may omit `kind` — gate + SessionFileTab normalize + tests.
5. Master completed Task 11 commit when worker stalled after evidence was already on disk (screenshots + diff).

**Rule (next agent):**
- Before archiving a multi-hour design/impl master: **push the feature branch** (or get explicit user OK to leave unpushed and accept loss).
- On resume after orb recreate: confirm git objects exist; if not, **thread-restore** then re-commit — do not invent a thinner plan.
- Grok SDD: one worker worktree per task; continue or master-wrap stuck finishes; no merge/push unless user asks when constrained.

**Codified into:**
- `AGENTS.md` standing rules (orb loss, grok45 no `-ox`, composer mounted, kind-omit, `.worktrees/` ignore)
- `web/src/lib/promote.test.ts` + `SessionFileTab.test.ts` (kind omit)
- `docs/superpowers/HANDOFF-session-focus-layout.md` (complete status)
- This entry (evidence only)

**Evidence:** Resume master https://ampcode.com/threads/T-01a01f00-30b2-748a-8a68-c625574d5a35 ; prior https://ampcode.com/threads/T-01a01eac-2694-7622-9396-d70e0086d08b ; feature `impl/session-focus-layout` @ tip after Task 11 (`c222cb6` harden + handoff). Screenshots `.amp/in/artifacts/session-focus-0*.png`.
