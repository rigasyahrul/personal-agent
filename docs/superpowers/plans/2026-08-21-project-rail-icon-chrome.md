# Project Rail Icon Chrome Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** After each task: `consulting-grok-review` via **new** `amp -m grok45 --no-archive-after-execute -x '…'` (never `-ox`, never Task/OpenAI self-review). Isolate product work in git worktrees when using local `-x`.
>
> **Spec:** `docs/superpowers/specs/2026-08-21-project-rail-icon-chrome-design.md`  
> **Lock:** `docs/superpowers/plans/2026-08-21-project-rail-icon-chrome-lock.md`  
> **Drafts:** `docs/superpowers/plans/2026-08-21-project-rail-icon-chrome-drafts/`

**Goal:** Replace the project hub right rail’s Memory|Files text tabs with Grok-style icon chrome (Config · Files left; Expand workspace · Collapse canvas right), three layout modes (open / expanded / collapsed), Config = Instructions only, Files behavior unchanged.

**Architecture:** Prefs module owns localStorage mode/tab. CSS `data-rail` on `.project-workspace` drives open/expanded/collapsed grids. `ProjectRail` renders icon toolbar + panels and emits mode/tab changes; `ProjectHubPage` owns controlled state and persistence. Expand covers main with the active panel; collapsed is a slim restore strip.

**Tech Stack:** Svelte 5 + TypeScript + Vite + Vitest + Testing Library; tokens in `web/src/app.css`; Go serves `web/dist` on `:8080`.

## Global Constraints

- Spec wins over older benchmark rail text tabs (Memory | Files).
- Node `>=22 <23` on `PATH` before any `npm test` / `make web-test`.
- Rebuild `web/dist` + cache-bust (`?v=<ts>#/route`) before claiming localhost vibe-pass (Go serves dist, not Vite HMR).
- No Memory field or Memory API; product memory is `/docs/memory` later.
- Files tree/search redesign deferred (refs 6–7 out of scope).
- Polled session UI: never remount/replace a focused composer.
- Tokens first in `app.css`; no indigo one-off soup.
- Every worker task: consulting-grok-review PASS before FF-merge.
- Do not commit large benchmark PNGs unless user explicitly asks.
- Ship = push when user allows; commit alone is not ship.

## File map

| File | Role |
|------|------|
| `web/src/lib/project-rail-prefs.ts` | Mode/tab localStorage helpers |
| `web/src/lib/project-rail-prefs.test.ts` | Prefs unit tests |
| `web/src/app.css` | `data-rail` layouts + `.rail-iconbar` / `.rail-icon*` |
| `web/src/styles-baseline.test.ts` | Token contract |
| `web/src/components/rail-icons.ts` | Rail SVG path helpers |
| `web/src/components/ProjectRail.svelte` | Icon chrome + Config/Files panels |
| `web/src/components/ProjectRail.test.ts` | Rail unit tests |
| `web/src/routes/ProjectHubPage.svelte` | Controlled mode/tab + `data-rail` |
| `web/src/routes/ProjectHubPage.test.ts` | Hub mode/persistence/file bridge |

## Canonical contracts

```ts
// web/src/lib/project-rail-prefs.ts
export type ProjectRailMode = 'open' | 'expanded' | 'collapsed'
export type ProjectRailTab = 'config' | 'files'
export const PROJECT_RAIL_MODE_KEY = 'pa.projectRail.mode'
export const PROJECT_RAIL_TAB_KEY = 'pa.projectRail.tab'
// defaults: mode open, tab config; invalid/missing → defaults; null storage no-op writes
```

```ts
// ProjectRail props
{
  projectId: string
  sessionId?: string | null
  workspaceFilesEnabled?: boolean
  tab?: ProjectRailTab
  mode?: ProjectRailMode
  onTabChange?: (tab: ProjectRailTab) => void
  onModeChange?: (mode: ProjectRailMode) => void
  onOpenFile?: (path: string, meta?: OpenFileMeta) => void
}
```

**Aria labels:** `Config`, `Files`, `Expand workspace` / `Exit expanded`, `Collapse canvas`, `Show canvas`.

**Layout:** `.project-workspace[data-rail={mode}]` — open ~300px rail; expanded main hidden; collapsed 46px slim restore.

**Icon bar markup:** left/right clusters use `.rail-iconbar__group` inside `.rail-iconbar` (role=toolbar).

---

### Task 1: Add project rail preference helpers with TDD

**Files:**
- Create: `web/src/lib/project-rail-prefs.ts`
- Create: `web/src/lib/project-rail-prefs.test.ts`

