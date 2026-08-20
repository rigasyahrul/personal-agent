// web/src/routes/VaultProjectsPage.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VaultProjectsPage from './VaultProjectsPage.svelte'
import { api } from '../lib/api/client'

vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } }))

const vaultProject = {
  id: 'p-v',
  name: 'Sleep',
  vault_id: 'v1',
  vault_name: 'HEALTH',
  note_count: 0,
}
const other = { id: 'p-o', name: 'Budget', vault_id: 'v2', vault_name: 'WORK', note_count: 0 }
const unfiled = { id: 'p-u', name: 'Inbox', note_count: 0 }

describe('VaultProjectsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    if (!HTMLDialogElement.prototype.showModal) {
      HTMLDialogElement.prototype.showModal = function showModal() {
        this.setAttribute('open', '')
      }
      HTMLDialogElement.prototype.close = function close() {
        this.removeAttribute('open')
      }
    }
  })

  it('lists only vault projects and searches by name', async () => {
    vi.mocked(api.get).mockResolvedValue({
      generated_at: '',
      projects: [vaultProject, other, unfiled],
    })
    render(VaultProjectsPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })
    expect(await screen.findByText('Sleep')).toBeInTheDocument()
    expect(screen.queryByText('Budget')).not.toBeInTheDocument()
    expect(screen.queryByText('Inbox')).not.toBeInTheDocument()
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'none' } })
    expect(screen.getByText('No matching projects')).toBeInTheDocument()
  })

  it('opens New project in a modal, locks the vault, and submits vault_id', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [] })
    vi.mocked(api.post).mockResolvedValue({
      id: 'new',
      name: 'Sleep',
      vault_id: 'v1',
      note_count: 0,
    })
    render(VaultProjectsPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })
    await fireEvent.click(await screen.findByRole('button', { name: /new project/i }))
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'New project' })).toBeInTheDocument()
    const vaultField = screen.getByLabelText('Vault')
    expect(vaultField).toHaveValue('HEALTH')
    expect(vaultField).toBeDisabled()
    await fireEvent.input(screen.getByLabelText('Project name'), { target: { value: 'Sleep' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Create project' }))
    expect(api.post).toHaveBeenCalledWith('/api/v1/projects', { name: 'Sleep', vault_id: 'v1' })
  })

  it('renders name-first rows not entity-card grid', async () => {
    vi.mocked(api.get).mockResolvedValue({
      generated_at: '',
      projects: [
        { id: 'p1', name: 'Project 1', vault_id: 'v1', vault_name: 'HEALTH', note_count: 2 },
        { id: 'p2', name: 'Project 2', vault_id: 'v1', vault_name: 'HEALTH', note_count: 0 },
      ],
    })
    render(VaultProjectsPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })
    expect(await screen.findByRole('button', { name: /Project 1/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Project 2/i })).toBeInTheDocument()
    expect(document.querySelector('.entity-card')).toBeNull()
    expect(document.querySelector('.catalog-grid')).toBeNull()
    expect(document.querySelector('.name-row')).toBeTruthy()
    expect(document.querySelector('.name-list')).toBeTruthy()
  })

  it('New project opens dialog', async () => {
    vi.mocked(api.get).mockResolvedValue({
      generated_at: '',
      projects: [{ id: 'p1', name: 'Project 1', vault_id: 'v1', note_count: 0 }],
    })
    render(VaultProjectsPage, { props: { vaultId: 'v1', vaultName: 'HEALTH' } })
    await screen.findByRole('button', { name: /Project 1/i })
    await fireEvent.click(screen.getByRole('button', { name: /new project/i }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})
