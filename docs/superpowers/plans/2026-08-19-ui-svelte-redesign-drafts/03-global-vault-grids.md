# Global Dashboard and Catalog Grids Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the global dashboard plus searchable, empty-friendly unfiled-project and vault grids, including create actions and vault navigation.

**Architecture:** Keep context and query filtering in pure functions, then compose small presentational components into route pages. Pages load through the typed API client, own loading/error/create state, and emit navigation through the hash router; no backend changes are required.

**Tech Stack:** Svelte 5, TypeScript, Vite, Tailwind CSS, Vitest, Testing Library

## Global Constraints

- Global Projects contains only projects whose `vault_id` is empty, null, or missing.
- Vault project filtering is an exact `vault_id === ctx.vaultId` comparison.
- Search is client-side, case-insensitive, trims the query, and searches `name` only.
- Vault badges show the vault name only for vaulted projects; never render a fake “Unfiled” badge.
- Every grid has a dedicated empty state and primary create action; loading uses skeletons.
- Use only existing `GET /api/v1/home`, `GET/POST /api/v1/vaults`, and `POST /api/v1/projects` APIs.
- Hash routing remains the navigation model.

### Shared contracts consumed from earlier tasks

```ts
// web/src/lib/api/types.ts
export interface Project {
  id: string
  vault_id?: string | null
  vault_name?: string
  name: string
  note_count: number
  session_count?: number
  due_count?: number
}

export interface Vault {
  id: string
  name: string
  created_at: string
  updated_at: string
}

export interface HomeResponse {
  projects: Project[]
  due_count?: number
  generated_at: string
}

// web/src/lib/stores/shell-context.ts
export type ShellContext =
  | { kind: 'global' }
  | { kind: 'vault'; vaultId: string; vaultName: string }

// web/src/lib/api/client.ts
export interface APIClient {
  get<T>(path: string): Promise<T>
  post<T>(path: string, body: unknown): Promise<T>
}
export const api: APIClient

// web/src/lib/router.ts
export function navigate(hash: string): void
```

---

### Task 20: Pure project context and search helpers

**Files:**
- Create: `web/src/lib/catalog.ts`
- Test: `web/src/lib/catalog.test.ts`

**Interfaces:**
- Consumes: `Project` and `ShellContext` contracts above.
- Produces: `isUnfiled(p)`, `filterProjectsByContext(projects, ctx)`, and `filterByQuery(items, query)` with the exact signatures below.

- [ ] **Step 1: Write the failing unit tests**

```ts
// web/src/lib/catalog.test.ts
import { describe, expect, it } from 'vitest'
import type { Project } from './api/types'
import { filterByQuery, filterProjectsByContext, isUnfiled } from './catalog'

const projects: Project[] = [
  { id: 'p0', name: 'Inbox', note_count: 0 },
  { id: 'p1', name: 'Loose notes', vault_id: '', note_count: 2 },
  { id: 'p2', name: 'Training Plan', vault_id: 'health', vault_name: 'HEALTH', note_count: 4 },
  { id: 'p3', name: 'Budget', vault_id: 'finance', vault_name: 'FINANCE', note_count: 1 },
]

describe('catalog helpers', () => {
  it('treats missing, null, and empty vault IDs as unfiled', () => {
    expect(isUnfiled({})).toBe(true)
    expect(isUnfiled({ vault_id: null })).toBe(true)
    expect(isUnfiled({ vault_id: '' })).toBe(true)
    expect(isUnfiled({ vault_id: 'health' })).toBe(false)
  })

  it('returns only unfiled projects in global context', () => {
    expect(filterProjectsByContext(projects, { kind: 'global' }).map((p) => p.id)).toEqual(['p0', 'p1'])
  })

  it('returns only exact vault matches in vault context', () => {
    expect(filterProjectsByContext(projects, { kind: 'vault', vaultId: 'health', vaultName: 'HEALTH' }).map((p) => p.id)).toEqual(['p2'])
  })

  it('trims and matches names case-insensitively without mutating input', () => {
    const result = filterByQuery(projects, '  PLAN ')
    expect(result.map((p) => p.id)).toEqual(['p2'])
    expect(projects).toHaveLength(4)
    expect(filterByQuery(projects, '   ')).toEqual(projects)
  })
})
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd web && npm test -- --run src/lib/catalog.test.ts`

