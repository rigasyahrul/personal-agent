// web/src/lib/api/client.test.ts
import { afterEach, describe, expect, it, vi } from 'vitest';
import { APIError, request } from './client';

afterEach(() => vi.unstubAllGlobals());

describe('request', () => {
  it('parses API errors into APIError', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ code: 'bad_request', message: 'Choose another name' }),
      { status: 400, headers: { 'Content-Type': 'application/json' } },
    )));
    await expect(request('/api/v1/vaults')).rejects.toEqual(
      expect.objectContaining<Partial<APIError>>({ status: 400, message: 'Choose another name' }),
    );
  });

  it('adds JSON and CSRF headers to POST requests', async () => {
    document.cookie = 'pa_csrf=token%2Fvalue; path=/';
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    await request('/api/v1/auth/logout', { method: 'POST', body: {} });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/logout', expect.objectContaining({
      method: 'POST',
      body: '{}',
      headers: expect.objectContaining({
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'token/value',
      }),
    }));
  });

  it('does not attach CSRF to GET and returns null for 204', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    await expect(request('/health')).resolves.toBeNull();
    expect(fetchMock.mock.calls[0][1].headers).not.toHaveProperty('X-CSRF-Token');
  });
});
