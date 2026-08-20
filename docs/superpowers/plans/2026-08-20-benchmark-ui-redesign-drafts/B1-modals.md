# Phase B1 — Shared Modal + migrate creates (draft)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** `amp -m grok45 --no-archive-after-execute -x '…'` — not Task/OpenAI. Isolate with git worktrees when using local `-x`.
>
> **Assembly:** This draft is Tasks 2–3 only. Master assembles into `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign.md`.

**Goal:** Ship a shared `Modal.svelte` primitive and migrate New project / New vault create flows off inline `panel form-inline` soup onto modals (spec §8).

**Spec:** `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md` §8 Modals  
**Lock:** `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign-lock.md`  
**Prior art:** `web/src/components/sessions/PromoteDialog.svelte` + `PromoteDialog.test.ts` (native `<dialog>`, `showModal`/`close`, jsdom polyfill)

**Why:** Catalog creates currently expand into the page as `form-inline` panels (`ProjectsPage`, `VaultsPage`, `VaultProjectsPage`). Spec requires backdrop modals with Esc, focus containment, focus return, primary + secondary actions, and errors inside the modal. `PromoteDialog` stays as-is (domain-specific promote flow); new creates share one primitive.

**Tech / gates:**
- Node `>=22 <23` for web tests (`export PATH="$HOME/.local/node-v22/bin:$PATH"`)
- Tokens first in `web/src/app.css` (`.modal` class)
- Svelte 5 runes + `Snippet` children (`@render children()`)
- TDD every task; no TBD
- Do **not** migrate PromoteDialog onto Modal in this phase (retain existing)
- Session “more options” modal is **out of scope** (optional later)
- No hub / vault list visual redesign here (B2/B4) — only create chrome

---

## File map (B1)

| Path | Action | Responsibility |
|------|--------|----------------|
| `web/src/components/Modal.svelte` | Create | Shared native `<dialog class="modal">` primitive |
| `web/src/components/Modal.test.ts` | Create | Unit tests: open/close, title, role=dialog, children |
| `web/src/components/ModalHarness.svelte` | Create | Test-only harness that supplies a real Svelte 5 snippet child (same pattern as `AppShellHarness.svelte`) |
| `web/src/app.css` | Modify | Add `.modal` (+ backdrop / layout tokens as needed) |
| `web/src/styles-baseline.test.ts` | Modify | Assert `.modal` is a declared craft primitive |
| `web/src/routes/ProjectsPage.svelte` | Modify | New project → Modal (unfiled `vault_id: null`) |
| `web/src/routes/ProjectsPage.test.ts` | Modify | Assert `role=dialog`; submit still mocks api |
| `web/src/routes/VaultsPage.svelte` | Modify | New vault → Modal |
| `web/src/routes/VaultsPage.test.ts` | Modify | Assert `role=dialog`; submit still mocks api |
| `web/src/routes/VaultProjectsPage.svelte` | Modify | New project → Modal; vault locked via context/`vault_id` |
| `web/src/routes/VaultProjectsPage.test.ts` | Modify | Assert `role=dialog`; vault field + submit still mocks api |

**Out of scope files:** PromoteDialog\*, hub/session pages, vault list row chrome (B4).

---

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
