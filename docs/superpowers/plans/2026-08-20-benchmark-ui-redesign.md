# Benchmark UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** After each task: `consulting-grok-review` via **new** `amp -m grok45 --no-archive-after-execute -x '…'` (never `-ox`, never Task/OpenAI self-review). Isolate product work in git worktrees when using local `-x`.
>
> **Spec:** `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md`  
> **Lock:** `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign-lock.md`

**Goal:** Make the authenticated UI structurally match the Claude / Grok / Amp benchmark screenshots: compact shell nav, Claude-style project hub (prompt + sessions below + right rail), Amp open-session tabs with bottom composer and assistant copy, Grok Memory|Files rail, vault name-first list, and modal creates.

**Architecture:** Path C surface pack. Phase A is a pure CSS density fix for stretched sidebar nav. Phase B adds a shared `<dialog>` Modal, rewrites ProjectHubPage into a two-pane workspace (main + default-open ProjectRail), embeds SessionChat in the hub main canvas, migrates vault projects to name-first rows, then hardens `frontend-ui-craft` with a benchmark fidelity gate and full vibe-pass. Memory/soul persist API is out of scope (chrome only).

**Tech Stack:** Svelte 5 + TypeScript + Vite + Tailwind + Vitest + Testing Library; tokens in `web/src/app.css`; Go serves `web/dist` on `:8080`.

## Global Constraints

- Spec wins over `2026-08-20-session-focus-layout-design.md` on rail default-open, Memory|Files tabs, assistant copy, hub structure, and bottom-composer gates.
- Node `>=22 <23` on `PATH` before any `npm test` / `make web-test`.
- Rebuild `web/dist` + cache-bust (`?v=<ts>#/route`) before claiming localhost vibe-pass (Go serves dist, not Vite HMR).
- Polled session UI: never remount/replace a focused composer (`SessionChat.focus.test.ts` is a hard gate).
- Memory tab: design fields only — no enabled save, toast, or “saved” claim without API.
- Creates: modals only for New project / New vault — no inline `form-inline` soup.
- App left nav: user collapse only — never auto-collapse on hub/session entry.
- Tokens first in `web/src/app.css` (`btn--*`, `panel`, `field-*`, new `.modal`, hub/rail classes) before one-off utilities.
- Every worker task: consulting-grok-review PASS before FF-merge (repo standing rule).
- Do not commit large benchmark PNGs unless user explicitly asks; refs live at repo root / `.amp/in/artifacts/` during vibe-pass.
- Ship = push when user allows; commit alone is not ship.

## File map

| Path | Responsibility |
|------|----------------|
| `web/src/app.css` | Shell density, modal, hub two-pane, rail tabs, session composer/copy tokens |
| `web/src/styles-baseline.test.ts` | CSS contract assertions for density/modal/hub/rail/session tokens |
| `web/src/shell/Sidebar.svelte` | Unchanged markup preferred; density via CSS |
| `web/src/shell/Sidebar.test.ts` | Existing nav behavior stays green |
| `web/src/components/Modal.svelte` | Shared native `<dialog>` primitive |
| `web/src/components/Modal.test.ts` | Open/close/Esc/title/children |
| `web/src/components/ProjectRail.svelte` | Right rail: Memory \| Files tabs; Memory chrome; Files tree |
| `web/src/components/ProjectRail.test.ts` | Tab switch; non-persistent memory; tree render |
| `web/src/routes/ProjectHubPage.svelte` | Claude hub: prompt + composer + sessions below + rail; embed SessionChat |
| `web/src/routes/ProjectHubPage.test.ts` | Hub craft + start-session + open-row contracts |
| `web/src/routes/ProjectSessionsPage.svelte` | Legacy: redirect/alias to hub |
| `web/src/routes/ProjectsPage.svelte` | New project → Modal |
| `web/src/routes/VaultsPage.svelte` | New vault → Modal |
| `web/src/routes/VaultProjectsPage.svelte` | Name-first list + Modal create |
| `web/src/components/sessions/SessionChat.svelte` | Continuous rail; dense bottom composer; assistant copy; files → tabs |
| `web/src/components/sessions/SessionChat.test.ts` | Tabs, copy, composer chrome |
| `web/src/components/sessions/SessionChat.focus.test.ts` | Must stay green |
| `web/src/App.svelte` | sessions route → hub if needed |
| `web/src/lib/api/index.ts` | Existing: createProjectSession, sendMessage, listProjectSessions, listModels, listProjectNotes, workspaceTree |
| `web/src/lib/workspace-tree.ts` | Existing hierarchy builder for Files tree |
| `.agents/skills/frontend-ui-craft/SKILL.md` | Benchmark fidelity gate |
| `.agents/skills/frontend-ui-craft/reference/craft.md` | Benchmark vibe-pass detail |

## Canonical contracts

### CSS (Phase A)

```css
.sidebar { width: 240px; padding: 12px 10px; /* was 16px 12px */ }
.sidebar nav {
  display: grid;
  gap: 2px;
  margin: 12px 0;
  flex: 1;
  align-content: start; /* CRITICAL: prevents ~147px stretched rows */
}
.sidebar nav a,
.sidebar__disabled { min-height: 40px; /* 36–40 allowed */ }
.sidebar__collapse { margin-top: auto; }
```

### Modal

```ts
// web/src/components/Modal.svelte
let {
  open = false,
  title,
  onclose,
  children,
}: {
  open?: boolean
  title: string
  onclose: () => void
  children: import('svelte').Snippet
} = $props()
// native <dialog class="modal">; $effect open → showModal() / close()
```

### Hub start session

```ts
// ProjectHubPage — on Send (non-empty draft):
const models = await api.listModels()
const m = models.models[0]
const session = await api.createProjectSession(projectId, {
  home: 'project',
  title: draft.trim().slice(0, 80) || 'Untitled',
  provider: m.provider,
  model_id: m.model_id,
  model_parameters: {},
  tool_grants: { workspace_files: false },
})
await api.sendMessage(session.id, { content: draft.trim(), request_key: crypto.randomUUID() })
// then set activeSession = session (main shows SessionChat)
```

### ProjectRail

```ts
// tabs: 'memory' | 'files' — default 'memory' or last local preference
// Memory fields: bind local state only; label non-persistent; no save button that claims success
// Files: buildHierarchy from notes tree and/or workspaceTree(sessionId) when session + workspace_files
// onOpenFile?: (path: string) => void  // hub may no-op; session opens file tab
```

### SessionChat (supersession)

- Right rail default **open** (not default-closed files bar).
- Agent tab: sticky **bottom** composer (no `Message` label soup).
- Each assistant message container: focusable copy control → `navigator.clipboard.writeText(plain)`.
- File tab active: hide composer without destroying form ancestry.
- Poll: patch messages/run only; composer form stays mounted.

### Legacy route

```ts
// Prefer: route name 'sessions' renders ProjectHubPage with same projectId
// or replace hash to #/projects/:id
```

### Benchmark acceptance (Task 12)

| Ref | Check |
|-----|--------|
| Shell | nav item ≤ 44px @ 1440×900 |
| claude.png | hub prompt top + session rows below; no metric destination grid |
| claude-2.png | vault name-first rows; modal create |
| grok / grok-2 | rail default open; Memory \| Files |
| amp.png | Agent + file tabs; bottom composer; assistant copy |

---

## Tasks

<!-- DRAFTS ASSEMBLED BELOW -->

---

## Phase A — Shell

### Task 1: Shell nav density fix

