---
name: frontend-ui-craft
description: >
  Use when touching any frontend or UI work — screens, components, styles, layout,
  spacing, empty states, visual polish, looks AI/generic, or UI bugs — before
  implementing visible changes or before claiming UI done.
---

# Frontend UI Craft

## Overview

**Code that compiles is not a UI.** A human-feeling UI is specified, built cheaply, seen in a real browser, and cleared of craft red flags before "done."

This skill is the agent loop for frontend work: screen spec → build → vibe-pass → craft gate → tests. It does not replace product design specs or `brainstorming` for new behavior.

**Violating the letter of these rules is violating the spirit.**

## When to use

- Any change that can affect what users see (markup, CSS, components, layout, copy on screen, empty/loading/error states)
- Visual polish, "feels AI/generic", UI bugs, restyles
- Before claiming UI fixed, polished, or shipped

**Skip browser vibe-pass only when** the change cannot affect rendered UI (e.g. API client types only). State the skip reason. When in doubt, do not skip.

## Mandatory loop

For every visible UI change:

1. **Screen spec (short)** — goal, surfaces, states (empty / loading / error / populated), primary action, out of scope. Freeze it.
2. **Build cheaply** — reuse tokens/components. Tangled or generic-soup draft → regenerate from a sharper spec; do not nurse AI markup forever.
3. **Browser vibe-pass (hard when possible)**
   - **Reachable:** open the real URL for the changed surface; a11y snapshot and/or screenshot; drive it like a naive user (primary actions, empty/error paths when relevant). Prefer Chrome DevTools MCP/CLI when configured.
   - **Not reachable:** say **blocked**; start the app or ask the user. **Do not** claim visual done.
