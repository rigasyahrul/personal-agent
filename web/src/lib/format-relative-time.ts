/** Relative activity label for session cards. Null if missing/invalid. */
export function formatRelativeTime(iso: string | undefined): string | null {
  if (iso == null || iso === '') return null
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return null

  const diffMs = Date.now() - t
  if (diffMs < 0) return 'just now'

  const sec = Math.floor(diffMs / 1000)
  if (sec < 60) return 'just now'

  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m ago`

  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h ago`

  const day = Math.floor(hr / 24)
  return `${day}d ago`
}
