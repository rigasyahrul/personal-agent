# Session Focus Layout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** `amp -m grok45 --no-archive-after-execute -x '…'` — not Task/OpenAI. Isolate with git worktrees when using local `-x`.

**Goal:** Ship session-only focus UI: Amp-style Agent + file tabs, optional files right bar (tree/search), 70/30 resizable split, bare markdown+Mermaid assistant output, and card-style vault/project session lists — without changing global shell or APIs.

**Architecture:** Own focus chrome in `SessionChat` (tabs, split, prefs). Extract `SessionFilesBar` (tree/search) and `SessionFileTab` (preview/source/promote) from `WorkspacePanel`. Shared `MarkdownView` for assistant messages and file Preview. List pages consume `SessionCardRow` / evolved `SessionList`. CSS tokens in `app.css`; prefs in `session-prefs.ts`.

**Tech Stack:** Svelte 5 + TypeScript + Vite + Tailwind + Vitest + Testing Library; add `markdown-it`, `dompurify`, `mermaid` (+ types as needed). Node `>=22 <23`.

**Spec:** `docs/superpowers/specs/2026-08-20-session-focus-layout-design.md`  
**Lock:** `docs/superpowers/plans/2026-08-20-session-focus-layout-lock.md`

## Global Constraints

- Spec wins on conflicts; no API / auth / review algorithm changes; hash routing unchanged.
- `workspace_files` grant gate unchanged (`workspaceEnabled(session)`).
- Poll must never remount/replace focused composer (`SessionChat.focus.test.ts` stays green).
- App left nav unchanged; no auto-collapse.
- Assistant: **no bubble**, **no "Assistant" label**; user keeps end-aligned bubble.
- Files bar: tree + search **only**; file body lives in **main tabs**.
- Default files bar **closed**; when open default **70% main / 30% files**; clamp main **50–85**; persist prefs.
- File tabs cap **8** (LRU among file tabs); Agent never counts toward cap.
- Tokens first in `app.css`; no `bg-indigo-600` scaffold soup.
- Web tests: `export PATH="$HOME/.local/node-v22/bin:$PATH"` before `npm --prefix web test`.
- After UI ship claims against Go static: rebuild `web/dist`, cache-bust vibe-pass.
- Do **not** merge to main or push unless the user explicitly asks.

## Canonical contracts

### localStorage

| Key | Values | Default |
|-----|--------|---------|
| `pa.session.filesBarOpen` | `"1"` \| `"0"` | `"0"` |
| `pa.session.filesBarWidthPct` | integer string = **main pane** % | `"70"` |

Invalid width → clamp to `[50, 85]`, else `70`.

### CSS classes (minimum)

`.content-canvas--session-focus`, `.session-focus`, `.session-focus__header`, `.session-tabs`, `.session-tab`, `.session-tab--active`, `.session-tab__close`, `.session-split`, `.session-split__main`, `.session-split__handle`, `.session-split__files`, `.session-files`, `.session-files__search`, `.session-file-tab`, `.session-file-tab__toolbar`, `.message-prose`, `.message-row--assistant` (bare), `.message-bubble--user` (keep), `.session-card`, `.session-card__title`, `.session-card__meta`, `.session-files-drawer` (narrow).

### Components / modules

| Path | Role |
|------|------|
| `web/src/lib/session-prefs.ts` | read/write/clamp prefs |
| `web/src/lib/markdown/render.ts` | `renderMarkdownToSafeHtml(source: string): string` |
| `web/src/lib/markdown/tree.ts` | optional: build tree nodes from flat paths |
| `web/src/components/markdown/MarkdownView.svelte` | `{ source: string }` — sanitized HTML + mermaid |
| `web/src/components/sessions/SessionChat.svelte` | focus shell owner |
| `web/src/components/sessions/SessionFilesBar.svelte` | tree + search; `onopen(path)` |
| `web/src/components/sessions/SessionFileTab.svelte` | preview/source + promote |
| `web/src/components/sessions/SessionCardRow.svelte` | list card presentational |
| `web/src/components/sessions/SessionList.svelte` | uses cards |
| `web/src/routes/VaultSessionsPage.svelte` | desk list cards |
| `web/src/routes/ProjectSessionsPage.svelte` | create + cards; mounts SessionChat |
| `web/src/app.css` | tokens |
| `web/src/shell/AppShell.svelte` or sessions page | apply `content-canvas--session-focus` when session open |

