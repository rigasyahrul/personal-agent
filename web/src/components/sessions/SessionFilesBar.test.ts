// web/src/components/sessions/SessionFilesBar.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SessionFilesBar from './SessionFilesBar.svelte'
import SessionChat from './SessionChat.svelte'
import { api } from '../../lib/api'

vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      listMessages: vi.fn(),
      currentRun: vi.fn(),
      sendMessage: vi.fn(),
      workspaceTree: vi.fn(),
      workspaceFile: vi.fn(),
      operationStatus: vi.fn(),
      promoteSession: vi.fn(),
      retryReviewPending: vi.fn(),
      getProject: vi.fn(),
    },
  }
})

const baseSession = {
  id: 's1',
  title: 'Chat',
  status: 'idle',
  provider: 'openai',
  model_id: 'gpt',
}

afterEach(cleanup)

describe('SessionFilesBar', () => {
  beforeEach(() => {
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({
      entries: [
        { path: 'draft.md', kind: 'file' },
        { path: 'notes/raw.txt', kind: 'file' },
        { path: 'notes', kind: 'directory' },
      ],
    })
    vi.mocked(api.workspaceFile).mockReset()
  })

  it('renders tree paths after load', async () => {
    render(SessionFilesBar, { props: { sessionId: 's1' } })
    expect(await screen.findByRole('button', { name: 'draft.md' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'notes/raw.txt' })).toBeVisible()
    expect(screen.getByLabelText('Session files')).toBeVisible()
  })

  it('filters tree with case-insensitive search', async () => {
    render(SessionFilesBar, { props: { sessionId: 's1' } })
    await screen.findByRole('button', { name: 'draft.md' })
    const search = screen.getByRole('searchbox', { name: /search files/i })
    await fireEvent.input(search, { target: { value: 'RAW' } })
    expect(screen.queryByRole('button', { name: 'draft.md' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'notes/raw.txt' })).toBeVisible()
  })

  it('calls onopen for files and does not open directories', async () => {
    const onopen = vi.fn()
    render(SessionFilesBar, {
      props: {
        sessionId: 's1',
        onopen,
        messages: [],
      },
    })
    await fireEvent.click(await screen.findByRole('button', { name: 'draft.md' }))
    expect(onopen).toHaveBeenCalledWith('draft.md')
    onopen.mockClear()
    const dir = screen.getByRole('button', { name: 'notes' })
    expect(dir).toBeDisabled()
    await fireEvent.click(dir)
    expect(onopen).not.toHaveBeenCalled()
  })

  it('highlights activePath and marks changed tool paths', async () => {
    render(SessionFilesBar, {
      props: {
        sessionId: 's1',
        activePath: 'draft.md',
        messages: [{ sequence: 1, role: 'tool', content: '', changed_path: 'notes/raw.txt' }],
      },
    })
    const active = await screen.findByRole('button', { name: 'draft.md' })
    expect(active.className).toMatch(/tree-item--active/)
    const changed = screen.getByRole('button', { name: 'notes/raw.txt' })
    expect(changed.className).toMatch(/bg-amber-50|tree-item--changed/)
  })

  it('has no promote button and no preview pre', async () => {
    render(SessionFilesBar, { props: { sessionId: 's1' } })
    await screen.findByRole('button', { name: 'draft.md' })
    expect(screen.queryByRole('button', { name: 'Save to source' })).not.toBeInTheDocument()
    expect(document.querySelector('pre.workspace-preview')).toBeNull()
    expect(document.querySelector('pre')).toBeNull()
  })

  it('shows empty copy when tree has no entries', async () => {
    vi.mocked(api.workspaceTree).mockResolvedValue({ entries: [] })
    render(SessionFilesBar, { props: { sessionId: 's1' } })
    expect(await screen.findByText(/no files yet/i)).toBeVisible()
  })

  it('shows inline error when tree load fails', async () => {
    vi.mocked(api.workspaceTree).mockRejectedValue(new Error('tree boom'))
    render(SessionFilesBar, { props: { sessionId: 's1' } })
    expect(await screen.findByRole('alert')).toHaveTextContent('tree boom')
  })

  it('refreshes tree after a newly polled tool message changes a path', async () => {
    let messages: Array<{ sequence: number; role: string; content: string; changed_path?: string }> =
      []
    vi.mocked(api.listMessages).mockReset().mockImplementation(async () => messages as never)
    vi.mocked(api.currentRun).mockReset().mockResolvedValue(null)
    vi.mocked(api.getProject).mockResolvedValue({ id: 'p1', name: 'P', note_count: 0 })

    const mem = (() => {
      const m = new Map<string, string>()
      return {
        get length() {
          return m.size
        },
        clear: () => m.clear(),
        getItem: (k: string) => m.get(k) ?? null,
        setItem: (k: string, v: string) => {
          m.set(k, String(v))
        },
        removeItem: (k: string) => {
          m.delete(k)
        },
        key: () => null,
      } satisfies Storage
    })()
    // Pref open so files bar mounts without extra click race
    mem.setItem('pa.session.filesBarOpen', '1')

    const { component } = render(SessionChat, {
      props: {
        session: { ...baseSession, tool_grants: { workspace_files: true } },
        projectId: 'p1',
        pollInterval: 60_000,
        storage: mem,
      },
    })
    expect(await screen.findByLabelText('Session files')).toBeVisible()
    await waitFor(() => expect(api.workspaceTree).toHaveBeenCalledTimes(1))
    messages = [{ sequence: 1, role: 'tool', content: '', changed_path: 'new.txt' }]
    const poll = (component as unknown as { poll: () => Promise<void> }).poll
    await poll()
    await waitFor(() => expect(api.workspaceTree).toHaveBeenCalledTimes(2))
  })
})
