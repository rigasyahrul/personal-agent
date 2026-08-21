// web/src/components/sessions/SessionChat.focus.test.ts
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

const initialMessages = [{ sequence: 1, role: 'user', content: 'hello' }]
const reply = { sequence: 2, role: 'assistant', content: 'reply' }

afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

describe('SessionChat focus invariant', () => {
  beforeEach(() => {
    vi.mocked(api.listMessages).mockReset().mockResolvedValue([...initialMessages])
    vi.mocked(api.currentRun).mockReset().mockResolvedValue(null)
    vi.mocked(api.sendMessage).mockReset()
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({ entries: [] })
    vi.mocked(api.getProject).mockReset().mockResolvedValue({
      id: 'p1',
      name: 'P',
      note_count: 0,
    })
  })

  it('patches messages and run state without replacing the focused composer', async () => {
    let messages = [...initialMessages]
    let run: { status: string } | null = null
    vi.mocked(api.listMessages).mockImplementation(async () => messages)
    vi.mocked(api.currentRun).mockImplementation(async () => run)

    const { component } = render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000 },
    })

    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await fireEvent.focus(composer)
    await fireEvent.input(composer, { target: { value: 'typing here' } })
    composer.focus()
    composer.setSelectionRange(6, 11)

    messages = [...initialMessages, reply]
    run = { status: 'running' }

    // Trigger an explicit poll if exposed; otherwise wait for internal poller via component export.
    const poll = (component as unknown as { poll?: () => Promise<void> }).poll
    if (typeof poll === 'function') {
      await poll()
    } else {
      // Fall back: re-invoke listMessages path by advancing if start already ran once.
      await waitFor(() => expect(api.listMessages.mock.calls.length).toBeGreaterThanOrEqual(1))
      // Force another poll through a second start isn't available — call list again by remounting is forbidden.
      // SessionChat must expose poll() for tests; assert after manual apply via API mock change + poll().
      throw new Error('SessionChat must expose poll() for focus regression harness')
    }

    expect(screen.getByLabelText('Message')).toBe(composer)
    expect(document.activeElement).toBe(composer)
    expect(composer.value).toBe('typing here')
    expect([composer.selectionStart, composer.selectionEnd]).toEqual([6, 11])
    expect(screen.getByText(reply.content)).toBeVisible()
    expect(screen.getByRole('status')).toHaveTextContent('Run: running')
  })
})