### Tab model (in-memory on SessionChat)

```ts
type TabId = 'agent' | `file:${string}` // file: full workspace path
type FileTabState = {
  path: string
  mode: 'preview' | 'source'
  // content loaded inside SessionFileTab
}
// openFileTabs: FileTabState[] max 8
// activeTab: TabId
// on session.id change: reset file tabs; keep prefs
```

### Split

- CSS variable or inline style: `--session-main-pct: 70` on `.session-split`
- Drag handle writes pct on pointerup (debounce optional); clamp 50–85

### Promote

- CTA only on file tab when `isPromotableWorkspaceFile`; same `PromoteDialog` + `lib/promote.ts`

---

## File map (summary)

See lock § File map. Do not create extra doc trees.

---

### Task 1: Session prefs helper

**Files:**
- Create: `web/src/lib/session-prefs.ts`
- Create: `web/src/lib/session-prefs.test.ts`

**Interfaces:**
- Produces:
  - `FILES_BAR_OPEN_KEY = 'pa.session.filesBarOpen'`
  - `FILES_BAR_WIDTH_KEY = 'pa.session.filesBarWidthPct'`
  - `DEFAULT_MAIN_PCT = 70`
  - `MIN_MAIN_PCT = 50`
  - `MAX_MAIN_PCT = 85`
  - `readFilesBarOpen(storage: Storage | null | undefined): boolean`
  - `writeFilesBarOpen(storage: Storage | null | undefined, open: boolean): void`
  - `readFilesBarWidthPct(storage: Storage | null | undefined): number`
  - `writeFilesBarWidthPct(storage: Storage | null | undefined, pct: number): void`
  - `clampMainPct(pct: number): number`

- [x] **Done** (restored on resume orb from prior worker `88e6dc9`)

---

### Task 2: Session-focus CSS tokens

**Files:**
- Modify: `web/src/app.css`
- Modify: `web/src/styles-baseline.test.ts`

**Interfaces:**
- Produces: CSS classes listed in Canonical contracts

- [ ] **Step 1: Extend baseline test**

Add expectation loop including:
`.session-focus`, `.session-tabs`, `.session-tab`, `.session-split`, `.session-files`, `.session-card`, `.message-prose`, `.content-canvas--session-focus`

- [ ] **Step 2: Run — expect FAIL**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web test -- src/styles-baseline.test.ts
```

- [ ] **Step 3: Implement CSS**

Add a `/* —— Session focus —— */` block:
- `.content-canvas--session-focus { width: 100%; max-width: none; padding-inline: 16px; }`
- `.session-focus { display: flex; flex-direction: column; min-height: calc(100vh - 52px - 40px); gap: 0; }`
- Header row flex; tabs strip horizontal scroll; split as `display:flex; flex:1; min-height:0`
- `.session-split__main { flex: 1 1 var(--session-main-pct, 70%); min-width: 0; }`
- `.session-split__files { flex: 0 0 calc(100% - var(--session-main-pct, 70%)); min-width: 0; max-width: 50%; }`
- Handle ~4–6px wide, `cursor: col-resize`
- `.message-prose` typography for assistant markdown (headings, pre, code, tables)
- `.session-card` based on entity-card density
- `@media (max-width: 1023px)` files as drawer (`.session-files-drawer`)

Keep existing `.session-layout` rules until Task 5 removes dead paths — can leave for now.

- [ ] **Step 4: Run — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add web/src/app.css web/src/styles-baseline.test.ts
git commit -m "style(web): session focus layout tokens"
```

