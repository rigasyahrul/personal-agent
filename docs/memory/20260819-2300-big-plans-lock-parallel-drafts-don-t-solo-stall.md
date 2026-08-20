# Big plans: lock + parallel drafts, don't solo-stall

**Date:** 2026-08-19  
**Tags:** plans, multi-agent, writing-plans, skills, ui


**Task:** Brainstorm + write implementation plan for Svelte UI redesign (context shell, vault UX, docker-dev HMR). Session wrap via compounding-engineering.

**Wrong / mistakes:**
1. After user approved the design, started **writing-plans solo** and went quiet for a long stretch gathering context / drafting in one thread — user had to interrupt (“why not subagent delivery?”). The multi-agent plan lesson already required **lock + phase drafts + assemble**; delay was process failure, not missing knowledge.
2. Amp **Skill tool** returned `compounding-engineering` “not found” even though the skill lives under `.agents/skills/` and is listed in `AGENTS.md`. Must **Read SKILL.md and continue**, not stop.
3. Early product talk used “unfiled” without defining it; user confusion until explained as **no vault** (`vault_id` empty). Backend already had vaults; UI gap looked like “missing Vault.”

**What worked:**
1. **Brainstorming hard gate** — design approved before any app code; decisions frozen (Svelte 5, context sidebar Global vs Vault, Home=dashboard, searchable grids, Inter, docker-dev instant UI).
2. After correction: **lock file** → **parallel Task drafts** (phases 01–06) → **one assembled plan** with **Canonical contracts** header that wins over draft drift → commit.
3. Spec path: `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md`. Plan: `docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md` (+ lock + `…-drafts/`).
4. Domain note for implementers: vault APIs exist (`GET/POST /api/v1/vaults`, project `vault_id` immutable); sessions remain **per-project**; vault Sessions/Review = client aggregate/filter.

**Rule (next agent):**
- When `writing-plans` scope is large (many phases / parallel-safe sections): **same turn** write the lock, dispatch **parallel draft agents**, then assemble. Do **not** solo-author a multi-thousand-line plan in silence.
- Canonical contracts section in the assembled plan beats conflicting task snippets.
- Skill tool miss → Read `.agents/skills/<name>/SKILL.md` (already standing).
- “Unfiled” = project with empty `vault_id`; say “no vault” in UI copy when clarity matters. Vault is first-class nav, not a buried optional select.

**Codified into:**
- `docs/memory/20260819-2300-big-plans-lock-parallel-drafts-don-t-solo-stall.md`
- `AGENTS.md` (standing bullet: big writing-plans → lock + parallel drafts immediately)
- `.agents/skills/writing-plans/SKILL.md` (large-plan parallel draft gate)
- Spec + plan artifacts under `docs/superpowers/` (already committed)

**Evidence:** Amp thread https://ampcode.com/threads/T-01a019d4-7a0b-76bb-a368-c98695f346f8 ; commits `20b62cd` (spec), `d3a2d3a` (assembled plan)

**Related:** supersedes delay mode of “Multi-agent plans need one authority” (2026-08-12) — authority still required; **also** require parallel draft delivery so the human is not left waiting.

---
