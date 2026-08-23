# CSS grid + display:none collapses the wrong track

**Date:** 2026-08-21  
**Tags:** frontend, css, rail, jsdom, craft, hub

**Task:** Project rail icon chrome — expanded mode must full-bleed the active panel over main.

**Wrong / mistakes:**
- Plan/CSS used `.project-workspace[data-rail="expanded"] { grid-template-columns: 0 minmax(0, 1fr); }` plus `.project-workspace__main { display: none; }`.
- With main out of the grid, the rail became the **only** item and auto-placed on **track 1 (`0`)** → ~1px rail, empty canvas.
- Unit tests that only checked `data-rail` / `display:none` / DOM flags still passed; vibe-pass caught full-width failure.
- jsdom often ignores pure CSS hide for `toBeVisible` — needed `hidden={railMode === 'expanded'}` dual-channel without weakening assertions.

**What worked:**
- Single column when expanded: `grid-template-columns: minmax(0, 1fr)` so the remaining rail child fills the workspace.
- Baseline test: assert single-column expanded **and** forbid `0 minmax(0, 1fr)`.
- Keep main `display: none` + `hidden` for a11y/jsdom; do not use multi-track + hide-one-item.
- Grok-only SDD + consulting-grok-review after Task 7 vibe found the Important; fix round before merge.

**Rule (next agent):**
1. Never hide a grid sibling with `display: none` while relying on a **zero-width track** to “reserve” it — that track is what the remaining child inherits.
2. Full-bleed one pane → **one** `minmax(0, 1fr)` track (or keep both items in-flow).
3. Layout mode tests: dual-channel CSS + `hidden`/attr; assert contracts in `styles-baseline` / hub tests; vibe-pass when screenshots named.
4. Green unit suite ≠ expanded visual fidelity.

**Codified into:**
- `AGENTS.md` standing rules (grid/`display:none`, jsdom dual-channel, project rail chrome pointer)
- `web/src/app.css` expanded single-column
- `web/src/styles-baseline.test.ts` expanded contract + ban old two-track
- `web/src/routes/ProjectHubPage.svelte` `hidden={railMode === 'expanded'}`
- Spec/plan: `docs/superpowers/specs/2026-08-21-project-rail-icon-chrome-design.md`, `docs/superpowers/plans/2026-08-21-project-rail-icon-chrome.md`

**Evidence:**
- Thread: https://ampcode.com/threads/T-01a0231c-f9d8-731e-aad9-27af0ac11035  
- Fix commit: `377f081`  
- Merge to main: `f97fc14` (local; push user-gated)  
- Re-review: T-01a0238f-435a-733b-beab-9e3d5b08928d  
