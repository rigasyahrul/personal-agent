### Task 3: ProjectRail icon chrome (Config/Files, no Memory)

**Files:**
- Create: `web/src/components/rail-icons.ts`
- Modify: `web/src/components/ProjectRail.svelte`
- Test: `web/src/components/ProjectRail.test.ts`

**Interfaces:**
- Consumes: `ProjectRailTab` and `ProjectRailMode` from `web/src/lib/project-rail-prefs.ts` (Task 1), plus the existing project-note/workspace APIs and `OpenFileMeta` contract.
- Produces: `railIconPath(name: RailIconName): string`; controlled-or-uncontrolled `tab?: ProjectRailTab`; hub-owned `mode?: ProjectRailMode`; `onTabChange?: (tab: ProjectRailTab) => void`; `onModeChange?: (mode: ProjectRailMode) => void`.

**Screen contract:** In open/expanded modes the rail has an icon toolbar ordered Config, Files, Expand workspace/Exit expanded, Collapse canvas. Config is the uncontrolled default and contains only Instructions (system) plus “Not saved yet — persistence coming later.” Files retains the current loading, error, empty, note-open, workspace-merge, and grant-gating behavior. Memory is absent. Files tree/search redesign is out of scope.

- [ ] **Step 1: Replace the component tests with failing Config/Files chrome and existing Files behavior tests**

Write `web/src/components/ProjectRail.test.ts` as follows (Task 4 adds mode callback tests to this same file):

```ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectRail from './ProjectRail.svelte'
import { api } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      listProjectNotes: vi.fn(),
      workspaceTree: vi.fn(),
    },
  }
})

afterEach(cleanup)

describe('ProjectRail', () => {
  beforeEach(() => {
    vi.mocked(api.listProjectNotes).mockReset().mockResolvedValue([])
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({ entries: [] })
  })

  it('defaults to Config with Instructions only and no Memory field', () => {
    render(ProjectRail, { props: { projectId: 'p1' } })

    expect(screen.getByRole('tab', { name: 'Config' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: 'Files' })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Instructions (system)' })).toBeInTheDocument()
    expect(screen.getByText('Not saved yet — persistence coming later.')).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Memory' })).not.toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Memory' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /save/i })).not.toBeInTheDocument()
  })

  it('orders panel tabs before workspace controls in the icon bar', () => {
    render(ProjectRail, { props: { projectId: 'p1' } })

    const iconbar = screen.getByRole('toolbar', { name: 'Project rail' })
    const controls = Array.from(iconbar.querySelectorAll('button'))
    expect(controls.map((control) => control.getAttribute('aria-label'))).toEqual([
      'Config',
      'Files',
      'Expand workspace',
      'Collapse canvas',
    ])
  })

  it('uses a controlled tab and reports tab changes', async () => {
    const onTabChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', tab: 'config', onTabChange } })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    expect(onTabChange).toHaveBeenCalledWith('files')
    expect(screen.getByRole('tab', { name: 'Config' })).toHaveAttribute('aria-selected', 'true')
  })

  it('switches an uncontrolled rail to Files and shows empty copy', async () => {
    render(ProjectRail, { props: { projectId: 'p1' } })
    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    expect(await screen.findByText('No project files available.')).toBeInTheDocument()
  })

  it('lists project notes as files and opens one with project-note metadata', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.listProjectNotes).mockResolvedValue([
      { path: 'notes/a.md', kind: 'file', note_id: 'n-a' },
      { path: 'notes', kind: 'folder' },
    ])
    render(ProjectRail, { props: { projectId: 'p1', sessionId: 's1', onOpenFile } })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'notes/a.md' }))
    expect(onOpenFile).toHaveBeenCalledWith('notes/a.md', {
      source: 'project-note',
      noteId: 'n-a',
    })
  })

  it('opens workspace rows with workspace metadata', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.workspaceTree).mockResolvedValue({
      entries: [{ path: 'scratch.txt', kind: 'file' }],
    })
    render(ProjectRail, {
      props: {
        projectId: 'p1',
        sessionId: 's1',
        workspaceFilesEnabled: true,
        onOpenFile,
      },
    })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'scratch.txt' }))
    expect(onOpenFile).toHaveBeenCalledWith('scratch.txt', { source: 'workspace' })
  })

  it('does not open a project note without a session', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.listProjectNotes).mockResolvedValue([
      { path: 'notes/a.md', kind: 'file', note_id: 'n-a' },
    ])
    render(ProjectRail, { props: { projectId: 'p1', onOpenFile } })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'notes/a.md' }))
    expect(onOpenFile).not.toHaveBeenCalled()
  })

  it('merges workspace files under Workspace when the session grant is enabled', async () => {
    vi.mocked(api.listProjectNotes).mockResolvedValue([{ path: 'readme.md', kind: 'file' }])
    vi.mocked(api.workspaceTree).mockResolvedValue({
      entries: [{ path: 'scratch.txt', kind: 'file' }],
    })
    render(ProjectRail, {
      props: {
        projectId: 'p1',
        sessionId: 's1',
        workspaceFilesEnabled: true,
      },
    })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await waitFor(() => expect(api.workspaceTree).toHaveBeenCalledWith('s1'))
    expect(await screen.findByText('Workspace')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'readme.md' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'scratch.txt' })).toBeInTheDocument()
  })

  it('does not load workspace files when the grant is disabled', async () => {
    render(ProjectRail, { props: { projectId: 'p1', sessionId: 's1' } })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await screen.findByText('No project files available.')
    expect(api.workspaceTree).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run the focused test and confirm the old Memory chrome fails the new contract**

Run (with Node `>=22 <23` first on `PATH`):

```bash
npm --prefix web test -- --run src/components/ProjectRail.test.ts
```

Expected: FAIL because `Config`, `Expand workspace`, `Collapse canvas`, and the toolbar do not exist, while Memory still renders.

- [ ] **Step 3: Add dedicated 24-viewBox rail SVG path helpers**

Create `web/src/components/rail-icons.ts`:

```ts
/** Inline SVG path data (24 viewBox) for project rail chrome. */
export type RailIconName =
  | 'config'
  | 'files'
  | 'expand-workspace'
  | 'collapse-canvas'
  | 'show-canvas'