**Interfaces:**
- Consumes: Browser-compatible `Storage | null | undefined` supplied by the project hub.
- Produces: `ProjectRailMode = 'open' | 'expanded' | 'collapsed'`, `ProjectRailTab = 'config' | 'files'`, constants `PROJECT_RAIL_MODE_KEY = 'pa.projectRail.mode'` and `PROJECT_RAIL_TAB_KEY = 'pa.projectRail.tab'`, plus `readProjectRailMode`, `writeProjectRailMode`, `readProjectRailTab`, and `writeProjectRailTab` with the exact signatures below. Reads default to `open` / `config` for missing, invalid, or unavailable storage; writes to unavailable storage are no-ops.

- [ ] **Step 1: Write the failing preference contract tests**

Create `web/src/lib/project-rail-prefs.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import {
  PROJECT_RAIL_MODE_KEY,
  PROJECT_RAIL_TAB_KEY,
  readProjectRailMode,
  readProjectRailTab,
  writeProjectRailMode,
  writeProjectRailTab,
} from './project-rail-prefs'

function mem(): Storage {
  const values = new Map<string, string>()
  return {
    get length() { return values.size },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => { values.set(key, String(value)) },
    removeItem: (key) => { values.delete(key) },
    key: (index) => [...values.keys()][index] ?? null,
  }
}

describe('project-rail-prefs', () => {
  it('defaults to an open rail with Config selected', () => {
    const storage = mem()

    expect(readProjectRailMode(storage)).toBe('open')
    expect(readProjectRailTab(storage)).toBe('config')
  })

  it('round-trips every supported mode and tab using the canonical keys', () => {
    const storage = mem()

    for (const mode of ['open', 'expanded', 'collapsed'] as const) {
      writeProjectRailMode(storage, mode)
      expect(storage.getItem(PROJECT_RAIL_MODE_KEY)).toBe(mode)
      expect(readProjectRailMode(storage)).toBe(mode)
    }
    for (const tab of ['config', 'files'] as const) {
      writeProjectRailTab(storage, tab)
      expect(storage.getItem(PROJECT_RAIL_TAB_KEY)).toBe(tab)
      expect(readProjectRailTab(storage)).toBe(tab)
    }

    expect(PROJECT_RAIL_MODE_KEY).toBe('pa.projectRail.mode')
    expect(PROJECT_RAIL_TAB_KEY).toBe('pa.projectRail.tab')
  })

  it('replaces missing or invalid stored values with defaults on read', () => {
    const storage = mem()
    storage.setItem(PROJECT_RAIL_MODE_KEY, 'wide')
    storage.setItem(PROJECT_RAIL_TAB_KEY, 'memory')

    expect(readProjectRailMode(storage)).toBe('open')
    expect(readProjectRailTab(storage)).toBe('config')
  })

  it('uses defaults and no-op writes when storage is unavailable', () => {
    expect(readProjectRailMode(null)).toBe('open')
    expect(readProjectRailMode(undefined)).toBe('open')
    expect(readProjectRailTab(null)).toBe('config')
    expect(readProjectRailTab(undefined)).toBe('config')

    expect(() => writeProjectRailMode(null, 'collapsed')).not.toThrow()
    expect(() => writeProjectRailMode(undefined, 'expanded')).not.toThrow()
    expect(() => writeProjectRailTab(null, 'files')).not.toThrow()
    expect(() => writeProjectRailTab(undefined, 'config')).not.toThrow()
  })
})
```

- [ ] **Step 2: Run the focused test and verify the red state**

Run:

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web test -- src/lib/project-rail-prefs.test.ts
```

Expected: FAIL because `./project-rail-prefs` does not exist (or its exports are unresolved).

- [ ] **Step 3: Implement the minimal typed preference module**

Create `web/src/lib/project-rail-prefs.ts`:

```ts
export type ProjectRailMode = 'open' | 'expanded' | 'collapsed'
export type ProjectRailTab = 'config' | 'files'

export const PROJECT_RAIL_MODE_KEY = 'pa.projectRail.mode'
export const PROJECT_RAIL_TAB_KEY = 'pa.projectRail.tab'

const DEFAULT_PROJECT_RAIL_MODE: ProjectRailMode = 'open'
const DEFAULT_PROJECT_RAIL_TAB: ProjectRailTab = 'config'

export function readProjectRailMode(storage: Storage | null | undefined): ProjectRailMode {
  if (!storage) return DEFAULT_PROJECT_RAIL_MODE
  const value = storage.getItem(PROJECT_RAIL_MODE_KEY)
  return value === 'open' || value === 'expanded' || value === 'collapsed'
    ? value
    : DEFAULT_PROJECT_RAIL_MODE
}

