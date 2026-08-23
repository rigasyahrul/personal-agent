// web/src/routes/NotesPage.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import NotesPage from './NotesPage.svelte'
import { api } from '../lib/api'
import { navigate } from '../lib/router'

vi.mock('../lib/router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/router')>()
  return {
    ...actual,
    navigate: vi.fn(),
  }
})

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getProject: vi.fn(),
      listProjectNotes: vi.fn(),
      getProjectNote: vi.fn(),
      listProjectNoteBacklinks: vi.fn(),
      searchProject: vi.fn(),
    },
  }
})

const project = {
  id: 'p1',
  name: 'Sleep',
  note_count: 1,
  vault_id: null,
}

const note = {
  kind: 'file',
  path: 'guide/a.md',
  note_id: 'n1',
}

const detail = {
  relative_path: 'guide/a.md',
  body: 'plain body',
  rendered_html: '<p>Rendered note</p>',
}

afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

describe('NotesPage', () => {
  beforeEach(() => {
    vi.mocked(api.getProject).mockReset().mockResolvedValue(project)
    vi.mocked(api.listProjectNotes).mockReset().mockResolvedValue([note])
    vi.mocked(api.getProjectNote).mockReset().mockResolvedValue(detail)
    vi.mocked(api.listProjectNoteBacklinks).mockReset().mockResolvedValue([])
    vi.mocked(api.searchProject).mockReset().mockResolvedValue([])
    vi.mocked(navigate).mockReset()
  })

  it('shows a knowledge search field when the project is open', async () => {
    render(NotesPage, { props: { projectId: 'p1' } })
    expect(await screen.findByRole('searchbox', { name: /search knowledge/i })).toBeVisible()
  })

  it('opens a source search hit on the v1 notes route', async () => {
    vi.mocked(api.searchProject).mockResolvedValue([
      {
        knowledge_id: 'k-intro',
        path: 'source/intro.md',
        title: 'Intro',
        snippet: 'alpha in the body',
        kind: 'source',
        source_note_id: 'note-src',
      },
    ])

    render(NotesPage, { props: { projectId: 'p1' } })
    const search = await screen.findByRole('searchbox', { name: /search knowledge/i })
    vi.useFakeTimers()
    await fireEvent.input(search, { target: { value: 'alpha' } })
    await vi.advanceTimersByTimeAsync(400)
    await fireEvent.click(await screen.findByRole('button', { name: /Intro/ }))

    expect(navigate).toHaveBeenCalledWith('#/projects/p1/notes/note-src')
  })

  it('shows tree and selected note in two panes', async () => {
    render(NotesPage, { props: { projectId: 'p1', noteId: 'n1' } })
    expect(await screen.findByRole('tree')).toBeVisible()
    expect(await screen.findByRole('article')).toHaveTextContent('Rendered note')
  })

  it('distinguishes an empty tree from no selection', async () => {
    vi.mocked(api.listProjectNotes).mockResolvedValueOnce([])
    const empty = render(NotesPage, { props: { projectId: 'p1' } })
    expect(await screen.findByText(/no notes yet/i)).toBeVisible()
    empty.unmount()

    vi.mocked(api.listProjectNotes).mockResolvedValueOnce([note])
    render(NotesPage, { props: { projectId: 'p1' } })
    expect(await screen.findByText(/select a note/i)).toBeVisible()
  })

  it('keeps the tree when detail loading fails', async () => {
    vi.mocked(api.getProjectNote).mockRejectedValueOnce(new Error('note gone'))
    render(NotesPage, { props: { projectId: 'p1', noteId: 'n1' } })
    expect(await screen.findByRole('tree')).toBeVisible()
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('note gone')
    })
  })

  it('shows a mocked backlink title for the open note', async () => {
    vi.mocked(api.listProjectNoteBacklinks).mockResolvedValueOnce([
      {
        title: 'Intro from memory',
        path: 'memory/pointer.md',
        knowledgeId: 'k-from',
      },
    ])
    render(NotesPage, { props: { projectId: 'p1', noteId: 'n1' } })
    expect(await screen.findByRole('button', { name: 'Intro from memory' })).toBeVisible()
    expect(api.listProjectNoteBacklinks).toHaveBeenCalledWith('p1', 'n1')
  })

  it('renders plain text body when no rendered HTML is provided', async () => {
    vi.mocked(api.getProjectNote).mockResolvedValueOnce({
      relative_path: 'guide/a.md',
      body: 'plain <b>body</b>',
    })
    render(NotesPage, { props: { projectId: 'p1', noteId: 'n1' } })
    const article = await screen.findByRole('article')
    expect(article.textContent).toContain('plain <b>body</b>')
    expect(article.querySelector('b')).toBeNull()
  })
})
