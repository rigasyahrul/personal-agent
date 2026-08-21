/** Short calendar label for session rows, e.g. "May 30". Null if missing/invalid. */
export function formatSessionDate(iso: string | undefined): string | null {
  if (iso == null || iso === '') return null
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return null
  return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' }).format(new Date(t))
}
