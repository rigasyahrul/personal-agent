### Task 5: Wire hub rail mode, selected tab, and persistence

**Files:**
- Modify: `web/src/routes/ProjectHubPage.test.ts`
- Modify: `web/src/routes/ProjectHubPage.svelte`

**Interfaces:**
- Consumes from Task 1: `ProjectRailMode`, `ProjectRailTab`, `readProjectRailMode`, `writeProjectRailMode`, `readProjectRailTab`, and `writeProjectRailTab` from `../lib/project-rail-prefs`; storage keys are `pa.projectRail.mode` and `pa.projectRail.tab`, with defaults `open` and `config`.
- Consumes from Tasks 3–4: controlled `ProjectRail` props `tab`, `mode`, `onTabChange`, and `onModeChange`; existing `onOpenFile` behavior remains unchanged.
- Produces: `.project-workspace[data-rail="open|expanded|collapsed"]` as the hub layout hook and immediate state/storage synchronization for rail controls.

- [ ] **Step 1: Add a focused failing hub wiring test**

Add this test inside the existing `describe('ProjectHubPage', ...)` block in `web/src/routes/ProjectHubPage.test.ts`. It proves the default from the preferences helper reaches both the workspace and controlled rail before adding the wider interaction matrix in Task 6.

```ts
  it('wires the default rail preferences into the hub and ProjectRail', async () => {
    render(ProjectHubPage, { props: { projectId: 'p1' } })

    await screen.findByRole('textbox', { name: /message/i })
    expect(document.querySelector('.project-workspace')).toHaveAttribute('data-rail', 'open')
    expect(screen.getByRole('tab', { name: 'Config' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })
```

- [ ] **Step 2: Run the focused test and verify it fails for the missing hub wiring**

Run with Node `>=22 <23` first on `PATH`:

```bash
npm --prefix web test -- --run src/routes/ProjectHubPage.test.ts -t "wires the default rail preferences"
```

Expected: FAIL because `.project-workspace` does not yet have `data-rail="open"` (and, before Tasks 3–4 land, the Config control may also be absent).

- [ ] **Step 3: Import the preference contract and initialize controlled state**

In the `<script lang="ts">` imports of `web/src/routes/ProjectHubPage.svelte`, add:

```ts
  import {
    readProjectRailMode,
    readProjectRailTab,
    writeProjectRailMode,
    writeProjectRailTab,
    type ProjectRailMode,
    type ProjectRailTab,
  } from '../lib/project-rail-prefs'
```

Immediately after the existing `activeSession` state, add the exact storage-backed state required by the contract:

```ts
  let railMode = $state<ProjectRailMode>(readProjectRailMode(localStorage))
  let railTab = $state<ProjectRailTab>(readProjectRailTab(localStorage))
```

- [ ] **Step 4: Put the mode on the hub root and control ProjectRail**

Change only the workspace opening tag:

```svelte
  <div class="project-workspace" data-rail={railMode}>
```

Then add the controlled props and persistence callbacks to the existing `ProjectRail` invocation. Preserve `projectId`, session/workspace props, and the complete existing `onOpenFile` callback unchanged:

```svelte
      <ProjectRail
        {projectId}
        sessionId={activeSession?.id}
        workspaceFilesEnabled={workspaceFilesEnabled}
        tab={railTab}
        mode={railMode}
        onTabChange={(tab) => {
          writeProjectRailTab(localStorage, tab)
          railTab = tab
        }}
        onModeChange={(mode) => {
          writeProjectRailMode(localStorage, mode)
          railMode = mode
        }}
        onOpenFile={(path, meta) => {
          if (!activeSession) return
          openFileRequest = {
            path,
            source: meta?.source ?? 'workspace',
            noteId: meta?.noteId,
          }
        }}
      />
```

Do not alter `openSession`, `closeSession`, `openFileRequest`, or project-note/workspace routing while adding rail state.

- [ ] **Step 5: Run the focused test and the preference tests**

```bash
npm --prefix web test -- --run src/routes/ProjectHubPage.test.ts -t "wires the default rail preferences"
npm --prefix web test -- --run src/lib/project-rail-prefs.test.ts
```

Expected: both commands PASS.

- [ ] **Step 6: Commit the hub wiring**

```bash
git add web/src/routes/ProjectHubPage.svelte web/src/routes/ProjectHubPage.test.ts
git commit -m "feat(web): wire project rail modes in hub"
```

### Task 6: Cover hub rail modes, persistence, and session file opening

**Files:**
- Modify: `web/src/routes/ProjectHubPage.test.ts`

**Interfaces:**
- Consumes from Task 5: `.project-workspace[data-rail]`, storage-backed controlled rail state, and unchanged `onOpenFile` bridge.
- Consumes from Tasks 3–4: accessible controls named `Config`, `Files`, `Expand workspace` (or `Exit expanded` while expanded), `Collapse canvas`, and `Show canvas`.
- Produces: integration regression coverage for all three hub modes, localStorage round-tripping, Memory removal, and rail-to-session file opening.

- [ ] **Step 1: Isolate localStorage in every hub test**

Add `localStorage.clear()` as the first statement in the existing `beforeEach`:

