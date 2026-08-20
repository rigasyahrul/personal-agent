// web/src/routes/NotesPage.test.ts
import { cleanup, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import NotesPage from './NotesPage.svelte'
import { api } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getProject: vi.fn(),
      listProjectNotes: vi.fn(),
      getProjectNote: vi.fn(),
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

afterEach(cleanup)

describe('NotesPage', () => {
  beforeEach(() => {
    vi.mocked(api.getProject).mockReset().mockResolvedValue(project)
    vi.mocked(api.listProjectNotes).mockReset().mockResolvedValue([note])
    vi.mocked(api.getProjectNote).mockReset().mockResolvedValue(detail)
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