---

### Task 3: Markdown render helper (safe HTML)

**Files:**
- Modify: `web/package.json` (via npm install)
- Create: `web/src/lib/markdown/render.ts`
- Create: `web/src/lib/markdown/render.test.ts`

**Interfaces:**
- Produces: `renderMarkdownToSafeHtml(source: string): string`
- Deps: `markdown-it`, `dompurify`, `@types/dompurify`

- [ ] **Step 1: Install deps**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web install markdown-it dompurify
npm --prefix web install -D @types/markdown-it @types/dompurify
```

- [ ] **Step 2: Failing tests for render**

```ts
import { describe, expect, it } from 'vitest'
import { renderMarkdownToSafeHtml } from './render'

describe('renderMarkdownToSafeHtml', () => {
  it('renders headings and lists', () => {
    const html = renderMarkdownToSafeHtml('# Title\n\n- a\n- b')
    expect(html).toContain('<h1>')
    expect(html).toContain('<li>')
  })
  it('strips script tags', () => {
    const html = renderMarkdownToSafeHtml('ok <script>alert(1)</script>')
    expect(html.toLowerCase()).not.toContain('<script')
  })
  it('keeps fenced code', () => {
    const html = renderMarkdownToSafeHtml('```js\nconst x = 1\n```')
    expect(html).toContain('<code')
  })
  it('marks mermaid fences for the view layer', () => {
    const html = renderMarkdownToSafeHtml('```mermaid\ngraph TD; A-->B\n```')
    expect(html.includes('language-mermaid') || html.includes('data-mermaid')).toBe(true)
  })
})
```

- [ ] **Step 3: Run — FAIL; implement with markdown-it + DOMPurify; PASS; commit**

```bash
git add web/package.json web/package-lock.json web/src/lib/markdown/
git commit -m "feat(web): safe markdown render helper"
```

**Note:** Mermaid **drawing** is Task 4 in `MarkdownView`; render only needs recognizable fences.

---

### Task 4: MarkdownView + Mermaid

**Files:**
- Create: `web/src/components/markdown/MarkdownView.svelte`
- Create: `web/src/components/markdown/MarkdownView.test.ts`
- Modify: `web/package.json` — add `mermaid`

**Interfaces:**
- Consumes: `renderMarkdownToSafeHtml`
- Produces: `<MarkdownView source={string} />`
- Behavior: set `@html` sanitized; `onMount`/`$effect` query mermaid nodes; dynamic `import('mermaid')`; `mermaid.run` or `render`; on failure show `<pre class="mermaid-fallback">` source — never blank

- [ ] **Step 1: Install mermaid**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web install mermaid
```

- [ ] **Step 2: Tests**

- Renders heading text from markdown source  
- Does not throw when source has invalid mermaid (shows fallback text)  
- Mock mermaid module if needed for jsdom  

- [ ] **Step 3: Implement component; PASS; commit**

```bash
git add web/package.json web/package-lock.json web/src/components/markdown/
git commit -m "feat(web): MarkdownView with lazy Mermaid"
```

---

### Task 5: SessionChat focus shell (tabs + Agent layout + toggle + split)

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `web/src/components/sessions/SessionChat.test.ts`
- Keep green: `web/src/components/sessions/SessionChat.focus.test.ts`
- Modify: `web/src/routes/ProjectSessionsPage.svelte` and/or `AppShell` to add `content-canvas--session-focus` when `activeSession` set

**Interfaces:**
- Consumes: `session-prefs`, CSS tokens, later FilesBar/FileTab
- In this task: may still mount legacy `WorkspacePanel` **inside** files pane when open OR empty files slot with toggle chrome only — prefer toggle + split chrome + Agent markdown now; wire real bar in Task 6

