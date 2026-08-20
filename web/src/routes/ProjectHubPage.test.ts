// web/src/routes/ProjectHubPage.test.ts
import { cleanup, render, screen } from '@testing-library/svelte'
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

afterEach(cleanup)

describe('ProjectHubPage', () => {
  beforeEach(() => {
    vi.mocked(api.getProject).mockReset()
    vi.mocked(api.getProject).mockResolvedValue(project)
  })

  it('renders project metrics and links without a second catalog', async () => {
    render(ProjectHubPage, { props: { projectId: 'p1' } })
    expect(await screen.findByRole('heading', { name: project.name })).toBeVisible()
    expect(screen.getByRole('link', { name: /notes/i })).toHaveAttribute('href', '#/projects/p1/notes')
    expect(screen.getByRole('link', { name: /sessions/i })).toHaveAttribute(
      'href',
      '#/projects/p1/sessions',
    )
    expect(screen.getByRole('link', { name: /review/i })).toHaveAttribute(
      'href',
      '#/projects/p1/review',
    )
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()
  })

  it('shows a retryable hard-load error', async () => {
    vi.mocked(api.getProject).mockRejectedValueOnce(new Error('project missing'))
    render(ProjectHubPage, { props: { projectId: 'p1' } })
    expect(await screen.findByRole('alert')).toHaveTextContent('project missing')
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
  })
})
