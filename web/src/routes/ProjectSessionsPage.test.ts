// web/src/routes/ProjectSessionsPage.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectSessionsPage from './ProjectSessionsPage.svelte'
import { api } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getProject: vi.fn(),
      listModels: vi.fn(),
      listProjectSessions: vi.fn(),
      createProjectSession: vi.fn(),
      listMessages: vi.fn(),
      currentRun: vi.fn(),
      sendMessage: vi.fn(),
    },
  }
})

const project = { id: 'p1', name: 'Sleep', note_count: 0 }
const models = { models: [{ provider: 'openai', model_id: 'gpt' }] }
const sessions = [
  { id: 's0', title: 'Old', status: 'idle', provider: 'openai', model_id: 'gpt' },
]

afterEach(cleanup)

describe('ProjectSessionsPage', () => {
  beforeEach(() => {
    vi.mocked(api.getProject).mockReset().mockResolvedValue(project)
    vi.mocked(api.listModels).mockReset().mockResolvedValue(models)
    vi.mocked(api.listProjectSessions).mockReset().mockResolvedValue(sessions)
    vi.mocked(api.createProjectSession).mockReset().mockResolvedValue({
      id: 's1',
      title: 'Plan',
      status: 'idle',
      provider: 'openai',
      model_id: 'gpt',
    })
    vi.mocked(api.listMessages).mockReset().mockResolvedValue([])
    vi.mocked(api.currentRun).mockReset().mockResolvedValue(null)
  })

  it('lists and creates only through the project endpoint', async () => {
    render(ProjectSessionsPage, { props: { projectId: 'p1' } })
    const title = await screen.findByLabelText('Title')
    await fireEvent.input(title, { target: { value: 'Plan' } })
    const select = screen.getByLabelText('Model') as HTMLSelectElement
    // value is provider\0model_id
    await fireEvent.change(select, { target: { value: 'openai\u0000gpt' } })
    await fireEvent.click(screen.getByRole('button', { name: 'New session' }))
    await waitFor(() => {
      expect(api.createProjectSession).toHaveBeenCalledWith('p1', {
        home: 'project',
        title: 'Plan',
        provider: 'openai',
        model_id: 'gpt',
        model_parameters: {},
        tool_grants: { workspace_files: false },
      })
    })
  })

  it('shows setup guidance rather than a form when models are empty', async () => {
    vi.mocked(api.listModels).mockResolvedValue({ models: [] })
    render(ProjectSessionsPage, { props: { projectId: 'p1' } })
    expect(await screen.findByText(/configure a model/i)).toBeVisible()
    expect(screen.queryByRole('button', { name: 'New session' })).not.toBeInTheDocument()
  })

  it('opens a session from the list into the chat shell slot', async () => {
    render(ProjectSessionsPage, { props: { projectId: 'p1' } })
    await fireEvent.click(await screen.findByRole('button', { name: /Old/i }))
    expect(await screen.findByRole('heading', { name: 'Old' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Back' })).toBeInTheDocument()
    expect(document.querySelector('.content-canvas--session-focus')).toBeTruthy()
  })
})