**Files:**
- Modify: `web/src/app.css` (`.sidebar`, `.sidebar nav`, nav row rules only as needed)
- Modify: `web/src/styles-baseline.test.ts` (density contract assertions)
- Modify: `web/src/shell/Sidebar.test.ts` only if a behavior assertion is needed; prefer CSS contract in baseline — existing Sidebar tests must stay green
- Test: `web/src/styles-baseline.test.ts`
- Test: `web/src/shell/Sidebar.test.ts`

**Interfaces / contracts (CSS source contracts, string-asserted):**
- `.sidebar` expanded width is `220px`–`240px` inclusive (keep `240px` or tighten within range; collapsed stays `64px`)
- `.sidebar` padding is compact **~12×10** (e.g. `padding: 12px 10px`) — not the current `16px 12px`
- `.sidebar nav` keeps `display: grid` and `flex: 1` (collapse control remains bottom-pinned)
- `.sidebar nav` includes `align-content: start` (or equivalent that packs rows to the start and prevents stretch distribution)
- `.sidebar nav` must **not** use stretch-distributing content alignment (`align-content: stretch` / omit that causes stretch)
- `.sidebar nav a` and `.sidebar__disabled` keep `min-height` in **36px–40px** (current `40px` is valid)
- Brand stays compact; collapse control stays at bottom (`margin-top: auto` on `.sidebar__collapse` unchanged)

**Acceptance A (spec):**
- At 1440×900, every primary nav item height ≤ 44px
- No row/gap visually consumes unused sidebar height

**Out of scope for this task:** hub, rail, modals, session chrome, `web/dist` rebuild (no vibe-pass gate on other surfaces here)

---

- [ ] **Step 1: Write the failing density contract tests**

Add a focused describe (or `it`) to `web/src/styles-baseline.test.ts` that locks the Phase A CSS contracts. Keep all existing baseline tests intact.

```ts
// web/src/styles-baseline.test.ts — add inside describe('visual baseline', …)
// (file already loads `const css = readFileSync(join(here, 'app.css'), 'utf8')`)

it('packs sidebar nav rows without stretch (benchmark Phase A density)', () => {
  // Expanded sidebar chrome: 220–240px width, ~12×10 padding
  expect(css).toMatch(/\.sidebar\s*\{[^}]*width:\s*(220|240|230|225|235)px/s);
  expect(css).toMatch(/\.sidebar\s*\{[^}]*padding:\s*12px\s+10px/s);

  // Nav grows to push collapse down, but rows pack to start (no stretch fill)
  const navBlock = css.match(/\.sidebar nav\s*\{[^}]*\}/);
  expect(navBlock?.[0]).toBeTruthy();
  expect(navBlock![0]).toMatch(/flex:\s*1/);
  expect(navBlock![0]).toMatch(/display:\s*grid/);
  expect(navBlock![0]).toMatch(/align-content:\s*start/);

  // Row min-height stays compact (36–40px)
  expect(css).toMatch(
    /\.sidebar nav a,\s*\n\s*\.sidebar__disabled\s*\{[^}]*min-height:\s*(36|37|38|39|40)px/s,
  );
});
```

Optional hardening in the same test (include if the implementer wants a negative guard):

```ts
// Still inside the same it(…)
// Default grid stretch is the bug; explicit stretch must not reappear on nav
expect(navBlock![0]).not.toMatch(/align-content:\s*stretch/);
```

Do **not** delete or weaken existing Sidebar behavior tests. Confirm `web/src/shell/Sidebar.test.ts` still covers:

- global labels + collapse persistence
- vault context swap
- real SVG icons (no bullet glyphs)
- labeled collapse control
- disabled global Sessions assistive text

No new Sidebar.svelte markup is required for density; CSS-only fix is sufficient.

- [ ] **Step 2: Run tests and verify the new density test fails**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/styles-baseline.test.ts src/shell/Sidebar.test.ts
```

**Expected:** FAIL on the new baseline assertion(s). Current `app.css` has:

```css
.sidebar {
  width: 240px;
  /* … */
  padding: 16px 12px; /* too tall vertical pad vs ~12×10 */
}
.sidebar nav { display: grid; gap: 2px; margin: 12px 0; flex: 1; }
/* missing align-content: start → rows stretch to ~147px in tall sidebars */
```

Failure modes (any is fine):

- missing `align-content: start` on `.sidebar nav`
- padding still `16px 12px` (not `12px 10px`)

Existing `Sidebar.test.ts` cases should still PASS (behavior unchanged).

- [ ] **Step 3: Fix sidebar CSS density (minimal change)**

In `web/src/app.css`, update only the shell density rules. Canonical target:

```css
.sidebar {
  width: 240px; /* within 220–240; 240px OK */
  display: flex;
  flex-direction: column;
  padding: 12px 10px; /* was 16px 12px */
  background: var(--sidebar);
  border-right: 1px solid var(--border);
}
.sidebar[data-collapsed='true'] { width: 64px; }

/* … brand / collapsed label rules unchanged … */

.sidebar nav {
  display: grid;
  gap: 2px;
  margin: 12px 0;
  flex: 1;                 /* keep: absorbs free space so collapse stays bottom */
  align-content: start;    /* fix: pack rows; do not stretch items to fill height */
}
.sidebar nav a,
.sidebar__disabled {
  display: flex;
  gap: 10px;
  align-items: center;
  min-height: 40px; /* 36–40 allowed; keep 40 */
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  color: #3f3f46;
}

/* … hover / current / disabled / context unchanged … */

.sidebar__collapse {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  margin-top: auto; /* keep bottom pin */
  padding: 8px 10px;
  /* … rest unchanged … */
}
```

**Do not:**

- remove `flex: 1` from `.sidebar nav` (would risk collapse floating up unless another spacer is introduced)
- change nav item markup in `Sidebar.svelte`
- touch mobile `@media` sidebar rules except if a conflict appears (collapsed width / off-canvas transform stay as-is)
- introduce one-off indigo/scaffold classes

**Why this works:** `flex: 1` still gives the nav the free column height; `align-content: start` packs grid tracks to the top so leftover height is empty space *below the last row*, not distributed into each row. Collapse stays bottom via flex column + `margin-top: auto` on `.sidebar__collapse`.

- [ ] **Step 4: Run the same tests and verify they pass**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/styles-baseline.test.ts src/shell/Sidebar.test.ts
```

**Expected:** PASS — density contract green; all existing Sidebar tests green.

If baseline fails on the width regex, adjust the test or the CSS so width remains in `220|225|230|235|240`. Prefer keeping `width: 240px` and matching the test to it.

- [ ] **Step 5: Commit**

```bash
git add web/src/app.css web/src/styles-baseline.test.ts web/src/shell/Sidebar.test.ts
git commit -m "$(cat <<'MSG'
fix(web): pack sidebar nav rows for benchmark density

Stop grid stretch on .sidebar nav (align-content: start), tighten
sidebar padding to 12×10, keep 36–40px row min-height and bottom collapse.
MSG
)"
```

Only stage files actually touched. If `Sidebar.test.ts` was not modified, omit it from `git add`.

---

### Task 1 done criteria

- [x] Failing density test written first, then CSS fix, then green
- [x] `.sidebar nav` has `align-content: start` and still `flex: 1` + `display: grid`
- [x] Nav row `min-height` ∈ 36–40px
- [x] Expanded sidebar width ∈ 220–240px; padding `12px 10px`
- [x] Collapse control remains at bottom; no IA/route changes
- [x] `styles-baseline.test.ts` + `Sidebar.test.ts` pass under Node 22
- [x] Commit created

