// web/src/lib/review/vault-filter.ts
import type { ReviewQueue } from '../api/types'

export function filterQueueByProjectIds(
  queue: ReviewQueue,
  ids: ReadonlySet<string>,
): ReviewQueue {
  const items = queue.items.filter((item) => ids.has(item.project_id))
  return { ...queue, items, caught_up: items.length === 0 }
}
