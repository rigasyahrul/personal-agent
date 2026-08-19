# Vault Context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a vault a URL-driven working context with scoped home, projects, sessions, review, and breadcrumbs.

**Architecture:** The hash route is the source of truth. `resolveShellContext` derives global/vault mode directly from vault routes and resolves project routes through `project.vault_id`; pages never retain an independent “selected vault.” Vault pages load the global project collection once and use pure filtering/aggregation helpers because sessions are available only per project and review `scope=all` is filtered by project membership.

**Tech Stack:** Svelte 5 runes, TypeScript, Vite, Tailwind CSS, Vitest, Testing Library

## Global Constraints

- Enter vault with `#/vaults/:id`; leave vault with `#/home`.
- A vaulted project deep link derives context from `project.vault_id`; an unfiled project uses global context.
- Project `vault_id` is immutable after creation; vault create UI must send and visually lock it.
- Sessions exist only at `GET/POST /api/v1/projects/{id}/sessions`; there is no vault sessions endpoint.
- Vault review loads `GET /api/v1/review/queue?scope=all` and filters items by vault project IDs.
- Keep hash routing, loading skeletons, dedicated empty states, inline errors, and visible keyboard focus.
- Do not add backend endpoints or product behavior.

---

### Task 30: Resolve URL-Driven Vault Shell Context

**Files:**
- Create: `web/src/lib/stores/shell-context.ts`
- Create: `web/src/lib/stores/shell-context.test.ts`
- Modify: `web/src/App.svelte`
- Modify: `web/src/shell/Sidebar.svelte`
- Test: `web/src/shell/Sidebar.test.ts`

**Interfaces:**
- Consumes: `Route` from `web/src/lib/router.ts`, `Project`/`Vault` and `api.getProject(id)`/`api.listVaults()` from `web/src/lib/api`.
- Produces: `type ShellContext = { kind: 'global' } | { kind: 'vault'; vault: Vault }` and `resolveShellContext(route, deps): Promise<ShellContext>`.

- [ ] **Step 1: Write failing context and sidebar tests**

```ts
it.each(['vault-home', 'vault-projects', 'vault-sessions', 'vault-review'])('uses route vault for %s', async name => {
  const context = await resolveShellContext({ name, vaultId: 'v1' } as Route, deps)
  expect(context).toEqual({ kind: 'vault', vault: { id: 'v1', name: 'HEALTH' } })
})

it('derives a project deep-link context from project.vault_id', async () => {
  deps.getProject.mockResolvedValue({ id: 'p1', name: 'Sleep', vault_id: 'v1', vault_name: 'HEALTH' })
  expect(await resolveShellContext({ name: 'project-notes', projectId: 'p1' }, deps))
    .toEqual({ kind: 'vault', vault: { id: 'v1', name: 'HEALTH' } })
})

it('renders replacement vault navigation and leaves to global home', async () => {
  render(Sidebar, { context: { kind: 'vault', vault: { id: 'v1', name: 'HEALTH' } }, route })
  expect(screen.getByText('HEALTH')).toBeVisible()
  expect(screen.queryByText('Vaults')).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: /leave vault/i })).toHaveAttribute('href', '#/home')
})
```

- [ ] **Step 2: Run tests and confirm the new module/behavior is absent**

Run: `rtk npm run test -- --run web/src/lib/stores/shell-context.test.ts web/src/shell/Sidebar.test.ts`
Expected: FAIL because `resolveShellContext` and vault-mode navigation do not exist.

- [ ] **Step 3: Implement resolution and replacement navigation**

```ts
export async function resolveShellContext(route: Route, deps: ContextDeps): Promise<ShellContext> {
  if ('vaultId' in route) {
    const vault = (await deps.listVaults()).find(candidate => candidate.id === route.vaultId)
    if (!vault) throw new Error('Vault not found')
    return { kind: 'vault', vault }
  }
  if ('projectId' in route) {
    const project = await deps.getProject(route.projectId)
    if (project.vault_id) return { kind: 'vault', vault: { id: project.vault_id, name: project.vault_name } }
  }
  return { kind: 'global' }
}
```

In `App.svelte`, resolve context after every parsed route, discard stale async resolutions with a generation counter, show the existing shell skeleton while resolving, and pass the result to `Sidebar`. In vault mode render only Home (`#/vaults/{id}`), Projects, Sessions, Review, Settings, plus `Leave vault` → `#/home`; close the mobile drawer after navigation.

