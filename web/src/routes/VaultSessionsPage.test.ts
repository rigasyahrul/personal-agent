// web/src/routes/VaultSessionsPage.test.ts
import { cleanup, render, screen } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import VaultSessionsPage from './VaultSessionsPage.svelte'
import { api } from '../lib/api/client'

vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } }))

const projectA = { id: 'a', name: 'Alpha', vault_id: 'v1', vault_name: 'HEALTH', note_count: 0 }
const projectB = { id: 'b', name: 'Beta', vault_id: 'v1', vault_name: 'HEALTH', note_count: 0 }
const unfiled = { id: 'u', name: 'Inbox', note_count: 0 }

afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

describe('VaultSessionsPage', () => {
  beforeEach(() => vi.clearAllMocks())

  it('aggregates sessions from vault projects only and never hits a vault sessions API', async () => {
    vi.mocked(api.get).mockImplementation(async (path: string) => {
      if (path === '/api/v1/home') {
        return { generated_at: '', projects: [projectA, projectB, unfiled] }
      }
      if (path === '/api/v1/projects/a/sessions') {
        return [{ id: 's-a', title: 'Alpha chat', status: 'active', provider: 'p', model_id: 'm' }]
      }
      if (path === '/api/v1/projects/b/sessions') {
        return [{ id: 's-b', title: 'Beta chat', status: 'active', provider: 'p', model_id: 'm' }]
      }
      throw new Error(`unexpected ${path}`)
    })

    render(VaultSessionsPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })

    expect(await screen.findByText('Alpha chat')).toBeInTheDocument()
    expect(screen.getByText('Beta chat')).toBeInTheDocument()
    // Project labels appear on each session row meta (and in the picker options).
    expect(screen.getAllByText(/Alpha/).length).toBeGreaterThan(0)
    const paths = vi.mocked(api.get).mock.calls.map(([path]) => path)
    expect(paths.some((p) => String(p).includes('/vaults/'))).toBe(false)
  })

  it('renders sessions as session-card rows with project · model meta and whole-row links', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-20T12:00:00.000Z'))

    vi.mocked(api.get).mockImplementation(async (path: string) => {
      if (path === '/api/v1/home') {
        return { generated_at: '', projects: [projectA] }
      }
      if (path === '/api/v1/projects/a/sessions') {
        return [
          {
            id: 's-a',
            title: 'Alpha chat',
            status: 'active',
            provider: 'openai',
            model_id: 'gpt',
            updated_at: '2026-08-20T11:55:00.000Z',
          },
        ]
      }
      throw new Error(`unexpected ${path}`)
    })

    render(VaultSessionsPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })

    expect(await screen.findByText('Alpha chat')).toBeInTheDocument()
    expect(document.querySelectorAll('.session-card')).toHaveLength(1)
    expect(document.querySelector('.session-card__title')?.textContent).toBe('Alpha chat')
    const meta = document.querySelector('.session-card__meta')
    expect(meta?.textContent).toContain('Alpha')
    expect(meta?.textContent).toContain('openai:gpt')
    expect(meta?.textContent).toContain('5m ago')
    const card = screen.getByRole('link', { name: /Alpha chat/i })
    expect(card).toHaveClass('session-card')
    expect(card).toHaveAttribute('href', '#/projects/a/sessions')
  })

  it('shows create-a-project guidance when the vault has no projects', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [unfiled] })
    render(VaultSessionsPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })
    expect(await screen.findByText(/create a project first/i)).toBeInTheDocument()
  })

  it('navigates new session to the selected project sessions route', async () => {
    vi.mocked(api.get).mockImplementation(async (path: string) => {
      if (path === '/api/v1/home') return { generated_at: '', projects: [projectA] }
      if (path === '/api/v1/projects/a/sessions') return []
      throw new Error(`unexpected ${path}`)
    })
    render(VaultSessionsPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })
    await screen.findByText(/no sessions yet/i)
    const link = screen.getByRole('link', { name: /new session/i })
    expect(link).toHaveAttribute('href', '#/projects/a/sessions')
  })
})
