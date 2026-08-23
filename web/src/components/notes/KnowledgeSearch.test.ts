// web/src/components/notes/KnowledgeSearch.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import KnowledgeSearch from './KnowledgeSearch.svelte'
import { api } from '../../lib/api'

vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      searchProject: vi.fn(),
    },
  }
})

const intro = {
  knowledge_id: 'k-intro',
  path: 'source/intro.md',
  title: 'Intro',
  snippet: 'alpha in the body',
  kind: 'source',
  source_note_id: 'note-src',
}

const agents = {
  knowledge_id: 'k-agents',
  path: 'AGENTS.md',
  title: '',
  snippet: 'project instructions',
  kind: 'agents',
}

afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

describe('KnowledgeSearch', () => {
  beforeEach(() => {
    vi.mocked(api.searchProject).mockReset().mockResolvedValue([])
  })

  it('renders a search field and does not query on an empty input', () => {
    render(KnowledgeSearch, { props: { projectId: 'p1', onopen: vi.fn() } })

    expect(screen.getByRole('searchbox', { name: /search knowledge/i })).toBeInTheDocument()
    expect(document.querySelector('.knowledge-search')).toBeTruthy()
    expect(screen.queryByText('No matches.')).not.toBeInTheDocument()
    expect(api.searchProject).not.toHaveBeenCalled()
  })

  it('debounces input then lists title, path, and snippet', async () => {
    vi.useFakeTimers()
    vi.mocked(api.searchProject).mockResolvedValue([intro, agents])

    render(KnowledgeSearch, { props: { projectId: 'p1', onopen: vi.fn() } })
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'alpha' } })

    await vi.advanceTimersByTimeAsync(200)
    expect(api.searchProject).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(200)
    expect(api.searchProject).toHaveBeenCalledOnce()
    expect(api.searchProject).toHaveBeenCalledWith('p1', 'alpha')

    expect(await screen.findByText('Intro')).toBeVisible()
    expect(screen.getByText('source/intro.md')).toBeVisible()
    expect(screen.getByText('alpha in the body')).toBeVisible()
    expect(screen.getByText('AGENTS.md')).toBeVisible()
    expect(screen.getByText('project instructions')).toBeVisible()
    expect(screen.queryByText('No matches.')).not.toBeInTheDocument()
  })

  it('shows No matches after a completed empty search', async () => {
    vi.useFakeTimers()
    vi.mocked(api.searchProject).mockResolvedValue([])

    render(KnowledgeSearch, { props: { projectId: 'p1', onopen: vi.fn() } })
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'zzz' } })
    await vi.advanceTimersByTimeAsync(400)

    expect(await screen.findByText('No matches.')).toBeVisible()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('shows an alert when search fails', async () => {
    vi.useFakeTimers()
    vi.mocked(api.searchProject).mockRejectedValue(new Error('search down'))

    render(KnowledgeSearch, { props: { projectId: 'p1', onopen: vi.fn() } })
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'alpha' } })
    await vi.advanceTimersByTimeAsync(400)

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('search down')
    })
    expect(screen.queryByText('No matches.')).not.toBeInTheDocument()
  })

  it('opens the clicked hit with the Canonical search contract', async () => {
    vi.useFakeTimers()
    vi.mocked(api.searchProject).mockResolvedValue([intro])
    const onopen = vi.fn()

    render(KnowledgeSearch, { props: { projectId: 'p1', onopen } })
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'alpha' } })
    await vi.advanceTimersByTimeAsync(400)

    await fireEvent.click(await screen.findByRole('button', { name: /Intro/ }))

    expect(onopen).toHaveBeenCalledOnce()
    expect(onopen).toHaveBeenCalledWith(intro)
  })

  it('ignores stale results when the query changes', async () => {
    vi.useFakeTimers()
    let resolveFirst!: (hits: typeof intro[]) => void
    const first = new Promise<typeof intro[]>((resolve) => {
      resolveFirst = resolve
    })
    vi.mocked(api.searchProject)
      .mockReturnValueOnce(first)
      .mockResolvedValueOnce([])

    render(KnowledgeSearch, { props: { projectId: 'p1', onopen: vi.fn() } })
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'alpha' } })
    await vi.advanceTimersByTimeAsync(400)

    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'beta' } })
    await vi.advanceTimersByTimeAsync(400)

    resolveFirst([intro])

    expect(await screen.findByText('No matches.')).toBeVisible()
    expect(screen.queryByText('Intro')).not.toBeInTheDocument()
  })
})
