// web/src/components/sessions/SessionChat.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
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

const session = {
  id: 's1',
  title: 'Chat',
  status: 'idle',
  provider: 'openai',
  model_id: 'gpt',
}

const deferred = <T>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((yes, no) => {
    resolve = yes
    reject = no
  })
  return { promise, resolve, reject }
}

afterEach(cleanup)

describe('SessionChat', () => {
  beforeEach(() => {
    vi.mocked(api.listMessages).mockReset().mockResolvedValue([
      { sequence: 1, role: 'user', content: 'cached' },
    ])
    vi.mocked(api.currentRun).mockReset().mockResolvedValue(null)
    vi.mocked(api.sendMessage).mockReset()
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({ entries: [] })
    vi.mocked(api.getProject).mockResolvedValue({ id: 'p1', name: 'P', note_count: 0 })
  })

  it('posts once with one stable request_key and retains draft after failure', async () => {
    const gate = deferred<null>()
    vi.mocked(api.sendMessage).mockReturnValue(gate.promise)
    render(SessionChat, {
      props: {
        session,
        projectId: 'p1',
        pollInterval: 60_000,
        uuid: () => 'stable-key',
      },
    })
    const composer = await screen.findByLabelText('Message')
    await fireEvent.input(composer, { target: { value: 'draft' } })
    const form = composer.closest('form')!
    const first = fireEvent.submit(form)
    const second = fireEvent.submit(form)
    await first
    await second
    expect(api.sendMessage).toHaveBeenCalledTimes(1)
    expect(api.sendMessage).toHaveBeenCalledWith(session.id, {
      content: 'draft',
      request_key: 'stable-key',
    })
    gate.reject(new Error('AI unavailable'))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('AI unavailable'))
    expect(composer).toHaveValue('draft')
    expect(screen.getByText('cached')).toBeInTheDocument()
  })

  it('retains cached history when a poll fails', async () => {
    const { component } = render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000 },
    })
    expect(await screen.findByText('cached')).toBeInTheDocument()
    vi.mocked(api.listMessages).mockRejectedValueOnce(new Error('network down'))
    await (component as unknown as { poll: () => Promise<void> }).poll()
    expect(screen.getByText('cached')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('network down')
  })

  it('suppresses duplicate submit while a send is in flight', async () => {
    const gate = deferred<null>()
    vi.mocked(api.sendMessage).mockReturnValue(gate.promise)
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000, uuid: () => 'one-key' },
    })
    const composer = await screen.findByLabelText('Message')
    await fireEvent.input(composer, { target: { value: 'keep draft' } })
    const form = composer.closest('form')!
    const first = fireEvent.submit(form)
    await first
    await fireEvent.submit(form)
    expect(api.sendMessage).toHaveBeenCalledTimes(1)
    gate.reject(new Error('send failed'))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('send failed'))
    expect(composer).toHaveValue('keep draft')
  })

  it('clears draft after a successful send', async () => {
    vi.mocked(api.sendMessage).mockResolvedValue(null)
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000, uuid: () => 'ok-key' },
    })
    const composer = await screen.findByLabelText('Message')
    await fireEvent.input(composer, { target: { value: 'hello' } })
    await fireEvent.submit(composer.closest('form')!)
    await waitFor(() => expect(composer).toHaveValue(''))
  })

  it('renders assistant as bare prose without Assistant label', async () => {
    vi.mocked(api.listMessages).mockResolvedValue([
      { sequence: 1, role: 'user', content: 'hello from me' },
      { sequence: 2, role: 'assistant', content: 'reply from agent' },
    ])
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000 },
    })
    expect(await screen.findByText('hello from me')).toBeInTheDocument()
    expect(screen.getByText('reply from agent')).toBeInTheDocument()
    expect(screen.queryByText('Assistant')).not.toBeInTheDocument()

    const userRow = screen.getByText('hello from me').closest('li')
    const assistantRow = screen.getByText('reply from agent').closest('li')
    expect(userRow).toHaveAttribute('data-role', 'user')
    expect(assistantRow).toHaveAttribute('data-role', 'assistant')
    expect(userRow?.className).toMatch(/message-row--user/)
    expect(assistantRow?.className).toMatch(/message-row--assistant/)
    expect(userRow?.querySelector('.message-bubble--user')).toBeTruthy()
    expect(assistantRow?.querySelector('.message-bubble')).toBeNull()
    expect(assistantRow?.querySelector('.message-prose')).toBeTruthy()
  })

  it('toggles files bar and persists pref', async () => {
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
    const wsSession = {
      ...session,
      tool_grants: { workspace_files: true as const },
    }
    render(SessionChat, {
      props: {
        session: wsSession,
        projectId: 'p1',
        pollInterval: 60_000,
        storage: mem,
      },
    })
    const toggle = await screen.findByRole('button', { name: /show files/i })
    expect(toggle).toHaveAttribute('aria-pressed', 'false')
    await fireEvent.click(toggle)
    expect(mem.getItem('pa.session.filesBarOpen')).toBe('1')
    expect(screen.getByRole('button', { name: /hide files/i })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  it('ignores stale poll results after session switch', async () => {
    const slow = deferred<Array<{ sequence: number; role: string; content: string }>>()
    let calls = 0
    vi.mocked(api.listMessages).mockImplementation(async (id: string) => {
      calls += 1
      if (id === 'old') return slow.promise as never
      return [{ sequence: 1, role: 'user', content: 'new history' }]
    })
    const first = render(SessionChat, {
      props: {
        session: { ...session, id: 'old', title: 'Old' },
        projectId: 'p1',
        pollInterval: 60_000,
      },
    })
    await waitFor(() => expect(calls).toBeGreaterThanOrEqual(1))
    first.unmount()
    render(SessionChat, {
      props: {
        session: { ...session, id: 'new', title: 'New' },
        projectId: 'p1',
        pollInterval: 60_000,
      },
    })
    expect(await screen.findByText('new history')).toBeInTheDocument()
    slow.resolve([{ sequence: 2, role: 'assistant', content: 'stale old' }])
    await waitFor(() => {
      expect(screen.queryByText('stale old')).not.toBeInTheDocument()
    })
    expect(screen.getByText('New')).toBeInTheDocument()
  })
})