Expected: FAIL because `./catalog` does not exist.

- [ ] **Step 3: Add the minimal pure implementation**

```ts
// web/src/lib/catalog.ts
import type { Project } from './api/types'
import type { ShellContext } from './stores/shell-context'

export function isUnfiled(p: { vault_id?: string | null }): boolean {
  return !p.vault_id
}

export function filterProjectsByContext(projects: Project[], ctx: ShellContext): Project[] {
  return ctx.kind === 'global'
    ? projects.filter(isUnfiled)
    : projects.filter((project) => project.vault_id === ctx.vaultId)
}

export function filterByQuery<T extends { name: string }>(items: T[], query: string): T[] {
  const normalized = query.trim().toLocaleLowerCase()
  if (!normalized) return items
  return items.filter((item) => item.name.toLocaleLowerCase().includes(normalized))
}
```

- [ ] **Step 4: Run the focused test and typecheck**

Run: `cd web && npm test -- --run src/lib/catalog.test.ts && npm run check`

Expected: PASS; Svelte/TypeScript check exits zero.

- [ ] **Step 5: Commit Task 20**

```bash
git add web/src/lib/catalog.ts web/src/lib/catalog.test.ts
git commit -m "feat(ui): add catalog filtering helpers"
```

---

### Task 21: Catalog UI primitives

**Files:**
- Create: `web/src/components/EmptyState.svelte`
- Create: `web/src/components/Skeleton.svelte`
- Create: `web/src/components/SearchField.svelte`
- Create: `web/src/components/Badge.svelte`
- Create: `web/src/components/ProjectCard.svelte`
- Create: `web/src/components/VaultCard.svelte`
- Test: `web/src/components/catalog-components.test.ts`

**Interfaces:**
- Consumes: `Project`, `Vault`, Svelte callback props, and Tailwind tokens established by shell work.
- Produces: `EmptyState { title, description, actionLabel, onaction }`, `Skeleton { class? }`, bindable `SearchField { value, label? }`, `Badge { text }`, `ProjectCard { project, onclick }`, and `VaultCard { vault, projectCount, onclick }`.

- [ ] **Step 1: Write failing component tests**

```ts
// web/src/components/catalog-components.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { describe, expect, it, vi } from 'vitest'
import EmptyState from './EmptyState.svelte'
import ProjectCard from './ProjectCard.svelte'
import SearchField from './SearchField.svelte'
import VaultCard from './VaultCard.svelte'

describe('catalog components', () => {
  it('renders an actionable empty state', async () => {
    const onaction = vi.fn()
    render(EmptyState, { title: 'No projects yet', description: 'Create your first project.', actionLabel: 'New project', onaction })
    await fireEvent.click(screen.getByRole('button', { name: 'New project' }))
    expect(onaction).toHaveBeenCalledOnce()
  })

  it('labels and updates search input', async () => {
    render(SearchField, { value: '', label: 'Search vaults' })
    await fireEvent.input(screen.getByRole('searchbox', { name: 'Search vaults' }), { target: { value: 'health' } })
    expect(screen.getByRole<HTMLInputElement>('searchbox').value).toBe('health')
  })

  it('shows vault name and project metrics on a vaulted project', () => {
    render(ProjectCard, { project: { id: 'p1', name: 'Training', vault_id: 'v1', vault_name: 'HEALTH', note_count: 3, session_count: 2, due_count: 1 }, onclick: vi.fn() })
    expect(screen.getByText('HEALTH')).toBeInTheDocument()
    expect(screen.getByText('3 notes')).toBeInTheDocument()
    expect(screen.getByText('2 sessions')).toBeInTheDocument()
    expect(screen.getByText('1 due')).toBeInTheDocument()
  })

  it('does not invent a badge for an unfiled project', () => {
    render(ProjectCard, { project: { id: 'p0', name: 'Inbox', vault_id: null, note_count: 0 }, onclick: vi.fn() })
    expect(screen.queryByText('Unfiled')).not.toBeInTheDocument()
  })

  it('renders a vault card as a named button with project count', async () => {
    const onclick = vi.fn()
    render(VaultCard, { vault: { id: 'v1', name: 'HEALTH', created_at: '', updated_at: '' }, projectCount: 4, onclick })
    await fireEvent.click(screen.getByRole('button', { name: /enter health vault/i }))
    expect(screen.getByText('4 projects')).toBeInTheDocument()
    expect(onclick).toHaveBeenCalledOnce()
  })
})
```

