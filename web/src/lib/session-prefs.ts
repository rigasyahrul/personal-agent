// web/src/lib/session-prefs.ts
export const FILES_BAR_OPEN_KEY = 'pa.session.filesBarOpen'
export const FILES_BAR_WIDTH_KEY = 'pa.session.filesBarWidthPct'
export const DEFAULT_MAIN_PCT = 70
export const MIN_MAIN_PCT = 50
export const MAX_MAIN_PCT = 85

export function clampMainPct(pct: number): number {
  if (!Number.isFinite(pct)) return DEFAULT_MAIN_PCT
  if (pct < MIN_MAIN_PCT) return MIN_MAIN_PCT
  if (pct > MAX_MAIN_PCT) return MAX_MAIN_PCT
  return pct
}

export function readFilesBarOpen(storage: Storage | null | undefined): boolean {
  if (!storage) return false
  return storage.getItem(FILES_BAR_OPEN_KEY) === '1'
}

export function writeFilesBarOpen(storage: Storage | null | undefined, open: boolean): void {
  if (!storage) return
  storage.setItem(FILES_BAR_OPEN_KEY, open ? '1' : '0')
}

export function readFilesBarWidthPct(storage: Storage | null | undefined): number {
  if (!storage) return DEFAULT_MAIN_PCT
  const raw = storage.getItem(FILES_BAR_WIDTH_KEY)
  if (raw == null) return DEFAULT_MAIN_PCT
  const n = Number(raw)
  if (!Number.isFinite(n)) return DEFAULT_MAIN_PCT
  return clampMainPct(n)
}

export function writeFilesBarWidthPct(storage: Storage | null | undefined, pct: number): void {
  if (!storage) return
  storage.setItem(FILES_BAR_WIDTH_KEY, String(clampMainPct(pct)))
}
