// web/src/routes/VaultHomePage.test.ts
import { render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VaultHomePage from './VaultHomePage.svelte'
import { api } from '../lib/api/client'

vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } }))

const vaultProject = {
  id: 'p-v',
  name: 'Sleep',
  vault_id: 'v1',
  vault_name: 'HEALTH',
  note_count: 2,
  session_count: 1,
}
const unfiledProject = { id: 'p-u', name: 'Inbox', note_count: 0 }

describe('VaultHomePage', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows only vault summary data and useful actions', async () => {
    vi.mocked(api.get).mockImplementation(async (path: string) => {
      if (path === '/api/v1/home') {
        return { generated_at: '', due_count: 5, projects: [vaultProject, unfiledProject] }
      }
      if (path.startsWith('/api/v1/review/queue')) {
        return { scope: 'all', items: [{ id: 'r1', project_id: 'p-v', prompt: 'Q1' }], caught_up: false }
      }
      throw new Error(`unexpected ${path}`)
    })

    render(VaultHomePage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })

    expect(await screen.findByText('1 project')).toBeInTheDocument()
    expect(screen.getByText('1 due')).toBeInTheDocument()
    expect(screen.getByText('Sleep')).toBeInTheDocument()
    expect(screen.queryByText('Inbox')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: /new project/i })).toHaveAttribute(
      'href',
      '#/vaults/v1/projects?new=1',
    )
  })
})
