# Phase B2 — Project hub (Claude start + rail) (draft)

> Tasks 4–6. Spec §5. Assembled into master plan.

**Goal:** Replace metric/destination ProjectHubPage with Claude-style main (prompt + composer + session rows below) + default-open ProjectRail (Memory | Files).

---

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