- [ ] **Step 2: Run the component test and verify RED**

Run: `cd web && npm test -- --run src/components/catalog-components.test.ts`

Expected: FAIL because the six components do not exist.

- [ ] **Step 3: Implement the primitives**

```svelte
<!-- web/src/components/EmptyState.svelte -->
<script lang="ts">
  let { title, description, actionLabel, onaction }: { title: string; description: string; actionLabel: string; onaction: () => void } = $props()
</script>
<section class="rounded-xl border border-dashed border-slate-300 bg-white px-6 py-12 text-center">
  <h2 class="text-base font-semibold text-slate-950">{title}</h2>
  <p class="mx-auto mt-2 max-w-md text-sm text-slate-600">{description}</p>
  <button class="mt-5 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2" type="button" onclick={onaction}>{actionLabel}</button>
</section>
```

```svelte
<!-- web/src/components/Skeleton.svelte -->
<script lang="ts">
  let { class: className = '' }: { class?: string } = $props()
</script>
<div aria-hidden="true" class={`animate-pulse rounded-lg bg-slate-200 ${className}`}></div>
```

```svelte
<!-- web/src/components/SearchField.svelte -->
<script lang="ts">
  let { value = $bindable(''), label = 'Search' }: { value?: string; label?: string } = $props()
</script>
<label class="relative block w-full sm:max-w-xs">
  <span class="sr-only">{label}</span>
  <input type="search" {value} oninput={(event) => value = event.currentTarget.value} aria-label={label} placeholder={label} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200" />
</label>
```

```svelte
<!-- web/src/components/Badge.svelte -->
<script lang="ts">let { text }: { text: string } = $props()</script>
<span class="inline-flex rounded-full bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700">{text}</span>
```

```svelte
<!-- web/src/components/ProjectCard.svelte -->
<script lang="ts">
  import type { Project } from '../lib/api/types'
  import Badge from './Badge.svelte'
  let { project, onclick }: { project: Project; onclick: () => void } = $props()
</script>
<button type="button" {onclick} aria-label={`Open ${project.name}`} class="w-full rounded-xl border border-slate-200 bg-white p-5 text-left hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500">
  <div class="flex items-start justify-between gap-3"><h2 class="font-semibold text-slate-950">{project.name}</h2>{#if project.vault_id && project.vault_name}<Badge text={project.vault_name} />{/if}</div>
  <div class="mt-5 flex flex-wrap gap-3 text-xs text-slate-500">
    <span>{project.note_count} {project.note_count === 1 ? 'note' : 'notes'}</span>
    {#if project.session_count !== undefined}<span>{project.session_count} {project.session_count === 1 ? 'session' : 'sessions'}</span>{/if}
    {#if project.due_count !== undefined}<span>{project.due_count} due</span>{/if}
  </div>
</button>
```

