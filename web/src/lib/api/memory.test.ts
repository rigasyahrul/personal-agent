// web/src/lib/api/memory.test.ts
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getProjectMemoryLessons } from './memory'

afterEach(() => vi.unstubAllGlobals())

describe('getProjectMemoryLessons', () => {
  it('GETs encoded project lessons path without CSRF', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ content: '# Lessons\n' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(getProjectMemoryLessons('proj/a b')).resolves.toEqual({ content: '# Lessons\n' })
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/projects/proj%2Fa%20b/memory/lessons')
    expect(init.method ?? 'GET').toBe('GET')
    expect(init.headers).not.toHaveProperty('X-CSRF-Token')
  })
})
