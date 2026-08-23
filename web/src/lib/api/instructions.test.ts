// web/src/lib/api/instructions.test.ts
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  getGlobalInstruction,
  getProjectInstruction,
  putGlobalInstruction,
  putProjectInstruction,
} from './instructions'

afterEach(() => {
  vi.unstubAllGlobals()
  document.cookie = 'pa_csrf=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
})

function stubFetch(body: unknown = { content: '# Soul\n' }, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('getGlobalInstruction', () => {
  it('GETs encoded global instruction path without CSRF', async () => {
    const fetchMock = stubFetch({ content: 'Be kind.\n' })

    await expect(getGlobalInstruction('soul')).resolves.toEqual({ content: 'Be kind.\n' })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/global/instructions/soul')
    expect(init.method ?? 'GET').toBe('GET')
    expect(init.headers).not.toHaveProperty('X-CSRF-Token')
  })
})

describe('putGlobalInstruction', () => {
  it('PUTs encoded global instruction with content and CSRF', async () => {
    document.cookie = 'pa_csrf=token%2Fvalue; path=/'
    const fetchMock = stubFetch({ content: 'Be kinder.\n' })

    await expect(putGlobalInstruction('system', 'Be kinder.\n')).resolves.toEqual({
      content: 'Be kinder.\n',
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/global/instructions/system')
    expect(init.method).toBe('PUT')
    expect(init.headers).toEqual(
      expect.objectContaining({
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'token/value',
      }),
    )
    expect(JSON.parse(init.body)).toEqual({ content: 'Be kinder.\n' })
  })
})

describe('getProjectInstruction', () => {
  it('GETs encoded project instruction path without CSRF', async () => {
    const fetchMock = stubFetch({ content: '# Agents\n' })

    await expect(getProjectInstruction('proj/a b', 'agents')).resolves.toEqual({
      content: '# Agents\n',
    })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/projects/proj%2Fa%20b/instructions/agents')
    expect(init.method ?? 'GET').toBe('GET')
    expect(init.headers).not.toHaveProperty('X-CSRF-Token')
  })
})

describe('putProjectInstruction', () => {
  it('PUTs encoded project instruction with content and CSRF', async () => {
    document.cookie = 'pa_csrf=csrf-token; path=/'
    const fetchMock = stubFetch({ content: 'Ship it.\n' })

    await expect(putProjectInstruction('proj/a b', 'soul', 'Ship it.\n')).resolves.toEqual({
      content: 'Ship it.\n',
    })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/projects/proj%2Fa%20b/instructions/soul')
    expect(init.method).toBe('PUT')
    expect(init.headers).toEqual(
      expect.objectContaining({ 'X-CSRF-Token': 'csrf-token' }),
    )
    expect(JSON.parse(init.body)).toEqual({ content: 'Ship it.\n' })
  })
})