```svelte
<!-- web/src/components/VaultCard.svelte -->
<script lang="ts">
  import type { Vault } from '../lib/api/types'
  let { vault, projectCount, onclick }: { vault: Vault; projectCount: number; onclick: () => void } = $props()
</script>
<button type="button" {onclick} aria-label={`Enter ${vault.name} vault`} class="w-full rounded-xl border border-slate-200 bg-white p-5 text-left hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500">
  <h2 class="font-semibold text-slate-950">{vault.name}</h2>
  <p class="mt-5 text-xs text-slate-500">{projectCount} {projectCount === 1 ? 'project' : 'projects'}</p>
</button>
```

- [ ] **Step 4: Run tests and checks**

Run: `cd web && npm test -- --run src/components/catalog-components.test.ts && npm run check`

Expected: PASS with no accessibility or TypeScript diagnostics.

- [ ] **Step 5: Commit Task 21**

```bash
git add web/src/components
git commit -m "feat(ui): add catalog card primitives"
```

---

### Task 22: Global Home dashboard

**Files:**
- Create: `web/src/routes/HomePage.svelte`
- Test: `web/src/routes/HomePage.test.ts`

**Interfaces:**
- Consumes: `api.get<HomeResponse>('/api/v1/home')`, `isUnfiled`, `ProjectCard`, `EmptyState`, `Skeleton`, and `navigate`.
- Produces: a route component with no props; quick actions navigate to `#/projects` and `#/vaults`, and recent cards navigate to `#/projects/:id`.

- [ ] **Step 1: Write the failing route test**

```ts
// web/src/routes/HomePage.test.ts
import { render, screen, waitFor } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import HomePage from './HomePage.svelte'
import { api } from '../lib/api/client'

vi.mock('../lib/api/client', () => ({ api: { get: vi.fn() } }))

describe('HomePage', () => {
  beforeEach(() => vi.mocked(api.get).mockReset())

  it('shows due summary and only recent unfiled projects', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '2026-08-19T00:00:00Z', due_count: 3, projects: [
      { id: 'loose', name: 'Inbox', note_count: 1 },
      { id: 'vaulted', name: 'Training', vault_id: 'health', vault_name: 'HEALTH', note_count: 2 },
    ] })
    render(HomePage)
    expect(await screen.findByText('3 items due')).toBeInTheDocument()
    expect(screen.getByText('Inbox')).toBeInTheDocument()
    expect(screen.queryByText('Training')).not.toBeInTheDocument()
  })

  it('is friendly when no projects or reviews are due', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '2026-08-19T00:00:00Z', projects: [] })
    render(HomePage)
    await waitFor(() => expect(screen.getByText('You’re all caught up')).toBeInTheDocument())
    expect(screen.getByText('No unfiled projects yet')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd web && npm test -- --run src/routes/HomePage.test.ts`

Expected: FAIL because `HomePage.svelte` does not exist.

- [ ] **Step 3: Implement the dashboard**

