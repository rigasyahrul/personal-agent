# Design: Benchmark UI redesign (Amp / Grok / Claude fidelity)

**Status:** Approved for planning (user sign-off 2026-08-20)  
**Date:** 2026-08-20  
**Stack:** Existing Svelte 5 + TypeScript + Vite + Tailwind SPA under `web/`  
**Backend:** No product-feature expansion required for v1 chrome; Memory/soul persist API is explicitly later  

**Visual refs (artifacts):**  
`.amp/in/artifacts/claude.png`, `claude-2.png`, `grok.png`, `grok-2.png`, `amp.png`  
(also present at repo root during design; prefer artifacts path in plans)

**Related:**  
- `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md` (shell IA baseline)  
- `docs/superpowers/specs/2026-08-20-session-focus-layout-design.md` (session parts extended/superseded here)  
- `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-lock.md` (decision lock)  
- `frontend-ui-craft` skill — must gain benchmark-fidelity gate  
- Standing rule: polled SPA must never remount focused composer  

**Path:** C — Phase A (shell density) first, then full surface pack (Approach 1).

---

## 1. Why this redesign (failure analysis)

The prior craft pass improved shared tokens and checklist compliance, but it did not achieve screenshot fidelity. Its process rewarded approved classes and the absence of generic scaffold patterns without requiring the rendered hierarchy, density, and interaction model to match the Claude, Grok, and Amp benchmarks the user supplied.

Concrete failures observed on live UI (`localhost:8080`, project hub / sessions / Test 1):

| Failure | Evidence |
|---------|----------|
| Nav rows ~147px tall | `.sidebar nav { flex: 1; display: grid }` stretched grid rows to fill sidebar height |
| Creates = inline form soup | Projects/Vaults/Sessions used `panel form-inline` / stacked forms, not modals |
| Project hub = desert cards | Metric strip + three destination cards; content ended ~y=356 on 900px viewport |
| No Claude start+list | Missing “How can I help you today?” with session rows **below** |
| No Grok multi-menu rail | Files only when grant on, default closed; no Memory \| Files header tabs |
| Session not Amp-grade | Tabs exist but composer is labeled form soup; Preview/file UX weak vs `amp.png` |
| Skill false done | Tokens + tests green while structure still AISLOP |

This redesign treats **benchmark fidelity** as a verifiable product requirement, not an aesthetic suggestion.

---

## 2. Goals and non-goals

### Goals

1. **Phase A immediately** — restore compact shell navigation density (nav rows 36–40px).
2. **Project hub** follows Claude: prompt + composer on top, session rows **below**, persistent right rail (no left session column, no New session button).
3. **Vault projects** follow `claude-2`: name-first list rows.
4. **Open session** follows Amp: Agent + file tabs, Preview/Source, sticky **bottom** composer, copy control on each assistant reply.
5. **Right rail** follows Grok: default **open**, header tabs **Memory \| Files**.
6. **Creates** (project, vault) use **modals**.
7. **Benchmark vibe-pass** is a hard completion gate per redesigned surface.
8. **Tokens first** in `web/src/app.css` before one-off surface styling.

### Non-goals

- Whole-app Claude clone or full global Home restyle in this pass  
- Memory/soul **persist API** in v1 (chrome only; no fake “saved”)  
- Auto-collapse of app left navigation  
- Starred/Latest tabs, Claude artifacts panel  
- Dark mode  
- New backend product features beyond what existing APIs already support for notes/source/workspace trees  

---

## 3. Approach

| # | Approach | Verdict |
|---|----------|---------|
| 1 | **Surface pack** — shell density + project hub + vault list + session + rail + modals | **Chosen** |
| 2 | Whole-app benchmark clone | Rejected — delays shell fix; expands IA fight |
| 3 | CSS-only polish | Rejected — cannot fix structure (user already rejected AISLOP) |

**Path C:** Phase A ships first; Phase B delivers the surface pack while preserving global IA (Home / Projects / Vaults / Review / Settings) and backend contracts.

---

## 4. Phase A — Shell density (ship first)

### Bug

`.sidebar nav { flex: 1; display: grid }` gives the nav the remaining sidebar height and lets implicit grid rows **stretch**, producing ~147px rows with few items.

### Constraints

- Nav rows: **36–40px** min-height; never stretch vertically  
- Nav container: start-packed (`align-content: start` or column flex without grow-distribution on items)  
- Expanded sidebar: **220–240px** width; ~**12×10** padding  
- Brand compact; vault context chip tight; collapse control stays bottom  

### Acceptance A

- At **1440×900**, every primary nav item height **≤ 44px**  
- No row/gap visually consumes unused sidebar height  

---

## 5. Project hub (Claude start + list)

**Route:** `#/projects/:id` (canonical)

### Layout

App left nav **unchanged**. Content area has **exactly two** containers:

1. **Main canvas** — start prompt, composer, session rows; becomes Amp session shell when a session is open  
2. **Right rail** — default **open**; tabs **Memory \| Files**

