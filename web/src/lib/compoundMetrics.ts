/** Time-to-finish in ms from RFC3339/ISO timestamps. Null if either is missing/invalid. */
export function compoundTimeToFinishMs(createdAt?: string, finishedAt?: string): number | null {
  const created = Date.parse(createdAt ?? '')
  const finished = Date.parse(finishedAt ?? '')
  if (Number.isNaN(created) || Number.isNaN(finished)) return null
  return finished - created
}
