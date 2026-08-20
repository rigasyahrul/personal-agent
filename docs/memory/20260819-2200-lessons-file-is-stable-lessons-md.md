# Lessons file is stable `lessons.md`

**Date:** 2026-08-19  
**Tags:** memory, docs, agents, compounding


**Task:** Make project memory scannable for AI agents; stop the filename date (`2026-08-12`) implying stale or per-day files. Session wrap via `compounding-engineering`.

**Wrong / mistakes:**
- Named the diary after the **creation** day while content kept growing across dates — agents misread freshness or opened/created a new dated file.
- Hardcoded `docs/memory/2026-08-12-lessons.md` in skills/AGENTS/handoffs.
- Template sat mid-file; order was oldest-first with no topic index.
- Assumed “every session updates the lessons file” improves output — only **standing rules** should load every session; the diary is on-demand evidence.
- Risk on wrap: re-prepending a second section for work that already wrote its lesson mid-session.

**What worked:**
1. Stable path `docs/memory/lessons.md`.
2. Newest-first sections + topic **Index** + fixed field template.
3. Hot path = `AGENTS.md`; this file = detail when compounding/synthesizing.
4. **Wrap hygiene:** when the lesson + AGENTS/skill pointers already landed in the same session, wrap = verify codification + attach commit/thread evidence; do **not** duplicate the `###` section.

**Rule (next agent):** Always use **`docs/memory/lessons.md`**. Prepend new lessons (newest first); refresh the Index row for touched topics. Do not create `YYYY-MM-DD-lessons.md` per session. Promote always-on rules to `AGENTS.md`; keep stories here. On “wrap / compound this session”: if a matching lesson already exists from the same work, **update that section** (evidence, supersession) instead of adding another.

**Codified into:**
- `docs/memory/20260819-2200-lessons-file-is-stable-lessons-md.md` (historical; layout now index + entry files)
- `AGENTS.md` (memory layers + stable-path standing bullet)
- `.agents/skills/compounding-engineering/SKILL.md` (stable path, prepend, Index, Tags template)
- `.agents/skills/synthesize-memory/SKILL.md` (primary path + newest-first reports)
- `docs/superpowers/HANDOFF-master-execution.md`, `docs/superpowers/README.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a0199e-6410-72f7-a6e9-1204cc4cad07 ; commit `4aa0ea2` (restructure + ship); wrap update this section 2026-08-19

---

**Superseded (layout), 2026-08-20:** Full bodies no longer live in `lessons.md`. That file is the **index only**; detail is `docs/memory/YYYYMMDD-HHmm-<slug>.md`. The stable *index* path and selective/on-demand rules still hold.