4. **Craft gate** — run [Red flags](#red-flags--stop) and [Positive recipe](#positive-recipe). Unwaived red flag ⇒ not done.
5. **Spec-as-test** — add/update at least one automated check for what you changed (component/unit preferred; interaction test when focus/keyboard/multi-step is the risk).
6. **Done means** — automated gates green **and** vibe-pass evidence (URL opened + what you checked) **and** no silent red flags.

Load **`reference/craft.md`** when polishing, when red flags fire, or when hierarchy/density/tokens need detail.

## Red flags — STOP

Do not ship or claim done while these remain unless the user **explicitly** waives:

| Red flag | Why |
|----------|-----|
| Bullet / `•` / emoji-as-icon nav | Scaffold chrome |
| Orphan controls (lone `‹`, unlabeled icon-only) | Unfinished shell |
| Default stack soup (indigo + slate + `rounded-xl` + Inter everywhere, no hierarchy) | AI dashboard demo |
| Huge empty canvas; content stuck top-left | No density intent |
| Metric cards ≡ action/nav cards (same weight) | Weak hierarchy |
| Disabled nav with no explanation | Broken mental model |
| "Done" without opening real UI when reachable | Guessing |
| Poll/timer full re-render or `innerHTML` killing focused inputs | Hostile UX |
| User named/supplied **benchmark screenshots** but agent only checked tokens/classes/tests | Tokens ≠ fidelity — structural match required |
| Claimed vibe-pass without **side-by-side** vs each named ref | Guessing against screenshots |
| Nav rows stretched to fill sidebar (`flex:1` grid without `align-content: start`) | Broken density (~147px rows) |
| Create flows as inline `form-inline` soup when product expects **modals** | AISLOP forms |

## Positive recipe

1. **Hierarchy** — one clear H1; eyebrows only if they earn space; one obvious primary action.
2. **Density** — medium dashboard; constrain width/grid; avoid desert empty. Nav rows ~36–40px, never stretched.
3. **Surfaces** — canvas vs panel vs sidebar from tokens; border + space, not heavy shadow/gradient/glass.
4. **Actions** — primary / secondary / ghost / destructive look different on purpose.
5. **Nav** — real icons (or icon components), clear current state, named collapse control.
6. **States** — empty + skeleton/loading + inline error on surfaces you touch.
7. **Focus** — visible focus rings; never steal focus on poll.
8. **Copy** — plain product language; cut filler labels.
9. **Benchmark fidelity** — if the user names or attaches reference screenshots, freeze a short fidelity table (region → must-match structure) and vibe-pass **side-by-side** against every named ref before done. Completion report lists each ref checked + intentional deviations.

## Rationalizations — not allowed

| Excuse | Reality |
|--------|---------|
| "I read the Svelte/CSS; it looks fine" | Code ≠ pixels. Open the browser. |
| "App isn't running so I'll skip" | Blocked ≠ passed. |
| "Just a small CSS tweak" | Small tweaks accumulate slop. Snapshot the surface. |
| "Craft is subjective" | Red flags block ship; waive only with user OK. |
| "Tests passed" | Green units ≠ hierarchy or chrome. |
| "I'll polish in a follow-up" | Gate now or record an explicit waiver. |

## Done checklist

- [ ] Short screen spec frozen (or explicitly tiny one-liner for trivial tweak)
- [ ] Browser vibe-pass done **or** explicit blocked (not silently skipped)
- [ ] Red flags clear or user-waived
- [ ] Automated check updated when behavior/contract changed
- [ ] Completion claim includes evidence (URL + what was checked)

## Personal-agent appendix

- **Stack:** Svelte 5 + TS + Vite + Tailwind; tokens/chrome in `web/src/app.css`
- **Token map (prefer these):** `btn btn--primary|secondary|ghost|danger`, `page-header` / `page-stack`, `metric-card` vs `destination-card` / `entity-card`, `panel` / `panel--dashed`, `field-input|select|textarea`, `form-stack` / `form-inline`, `scope-chip`, `list-panel` / `list-row`, `link-accent`, `alert alert--error|warn`, `catalog-grid`, `badge-chip`, `tree-item` / `tree-item--active`, `modal`, `project-workspace` / `rail-tab` / `hub-start` / `name-row`, `session-composer` / `message-copy`. Extend `app.css` before inventing new one-offs.
- **Routes:** hash router; shell = global desk vs vault context. Global pages: no “Global desk” eyebrow. Vault pages: vault name eyebrow OK.
- **Sessions:** poll **patches in place** — never replace a focused composer (`AGENTS.md`). SessionChat polish = class/chrome only; keep composer form ancestry stable. Assistant replies get a focusable **copy** control when on the benchmark session surface.
- **Health pill:** pass status keys (`unknown` / `ready` / `error`) from `/health`; never hardcode “Storage status unavailable”.
- **Localhost / orb serve path:** process on `:8080` is Go → **`web/dist`**. After edits: rebuild dist, match asset hashes in `index.html`, `curl` for new tokens, vibe-pass with `?v=<timestamp>#/route` (browser caches old JS/CSS).
- **Docker:** `make docker-dev` for live API+web; prod compose stays image-baked
- **Web tests:** Node `>=22 <23` on `PATH` before `make web-test` (orb may only have Node 20 — install Node 22 under `~/.local/node-v22` if needed)
- **Browser:** `agent-browser` against localhost or portal; Chrome DevTools at `http://localhost:9222` when configured. A11y snapshot is enough for craft gates; screenshots may land under temp dirs — copy to `.amp/in/artifacts/` if durable. Prefer **local ref files** the user dropped in-repo over `read_thread` for attachments.
- **Full-surface rule:** polishing shell/home does **not** finish catalogs/review/settings/sessions/notes. Audit every route or explicitly scope the task.
- **Product look intent:** clean dashboard, light-first, Inter — `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md` (shell IA). **Benchmark fidelity redesign** (Claude hub / Grok rail / Amp session): `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md` + plan `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign.md`. Session-focus design is superseded where they conflict (rail default-open, copy, hub structure).
- **Plan for multi-screen polish (older token pass):** `docs/superpowers/plans/2026-08-20-ui-full-surface-craft.md` — do not treat token polish as benchmark fidelity.

## Related

- Design for this skill: `docs/superpowers/specs/2026-08-20-frontend-ui-craft-skill-design.md`
- Craft detail: `reference/craft.md`
- Also use: `brainstorming` (new behavior), `test-driven-development`, `verification-before-completion`
- Patterns: Visual AI, Rich Feedback Loops, Spec-as-Test, Disposable Scaffolding (agentic-patterns / coding-standards Frontend domain)
