# Design: Session Focus Layout (tabs + files bar)

**Status:** Approved  
**Date:** 2026-08-20  
**Scope:** Session surface pack only — open-session focus layout, vault Sessions desk list restyle, light Project Sessions list alignment  
**Stack:** Existing Svelte 5 + TypeScript + Vite + Tailwind SPA under `web/`  
**Backend:** No product API expansion; hash routing unchanged; `workspace_files` grant behavior unchanged  

**Related:**
- `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md` — app shell stays; this extends **sessions only**
- `docs/superpowers/specs/2026-08-12-personal-agent-design.md` — domain / promote / workspace tools
- Standing rule: polled SPA must never remount/replace a focused composer
- Visual refs (artifacts): `.amp/in/artifacts/ref-amp-tabs.png` (main pane tabs), `.amp/in/artifacts/ref-grok-rightbar.png` (files right bar concept), `.amp/in/artifacts/ref-ss.png` (Claude-style chat list cards)

---

## 1. Goals and non-goals

### Goals

1. **Session focus layout** — when a session is open, give chat + workspace a denser, IDE-like surface: Amp-style main tabs + optional files right bar (tree only), not a nested preview column.
2. **Readable agent output** — assistant messages are bare markdown on the canvas (no bubble, no “Assistant” label), including Mermaid diagrams from ` ```mermaid ` fences.
3. **Preserve chat continuity** — switching file tabs must not destroy Agent draft/scroll; poll must never remount the focused composer.
4. **Vault Sessions desk** — restyle to card rows (Claude Projects “Your chats” language); keep desk-list IA (not a focus left rail).
5. **Light Project Sessions alignment** — same card language on the project sessions list when no session is open.
6. **Tokens first** — shared classes in `web/src/app.css`; no one-off indigo scaffold soup.

### Non-goals

- Global app shell redesign (left nav, top bar, Home / Projects / Vaults / Review / Settings layouts)
- Auto-collapse of app sidebar (collapse remains **user-controlled only**)
- Notes page redesign (only reuse tree/search patterns inside the files bar)
- New APIs, promote/review behavior changes, auth changes
- Dark mode
- Amp Changes / Portal / Terminal sidebar modes
- Edit-in-tab (preview/source are **read-only**)
- Focus-mode left session-history rail
- Multi-user Memory / Instructions rails on list pages

---

## 2. Decisions (locked)

| # | Decision | Choice |
|---|----------|--------|
| D1 | Approach | **Approach 1 — Session surface pack** (not whole-app shell) |
| D2 | App left nav while session open | **Unchanged**; no auto-collapse |
| D3 | Content width in focus | May use **fuller width** — relax ~1120px `.content-canvas` cap **for session focus only** |
| D4 | Main pane model | **Tabs** (Amp pattern): always-on **Agent** + optional file tabs |
| D5 | File open target | Click in files tree → open/focus **filename tab** in main pane (not inline preview in bar) |
| D6 | File tab body | **Preview** (default) / **Source** toggle; read-only |
| D7 | File tab cap | ~**8** file tabs; same path reuses tab; at cap close **oldest** file tab (LRU among file tabs) |
| D8 | Files UI | **Right bar**, toggle from session header; default **closed**; tree + search only |
| D9 | Files bar visibility gate | Same as today: only if session has workspace tools (`workspace_files`); else hide toggle + bar |
| D10 | Split when files open | Default **70% main / 30% files**; drag-resize; clamp main **~50–85%**; persist % |
| D11 | Narrow viewport | **< ~1024px**: stack; files as **drawer/sheet** when toggled |
| D12 | Chat assistant chrome | **No bubble**, **no “Assistant” label**; proper Markdown + Mermaid |
| D13 | User messages | Keep distinct end-aligned bubble; calmer than today if needed |
| D14 | Vault Sessions IA | Stay a **desk list** (option A), not a focus left rail |
| D15 | List visual language | **Card rows** (title + meta); shared component/classes for vault + project lists |
| D16 | Activity timestamp on cards | Show relative activity **only if** API already exposes a timestamp (`created_at` / `updated_at`); **omit** if absent — do not invent |
| D17 | Promote | “Save to source” from **file tab** when promotable; same `PromoteDialog` |
| D18 | APIs / routing / grants | No expansion; hash routing unchanged; `workspace_files` unchanged |
| D19 | Markdown stack | Add maintained renderer + Mermaid; sanitize HTML; recommend default below |
| D20 | Composer | Multi-line + primary Send; disabled while run active; **stable form ancestry** |

---

## 3. Session focus information architecture

While a session is open (project sessions route with active session):

```
┌─ App shell (unchanged) ─────────────────────────────────────────┐
│ Left nav │ Top bar                                              │
│          ├─ Content (session focus: fuller width allowed) ──────┤
│          │ ┌─ Session header ─────────────────────────────────┐ │
│          │ │ Back · Title · provider/model · run · Files ▢   │ │
│          │ │ [Operation badges / alerts under header]         │ │
│          │ ├─ Main pane (tabs) ──────────┬─ Files bar (opt) ──┤ │
│          │ │ [Agent] [file.md ✕] […]     │ Search             │ │
│          │ │                             │ hierarchical tree  │ │
│          │ │  tab body:                  │ (notes-like)       │ │
│          │ │  Agent = messages+composer  │                    │ │
│          │ │  File  = Preview|Source     │                    │ │
│          │ │         + Save to source?   │                    │ │
│          │ └─────────────────────────────┴────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

