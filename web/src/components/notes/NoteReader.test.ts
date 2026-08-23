// web/src/components/notes/NoteReader.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import NoteReader from './NoteReader.svelte'
import { api } from '../../lib/api'
import { navigate } from '../../lib/router'

vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      listProjectNoteBacklinks: vi.fn(),
    },
  }
})

vi.mock('../../lib/router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/router')>()
  return {
    ...actual,
    navigate: vi.fn(),
  }
})

const note = {
  relative_path: 'guide/a.md',
  body: 'plain body',
  rendered_html: '<p>Rendered note</p>',
}

afterEach(cleanup)

describe('NoteReader', () => {
  beforeEach(() => {
    vi.mocked(api.listProjectNoteBacklinks).mockReset().mockResolvedValue([])
    vi.mocked(navigate).mockReset()
  })

  it('shows a mocked backlink title for the open note', async () => {
    vi.mocked(api.listProjectNoteBacklinks).mockResolvedValueOnce([
      {
        title: 'Memory pointer',
        path: 'memory/pointer.md',
        knowledgeId: 'k-from',
      },
    ])

    render(NoteReader, { props: { note, projectId: 'p1', noteId: 'n1' } })

    expect(await screen.findByRole('button', { name: 'Memory pointer' })).toBeVisible()
    expect(api.listProjectNoteBacklinks).toHaveBeenCalledWith('p1', 'n1')
  })

  it('shows the empty backlinks state when there are none', async () => {
    render(NoteReader, { props: { note, projectId: 'p1', noteId: 'n1' } })

    expect(await screen.findByText('No backlinks yet.')).toBeVisible()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('shows an alert when backlinks fail to load', async () => {
    vi.mocked(api.listProjectNoteBacklinks).mockRejectedValueOnce(new Error('backlinks down'))

    render(NoteReader, { props: { note, projectId: 'p1', noteId: 'n1' } })

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('backlinks down')
    })
    expect(screen.queryByText('No backlinks yet.')).not.toBeInTheDocument()
  })

  it('navigates to a source backlink note id', async () => {
    vi.mocked(api.listProjectNoteBacklinks).mockResolvedValueOnce([
      {
        title: 'Intro',
        path: 'source/intro.md',
        knowledgeId: 'k-intro',
        kind: 'source',
        sourceNoteId: 'note-src',
      },
    ])

    render(NoteReader, { props: { note, projectId: 'p1', noteId: 'n1' } })
    await fireEvent.click(await screen.findByRole('button', { name: 'Intro' }))

    expect(navigate).toHaveBeenCalledWith('#/projects/p1/notes/note-src')
  })
})