**Manual spot-check (not a separate task):** at ~1440×900, primary nav items read as single dense rows (≤44px), unused sidebar height is empty gap above the collapse control — not fat stretched links.

---

## Phase B1 — Modals

### Task 2: Modal.svelte primitive

**Files:**
- Create: `web/src/components/Modal.svelte`
- Create: `web/src/components/Modal.test.ts`
- Create: `web/src/components/ModalHarness.svelte` (test-only; may live next to Modal)
- Modify: `web/src/app.css` (add `.modal` block near full-surface craft primitives)
- Modify: `web/src/styles-baseline.test.ts` (assert `.modal` in craft primitives list)

**Interfaces / contracts:**

```ts
// Modal.svelte props (Svelte 5 runes)
{
  open?: boolean              // default false
  title: string               // required heading text
  onclose?: () => void        // called on Cancel, Esc/native dialog close, backdrop dismiss
  children: Snippet           // body content (form fields, alerts, actions)
}
```

- Root element: native `<dialog bind:this={dialogEl} class="modal" …>`
- When `open` becomes true → `queueMicrotask(() => dialogEl?.showModal())` (same pattern as PromoteDialog)
- When `open` becomes false → `dialogEl.close()` if open
- `onclose` on the dialog element forwards to prop `onclose` (native close / Esc)
- Title rendered as `<h2>` (or equivalent) inside dialog; visible and findable via `getByRole('heading', { name: title })`
- Children via `{@render children()}`
- CSS class `.modal` exists in `app.css` and is asserted by `styles-baseline`
- jsdom: polyfill `HTMLDialogElement.prototype.showModal` / `close` exactly like PromoteDialog.test

**Do not:**
- Put create-project / create-vault API logic inside Modal (pages own submit)
- Replace PromoteDialog
- Use non-dialog overlays (`div` + fixed) — native `<dialog>` only

---

- [ ] **Step 1: Write the failing Modal unit tests + baseline assertion**

Create `web/src/components/ModalHarness.svelte` so tests can pass a real snippet child (Vitest/testing-library cannot easily pass `Snippet` props from plain objects):

```svelte
<!-- web/src/components/ModalHarness.svelte -->
<script lang="ts">
  import Modal from './Modal.svelte'

  let {
    open = false,
    title = 'Test modal',
    onclose,
  }: {
    open?: boolean
    title?: string
    onclose?: () => void
  } = $props()
</script>

<Modal {open} {title} {onclose}>
  <p>Harness body</p>
  <button type="button" class="btn btn--secondary" onclick={() => onclose?.()}>Cancel</button>
</Modal>
```

Create `web/src/components/Modal.test.ts`:

```ts
// web/src/components/Modal.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ModalHarness from './ModalHarness.svelte'

afterEach(cleanup)

describe('Modal', () => {
  beforeEach(() => {
    // jsdom dialog polyfill (same as PromoteDialog.test.ts)
    if (!HTMLDialogElement.prototype.showModal) {
      HTMLDialogElement.prototype.showModal = function showModal() {
        this.setAttribute('open', '')
      }
      HTMLDialogElement.prototype.close = function close() {
        this.removeAttribute('open')
      }
    }
  })

  it('does not expose a dialog when closed', () => {
    render(ModalHarness, { props: { open: false, title: 'New project' } })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'New project' })).not.toBeInTheDocument()
  })

  it('opens a native dialog with title and children when open', async () => {
    render(ModalHarness, { props: { open: true, title: 'New project' } })
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toBeInTheDocument()
    expect(dialog.tagName).toBe('DIALOG')
    expect(dialog).toHaveClass('modal')
    expect(screen.getByRole('heading', { name: 'New project' })).toBeInTheDocument()
    expect(screen.getByText('Harness body')).toBeInTheDocument()
  })

  it('calls onclose from Cancel', async () => {
    const onclose = vi.fn()
    render(ModalHarness, { props: { open: true, title: 'New vault', onclose } })
    await screen.findByRole('dialog')
    await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onclose).toHaveBeenCalledTimes(1)
  })
})
```

Add `.modal` to the craft-primitives list in `web/src/styles-baseline.test.ts` (existing `it('declares full-surface craft primitives', …)`):

```ts
// web/src/styles-baseline.test.ts — inside the token array of
// it('declares full-surface craft primitives', () => { … })
for (const token of [
  '.panel',
  '.form-stack',
  '.field-input',
  '.scope-chip',
  '.list-panel',
  '.link-accent',
  '.catalog-grid',
  '.alert--error',
  '.btn--primary',
  '.entity-card',
  '.metric-card',
  '.modal', // benchmark B1 shared dialog primitive
]) {
  expect(css).toContain(token);
}
```

- [ ] **Step 2: Run tests and verify they fail**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/components/Modal.test.ts src/styles-baseline.test.ts
```

**Expected:** FAIL — `Modal.svelte` / harness missing, and/or baseline missing `.modal` in `app.css`.

- [ ] **Step 3: Implement Modal.svelte + `.modal` CSS (minimal)**

`web/src/components/Modal.svelte`:

```svelte
<!-- web/src/components/Modal.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import type { Snippet } from 'svelte'

  let {
    open = false,
    title,
    onclose,
    children,
  }: {
    open?: boolean
    title: string
    onclose?: () => void
    children: Snippet
  } = $props()

  let dialogEl = $state<HTMLDialogElement | null>(null)

  $effect(() => {
    if (open) {
      queueMicrotask(() => dialogEl?.showModal())
    } else {
      try {
        if (dialogEl?.open) dialogEl.close()
      } catch {
        /* ignore */
      }
    }
  })

  onMount(() => {
    return () => {
      try {
        if (dialogEl?.open) dialogEl.close()
      } catch {
        /* ignore */
      }
    }
  })

  function handleClose() {
    onclose?.()
  }
</script>

<dialog
  bind:this={dialogEl}
  class="modal"
  onclose={handleClose}
>
  <div class="modal__chrome">
    <h2 class="modal__title">{title}</h2>
    <div class="modal__body">
      {@render children()}
    </div>
  </div>
</dialog>
```

Add to `web/src/app.css` (near full-surface craft primitives, after `.panel` / before or after `.form-stack` is fine):

```css
/* Shared modal primitive (benchmark B1) — native <dialog class="modal"> */
.modal {
  width: min(100% - 2rem, 28rem);
  max-width: 28rem;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--panel);
  color: #18181b;
  box-shadow: 0 20px 40px rgb(15 23 42 / 0.18);
}
.modal::backdrop {
  background: rgb(15 23 42 / 0.4);
}
.modal__chrome {
  display: grid;
  gap: 12px;
  padding: 20px;
}
.modal__title {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  line-height: 1.3;
}
.modal__body {
  display: grid;
  gap: 12px;
}
/* Forms inside modals stack like form-stack */
.modal__body .form-stack {
  margin: 0;
}
```

Notes:
- Use existing tokens (`--panel`, `--border`, `--radius-lg`) — no indigo/scaffold one-offs.
- `PromoteDialog` keeps its own `promote-dialog` classes; do not force-migrate it.
- Focus containment / Esc / backdrop click are native `<dialog showModal()>` behaviors; do not reimplement unless a browser gap appears in tests.

- [ ] **Step 4: Run tests and verify they pass**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/components/Modal.test.ts src/styles-baseline.test.ts
```

**Expected:** PASS — Modal open/close/title/children green; baseline includes `.modal`.