**Width:** For the open-session surface only, allow the session layout to use available content width (relax the global `min(100%, 1120px)` canvas cap on this route/state). Other routes keep the existing canvas measure.

**Sidebar:** App left nav stays as the user left it. Do not auto-collapse on session open.

---

## 4. Tabs + files bar behavior

### 4.1 Session header

Left → right (wrap allowed on small widths):

| Control | Behavior |
|---------|----------|
| Back | “Sessions” (or equivalent) → return to project sessions list; existing `onclose` |
| Title | `session.title` |
| Provider/model chip | `provider:model_id` badge (existing chip language) |
| Run status | Subtle status text / live region (existing run label), not a loud banner |
| Files toggle | “Show files” / “Hide files” — **only if** workspace tools enabled for session |
| Op badges | Under header row, as today (`OperationBadges`) |
| Alerts | Setup/grant/poll/send errors under header / above tab body as today |

### 4.2 Main pane tabs

**Agent tab (always present)**
- Label: **Agent**
- Not closable
- Body: full-height chat (see §5): badges already in header stack; flex message scroller; sticky composer
- Selecting Agent does not unload file tab state (keep open file tabs and their loaded content/mode)

**File tabs**
- Opened by activating a **file** node in the files tree (directories expand/collapse only; do not open tabs)
- Label: file name (basename); tooltip or `title` attribute = full path
- Close control (✕) on each file tab
- Activating an already-open path **focuses** that tab (no duplicate)
- Cap: **8** file tabs. If opening a 9th distinct path, close the **least-recently-activated** file tab (Agent never counts toward cap), then open the new one
- Tab strip: horizontal; overflow may scroll the strip (do not wrap into multiple rows)
- Keyboard: keep click/pointer primary for v1; ensure tabs and close are focusable and activatable from keyboard

**Tab body — file**
- Toolbar: **Preview** | **Source** segmented control (Preview default per tab)
- **Preview:**
  - Markdown paths (`.md` and existing promotable markdown detection): rendered HTML (headings, lists, code, links, tables as reasonable)
  - Other text: plain monospace block
  - Binary / unreadable: clear message, not a blank pane
- **Source:** monospace full text of file content (read-only)
- Both modes read-only — no edit-in-tab
- **Save to source:** button when `isPromotableWorkspaceFile` for the focused file; opens existing `PromoteDialog` with same payload contract as today
- Loading / error for file fetch: inline in tab body; failed load does not crash other tabs

**State isolation**
- Agent: draft text, message list scroll position, composer focus/selection must survive file-tab switches and polls
- Each file tab: path, content snapshot (refresh policy below), preview|source mode, scroll position best-effort
- Closing a file tab drops that tab’s UI state; reopening refetches

### 4.3 Files right bar

**Not** a nested preview. Tree + search only.

| Item | Spec |
|------|------|
| Toggle | Session header; labels “Show files” / “Hide files” |
| Default | **Closed** |
| Persist open/closed | `localStorage` (see §8) |
| Gate | `workspaceEnabled(session)` / `workspace_files` — same as current `WorkspacePanel` mount gate. If false: no toggle, no bar, no split |
| Search | Search input at top; filters tree client-side by path/name substring (case-insensitive) |
| Tree | Hierarchical, notes-like (`tree-item` tokens). Prefer path segments as folders when API returns flat paths; directories non-selectable for open |
| Active highlight | Path matching the focused **file tab** (if any) |
| Changed hint | Amber hint on paths recently changed via tool messages (keep current `changed_path` derivation from tool role messages) |
| Refresh | Load tree on open/session enter; refresh when tool-message changed-path signature changes (carry forward `WorkspacePanel` behavior) |
| Empty | “No files yet” (or equivalent EmptyState copy) when tree loaded empty |
| Loading | Skeleton rows in bar |
| Error | Inline `alert alert--error` in bar; retry via re-toggle or explicit retry control if cheap |

