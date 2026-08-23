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
