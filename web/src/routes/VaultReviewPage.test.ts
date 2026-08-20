// web/src/routes/VaultReviewPage.test.ts
import { render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VaultReviewPage from './VaultReviewPage.svelte'
import { api } from '../lib/api/client'

vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } }))

const vaultItem = {
  id: 'r1',
  project_id: 'p-v',
  prompt: 'Vault prompt',
  kind: 'bite',
  answer: 'A',
  note_id: 'n1',
  row_version: 1,
}
const otherItem = {
  id: 'r2',
  project_id: 'p-o',
  prompt: 'Other prompt',
  kind: 'bite',
  answer: 'B',
  note_id: 'n2',
  row_version: 1,
}

describe('VaultReviewPage', () => {
  beforeEach(() => vi.clearAllMocks())

  it('requests all and passes only vault items to the runner', async () => {
    vi.mocked(api.get).mockImplementation(async (path: string) => {
      if (path === '/api/v1/home') {
        return {
          generated_at: '',
          projects: [
            { id: 'p-v', name: 'Sleep', vault_id: 'v1', vault_name: 'HEALTH', note_count: 0 },
            { id: 'p-o', name: 'Budget', vault_id: 'v2', vault_name: 'WORK', note_count: 0 },
          ],
        }
      }
      if (path.startsWith('/api/v1/review/queue')) {
        return { scope: 'all', items: [vaultItem, otherItem], caught_up: false }
      }
      throw new Error(`unexpected ${path}`)
    })

    render(VaultReviewPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })

    expect(await screen.findByText('Vault prompt')).toBeInTheDocument()
    expect(screen.queryByText('Other prompt')).not.toBeInTheDocument()
    expect(api.get).toHaveBeenCalledWith(expect.stringContaining('/api/v1/review/queue?scope=all'))
  })

  it('shows caught up when no vault items remain', async () => {
    vi.mocked(api.get).mockImplementation(async (path: string) => {
      if (path === '/api/v1/home') {
        return {
          generated_at: '',
          projects: [{ id: 'p-v', name: 'Sleep', vault_id: 'v1', note_count: 0 }],
        }
      }
      if (path.startsWith('/api/v1/review/queue')) {
        return { scope: 'all', items: [otherItem], caught_up: false }
      }
      throw new Error(`unexpected ${path}`)
    })
    render(VaultReviewPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })
    expect(await screen.findByText(/caught up/i)).toBeInTheDocument()
  })
})