If `getByRole('dialog')` fails while attribute `open` is set: ensure polyfill sets `open` attribute (as above). If closed state still finds dialog: prefer not rendering children visibility claims on closed; with polyfill, closed dialogs lack `open` and should not match `role=dialog` in testing-library — if jsdom always exposes the node, assert `expect(dialogEl.open).toBe(false)` via container query instead, but keep the public contract as “no dialog role when closed” if the environment supports it.

- [ ] **Step 5: Commit**

```bash
git add \
  web/src/components/Modal.svelte \
  web/src/components/Modal.test.ts \
  web/src/components/ModalHarness.svelte \
  web/src/app.css \
  web/src/styles-baseline.test.ts
git commit -m "$(cat <<'MSG'
feat(web): add shared Modal primitive for benchmark creates

Native dialog.modal with open/title/onclose/children, app.css tokens,
and styles-baseline assertion. PromoteDialog unchanged.
MSG
)"
```

Only stage files actually created/modified.

---

### Task 2 done criteria

- [ ] `Modal.svelte` exists with props `open` / `title` / `onclose` / `children` (Snippet)
- [ ] Native `<dialog class="modal">`; `showModal` when open (PromoteDialog pattern)
- [ ] `.modal` in `app.css`; `styles-baseline` asserts `.modal`
- [ ] `Modal.test.ts` green under Node 22 (polyfill showModal/close)
- [ ] Commit created

---

### Task 3: Migrate catalog creates to Modal

**Files:**
- Modify: `web/src/routes/ProjectsPage.svelte`
- Modify: `web/src/routes/ProjectsPage.test.ts`
- Modify: `web/src/routes/VaultsPage.svelte`
- Modify: `web/src/routes/VaultsPage.test.ts`
- Modify: `web/src/routes/VaultProjectsPage.svelte`
- Modify: `web/src/routes/VaultProjectsPage.test.ts`

**Interfaces / contracts:**
- Consumes: `Modal` from `../components/Modal.svelte` (Task 2)
- Each page keeps its existing create API payload:
  - ProjectsPage: `POST /api/v1/projects` `{ name, vault_id: null }`
  - VaultsPage: `POST /api/v1/vaults` `{ name }`
  - VaultProjectsPage: `POST /api/v1/projects` via `createVaultProjectInput(name, vaultId)` → `{ name, vault_id }`
- `creating` boolean still gates the modal (`open={creating}`); Cancel / dialog close sets `creating = false`
- Empty-state primary actions still set `creating = true` (same as today)
- **No** remaining `class="panel form-inline"` create forms on these three pages
- Errors from create stay visible; prefer error **inside** the modal (`role="alert"` in modal body). Page-level load errors may remain outside.
- Secondary action: Cancel button (`btn btn--secondary`) calls close
- Primary action: Create … submit button (`btn btn--primary`) unchanged labels

**Polyfill:** page tests that open a dialog must install the same jsdom `showModal`/`close` polyfill in `beforeEach` (copy from PromoteDialog / Modal tests).

---

- [ ] **Step 1: Write / extend failing page tests (dialog + api)**

Update each page test file. Keep existing list/search/craft tests. Change create tests so opening “New …” yields `role=dialog`, and submit still hits mocked api.

**ProjectsPage.test.ts** — replace/extend the create test and add polyfill:

```ts
// web/src/routes/ProjectsPage.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectsPage from './ProjectsPage.svelte'
import { api } from '../lib/api/client'
vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } }))

describe('ProjectsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    if (!HTMLDialogElement.prototype.showModal) {
      HTMLDialogElement.prototype.showModal = function showModal() {
        this.setAttribute('open', '')
      }
      HTMLDialogElement.prototype.close = function close() {
        this.removeAttribute('open')
      }
    }
  })

  it('shows only searched unfiled projects', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [
      { id: 'a', name: 'Alpha', note_count: 0 }, { id: 'b', name: 'Beta', vault_id: 'v1', vault_name: 'WORK', note_count: 0 },
    ] })
    render(ProjectsPage)
    expect(await screen.findByText('Alpha')).toBeInTheDocument()
    expect(screen.queryByText('Beta')).not.toBeInTheDocument()
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'none' } })
    expect(screen.getByText('No matching projects')).toBeInTheDocument()
  })

  it('opens New project in a modal and creates an unfiled project', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [] })
    vi.mocked(api.post).mockResolvedValue({ id: 'new', name: 'Inbox', vault_id: null, note_count: 0 })
    render(ProjectsPage)
    await fireEvent.click(await screen.findByRole('button', { name: 'New project' }))
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'New project' })).toBeInTheDocument()
    await fireEvent.input(screen.getByLabelText('Project name'), { target: { value: 'Inbox' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Create project' }))
    expect(api.post).toHaveBeenCalledWith('/api/v1/projects', { name: 'Inbox', vault_id: null })
  })

  it('uses craft hierarchy without Global desk eyebrow', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [] })
    render(ProjectsPage)
    expect(await screen.findByRole('heading', { level: 1, name: 'Projects' })).toBeInTheDocument()
    expect(screen.queryByText('Global desk')).not.toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'New project' })[0].className).toMatch(/btn--primary/)
  })
})
```

**VaultsPage.test.ts** — same polyfill; create test asserts dialog:

```ts
// web/src/routes/VaultsPage.test.ts — create test body (keep other its)
it('opens New vault in a modal and creates a vault', async () => {
  vi.mocked(api.get).mockResolvedValueOnce([]).mockResolvedValueOnce({ projects: [], generated_at: '' })
  vi.mocked(api.post).mockResolvedValue({ id: 'v2', name: 'WORK', created_at: '', updated_at: '' })
  render(VaultsPage)
  await fireEvent.click(await screen.findByRole('button', { name: 'New vault' }))
  expect(await screen.findByRole('dialog')).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: 'New vault' })).toBeInTheDocument()
  await fireEvent.input(screen.getByLabelText('Vault name'), { target: { value: 'WORK' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Create vault' }))
  expect(api.post).toHaveBeenCalledWith('/api/v1/vaults', { name: 'WORK' })
  expect(navigate).toHaveBeenCalledWith('#/vaults/v2')
})
```

Add the same `showModal`/`close` polyfill in `beforeEach` of VaultsPage tests (alongside `vi.clearAllMocks()`).

**VaultProjectsPage.test.ts** — polyfill + dialog assertion; keep vault lock:

```ts
// web/src/routes/VaultProjectsPage.test.ts — update the create it
it('opens New project in a modal, locks the vault, and submits vault_id', async () => {
  vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [] })
  vi.mocked(api.post).mockResolvedValue({
    id: 'new',
    name: 'Sleep',
    vault_id: 'v1',
    note_count: 0,
  })
  render(VaultProjectsPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })
  await fireEvent.click(await screen.findByRole('button', { name: /new project/i }))
  expect(await screen.findByRole('dialog')).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: 'New project' })).toBeInTheDocument()
  const vaultField = screen.getByLabelText('Vault')
  expect(vaultField).toHaveValue('HEALTH')
  expect(vaultField).toBeDisabled()
  await fireEvent.input(screen.getByLabelText('Project name'), { target: { value: 'Sleep' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Create project' }))
  expect(api.post).toHaveBeenCalledWith('/api/v1/projects', { name: 'Sleep', vault_id: 'v1' })
})
```

Add polyfill in `beforeEach` for VaultProjectsPage tests.

- [ ] **Step 2: Run page tests and verify dialog assertions fail**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- \
  src/routes/ProjectsPage.test.ts \
  src/routes/VaultsPage.test.ts \
  src/routes/VaultProjectsPage.test.ts