**Click file:** open/focus main file tab and load content via existing `workspaceFile` API.  
**Do not** show preview `<pre>` inside the bar.

### 4.4 Split layout (files open, viewport ≥ ~1024px)

```
[  Main pane (tabs)  |‖|  Files bar  ]
     ~70% default         ~30%
```

- Vertical drag handle between panes
- Default main width **70%** of session body row
- Clamp: main pane **50%–85%** (files **15%–50%**)
- Persist width percent in `localStorage`
- Main pane `min-width: 0` so tabs/chat do not force horizontal page scroll
- Files bar minimum usable width respected by the 15% floor on typical desktop widths

### 4.5 Narrow layout (< ~1024px)

- Main pane full width
- Files toggle opens a **drawer/sheet** over or below content (not a persistent side column)
- No drag-split requirement on narrow; ignore stored width until desktop again
- Drawer dismiss: hide toggle, backdrop click, or Escape

### 4.6 Evolution of `WorkspacePanel`

Refactor responsibilities:

| Concern | Where |
|---------|--------|
| Tree fetch, search, changed paths, empty/loading/error | Files bar component (evolve `WorkspacePanel.svelte` or split `SessionFilesBar.svelte`) |
| File content fetch + Preview/Source + promote CTA | File tab body in session focus shell |
| Gate + toggle + split chrome | Parent session focus layout (`SessionChat.svelte` or thin wrapper) |

Public promote entry moves from bar footer to **file tab** actions. `PromoteDialog` usage unchanged.

---

## 5. Chat / Markdown / Mermaid (Agent tab)

### 5.1 Layout

Full-height column inside Agent tab:

1. (Header + badges live above tabs — shared)  
2. Flex **message scroller** (`flex: 1; min-height: 0; overflow: auto`)  
3. Optional alert line above composer  
4. **Sticky composer** at bottom of pane  

When files bar is closed, use a comfortable reading measure for assistant content inside the main pane (centered or max-width ~65–72ch for bare assistant prose). User bubbles remain end-aligned within the thread. When files bar is open, measure follows the narrower main pane.

### 5.2 Message presentation

| Role | Presentation |
|------|----------------|
| **user** | Distinct bubble, end-aligned; clear but calmer (keep accent or softened accent; no loud shadows) |
| **assistant** | **No bubble**, **no “Assistant” label** — bare content block on canvas |
| **tool** / other | Keep discreet treatment (existing raw-role data attributes OK); do not impersonate assistant markdown hero styling. Prefer compact/muted; exact chrome can match current non-user rows without “Assistant” labeling |

### 5.3 Markdown

- Assistant (and file Preview for markdown) must render: headings, lists, blockquotes, links, tables (reasonable), inline code, fenced code blocks
- **Sanitize** to safe HTML — no raw `<script>`, no attacker-controlled event handlers
- Links: safe schemes only (`http:`, `https:`, `mailto:`); `rel` appropriate for external (`noopener noreferrer`)
- User messages: plain text (or minimal escaping) inside bubble — do not run full markdown pipeline on user content in v1 unless already trivial

### 5.4 Mermaid

- Fenced blocks with language `mermaid` render as diagrams in assistant markdown (and in file Preview when content is markdown)
- **Lazy-render** diagrams (import/initialize on first mermaid fence in view)
- Failure: show original fence/source (or error callout + source), **never** a blank hole
- Re-render on content change; avoid leaking multiple Mermaid runtimes

### 5.5 Library recommendation (default, not exclusive)

`web/package.json` today has **no** markdown or mermaid dependency. Implementation must add maintained packages.

**Default recommendation:**
- **markdown-it** (or **marked**) + a well-maintained HTML sanitizer (**DOMPurify** or equivalent)
- **mermaid** for diagrams
- Thin Svelte wrapper component(s), e.g. `MarkdownView.svelte`, used by Agent messages and file Preview

Equivalent Svelte-friendly stacks are acceptable if they meet acceptance criteria (sanitize, mermaid fences, no XSS, testable). Do not block on package bikeshed if criteria pass.