export function writeProjectRailMode(
  storage: Storage | null | undefined,
  mode: ProjectRailMode,
): void {
  if (!storage) return
  storage.setItem(PROJECT_RAIL_MODE_KEY, mode)
}

export function readProjectRailTab(storage: Storage | null | undefined): ProjectRailTab {
  if (!storage) return DEFAULT_PROJECT_RAIL_TAB
  const value = storage.getItem(PROJECT_RAIL_TAB_KEY)
  return value === 'config' || value === 'files' ? value : DEFAULT_PROJECT_RAIL_TAB
}

export function writeProjectRailTab(
  storage: Storage | null | undefined,
  tab: ProjectRailTab,
): void {
  if (!storage) return
  storage.setItem(PROJECT_RAIL_TAB_KEY, tab)
}
```

- [ ] **Step 4: Run the focused test and verify the green state**

Run:

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web test -- src/lib/project-rail-prefs.test.ts
```

Expected: PASS with 4 tests passing and no uncaught errors.

- [ ] **Step 5: Commit the preference module**

```bash
git add web/src/lib/project-rail-prefs.ts web/src/lib/project-rail-prefs.test.ts
git commit -m "feat(web): add project rail preferences"
```

### Task 2: Add project rail mode and icon chrome CSS tokens

**Files:**
- Modify: `web/src/styles-baseline.test.ts` (`declares hub/rail workspace tokens` test)
- Modify: `web/src/app.css:666-790` (project hub workspace and rail section)

**Interfaces:**
- Consumes: `.project-workspace[data-rail="open|expanded|collapsed"]` emitted by the hub in a later task, existing `.project-workspace__main`, `.project-workspace__rail`, `.project-rail`, and `.rail-panel` markup, and future `.rail-iconbar`, `.rail-icon`, `.rail-icon--active` markup.
- Produces: Open two-column layout with a 300px rail, expanded rail-only layout, collapsed 46px rail layout, and reusable icon toolbar/button tokens. The legacy `.rail-tabs`, `.rail-tab`, and `.rail-tab--active` rules remain temporarily so current markup stays styled until Task 3 replaces it; the baseline contract asserts the new tokens only.

- [ ] **Step 1: Replace the baseline token expectations with the new icon and mode contract**

In `web/src/styles-baseline.test.ts`, replace the existing `declares hub/rail workspace tokens` test with:

```ts
  it('declares hub/rail workspace tokens', () => {
    for (const token of [
      '.project-workspace',
      '.project-workspace__main',
      '.project-workspace__rail',
      '.project-rail',
      '.rail-iconbar',
      '.rail-icon',
      '.rail-icon--active',
      '.rail-panel',
      '.hub-start',
      '.hub-start__title',
      '.hub-composer',
      '.hub-session-list',
      '.hub-session-list__label',
      '.session-row',
      '.session-row__icon',
      '.session-row__title',
      '.session-row__date',
      '.session-row__menu',
      '.content-canvas--project-workspace',
    ]) {
      expect(css).toContain(token)
    }

    expect(css).toMatch(/\.project-workspace\[data-rail=['"]open['"]\]/)
    expect(css).toMatch(/\.project-workspace\[data-rail=['"]expanded['"]\]/)
    expect(css).toMatch(/\.project-workspace\[data-rail=['"]collapsed['"]\]/)
    expect(css).toMatch(/data-rail=['"]expanded['"][^}]*grid-template-columns:\s*0\s+minmax\(0,\s*1fr\)/s)
    expect(css).toMatch(/data-rail=['"]collapsed['"][^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)\s+46px/s)
  })
```

- [ ] **Step 2: Run the baseline test and verify the red state**

Run:

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web test -- src/styles-baseline.test.ts
```

Expected: FAIL because `.rail-iconbar`, `.rail-icon`, `.rail-icon--active`, and the three `data-rail` selectors are absent.

- [ ] **Step 3: Add explicit mode layouts to the project workspace CSS**

In the project hub workspace section of `web/src/app.css`, replace the base `.project-workspace` grid declaration with these base and mode rules; leave the following `.project-workspace__main` rules intact:

```css
.project-workspace {
  display: grid;
  gap: 0;
  min-height: calc(100vh - 52px);
}
.project-workspace[data-rail="open"] {
  grid-template-columns: minmax(0, 1fr) 300px;
}
.project-workspace[data-rail="expanded"] {
  grid-template-columns: 0 minmax(0, 1fr);
}
.project-workspace[data-rail="expanded"] .project-workspace__main {
  display: none;
}
.project-workspace[data-rail="collapsed"] {
  grid-template-columns: minmax(0, 1fr) 46px;
}
```

This makes the mode contract explicit rather than relying on the prior default `minmax(260px, 320px)` rail. The hub will always provide `data-rail`; `open` is the preference default from Task 1.

- [ ] **Step 4: Add the icon toolbar and icon button tokens while retaining legacy tab rules**

Immediately after `.project-workspace__rail` and before the existing `.rail-tabs` rules in `web/src/app.css`, add:

```css
.rail-iconbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 48px;
  padding: 4px 6px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.rail-iconbar__group {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}
