# UI craft tokens + Go dist cache-bust

**Date:** 2026-08-20  
**Tags:** frontend, ui, web, craft, dist, orb

**Task:** Polish Personal Agent UI (shell → full surface) so it feels human; ship + compound.

**Wrong / mistakes:**
- Treating shell/home polish as “UI done” while catalogs/review/settings/sessions still used indigo scaffold soup.
- Vibe-passing Go-served `:8080` without rebuilding `web/dist` or cache-busting — browser kept old hashed JS/CSS and falsely reported “Global desk” / indigo still present.
- Hardcoding noisy health copy instead of `/health` status keys.

**What worked:**
- Shared primitives in `web/src/app.css` (`btn--*`, `panel`, `field-*`, `entity-card`, `metric-card`, …) then restyle every route to consume them.
- After UI edits: `npm --prefix web run build` → confirm `index.html` asset hashes → `curl` for new class strings → open `?v=<ts>#/route`.
- Craft-scaffold regression in `web/src/styles-baseline.test.ts` bans `bg-indigo-600`, `Global desk`, bullet glyphs, unavailable health string.
- Plan: `docs/superpowers/plans/2026-08-20-ui-full-surface-craft.md`.

**Rule (next agent):**
1. Load `frontend-ui-craft`; extend tokens before one-off Tailwind indigo.
2. Full-surface audit or explicit scope — shell ≠ all screens.
3. Go/`amp orb` serves **dist**, not Vite HMR; rebuild + cache-bust before claiming browser green.

**Codified into:**
- `AGENTS.md` (UI tokens first; UI fix ≠ served fix / dist)
- `.agents/skills/frontend-ui-craft/SKILL.md` + `reference/craft.md`
- `web/src/styles-baseline.test.ts` (scaffold ban + primitives)
- Plan under `docs/superpowers/plans/`

**Evidence:** commits `a7901f8`, `2b3db08`; thread `T-01a01e80-5d76-7230-837c-f882b932888a`