```ts
  beforeEach(() => {
    localStorage.clear()
    vi.mocked(api.getProject).mockReset().mockResolvedValue(project)
    vi.mocked(api.listProjectSessions).mockReset().mockResolvedValue([])
    vi.mocked(api.listModels).mockReset().mockResolvedValue(models)
    vi.mocked(api.listProjectNotes).mockReset().mockResolvedValue([])
    vi.mocked(api.createProjectSession).mockReset()
    vi.mocked(api.renameSession).mockReset()
    vi.mocked(api.deleteSession).mockReset()
    vi.mocked(api.sendMessage).mockReset().mockResolvedValue(undefined)
    vi.mocked(api.listMessages).mockReset().mockResolvedValue([])
    vi.mocked(api.currentRun).mockReset().mockResolvedValue(null)
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({ entries: [] })
    vi.mocked(api.workspaceFile).mockReset()
    vi.mocked(api.getProjectNote).mockReset()
  })
```

- [ ] **Step 2: Replace obsolete Memory assertions with the Config control**

In `shows Claude start prompt and no metric destination grid`, replace:

```ts
    expect(screen.getByRole('tab', { name: 'Memory' })).toBeInTheDocument()
```

with:

```ts
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Memory' })).toBeNull()
```

In `opens a session row into chat while keeping the rail` and both rail assertions in `clicking a session row shows chat and Back returns to prompt`, replace each Memory assertion with:

```ts
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
```

- [ ] **Step 3: Add failing mode-transition integration coverage**

Add this complete test inside the existing `describe` block:

```ts
  it('expands, collapses, and restores the project rail canvas', async () => {
    render(ProjectHubPage, { props: { projectId: 'p1' } })

    await screen.findByRole('textbox', { name: /message/i })
    const workspace = document.querySelector('.project-workspace')
    expect(workspace).not.toBeNull()
    expect(workspace).toHaveAttribute('data-rail', 'open')

    await fireEvent.click(screen.getByRole('button', { name: 'Expand workspace' }))
    expect(workspace).toHaveAttribute('data-rail', 'expanded')
    expect(document.querySelector('.project-workspace__main')).not.toBeVisible()

    await fireEvent.click(screen.getByRole('button', { name: 'Collapse canvas' }))
    expect(workspace).toHaveAttribute('data-rail', 'collapsed')
    expect(screen.getByRole('button', { name: 'Show canvas' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Config' })).toBeNull()
    expect(screen.queryByRole('tab', { name: 'Files' })).toBeNull()

    await fireEvent.click(screen.getByRole('button', { name: 'Show canvas' }))
    expect(workspace).toHaveAttribute('data-rail', 'open')
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
  })
```

- [ ] **Step 4: Add failing persistence coverage**

Add this complete test. It checks both hydration and a subsequent write through the canonical mode key:

```ts
  it('hydrates rail mode from localStorage and persists mode changes', async () => {
    localStorage.setItem('pa.projectRail.mode', 'collapsed')

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    await screen.findByRole('textbox', { name: /message/i })
    const workspace = document.querySelector('.project-workspace')
    expect(workspace).toHaveAttribute('data-rail', 'collapsed')
    expect(screen.getByRole('button', { name: 'Show canvas' })).toBeInTheDocument()

    await fireEvent.click(screen.getByRole('button', { name: 'Show canvas' }))
    expect(workspace).toHaveAttribute('data-rail', 'open')
    expect(localStorage.getItem('pa.projectRail.mode')).toBe('open')

    await fireEvent.click(screen.getByRole('button', { name: 'Collapse canvas' }))
    expect(workspace).toHaveAttribute('data-rail', 'collapsed')
    expect(localStorage.getItem('pa.projectRail.mode')).toBe('collapsed')
  })
```

- [ ] **Step 5: Run the new tests and verify red before relying on implementation**

If these tests were added before Task 5 or Tasks 3–4 implementation, run:

```bash
npm --prefix web test -- --run src/routes/ProjectHubPage.test.ts -t "expands, collapses, and restores|hydrates rail mode"
```

Expected before implementation: FAIL due to missing mode controls and/or `data-rail`. Expected after Tasks 3–5: PASS. Do not weaken visibility or accessible-name assertions to make the test green; fix the owning implementation task instead.

- [ ] **Step 6: Keep the existing session file bridge test on the new Files selector**

Retain the complete existing `rail Files open drives SessionChat file tab and hides Show files` setup and assertions. Its interaction must use the icon tab selector:

```ts
    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'notes/a.md' }))

    const fileTab = await screen.findByRole('tab', { name: /a\.md/i })
    expect(fileTab).toHaveAttribute('aria-selected', 'true')
    expect(fileTab).toHaveAttribute('title', 'notes/a.md')
    await waitFor(() => expect(api.getProjectNote).toHaveBeenCalledWith('p1', 'n-a'))
    expect(api.workspaceFile).not.toHaveBeenCalled()
```

This confirms Files remains openable from the rail while a session is open and that project notes still route through `getProjectNote`, not `workspaceFile`.

- [ ] **Step 7: Run the complete hub test file**

```bash
npm --prefix web test -- --run src/routes/ProjectHubPage.test.ts
```

Expected: PASS with no uncaught Svelte/jsdom errors. Confirm the output includes the mode, persistence, and session file-opening tests.

- [ ] **Step 8: Commit the hub integration coverage**

```bash
git add web/src/routes/ProjectHubPage.test.ts
git commit -m "test(web): cover project rail hub modes"
```
