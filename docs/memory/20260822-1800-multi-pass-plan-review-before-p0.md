# Multi-pass consulting-grok-review before P0

**Date:** 2026-08-22  
**Tags:** grok, review, plans, writing-plans, master, knowledge

**Task:** Lock Obsidian memory + knowledge design/plan (P0–P3) via repeated consulting-grok-review before any product code.

**Wrong / mistakes:**
1. Single “looks good” self-review would have shipped dual path namespaces wrong (`notes` source-relative vs knowledge scope-root), stock `fsroot`+`ValidateRelPath` rejecting `memory/**`, 001-only migrator, and Runner workspace-only tool dispatch (default `workspace_files:false`).
2. Grok plan-draft worker stream-timeout with no file — triple-retry wastes time; master must finish drafts (already a standing rule).
3. Stopping after first “Safe P0” missed P1–P3 E2E Critical (tool dispatch).

**What worked:**
1. Multi-pass gates: design lock → fix → re-review → P0 confirm → P1–P3 E2E → fix → confirm → FINAL GATE.
2. Absorb every Important into **Canonical contracts** + conflicting task lines; reassemble plan; confirm with new Grok thread.
3. Final gate T-01a02a5c: Safe with Minor; minor polish into task checkboxes; master handoff + STATUS + PROMPT for later manual run.

**Rule (next agent):** Before implementing a multi-phase plan, run consulting-grok-review until Safe (no Critical/Important). Include at least: package lock review, phase-slice E2E if multi-phase, and a FINAL GATE. Canonical must name live collisions (paths, migrator, runner tools). Execution master: every **task** still gets its own consulting-grok-review before merge.

**Codified into:**
- `AGENTS.md` (design/plan lock before code; Obsidian memory execution handoff)
- `docs/superpowers/HANDOFF-obsidian-memory-knowledge-master.md`
- `docs/superpowers/STATUS-obsidian-memory-knowledge.md`
- `docs/superpowers/PROMPT-obsidian-memory-knowledge-master.md`
- Plan Canonical in `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md`

**Evidence:** Threads T-01a02a38, T-01a02a3d, T-01a02a41, T-01a02a48, T-01a02a53, T-01a02a59, T-01a02a5c; this Amp thread.
