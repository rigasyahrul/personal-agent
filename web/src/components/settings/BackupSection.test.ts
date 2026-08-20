// web/src/components/settings/BackupSection.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import BackupSection from './BackupSection.svelte'
import { api } from '../../lib/api'
import type { Settings } from '../../lib/api/types'

vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      updateSettings: vi.fn(),
      listBackups: vi.fn(),
      createBackup: vi.fn(),
    },
  }
})

const baseSettings: Settings = {
  timezone: 'UTC',
  default_provider: 'openai',
  default_model_id: 'gpt',
  backup_schedule: 'off',
  backup: {
    sink_configured: false,
    schedule: 'off',
    last_success: null,
    last_failure: {
      id: 'f1',
      status: 'failed',
      completed_at: '2026-08-19T12:00:00Z',
      error: 'disk full',
    },
  },
}

afterEach(cleanup)

describe('BackupSection', () => {
  beforeEach(() => {
    vi.mocked(api.updateSettings).mockReset().mockResolvedValue({
      ...baseSettings,
      backup_schedule: 'daily',
    })
    vi.mocked(api.listBackups).mockReset().mockResolvedValue({
      backups: [
        {
          id: 'f1',
          status: 'failed',
          completed_at: '2026-08-19T12:00:00Z',
          error: 'disk full',
        },
      ],
      last_failure: {
        id: 'f1',
        status: 'failed',
        completed_at: '2026-08-19T12:00:00Z',
        error: 'disk full',
      },
    })
    vi.mocked(api.createBackup).mockReset().mockResolvedValue({
      id: 'b2',
      status: 'succeeded',
      completed_at: '2026-08-19T13:00:00Z',
    })
  })

  it('shows sink status and newer last failure', async () => {
    render(BackupSection, { props: { settings: baseSettings } })
    expect(await screen.findByText(/Remote sink configured: no/i)).toBeInTheDocument()
    expect(screen.getByText(/Last attempt failed: disk full/i)).toBeInTheDocument()
  })

  it('lists backup history after load', async () => {
    render(BackupSection, { props: { settings: baseSettings } })
    expect(await screen.findByText(/failed/i)).toBeInTheDocument()
    expect(api.listBackups).toHaveBeenCalled()
  })

  it('disables only Backup now while running', async () => {
    let resolveBackup: (value: { id: string; status: string }) => void = () => {}
    vi.mocked(api.createBackup).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveBackup = resolve
        }),
    )
    render(BackupSection, { props: { settings: baseSettings } })
    const btn = await screen.findByRole('button', { name: 'Backup now' })
    await fireEvent.click(btn)
    expect(btn).toBeDisabled()
    expect(screen.getByLabelText('Schedule')).not.toBeDisabled()
    resolveBackup({ id: 'b2', status: 'succeeded' })
    await waitFor(() => expect(btn).not.toBeDisabled())
  })

  it('keeps history and shows error when backup fails', async () => {
    vi.mocked(api.createBackup).mockRejectedValue(new Error('backup failed'))
    render(BackupSection, { props: { settings: baseSettings } })
    await fireEvent.click(await screen.findByRole('button', { name: 'Backup now' }))
    expect(await screen.findByText('backup failed')).toBeVisible()
    // history still present (also shown in the newer-failure banner)
    expect(screen.getAllByText(/disk full/i).length).toBeGreaterThanOrEqual(1)
  })
})
