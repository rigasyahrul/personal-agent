/** Short calendar label for session rows, e.g. "May 30". Null if missing/invalid. */
export function formatSessionDate(iso: string | undefined): string | null {
  if (iso == null || iso === '') return null
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return null
  return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' }).format(new Date(t))
}

/**
 * Full local datetime for tooltips, e.g. "May 30, 2026 10:36 PM".
 * Null if missing/invalid.
 */
export function formatMessageDateTime(iso: string | undefined): string | null {
  if (iso == null || iso === '') return null
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return null
  const d = new Date(t)
  const datePart = new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  }).format(d)
  const timePart = new Intl.DateTimeFormat('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  }).format(d)
  return `${datePart} ${timePart}`
}