```

**Expected:** FAIL on `findByRole('dialog')` (inline `form-inline` is still in the page, not a dialog). Existing create/api assertions may still pass until markup moves — the new dialog expectation is the intentional red.

- [ ] **Step 3: Migrate the three pages to Modal**

**ProjectsPage.svelte** — canonical target shape:

```svelte
<!-- web/src/routes/ProjectsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import Modal from '../components/Modal.svelte'
  import ProjectCard from '../components/ProjectCard.svelte'
  import SearchField from '../components/SearchField.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Project } from '../lib/api/types'
  import { filterByQuery, isUnfiled } from '../lib/catalog'
  import { navigate } from '../lib/router'

  let projects = $state<Project[]>([])
  let query = $state('')
  let loading = $state(true)
  let creating = $state(false)
  let saving = $state(false)
  let name = $state('')
  let error = $state('')
  let createError = $state('')

  let visible = $derived(filterByQuery(projects, query))

  onMount(async () => {
    try {
      projects = (await api.get<HomeResponse>('/api/v1/home')).projects.filter(isUnfiled)
    } catch (e) {
      error = e instanceof Error ? e.message : 'Could not load projects.'
    } finally {
      loading = false
    }
  })

  function openCreate() {
    creating = true
    name = ''
    createError = ''
  }

  function closeCreate() {
    creating = false
    name = ''
    createError = ''
  }

  async function createProject() {
    const clean = name.trim()
    if (!clean) return
    saving = true
    createError = ''
    try {
      const project = await api.post<Project>('/api/v1/projects', { name: clean, vault_id: null })
      closeCreate()
      navigate(`#/projects/${encodeURIComponent(project.id)}`)
    } catch (e) {
      createError = e instanceof Error ? e.message : 'Could not create project.'
    } finally {
      saving = false
    }
  }
</script>