**Behavior checklist:**
1. Root `.session-focus`
2. Header: Back (onclose), title, model chip, run status, **Files** toggle (only if `workspaceEnabled(session)`), aria-pressed
3. Tab strip: Agent + file tabs (file tabs empty until Task 6/7)
4. Agent body: scroller + sticky composer (stable `<form>`)
5. Assistant messages: `MarkdownView` / `.message-prose`, **no** bubble wrapper, **no** "Assistant"/role strong label
6. User: `.message-bubble--user` end-aligned
7. Tool/other: muted compact row, no Assistant labeling
8. Files open → split with handle; width from prefs; drag updates prefs
9. Toggle writes `pa.session.filesBarOpen`

- [ ] **Step 1: Update tests**

```ts
// SessionChat.test.ts — replace bubble alignment test:
it('renders assistant as bare prose without Assistant label', async () => {
  // mock listMessages with user + assistant
  // expect no text "Assistant"
  // expect .message-prose or markdown content
  // expect .message-bubble--user for user only
})
it('toggles files bar and persists pref', async () => {
  // session with workspace_files true
  // click Show files
  // expect localStorage pa.session.filesBarOpen === '1'
})
```

- [ ] **Step 2: FAIL → implement shell → PASS**

- [ ] **Step 3: Run focus test**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web test -- src/components/sessions/SessionChat.focus.test.ts src/components/sessions/SessionChat.test.ts
```

- [ ] **Step 4: Commit**

```bash
git add web/src/components/sessions/SessionChat.svelte web/src/components/sessions/SessionChat.test.ts web/src/routes/ProjectSessionsPage.svelte web/src/shell/AppShell.svelte web/src/app.css
git commit -m "feat(web): session focus shell with tabs chrome and files toggle"
```

---

### Task 6: SessionFilesBar (tree + search)

**Files:**
- Create: `web/src/components/sessions/SessionFilesBar.svelte`
- Create: `web/src/components/sessions/SessionFilesBar.test.ts`
- Create: `web/src/lib/workspace-tree.ts` (+ test) — build hierarchy from flat `WorkspaceEntry[]` paths if API is flat
- Deprecate preview UI in `WorkspacePanel.svelte` (delete or thin-reexport)

**Interfaces:**
- Props: `{ sessionId: string; messages?: ChatMessage[]; activePath?: string | null; onopen?: (path: string) => void }`
- Consumes: `api.workspaceTree`, changed_path logic from current WorkspacePanel
- Search: case-insensitive substring on path
- Click file → `onopen(path)`; directories expand only
- No `<pre>` preview in bar

- [ ] **Step 1: Port tests from WorkspacePanel.test.ts** (tree refresh on tool path; no promote-in-bar)

- [ ] **Step 2: Implement bar; PASS; commit**

```bash
git add web/src/components/sessions/SessionFilesBar.svelte web/src/components/sessions/SessionFilesBar.test.ts web/src/lib/workspace-tree.ts web/src/lib/workspace-tree.test.ts
git commit -m "feat(web): session files bar tree and search"
```

---

### Task 7: File tabs + SessionFileTab + promote move

**Files:**
- Create: `web/src/components/sessions/SessionFileTab.svelte`
- Create: `web/src/components/sessions/SessionFileTab.test.ts`
- Modify: `SessionChat.svelte` — open/close/cap/LRU tabs; render `SessionFileTab` when active
- Modify/remove: `WorkspacePanel.svelte` / tests — promote only from file tab
- Update: `WorkspacePanel.test.ts` or replace with FilesBar + FileTab tests

**Interfaces:**
- `SessionFileTab` props: `{ sessionId, path, projectId, onpromote?: (file) => void }`
- Preview default; Source toggle; MarkdownView for md; monospace otherwise
- Promote button when promotable
- SessionChat: `openFile(path)`, `closeFile(path)`, LRU when >8

- [ ] **Step 1: Tests**

- Open file from bar focuses tab  
- Reuse same path  
- 9th path closes least-recently-activated file tab  
- Close button removes tab  
- Promote visible for `.md`  
- Preview/Source switch  

- [ ] **Step 2: Implement; run SessionChat + FileTab + FilesBar + focus tests; commit**

```bash
git add web/src/components/sessions/
git commit -m "feat(web): file tabs with preview source and promote"
```

---

### Task 8: Narrow drawer + split polish

**Files:**
- Modify: `SessionChat.svelte`, `app.css`
- Test: resize clamp unit already in prefs; add component test for `data-files-open` / drawer class when `matchMedia` mocked if feasible — otherwise CSS + manual vibe note

- [ ] Implement pointer drag on handle (pointerdown/move/up); set `--session-main-pct`
- [ ] `<1024px`: files overlay drawer; Escape/backdrop closes; write open pref
- [ ] Commit: `fix(web): session split drag and narrow files drawer`

---

### Task 9: SessionCardRow + SessionList cards

**Files:**
- Create: `web/src/components/sessions/SessionCardRow.svelte`
- Modify: `web/src/components/sessions/SessionList.svelte`
- Modify: `web/src/components/sessions/SessionList.test.ts`
- CSS: `.session-card*` already in Task 2

**Interfaces:**
- Props example:
```ts
{
  title: string
  meta: string
  onclick?: () => void
  href?: string
}
```
- Relative time helper: `formatRelativeTime(iso: string | undefined): string | null` — return null if missing

- [ ] Tests: empty list; click emits open; meta omits time when no timestamp
- [ ] Implement; commit `feat(web): session list card rows`

---

### Task 10: Vault + Project sessions pages

**Files:**
- Modify: `web/src/routes/VaultSessionsPage.svelte`
- Modify: `web/src/routes/VaultSessionsPage.test.ts`
- Modify: `web/src/routes/ProjectSessionsPage.svelte`
- Modify: `web/src/routes/ProjectSessionsPage.test.ts`

- [ ] Vault: card rows with project name · model · relative time if `created_at`/`updated_at`
- [ ] Whole-row navigation preferred
- [ ] Project page: create form stays primary; list uses same cards; opening session still sets `activeSession`
- [ ] Apply `content-canvas--session-focus` when chat open (if not done in Task 5)
- [ ] Tests stay green; add assertion for `.session-card` or role
- [ ] Commit `feat(web): vault and project session card lists`

---

### Task 11: Harden — full web tests + craft gates

**Files:**
- Touch tests only as needed

- [ ] **Step 1: Run full web suite**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web test
```