### 5.6 Composer (non-negotiable)

- Multi-line textarea + primary **Send**
- Disabled while run active / send in flight (existing `sendDisabled` semantics)
- Compact sticky footer
- **Composer form ancestry stable** — poll patches messages/status/disabled in place; never conditionally remount or `innerHTML`-replace the focused composer (standing invariant + `SessionChat.focus.test.ts`)
- Draft retention on failed send; clear on success (existing behavior)

---

## 6. Session lists (vault + project)

### 6.1 Vault Sessions (`VaultSessionsPage.svelte`)

**IA stays desk list (option A):**
- Page header: vault eyebrow + “Sessions” + **New session** (existing href rules)
- Project filter select (All projects / per project)
- Partial failure banner when some projects fail to load
- Empty states: no projects → create project; no sessions → start session
- Loading skeletons

**Visual restyle → card rows** (Claude-inspired “Your chats”):
- Each row/card:
  - **Title** (session title or “Untitled session”)
  - **Meta line:** project name · model (`provider:model` or `model_id`) · **relative activity only if** `created_at` or `updated_at` is present on the session object from the API
- Prefer whole-row click/href to open session (or keep explicit Open control with larger hit target — pick one consistent pattern; whole-row navigation preferred)
- Tokens: `entity-card` / list card classes in `app.css`; shared session row component optional but preferred

### 6.2 Project Sessions list (`ProjectSessionsPage.svelte` when no session open)

- **Create session** block stays primary **above** the list (model picker, workspace_files checkbox, submit)
- Below: session list using the **same card-row language** as vault (title + model meta + relative time if available)
- Opening a session enters **session focus layout** (§3–5), not a third list style
- Do not add Memory / Instructions / multi-user rails

### 6.3 Shared list building blocks

- Extract shared presentational piece or CSS: e.g. `SessionCardRow` / `.session-card` with title + meta slots
- `SessionList.svelte` evolves to card rows or becomes a thin wrapper used by both pages
- Filter/search on vault page remains project `<select>` only for v1 (no new server search)

---

## 7. State (localStorage keys)

Propose namespaced keys (implementation may prefix with app id if a convention already exists):

| Key | Value | Default |
|-----|--------|---------|
| `pa.session.filesBarOpen` | `"1"` / `"0"` | `"0"` (closed) |
| `pa.session.filesBarWidthPct` | integer string **main pane** percent, e.g. `"70"` | `"70"` |
| Existing operation-id keys | unchanged | unchanged |
| App sidebar collapsed | unchanged; **not** written by this feature | user-controlled |

**Rules:**
- Read files-bar prefs when mounting session focus; write on toggle / resize end (debounce resize writes)
- Invalid / out-of-clamp width → fall back to `70` and clamp to 50–85
- Prefs are **global per browser profile**, not per-session id (simpler v1)
- SSR/jsdom: guard `localStorage` as `SessionChat` already does

**In-memory UI state (not localStorage):**
- Open file tabs ordered list, active tab id (`agent` | path), per-tab preview|source mode, per-tab content cache
- Reset file tabs when `session.id` changes
- Agent draft/messages owned by existing session poller state

---

## 8. Components and files touched

| Area | Likely files |
|------|----------------|
| Session focus shell / chat | `web/src/components/sessions/SessionChat.svelte` |
| Files bar / workspace | `web/src/components/sessions/WorkspacePanel.svelte` (evolve or split) |
| Promote | `PromoteDialog.svelte`, `lib/promote.ts` (wire from file tab only) |
| Lists | `SessionList.svelte`, `VaultSessionsPage.svelte`, `ProjectSessionsPage.svelte` |
| Markdown | new `web/src/components/MarkdownView.svelte` (or under `components/md/`) |
| Styles | `web/src/app.css` — session focus layout, tabs, split handle, drawer, card rows, assistant prose, mermaid container |
| Types/API | existing `workspaceTree` / `workspaceFile` / session types only |
| Deps | `web/package.json` — markdown + sanitizer + mermaid |
| Tests | see §10 |
| Shell canvas width | `AppShell` / route wrapper / `.content-canvas` modifier for session-focus only |

**Do not** change Go API handlers for this v1 layout.

---

## 9. UX states

