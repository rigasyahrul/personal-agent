// web/src/lib/api/compound.test.ts
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createCompound, decideCompound, getCompound } from './compound'

afterEach(() => {
  vi.unstubAllGlobals()
  document.cookie = 'pa_csrf=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
})

const proposal = {
  id: 'prop-1',
  status: 'pending',
  items: [
    {
      kind: 'agents_patch',
      path: 'AGENTS.md',
      action: 'update',
      content: '# Agents\n',
      content_sha256: 'abc123',
    },
  ],
  created_at: '2026-08-22T12:00:00Z',
}

function stubFetch(body: unknown = proposal, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('createCompound', () => {
  it('POSTs encoded session path with request_key and optional fields only', async () => {
    document.cookie = 'pa_csrf=token%2Fvalue; path=/'
    const fetchMock = stubFetch()

    const got = await createCompound('sess/a b', {
      request_key: 'rk-1',
      user_context: 'last turn',
      items: [proposal.items[0]],
    })

    expect(got).toEqual(proposal)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/sessions/sess%2Fa%20b/compound')
    expect(init.method).toBe('POST')
    expect(init.headers).toEqual(
      expect.objectContaining({
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'token/value',
      }),
    )
    expect(JSON.parse(init.body)).toEqual({
      request_key: 'rk-1',
      user_context: 'last turn',
      items: [proposal.items[0]],
    })
    expect(JSON.parse(init.body)).not.toHaveProperty('scope')
    expect(JSON.parse(init.body)).not.toHaveProperty('project_id')
    expect(JSON.parse(init.body)).not.toHaveProperty('vault_id')
  })
})

describe('getCompound', () => {
  it('GETs encoded session and proposal ids without CSRF', async () => {
    const fetchMock = stubFetch({
      ...proposal,
      decided_at: '2026-08-22T12:05:00Z',
      finished_at: '2026-08-22T12:06:00Z',
    })

    const got = await getCompound('sess/a', 'prop/1')

    expect(got.id).toBe('prop-1')
    expect(got.decided_at).toBe('2026-08-22T12:05:00Z')
    expect(got.finished_at).toBe('2026-08-22T12:06:00Z')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/sessions/sess%2Fa/compound/prop%2F1')
    expect(init.method).toBe('GET')
    expect(init.headers).not.toHaveProperty('X-CSRF-Token')
  })
})

describe('decideCompound', () => {
  it('POSTs decide with request_key, decision, and optional edited items', async () => {
    document.cookie = 'pa_csrf=csrf-token; path=/'
    const approved = { ...proposal, status: 'approved', decided_at: '2026-08-22T12:05:00Z' }
    const fetchMock = stubFetch(approved)
    const edited = { ...proposal.items[0], content: '# Edited\n', title: 'Agents' }

    const got = await decideCompound('sess-1', 'prop-1', {
      request_key: 'rk-decide',
      decision: 'approve',
      items: [edited],
    })

    expect(got.status).toBe('approved')
    expect(got.decided_at).toBe('2026-08-22T12:05:00Z')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/sessions/sess-1/compound/prop-1/decide')
    expect(init.method).toBe('POST')
    expect(init.headers).toEqual(
      expect.objectContaining({ 'X-CSRF-Token': 'csrf-token' }),
    )
    expect(JSON.parse(init.body)).toEqual({
      request_key: 'rk-decide',
      decision: 'approve',
      items: [edited],
    })
    expect(JSON.parse(init.body)).not.toHaveProperty('scope')
  })
})
