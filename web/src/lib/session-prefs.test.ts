// web/src/lib/session-prefs.test.ts
import { describe, expect, it } from 'vitest'
import {
  clampMainPct,
  readFilesBarOpen,
  readFilesBarWidthPct,
  writeFilesBarOpen,
  writeFilesBarWidthPct,
} from './session-prefs'

function mem(): Storage {
  const m = new Map<string, string>()
  return {
    get length() { return m.size },
    clear: () => m.clear(),
    getItem: (k) => m.get(k) ?? null,
    setItem: (k, v) => { m.set(k, String(v)) },
    removeItem: (k) => { m.delete(k) },
    key: () => null,
  }
}

describe('session-prefs', () => {
  it('defaults files bar closed and main width 70', () => {
    const s = mem()
    expect(readFilesBarOpen(s)).toBe(false)
    expect(readFilesBarWidthPct(s)).toBe(70)
  })
  it('persists open flag as 1/0', () => {
    const s = mem()
    writeFilesBarOpen(s, true)
    expect(s.getItem('pa.session.filesBarOpen')).toBe('1')
    expect(readFilesBarOpen(s)).toBe(true)
    writeFilesBarOpen(s, false)
    expect(s.getItem('pa.session.filesBarOpen')).toBe('0')
  })
  it('clamps width to 50–85 and rejects garbage', () => {
    expect(clampMainPct(40)).toBe(50)
    expect(clampMainPct(90)).toBe(85)
    expect(clampMainPct(70)).toBe(70)
    const s = mem()
    s.setItem('pa.session.filesBarWidthPct', 'nope')
    expect(readFilesBarWidthPct(s)).toBe(70)
    writeFilesBarWidthPct(s, 12)
    expect(readFilesBarWidthPct(s)).toBe(50)
  })
  it('null storage is safe', () => {
    expect(readFilesBarOpen(null)).toBe(false)
    expect(readFilesBarWidthPct(undefined)).toBe(70)
    writeFilesBarOpen(null, true)
    writeFilesBarWidthPct(undefined, 60)
  })
})