.rail-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: #52525b;
  cursor: pointer;
}
.rail-icon:hover {
  border-color: var(--border);
  background: #fafafa;
  color: #18181b;
}
.rail-icon:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
.rail-icon--active {
  border-color: color-mix(in srgb, var(--accent) 28%, var(--border));
  background: var(--accent-soft);
  color: var(--accent);
}
.rail-icon svg {
  width: 18px;
  height: 18px;
}
```

Do not remove `.rail-tabs`, `.rail-tab`, or `.rail-tab--active` in this task: current `ProjectRail` markup still consumes them until the icon chrome task lands. Keep the existing `.rail-panel`, `.project-rail`, and all `.project-workspace__*` rules.

- [ ] **Step 5: Run the focused baseline test and verify the green state**

Run:

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web test -- src/styles-baseline.test.ts
```

Expected: PASS; the baseline finds all three mode selectors and the new iconbar tokens.

- [ ] **Step 6: Run both Task 1 and Task 2 focused tests together**

Run:

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
npm --prefix web test -- src/lib/project-rail-prefs.test.ts src/styles-baseline.test.ts
```

Expected: PASS with both test files green and no uncaught errors.

- [ ] **Step 7: Commit the CSS mode and icon token contract**

```bash
git add web/src/app.css web/src/styles-baseline.test.ts
git commit -m "style(web): add project rail icon chrome tokens"
```


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
    <div class="rail-iconbar__group" role="tablist" aria-label="Project rail panels">
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
    <div class="rail-iconbar__group">
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


### Task 7: Full verification, production build, and browser vibe-pass

**Files:**
- Modify only if verification exposes renamed-tab fallout: `web/src/App.test.ts`
- Verify: `web/src/components/ProjectRail.test.ts`
- Verify: `web/src/routes/ProjectHubPage.test.ts`
- Verify: `web/src/styles-baseline.test.ts`
- Generated by build (do not commit unless repository policy requires it): `web/dist/`

**Scope:** This task verifies Tasks 1–6. Do not redesign the Files tree, add Files search, add Memory rail UI, or make unrelated product changes.

- [ ] **Step 1: Put Node 22 first on `PATH` and verify the active version**

```bash
export PATH="/opt/homebrew/opt/node@22/bin:$PATH"
node --version
```

Expected: `v22.x.x` (`>=22 <23`). Keep this `PATH` in the same shell for every web command below.

- [ ] **Step 2: Run the complete web test suite**

```bash
npm --prefix web test
```

Expected: PASS with no failed tests and no uncaught jsdom errors. `make web-test` is an acceptable equivalent after putting Node 22 first on `PATH`.

- [ ] **Step 3: Fix only renamed-tab fallout, if the full suite finds any**

If `web/src/App.test.ts` or another existing assertion still expects a `Memory` rail tab, update it to the approved chrome contract: Config and Files selectors, no Memory rail control. Do not weaken unrelated assertions or alter product behavior in this verification task.

Re-run the exact failing test first, then the full suite:

```bash
npm --prefix web test -- App.test.ts
npm --prefix web test
```

Expected: the focused test and complete suite PASS. If there was no renamed-tab fallout, make no source or test edit in this step.

- [ ] **Step 4: Build the production web bundle served by Go**

```bash
npm --prefix web run build
```

Expected: PASS and `web/dist/index.html` references newly generated hashed assets. Record the hashes with:

```bash
grep -Eo 'assets/[^" ]+\.(js|css)' web/dist/index.html
```

Expected: hashed `.js` and `.css` asset names are present; these are the assets the Go server on `:8080` must serve.

- [ ] **Step 5: Optionally confirm the built bundle contains the new rail contract**

```bash
grep -R -E 'rail-iconbar|rail-icon--active|Expand workspace|Collapse canvas|Show canvas' web/dist/assets
```

Expected: matches in built CSS/JS for the new class strings and accessible labels. This is a smoke check, not a substitute for tests or browser inspection.

- [ ] **Step 6: Start the Go-served production bundle and open a cache-busted project hub**

In one shell:

```bash
go run ./cmd/personal-agent
```

In another shell, verify the rebuilt app is served and print a cache-busted URL:

```bash
curl -fsS http://localhost:8080/ | grep -Eo 'assets/[^" ]+\.(js|css)'
printf 'http://localhost:8080/?v=%s#/projects/<PROJECT_ID>\n' "$(date +%s)"
```

Expected: `curl` reports the same current asset hashes from `web/dist/index.html`. Replace `<PROJECT_ID>` with an existing project ID, open that URL in the browser, and keep DevTools responsive-width checks within the product's supported desktop layout.

- [ ] **Step 7: Complete the side-by-side browser vibe-pass**

Use the repo-root reference images beside the live cache-busted hub. Exercise Config and Files in `open`, `expanded`, and `collapsed` modes; do not mark this step complete from unit tests alone.

| Reference / product baseline | Browser action | Required visual and behavioral result | Pass |
|---|---|---|---|
| `expected-right-sidebar-1.png` | Load the hub with mode `open`; select Config, then Files | Horizontal icon bar has Config and Files in the left cluster and Expand and Collapse in the right cluster; active icon, spacing, panel boundary, and hit targets read as one coherent rail | [ ] |
| `expected-icon-right-sidebar-1.png` | Click **Collapse canvas** | Rail becomes a slim ~44–48px strip; only the restore control remains; main canvas uses the released width. Click **Show canvas** and confirm `open` returns with the prior selected panel | [ ] |
| `expected-icon-right-sidebar-2.png` | From `open`, click **Expand workspace** | The selected rail panel covers the project main canvas at full content width; selected panel does not change. Exit expanded and confirm normal side-by-side layout returns | [ ] |
| `expected-icon-right-sidebar-3.png` | Hover and keyboard-focus the Files icon | Tooltip/accessibility label reads **Files**; folder icon is recognizable; selecting it shows current product Files behavior | [ ] |
| `expected-icon-right-sidebar-4.png` | Hover and keyboard-focus the Config icon | Tooltip/accessibility label reads **Config**; settings/sliders icon is recognizable; selected state is clear | [ ] |
| Current product Files behavior (not the Files search reference PNGs) | Select Files; open a project note and, when enabled, a workspace file; inspect loading/error/empty states as available | Existing tree, note/workspace grouping, grant gating, and open-file routing retain parity. No search control or tree redesign is required | [ ] |

Also verify with keyboard Tab + Enter/Space that every icon control has a visible focus indicator and activates correctly. Confirm Config contains **Instructions (system)** and helper copy only, with no Memory field.

- [ ] **Step 8: Record intentional deviations explicitly**

- Files tree/search styling shown in `expected-files-only-tree-right-sidebar-4.png` and `expected-files-tree-with-search-right-sidebar-4.png` is intentionally deferred; Files content must retain current product parity instead.
- Product Memory is intentionally absent from the project rail; Config contains non-persistent Instructions only.
- The app's existing left sidebar and dark mode are intentionally unchanged.
- Exact pixel dimensions may follow existing shared tokens, while preserving the specified icon order, slim collapsed rail, full-width expanded panel, accessible labels, and visual hierarchy.

- [ ] **Step 9: Request the required independent implementation review**

Each implementation worker must package its task diff and pass a **new** `consulting-grok-review` before merge; this checklist and implementer self-review do not satisfy that gate. Fix all Critical and Important findings, re-run affected tests, and record the review PASS according to the repository worker ledger rules.

- [ ] **Step 10: Commit only verification fixes, if any were required**

If Step 3 required tracked test fixes, inspect and commit only those files:

```bash
git diff -- web/src/App.test.ts
git add web/src/App.test.ts
git commit -m "test: update app rail expectations"
```

If no tracked verification fix was needed, do not create an empty commit. Do not commit generated `web/dist/` or reference PNGs unless the repository policy or user explicitly requests it. **Ship means push**, but pushing is user-gated: do not run `git push` without user authorization; report the local commit hash and unpushed status instead.


---

## Spec coverage checklist (assembler)

| Spec requirement | Task |
|------------------|------|
| Icon bar Config·Files left, Expand·Collapse right | 3 |
| Modes open/expanded/collapsed | 2, 4, 5, 6 |
| Default open + Config | 1, 3, 5 |
| Config = Instructions only; no Memory | 3, 6 |
| Files existing behavior | 3, 6 |
| localStorage mode + tab | 1, 5, 6 |
| Tokens / baseline | 2 |
| Vibe-pass vs icon/layout refs | 7 |
| Files search/tree redesign | out of scope |
| Memory product /docs/memory | out of scope |
