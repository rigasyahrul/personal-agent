# Design: Project right rail — icon chrome (Grok-style)

**Status:** Approved for planning (user sign-off 2026-08-21). **Config body superseded 2026-08-26:** persist SOUL / SYSTEM / AGENTS in the rail (see §5). Icon chrome / modes unchanged.  
**Date:** 2026-08-21  
**Stack:** Existing Svelte 5 + TypeScript + Vite + Tailwind SPA under `web/`  
**Backend:** Instruction GET/PUT APIs for Config. Files uses existing note/workspace APIs.

**Visual refs (repo root during design; prefer durable paths in plans):**

| Ref | File | Intent |
|-----|------|--------|
| Layout | `expected-right-sidebar-1.png` | Icon toolbar + panel (not text Memory \| Files tabs) |
| Expand | `expected-icon-right-sidebar-2.png` | Expand workspace covers main canvas |
| Files icon | `expected-icon-right-sidebar-3.png` | Files control + tooltip |
| Config icon | `expected-icon-right-sidebar-4.png` | Config control + tooltip |
| Collapse | `expected-icon-right-sidebar-1.png` | Collapse canvas show/hide rail |
| Files tree (later) | `expected-files-only-tree-right-sidebar-4.png`, `expected-files-tree-with-search-right-sidebar-4.png` | **Out of scope this pass** — content parity only |

**Related:**

- `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md` (rail was Memory \| Files text tabs; this supersedes rail chrome)
- `docs/superpowers/specs/2026-08-21-project-open-session-claude-chat-design.md` (hub main canvas)
- `frontend-ui-craft` — benchmark fidelity gate vs named refs above
- Standing rules: polled SPA focus; hub soft-fail; rail Files note routing

**Approach:** 1 — Restructure `ProjectRail` + hub layout modes (approved). Rejected: generic IconRailChrome extraction (YAGNI); CSS-only tab restyle (misses expand/collapse).

---

## 1. Why

Current project right rail uses labeled text tabs **Memory | Files**. Benchmark / user expectation is a **Grok-style icon toolbar**:

- Icons only, hover tooltips describing actions
- Config and Files as panel selectors
- Expand workspace (active panel full-bleed over main)
- Collapse canvas (hide rail content; slim restore control)

Also: product **Memory** does not belong in this rail. Project/docs memory lives under `/docs/memory` (and related product flows) later — not a freeform rail textarea.

---

## 2. Goals and non-goals

### Goals

1. Replace rail text tabs with an **icon-only toolbar** matching the approved layout and tooltip labels.
2. Three layout modes: **open**, **expanded**, **collapsed**.
3. Default on hub enter (no saved state): **open** + **Config** selected.
4. **Config** panel = persisted SOUL / SYSTEM / AGENTS (supersedes the original non-persistent helper copy).
5. **Files** panel = **keep existing** tree behavior and data sources (no tree/search redesign).
6. Persist last **mode** + **selected panel** in `localStorage`.
7. Spec-as-test + browser vibe-pass against named icon/layout refs before done.

### Non-goals

- Memory field or Memory persist API in the rail  
- Product `/docs/memory` behavior  
- Files tree redesign, basename polish, or search UI (refs 6–7 deferred)  
- Left app sidebar changes  
- Standalone session route rail outside project hub  
- Dark mode  

---

## 3. Surface and ownership

**Route:** `#/projects/:id` (hub and open session embedded in hub).

| Owner | Responsibility |
|-------|----------------|
| `ProjectHubPage` | Layout modes via `data-rail` on `.project-workspace`; show/hide main; wire collapse/expand/restore |
| `ProjectRail` | Icon bar, panel selection, Config + Files bodies, callbacks for expand/collapse |
| `app.css` | Tokens for icon bar, modes, widths |

Main canvas content (hub start, session chat) is unchanged by this design except width/visibility driven by rail mode.

---

## 4. Layout architecture

```text
┌─ App sidebar ─┬─ Main canvas ─────────────────────┬─ Right rail ──────────────┐
│ (unchanged)   │ hub start / session chat          │ ┌ icon bar ─────────────┐ │
│               │                                   │ │ ⚙  📁        ⛶  ⊟   │ │
│               │                                   │ └───────────────────────┘ │
│               │                                   │ panel: Config | Files     │
└───────────────┴───────────────────────────────────┴───────────────────────────┘
```

### Modes (`data-rail` on `.project-workspace`)

| Mode | Rail width | Visible chrome | Main canvas |
|------|------------|----------------|-------------|
| **open** | ~280–320px | Full icon bar + active panel | Normal flex beside rail |
| **expanded** | Fills **all** content width (covers main) | Full icon bar + active panel full-bleed | Hidden / zero width |
| **collapsed** | Slim ~44–48px | **Only** Collapse Canvas control (restore) | Full width |

**Expand** toggles **open ↔ expanded** and does **not** change the selected panel. Expanded surface = currently selected panel content only (user choice A).

**Collapse canvas** enters **collapsed**. In collapsed mode the single control restores **open** (last selected panel restored with mode).

### Icon bar (open and expanded only)

Horizontal bar at top of rail:

| Side | Control | Tooltip / `aria-label` | Action |
|------|---------|------------------------|--------|
| Left | **Config** | Config | Select Config panel |
| Left | **Files** | Files | Select Files panel |
| Right | **Expand workspace** | Expand workspace (or “Exit expanded” when expanded) | Toggle expanded ↔ open |
| Right | **Collapse canvas** | Collapse canvas | Enter collapsed |

