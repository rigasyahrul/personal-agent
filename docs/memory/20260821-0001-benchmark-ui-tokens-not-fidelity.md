# Benchmark UI: tokens ≠ fidelity

**Date:** 2026-08-21  
**Tags:** craft, frontend, ui, benchmark, aislope

**Task:** User rejected UI after “craft” pass; demanded Claude/Grok/Amp screenshot fidelity.

**Wrong / mistakes:**
- Prior craft optimized shared CSS tokens + checklist red flags (no bullets, no indigo) while structure stayed generic: stretched nav (~147px from `.sidebar nav { flex:1; display:grid }` without `align-content: start`), metric+destination hub desert, inline create forms, no Grok rail, weak Amp session chrome.
- Agents treated green unit tests + token classes as “done.”
- Attachment hunt via `read_thread` wasted time; user put refs at repo root (`claude.png`, `grok.png`, `amp.png`, …).

**What worked:**
- Live measure + screenshots of real `:8080` UI.
- Brainstorm against named refs; lock Path C (shell fix first, then surface pack).
- Spec + 12-task plan with canonical contracts.
- Codify into AGENTS + `frontend-ui-craft` benchmark gate.

**Rule (next agent):**
If the user names screenshot refs, freeze a fidelity table and side-by-side vibe-pass each ref. Tokens alone = AISLOP. Prefer local ref files over thread attachment archaeology.

**Codified into:**
- `AGENTS.md` (Benchmark UI ≠ tokens; sidebar density; creates use modals)
- `.agents/skills/frontend-ui-craft/SKILL.md` + `reference/craft.md`
- Spec/plan under `docs/superpowers/`

**Evidence:** thread T-01a01feb-df87-76bf-9b8c-a42ee477e118; commits `9f557b6`, `796cce8`
