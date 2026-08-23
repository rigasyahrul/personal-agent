// web/src/routes/SettingsPage.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsPage from './SettingsPage.svelte'
import { api } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getSettings: vi.fn(),
      updateSettings: vi.fn(),
      listBackups: vi.fn(),
      createBackup: vi.fn(),
      getGlobalInstruction: vi.fn(),
      putGlobalInstruction: vi.fn(),
    },
  }
})

const settings = {
  timezone: 'Asia/Jakarta',
  default_provider: 'openai',
  default_model_id: 'gpt',
  backup_schedule: 'off',
  backup: {
    sink_configured: true,
    schedule: 'off',
    last_success: {
      id: 'b1',
      status: 'succeeded',
      completed_at: '2026-08-18T10:00:00Z',
    },
    last_failure: null,
  },
}

afterEach(cleanup)

describe('SettingsPage', () => {
  beforeEach(() => {
    vi.mocked(api.getSettings).mockReset().mockResolvedValue(settings)
    vi.mocked(api.updateSettings).mockReset().mockResolvedValue({
      ...settings,
      backup_schedule: 'daily',
    })
    vi.mocked(api.listBackups).mockReset().mockResolvedValue({
      backups: [
        {
          id: 'b1',
          status: 'succeeded',
          completed_at: '2026-08-18T10:00:00Z',
        },
      ],
      last_success: {
        id: 'b1',
        status: 'succeeded',
        completed_at: '2026-08-18T10:00:00Z',
      },
    })
    vi.mocked(api.createBackup).mockReset().mockResolvedValue({
      id: 'b2',
      status: 'succeeded',
      completed_at: '2026-08-19T10:00:00Z',
    })
    vi.mocked(api.getGlobalInstruction).mockReset().mockResolvedValue({ content: 'Be kind.\n' })
    vi.mocked(api.putGlobalInstruction).mockReset().mockResolvedValue({ content: 'Be kinder.\n' })
  })

  it('saves schedule while preserving the complete settings payload', async () => {
    render(SettingsPage)
    const select = await screen.findByLabelText('Schedule')
    await fireEvent.change(select, { target: { value: 'daily' } })
    await waitFor(() => {
      expect(api.updateSettings).toHaveBeenCalledWith({
        timezone: 'Asia/Jakarta',
        default_provider: 'openai',
        default_model_id: 'gpt',
        backup_schedule: 'daily',
      })
    })
    expect(await screen.findByText('Schedule saved.')).toBeVisible()
  })

  it('runs backup, refreshes history, and reports completion', async () => {
    render(SettingsPage)
    expect(await screen.findByRole('button', { name: 'Backup now' })).toBeInTheDocument()
    // initial listBackups
    expect(api.listBackups).toHaveBeenCalledTimes(1)
    await fireEvent.click(screen.getByRole('button', { name: 'Backup now' }))
    await waitFor(() => {
      expect(api.createBackup).toHaveBeenCalledTimes(1)
      expect(api.listBackups).toHaveBeenCalledTimes(2)
    })
    expect(await screen.findByText('Backup completed.')).toBeVisible()
  })

  it('shows timezone and model values', async () => {
    render(SettingsPage)
    expect(await screen.findByText('Asia/Jakarta')).toBeInTheDocument()
    expect(screen.getByText('openai')).toBeInTheDocument()
    expect(screen.getByText('gpt')).toBeInTheDocument()
  })

  it('keeps settings on schedule save failure and shows inline error', async () => {
    vi.mocked(api.updateSettings).mockRejectedValue(new Error('save failed'))
    render(SettingsPage)
    const select = await screen.findByLabelText('Schedule')
    await fireEvent.change(select, { target: { value: 'daily' } })
    expect(await screen.findByText('save failed')).toBeVisible()
    expect(screen.getByText('Asia/Jakarta')).toBeInTheDocument()
  })

  it('saves a global instruction with PUT', async () => {
    render(SettingsPage)
    const textarea = await screen.findByRole('textbox', { name: 'SOUL' })
    expect(textarea).toHaveValue('Be kind.\n')
    await fireEvent.input(textarea, { target: { value: 'Be kinder.\n' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => {
      expect(api.putGlobalInstruction).toHaveBeenCalledWith('soul', 'Be kinder.\n')
    })
  })

  it('offers retry when initial load fails', async () => {
    vi.mocked(api.getSettings).mockRejectedValueOnce(new Error('offline'))
    render(SettingsPage)
    expect(await screen.findByRole('alert')).toHaveTextContent('offline')
    vi.mocked(api.getSettings).mockResolvedValue(settings)
    await fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('Asia/Jakarta')).toBeInTheDocument()
  })
})