- [ ] **Step 2: Fix failures without weakening focus invariant**

- [ ] **Step 3: Browser vibe-pass** (frontend-ui-craft)

1. `npm --prefix web run build`
2. Serve via existing personal-agent / docker-dev / portal
3. Open project sessions → create/open session
4. Check: Agent bare markdown, user bubble, files toggle, tree open file tab, preview/source, promote if md, resize, vault sessions cards
5. Cache-bust `?v=<ts>#/projects/.../sessions`

- [ ] **Step 4: Commit any fixes** — do **not** push/merge unless user asks

---

## Acceptance checklist (from spec)

- [ ] Session focus only; shell nav unchanged  
- [ ] Agent + file tabs; Preview/Source; close on file tabs only  
- [ ] Files bar tree+search; default closed; 70/30; clamp 50–85; prefs  
- [ ] Assistant: no bubble, no label; markdown + mermaid  
- [ ] User: end-aligned bubble  
- [ ] Composer sticky, focus-safe under poll  
- [ ] Promote from file tab when promotable  
- [ ] Vault/project session cards  
- [ ] Narrow drawer  
- [ ] `npm --prefix web test` green  
- [ ] Vibe-pass evidence  

## Out of scope (do not implement)

Global shell redesign, auto-collapse, notes redesign, edit-in-tab, Amp Changes/Portal/Terminal, focus left history rail, new APIs, dark mode.

---

## Execution note

Parallel Grok draft writers (A–E) hit model stream timeouts (2026-08-20); this plan was assembled by the master thread from the approved spec + lock. Implementation workers should still be **grok45** per task/phase with worktree isolation.
