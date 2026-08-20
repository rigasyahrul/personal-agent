# consulting-grok-review via Grok thread, not Task/OpenAI

**Date:** 2026-08-19  
**Tags:** review, grok


**Task:** High-stakes Oracle-shaped review without ChatGPT / OpenAI Task.

**Wrong / mistakes:** `consulting-grok-review` skill said dispatch Task/subagent; Task tool routes through OpenAI and fails when ChatGPT is unsubscribed. Worker stuck or fell back to self-review.

**What worked:** New Amp thread in grok45 mode with filled `reviewer-prompt` contract; poll until `## Verdict`.

**Rule (next agent):** For consulting-grok-review on this project: spawn a **new Amp thread in grok45 mode** with `amp -m grok45 --no-archive-after-execute -x '<prompt>'` (`-ox` best-effort only — see 2026-08-20 Master Grok spawn), poll `amp threads markdown T-…` until `## Verdict`, then act on Critical/Important. Do **not** use the Task tool or built-in oracle. Do **not** treat self-review as the gate.

**Superseded dispatch detail:** mandatory `-ox` replaced by 2026-08-20 lesson (local `-x` + worktree isolation).

**Codified into:** `AGENTS.md` standing bullet; `.agents/skills/consulting-grok-review/`

**Evidence:** Phase 6 review thread https://ampcode.com/threads/T-01a01801-6408-7553-a0d4-58af60b7885d

**Supersedes:** 2026-08-19 — Worker high-stakes review = consulting-grok-review, not built-in oracle (Task dispatch path obsolete).

---
