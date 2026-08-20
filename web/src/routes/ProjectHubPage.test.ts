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
      createProjectSession: vi.fn(),
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
    vi.mocked(api.getProject).mockReset().mockResolvedValue(project)
    vi.mocked(api.listProjectSessions).mockReset().mockResolvedValue([])
    vi.mocked(api.listModels).mockReset().mockResolvedValue(models)
    vi.mocked(api.listProjectNotes).mockReset().mockResolvedValue([])
    vi.mocked(api.createProjectSession).mockReset()
    vi.mocked(api.sendMessage).mockReset().mockResolvedValue(undefined)
    vi.mocked(api.listMessages).mockReset().mockResolvedValue([])
    vi.mocked(api.currentRun).mockReset().mockResolvedValue(null)
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({ entries: [] })
    vi.mocked(api.workspaceFile).mockReset()
  })

  it('shows Claude start prompt and no metric destination grid', async () => {
    render(ProjectHubPage, { props: { projectId: 'p1' } })

    expect(await screen.findByRole('heading', { name: /how can i help you today/i })).toBeVisible()
    expect(screen.queryByRole('region', { name: 'Project metrics' })).toBeNull()
    expect(screen.queryByRole('region', { name: 'Project surfaces' })).toBeNull()
    expect(screen.queryByRole('button', { name: /new session/i })).toBeNull()
    expect(screen.getByRole('link', { name: /notes/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /review/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Memory' })).toBeInTheDocument()
  })

  it('lists sessions below the composer', async () => {
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

    const heading = await screen.findByRole('heading', { name: /how can i help you today/i })
    const composer = screen.getByRole('textbox', { name: /message/i })
    const sessionBtn = await screen.findByRole('button', { name: /Test 1/i })

    const position = heading.compareDocumentPosition(composer)
    expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()

    const afterComposer = composer.compareDocumentPosition(sessionBtn)
    expect(afterComposer & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('Send creates session and first message then opens chat', async () => {
    const created = {
      id: 's-new',
      title: 'Plan the week',
      status: 'idle',
      provider: 'openai',
      model_id: 'gpt',
    }
    vi.mocked(api.createProjectSession).mockResolvedValue(created)
    vi.mocked(api.sendMessage).mockResolvedValue(undefined)

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    const composer = await screen.findByRole('textbox', { name: /message/i })
    await fireEvent.input(composer, { target: { value: 'Plan the week' } })
    await fireEvent.click(screen.getByRole('button', { name: /^send$/i }))

    await waitFor(() => {
      expect(api.createProjectSession).toHaveBeenCalledWith('p1', {
        home: 'project',
        title: 'Plan the week',
        provider: 'openai',
        model_id: 'gpt',
        model_parameters: {},
        tool_grants: { workspace_files: false },
      })
    })
    await waitFor(() => {
      expect(api.sendMessage).toHaveBeenCalledWith(
        's-new',
        expect.objectContaining({ content: 'Plan the week', request_key: expect.any(String) }),
      )
    })

    expect(await screen.findByRole('heading', { name: 'Plan the week' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Back' })).toBeInTheDocument()
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
    expect(await screen.findByRole('heading', { name: 'Test 1' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Back' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Memory' })).toBeInTheDocument()
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
      { path: 'notes/a.md', kind: 'file' },
    ])
    vi.mocked(api.workspaceFile).mockImplementation(async (_sid, path) => ({
      path,
      kind: 'file',
      content: `# ${path}`,
    }))

    render(ProjectHubPage, { props: { projectId: 'p1' } })

    await fireEvent.click(await screen.findByRole('button', { name: /Test 1/i }))
    expect(await screen.findByRole('heading', { name: 'Test 1' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /show files/i })).toBeNull()

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'notes/a.md' }))

    const fileTab = await screen.findByRole('tab', { name: /a\.md/i })
    expect(fileTab).toHaveAttribute('aria-selected', 'true')
    expect(fileTab).toHaveAttribute('title', 'notes/a.md')
  })

  it('shows a retryable hard-load error', async () => {
    vi.mocked(api.getProject).mockRejectedValueOnce(new Error('project missing'))
    render(ProjectHubPage, { props: { projectId: 'p1' } })
    expect(await screen.findByRole('alert')).toHaveTextContent('project missing')
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
  })

  it('wires quiet Notes and Review header links', async () => {
    render(ProjectHubPage, { props: { projectId: 'p1' } })
    await screen.findByRole('heading', { name: /how can i help you today/i })
    expect(screen.getByRole('link', { name: /notes/i })).toHaveAttribute(
      'href',
      '#/projects/p1/notes',
    )
    expect(screen.getByRole('link', { name: /review/i })).toHaveAttribute(
      'href',
      '#/projects/p1/review',
    )
  })
})
