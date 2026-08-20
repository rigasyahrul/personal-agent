// web/src/lib/review/vault-filter.test.ts
import { describe, expect, it } from 'vitest'
import { filterQueueByProjectIds } from './vault-filter'
import type { ReviewQueue } from '../api/types'

const vaultItem = { id: 'r1', project_id: 'p-v', prompt: 'Vault Q', kind: 'bite' }
const otherItem = { id: 'r2', project_id: 'p-o', prompt: 'Other Q', kind: 'bite' }

describe('filterQueueByProjectIds', () => {
  it('keeps only matching project items and sets caught_up when empty', () => {
    const queue: ReviewQueue = {
      scope: 'all',
      items: [vaultItem, otherItem],
      caught_up: false,
    }
    expect(filterQueueByProjectIds(queue, new Set(['p-v']))).toEqual({
      scope: 'all',
      items: [vaultItem],
      caught_up: false,
    })
    expect(filterQueueByProjectIds(queue, new Set(['missing']))).toEqual({
      scope: 'all',
      items: [],
      caught_up: true,
    })
  })
})
