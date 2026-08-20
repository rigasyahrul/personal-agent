// web/src/lib/vault-scope.test.ts
import { describe, expect, it } from 'vitest'
import type { Project } from './api/types'
import { createVaultProjectInput, filterVaultProjects } from './vault-scope'

const vaultProject: Project = {
  id: 'p-v',
  name: 'Sleep',
  vault_id: 'v1',
  vault_name: 'HEALTH',
  note_count: 0,
}
const otherVaultProject: Project = {
  id: 'p-o',
  name: 'Budget',
  vault_id: 'v2',
  vault_name: 'WORK',
  note_count: 0,
}
const unfiledProject: Project = { id: 'p-u', name: 'Inbox', note_count: 0 }

describe('vault-scope', () => {
  it('filters projects to the active vault only', () => {
    expect(filterVaultProjects([vaultProject, otherVaultProject, unfiledProject], 'v1')).toEqual([
      vaultProject,
    ])
  })

  it('builds create payload with trimmed name and locked vault_id', () => {
    expect(createVaultProjectInput(' Sleep ', 'v1')).toEqual({ name: 'Sleep', vault_id: 'v1' })
  })
})