```svelte
<!-- web/src/routes/HomePage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import ProjectCard from '../components/ProjectCard.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Project } from '../lib/api/types'
  import { isUnfiled } from '../lib/catalog'
  import { navigate } from '../lib/router'
  let loading = $state(true), error = $state(''), dueCount = $state(0), projects = $state<Project[]>([])
  onMount(async () => {
    try { const data = await api.get<HomeResponse>('/api/v1/home'); dueCount = data.due_count ?? 0; projects = data.projects.filter(isUnfiled).slice(0, 6) }
    catch (cause) { error = cause instanceof Error ? cause.message : 'Could not load your dashboard.' }
    finally { loading = false }
  })
</script>
<svelte:head><title>Home · Personal Agent</title></svelte:head>
<div class="space-y-8">
  <header><p class="text-sm text-slate-500">Global desk</p><h1 class="text-2xl font-semibold text-slate-950">Home</h1></header>
  <section aria-label="Quick actions" class="flex flex-wrap gap-3">
    <button class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white" onclick={() => navigate('#/projects')}>New project</button>
    <button class="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium" onclick={() => navigate('#/vaults')}>New vault</button>
  </section>
  {#if error}<p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>{/if}
  {#if loading}
    <div class="grid gap-4 md:grid-cols-3"><Skeleton class="h-28" /><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
  {:else}
    <section class="grid gap-4 md:grid-cols-2">
      <div class="rounded-xl border border-slate-200 bg-white p-5"><p class="text-sm text-slate-500">Review</p><p class="mt-2 text-xl font-semibold">{dueCount ? `${dueCount} ${dueCount === 1 ? 'item' : 'items'} due` : 'You’re all caught up'}</p></div>
      <div class="rounded-xl border border-slate-200 bg-white p-5"><p class="text-sm text-slate-500">Unfiled projects</p><p class="mt-2 text-xl font-semibold">{projects.length}</p></div>
    </section>
    <section class="space-y-4"><div class="flex items-center justify-between"><h2 class="text-lg font-semibold">Recent projects</h2><button class="text-sm font-medium text-indigo-700" onclick={() => navigate('#/projects')}>View all</button></div>
      {#if projects.length}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{#each projects as project (project.id)}<ProjectCard {project} onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)} />{/each}</div>
      {:else}<EmptyState title="No unfiled projects yet" description="Create a project on your global desk, or organize work inside a vault." actionLabel="New project" onaction={() => navigate('#/projects')} />{/if}
    </section>
  {/if}
</div>
```

- [ ] **Step 4: Run focused tests and checks**

Run: `cd web && npm test -- --run src/routes/HomePage.test.ts && npm run check`

Expected: PASS.

- [ ] **Step 5: Commit Task 22**

```bash
git add web/src/routes/HomePage.svelte web/src/routes/HomePage.test.ts
git commit -m "feat(ui): add global home dashboard"
```

---

### Task 23: Global unfiled Projects grid

**Files:**
- Create: `web/src/routes/ProjectsPage.svelte`
- Test: `web/src/routes/ProjectsPage.test.ts`

**Interfaces:**
- Consumes: `/api/v1/home`, `POST /api/v1/projects` with `{ name, vault_id: null }`, catalog helpers/primitives, and `navigate`.
- Produces: global projects route with client search and an inline create form; successful creation navigates to `#/projects/:id`.

- [ ] **Step 1: Write failing behavior tests**

```ts
// web/src/routes/ProjectsPage.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectsPage from './ProjectsPage.svelte'
import { api } from '../lib/api/client'
vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } }))

describe('ProjectsPage', () => {
  beforeEach(() => vi.clearAllMocks())
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
  it('creates an unfiled project', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [] })
    vi.mocked(api.post).mockResolvedValue({ id: 'new', name: 'Inbox', vault_id: null, note_count: 0 })
    render(ProjectsPage)
    await fireEvent.click(await screen.findByRole('button', { name: 'New project' }))
    await fireEvent.input(screen.getByLabelText('Project name'), { target: { value: 'Inbox' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Create project' }))
    expect(api.post).toHaveBeenCalledWith('/api/v1/projects', { name: 'Inbox', vault_id: null })
  })
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd web && npm test -- --run src/routes/ProjectsPage.test.ts`

Expected: FAIL because the route component is absent.

- [ ] **Step 3: Implement the grid, search, empty states, and create form**

```svelte
<!-- web/src/routes/ProjectsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'; import EmptyState from '../components/EmptyState.svelte'; import ProjectCard from '../components/ProjectCard.svelte'; import SearchField from '../components/SearchField.svelte'; import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'; import type { HomeResponse, Project } from '../lib/api/types'; import { filterByQuery, isUnfiled } from '../lib/catalog'; import { navigate } from '../lib/router'
  let projects = $state<Project[]>([]), query = $state(''), loading = $state(true), creating = $state(false), saving = $state(false), name = $state(''), error = $state('')
  let visible = $derived(filterByQuery(projects, query))
  onMount(async () => { try { projects = (await api.get<HomeResponse>('/api/v1/home')).projects.filter(isUnfiled) } catch (e) { error = e instanceof Error ? e.message : 'Could not load projects.' } finally { loading = false } })
  async function createProject() { const clean = name.trim(); if (!clean) return; saving = true; error = ''; try { const project = await api.post<Project>('/api/v1/projects', { name: clean, vault_id: null }); navigate(`#/projects/${encodeURIComponent(project.id)}`) } catch (e) { error = e instanceof Error ? e.message : 'Could not create project.' } finally { saving = false } }