const icons: Record<RailIconName, string> = {
  config:
    'M4 7h10m4 0h2M4 17h2m4 0h10M14 4v6M6 14v6',
  files:
    'M4 7.5A2.5 2.5 0 0 1 6.5 5h3.2c.5 0 1 .2 1.3.6L12 7h5.5A2.5 2.5 0 0 1 20 9.5v7A2.5 2.5 0 0 1 17.5 19h-11A2.5 2.5 0 0 1 4 16.5v-9Z',
  'expand-workspace':
    'M9 4H4v5M4 4l6 6m5-6h5v5m0-5-6 6M9 20H4v-5m0 5 6-6m5 6h5v-5m0 5-6-6',
  'collapse-canvas':
    'M5 5h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm10 2v10h4V7h-4Zm-2.5 2.5L10 12l2.5 2.5',
  'show-canvas':
    'M5 5h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm10 2v10h4V7h-4Zm-5.5 2.5L12 12l-2.5 2.5',
}

export function railIconPath(name: RailIconName): string {
  return icons[name]
}
```

These paths intentionally follow the existing `nav-icons.ts` pattern: a closed name union, a complete record, and one lookup helper. Render them with `viewBox="0 0 24 24"`, `aria-hidden="true"`, and the same stroke treatment for every icon (`fill="none"`, `stroke="currentColor"`, `stroke-width="1.8"`, `stroke-linecap="round"`, `stroke-linejoin="round"`).

- [ ] **Step 4: Rewrite ProjectRail state and icon chrome while preserving the Files pipeline**

In `web/src/components/ProjectRail.svelte`, import the canonical types and icon helper, and replace the prop/state declarations with:

```svelte
<script lang="ts">
  import { api } from '../lib/api'
  import type { NoteTreeEntry, WorkspaceEntry } from '../lib/api/types'
  import type { ProjectRailMode, ProjectRailTab } from '../lib/project-rail-prefs'
  import { buildHierarchy, flattenTree, type TreeNode } from '../lib/workspace-tree'
  import { railIconPath, type RailIconName } from './rail-icons'
  import Skeleton from './Skeleton.svelte'

  export type OpenFileMeta = {
    source: 'project-note' | 'workspace'
    noteId?: string
  }

  let {
    projectId,
    sessionId = null,
    workspaceFilesEnabled = false,
    tab: controlledTab,
    mode,
    onTabChange,
    onModeChange,
    onOpenFile,
  }: {
    projectId: string
    sessionId?: string | null
    workspaceFilesEnabled?: boolean
    tab?: ProjectRailTab
    mode?: ProjectRailMode
    onTabChange?: (tab: ProjectRailTab) => void
    onModeChange?: (mode: ProjectRailMode) => void
    onOpenFile?: (path: string, meta?: OpenFileMeta) => void
  } = $props()

  type RailEntry = WorkspaceEntry & { note_id?: string }

  let localTab = $state<ProjectRailTab>('config')
  let instructions = $state('')
  const activeTab = $derived(controlledTab ?? localTab)
  const activeMode = $derived(mode ?? 'open')
