// web/src/routes/VaultReviewPage.test.ts
import { cleanup, render, screen } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import VaultReviewPage from './VaultReviewPage.svelte'
import { api } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      get: vi.fn(),
      getReviewQueue: vi.fn(),
    },
  }
})

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

afterEach(cleanup)

describe('VaultReviewPage', () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset()
    vi.mocked(api.getReviewQueue).mockReset()
  })

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
      throw new Error(`unexpected ${path}`)
    })
    vi.mocked(api.getReviewQueue).mockResolvedValue({
      scope: 'all',
      items: [vaultItem, otherItem],
      caught_up: false,
    })

    render(VaultReviewPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })

    expect(await screen.findByText('Vault prompt')).toBeInTheDocument()
    expect(screen.queryByText('Other prompt')).not.toBeInTheDocument()
    expect(api.getReviewQueue).toHaveBeenCalledWith('all')
  })

  it('shows caught up when no vault items remain', async () => {
    vi.mocked(api.get).mockImplementation(async (path: string) => {
      if (path === '/api/v1/home') {
        return {
          generated_at: '',
          projects: [{ id: 'p-v', name: 'Sleep', vault_id: 'v1', note_count: 0 }],
        }
      }
      throw new Error(`unexpected ${path}`)
    })
    vi.mocked(api.getReviewQueue).mockResolvedValue({
      scope: 'all',
      items: [otherItem],
      caught_up: false,
    })
    render(VaultReviewPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })
    expect(await screen.findByText(/caught up/i)).toBeInTheDocument()
  })
})