- [ ] **Step 4: Run focused tests**

Run: `rtk npm run test -- --run web/src/lib/stores/shell-context.test.ts web/src/shell/Sidebar.test.ts`
Expected: PASS, including vaulted and unfiled project deep links.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/lib/stores/shell-context.ts web/src/lib/stores/shell-context.test.ts web/src/App.svelte web/src/shell/Sidebar.svelte web/src/shell/Sidebar.test.ts
rtk git commit -m "feat(web): derive vault shell context from routes"
```

### Task 31: Build the Vault Home Dashboard

**Files:**
- Create: `web/src/routes/VaultHomePage.svelte`
- Create: `web/src/routes/VaultHomePage.test.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: `vaultId: string`, `api.listProjects()`, and `api.getReviewQueue('all')`.
- Produces: route `{ name: 'vault-home'; vaultId: string }` for `#/vaults/:vaultId` and dashboard links to vault projects/sessions/review.

- [ ] **Step 1: Write failing dashboard test**

```ts
it('shows only vault summary data and useful actions', async () => {
  api.listProjects.mockResolvedValue([vaultProject, unfiledProject])
  api.getReviewQueue.mockResolvedValue({ items: [{ id: 'r1', project_id: 'p-v' }], caught_up: false })
  render(VaultHomePage, { vaultId: 'v1' })
  expect(await screen.findByText('1 project')).toBeVisible()
  expect(screen.getByText('1 due')).toBeVisible()
  expect(screen.queryByText(unfiledProject.name)).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: /new project/i })).toHaveAttribute('href', '#/vaults/v1/projects?new=1')
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/routes/VaultHomePage.test.ts`
Expected: FAIL because the route and page are missing.

- [ ] **Step 3: Implement the scoped dashboard**

Load projects and queue concurrently, calculate project/session/note/due totals from only matching projects/items, and render quick actions, recent project cards, skeletons, a no-project empty state, and an inline retry alert. Do not render the full catalog.

```ts
const [allProjects, queue] = await Promise.all([api.listProjects(), api.getReviewQueue('all')])
projects = allProjects.filter(project => project.vault_id === vaultId)
const ids = new Set(projects.map(project => project.id))
due = queue.items.filter(item => ids.has(item.project_id)).length
```

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/routes/VaultHomePage.test.ts web/src/lib/router.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/routes/VaultHomePage.svelte web/src/routes/VaultHomePage.test.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): add vault dashboard"
```

### Task 32: Add Vault Projects and Locked Project Creation

**Files:**
- Create: `web/src/lib/vault-scope.ts`
- Create: `web/src/lib/vault-scope.test.ts`
- Create: `web/src/routes/VaultProjectsPage.svelte`
- Create: `web/src/routes/VaultProjectsPage.test.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Produces: `filterVaultProjects(projects: Project[], vaultId: string): Project[]` and `createVaultProjectInput(name: string, vaultId: string): { name: string; vault_id: string }`.
- Consumes: shared `ProjectGrid`, `ProjectCreateDialog`, and `api.createProject` established by the global projects plan.

- [ ] **Step 1: Write failing helper and component tests**

```ts
expect(filterVaultProjects([vaultProject, otherVaultProject, unfiledProject], 'v1')).toEqual([vaultProject])
expect(createVaultProjectInput(' Sleep ', 'v1')).toEqual({ name: 'Sleep', vault_id: 'v1' })

it('locks the vault and submits it even when the dialog is reopened', async () => {
  render(VaultProjectsPage, { vaultId: 'v1', vaultName: 'HEALTH' })
  await user.click(await screen.findByRole('button', { name: /new project/i }))
  expect(screen.getByLabelText('Vault')).toHaveValue('HEALTH')
  expect(screen.getByLabelText('Vault')).toBeDisabled()
  await user.type(screen.getByLabelText('Project name'), 'Sleep')
  await user.click(screen.getByRole('button', { name: 'Create project' }))
  expect(api.createProject).toHaveBeenCalledWith({ name: 'Sleep', vault_id: 'v1' })
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/lib/vault-scope.test.ts web/src/routes/VaultProjectsPage.test.ts`
Expected: FAIL because vault filtering and locked creation are absent.

- [ ] **Step 3: Implement grid, search, empty state, and locked create**

