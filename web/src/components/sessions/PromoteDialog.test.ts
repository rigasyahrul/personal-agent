// web/src/components/sessions/PromoteDialog.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import PromoteDialog from './PromoteDialog.svelte'
import { api } from '../../lib/api'

vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getProject: vi.fn(),
      promoteSession: vi.fn(),
    },
  }
})

const source = { path: 'draft.md', kind: 'file' as const }

afterEach(cleanup)

describe('PromoteDialog', () => {
  beforeEach(() => {
    vi.mocked(api.getProject).mockReset().mockResolvedValue({ id: 'p1', name: 'Sleep', note_count: 0 })
    vi.mocked(api.promoteSession).mockReset().mockResolvedValue({ operation_id: 'op1' })
    // jsdom dialog polyfill
    if (!HTMLDialogElement.prototype.showModal) {
      HTMLDialogElement.prototype.showModal = function showModal() {
        this.setAttribute('open', '')
      }
      HTMLDialogElement.prototype.close = function close() {
        this.removeAttribute('open')
      }
    }
  })

  it('posts promote with captured source and idempotency key', async () => {
    let key = 0
    const onsuccess = vi.fn()
    render(PromoteDialog, {
      props: {
        open: true,
        sessionId: 's1',
        projectId: 'p1',
        source,
        uuid: () => `key-${++key}`,
        onsuccess,
      },
    })
    expect(await screen.findByRole('heading', { name: 'Save to source' })).toBeInTheDocument()
    const target = screen.getByLabelText(/Target path/i) as HTMLInputElement
    await fireEvent.input(target, { target: { value: 'notes/draft.md' } })
    await fireEvent.click(screen.getByLabelText('bites'))
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => {
      expect(api.promoteSession).toHaveBeenCalledWith(
        's1',
        {
          workspace_path: 'draft.md',
          target_relative_path: 'notes/draft.md',
          review_mode: 'bites',
        },
        'key-1',
      )
    })
    expect(onsuccess).toHaveBeenCalledWith('op1')
  })

  it('rejects non-md targets and keeps the dialog open', async () => {
    render(PromoteDialog, {
      props: { open: true, sessionId: 's1', projectId: 'p1', source },
    })
    const target = await screen.findByLabelText(/Target path/i)
    await fireEvent.input(target, { target: { value: 'nope.txt' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/must end in \.md/i)
    expect(api.promoteSession).not.toHaveBeenCalled()
  })
})