No left session-list column. No **New session** button. Fuller width than the global ~1120px canvas cap on hub/session surfaces.

```text
┌─ App sidebar ─┬─ Main canvas ────────────────────────────┬─ Right rail (default ON) ─┐
│ Home …        │ Project name          Notes · Review     │ [ Memory ] [ Files ]       │
│ (unchanged)   │                                          │                            │
│               │ How can I help you today?                │ Memory tab:                │
│               │ ┌──────────────────────────────────────┐ │  · Memory (field)          │
│               │ │ multi-line composer            Send  │ │  · Instructions (system)   │
│               │ └──────────────────────────────────────┘ │  (persist API later)       │
│               │ Your sessions                            │ Files tab:                 │
│               │ · title · provider:model · relative time │  Amp-style directory tree  │
│               │ · …                                      │                            │
└───────────────┴──────────────────────────────────────────┴────────────────────────────┘
```

### Behaviors

| Trigger | Result |
|---------|--------|
| Submit non-empty prompt | Create session (defaults) + first message → open Amp shell in main; rail stays |
| Empty/whitespace prompt | No request; validation; keep focus |
| Click session row | Open that session in main; rail stays |
| Notes / Review | Quiet header links only (not destination cards) |
| Legacy `#/projects/:id/sessions` | **Prefer replace-redirect** to `#/projects/:id` (or identical hub state) |

### Removed from hub

Metric strip; three destination cards; inline create-session form soup; separate “New session” CTA.

### Session rows

Directly **below** the prompt/composer block (Claude stack). Title primary; `provider:model` secondary; relative time only if API exposes a timestamp (omit if absent).

### Right rail

| Tab | Spec |
|-----|------|
| **Memory** | Grok-2-style labeled **Memory** + **Instructions (system)**. Chrome only in v1. No enabled save, toast, or “saved” claim without API. Editable-for-design must be labeled non-persistent, or read-only/disabled if clearer. |
| **Files** | Amp-style hierarchical tree (disclosure + indent). Notes/source via existing APIs. When active session has `workspace_files`, include workspace group. Click file → open/focus main file tab (when session open). |

### Empty / loading / error (hub)

- **No sessions:** keep full start area; under list heading: “No sessions yet. Send a message above to start one.”  
- **List error:** composer stays usable; inline retry in list region  
- **Create/send error:** keep draft; no fake success navigation  
- **Files empty/error:** local to Files tab  
- **Memory:** no fake load/save lifecycle  

### Out of scope (hub)

Memory persist API; model picker on every send (defaults; optional More options modal later); redesign of Notes/Review pages beyond header links.

---

## 6. Vault projects (`claude-2`)

**Route:** vault projects list

| Item | Spec |
|------|------|
| **Look** | Name-first rows (project name hero); not fat entity cards |
| **Meta** | At most one muted line, or meta on hover; chevron OK |
| **Header** | Vault eyebrow + **Projects** + **New project** → **modal** |
| **Click** | Enter project hub (§5) |
| **Empty** | Purposeful empty + primary opens same modal |

---

## 7. Open session (Amp main + Grok rail)

Entered from hub Send or session row click. Main becomes session surface; **right rail continues** (default open). App left nav unchanged (user collapse only).

### Main header

**Back** (→ hub) · title · model chip · run status.  
No duplicate Files control in header when rail is open (Files lives on the rail).

### Tabs (Amp)

- Always **Agent** (not closable)  
- File tabs: basename, close, tooltip = full path; max **8** file tabs; LRU eviction among file tabs; same path focuses existing tab  

### Agent tab body

| Element | Spec |
|---------|------|
| Messages | Scrollable region |
| User | End-aligned bubble |
| Assistant | Bare markdown on canvas (no “Assistant” label, no assistant bubble) |
| **Copy** | Small **focusable** copy control on **each** assistant response container; copies full plain text of that reply; brief **Copied** feedback |
| **Composer** | **Sticky bottom** of Agent pane: multi-line textarea + Send; dense Amp-style; **no** fat “Message” label form soup |
| Run active | Composer disabled (send) |
| Poll | Patch in place; **never remount** composer; draft/focus/selection survive |
| File tab active | Hide composer (do not destroy); restore on Agent |

### File tab body

**Preview \| Source** (Preview default); read-only; **Save to source** when promotable → existing `PromoteDialog`. Loading/errors local to tab.

Clicking a file in the rail Files tree opens/focuses that tab in main — rail is **not** a second preview pane.

### Open-session wireframe

```text
┌─ App shell ──────────────────────────────────────────────────────────────┐
│ Left nav │ Back · title · [model] · run                                  │
│          ├─ [Agent] [file.md ×] … ─────────────┬─ [Memory] [Files] ─────┤
│          │  scrollable messages                │  rail (default open)   │
│          │   assistant md …              [copy]│  Memory fields or      │
│          │                    ┌ user bubble ┐  │  Files tree            │
│          │                    └─────────────┘  │                        │
│          ├─ BOTTOM COMPOSER (sticky) ──────────┤                        │
│          │  multi-line draft              Send │                        │
└──────────┴─────────────────────────────────────┴────────────────────────┘
```

