// web/src/routes/ProjectsPage.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectsPage from './ProjectsPage.svelte'
import { api } from '../lib/api/client'
vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } }))

describe('ProjectsPage', () => {
  beforeEach(() => vi.clearAllMocks())
  it('shows only searched unfiled projects', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [
      { id: 'a', name: 'Alpha', note_count: 0 }, { id: 'b', name: 'Beta', vault_id: 'v1', vault_name: 'WORK', note_count: 0 },
    ] })
    render(ProjectsPage)
    expect(await screen.findByText('Alpha')).toBeInTheDocument()
    expect(screen.queryByText('Beta')).not.toBeInTheDocument()
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'none' } })
    expect(screen.getByText('No matching projects')).toBeInTheDocument()
  })
  it('creates an unfiled project', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [] })
    vi.mocked(api.post).mockResolvedValue({ id: 'new', name: 'Inbox', vault_id: null, note_count: 0 })
    render(ProjectsPage)
    await fireEvent.click(await screen.findByRole('button', { name: 'New project' }))
    await fireEvent.input(screen.getByLabelText('Project name'), { target: { value: 'Inbox' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Create project' }))
    expect(api.post).toHaveBeenCalledWith('/api/v1/projects', { name: 'Inbox', vault_id: null })
  })
})
