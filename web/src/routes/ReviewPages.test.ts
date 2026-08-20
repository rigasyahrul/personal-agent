// web/src/routes/ReviewPages.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ReviewPage from './ReviewPage.svelte'
import ProjectReviewPage from './ProjectReviewPage.svelte'
import { api } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getReviewQueue: vi.fn(),
      rateReviewItem: vi.fn(),
      suspendReviewItem: vi.fn(),
      getProject: vi.fn(),
      listProjects: vi.fn(),
    },
  }
})

const item = {
  id: 'r1',
  project_id: 'p1',
  prompt: 'Card prompt',
  kind: 'bite',
  answer: 'A',
  note_id: 'n1',
  row_version: 4,
}

afterEach(cleanup)

describe('Review pages', () => {
  beforeEach(() => {
    location.hash = '#/review'
    vi.mocked(api.getReviewQueue).mockReset().mockResolvedValue({
      scope: 'all',
      caught_up: false,
      items: [item],
    })
    vi.mocked(api.rateReviewItem).mockReset().mockResolvedValue(null)
    vi.mocked(api.suspendReviewItem).mockReset().mockResolvedValue(null)
    vi.mocked(api.getProject).mockReset().mockResolvedValue({
      id: 'p1',
      name: 'Sleep',
      note_count: 0,
    })
    vi.mocked(api.listProjects).mockReset().mockResolvedValue([
      { id: 'p1', name: 'Sleep', note_count: 0 },
      { id: 'p2', name: 'Budget', note_count: 0 },
    ])
  })

  it('defaults global review to all', async () => {
    render(ReviewPage, { props: { query: new URLSearchParams() } })
    await waitFor(() => {
      expect(api.getReviewQueue).toHaveBeenCalledWith('all')
    })
    expect(await screen.findByRole('heading', { name: 'Review' })).toBeInTheDocument()
  })

  it('honors an explicit global scope query', async () => {
    render(ReviewPage, {
      props: { query: new URLSearchParams('scope=project:p1') },
    })
    await waitFor(() => {
      expect(api.getReviewQueue).toHaveBeenCalledWith('project:p1')
    })
  })

  it('uses project scope and sends concurrency/timing fields', async () => {
    render(ProjectReviewPage, { props: { projectId: 'p1' } })
    await waitFor(() => {
      expect(api.getReviewQueue).toHaveBeenCalledWith('project:p1')
    })
    expect(await screen.findByText('Card prompt')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Good' }))
    await waitFor(() => {
      expect(api.rateReviewItem).toHaveBeenCalledWith(
        'r1',
        expect.objectContaining({
          rating: 'good',
          request_key: expect.any(String),
          row_version: 4,
          duration_ms: expect.any(Number),
        }),
      )
    })
  })

  it('shows project breadcrumbs on project review', async () => {
    render(ProjectReviewPage, { props: { projectId: 'p1' } })
    expect(await screen.findByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument()
    expect(screen.getByText('Sleep')).toBeInTheDocument()
  })
})
