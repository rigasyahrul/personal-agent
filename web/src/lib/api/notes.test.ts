// web/src/lib/api/notes.test.ts
import { afterEach, describe, expect, it, vi } from 'vitest'
import { listProjectNoteBacklinks, searchProject } from './notes'

afterEach(() => vi.unstubAllGlobals())

describe('listProjectNoteBacklinks', () => {
  it('GETs encoded note backlinks and maps knowledge_id to knowledgeId', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          items: [
            {
              knowledge_id: 'k-from',
              path: 'memory/pointer.md',
              title: 'Pointer',
              snippet: 'See [[source/guide]]',
              kind: 'source',
              source_note_id: 'note-src',
            },
          ],
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(listProjectNoteBacklinks('proj/a b', 'note/1')).resolves.toEqual([
      {
        knowledgeId: 'k-from',
        path: 'memory/pointer.md',
        title: 'Pointer',
        snippet: 'See [[source/guide]]',
        kind: 'source',
        sourceNoteId: 'note-src',
      },
    ])

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/projects/proj%2Fa%20b/notes/note%2F1/backlinks')
    expect(init.method ?? 'GET').toBe('GET')
    expect(init.headers).not.toHaveProperty('X-CSRF-Token')
  })

  it('treats a missing items list as empty', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({}), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )

    await expect(listProjectNoteBacklinks('p1', 'n1')).resolves.toEqual([])
  })
})

describe('searchProject', () => {
  it('GETs encoded project search without CSRF and returns API-shaped hits', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          hits: [
            {
              knowledge_id: 'k-intro',
              path: 'source/intro.md',
              title: 'Intro',
              snippet: 'alpha in the body',
              kind: 'source',
              source_note_id: 'note-src',
            },
          ],
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(searchProject('proj/a b', 'alpha quark')).resolves.toEqual([
      {
        knowledge_id: 'k-intro',
        path: 'source/intro.md',
        title: 'Intro',
        snippet: 'alpha in the body',
        kind: 'source',
        source_note_id: 'note-src',
      },
    ])

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/projects/proj%2Fa%20b/search?q=alpha%20quark')
    expect(init.method ?? 'GET').toBe('GET')
    expect(init.headers).not.toHaveProperty('X-CSRF-Token')
  })

  it('treats a missing hits list as empty', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({}), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )

    await expect(searchProject('p1', 'alpha')).resolves.toEqual([])
  })
})
