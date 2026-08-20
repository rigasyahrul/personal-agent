// web/src/components/sessions/WorkspacePanel.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SessionChat from './SessionChat.svelte'
import WorkspacePanel from './WorkspacePanel.svelte'
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

describe('WorkspacePanel', () => {
  beforeEach(() => {
    vi.mocked(api.listMessages).mockReset().mockResolvedValue([])
    vi.mocked(api.currentRun).mockReset().mockResolvedValue(null)
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({
      entries: [{ path: 'draft.md', kind: 'file' }],
    })
    vi.mocked(api.workspaceFile).mockReset().mockResolvedValue({
      path: 'draft.md',
      kind: 'file',
      content: '# hi',
    })
    vi.mocked(api.getProject).mockResolvedValue({ id: 'p1', name: 'P', note_count: 0 })
  })

  it.each([
    { grants: '{bad', visible: false },
    { grants: '{"workspace_files":false}', visible: false },
    { grants: '{"workspace_files":true}', visible: true },
  ])('gates workspace from persisted grants %#', async ({ grants, visible }) => {
    render(SessionChat, {
      props: {
        session: { ...baseSession, tool_grants_json: grants },
        projectId: 'p1',
        pollInterval: 60_000,
      },
    })
    await screen.findByLabelText('Message')
    if (visible) {
      expect(await screen.findByRole('complementary', { name: 'Workspace' })).toBeVisible()
    } else {
      expect(screen.queryByRole('complementary', { name: 'Workspace' })).not.toBeInTheDocument()
    }
  })

  it('refreshes tree after a newly polled tool message changes a path', async () => {
    let messages: Array<{ sequence: number; role: string; content: string; changed_path?: string }> = []
    vi.mocked(api.listMessages).mockImplementation(async () => messages as never)
    const { component } = render(SessionChat, {
      props: {
        session: { ...baseSession, tool_grants: { workspace_files: true } },
        projectId: 'p1',
        pollInterval: 60_000,
      },
    })
    await screen.findByRole('complementary', { name: 'Workspace' })
    await waitFor(() => expect(api.workspaceTree).toHaveBeenCalledTimes(1))
    messages = [{ sequence: 1, role: 'tool', content: '', changed_path: 'new.txt' }]
    const poll = (component as unknown as { poll: () => Promise<void> }).poll
    await poll()
    await waitFor(() => expect(api.workspaceTree).toHaveBeenCalledTimes(2))
  })

  it('loads file content and offers Save to source for md files', async () => {
    const onpromote = vi.fn()
    render(WorkspacePanel, {
      props: { sessionId: 's1', messages: [], onpromote },
    })
    await fireEvent.click(await screen.findByRole('button', { name: 'draft.md' }))
    expect(await screen.findByText('# hi')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Save to source' }))
    expect(onpromote).toHaveBeenCalledWith(expect.objectContaining({ path: 'draft.md' }))
  })
})
