// web/src/routes/ProjectHubPage.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectHubPage from './ProjectHubPage.svelte'
import { api } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getProject: vi.fn(),
      listProjectSessions: vi.fn(),
      listModels: vi.fn(),
      listProjectNotes: vi.fn(),
      getProjectNote: vi.fn(),
      createProjectSession: vi.fn(),
      renameSession: vi.fn(),
      deleteSession: vi.fn(),
      sendMessage: vi.fn(),
      listMessages: vi.fn(),
      currentRun: vi.fn(),
      workspaceTree: vi.fn(),
      workspaceFile: vi.fn(),
    },
  }
})

const project = {
  id: 'p1',
  name: 'Sleep Protocol',
  vault_id: 'v1',
  vault_name: 'HEALTH',
  note_count: 3,
  session_count: 2,
  due_count: 1,
}

const models = { models: [{ provider: 'openai', model_id: 'gpt' }] }

afterEach(cleanup)

describe('ProjectHubPage', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.mocked(api.getProject).mockReset().mockResolvedValue(project)
    vi.mocked(api.listProjectSessions).mockReset().mockResolvedValue([])
    vi.mocked(api.listModels).mockReset().mockResolvedValue(models)
    vi.mocked(api.listProjectNotes).mockReset().mockResolvedValue([])
    vi.mocked(api.createProjectSession).mockReset()
    vi.mocked(api.renameSession).mockReset()
    vi.mocked(api.deleteSession).mockReset()
    vi.mocked(api.sendMessage).mockReset().mockResolvedValue(undefined)
    vi.mocked(api.listMessages).mockReset().mockResolvedValue([])
    vi.mocked(api.currentRun).mockReset().mockResolvedValue(null)
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({ entries: [] })
    vi.mocked(api.workspaceFile).mockReset()
    vi.mocked(api.getProjectNote).mockReset()
  })

  it('shows Claude start prompt and no metric destination grid', async () => {
    render(ProjectHubPage, { props: { projectId: 'p1' } })

    const composer = await screen.findByRole('textbox', { name: /message/i })
    expect(composer).toHaveAttribute('placeholder', 'How can I help you today?')
    expect(screen.queryByRole('heading', { name: /how can i help you today/i })).toBeNull()
    expect(screen.getByRole('heading', { name: 'Sleep Protocol' })).toHaveClass('hub-header__title')
    expect(await screen.findByText('openai:gpt')).toBeInTheDocument()
    expect(document.querySelector('.session-composer__model')).toBeTruthy()
    expect(document.querySelector('.session-composer__send')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Send' })).toHaveClass('session-composer__send')
    expect(screen.queryByRole('region', { name: 'Project metrics' })).toBeNull()
    expect(screen.queryByRole('region', { name: 'Project surfaces' })).toBeNull()
    expect(screen.queryByRole('button', { name: /new session/i })).toBeNull()
    expect(screen.getByRole('link', { name: /notes/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /review/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Memory' })).toBeNull()
  })

  it('lists sessions below the composer as icon/title/date rows', async () => {
    vi.mocked(api.listProjectSessions).mockResolvedValue([
      {
        id: 's1',
        title: 'Test 1',
        status: 'idle',
        provider: 'openai',
        model_id: 'gpt',
        created_at: '2026-05-30T12:00:00.000Z',
      },
    ])

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    const composer = await screen.findByRole('textbox', { name: /message/i })
    expect(composer).toHaveAttribute('placeholder', 'How can I help you today?')
    const recent = await screen.findByRole('heading', { name: /^recent$/i })
    expect(recent).toHaveClass('hub-session-list__label')
    const sessionBtn = await screen.findByRole('button', { name: /Test 1/i })
    expect(sessionBtn.className).toMatch(/session-row/)
    expect(document.querySelector('.session-row__icon')).toBeTruthy()
    expect(screen.getByText('May 30')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /session actions/i })).toBeInTheDocument()

    const afterComposer = composer.compareDocumentPosition(sessionBtn)
    expect(afterComposer & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(recent.compareDocumentPosition(sessionBtn) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('keeps composer when sessions list fails and shows list retry', async () => {
    vi.mocked(api.listProjectSessions).mockRejectedValue(new Error('sessions down'))

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    expect(await screen.findByRole('textbox', { name: /message/i })).toBeVisible()
    expect(screen.getByRole('textbox', { name: /message/i })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent(/sessions down/i)
    expect(screen.getByRole('button', { name: /retry sessions/i })).toBeInTheDocument()

    vi.mocked(api.listProjectSessions).mockResolvedValue([
      {
        id: 's1',
        title: 'Recovered',
        status: 'idle',
        provider: 'openai',
        model_id: 'gpt',
      },
    ])
    await fireEvent.click(screen.getByRole('button', { name: /retry sessions/i }))
    expect(await screen.findByRole('button', { name: /Recovered/i })).toBeInTheDocument()
  })

  it('opens chat after create even if first message fails (no orphan on retry)', async () => {
    const created = {
      id: 's-orphan',
      title: 'calm river',
      status: 'idle',
      provider: 'openai',
      model_id: 'gpt',
    }
    vi.mocked(api.createProjectSession).mockImplementation(async (_pid, input) => ({
      ...created,
      title: input.title,
    }))
    vi.mocked(api.sendMessage).mockRejectedValue(new Error('send failed'))
    vi.mocked(api.listMessages).mockResolvedValue([])
    vi.mocked(api.currentRun).mockResolvedValue(null)
    // initial load empty; after return-to-hub reload returns created session
    vi.mocked(api.listProjectSessions)
      .mockReset()
      .mockResolvedValueOnce([])
      .mockImplementation(async () => {
        const call = vi.mocked(api.createProjectSession).mock.calls[0]
        const title = (call?.[1] as { title: string } | undefined)?.title ?? created.title
        return [{ ...created, title }]
      })

    render(ProjectHubPage, { props: { projectId: 'p1' } })
    await screen.findByRole('textbox', { name: /message/i })
    await fireEvent.input(screen.getByRole('textbox', { name: /message/i }), {
      target: { value: 'Plan the week' },
    })
    await fireEvent.click(screen.getByRole('button', { name: /^send$/i }))

    expect(await screen.findByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sleep Protocol' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull()
    expect(api.createProjectSession).toHaveBeenCalledTimes(1)
    const createArg = vi.mocked(api.createProjectSession).mock.calls[0]![1] as { title: string }
    expect(createArg.title).toMatch(/^[a-z]+ [a-z]+$/)
    await fireEvent.click(screen.getByRole('link', { name: 'Sleep Protocol' }))
    expect(await screen.findByRole('button', { name: new RegExp(createArg.title, 'i') })).toBeInTheDocument()
  })

  it('Send creates session and first message then opens chat', async () => {
    vi.mocked(api.createProjectSession).mockImplementation(async (_pid, input) => ({
      id: 's-new',
      title: input.title,
      status: 'idle',
      provider: 'openai',
      model_id: 'gpt',
    }))
    vi.mocked(api.sendMessage).mockResolvedValue(undefined)

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    const composer = await screen.findByRole('textbox', { name: /message/i })
    await fireEvent.input(composer, { target: { value: 'Plan the week' } })
    await fireEvent.click(screen.getByRole('button', { name: /^send$/i }))

    await waitFor(() => {
      expect(api.createProjectSession).toHaveBeenCalledWith(
        'p1',
        expect.objectContaining({
          home: 'project',
          provider: 'openai',
          model_id: 'gpt',
          model_parameters: {},
          tool_grants: { workspace_files: false },
          title: expect.stringMatching(/^[a-z]+ [a-z]+$/),
        }),
      )
    })
    const createdTitle = (vi.mocked(api.createProjectSession).mock.calls[0]![1] as { title: string })
      .title
    expect(createdTitle).not.toBe('Plan the week')
    await waitFor(() => {
      expect(api.sendMessage).toHaveBeenCalledWith(
        's-new',
        expect.objectContaining({ content: 'Plan the week', request_key: expect.any(String) }),
      )
    })

    expect(await screen.findByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument()
    expect(screen.getByText(createdTitle)).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Sleep Protocol' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull()
  })

  it('Enter in hub composer creates session; Shift+Enter does not submit', async () => {
    vi.mocked(api.createProjectSession).mockImplementation(async (_pid, input) => ({
      id: 's-enter',
      title: input.title,
      status: 'idle',
      provider: 'openai',
      model_id: 'gpt',
    }))
    vi.mocked(api.sendMessage).mockResolvedValue(undefined)

    render(ProjectHubPage, { props: { projectId: 'p1' } })
    const composer = await screen.findByRole('textbox', { name: /message/i })
    await fireEvent.input(composer, { target: { value: 'Hello from Enter' } })

    await fireEvent.keyDown(composer, { key: 'Enter', shiftKey: true })
    expect(api.createProjectSession).not.toHaveBeenCalled()

    await fireEvent.keyDown(composer, { key: 'Enter', shiftKey: false })
    await waitFor(() => {
      expect(api.createProjectSession).toHaveBeenCalledTimes(1)
    })
    await waitFor(() => {
      expect(api.sendMessage).toHaveBeenCalledWith(
        's-enter',
        expect.objectContaining({ content: 'Hello from Enter' }),
      )
    })
    expect(await screen.findByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sleep Protocol' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull()
  })

  it('opens a session row into chat while keeping the rail', async () => {
    vi.mocked(api.listProjectSessions).mockResolvedValue([
      {
        id: 's1',
        title: 'Test 1',
        status: 'idle',
        provider: 'openai',
        model_id: 'gpt',
      },
    ])

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    await fireEvent.click(await screen.findByRole('button', { name: /Test 1/i }))
    expect(await screen.findByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument()
    expect(screen.getByText('Test 1')).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Sleep Protocol' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull()
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
  })

  it('clicking a session row shows chat and project crumb returns to prompt', async () => {
    vi.mocked(api.listProjectSessions).mockResolvedValue([
      {
        id: 's1',
        title: 'Test 1',
        status: 'idle',
        provider: 'openai',
        model_id: 'gpt',
      },
    ])

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    expect(await screen.findByRole('textbox', { name: /message/i })).toBeVisible()
    await fireEvent.click(await screen.findByRole('button', { name: /Test 1/i }))

    expect(await screen.findByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument()
    expect(screen.getByText('Test 1')).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Sleep Protocol' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull()
    expect(screen.queryByRole('heading', { name: /how can i help you today/i })).toBeNull()
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()

    await fireEvent.click(screen.getByRole('link', { name: 'Sleep Protocol' }))

    expect(await screen.findByRole('textbox', { name: /message/i })).toBeVisible()
    // Hub start also has breadcrumbs; session leaf should be gone.
    expect(screen.queryByText('Test 1', { selector: '[aria-current="page"]' })).toBeNull()
    expect(screen.getByRole('heading', { name: 'Sleep Protocol' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
  })

  it('rail Files open drives SessionChat file tab and hides Show files', async () => {
    vi.mocked(api.listProjectSessions).mockResolvedValue([
      {
        id: 's1',
        title: 'Test 1',
        status: 'idle',
        provider: 'openai',
        model_id: 'gpt',
        tool_grants: { workspace_files: true },
      },
    ])
    vi.mocked(api.listProjectNotes).mockResolvedValue([
      { path: 'notes/a.md', kind: 'file', note_id: 'n-a' },
    ])
    vi.mocked(api.getProjectNote).mockResolvedValue({
      note_id: 'n-a',
      relative_path: 'notes/a.md',
      body: '# notes/a.md',
    })
    vi.mocked(api.workspaceFile).mockImplementation(async (_sid, path) => ({
      path,
      kind: 'file',
      content: `# ${path}`,
    }))

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    await fireEvent.click(await screen.findByRole('button', { name: /Test 1/i }))
    expect(await screen.findByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument()
    expect(screen.getByText('Test 1')).toHaveAttribute('aria-current', 'page')
    expect(screen.queryByRole('button', { name: /show files/i })).toBeNull()

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'notes/a.md' }))

    const fileTab = await screen.findByRole('tab', { name: /a\.md/i })
    expect(fileTab).toHaveAttribute('aria-selected', 'true')
    expect(fileTab).toHaveAttribute('title', 'notes/a.md')
    await waitFor(() => expect(api.getProjectNote).toHaveBeenCalledWith('p1', 'n-a'))
    expect(api.workspaceFile).not.toHaveBeenCalled()
  })

  it('shows a retryable hard-load error', async () => {
    vi.mocked(api.getProject).mockRejectedValueOnce(new Error('project missing'))
    render(ProjectHubPage, { props: { projectId: 'p1' } })
    expect(await screen.findByRole('alert')).toHaveTextContent('project missing')
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
  })

  it('wires quiet Notes and Review header links', async () => {
    render(ProjectHubPage, { props: { projectId: 'p1' } })
    await screen.findByRole('textbox', { name: /message/i })
    expect(screen.getByRole('link', { name: /notes/i })).toHaveAttribute(
      'href',
      '#/projects/p1/notes',
    )
    expect(screen.getByRole('link', { name: /review/i })).toHaveAttribute(
      'href',
      '#/projects/p1/review',
    )
  })

  it('wires the default rail preferences into the hub and ProjectRail', async () => {
    render(ProjectHubPage, { props: { projectId: 'p1' } })

    await screen.findByRole('textbox', { name: /message/i })
    expect(document.querySelector('.project-workspace')).toHaveAttribute('data-rail', 'open')
    expect(screen.getByRole('tab', { name: 'Config' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('expands, collapses, and restores the project rail canvas', async () => {
    render(ProjectHubPage, { props: { projectId: 'p1' } })

    await screen.findByRole('textbox', { name: /message/i })
    const workspace = document.querySelector('.project-workspace')
    expect(workspace).not.toBeNull()
    expect(workspace).toHaveAttribute('data-rail', 'open')

    await fireEvent.click(screen.getByRole('button', { name: 'Expand workspace' }))
    expect(workspace).toHaveAttribute('data-rail', 'expanded')
    expect(document.querySelector('.project-workspace__main')).not.toBeVisible()

    await fireEvent.click(screen.getByRole('button', { name: 'Collapse canvas' }))
    expect(workspace).toHaveAttribute('data-rail', 'collapsed')
    expect(screen.getByRole('button', { name: 'Show canvas' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Config' })).toBeNull()
    expect(screen.queryByRole('tab', { name: 'Files' })).toBeNull()

    await fireEvent.click(screen.getByRole('button', { name: 'Show canvas' }))
    expect(workspace).toHaveAttribute('data-rail', 'open')
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
  })

  it('hydrates rail mode from localStorage and persists mode changes', async () => {
    localStorage.setItem('pa.projectRail.mode', 'collapsed')

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    await screen.findByRole('textbox', { name: /message/i })
    const workspace = document.querySelector('.project-workspace')
    expect(workspace).toHaveAttribute('data-rail', 'collapsed')
    expect(screen.getByRole('button', { name: 'Show canvas' })).toBeInTheDocument()

    await fireEvent.click(screen.getByRole('button', { name: 'Show canvas' }))
    expect(workspace).toHaveAttribute('data-rail', 'open')
    expect(localStorage.getItem('pa.projectRail.mode')).toBe('open')

    await fireEvent.click(screen.getByRole('button', { name: 'Collapse canvas' }))
    expect(workspace).toHaveAttribute('data-rail', 'collapsed')
    expect(localStorage.getItem('pa.projectRail.mode')).toBe('collapsed')
  })
})