</script>
<svelte:head><title>Projects · Personal Agent</title></svelte:head>
<div class="space-y-6"><header class="flex flex-wrap items-end justify-between gap-4"><div><p class="text-sm text-slate-500">Global desk</p><h1 class="text-2xl font-semibold">Projects</h1></div><button class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white" onclick={() => creating = true}>New project</button></header>
  <SearchField bind:value={query} label="Search projects" />
  {#if error}<p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>{/if}
  {#if creating}<form class="flex max-w-lg gap-2 rounded-xl border bg-white p-4" onsubmit={(e) => { e.preventDefault(); createProject() }}><label class="flex-1"><span class="text-sm font-medium">Project name</span><input class="mt-1 w-full rounded-md border px-3 py-2" bind:value={name} /></label><button disabled={saving || !name.trim()} class="self-end rounded-md bg-indigo-600 px-4 py-2 text-sm text-white">Create project</button></form>{/if}
  {#if loading}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"><Skeleton class="h-32" /><Skeleton class="h-32" /></div>
  {:else if visible.length}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{#each visible as project (project.id)}<ProjectCard {project} onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)} />{/each}</div>
  {:else if query.trim()}<EmptyState title="No matching projects" description="Try a different project name." actionLabel="Clear search" onaction={() => query = ''} />
  {:else}<EmptyState title="No unfiled projects yet" description="Create your first project on the global desk." actionLabel="New project" onaction={() => creating = true} />{/if}
</div>
```

- [ ] **Step 4: Run tests and checks**

Run: `cd web && npm test -- --run src/routes/ProjectsPage.test.ts && npm run check`

Expected: PASS.

- [ ] **Step 5: Commit Task 23**

```bash
git add web/src/routes/ProjectsPage.svelte web/src/routes/ProjectsPage.test.ts
git commit -m "feat(ui): add unfiled projects grid"
```

---

### Task 24: Vaults grid, creation, and enter navigation

**Files:**
- Create: `web/src/routes/VaultsPage.svelte`
- Test: `web/src/routes/VaultsPage.test.ts`

**Interfaces:**
- Consumes: `GET/POST /api/v1/vaults`, `/api/v1/home` for project counts, catalog primitives, and `navigate`.
- Produces: vault catalog route; card click and successful create both navigate to `#/vaults/:vaultId`.

- [ ] **Step 1: Write failing route tests**

```ts
// web/src/routes/VaultsPage.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VaultsPage from './VaultsPage.svelte'; import { api } from '../lib/api/client'; import { navigate } from '../lib/router'
vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } })); vi.mock('../lib/router', () => ({ navigate: vi.fn() }))
describe('VaultsPage', () => {
  beforeEach(() => vi.clearAllMocks())
  it('searches vaults, shows project count, and enters a vault', async () => {
    vi.mocked(api.get).mockImplementation(async (path) => path === '/api/v1/vaults' ? [{ id: 'v1', name: 'HEALTH', created_at: '', updated_at: '' }] : { projects: [{ id: 'p1', name: 'Training', vault_id: 'v1', note_count: 0 }], generated_at: '' })
    render(VaultsPage); expect(await screen.findByText('1 project')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: /enter health vault/i })); expect(navigate).toHaveBeenCalledWith('#/vaults/v1')
  })
  it('creates and enters a vault', async () => {
    vi.mocked(api.get).mockResolvedValueOnce([]).mockResolvedValueOnce({ projects: [], generated_at: '' })
    vi.mocked(api.post).mockResolvedValue({ id: 'v2', name: 'WORK', created_at: '', updated_at: '' })
    render(VaultsPage); await fireEvent.click(await screen.findByRole('button', { name: 'New vault' })); await fireEvent.input(screen.getByLabelText('Vault name'), { target: { value: 'WORK' } }); await fireEvent.click(screen.getByRole('button', { name: 'Create vault' }))
    expect(api.post).toHaveBeenCalledWith('/api/v1/vaults', { name: 'WORK' }); expect(navigate).toHaveBeenCalledWith('#/vaults/v2')
  })
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd web && npm test -- --run src/routes/VaultsPage.test.ts`

Expected: FAIL because `VaultsPage.svelte` is absent.

- [ ] **Step 3: Implement vault loading, search, counts, create, and enter**

```svelte
<!-- web/src/routes/VaultsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'; import EmptyState from '../components/EmptyState.svelte'; import SearchField from '../components/SearchField.svelte'; import Skeleton from '../components/Skeleton.svelte'; import VaultCard from '../components/VaultCard.svelte'
  import { api } from '../lib/api/client'; import type { HomeResponse, Vault } from '../lib/api/types'; import { filterByQuery } from '../lib/catalog'; import { navigate } from '../lib/router'
  let vaults = $state<Vault[]>([]), counts = $state<Record<string, number>>({}), query = $state(''), loading = $state(true), creating = $state(false), saving = $state(false), name = $state(''), error = $state('')
  let visible = $derived(filterByQuery(vaults, query))
  onMount(async () => { try { const [listed, home] = await Promise.all([api.get<Vault[]>('/api/v1/vaults'), api.get<HomeResponse>('/api/v1/home')]); vaults = listed; counts = home.projects.reduce<Record<string, number>>((all, p) => { if (p.vault_id) all[p.vault_id] = (all[p.vault_id] ?? 0) + 1; return all }, {}) } catch (e) { error = e instanceof Error ? e.message : 'Could not load vaults.' } finally { loading = false } })
  async function createVault() { const clean = name.trim(); if (!clean) return; saving = true; error = ''; try { const vault = await api.post<Vault>('/api/v1/vaults', { name: clean }); navigate(`#/vaults/${encodeURIComponent(vault.id)}`) } catch (e) { error = e instanceof Error ? e.message : 'Could not create vault.' } finally { saving = false } }
</script>
<svelte:head><title>Vaults · Personal Agent</title></svelte:head>
<div class="space-y-6"><header class="flex flex-wrap items-end justify-between gap-4"><div><p class="text-sm text-slate-500">Global desk</p><h1 class="text-2xl font-semibold">Vaults</h1></div><button class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white" onclick={() => creating = true}>New vault</button></header>
  <SearchField bind:value={query} label="Search vaults" />{#if error}<p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>{/if}
  {#if creating}<form class="flex max-w-lg gap-2 rounded-xl border bg-white p-4" onsubmit={(e) => { e.preventDefault(); createVault() }}><label class="flex-1"><span class="text-sm font-medium">Vault name</span><input class="mt-1 w-full rounded-md border px-3 py-2" bind:value={name} /></label><button disabled={saving || !name.trim()} class="self-end rounded-md bg-indigo-600 px-4 py-2 text-sm text-white">Create vault</button></form>{/if}
  {#if loading}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
  {:else if visible.length}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{#each visible as vault (vault.id)}<VaultCard {vault} projectCount={counts[vault.id] ?? 0} onclick={() => navigate(`#/vaults/${encodeURIComponent(vault.id)}`)} />{/each}</div>
  {:else if query.trim()}<EmptyState title="No matching vaults" description="Try a different vault name." actionLabel="Clear search" onaction={() => query = ''} />
  {:else}<EmptyState title="No vaults yet" description="Create a vault to organize related projects." actionLabel="New vault" onaction={() => creating = true} />{/if}
</div>
```

- [ ] **Step 4: Run tests and checks**

Run: `cd web && npm test -- --run src/routes/VaultsPage.test.ts && npm run check`

Expected: PASS.

- [ ] **Step 5: Commit Task 24**

```bash
git add web/src/routes/VaultsPage.svelte web/src/routes/VaultsPage.test.ts
git commit -m "feat(ui): add searchable vaults grid"
```

---

### Task 25: Register global routes in the app router

**Files:**
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`
- Test: `web/src/App.test.ts`

**Interfaces:**
- Consumes: existing `parseHash()`/route union and authenticated shell from the shell/router task, plus `HomePage`, `ProjectsPage`, and `VaultsPage` from Tasks 22–24.
- Produces: exact matches for `#/home`, `#/projects`, and `#/vaults`; unknown hashes retain the router’s existing fallback behavior. Parameterized `#/vaults/:vaultId` remains owned by the vault-context task.

- [ ] **Step 1: Extend the router integration test first**

```ts
// Add to web/src/App.test.ts
import { render, screen } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'
import App from './App.svelte'

describe('global catalog routes', () => {
  afterEach(() => { window.location.hash = '' })
  for (const [hash, heading] of [['#/home', 'Home'], ['#/projects', 'Projects'], ['#/vaults', 'Vaults']] as const) {
    it(`renders ${hash}`, () => {
      window.location.hash = hash
      render(App)
      expect(screen.getByRole('heading', { level: 1, name: heading })).toBeInTheDocument()
    })
  }
})
```

- [ ] **Step 2: Run the integration test and verify RED**

Run: `cd web && npm test -- --run src/App.test.ts`

Expected: FAIL because at least one global route does not render its new page.

- [ ] **Step 3: Add route variants and page selection**

Add these exact variants to the existing router union and exact-hash branch in `web/src/lib/router.ts` (preserve all routes implemented by other tasks):

```ts
export type GlobalRoute =
  | { name: 'home' }
  | { name: 'projects' }
  | { name: 'vaults' }

export function parseGlobalHash(hash: string): GlobalRoute | null {
  const path = hash.replace(/^#/, '').replace(/\/$/, '') || '/home'
  if (path === '/home') return { name: 'home' }
  if (path === '/projects') return { name: 'projects' }
  if (path === '/vaults') return { name: 'vaults' }
  return null
}
```

Call `parseGlobalHash(hash)` from the existing `parseHash` before its unknown-route fallback, but after more-specific project/vault parameter routes. Then add imports and these branches to the existing authenticated content switch in `web/src/App.svelte` without replacing the shell:

```svelte
<script lang="ts">
  import HomePage from './routes/HomePage.svelte'
  import ProjectsPage from './routes/ProjectsPage.svelte'
  import VaultsPage from './routes/VaultsPage.svelte'
  // Keep the shell, auth bootstrap, current route state, and other page imports already present.
</script>

{#if route.name === 'home'}
  <HomePage />
{:else if route.name === 'projects'}
  <ProjectsPage />
{:else if route.name === 'vaults'}
  <VaultsPage />
{:else}
  <!-- Keep the existing branches for every other route here. -->
{/if}
```

The comment in the snippet marks existing code that must remain; do not paste it as a runtime fallback or delete other route branches.

- [ ] **Step 4: Run the focused suite, complete frontend suite, and check**

Run: `cd web && npm test -- --run src/App.test.ts src/routes/HomePage.test.ts src/routes/ProjectsPage.test.ts src/routes/VaultsPage.test.ts && npm test -- --run && npm run check`

Expected: all tests PASS and check exits zero.

- [ ] **Step 5: Commit Task 25**

```bash
git add web/src/lib/router.ts web/src/App.svelte web/src/App.test.ts
git commit -m "feat(ui): wire global dashboard routes"
```
