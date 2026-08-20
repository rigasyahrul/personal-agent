# Design: Benchmark UI redesign (Amp / Grok / Claude fidelity)

**Status:** Approved for planning (user sign-off 2026-08-20)  
**Date:** 2026-08-20  
**Stack:** Existing Svelte 5 + TypeScript + Vite + Tailwind SPA under `web/`  
**Backend:** No product-feature expansion required for v1 chrome; Memory/soul persist API is explicitly later  

**Visual refs (committed artifacts):**  
`.amp/in/artifacts/claude.png`, `claude-2.png`, `grok.png`, `grok-2.png`, `amp.png`

**Related:**  
- `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md` (shell IA baseline)
- `docs/superpowers/specs/2026-08-20-session-focus-layout-design.md` (superseded/extended for session focus by this doc)
- `frontend-ui-craft` skill — must gain benchmark-fidelity gate
- Standing rule: polled SPA must never remount focused composer

---

## 1. Why this redesign (failure analysis)

The prior craft pass improved shared tokens and checklist compliance, but it did not achieve screenshot fidelity. Its process rewarded use of approved classes and the absence of generic scaffold patterns without requiring the rendered hierarchy, density, and interaction model to match the Claude, Grok, and Amp benchmarks.

That gap produced concrete failures:

- A live shell bug made the navigation unusably sparse: `.sidebar nav { flex: 1; display: grid }` allowed grid rows to stretch across the available sidebar height, producing rows approximately 147px tall.
- Create flows remained inline forms, creating form soup instead of focused modal interactions.
- The project hub used metric and destination cards, leaving a large visual desert instead of Claude's prompt-first start surface with a session list beneath it.
- The right rail did not reproduce Grok's persistent multi-menu structure, including its Memory and Files modes.
- The open-session workspace lacked Amp-grade tabs, file preview/source treatment, and a dense bottom composer.

This redesign therefore treats benchmark fidelity as a verifiable product requirement, not an aesthetic suggestion.

## 2. Goals and non-goals

### Goals

1. Ship Phase A immediately to restore compact, predictable shell navigation density.
2. Make the project hub follow the Claude project pattern: prompt and composer at the top, session rows below, and a persistent right rail.
3. Make vault projects follow the name-first list hierarchy shown in `claude-2.png`.
4. Make an open session follow Amp's workspace model: Agent and file tabs, Preview/Source modes, a dense bottom composer, and copy controls on assistant responses.
5. Make the right rail follow Grok's model: open by default with `Memory | Files` header tabs.
6. Move every create-project and create-vault flow into a modal.
7. Make a benchmark comparison vibe-pass a hard completion gate for each redesigned surface.
8. Define or extend shared tokens in `web/src/app.css` before introducing surface-specific styling.

### Non-goals

- Cloning Claude across the whole application or globally restyling Home in this pass.
- Implementing the Memory/soul persistence API in v1; only its chrome is in scope.
- Automatically collapsing the application left navigation.
- Adding Starred/Latest tabs or Claude's artifacts panel.
- Adding dark mode.
- Showing fake Memory “saved” success without a persistence API.

## 3. Approach

Three approaches were considered:

1. **Surface pack (chosen):** redesign only the benchmark-critical shell, project hub, vault project list, session workspace, right rail, and create modals. This provides high visual fidelity without destabilizing unrelated routes.
2. **Whole-app benchmark clone:** maximizes visual consistency but expands scope, risks erasing the existing product IA, and delays correction of the live shell defect.
3. **CSS-only correction:** is fastest but cannot fix structural mismatches such as inline creates, prompt/session ordering, modal behavior, tabs, or the multi-mode rail.

The approved choice is **Path C: Phase A, then Phase B**. Phase A ships the shell density correction first. Phase B delivers the bounded surface pack while preserving the existing global IA and backend contracts.

## 4. Phase A — Shell density (ship first)

### Bug

The shell's excessive spacing is a layout defect, not intentional padding. The selector `.sidebar nav { flex: 1; display: grid }` gives the navigation the remaining sidebar height and permits implicit grid rows to distribute through that space. With few items, each row expands to approximately 147px.

Phase A must establish these constraints:

- Navigation rows have a compact 36–40px minimum height and never stretch vertically.
- The navigation container must not grow or distribute its children to fill the sidebar. Use start-aligned grid content or a column flex layout with start-packed children.
- Expanded sidebar width remains within 220–240px, with approximately 12px vertical and 10px horizontal inner padding.
- Brand treatment is compact and does not create a second header-sized block.
- The vault context chip is tight, legible, and subordinate to primary navigation.
- The named collapse control remains anchored at the bottom of the sidebar.

### Acceptance A

- At a 1440×900 viewport, every rendered navigation item measures **44px high or less**.
- No navigation gap or row visually expands to consume the unused height of the sidebar column.
