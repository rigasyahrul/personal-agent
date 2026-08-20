# Grok plan drafts: one at a time, no triple-retry

**Date:** 2026-08-21  
**Tags:** grok45, amp, plans, writing-plans, spawn

**Task:** writing-plans for benchmark UI via user-requested Grok workers (`amp -m grok45 -x`).

**Wrong / mistakes:**
- Amp **Task** tool is **not** Grok — user caught misuse immediately.
- Long Grok draft workers can stream-timeout with **no file on disk** after minutes.
- Sitting on `shell_command_status` without checking file existence feels “stuck.”
- Triple-retrying the same timed-out draft wastes the session.

**What worked:**
- Explicit: only `amp -m grok45 --no-archive-after-execute -x` (never `-ox`, never Task-as-Grok).
- One long Grok at a time (standing rule).
- After first timeout / ~3–4 min no file: master writes remaining phase drafts and assembles.
- Parallel prep of prompt files on disk while one worker runs.
- A-shell + B1 from Grok; B2–B5 master-written; assemble header + drafts → one plan.

**Rule (next agent):**
For multi-phase plan drafts prefer Grok `-x` when asked; never call Task “Grok.” One worker; if no output file promptly, finish drafts on master — do not triple-retry.

**Codified into:**
- `AGENTS.md` (Grok plan drafts bullet)
- Existing spawn rules (one Grok; never `-ox`)

**Evidence:** thread T-01a01feb…; plan `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign.md`
