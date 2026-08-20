// web/src/components/review/ReviewRunner.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ReviewRunner from './ReviewRunner.svelte'
import { api, APIError } from '../../lib/api'

vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getReviewQueue: vi.fn(),
      rateReviewItem: vi.fn(),
      suspendReviewItem: vi.fn(),
      retryReviewPending: vi.fn(),
    },
  }
})

const item = {
  id: 'r1',
  project_id: 'p1',
  prompt: 'What is SM-2?',
  kind: 'bite',
  answer: 'Spaced repetition',
  note_id: 'n1',
  row_version: 4,
}

afterEach(cleanup)

describe('ReviewRunner', () => {
  beforeEach(() => {
    vi.mocked(api.getReviewQueue).mockReset().mockResolvedValue({
      scope: 'all',
      caught_up: false,
      items: [item],
    })
    vi.mocked(api.rateReviewItem).mockReset().mockResolvedValue(null)
    vi.mocked(api.suspendReviewItem).mockReset().mockResolvedValue(null)
  })

  it('loads queue for the given scope and shows a card', async () => {
    render(ReviewRunner, { props: { scope: 'all' } })
    expect(await screen.findByText('What is SM-2?')).toBeInTheDocument()
    expect(api.getReviewQueue).toHaveBeenCalledWith('all')
  })

  it('shows caught-up empty state from caught_up', async () => {
    vi.mocked(api.getReviewQueue).mockResolvedValue({
      scope: 'all',
      caught_up: true,
      items: [],
    })
    render(ReviewRunner, { props: { scope: 'all' } })
    expect(await screen.findByText(/caught up/i)).toBeInTheDocument()
  })

  it('sends concurrency and timing fields on rate', async () => {
    let now = 100
    render(ReviewRunner, {
      props: {
        scope: 'project:p1',
        now: () => now,
        uuid: () => 'req-1',
      },
    })
    expect(await screen.findByText('What is SM-2?')).toBeInTheDocument()
    now = 142
    await fireEvent.click(screen.getByRole('button', { name: 'Good' }))
    await waitFor(() => {
      expect(api.rateReviewItem).toHaveBeenCalledWith(
        'r1',
        expect.objectContaining({
          rating: 'good',
          request_key: 'req-1',
          row_version: 4,
          duration_ms: 42,
        }),
      )
    })
  })

  it('suppresses duplicate ratings while a request is in flight', async () => {
    let resolveRate: (value: null) => void = () => {}
    vi.mocked(api.rateReviewItem).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRate = resolve
        }),
    )
    render(ReviewRunner, { props: { scope: 'all', uuid: () => 'k1' } })
    expect(await screen.findByText('What is SM-2?')).toBeInTheDocument()
    const good = screen.getByRole('button', { name: 'Good' })
    await fireEvent.click(good)
    await fireEvent.click(good)
    expect(api.rateReviewItem).toHaveBeenCalledTimes(1)
    resolveRate(null)
  })

  it('keeps the card and shows an inline error when rating fails', async () => {
    vi.mocked(api.rateReviewItem).mockRejectedValue(new Error('rate failed'))
    render(ReviewRunner, { props: { scope: 'all', uuid: () => 'k1' } })
    expect(await screen.findByText('What is SM-2?')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Good' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('rate failed')
    expect(screen.getByText('What is SM-2?')).toBeInTheDocument()
  })

  it('reloads the queue after a successful rate', async () => {
    vi.mocked(api.getReviewQueue)
      .mockResolvedValueOnce({ scope: 'all', caught_up: false, items: [item] })
      .mockResolvedValueOnce({ scope: 'all', caught_up: true, items: [] })
    render(ReviewRunner, { props: { scope: 'all', uuid: () => 'k1' } })
    expect(await screen.findByText('What is SM-2?')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Good' }))
    expect(await screen.findByText(/caught up/i)).toBeInTheDocument()
    expect(api.getReviewQueue).toHaveBeenCalledTimes(2)
  })

  it('reloads the queue on 409 conflict', async () => {
    vi.mocked(api.rateReviewItem).mockRejectedValue(new APIError(409, 'conflict'))
    vi.mocked(api.getReviewQueue)
      .mockResolvedValueOnce({ scope: 'all', caught_up: false, items: [item] })
      .mockResolvedValueOnce({
        scope: 'all',
        caught_up: false,
        items: [{ ...item, id: 'r2', prompt: 'Next card', row_version: 1 }],
      })
    render(ReviewRunner, { props: { scope: 'all', uuid: () => 'k1' } })
    expect(await screen.findByText('What is SM-2?')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Good' }))
    expect(await screen.findByText('Next card')).toBeInTheDocument()
  })

  it('suspends the active item and reloads', async () => {
    vi.mocked(api.getReviewQueue)
      .mockResolvedValueOnce({ scope: 'all', caught_up: false, items: [item] })
      .mockResolvedValueOnce({ scope: 'all', caught_up: true, items: [] })
    render(ReviewRunner, { props: { scope: 'all' } })
    expect(await screen.findByText('What is SM-2?')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Suspend' }))
    await waitFor(() => expect(api.suspendReviewItem).toHaveBeenCalledWith('r1'))
    expect(await screen.findByText(/caught up/i)).toBeInTheDocument()
  })

  it('reveals bite answers', async () => {
    render(ReviewRunner, { props: { scope: 'all' } })
    expect(await screen.findByText('What is SM-2?')).toBeInTheDocument()
    expect(screen.queryByText('Spaced repetition')).not.toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Reveal answer' }))
    expect(screen.getByText('Spaced repetition')).toBeInTheDocument()
  })

  it('links whole notes to the current note', async () => {
    vi.mocked(api.getReviewQueue).mockResolvedValue({
      scope: 'all',
      caught_up: false,
      items: [
        {
          id: 'w1',
          project_id: 'p1',
          prompt: 'Whole note',
          kind: 'whole',
          note_id: 'n9',
          row_version: 1,
        },
      ],
    })
    render(ReviewRunner, { props: { scope: 'all' } })
    const link = await screen.findByRole('link', { name: 'Open current note' })
    expect(link).toHaveAttribute('href', '#/projects/p1/notes/n9')
  })

  it('renders scope chips that write scope= into the hash', async () => {
    render(ReviewRunner, {
      props: {
        scope: 'all',
        projectScopes: [{ scope: 'project:p1', label: 'Sleep' }],
      },
    })
    expect(await screen.findByRole('navigation', { name: 'Review scope' })).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Sleep' }))
    expect(location.hash).toContain('scope=')
    expect(location.hash).toContain(encodeURIComponent('project:p1'))
  })

  it('accepts a custom loadQueue for vault filtering', async () => {
    const loadQueue = vi.fn().mockResolvedValue({
      scope: 'all',
      caught_up: false,
      items: [{ ...item, prompt: 'Vault only' }],
    })
    render(ReviewRunner, { props: { scope: 'all', loadQueue, showScopeChips: false } })
    expect(await screen.findByText('Vault only')).toBeInTheDocument()
    expect(loadQueue).toHaveBeenCalled()
    expect(api.getReviewQueue).not.toHaveBeenCalled()
  })
})
