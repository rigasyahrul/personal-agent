# Every worker task must pass consulting-grok-review

**Date:** 2026-08-20  
**Tags:** review, grok45, sdd, master, session-focus

**Task:** User corrected session-focus master: every worker must pass consulting-grok-review; re-run all task reviews.

**Wrong / mistakes:**
1. Master treated implementer self-report + green tests as enough to FF-merge Tasks 1–11 — skipped per-task and whole-branch consulting-grok-review.
2. Claimed “implementation complete” without the high-stakes review gate AGENTS already pointed at for workers.

**What worked:**
1. Codified hard standing rule in AGENTS.md.
2. Re-ran consulting-grok-review for each task range (1–11) + whole branch via `amp -m grok45 --no-archive-after-execute -x` with reviewer-prompt contract.
3. All 11 tasks + branch: Critical none, Important none (PASS). Minor polish only (message scroll unmount on file tab, etc.).

**Rule (next agent):** After each implementer finishes a plan task, **before** merge into the feature branch: new grok45 consulting-grok-review on that task’s BASE..HEAD. Fix Critical/Important (or re-review fixes). Tests green ≠ reviewed. Whole-branch review still required before ship.

**Codified into:**
- `AGENTS.md` standing rule (mandatory per worker task + `check-review-gate`)
- `.agents/skills/subagent-driven-development/SKILL.md` (HARD GATE, ledger order, rationalizations)
- `.agents/skills/subagent-driven-development/scripts/check-review-gate` (executable ledger proof)
- `.agents/skills/consulting-grok-review/SKILL.md` (personal-agent override: every SDD task)
- `.agents/skills/verification-before-completion/SKILL.md` (task complete ≠ tests green)
- This entry


**Evidence:** Master https://ampcode.com/threads/T-01a01f00-30b2-748a-8a68-c625574d5a35 ; branch review T-01a01fb2-d9e7-7718-a7bb-b584d06b6786 ; feature `impl/session-focus-layout`.
