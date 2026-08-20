// web/src/lib/vault-sessions.test.ts
import { describe, expect, it, vi } from 'vitest'
import { loadVaultSessions } from './vault-sessions'

const projectA = { id: 'a', name: 'Alpha', vault_id: 'v1', vault_name: 'HEALTH', note_count: 0 }
const projectB = { id: 'b', name: 'Beta', vault_id: 'v1', vault_name: 'HEALTH', note_count: 0 }
const unfiledProject = { id: 'u', name: 'Inbox', note_count: 0 }
const sessionA = { id: 's-a', title: 'A session', status: 'active', provider: 'p', model_id: 'm' }
const sessionB = { id: 's-b', title: 'B session', status: 'active', provider: 'p', model_id: 'm' }

describe('loadVaultSessions', () => {
  it('calls sessions once per vault project and annotates results', async () => {
    const api = {
      listProjects: vi.fn().mockResolvedValue([projectA, projectB, unfiledProject]),
      listProjectSessions: vi
        .fn()
        .mockImplementation(async (id: string) => (id === 'a' ? [sessionA] : [sessionB])),
    }
    const result = await loadVaultSessions('v1', api)
    expect(api.listProjectSessions.mock.calls.map(([id]) => id).sort()).toEqual(['a', 'b'])
    expect(result.sessions).toEqual([
      { ...sessionA, project_id: 'a', project_name: projectA.name },
      { ...sessionB, project_id: 'b', project_name: projectB.name },
    ])
    expect(result.failures).toEqual([])
  })

  it('keeps successful projects and reports a partial failure', async () => {
    const api = {
      listProjects: vi.fn().mockResolvedValue([projectA, projectB]),
      listProjectSessions: vi
        .fn()
        .mockRejectedValueOnce(new Error('offline'))
        .mockResolvedValueOnce([sessionB]),
    }
    expect((await loadVaultSessions('v1', api)).failures).toEqual([projectA.name])
  })
})