### Supersession vs session-focus design

This doc **extends** `2026-08-20-session-focus-layout-design.md` and **supersedes** it where they conflict:

| Topic | Prior session-focus | This redesign |
|-------|---------------------|---------------|
| Files UI | Default-closed toggle bar | **Default-open** continuous Grok rail (Memory \| Files) |
| Copy | Not specified | Per-assistant-response copy control |
| Bottom composer | Implied | **Explicit** benchmark gate |
| Compatible keep | Agent/file tabs, 8 LRU, Preview/Source, PromoteDialog, poll-safe composer, markdown | Same |

---

## 8. Modals

Shared modal primitive: backdrop, Esc, focus containment, focus return to opener, primary + secondary actions. Errors stay inside modal.

| Flow | UI |
|------|-----|
| New project (global / vault) | Modal (name); context supplies `vault_id` |
| New vault | Modal (name) |
| Session more options (optional) | Model + `workspace_files`; hub quick-start uses **defaults** without requiring modal |
| Promote | Existing `PromoteDialog` retained |

**No** inline expand-into-page create forms on catalogs or hub.

---

## 9. Delivery phases

| Phase | Deliverable |
|-------|-------------|
| **A — Shell** | Nav density fix + compact sidebar tokens; acceptance A |
| **B1 — Modals** | Shared modal; migrate New project / New vault |
| **B2 — Hub** | Prompt + composer + sessions below + default-open rail (Memory chrome + Files tree) |
| **B3 — Open session** | Amp tabs, assistant copy, sticky bottom composer, file Preview/Source, continuous rail |
| **B4 — Vault list** | `claude-2` name-first rows + modal create |
| **B5 — Vibe-pass + skill** | Side-by-side vs all five refs; update `frontend-ui-craft` benchmark gate |
| **Later** | Memory/soul read/write API wired to existing Memory tab fields |

---

## 10. Benchmark acceptance (hard done gate)

Not done because tokens exist or unit tests pass. Done only after browser vibe-pass vs named refs:

| Reference | Must show |
|-----------|-----------|
| Shell / density | Nav item height **≤ 44px** at 1440×900; no stretched nav column |
| `claude.png` | Project hub: **“How can I help you today?”** + composer on top; session rows **directly below**; no left session column; no metric/destination-card grid |
| `claude-2.png` | Vault projects: **name-first** rows, not fat cards; create via modal |
| `grok.png` / `grok-2.png` | Right rail **default open**; header **Memory \| Files**; Memory fields layout; Files = hierarchical tree (not second main panel) |
| `amp.png` | Open session: **Agent** + file tabs; file **Preview**; sticky **bottom** composer on Agent; assistant bare md + **copy** control |

### Checklist

- [ ] App left nav never auto-collapsed by hub/session entry  
- [ ] Hub screenshot: prompt then session rows (not detached create form / card grid)  
- [ ] Open-session screenshot: composer at **bottom** of Agent pane  
- [ ] Composer identity + draft survive poll and Agent/File tab switches  
- [ ] File-tab cap 8 + LRU  
- [ ] All New project / New vault entry points use modals  
- [ ] Assistant copy is keyboard-focusable with Copied feedback  
- [ ] Memory UI does not claim persistence  
- [ ] Material density/hierarchy gaps vs refs fixed before sign-off  

---

## 11. Skill / process change

Update **`frontend-ui-craft`** so that when the user supplies or names benchmark screenshots:

1. Spec must include **explicit fidelity criteria** per named ref  
2. Done requires **side-by-side vibe-pass** against every named ref  
3. Completion report lists each ref checked and any intentional deviation  
4. **Tokens alone ≠ benchmark match**  
5. Blocked screenshot comparison = blocked, never passed  

Optional: store durable refs under `.amp/in/artifacts/` (or `docs/` if user wants committed product refs — prefer gitignored artifacts unless user asks to commit PNGs).

---

## 12. Locked decisions summary

| ID | Decision |
|----|----------|
| PATH | C: A then B surface pack |
| HUB | 2 content panes (main + rail) + app nav; prompt top; sessions below; no New session btn; no left list |
| START | Composer Send = create session + first message |
| RAIL | Default open; Memory \| Files |
| MEMORY | Chrome now; API later; no fake save |
| FILES | Amp tree; click → main file tab when session open |
| VAULT | Name-first list (`claude-2`) |
| SESSION | Amp tabs + Preview; bottom composer; assistant copy |
| MODALS | Project + vault create |
| SUPERSEDE | Session-focus design where rail/copy/composer gates conflict |

---

## 13. Next step

User reviews this spec file. On approval → invoke **writing-plans** for implementation plan under `docs/superpowers/plans/`. No implementation until plan exists and is executed under TDD + craft vibe-pass gates.
