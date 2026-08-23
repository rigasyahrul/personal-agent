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