Use `#/vaults/:vaultId/projects`, filter before client-side name search, preserve the vault ID in component state as an immutable prop (never derive it from form input), and navigate a created project to `#/projects/{encodedId}`. Show no fake “Unfiled” badge.

```ts
export const filterVaultProjects = (projects: Project[], vaultId: string) =>
  projects.filter(project => project.vault_id === vaultId)

export function createVaultProjectInput(name: string, vaultId: string) {
  return { name: name.trim(), vault_id: vaultId }
}
```

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/lib/vault-scope.test.ts web/src/routes/VaultProjectsPage.test.ts web/src/lib/router.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/lib/vault-scope.ts web/src/lib/vault-scope.test.ts web/src/routes/VaultProjectsPage.svelte web/src/routes/VaultProjectsPage.test.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): add locked vault project creation"
```

### Task 33: Aggregate Vault Sessions Through Project Endpoints

**Files:**
- Create: `web/src/lib/vault-sessions.ts`
- Create: `web/src/lib/vault-sessions.test.ts`
- Create: `web/src/routes/VaultSessionsPage.svelte`
- Create: `web/src/routes/VaultSessionsPage.test.ts`
- Modify: `web/src/lib/api/types.ts`
- Modify: `web/src/lib/api/index.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Produces: `type VaultSession = Session & { project_id: string; project_name: string }` and `loadVaultSessions(vaultId, api): Promise<{ projects: Project[]; sessions: VaultSession[]; failures: string[] }>`.
- Consumes only `listProjects()` and `listProjectSessions(projectId)` (`GET /api/v1/projects/{id}/sessions`).

- [ ] **Step 1: Write failing aggregation tests**

```ts
it('calls sessions once per vault project and annotates results', async () => {
  api.listProjects.mockResolvedValue([projectA, projectB, unfiledProject])
  api.listProjectSessions.mockImplementation(async id => id === 'a' ? [sessionA] : [sessionB])
  const result = await loadVaultSessions('v1', api)
  expect(api.listProjectSessions.mock.calls.map(([id]) => id).sort()).toEqual(['a', 'b'])
  expect(result.sessions).toEqual([
    { ...sessionA, project_id: 'a', project_name: projectA.name },
    { ...sessionB, project_id: 'b', project_name: projectB.name },
  ])
})

it('keeps successful projects and reports a partial failure', async () => {
  api.listProjectSessions.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce([sessionB])
  expect((await loadVaultSessions('v1', api)).failures).toEqual([projectA.name])
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/lib/vault-sessions.test.ts web/src/routes/VaultSessionsPage.test.ts`
Expected: FAIL because no aggregate loader/page exists.

- [ ] **Step 3: Implement bounded client-side aggregation and picker**

Use `Promise.allSettled` over vault projects. Render a project picker plus a combined, project-labelled session list; selecting “New session” navigates to that project's `#/projects/:id/sessions`. If there are no vault projects, show “Create a project first”; if projects have no sessions, show a session empty state. A partial request failure must retain successful rows and show an inline warning listing failed project names.

```ts
const settled = await Promise.allSettled(projects.map(project => api.listProjectSessions(project.id)))
settled.forEach((result, index) => {
  const project = projects[index]
  if (result.status === 'rejected') failures.push(project.name)
  else sessions.push(...result.value.map(session => ({ ...session, project_id: project.id, project_name: project.name })))
})
```

- [ ] **Step 4: Verify green and endpoint discipline**

Run: `rtk npm run test -- --run web/src/lib/vault-sessions.test.ts web/src/routes/VaultSessionsPage.test.ts`
Expected: PASS; request assertions contain no `/vaults/{id}/sessions` API call.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/lib/vault-sessions.ts web/src/lib/vault-sessions.test.ts web/src/routes/VaultSessionsPage.svelte web/src/routes/VaultSessionsPage.test.ts web/src/lib/api/types.ts web/src/lib/api/index.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): aggregate sessions across vault projects"
```

### Task 34: Filter the Vault Review Queue Client-Side

**Files:**
- Create: `web/src/lib/review/vault-filter.ts`
- Create: `web/src/lib/review/vault-filter.test.ts`
- Create: `web/src/routes/VaultReviewPage.svelte`
- Create: `web/src/routes/VaultReviewPage.test.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Produces: `filterQueueByProjectIds(queue: ReviewQueue, projectIds: ReadonlySet<string>): ReviewQueue`.
- Consumes: `api.getReviewQueue('all')`, vault projects, and the shared `ReviewRunner` from Task 50.

