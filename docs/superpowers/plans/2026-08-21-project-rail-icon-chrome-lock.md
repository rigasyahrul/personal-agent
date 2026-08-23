# Lock: Project rail icon chrome

**Date:** 2026-08-21  
**Spec:** `docs/superpowers/specs/2026-08-21-project-rail-icon-chrome-design.md`  
**Assembled plan (target):** `docs/superpowers/plans/2026-08-21-project-rail-icon-chrome.md`  
**Drafts dir:** `docs/superpowers/plans/2026-08-21-project-rail-icon-chrome-drafts/`

## Scope freeze (do not expand)

| In | Out |
|----|-----|
| Icon toolbar: Config · Files (left); Expand workspace · Collapse canvas (right) | Memory field / Memory API |
| Modes: open / expanded / collapsed on hub `.project-workspace` | Files tree redesign / search UI (refs 6–7) |
| Default: open + Config | Left app sidebar changes |
| Config = Instructions (system) only, non-persistent | Standalone session route rail |
| Files = existing tree + open behavior | Dark mode |
| localStorage mode + tab | Product `/docs/memory` |
| Tokens in `app.css`; vibe-pass vs icon/layout refs | Committing large PNGs unless user asks |

## Authority

1. Spec wins over older benchmark rail text (Memory \| Files tabs).  
2. This lock freezes task ranges and file ownership for draft agents.  
3. Master assembles drafts into one plan; no solo mega-plan stall.  
4. Every implementation worker task later: consulting-grok-review before merge (repo standing rule).

## Draft file map (parallel writers)

| Draft file | Tasks | Owner focus |
|------------|-------|-------------|
| `01-prefs-tokens.md` | 1–2 | `project-rail-prefs` + CSS tokens / baseline |
| `02-project-rail.md` | 3–4 | `ProjectRail` icon chrome + unit tests |
| `03-hub-modes.md` | 5–6 | `ProjectHubPage` modes + hub tests |
| `04-verify-vibe.md` | 7 | suite, dist, vibe-pass checklist |

## Canonical contracts (drafts must use these names)

```ts
// web/src/lib/project-rail-prefs.ts
export type ProjectRailMode = 'open' | 'expanded' | 'collapsed'
export type ProjectRailTab = 'config' | 'files'

export const PROJECT_RAIL_MODE_KEY = 'pa.projectRail.mode'
export const PROJECT_RAIL_TAB_KEY = 'pa.projectRail.tab'

export function readProjectRailMode(storage: Storage | null | undefined): ProjectRailMode
export function writeProjectRailMode(storage: Storage | null | undefined, mode: ProjectRailMode): void
export function readProjectRailTab(storage: Storage | null | undefined): ProjectRailTab
export function writeProjectRailTab(storage: Storage | null | undefined, tab: ProjectRailTab): void
// Defaults: mode 'open', tab 'config'. Invalid/missing → defaults. null storage no-op writes / default reads.
```

```ts
// ProjectRail props (additive)
{
  projectId: string
  sessionId?: string | null
  workspaceFilesEnabled?: boolean
  tab?: ProjectRailTab           // controlled from hub when provided
  mode?: ProjectRailMode         // for collapsed UI: hide iconbar except restore
  onTabChange?: (tab: ProjectRailTab) => void
  onModeChange?: (mode: ProjectRailMode) => void
  onOpenFile?: (path: string, meta?: OpenFileMeta) => void
}
```

**Aria labels (exact):**
- Config → `Config`
- Files → `Files`
- Expand (when open) → `Expand workspace`
- Expand (when expanded) → `Exit expanded` (or keep Expand workspace + `aria-pressed`)
- Collapse (when open/expanded) → `Collapse canvas`
- Restore (when collapsed) → `Show canvas`

**data-rail:** `.project-workspace` gets `data-rail={mode}`.

**CSS tokens (required classes):**  
`.rail-iconbar`, `.rail-icon`, `.rail-icon--active`, keep `.project-rail`, `.rail-panel`, `.project-workspace__*`.  
Replace baseline asserts: drop `.rail-tabs` / `.rail-tab` / `.rail-tab--active` if removed; add iconbar tokens.

**Icon SVGs:** extend `web/src/shell/nav-icons.ts` or add `web/src/components/rail-icons.ts` with paths for: `config` (settings/sliders), `files` (folder), `expand-workspace`, `collapse-canvas` / `show-canvas`. Prefer dedicated `rail-icons.ts` to avoid bloating shell nav types — drafts pick one and stay consistent.

## File ownership (avoid conflicts)

| File | Primary draft |
|------|----------------|
| `web/src/lib/project-rail-prefs.ts` + test | 01 |
| `web/src/app.css` (rail section) + `styles-baseline.test.ts` | 01 |
| `web/src/components/rail-icons.ts` (if new) | 02 |
| `web/src/components/ProjectRail.svelte` + `.test.ts` | 02 |
| `web/src/routes/ProjectHubPage.svelte` + `.test.ts` | 03 |
| Verify / vibe notes in plan Task 7 only | 04 |

## Task ranges (final plan)

1. Prefs module TDD  
2. CSS tokens + baseline  
3. ProjectRail icon chrome TDD (Config/Files/no Memory)  
4. ProjectRail expand/collapse callbacks + collapsed chrome  
5. Hub wiring modes + persistence  
6. Hub integration tests (expand covers main, collapse slim)  
7. Full web-test, dist build, vibe-pass checklist  

## Node / verify

- Node `>=22 <23` on PATH before `npm --prefix web test` / `make web-test`  
- Go serves `web/dist` — rebuild before localhost vibe-pass; cache-bust  

## Draft writing rules

- Full TDD steps with real test code (no “similar to Task N”)  
- Checkbox steps `- [ ]`  
- Exact run commands  
- Commit step per task  
- No scope creep past lock table  
