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
