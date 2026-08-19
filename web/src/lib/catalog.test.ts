// web/src/lib/catalog.test.ts
import { describe, expect, it } from 'vitest'
import type { Project } from './api/types'
import { filterByQuery, filterProjectsByContext, filterReviewByProjectIds, isUnfiled } from './catalog'

const projects: Project[] = [
  { id: 'p0', name: 'Inbox', note_count: 0 },
  { id: 'p1', name: 'Loose notes', vault_id: '', note_count: 2 },
  { id: 'p2', name: 'Training Plan', vault_id: 'health', vault_name: 'HEALTH', note_count: 4 },
  { id: 'p3', name: 'Budget', vault_id: 'finance', vault_name: 'FINANCE', note_count: 1 },
]

describe('catalog helpers', () => {
  it('treats missing, null, and empty vault IDs as unfiled', () => {
    expect(isUnfiled({})).toBe(true)
    expect(isUnfiled({ vault_id: null })).toBe(true)
    expect(isUnfiled({ vault_id: '' })).toBe(true)
    expect(isUnfiled({ vault_id: 'health' })).toBe(false)
  })

  it('returns only unfiled projects in global context', () => {
    // Canonical ShellContext uses mode (not kind)
    expect(filterProjectsByContext(projects, { mode: 'global' }).map((p) => p.id)).toEqual(['p0', 'p1'])
  })

  it('returns only exact vault matches in vault context', () => {
    expect(
      filterProjectsByContext(projects, { mode: 'vault', vaultId: 'health', vaultName: 'HEALTH' }).map((p) => p.id),
    ).toEqual(['p2'])
  })

  it('trims and matches names case-insensitively without mutating input', () => {
    const result = filterByQuery(projects, '  PLAN ')
    expect(result.map((p) => p.id)).toEqual(['p2'])
    expect(projects).toHaveLength(4)
    expect(filterByQuery(projects, '   ')).toEqual(projects)
  })

  it('filters review items by project id set', () => {
    const items = [
      { id: 'r1', project_id: 'p0' },
      { id: 'r2', project_id: 'p2' },
      { id: 'r3', project_id: 'missing' },
    ]
    expect(filterReviewByProjectIds(items, new Set(['p0', 'p2'])).map((i) => i.id)).toEqual(['r1', 'r2'])
  })
})