<svelte:head><title>Projects · Personal Agent</title></svelte:head>
<div class="page-stack">
  <header class="page-header">
    <div><h1>Projects</h1></div>
    <div class="page-header__actions">
      <button type="button" class="btn btn--primary" onclick={openCreate}>New project</button>
    </div>
  </header>
  <SearchField bind:value={query} label="Search projects" />
  {#if error}<p role="alert" class="alert alert--error">{error}</p>{/if}

  <Modal open={creating} title="New project" onclose={closeCreate}>
    <form
      class="form-stack"
      onsubmit={(e) => {
        e.preventDefault()
        createProject()
      }}
    >
      <label>
        Project name
        <input class="field-input" bind:value={name} aria-label="Project name" />
      </label>
      {#if createError}
        <p role="alert" class="alert alert--error">{createError}</p>
      {/if}
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn--secondary" onclick={closeCreate}>Cancel</button>
        <button disabled={saving || !name.trim()} class="btn btn--primary" type="submit">Create project</button>
      </div>
    </form>
  </Modal>

  {#if loading}
    <div class="catalog-grid" aria-busy="true"><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
  {:else if visible.length}
    <div class="catalog-grid">
      {#each visible as project (project.id)}
        <ProjectCard {project} onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)} />
      {/each}
    </div>
  {:else if query.trim()}
    <EmptyState title="No matching projects" description="Try a different project name." actionLabel="Clear search" onaction={() => query = ''} />
  {:else}
    <EmptyState title="No unfiled projects yet" description="Create your first project on the global desk." actionLabel="New project" onaction={openCreate} />
  {/if}
</div>
```

**VaultsPage.svelte** — same pattern; title `"New vault"`; field label `Vault name`; submit `Create vault`; `POST /api/v1/vaults` `{ name: clean }`; navigate `#/vaults/${id}`.

```svelte
<!-- Key Modal block for VaultsPage (rest of page structure unchanged aside from removing form-inline) -->
<script lang="ts">
  // …existing imports…
  import Modal from '../components/Modal.svelte'
  // openCreate / closeCreate / createVault mirror ProjectsPage
  // createVault posts { name: clean } to /api/v1/vaults
</script>

<!-- header New vault → openCreate; EmptyState onaction={openCreate} -->

<Modal open={creating} title="New vault" onclose={closeCreate}>
  <form
    class="form-stack"
    onsubmit={(e) => {
      e.preventDefault()
      createVault()
    }}
  >
    <label>
      Vault name
      <input class="field-input" bind:value={name} aria-label="Vault name" />
    </label>
    {#if createError}
      <p role="alert" class="alert alert--error">{createError}</p>
    {/if}
    <div class="flex justify-end gap-2">
      <button type="button" class="btn btn--secondary" onclick={closeCreate}>Cancel</button>
      <button disabled={saving || !name.trim()} class="btn btn--primary" type="submit">Create vault</button>
    </div>
  </form>
</Modal>
```

Full VaultsPage implementation must retain: dual fetch (`/api/v1/vaults` + home for counts), search, VaultCard grid, craft header without Global desk eyebrow. Delete the `{#if creating}<form class="panel form-inline"…>` block entirely.

**VaultProjectsPage.svelte** — Modal title `"New project"`; keep disabled Vault field for context clarity; still submit via `createVaultProjectInput`; honor `?new=1` / hash `new=1` by setting `creating = true` on mount (existing behavior).

```svelte
<!-- Key Modal block for VaultProjectsPage -->
<script lang="ts">
  // …existing imports + props vaultId / vaultName…
  import Modal from '../components/Modal.svelte'
  // openCreate / closeCreate; createProject uses createVaultProjectInput(name, vaultId)
</script>

<Modal open={creating} title="New project" onclose={closeCreate}>
  <form
    class="form-stack"
    onsubmit={(e) => {
      e.preventDefault()
      createProject()
    }}
  >
    <label>
      Vault
      <input class="field-input" value={vaultName} disabled aria-label="Vault" />
    </label>
    <label>
      Project name
      <input class="field-input" bind:value={name} aria-label="Project name" />
    </label>
    {#if createError}
      <p role="alert" class="alert alert--error">{createError}</p>
    {/if}
    <div class="flex justify-end gap-2">
      <button type="button" class="btn btn--secondary" onclick={closeCreate}>Cancel</button>
      <button disabled={saving || !name.trim()} class="btn btn--primary" type="submit">Create project</button>
    </div>
  </form>
</Modal>
```

**Hard requirements after migrate:**
- Zero matches for `form-inline` in these three page files
- Header + empty-state both open the same modal
- Create errors render inside the modal (`createError`), not only as page banner
- Load errors can remain page-level (`error`)

- [ ] **Step 4: Run page + Modal tests and verify green**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- \
  src/components/Modal.test.ts \
  src/styles-baseline.test.ts \
  src/routes/ProjectsPage.test.ts \
  src/routes/VaultsPage.test.ts \
  src/routes/VaultProjectsPage.test.ts
```

**Expected:** PASS — each New project / New vault path exposes `role=dialog`; api.post payloads unchanged; vault lock still posts `vault_id: 'v1'`.

Optional sanity (not required if focused run is green):

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test
```

- [ ] **Step 5: Commit**

```bash
git add \
  web/src/routes/ProjectsPage.svelte \
  web/src/routes/ProjectsPage.test.ts \
  web/src/routes/VaultsPage.svelte \
  web/src/routes/VaultsPage.test.ts \
  web/src/routes/VaultProjectsPage.svelte \
  web/src/routes/VaultProjectsPage.test.ts
git commit -m "$(cat <<'MSG'
feat(web): migrate project and vault creates to Modal

Replace inline form-inline create panels on Projects, Vaults, and
VaultProjects with shared Modal; keep API payloads and vault lock.
MSG
)"
```

---

### Task 3 done criteria

- [ ] ProjectsPage / VaultsPage / VaultProjectsPage use `<Modal>` for creates
- [ ] No `panel form-inline` create forms remain on those pages
- [ ] Tests: New project / New vault open `role=dialog`; submit still mocks api with same payloads
- [ ] VaultProjectsPage still locks vault (disabled field + `vault_id` in POST)
- [ ] Create errors surface inside the modal
- [ ] Node 22 focused test run green
- [ ] Commit created

---

## Spec coverage (B1 only)

| Spec §8 requirement | Task |
|---------------------|------|
| Shared modal primitive (backdrop, Esc, focus via native dialog) | Task 2 |
| New project (global) → modal; `vault_id` null | Task 3 ProjectsPage |
| New project (vault) → modal; context supplies `vault_id` | Task 3 VaultProjectsPage |
| New vault → modal (name) | Task 3 VaultsPage |
| Promote retained | Explicit non-goal (no change) |
| No inline expand-into-page create forms on catalogs | Task 3 |
| Session more options optional | Out of scope |

## Placeholder scan

No TBD/TODO. Real test code, CSS, Svelte markup, PATH+npm commands, and commit messages included.

## Type / name consistency

- Prop names: `open`, `title`, `onclose`, `children` (Modal) — pages use `open={creating}` `onclose={closeCreate}`
- CSS: `.modal`, `.modal__chrome`, `.modal__title`, `.modal__body`
- Button copy unchanged: `New project` / `Create project` / `New vault` / `Create vault` / `Cancel`
- API paths and payloads unchanged from current pages

---

## Phase B2 — Project hub

### Task 4: Hub/rail CSS tokens + ProjectRail shell

**Files:**
- Modify: `web/src/app.css`
- Modify: `web/src/styles-baseline.test.ts`
- Create: `web/src/components/ProjectRail.svelte`
- Create: `web/src/components/ProjectRail.test.ts`

**CSS contracts to add (assert in styles-baseline):**
```css
.project-workspace { display: grid; grid-template-columns: minmax(0, 1fr) minmax(260px, 320px); gap: 0; min-height: calc(100vh - 52px); }
.project-workspace__main { min-width: 0; padding: 16px 20px 32px; }
.project-workspace__rail { border-left: 1px solid var(--border); background: var(--panel); display: flex; flex-direction: column; min-height: 0; }
.rail-tabs { display: flex; gap: 0; border-bottom: 1px solid var(--border); }
.rail-tab { /* quiet tab button */ }
.rail-tab--active { /* accent bottom border or soft bg */ }
.rail-panel { flex: 1; overflow: auto; padding: 12px; }
.hub-start { /* prompt block */ }
.hub-start__title { font-size: 1.75rem; font-weight: 650; letter-spacing: -0.03em; margin: 0 0 16px; }
.hub-composer { /* dense multi-line + send row */ }
.hub-session-list { margin-top: 24px; display: grid; gap: 8px; }
/* Relax width: when hub/session, content-canvas full width */
.content-canvas--project-workspace { width: 100%; max-width: none; margin: 0; padding: 0; }
```

**ProjectRail props:**
```ts
{
  projectId: string
  sessionId?: string | null
  workspaceFilesEnabled?: boolean
  onOpenFile?: (path: string) => void
}
```

- Tabs: `memory` | `files` (buttons role=tab, tablist aria-label="Project rail")
- Memory: two labeled `field-textarea` — "Memory", "Instructions (system)"; local `$state` only; helper text: "Not saved yet — persistence coming later." **No Save button.**
- Files: load `api.listProjectNotes(projectId)` → map to tree via `buildHierarchy` (adapt NoteTreeEntry→WorkspaceEntry: path + kind file/directory). If `sessionId` && workspaceFilesEnabled, also `api.workspaceTree(sessionId)` and merge/group under "Workspace". Click file → `onOpenFile?.(path)`. Empty: "No project files available."

- [ ] **Step 1: Failing baseline + ProjectRail tests**

```ts
// styles-baseline: expect(css).toContain('.project-workspace'); expect(css).toContain('.rail-tab');
// ProjectRail.test.ts
it('shows Memory and Files tabs; Memory has non-persistent helper', async () => {
  vi.mocked(api.listProjectNotes).mockResolvedValue([])
  render(ProjectRail, { props: { projectId: 'p1' } })
  expect(screen.getByRole('tab', { name: 'Memory' })).toBeInTheDocument()
  expect(screen.getByRole('tab', { name: 'Files' })).toBeInTheDocument()
  expect(screen.getByLabelText(/memory/i)).toBeInTheDocument()
  expect(screen.getByLabelText(/instructions/i)).toBeInTheDocument()
  expect(screen.getByText(/not saved yet/i)).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /save/i })).toBeNull()
})
it('switches to Files and shows empty copy', async () => {
  // click Files tab → "No project files available."
})
```

- [ ] **Step 2: Run fail**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/styles-baseline.test.ts src/components/ProjectRail.test.ts
```

- [ ] **Step 3: Implement CSS + ProjectRail**
- [ ] **Step 4: Run pass**
- [ ] **Step 5: Commit** `feat(web): add ProjectRail shell and hub workspace tokens`

---

### Task 5: Rewrite ProjectHubPage (Claude stack)

**Files:**
- Modify: `web/src/routes/ProjectHubPage.svelte`
- Rewrite: `web/src/routes/ProjectHubPage.test.ts`
- May use: `web/src/components/sessions/SessionCardRow.svelte` or simple button rows
- Modify: `web/src/shell/AppShell.svelte` or hub only: wrap content so canvas uses `content-canvas--project-workspace` when on hub (prefer set class from page root + :global, or pass flag — simplest: hub root includes full-bleed class and AppShell main already has content-canvas — **override** by making hub the only child with negative margin OR change AppShell to not pad when child requests full bleed via data attribute). **Chosen approach:** In `App.svelte`, when route is `project` or `sessions`, set `<main class="content-canvas content-canvas--project-workspace">`. Minimal AppShell change:

```svelte
<!-- AppShell: accept optional canvasClass prop default '' -->
<main class="content-canvas {canvasClass}">{@render children()}</main>
```

**Hub structure:**
```svelte
<div class="project-workspace">
  <div class="project-workspace__main">
    {#if activeSession}
      <SessionChat ... onclose={() => { activeSession = null; void reloadSessions() }} />
    {:else}
      <header> breadcrumbs / project name · Notes · Review links </header>
      <section class="hub-start">
        <h1 class="hub-start__title">How can I help you today?</h1>
        <form class="hub-composer" onsubmit={startSession}>
          <textarea class="field-textarea" bind:value={draft} aria-label="Message" />
          <button class="btn btn--primary" type="submit" disabled={starting || !draft.trim()}>Send</button>
        </form>
      </section>
      <section class="hub-session-list" aria-label="Sessions">
        {#each sessions as s}
          <button type="button" class="session-card" onclick={() => activeSession = s}>...</button>
        {/each}
        {#if !sessions.length && !loading}
          <p class="text-sm text-muted">No sessions yet. Send a message above to start one.</p>
        {/if}
      </section>
    {/if}
  </div>
  <aside class="project-workspace__rail">
    <ProjectRail projectId={...} sessionId={activeSession?.id} workspaceFilesEnabled={...} />
  </aside>
</div>
```

**startSession:**
```ts
async function startSession(e: Event) {
  e.preventDefault()
  const content = draft.trim()
  if (!content || starting) return
  starting = true
  error = ''
  try {
    const { models } = await api.listModels()
    if (!models?.length) throw new Error('Configure a model in Settings before starting a session.')
    const m = models[0]
    const session = await api.createProjectSession(projectId, {
      home: 'project',
      title: content.slice(0, 80) || 'Untitled',
      provider: m.provider,
      model_id: m.model_id,
      model_parameters: {},
      tool_grants: { workspace_files: false },
    })
    await api.sendMessage(session.id, { content, request_key: crypto.randomUUID() })
    draft = ''
    sessions = [session, ...sessions.filter((x) => x.id !== session.id)]
    activeSession = session
  } catch (cause) {
    error = cause instanceof Error ? cause.message : 'Could not start session.'
  } finally {
    starting = false
  }
}
```

**Remove:** metric strip, destination cards, "New session" button, Sessions destination link as primary (Notes + Review stay quiet header links).

**Tests (replace old metric/destination tests):**
```ts
it('shows Claude start prompt and no metric destination grid', async () => {
  // mock getProject, listProjectSessions=[], listModels, listProjectNotes
  expect(await screen.findByRole('heading', { name: /how can i help you today/i })).toBeVisible()
  expect(screen.queryByRole('region', { name: 'Project metrics' })).toBeNull()
  expect(screen.queryByRole('region', { name: 'Project surfaces' })).toBeNull()
  expect(screen.queryByRole('button', { name: /new session/i })).toBeNull()
  expect(screen.getByRole('link', { name: /notes/i })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: /review/i })).toBeInTheDocument()
  expect(screen.getByRole('tab', { name: 'Memory' })).toBeInTheDocument()
})
it('lists sessions below the composer', async () => {
  // mock sessions [{title:'Test 1',...}]
  // assert order: heading then textarea then session button
})
it('Send creates session and first message then opens chat', async () => {
  // type draft, click Send
  // expect createProjectSession + sendMessage called
  // expect SessionChat / Back or session title visible
})
```

- [ ] **Step 1: Rewrite failing tests first**
- [ ] **Step 2: Run fail**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/routes/ProjectHubPage.test.ts
```

- [ ] **Step 3: Implement hub + AppShell canvasClass if needed**
- [ ] **Step 4: Pass + commit** `feat(web): Claude-style project hub with rail`

---

### Task 6: Legacy `#/projects/:id/sessions` → hub

**Files:**
- Modify: `web/src/App.svelte` — `route.name === 'sessions'` render `<ProjectHubPage projectId={...} />` (same as project)
- Modify: `web/src/routes/ProjectSessionsPage.test.ts` — either delete obsolete page tests or keep page as thin re-export; **prefer** hub-only and slim/remove ProjectSessionsPage route usage
- Optional: leave `ProjectSessionsPage.svelte` file but unused, or make it `export { default } from './ProjectHubPage.svelte'` — cleaner: App routes sessions → ProjectHubPage; delete later if unused

```svelte
{:else if route.name === 'project' || route.name === 'sessions'}
  <ProjectHubPage projectId={route.projectId} onProjectLoad={...} />
```

- [ ] **Step 1: Test App or hub that sessions hash shows hub prompt**

```ts
// If App.test exists, assert; else ProjectHubPage is enough + router comment
// Update ProjectSessionsPage.test.ts: mark skipped OR change to test redirect helper
```

- [ ] **Step 2–4: Implement, pass, commit** `fix(web): route project sessions URL to hub`

**Done B2:** hub matches §5 structure; rail default on; no left session column; no New session button.

---

## Phase B3 — Open session

### Task 7: SessionChat continuous rail + Files → tabs

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `web/src/components/sessions/SessionChat.test.ts`
- May deprecate default-closed-only UX of SessionFilesBar as the sole files UI; prefer embedding `ProjectRail` **from hub** (Task 5 already mounts rail outside SessionChat).

**Architecture decision (locked for implementers):**
- **Rail lives on ProjectHubPage**, not inside SessionChat — so rail stays mounted across hub ↔ session without remount.
- SessionChat props gain: `onOpenFile` not required if Files tree is in hub rail; hub passes `onOpenFile` into ProjectRail that calls into SessionChat via bindable callback or small store.

**Chosen wiring:**
```ts
// ProjectHubPage
let openFileHandler = $state<(path: string) => void>(() => {})
// SessionChat exposes onMount register: onReady={{ openFile: (p) => ... }}
// Simpler: SessionChat accepts optional `externalOpenPath` bindable or:
let fileToOpen = $state<string | null>(null)
<ProjectRail onOpenFile={(p) => { fileToOpen = p }} />
<SessionChat bind:openPath={fileToOpen} ... />
```

Or SessionChat keeps internal files bar **in addition** when not in hub — but hub always provides rail. Spec: rail continuous. **Minimal path:**

1. When SessionChat is used inside hub, pass `embeddedInHub={true}` → hide internal files toggle/bar; hub ProjectRail drives `openPath`.
2. SessionChat `$effect` on `openPath` → open file tab (existing file tab logic).

**Tests:**
```ts
it('opens file tab when openPath prop is set', async () => {
  // render SessionChat with workspace enabled, set openPath to 'notes/a.md'
  // expect tab with name a.md
})
it('when embeddedInHub, does not show Show files toggle', () => {
  // queryByRole button /show files/i null
})
```

- [ ] **Step 1: Failing tests**
- [ ] **Step 2: Run**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/components/sessions/SessionChat.test.ts src/components/sessions/SessionChat.focus.test.ts
```

- [ ] **Step 3: Implement openPath + embeddedInHub**
- [ ] **Step 4: Pass (focus test still green)**
- [ ] **Step 5: Commit** `feat(web): session file tabs open from hub ProjectRail`

---

### Task 8: Dense bottom composer + assistant copy

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte` (Agent tab markup only; keep form ancestry)
- Modify: `web/src/app.css` — `.session-composer`, `.message-copy`
- Modify: `web/src/components/sessions/SessionChat.test.ts`
- Gate: `SessionChat.focus.test.ts` must remain green

**Composer chrome:**
- Remove visible "Message" label soup; use `aria-label="Message"` on textarea
- Classes: `session-composer` sticky bottom; `field-textarea` + `btn btn--primary` Send
- Still a single stable `<form>` wrapping textarea+button (focus test depends on this)

**Assistant copy:**
```svelte
{#if message.role === 'assistant' || message.role === 'model'}
  <div class="message-row message-row--assistant">
    <div class="message-prose">...</div>
    <button
      type="button"
      class="message-copy"
      aria-label="Copy response"
      onclick={() => copyAssistant(message.content)}
    >...</button>
  </div>
{/if}
```

```ts
async function copyAssistant(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    copiedSeq = message.sequence // show "Copied" briefly
  } catch { /* ignore */ }
}
```

Polyfill clipboard in tests if needed:
```ts
Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
```

**Tests:**
```ts
it('copy control copies assistant plain text', async () => {
  // mock messages with assistant content
  await fireEvent.click(screen.getByRole('button', { name: 'Copy response' }))
  expect(navigator.clipboard.writeText).toHaveBeenCalledWith('Hi — how can I help you today?')
})
it('composer has no visible Message label text node soup', () => {
  expect(screen.queryByText('Message', { selector: 'span.font-medium' })).toBeNull()
  expect(screen.getByLabelText('Message')).toBeInTheDocument()
})
```

- [ ] **Step 1: Failing tests**
- [ ] **Step 2: Run SessionChat tests + focus test**
- [ ] **Step 3: Implement CSS + markup (do not remount form)**
- [ ] **Step 4: Pass**
- [ ] **Step 5: Commit** `feat(web): dense session composer and assistant copy`

---

### Task 9: Hub embeds SessionChat; Back restores start stack

**Files:**
- Modify: `web/src/routes/ProjectHubPage.svelte` (if not complete in Task 5)
- Modify: `web/src/routes/ProjectHubPage.test.ts`

**Behaviors:**
- `activeSession` set → main shows SessionChat with `embeddedInHub` + `onclose` clears activeSession and reloads sessions
- ProjectRail stays mounted (sibling, not child of SessionChat)
- Back button in SessionChat calls onclose

**Tests:**
```ts
it('clicking a session row shows chat and Back returns to prompt', async () => {
  // mock sessions, click row, find Back, click, find How can I help you today
})
```

- [ ] **Step 1–5: TDD + commit** `feat(web): open session inside project hub canvas`

**Done B3:** Amp session + bottom composer + copy + rail continuous + focus tests green.

---

## Phase B4 — Vault list

### Task 10: Vault projects name-first rows

**Files:**
- Modify: `web/src/routes/VaultProjectsPage.svelte`
- Modify: `web/src/routes/VaultProjectsPage.test.ts`
- Modify: `web/src/app.css` — `.name-row` list tokens
- Modify: `web/src/styles-baseline.test.ts` — assert `.name-row`
- Optional: Create `web/src/components/NameRow.svelte` if reuse helps; else inline button rows

**Replace:** `catalog-grid` + `ProjectCard` / `entity-card` fat cards  
**With:** vertical list of name-first rows:

```svelte
<ul class="name-list" role="list">
  {#each visible as project (project.id)}
    <li>
      <button type="button" class="name-row" onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)}>
        <span class="name-row__title">{project.name}</span>
        <span class="name-row__meta">{/* optional quiet counts */}</span>
        <span class="name-row__chevron" aria-hidden="true">→</span>
      </button>
    </li>
  {/each}
</ul>
```

**CSS:**
```css
.name-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 4px; }
.name-row {
  display: flex; align-items: center; gap: 12px; width: 100%;
  min-height: 44px; padding: 10px 12px; text-align: left;
  border: 1px solid transparent; border-radius: var(--radius-sm);
  background: transparent; cursor: pointer;
}
.name-row:hover { background: #fafafa; border-color: var(--border); }
.name-row__title { font-weight: 600; flex: 1; min-width: 0; }
.name-row__meta { font-size: 12px; color: var(--muted); }
```

**Keep:** vault eyebrow + Projects h1 + New project → Modal (from Task 3). Empty → modal. Search may stay above list.

**Tests:**
```ts
it('renders name-first rows not entity-card grid', async () => {
  // mock projects
  expect(screen.getByRole('button', { name: /Project 1/i })).toBeInTheDocument()
  expect(document.querySelector('.entity-card')).toBeNull()
  expect(document.querySelector('.name-row')).toBeTruthy()
})
it('New project opens dialog', async () => {
  await fireEvent.click(screen.getByRole('button', { name: /new project/i }))
  expect(screen.getByRole('dialog')).toBeInTheDocument()
})
```

- [ ] **Step 1: Failing tests + baseline `.name-row`**
- [ ] **Step 2: Run**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/styles-baseline.test.ts src/routes/VaultProjectsPage.test.ts
```

- [ ] **Step 3: Implement list + CSS**
- [ ] **Step 4: Pass**
- [ ] **Step 5: Commit** `feat(web): vault projects name-first list`

**Done B4:** vault list matches claude-2 hierarchy; create still modal.

---

## Phase B5 — Skill + vibe-pass

### Task 11: frontend-ui-craft benchmark fidelity gate

**Files:**
- Modify: `.agents/skills/frontend-ui-craft/SKILL.md`
- Modify: `.agents/skills/frontend-ui-craft/reference/craft.md`
- Optional: `.agents/skills/frontend-ui-craft/baseline-red.md` note

**Add to SKILL.md (Mandatory loop / Red flags):**

| Red flag | Why |
|----------|-----|
| User named benchmark screenshots but agent only checked tokens/classes | Tokens ≠ fidelity |
| Claimed vibe-pass without side-by-side vs each named ref | Guessing |
| Blocked browser treated as passed | Blocked ≠ passed |

**Add Positive recipe item:** When refs are named (`claude.png`, `amp.png`, etc.), completion report must list each ref + structural checks (layout regions, not pixel-perfect).

**craft.md section "Benchmark fidelity":**
- Require short fidelity criteria table in screen spec when refs exist
- Side-by-side: open product URL + view ref images
- personal-agent benchmark redesign refs: `.amp/in/artifacts/{claude,claude-2,grok,grok-2,amp}.png` or repo root

- [ ] **Step 1: Edit skill files** (no test required beyond grep/self-check that wording exists)
- [ ] **Step 2: Commit** `docs(skills): require benchmark screenshot fidelity in frontend-ui-craft`

---

### Task 12: Full vibe-pass + dist + checklist

**Files:** none required beyond rebuild artifacts; fix any gaps found in prior tasks.

**Commands:**
```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test
npm run build
# confirm dist hashes
grep -E 'assets/.*\.(js|css)' dist/index.html
# if Go serves :8080, restart or ensure dist copied as project expects
```

**Browser checklist (password login if needed):**

| Check | Pass? |
|-------|-------|
| Nav item height ≤ 44px @ 1440×900 | |
| Hub: “How can I help you today?” + composer + sessions **below** | |
| No metric/destination card grid on hub | |
| No “New session” button | |
| Right rail default open; Memory \| Files tabs | |
| Memory shows non-persistent helper; no fake Save success | |
| Files tree or empty copy | |
| Open session: Agent tab + sticky **bottom** composer | |
| Assistant message has Copy control | |
| Composer has no fat “Message” label soup | |
| Vault projects: name-first rows | |
| New project / New vault open `role=dialog` | |
| `#/projects/:id/sessions` shows hub | |
| SessionChat.focus still green in CI | |

Compare against: `claude.png`, `claude-2.png`, `grok.png`, `grok-2.png`, `amp.png`.

- [ ] **Step 1: Full web test suite green**
- [ ] **Step 2: Build dist; cache-bust vibe-pass each surface**
- [ ] **Step 3: Fix any fidelity gaps (loop to earlier tasks)**
- [ ] **Step 4: Commit remaining + push only if user allows**

**Done B5:** skill updated; vibe-pass evidence recorded in final worker summary (URL + what checked).

---

## Spec coverage checklist (self-review)

| Spec requirement | Task(s) |
|------------------|---------|
| Phase A nav ≤44px / align-content start | 1 |
| Shared modal primitive | 2 |
| New project/vault modals | 3 |
| ProjectRail Memory\|Files default open | 4–5 |
| Hub prompt + sessions below; no New session; no metrics grid | 5 |
| Legacy sessions → hub | 6 |
| Rail continuous; file open → tab | 7 |
| Bottom composer + assistant copy | 8 |
| Hub embeds SessionChat; Back | 9 |
| Vault name-first list | 10 |
| frontend-ui-craft benchmark gate | 11 |
| Full vibe-pass vs 5 refs | 12 |
| Memory no fake save | 4–5, 11–12 |
| Composer focus/poll safe | 8–9 + SessionChat.focus.test.ts |
| consulting-grok-review per task | Global constraint / SDD |

## Execution handoff

Plan complete. Two options:

1. **Subagent-Driven (recommended)** — `subagent-driven-development`; fresh `amp -m grok45 -x` worker per task + consulting-grok-review before merge
2. **Inline Execution** — `executing-plans` in this session with checkpoints

Which approach?