```

Keep `projectEntries`, `workspaceEntries`, `noteIdByPath`, `loading`, `error`, `loadToken`, `noteKindToWorkspace`, `notesToWorkspace`, and `loadFiles` exactly as currently implemented. Change the effect and selector to use controlled-or-uncontrolled state:

```ts
  $effect(() => {
    void projectId
    void sessionId
    void workspaceFilesEnabled
    if (activeTab === 'files') void loadFiles()
  })

  function selectTab(next: ProjectRailTab) {
    if (controlledTab === undefined) localTab = next
    onTabChange?.(next)
  }
```

Keep the current derived tree rows and both row-click handlers exactly as implemented, then close `</script>`. Replace the old `.rail-tabs` and Memory panel markup with:

```svelte
{#snippet icon(name: RailIconName)}
  <svg
    viewBox="0 0 24 24"
    aria-hidden="true"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <path d={railIconPath(name)}></path>
  </svg>
{/snippet}

<div class="project-rail">
  <div class="rail-iconbar" role="toolbar" aria-label="Project rail">
    <div role="tablist" aria-label="Project rail panels">
      <button
        type="button"
        role="tab"
        id="rail-tab-config"
        class="rail-icon {activeTab === 'config' ? 'rail-icon--active' : ''}"
        aria-label="Config"
        title="Config"
        aria-selected={activeTab === 'config'}
        aria-controls="rail-panel-config"
        tabindex={activeTab === 'config' ? 0 : -1}
        onclick={() => selectTab('config')}
      >{@render icon('config')}</button>
      <button
        type="button"
        role="tab"
        id="rail-tab-files"
        class="rail-icon {activeTab === 'files' ? 'rail-icon--active' : ''}"
        aria-label="Files"
        title="Files"
        aria-selected={activeTab === 'files'}
        aria-controls="rail-panel-files"
        tabindex={activeTab === 'files' ? 0 : -1}
        onclick={() => selectTab('files')}
      >{@render icon('files')}</button>
    </div>
    <div>
      <button
        type="button"
        class="rail-icon"
        aria-label={activeMode === 'expanded' ? 'Exit expanded' : 'Expand workspace'}
        title={activeMode === 'expanded' ? 'Exit expanded' : 'Expand workspace'}
        aria-pressed={activeMode === 'expanded'}
        onclick={() => onModeChange?.(activeMode === 'expanded' ? 'open' : 'expanded')}
      >{@render icon('expand-workspace')}</button>
      <button
        type="button"
        class="rail-icon"
        aria-label="Collapse canvas"
        title="Collapse canvas"
        onclick={() => onModeChange?.('collapsed')}
      >{@render icon('collapse-canvas')}</button>
    </div>
  </div>

  {#if activeTab === 'config'}
    <div class="rail-panel form-stack" role="tabpanel" id="rail-panel-config" aria-labelledby="rail-tab-config">
      <label class="block text-sm" for="rail-instructions">
        Instructions (system)
        <textarea
          id="rail-instructions"
          class="field-textarea mt-1"
          aria-label="Instructions (system)"
          bind:value={instructions}
          rows="6"
        ></textarea>
      </label>
      <p class="text-sm text-slate-500" style="margin:0">Not saved yet — persistence coming later.</p>
    </div>
  {:else}
    <div class="rail-panel form-stack" role="tabpanel" id="rail-panel-files" aria-labelledby="rail-tab-files">
      <!-- Move the complete existing Files loading/error/empty/projectRows/workspaceRows block here unchanged. -->
    </div>
  {/if}
</div>
```

For the marked Files location, copy the existing block beginning `{#if loading}` and ending at its matching `{/if}` byte-for-byte; do not alter API calls, skeletons, copy, tree labels, row indentation, disabled-directory behavior, or click handlers. The comment is an editing instruction for this plan and must not remain in product code.

- [ ] **Step 5: Run the focused tests and confirm the chrome and Files regressions pass**

Run:

```bash
npm --prefix web test -- --run src/components/ProjectRail.test.ts
```

Expected: all `ProjectRail.test.ts` tests PASS. The controlled-tab test must prove that clicking Files emits `files` but does not mutate the Config selection until the parent changes the prop.

- [ ] **Step 6: Commit Task 3**

```bash
git add web/src/components/rail-icons.ts web/src/components/ProjectRail.svelte web/src/components/ProjectRail.test.ts
git commit -m "feat(web): replace project rail tabs with icon chrome"
```

### Task 4: ProjectRail expand/collapse callbacks and collapsed chrome

**Files:**
- Modify: `web/src/components/ProjectRail.svelte`
- Test: `web/src/components/ProjectRail.test.ts`

**Interfaces:**
- Consumes: Task 3's `activeMode = mode ?? 'open'`, icon helper, and optional `onModeChange` callback.
- Produces: collapsed-only `Show canvas` chrome; callback transitions `open → expanded`, `expanded → open`, `open|expanded → collapsed`, and `collapsed → open`. `ProjectRail` never owns or mutates mode locally.

- [ ] **Step 1: Add failing tests for collapsed chrome and exact mode callback arguments**

Append these tests inside the existing `describe('ProjectRail', ...)` block in `web/src/components/ProjectRail.test.ts`:

```ts
  it('renders only Show canvas chrome when collapsed', () => {
    render(ProjectRail, { props: { projectId: 'p1', mode: 'collapsed' } })

    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Show canvas' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Config' })).not.toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Files' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Expand workspace' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Collapse canvas' })).not.toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Instructions (system)' })).not.toBeInTheDocument()
    expect(screen.queryByRole('tabpanel')).not.toBeInTheDocument()
  })

  it('requests expanded mode from the open rail', async () => {
    const onModeChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', mode: 'open', onModeChange } })

    await fireEvent.click(screen.getByRole('button', { name: 'Expand workspace' }))
    expect(onModeChange).toHaveBeenCalledWith('expanded')
  })

  it('requests open mode from the expanded rail', async () => {
    const onModeChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', mode: 'expanded', onModeChange } })

    const exit = screen.getByRole('button', { name: 'Exit expanded' })
    expect(exit).toHaveAttribute('aria-pressed', 'true')
    await fireEvent.click(exit)
    expect(onModeChange).toHaveBeenCalledWith('open')
  })

  it('requests collapsed mode from Collapse canvas', async () => {
    const onModeChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', mode: 'open', onModeChange } })

    await fireEvent.click(screen.getByRole('button', { name: 'Collapse canvas' }))
    expect(onModeChange).toHaveBeenCalledWith('collapsed')
  })

  it('requests open mode from Show canvas', async () => {
    const onModeChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', mode: 'collapsed', onModeChange } })

    await fireEvent.click(screen.getByRole('button', { name: 'Show canvas' }))
    expect(onModeChange).toHaveBeenCalledWith('open')
  })
```

- [ ] **Step 2: Run the focused test and verify collapsed mode still exposes the full rail**

Run:

```bash
npm --prefix web test -- --run src/components/ProjectRail.test.ts
```

Expected: the collapsed-only test FAILS because Task 3 still renders Config, Files, both workspace controls, and the Config panel when `mode="collapsed"`. Callback tests may already pass from Task 3's toolbar wiring.

- [ ] **Step 3: Gate the complete rail behind collapsed-only restore chrome**

In `web/src/components/ProjectRail.svelte`, wrap the contents of `.project-rail` so collapsed mode has exactly one button and no panel body:

```svelte
<div class="project-rail">
  {#if activeMode === 'collapsed'}
    <button
      type="button"
      class="rail-icon"
      aria-label="Show canvas"
      title="Show canvas"
      onclick={() => onModeChange?.('open')}
    >{@render icon('show-canvas')}</button>
  {:else}
    <div class="rail-iconbar" role="toolbar" aria-label="Project rail">
      <!-- Keep Task 3's complete Config/Files tab cluster and Expand/Collapse button cluster here. -->
    </div>

    {#if activeTab === 'config'}
      <!-- Keep Task 3's complete Config tabpanel here. -->
    {:else}
      <!-- Keep Task 3's complete Files tabpanel here. -->
    {/if}
  {/if}
</div>
```

The three comments above are move instructions only: retain Task 3's complete markup in those positions and remove the comments from product code. Do not add local mode state. Keep the exact actions already introduced in Task 3:

```ts
// Expand workspace (open)
onModeChange?.('expanded')

// Exit expanded (expanded)
onModeChange?.('open')

// Collapse canvas (open or expanded)
onModeChange?.('collapsed')

// Show canvas (collapsed)
onModeChange?.('open')
```

When `mode` and/or `onModeChange` is omitted, `activeMode` remains visually `open`; clicks are safe optional-callback no-ops. The hub in Tasks 5–6 owns actual mode transitions.

- [ ] **Step 4: Run the complete ProjectRail test file**

Run:

```bash
npm --prefix web test -- --run src/components/ProjectRail.test.ts
```

Expected: all tests PASS; collapsed mode has only `Show canvas`, open mode labels expand as `Expand workspace`, expanded mode labels it `Exit expanded`, and callback arguments exactly match the canonical modes.

- [ ] **Step 5: Run the production build to type-compile the new props and SVG helper**

Run:

```bash
npm --prefix web run build
```

Expected: PASS with no Svelte or TypeScript diagnostics and a successful Vite build. Task 7 owns the final served-dist verification and browser vibe-pass; do not perform those here.

- [ ] **Step 6: Commit Task 4**

```bash
git add web/src/components/ProjectRail.svelte web/src/components/ProjectRail.test.ts
git commit -m "feat(web): add project rail mode controls"
```