- [ ] **Step 1: Write failing filter/page tests**

```ts
expect(filterQueueByProjectIds(
  { items: [vaultItem, otherItem], caught_up: false }, new Set(['p-v'])
)).toEqual({ items: [vaultItem], caught_up: false })

it('requests all and passes only vault items to the runner', async () => {
  render(VaultReviewPage, { vaultId: 'v1' })
  expect(api.getReviewQueue).toHaveBeenCalledWith('all')
  expect(await screen.findByText(vaultItem.prompt)).toBeVisible()
  expect(screen.queryByText(otherItem.prompt)).not.toBeInTheDocument()
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/lib/review/vault-filter.test.ts web/src/routes/VaultReviewPage.test.ts`
Expected: FAIL because the filter/page are absent.

- [ ] **Step 3: Implement filtering without falsifying server state**

Filter `items` by `project_id`; set `caught_up` to `true` when the filtered items are empty so the vault page has a caught-up state even if other projects are due globally. Refresh projects and `scope=all` after rate/suspend/retry callbacks.

```ts
export function filterQueueByProjectIds(queue: ReviewQueue, ids: ReadonlySet<string>): ReviewQueue {
  const items = queue.items.filter(item => ids.has(item.project_id))
  return { ...queue, items, caught_up: items.length === 0 }
}
```

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/lib/review/vault-filter.test.ts web/src/routes/VaultReviewPage.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/lib/review/vault-filter.ts web/src/lib/review/vault-filter.test.ts web/src/routes/VaultReviewPage.svelte web/src/routes/VaultReviewPage.test.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): scope review queue to vault projects"
```

### Task 35: Add Context-Aware Breadcrumbs

**Files:**
- Create: `web/src/components/Breadcrumbs.svelte`
- Create: `web/src/components/Breadcrumbs.test.ts`
- Modify: `web/src/routes/ProjectHubPage.svelte`
- Modify: `web/src/routes/NotesPage.svelte`
- Modify: `web/src/routes/ProjectSessionsPage.svelte`
- Modify: `web/src/routes/ProjectReviewPage.svelte`

**Interfaces:**
- Consumes: `project: Project` and optional `leaf: string`.
- Produces: accessible `<nav aria-label="Breadcrumb">`; vaulted path `Vaults / {vault} / {project} [/ leaf]`, unfiled path `Projects / {project} [/ leaf]`.

- [ ] **Step 1: Write failing breadcrumb tests**

```ts
it('links a vaulted project back through its vault', () => {
  render(Breadcrumbs, { project: vaultedProject, leaf: 'Sessions' })
  expect(screen.getByRole('link', { name: 'Vaults' })).toHaveAttribute('href', '#/vaults')
  expect(screen.getByRole('link', { name: 'HEALTH' })).toHaveAttribute('href', '#/vaults/v1')
  expect(screen.getByRole('link', { name: 'Sleep' })).toHaveAttribute('href', '#/projects/p1')
  expect(screen.getByText('Sessions')).toHaveAttribute('aria-current', 'page')
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/components/Breadcrumbs.test.ts`
Expected: FAIL because the component is missing.

- [ ] **Step 3: Implement and place breadcrumbs on every project surface**

Encode all IDs with `encodeURIComponent`, use an ordered list with separators hidden from assistive technology, and truncate long names visually without removing their full accessible text. Ensure project pages use the already-loaded `Project`, avoiding duplicate project requests.

- [ ] **Step 4: Verify focused and vault suites**

Run: `rtk npm run test -- --run web/src/components/Breadcrumbs.test.ts web/src/routes/VaultHomePage.test.ts web/src/routes/VaultProjectsPage.test.ts web/src/routes/VaultSessionsPage.test.ts web/src/routes/VaultReviewPage.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/components/Breadcrumbs.svelte web/src/components/Breadcrumbs.test.ts web/src/routes/ProjectHubPage.svelte web/src/routes/NotesPage.svelte web/src/routes/ProjectSessionsPage.svelte web/src/routes/ProjectReviewPage.svelte
rtk git commit -m "feat(web): add vault-aware project breadcrumbs"
```