Icons: inline SVG (settings/sliders, folder, expand arrows, panel). No emoji or bullet-as-icon.

Active panel control uses `.rail-icon--active` (or equivalent) + `aria-selected` / pressed state.

### Collapsed control

- Single icon button; width of rail adjusted to slim strip  
- Label when collapsed: **Show canvas** (or equivalent clear restore wording) via `title` + `aria-label`  
- Click → mode **open**

---

## 5. Panel contents

### Config (default tab)

**Superseded 2026-08-26** (user: put SOUL / SYSTEM / AGENTS in this rail, not a hub canvas card):

- `InstructionEditor` `variant="rail"` — tabs SOUL / SYSTEM / AGENTS, persisted GET/PUT, Save  
- No page `panel` card inside the rail; no duplicate editor on the hub canvas  
- **No Memory field** / no lessons preview here — product memory is `/docs/memory` later  
- Do **not** restore the fake unsaved “Instructions (system)” textarea or “Not saved yet — persistence coming later.”  

### Files

**Unchanged** from current `ProjectRail` Files behavior:

- Load project notes via `listProjectNotes`; map to tree  
- When `sessionId` + `workspaceFilesEnabled`, merge workspace tree under Workspace group  
- File click → `onOpenFile` with project-note vs workspace meta (existing hub/session tab routing)  
- Directories not openable  
- Loading skeletons, error alert, empty copy as today  

Deferred (do not implement in this plan unless user re-opens scope): search toggle, filter field, tree visual restyle per files-only / files-with-search refs.

---

## 6. State and persistence

| Key (suggested) | Values | Default if missing |
|-----------------|--------|--------------------|
| e.g. `pa.projectRail.mode` | `open` \| `expanded` \| `collapsed` | `open` |
| e.g. `pa.projectRail.tab` | `config` \| `files` | `config` |

- Read on hub mount; write on mode/tab change  
- Invalid values → defaults  
- Search query (if ever added) need not persist  

Optional: scope keys per `projectId` only if multi-project simultaneous UX needs it; **v1 global keys are enough** (same as sidebar collapse pattern).

---

## 7. Tokens / CSS

Extend existing project workspace tokens in `web/src/app.css`; prefer shared classes over one-offs.

| Token / hook | Role |
|--------------|------|
| `.project-workspace[data-rail="open\|expanded\|collapsed"]` | Mode layout |
| `.project-workspace__main` | Hidden or zero flex basis when expanded |
| `.project-workspace__rail` | Width per mode; slim when collapsed |
| `.project-rail` | Column flex: icon bar + panel |
| `.rail-iconbar` | Flex row; space-between left/right clusters |
| `.rail-icon` / `.rail-icon--active` | Icon button hit target ~36–40px; focus ring |
| `.rail-panel` | Scrollable body |

Collapsed rail must not leave a large empty panel; only the restore control.

---

## 8. Accessibility

- Every icon control: `type="button"`, visible focus, `aria-label` + `title` aligned with ref tooltips  
- Config/Files: tab-like selection (`role="tab"` / `tablist` or equivalent pressed pattern) with `aria-controls` → panel  
- Panels: `role="tabpanel"` when using tabs pattern  
- Expand/Collapse: not tabs; plain buttons with pressed/expanded state where useful (`aria-expanded` on expand)  
- Keyboard: Tab to controls; Enter/Space activates  

---

## 9. Testing

### Automated

- Icon order: Config, Files (left cluster); Expand, Collapse (right cluster)  
- Tooltips/accessible names match labels above  
- Default: open + Config (Instructions present; Memory absent)  
- Expand: main not visible / rail expanded; panel unchanged  
- Collapse: slim rail, only restore control; restore → open + last tab  
- localStorage round-trip for mode + tab  
- Existing Files tests: notes open, workspace merge, grant gating still pass  
- Hub/layout tests updated for `data-rail` and no Memory tab  
- Baseline/token tests if new required classes are asserted elsewhere  

### Browser vibe-pass (hard gate)

Side-by-side against each named ref in the table (layout, four icon tooltips, expand, collapse). Files content = parity with current product, not refs 6–7.  
Serve path: rebuild `web/dist` when verifying via Go `:8080`; cache-bust query.

---

## 10. Implementation sketch (not a plan)

1. Tokens + `data-rail` layout in hub CSS  
2. `ProjectRail` icon bar + Config (Instructions only) + Files unchanged  
3. Hub wiring for modes + persistence helpers  
4. Tests green  
5. Dist build + vibe-pass vs refs  
6. Commit / review per project gates  

Detailed steps belong in `docs/superpowers/plans/` via writing-plans after this spec is accepted.

---

## 11. Fidelity table (craft)

| Region | Must match |
|--------|------------|
| Icon bar layout | Left: Config, Files · Right: Expand, Collapse |
| Tooltips | Config; Files; Expand workspace; Collapse canvas (and restore wording when collapsed) |
| Expand | Active panel covers project dashboard / chat |
| Collapse | Slim rail; only collapse/restore control; main full width |
| Config body | Instructions only; no Memory |
| Files body | Current tree behavior (search/tree polish later) |

Intentional deviations: Files visual/search refs deferred; product Memory not in rail.

---

## 12. Approval record

- Approach 1 chosen (2026-08-21)  
- Expand = active panel full-bleed (A)  
- Icon order: Config, Files left; Expand, Collapse right  
- Collapse = slim width, Collapse/restore control only  
- Default: open + Config  
- Remove Memory from rail; Config = Instructions only  
- Files keep existing (defer tree/search)  
- Full design approved to write spec (this document)  