| Surface | Empty | Loading | Error |
|---------|-------|---------|-------|
| Agent messages | Quiet empty thread; composer ready | Initial snapshot: skeleton or prior pattern; polls patch in place | Inline alert; composer stays mounted |
| Files bar | “No files yet” | Skeleton tree rows | Alert in bar; tree unavailable |
| File tab | n/a (tab only exists after open) | Body skeleton / “Loading…” | “Unable to read file” in body; tab remains closable |
| File Preview mermaid | n/a | Optional diagram placeholder then draw | Fence/source fallback |
| Vault sessions | EmptyState + New session / New project | Page skeletons | Page alert; partial failure warn |
| Project sessions list | EmptyState under create block | Skeletons on list region | Inline/page alert as today |

Run active: composer disabled; run status subtle in header; messages continue to patch in place.

---

## 10. Testing and acceptance criteria

### 10.1 Automated (Vitest / Testing Library)

1. **Files toggle** — with `workspace_files`, toggle shows/hides bar; without grant, toggle absent  
2. **Tab open / focus / close** — tree file click opens tab; second click same path focuses; ✕ closes  
3. **Tab reuse + cap** — same path one tab; opening beyond 8 file tabs closes oldest file tab  
4. **Split preference** — width clamp 50–85; localStorage read/write for open + width (jsdom mock)  
5. **Composer focus invariant** — poll updates messages/run without replacing focused composer DOM node or selection (extend `SessionChat.focus.test.ts`)  
6. **Agent tab survival** — switching to file tab and back keeps draft value  
7. **Markdown smoke** — assistant content with heading/list/code renders elements (not raw fences only)  
8. **Mermaid smoke** — ` ```mermaid ` block invokes renderer path; failure path shows fallback, not empty  
9. **List cards** — vault and project lists render title + meta; timestamp omitted when fields missing  
10. **Tree search** — filter hides non-matching paths  
11. **Promote** — Save to source visible on promotable file tab; opens dialog with path  
12. **Styles baseline** — new shared classes live in `app.css` tokens patterns (no required one-off indigo)

### 10.2 Manual / craft (implementation time)

- `frontend-ui-craft`: short screen specs for (1) session focus with files closed, (2) files open + file tab Preview, (3) vault sessions desk  
- Browser vibe-pass when UI reachable  
- After UI edits served by Go static: `npm --prefix web run build`, confirm `web/dist` hashes + cache-bust vibe-pass  
- Node 22 on PATH for web tests (`>=22 <23`)

### 10.3 Acceptance checklist (product)

- [ ] App left nav unchanged; no auto-collapse on session open  
- [ ] Session focus can use fuller width than 1120px cap  
- [ ] Agent tab always present; file tabs behave per §4.2  
- [ ] Files bar default closed; tree-only; toggle + width persisted  
- [ ] Assistant: no bubble, no label; markdown + mermaid  
- [ ] User: end-aligned bubble  
- [ ] Composer sticky, disabled while run active, focus-safe under poll  
- [ ] Promote from file tab only when promotable  
- [ ] Vault desk = card list; project list aligned; no new APIs  
- [ ] Narrow <1024px: files drawer/sheet  

---

## 11. Out of scope / follow-ups

**Out of scope (v1)** — see §1 non-goals.

**Reasonable follow-ups (not required to ship this design):**
- Per-session or per-project files-bar prefs  
- Keyboard shortcuts (e.g. toggle files, next/prev tab)  
- Richer tree (create/delete, multi-select)  
- Edit-in-tab + save back to workspace  
- Amp-style Changes / diff sidebar  
- Session history left rail / multi-session tabs  
- Dark mode  
- Streaming token paint polish beyond current poll  
- Server-driven session search on vault desk  

---

## 12. Implementation notes (for planning)

1. Prefer **one** session focus layout component tree owned by `SessionChat` so poller, promote, and composer invariant stay centralized.  
2. Extract markdown rendering before restyling bubbles so assistant chrome and file Preview share one path.  
3. Keep `workspaceTree` / `workspaceFile` / promote request shapes identical.  
4. CSS: introduce explicit blocks e.g. `.session-focus`, `.session-tabs`, `.session-files`, `.session-split-handle`, `.message-prose`, `.session-card` — tokens first.  
5. Content canvas: e.g. `.content-canvas--session-focus { width: 100%; max-width: none; }` applied only when a session is open.  
6. Do not implement UI in the design-commit thread; next step is `writing-plans` / execution on a feature branch.

---

## 13. Summary

Ship a **session-only** focus pack: Amp-like **Agent + file tabs**, optional **files right bar** (tree/search), calmer **markdown/Mermaid** agent canvas, and **card-style** vault/project session lists — without touching global shell, APIs, or composer safety invariants.
