// web/src/routes/HomePage.test.ts
import { render, screen, waitFor } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import HomePage from './HomePage.svelte'
import { api } from '../lib/api/client'

vi.mock('../lib/api/client', () => ({ api: { get: vi.fn() } }))

describe('HomePage', () => {
  beforeEach(() => vi.mocked(api.get).mockReset())

  it('shows due summary and only recent unfiled projects', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '2026-08-19T00:00:00Z', due_count: 3, projects: [
      { id: 'loose', name: 'Inbox', note_count: 1 },
      { id: 'vaulted', name: 'Training', vault_id: 'health', vault_name: 'HEALTH', note_count: 2 },
    ] })
    render(HomePage)
    expect(await screen.findByText('3 items due')).toBeInTheDocument()
    expect(screen.getByText('Inbox')).toBeInTheDocument()
    expect(screen.queryByText('Training')).not.toBeInTheDocument()
  })

  it('is friendly when no projects or reviews are due', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '2026-08-19T00:00:00Z', projects: [] })
    render(HomePage)
    await waitFor(() => expect(screen.getByText('You’re all caught up')).toBeInTheDocument())
    expect(screen.getByText('No unfiled projects yet')).toBeInTheDocument()
  })
})
