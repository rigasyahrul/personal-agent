// web/src/components/catalog-components.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { describe, expect, it, vi } from 'vitest'
import EmptyState from './EmptyState.svelte'
import ProjectCard from './ProjectCard.svelte'
import SearchField from './SearchField.svelte'
import VaultCard from './VaultCard.svelte'

describe('catalog components', () => {
  it('renders an actionable empty state', async () => {
    const onaction = vi.fn()
    render(EmptyState, { title: 'No projects yet', description: 'Create your first project.', actionLabel: 'New project', onaction })
    await fireEvent.click(screen.getByRole('button', { name: 'New project' }))
    expect(onaction).toHaveBeenCalledOnce()
  })

  it('labels and updates search input', async () => {
    render(SearchField, { value: '', label: 'Search vaults' })
    await fireEvent.input(screen.getByRole('searchbox', { name: 'Search vaults' }), { target: { value: 'health' } })
    expect(screen.getByRole<HTMLInputElement>('searchbox').value).toBe('health')
  })

  it('shows vault name and project metrics on a vaulted project', () => {
    render(ProjectCard, { project: { id: 'p1', name: 'Training', vault_id: 'v1', vault_name: 'HEALTH', note_count: 3, session_count: 2, due_count: 1 }, onclick: vi.fn() })
    expect(screen.getByText('HEALTH')).toBeInTheDocument()
    expect(screen.getByText('3 notes')).toBeInTheDocument()
    expect(screen.getByText('2 sessions')).toBeInTheDocument()
    expect(screen.getByText('1 due')).toBeInTheDocument()
  })

  it('does not invent a badge for an unfiled project', () => {
    render(ProjectCard, { project: { id: 'p0', name: 'Inbox', vault_id: null, note_count: 0 }, onclick: vi.fn() })
    expect(screen.queryByText('Unfiled')).not.toBeInTheDocument()
  })

  it('renders a vault card as a named button with project count', async () => {
    const onclick = vi.fn()
    render(VaultCard, { vault: { id: 'v1', name: 'HEALTH', created_at: '', updated_at: '' }, projectCount: 4, onclick })
    await fireEvent.click(screen.getByRole('button', { name: /enter health vault/i }))
    expect(screen.getByText('4 projects')).toBeInTheDocument()
    expect(onclick).toHaveBeenCalledOnce()
  })
})
