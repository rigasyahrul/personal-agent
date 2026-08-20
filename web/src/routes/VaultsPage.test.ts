// web/src/routes/VaultsPage.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VaultsPage from './VaultsPage.svelte'
import { api } from '../lib/api/client'
import { navigate } from '../lib/router'
vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } }))
vi.mock('../lib/router', () => ({ navigate: vi.fn() }))

describe('VaultsPage', () => {
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

  it('searches vaults, shows project count, and enters a vault', async () => {
    vi.mocked(api.get).mockImplementation(async (path) => path === '/api/v1/vaults' ? [{ id: 'v1', name: 'HEALTH', created_at: '', updated_at: '' }] : { projects: [{ id: 'p1', name: 'Training', vault_id: 'v1', note_count: 0 }], generated_at: '' })
    render(VaultsPage); expect(await screen.findByText('1 project')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: /enter health vault/i })); expect(navigate).toHaveBeenCalledWith('#/vaults/v1')
  })

  it('opens New vault in a modal and creates a vault', async () => {
    vi.mocked(api.get).mockResolvedValueOnce([]).mockResolvedValueOnce({ projects: [], generated_at: '' })
    vi.mocked(api.post).mockResolvedValue({ id: 'v2', name: 'WORK', created_at: '', updated_at: '' })
    render(VaultsPage)
    await fireEvent.click(await screen.findByRole('button', { name: 'New vault' }))
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'New vault' })).toBeInTheDocument()
    await fireEvent.input(screen.getByLabelText('Vault name'), { target: { value: 'WORK' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Create vault' }))
    expect(api.post).toHaveBeenCalledWith('/api/v1/vaults', { name: 'WORK' })
    expect(navigate).toHaveBeenCalledWith('#/vaults/v2')
  })

  it('uses craft hierarchy without Global desk eyebrow', async () => {
    vi.mocked(api.get).mockResolvedValueOnce([]).mockResolvedValueOnce({ projects: [], generated_at: '' })
    render(VaultsPage)
    expect(await screen.findByRole('heading', { level: 1, name: 'Vaults' })).toBeInTheDocument()
    expect(screen.queryByText('Global desk')).not.toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'New vault' })[0].className).toMatch(/btn--primary/)
  })
})
